/*
Copyright 2019 Hypo.

Licensed under the GNU General Public License, Version 3 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/Coderhypo/canary-crd/blob/master/LICENSE

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package app

import (
	appv1 "canary-crd/pkg/apis/app/v1"
	"context"
	"k8s.io/apimachinery/pkg/types"
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add 创建一个新的 App 控制器并将其添加到 Manager 中。Manager 会设置控制器的字段
// 并在 Manager 启动时启动它。
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler 返回一个新的 reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileApp{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add 将新的 Controller 添加到 mgr 中，r 作为 reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// 创建一个新的控制器
	c, err := controller.New("app-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// 监视 App 的变化
	err = c.Watch(&source.Kind{Type: &appv1.App{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// 监视由 App 创建的 MicroService
	err = c.Watch(&source.Kind{Type: &appv1.MicroService{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &appv1.App{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileApp{}

// ReconcileApp 是一个实现了 reconcile.Reconciler 的结构体，它用于处理 App 对象的变化
type ReconcileApp struct {
	client.Client
	scheme *runtime.Scheme
}

//这个方法的主要作用是处理 App 对象的变化，包括创建、更新和删除。当 App 对象发生变化时，Kubernetes 会调用这个方法。
//这个方法首先会获取 App 对象的当前状态，然后根据 App 对象的状态和 App.Spec 中的内容进行相应的操作，
//例如创建或更新 MicroService。如果 App 对象被删除，这个方法会清理与 App 对象关联的资源。

// Reconcile 读取集群中 App 对象的状态，并根据读取的状态和 App.Spec 中的内容进行变更
// 自动生成 RBAC 规则以允许控制器读取和写入 Deployments
// Reconcile reads that state of the cluster for a App object and makes changes based on the state read
// and what is in the App.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=app.o0w0o.cn,resources=apps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.o0w0o.cn,resources=apps/status,verbs=get;update;patch
func (r *ReconcileApp) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// 获取 App 实例
	instance := &appv1.App{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// 如果对象不存在，则返回。创建的对象会自动被垃圾回收。
			// 对于额外的清理逻辑，使用 finalizers。
			return reconcile.Result{}, nil
		}
		// 读取对象出错 - 重新加入请求队列。
		return reconcile.Result{}, err
	}
	if instance.DeletionTimestamp != nil {
		// 如果 App 已被删除，清理子资源。
		log.Info("Get deleted App, clean up subResources.")
		return reconcile.Result{}, nil
	}

	// 同步 App 的状态
	if err := r.syncAppStatus(instance); err != nil {
		log.Info("Sync App error", err)
		return reconcile.Result{}, err
	}

	// 处理与 App 关联的 MicroService
	if err := r.reconcileMicroService(request, instance); err != nil {
		log.Info("Creating MicroService error", err)
		return reconcile.Result{}, err
	}

	// 获取旧的 App 对象
	oldApp := &appv1.App{}
	if err := r.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, oldApp); err != nil {
		return reconcile.Result{}, err
	}
	// 如果旧的 App 对象和当前的 App 对象不同，则更新旧的 App 对象
	if !reflect.DeepEqual(oldApp.Spec, instance.Spec) {
		oldApp.Spec = instance.Spec
		if err := r.Update(context.TODO(), oldApp); err != nil {
			return reconcile.Result{}, err
		}
	}
	// 返回 reconcile.Result 和 nil 错误，表示 reconcile 操作成功完成
	return reconcile.Result{}, nil
}
