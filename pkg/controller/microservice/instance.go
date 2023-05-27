package microservice

import (
	appv1 "canary-crd/pkg/apis/app/v1"
	"context"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//instance.go: 这个文件主要负责处理 MicroService 对象的实例。它包含了一些关键的方法，如 reconcileInstance 和 syncMicroServiceStatus。
//reconcileInstance 方法负责处理 MicroService 对象的实例，包括创建、更新和删除。syncMicroServiceStatus 方法则负责同步 MicroService 对象的状态。

// *reconcileInstance(microService appv1.MicroService) error：这个方法负责处理 MicroService 对象的实例。
// 它首先创建一个新的 Deployment 映射，然后遍历 MicroService 对象的 Versions 字段，为每个版本创建一个 Deployment，
// 并将其添加到映射中。然后，它会检查每个新的 Deployment 是否已经存在，如果不存在，它会创建一个新的 Deployment，如果已经存在，
// 它会检查 Deployment 的 Spec 字段是否发生了变化，如果发生了变化，它会更新 Deployment。
// 最后，它会清理那些在新的 Deployment 映射中不存在，但在 Kubernetes 集群中存在的 Deployment。
func (r *ReconcileMicroService) reconcileInstance(microService *appv1.MicroService) error {

	newDeploys := make(map[string]*appsv1.Deployment)
	for i := range microService.Spec.Versions {
		version := &microService.Spec.Versions[i]

		deploy, err := makeVersionDeployment(version, microService)
		if err != nil {
			log.Error(err, "Make Deployment for version error", "versionName", version.Name)
			return err
		}
		if err := controllerutil.SetControllerReference(microService, deploy, r.scheme); err != nil {
			log.Error(err, "Set DeployVersion CtlRef Error", "versionName", version.Name)
			return err
		}

		newDeploys[deploy.Name] = deploy
		found := &appsv1.Deployment{}
		err = r.Get(context.TODO(), types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}, found)

		if err != nil && errors.IsNotFound(err) {

			log.Info("Old Deployment NotFound and Creating new one", "namespace", deploy.Namespace, "name", deploy.Name)
			if err = r.Create(context.TODO(), deploy); err != nil {
				return err
			}

		} else if err != nil {

			log.Error(err, "Get Deployment info Error", "namespace", deploy.Namespace, "name", deploy.Name)
			return err

		} else if !reflect.DeepEqual(deploy.Spec, found.Spec) {

			// Update the found object and write the result back if there are any changes
			found.Spec = deploy.Spec
			log.Info("Old deployment changed and Updating Deployment to reconcile", "namespace", deploy.Namespace, "name", deploy.Name)
			err = r.Update(context.TODO(), found)
			if err != nil {
				return err
			}

		}
	}
	return r.cleanUpDeploy(microService, newDeploys)
}

// makeVersionDeployment(version *appv1.DeployVersion, microService *appv1.MicroService) (*appsv1.Deployment, error)：
// 这个方法创建一个新的 Deployment 对象。它接收一个 DeployVersion 对象和一个 MicroService 对象，然后返回一个新的 Deployment 对象。
func makeVersionDeployment(version *appv1.DeployVersion, microService *appv1.MicroService) (*appsv1.Deployment, error) {

	labels := microService.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["app.o0w0o.cn/service"] = microService.Name
	labels["app.o0w0o.cn/version"] = version.Name

	deploySpec := version.Template

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      microService.Name + "-" + version.Name,
			Namespace: microService.Namespace,
			Labels:    labels,
		},
		Spec: deploySpec,
	}

	return deploy, nil
}

// **cleanUpDeploy(microService appv1.MicroService, newDeployList map[string]appsv1.Deployment) error：
// 这个方法负责清理那些在新的 Deployment 映射中不存在，但在 Kubernetes 集群中存在的 Deployment。
// 它会列出所有的 Deployment，然后删除那些不在 newDeployList 映射中的 Deployment。
func (r *ReconcileMicroService) cleanUpDeploy(microService *appv1.MicroService, newDeployList map[string]*appsv1.Deployment) error {
	// Check if the MicroService not exists
	ctx := context.Background()

	deployList := appsv1.DeploymentList{}
	labels := make(map[string]string)
	labels["app.o0w0o.cn/service"] = microService.Name

	if err := r.List(ctx, client.InNamespace(microService.Namespace).
		MatchingLabels(labels), &deployList); err != nil {
		log.Error(err, "unable to list old MicroServices")
		return err
	}

	for _, oldDeploy := range deployList.Items {
		if _, exist := newDeployList[oldDeploy.Name]; exist == false {
			log.Info("Find orphan Deployment", "namespace", microService.Namespace, "MicroService", microService.Name, "Deployment", oldDeploy.Name)
			err := r.Delete(context.TODO(), &oldDeploy)
			if err != nil {
				log.Error(err, "Delete orphan Deployment error", "namespace", oldDeploy.Namespace, "name", oldDeploy.Name)
				return err
			}
		}
	}
	return nil
}

// *syncMicroServiceStatus(microService appv1.MicroService) error：
// 这个方法负责同步 MicroService 对象的状态。它首先检查 MicroService 对象的 AvailableVersions 字段和 TotalVersions 字段，
// 如果这两个字段相等，它会直接返回。然后，它会计算 MicroService 对象的新状态，并更新 MicroService 对象的状态。
func (r *ReconcileMicroService) syncMicroServiceStatus(microService *appv1.MicroService) error {
	if microService.Status.AvailableVersions != 0 && microService.Status.TotalVersions == microService.Status.AvailableVersions {
		return nil
	}

	ctx := context.Background()
	newStatus, err := r.calculateStatus(microService)
	if err != nil {
		return err
	}

	condType := appv1.MicroServiceProgressing
	status := appv1.ConditionTrue
	reason := ""
	message := ""
	if newStatus.AvailableVersions == newStatus.TotalVersions {
		condType = appv1.MicroServiceAvailable
		reason = "All deploy have updated."
	} else if newStatus.AvailableVersions > newStatus.TotalVersions {
		reason = "Some deploys got to be deleted."
	} else {
		reason = "Some deploys got to be created."
	}
	condition := appv1.MicroServiceCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
	conditions := microService.Status.Conditions
	for i := range conditions {
		newStatus.Conditions = append(newStatus.Conditions, conditions[i])
	}
	newStatus.Conditions = append(newStatus.Conditions, condition)
	microService.Status = newStatus
	err = r.Status().Update(ctx, microService)
	return err
}

// calculateStatus(microService *appv1.MicroService) (appv1.MicroServiceStatus, error)：
// 这个方法计算 MicroService 对象的新状态。它会列出所有的 Deployment，然后计算 AvailableVersions 和 TotalVersions，然后返回一个新的 MicroServiceStatus 对象。
func (r *ReconcileMicroService) calculateStatus(microService *appv1.MicroService) (appv1.MicroServiceStatus, error) {
	// Check if the MicroService not exists
	ctx := context.Background()

	deployList := appsv1.DeploymentList{}
	labels := make(map[string]string)
	labels["app.o0w0o.cn/service"] = microService.Name

	al := int32(len(deployList.Items))
	tl := int32(len(microService.Spec.Versions))
	newStatus := appv1.MicroServiceStatus{
		AvailableVersions: al,
		TotalVersions:     tl,
	}
	if err := r.List(ctx, client.InNamespace(microService.Namespace).
		MatchingLabels(labels), &deployList); err != nil {
		log.Error(err, "unable to list old MicroServices")
		return newStatus, err
	}
	newStatus.AvailableVersions = int32(len(deployList.Items))

	return newStatus, nil
}
