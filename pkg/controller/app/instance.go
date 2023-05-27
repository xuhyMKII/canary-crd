package app

import (
	appv1 "canary-crd/pkg/apis/app/v1"
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

//这个文件定义了 App 控制器的行为。App 控制器负责监视 App 资源的变化，并根据 App 资源的状态进行相应的操作。
//这个文件中的函数主要负责创建、更新和删除 MicroService 资源，以及同步 App 资源的状态。

//reconcileMicroService 方法在 instance.go 文件中，是 ReconcileApp 结构体的一个方法。它负责处理与 App 对象关联的 MicroService 对象。以下是该方法的主要逻辑：
//
//定义新的 MicroService 对象：首先，方法会创建一个新的 MicroService 对象的映射，这些对象是基于 App 对象的 Spec.MicroServices 字段创建的。
//
//处理每个 MicroService：对于 App 对象的 Spec.MicroServices 字段中的每个 MicroService，方法会检查是否已经存在一个相同的 MicroService 对象。
//如果不存在，则创建一个新的 MicroService 对象。如果存在，但其 Spec 字段与新的 MicroService 对象不同，则更新已存在的 MicroService 对象。
//
//清理旧的 MicroService 对象：最后，方法会清理那些在新的 MicroService 对象映射中不存在，但在 Kubernetes 集群中存在的 MicroService 对象。这些对象是旧的 MicroService 对象，它们不再与 App 对象关联。
//
//总的来说，reconcileMicroService 方法负责同步 App 对象和 MicroService 对象。当 App 对象发生变化时，方法会确保 Kubernetes 集群中的 MicroService 对象与 App 对象的状态保持一致。

func (r *ReconcileApp) reconcileMicroService(req reconcile.Request, app *appv1.App) error {
	// Define the desired MicroService object
	labels := app.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["app.o0w0o.cn/app"] = app.Name
	newMicroServices := make(map[string]*appv1.MicroService)

	for i := range app.Spec.MicroServices {
		microService := &app.Spec.MicroServices[i]

		ms := &appv1.MicroService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      app.Name + "-" + microService.Name,
				Namespace: app.Namespace,
				Labels:    labels,
			},
			Spec: microService.Spec,
		}
		if err := controllerutil.SetControllerReference(app, ms, r.scheme); err != nil {
			return err
		}

		newMicroServices[ms.Name] = ms
		// Check if the MicroService already exists
		found := &appv1.MicroService{}
		err := r.Get(context.TODO(), types.NamespacedName{Name: ms.Name, Namespace: ms.Namespace}, found)

		if err != nil && errors.IsNotFound(err) {
			log.Info("Creating MicroService", "namespace", ms.Namespace, "name", ms.Name)
			if err = r.Create(context.TODO(), ms); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		if !reflect.DeepEqual(ms.Spec, found.Spec) {

			found.Spec = ms.Spec
			log.Info("find MS changed and Updating MicroService", "namespace", ms.Namespace, "name", ms.Name)
			err = r.Update(context.TODO(), found)
			if err != nil {
				return err
			}

			err := r.Get(context.TODO(), types.NamespacedName{Name: ms.Name, Namespace: ms.Namespace}, found)
			if err != nil {
				return err
			}
			microService.Spec = found.Spec

		}
	}
	return r.cleanUpMicroServices(app, newMicroServices)
}

// reconcileMicroService 方法的主要任务是确保 App 对象的 MicroService 子资源与 App 对象的期望状态保持一致。
// 这个方法首先会根据 App 对象的 Spec.MicroServices 字段创建一个新的 MicroService 对象的映射，
// 然后对比 Kubernetes 集群中实际存在的 MicroService 对象。
//
// 在以下情况下，可能会出现 "映射中不存在，但在集群中存在的 MicroService 对象"：
//
// App 对象的 Spec.MicroServices 字段发生变化：如果 App 对象的 Spec.MicroServices 字段发生变化，例如删除了一个 MicroService，
// 那么新的 MicroService 对象的映射中就不会包含这个 MicroService。但是，这个 MicroService 对象可能仍然在 Kubernetes 集群中存在，因为它是在之前的 App 对象状态下创建的。
//
// App 对象被删除：如果 App 对象被删除，那么新的 MicroService 对象的映射将为空，因为 App 对象不再存在。但是，如果 MicroService 对象没有被正确清理，它们可能仍然在 Kubernetes 集群中存在。
//
// 在这两种情况下，reconcileMicroService 方法都会清理那些在新的 MicroService 对象映射中不存在，
// 但在 Kubernetes 集群中存在的 MicroService 对象，以确保 Kubernetes 集群中的 MicroService 对象与 App 对象的期望状态保持一致。
func (r *ReconcileApp) cleanUpMicroServices(app *appv1.App, msList map[string]*appv1.MicroService) error {
	// Check if the MicroService not exists
	ctx := context.Background()

	microServiceList := appv1.MicroServiceList{}
	labels := make(map[string]string)
	labels["app.o0w0o.cn/app"] = app.Name

	if err := r.List(ctx, client.InNamespace(app.Namespace).
		MatchingLabels(labels), &microServiceList); err != nil {
		log.Error(err, "unable to list old MicroServices")
		return err
	}

	for i := range microServiceList.Items {
		oldMs := &microServiceList.Items[i]
		if _, exist := msList[oldMs.Name]; exist == false {
			log.Info("Deleted orphan MS and will delete it", "namespace", app.Namespace, "App", app.Namespace, "MS", oldMs.Name)
			err := r.Delete(context.TODO(), oldMs)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// syncAppStatus 方法的主要任务是同步 App 对象的状态。以下是该方法的主要逻辑：
//
// 检查 App 对象的状态是否已经同步：如果 App 对象的状态已经同步（即 AvailableMicroServices 等于 TotalMicroServices），那么方法直接返回，不进行任何操作。
//
// 计算新的 App 对象状态：方法会调用 calculateStatus 方法来计算新的 App 对象状态。
// calculateStatus 方法会获取 Kubernetes 集群中与 App 对象关联的所有 MicroService 对象，
// 然后计算 AvailableMicroServices 和 TotalMicroServices 的值。
//
// 更新 App 对象的状态：如果新的 App 对象状态与当前的状态不同，那么方法会更新 App 对象的状态，并将新的状态写入 Kubernetes API。
//
// 总的来说，syncAppStatus 方法负责同步 App 对象的状态。当 App 对象或其关联的 MicroService 对象发生变化时，方法会确保 App 对象的状态与 Kubernetes 集群中的实际状态保持一致。
func (r *ReconcileApp) syncAppStatus(app *appv1.App) error {
	if app.Status.AvailableMicroServices != 0 && app.Status.AvailableMicroServices == app.Status.TotalMicroServices {
		return nil
	}

	ctx := context.Background()
	newStatus, err := r.calculateStatus(app)
	if err != nil {
		return err
	}

	condType := appv1.AppProgressing
	status := appv1.ConditionTrue
	reason := ""
	message := ""
	if newStatus.AvailableMicroServices == newStatus.TotalMicroServices {
		condType = appv1.AppAvailable
		reason = "All deploy have updated."
	} else if newStatus.AvailableMicroServices > newStatus.TotalMicroServices {
		reason = "Some microservices got to be deleted."
	} else {
		reason = "Some microservices got to be created."
	}
	condition := appv1.AppCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
	conditions := app.Status.Conditions
	for i := range conditions {
		newStatus.Conditions = append(newStatus.Conditions, conditions[i])
	}
	newStatus.Conditions = append(newStatus.Conditions, condition)
	app.Status = newStatus
	err = r.Status().Update(ctx, app)
	return err
}

//calculateStatus 方法的主要任务是计算 App 对象的新状态。以下是该方法的主要逻辑：
//获取所有的 MicroService 对象：方法首先会获取 Kubernetes 集群中与 App 对象关联的所有 MicroService 对象。这些对象是通过匹配 App 对象的标签来获取的。
//计算 AvailableMicroServices 和 TotalMicroServices：然后，方法会计算 AvailableMicroServices 和 TotalMicroServices 的值。
//AvailableMicroServices 的值是 Kubernetes 集群中实际存在的 MicroService 对象的数量，
//TotalMicroServices 的值是 App 对象的 Spec.MicroServices 字段的长度，即 App 对象期望存在的 MicroService 对象的数量。
//返回新的 App 对象状态：最后，方法会返回一个新的 AppStatus 对象，这个对象包含了计算出的 AvailableMicroServices 和 TotalMicroServices 的值。
//总的来说，calculateStatus 方法负责计算 App 对象的新状态。这个状态反映了 App 对象期望存在的 MicroService 对象的数量和 Kubernetes 集群中实际存在的 MicroService 对象的数量。

func (r *ReconcileApp) calculateStatus(app *appv1.App) (appv1.AppStatus, error) {
	// Check if the MicroService not exists
	ctx := context.Background()

	msList := appv1.MicroServiceList{}
	labels := make(map[string]string)
	labels["app.o0w0o.cn/app"] = app.Name

	al := int32(len(msList.Items))
	tl := int32(len(app.Spec.MicroServices))
	newStatus := appv1.AppStatus{
		AvailableMicroServices: al,
		TotalMicroServices:     tl,
	}
	if err := r.List(ctx, client.InNamespace(app.Namespace).
		MatchingLabels(labels), &msList); err != nil {
		log.Error(err, "unable to list old MicroServices")
		return newStatus, err
	}
	newStatus.AvailableMicroServices = int32(len(msList.Items))

	return newStatus, nil
}
