package microservice

import (
	appv1 "canary-crd/pkg/apis/app/v1"
	"context"
	v1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
)

//loadbalance.go: 这个文件主要负责处理 MicroService 对象的负载均衡。
//它包含了一些关键的方法，如 reconcileLoadBalance 和 clearUpLB。
//reconcileLoadBalance 方法负责处理 MicroService 对象的负载均衡，
//包括创建、更新和删除。clearUpLB 方法则负责清理不再需要的负载均衡资源。

//*reconcileLoadBalance(microService appv1.MicroService) error：这个方法负责处理 MicroService 对象的负载均衡配置。
//它首先检查 MicroService 对象的 LoadBalance 字段和 Versions 字段。如果 LoadBalance 字段为空或者 Versions 字段为空，
//它会清理旧的负载均衡配置。然后，它会处理 MicroService 对象的 Service 和 Ingress 负载均衡配置。最后，它会清理那些不再需要的 Service 和 Ingress 对象。

func (r *ReconcileMicroService) reconcileLoadBalance(microService *appv1.MicroService) error {
	lb := microService.Spec.LoadBalance
	staySVCName := make([]string, 5)
	stayIngressName := make([]string, 5)

	if lb == nil || len(microService.Spec.Versions) == 0 {
		log.Info("microService has NONE LB config, and clear up old LB", "namespace", microService.Namespace, "name", microService.Name)
		return r.clearUpLB(microService, &staySVCName, &stayIngressName)
	}

	currentVersion := &microService.Spec.Versions[0]
	defaultVersion := true
	for i := range microService.Spec.Versions {
		version := &microService.Spec.Versions[i]
		if version.Name == microService.Spec.CurrentVersionName {
			log.Info("Get current Version", "name", version.Name)
			currentVersion = version
			defaultVersion = false
			break
		}
	}

	if defaultVersion {
		log.Info("microService do not set currentVersion, and choose first for current", "namespace", microService.Namespace, "microService", microService.Name, "defaultCurrentVersion", currentVersion.Name)
	}

	enableSVC := false
	if lb.Service != nil {

		svcLB := lb.Service
		enableSVC = true
		log.Info("microService enable SVC LB, and every version has independent SVC", "namespace", microService.Namespace, "microService", microService.Name)

		svcLB.Spec.Selector = currentVersion.Template.Selector.MatchLabels
		svc, err := makeService(svcLB.Name, microService.Namespace, microService.Labels, &svcLB.Spec)
		if err != nil {
			return err
		}

		for k, v := range microService.Labels {
			svc.Labels[k] = v
		}
		if err := controllerutil.SetControllerReference(microService, svc, r.scheme); err != nil {
			return err
		}

		if err := r.updateOrCreateSVC(svc); err != nil {
			log.Error(err, "Set SVC LB error", "namespace", microService.Namespace, "microService", microService.Name)
			return err
		}
		staySVCName = append(staySVCName, svc.Name)
	}

	enableIngress := false
	if lb.Ingress != nil {
		enableIngress = true
		ingressLB := lb.Ingress
		ingress := &extensionsv1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      ingressLB.Name,
				Namespace: microService.Namespace,
				Labels:    microService.Labels,
			},
			Spec: ingressLB.Spec,
		}
		if err := controllerutil.SetControllerReference(microService, ingress, r.scheme); err != nil {
			return err
		}
		if err := r.updateOrCreateIngress(ingress); err != nil {
			log.Error(err, "Set Ingress LB error", "namespace", microService.Namespace, "microService", microService.Name)
			return err
		}
		stayIngressName = append(stayIngressName, ingress.Name)
	}

	if enableSVC {
		for i := range microService.Spec.Versions {
			version := &microService.Spec.Versions[i]
			spec := lb.Service.Spec.DeepCopy()
			spec.Selector = version.Template.Selector.MatchLabels
			serviceName := version.ServiceName
			if serviceName == "" {
				serviceName = microService.Name + "-" + version.Name
			}
			log.Info("Set DeployVersion SVC", "namespace", microService.Namespace, "microService", microService.Name, "Version", version.Name, "SVC", serviceName)
			svc, err := makeService(serviceName, microService.Namespace, microService.Labels, spec)
			if err != nil {
				return err
			}
			if err := controllerutil.SetControllerReference(microService, svc, r.scheme); err != nil {
				return err
			}

			if err := r.updateOrCreateSVC(svc); err != nil {
				log.Error(err, "Set DeployVersion SVC Error", "namespace", microService.Namespace, "microService", microService.Name, "Version", version.Name)
				return err
			}
			version.ServiceName = serviceName
			staySVCName = append(staySVCName, serviceName)
		}
	}

	if enableIngress {
		for _, version := range microService.Spec.Versions {
			if version.Canary == nil {
				continue
			}
			log.Info("Set Canary Ingress", "namespace", microService.Namespace, "microService", microService.Name, "Version", version.Name)
			ingress, err := makeCanaryIngress(microService, &lb.Ingress.Spec, &version)
			if err != nil {
				return err
			}
			if err := controllerutil.SetControllerReference(microService, ingress, r.scheme); err != nil {
				return err
			}
			if err := r.updateOrCreateIngress(ingress); err != nil {
				log.Error(err, "Set Canary Ingress error", "namespace", microService.Namespace, "microService", microService.Name, "Version", version.Name)
				return err
			}
			stayIngressName = append(stayIngressName, ingress.Name)
		}
	}

	return r.clearUpLB(microService, &staySVCName, &stayIngressName)
}

//*updateOrCreateSVC(svc v1.Service) error：这个方法负责创建或更新 Service 对象。如果 Service 对象不存在，
//它会创建一个新的 Service 对象。如果 Service 对象已经存在，它会检查 Service 对象的 Spec 字段是否发生了变化，如果发生了变化，它会更新 Service 对象。

func (r *ReconcileMicroService) updateOrCreateSVC(svc *v1.Service) error {
	// Check if the Service already exists
	found := &v1.Service{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Service", "namespace", svc.Namespace, "name", svc.Name)
		if err := r.Create(context.TODO(), svc); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if !reflect.DeepEqual(svc.Spec, found.Spec) {
		svc.Spec.ClusterIP = found.Spec.ClusterIP
		found.Spec = svc.Spec
		if err = r.Update(context.TODO(), found); err != nil {
			return err
		}
		log.Info("Find SVC as been modified, update", "namespace", svc.Namespace, "name", svc.Name)
	}
	return nil
}

// *updateOrCreateIngress(ingress extensionsv1beta1.Ingress) error：这个方法负责创建或更新 Ingress 对象。如果 Ingress 对象不存在，
// 它会创建一个新的 Ingress 对象。如果 Ingress 对象已经存在，它会检查 Ingress 对象的 Spec 字段和 Annotations 字段是否发生了变化，如果发生了变化，它会更新 Ingress 对象。
func (r *ReconcileMicroService) updateOrCreateIngress(ingress *extensionsv1beta1.Ingress) error {
	found := &extensionsv1beta1.Ingress{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: ingress.Name, Namespace: ingress.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Ingress", "namespace", ingress.Namespace, "name", ingress.Name)
		if err = r.Create(context.TODO(), ingress); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if !reflect.DeepEqual(ingress.Spec, found.Spec) || !reflect.DeepEqual(ingress.Annotations, found.Annotations) {
		found.Spec = ingress.Spec
		found.Annotations = ingress.Annotations
		if err = r.Update(context.TODO(), found); err != nil {
			return err
		}
		log.Info("Find Ingress as been modified", "namespace", ingress.Namespace, "name", ingress.Name)
	}
	return nil
}

//**clearUpLB(microService *appv1.MicroService, staySVCName []string, stayIngressName []string) error：这个方法负责清理那些不再需要的 Service 和 Ingress 对象。
//它会列出所有的 Service 和 Ingress 对象，然后删除那些不在 staySVCName 和 stayIngressName 列表中的对象。

func (r *ReconcileMicroService) clearUpLB(microService *appv1.MicroService, staySVCName *[]string, stayIngressName *[]string) error {
	opts := &client.ListOptions{}
	opts.InNamespace(microService.Namespace)
	opts.MatchingLabels(map[string]string{"app.o0w0o.cn/service": microService.Name})

	allSVC := &v1.ServiceList{}
	if err := r.List(context.TODO(), opts, allSVC); err != nil {
		return err
	}
	for _, svc := range allSVC.Items {
		found := false
		for _, svcName := range *staySVCName {
			if svcName == svc.Name {
				found = true
				break
			}
		}
		if !found {
			if err := r.Client.Delete(context.TODO(), svc.DeepCopy()); err != nil {
				return err
			}
		}
	}

	allIngress := &extensionsv1beta1.IngressList{}
	if err := r.List(context.TODO(), opts, allIngress); err != nil {
		return err
	}
	for _, ingress := range allIngress.Items {
		found := false
		for _, ingressName := range *stayIngressName {
			if ingressName == ingress.Name {
				found = true
				break
			}
		}
		if !found {
			if err := r.Client.Delete(context.TODO(), ingress.DeepCopy()); err != nil {
				return err
			}
		}
	}

	return nil
}

// makeService(name string, namespace string, label map[string]string, svcSpec *v1.ServiceSpec) (*v1.Service, error)：这个方法创建一个新的 Service 对象。
// 它接收一个名字、一个命名空间、一个标签映射和一个 ServiceSpec 对象，然后返回一个新的 Service 对象。
func makeService(name string, namespace string, label map[string]string, svcSpec *v1.ServiceSpec) (*v1.Service, error) {
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    label,
		},
		Spec: *svcSpec,
	}
	return svc, nil
}

// makeCanaryIngress(microService *appv1.MicroService, ingressSpec *extensionsv1beta1.IngressSpec, version *appv1.DeployVersion) (*extensionsv1beta1.Ingress, error)：
// 这个方法创建一个新的 Ingress 对象。它接收一个 MicroService 对象、一个 IngressSpec 对象和一个 DeployVersion 对象，然后返回一个新的 Ingress 对象。
func makeCanaryIngress(microService *appv1.MicroService, ingressSpec *extensionsv1beta1.IngressSpec, version *appv1.DeployVersion) (*extensionsv1beta1.Ingress, error) {
	// TODO nginx ingress controller support ONLY now
	canary := version.Canary
	annotations := map[string]string{
		"nginx.ingress.kubernetes.io/canary":        "true",
		"nginx.ingress.kubernetes.io/canary-weight": strconv.Itoa(canary.Weight),
	}

	if canary.Header != "" {
		annotations["nginx.ingress.kubernetes.io/canary-by-header"] = canary.Header
		annotations["nginx.ingress.kubernetes.io/canary-by-header-value"] = canary.HeaderValue
	}

	if canary.Cookie != "" {
		annotations["nginx.ingress.kubernetes.io/canary-by-cookie"] = canary.Cookie
	}

	if canary.CanaryIngressName == "" {
		canary.CanaryIngressName = microService.Name + "-" + version.Name + "-canary"
	}

	ingressSpec = ingressSpec.DeepCopy()

	if ingressSpec.Rules != nil {
		for i, rule := range ingressSpec.Rules {
			if rule.IngressRuleValue.HTTP == nil {
				continue
			}
			for j, path := range rule.IngressRuleValue.HTTP.Paths {
				if path.Backend.ServiceName == microService.Spec.LoadBalance.Service.Name {
					ingressSpec.Rules[i].IngressRuleValue.HTTP.Paths[j].Backend.ServiceName = version.ServiceName
				}
			}
		}
	}
	ingress := &extensionsv1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        canary.CanaryIngressName,
			Namespace:   microService.Namespace,
			Labels:      microService.Labels,
			Annotations: annotations,
		},
		Spec: *ingressSpec,
	}

	return ingress, nil
}
