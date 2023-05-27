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

package microservice

import (
	appv1 "canary-crd/pkg/apis/app/v1"
	"context"
	"k8s.io/apimachinery/pkg/types"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
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

//microservice_controller.go: 这个文件是 MicroService 控制器的主要文件。它包含了一些关键的方法，
//如 Reconcile。Reconcile 方法负责读取集群中的 MicroService 对象的状态，并根据读取的状态和 MicroService 对象的 Spec 进行相应的操作。
//这些操作可能包括创建、更新和删除 Deployment，以及更新 MicroService 对象的状态。

//在 Kubernetes 中，App 和 MicroService 是两种不同的自定义资源（Custom Resource）。虽
//然 MicroService 对象通常作为 App 对象的一部分被处理，但它们也可以作为独立的资源存在和被操作。这种设计可以提供更大的灵活性和更细粒度的控制。
//
//以下是一些可能需要单独处理 MicroService 对象的情况：
//
//独立的 MicroService 对象：在某些情况下，你可能需要创建一个不属于任何 App 对象的 MicroService 对象。例如，你可能有一个公共的 MicroService，它被多个 App 对象共享。在这种情况下，你需要能够单独管理这个 MicroService 对象。
//
//更细粒度的控制：MicroService 控制器可以提供更细粒度的控制。例如，你可能想要在 MicroService 对象发生变化时执行一些特定的操作，这些操作可能与 App 对象无关，或者超出了 App 控制器的职责范围。
//
//解耦 App 和 MicroService：将 App 和 MicroService 的管理分开可以降低复杂性和耦合度。这样，你可以单独更新和扩展 App 控制器和 MicroService 控制器，而不需要担心它们会相互影响。
//
//总的来说，虽然在许多情况下，处理 App 相关的 MicroService 对象可能就足够了，但在某些情况下，单独处理 MicroService 对象可以提供更大的灵活性和更细粒度的控制。

// Add creates a new MicroService Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
// Add(mgr manager.Manager) error：这个方法创建一个新的 MicroService 控制器并将其添加到 Manager。当 Manager 启动时，它会设置控制器的字段并启动控制器。
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
// newReconciler(mgr manager.Manager) reconcile.Reconciler：这个方法返回一个新的 reconcile.Reconciler，
// 它是一个 ReconcileMicroService 结构体的实例，该结构体实现了 reconcile.Reconciler 接口。
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileMicroService{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
// add(mgr manager.Manager, r reconcile.Reconciler) error：这个方法将一个新的控制器添加到 mgr，r 是 reconcile.Reconciler。
// 它创建一个新的控制器，并设置其 Reconciler 为 r。然后，它为 MicroService 对象和由 MicroService 对象创建的 Deployment、Service 和 Ingress 资源设置了 Watch。
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("microservice-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	//EnqueueRequestForOwner 和 EnqueueRequestForObject 是 Kubernetes controller-runtime 库中的两种事件处理函数，它们决定了当 Watch 的资源发生变化时，应该将哪些请求加入到工作队列中。
	//
	//EnqueueRequestForOwner：这个函数用于处理拥有者资源的事件。当一个被 Watch 的资源发生变化时，它会查找这个资源的所有者，
	//并将所有者的名字和命名空间作为请求加入到工作队列中。这样，Reconcile 方法就会被调用来处理所有者资源。
	//这个函数常常用于处理那些由某个资源（如 Custom Resource）创建并管理的其他资源（如 Deployment、Service）的事件。
	//
	//EnqueueRequestForObject：这个函数用于处理资源本身的事件。当一个被 Watch 的资源发生变化时，
	//它会直接将这个资源的名字和命名空间作为请求加入到工作队列中。这样，Reconcile 方法就会被调用来处理这个资源。
	//这个函数常常用于处理那些自身就需要被处理的资源（如 Custom Resource）的事件。
	//
	//总的来说，EnqueueRequestForOwner 和 EnqueueRequestForObject 的主要区别在于它们处理的是资源的所有者还是资源本身。
	//
	//让我们以一个自定义资源（CRD）MicroService 和它管理的 Kubernetes 原生资源 Deployment, Service (简称 svc), 和 Ingress 为例。
	//
	//假设我们有一个 MicroService CRD，它的 Spec 字段定义了一个应用的部署配置（对应一个 Deployment），
	//一个服务配置（对应一个 Service），和一个 Ingress 配置（对应一个 Ingress）。当 MicroService 对象被创建或更新时，
	//我们希望相应地创建或更新对应的 Deployment, Service, 和 Ingress。
	//
	//在这种情况下，我们可以在 MicroService 控制器中使用 EnqueueRequestForObject 来处理 MicroService 对象的事件。
	//当 MicroService 对象发生变化时，EnqueueRequestForObject 会将 MicroService 对象的名字和命名空间作为请求加入到工作队列中，
	//然后 Reconcile 方法会被调用来处理 MicroService 对象。
	//
	//同时，我们也可以在 MicroService 控制器中使用 EnqueueRequestForOwner 来处理 Deployment, Service, 和 Ingress 的事件。
	//当这些资源发生变化时，EnqueueRequestForOwner 会查找这些资源的所有者（即 MicroService 对象），
	//并将所有者的名字和命名空间作为请求加入到工作队列中，然后 Reconcile 方法会被调用来处理 MicroService 对象。
	//
	//所以，总的来说，EnqueueRequestForObject 用于处理资源本身的事件，而 EnqueueRequestForOwner 用于处理由某个资源管理的其他资源的事件。
	//在这个例子中，MicroService 对象是资源本身，而 Deployment, Service, 和 Ingress 是由 MicroService 对象管理的其他资源。

	// Watch for changes to MicroService
	err = c.Watch(&source.Kind{Type: &appv1.MicroService{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch resource created by MicroService
	err = c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &appv1.MicroService{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &appv1.MicroService{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &extensionsv1beta1.Ingress{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &appv1.MicroService{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileMicroService{}

// ReconcileMicroService reconciles a MicroService object
type ReconcileMicroService struct {
	client.Client
	scheme *runtime.Scheme
}

//Reconcile(request reconcile.Request) (reconcile.Result, error)：这个方法读取集群中的 MicroService 对象的状态，
//并根据读取的状态和 MicroService 对象的 Spec 进行相应的操作。这些操作可能包括同步 MicroService 对象的状态，
//处理 MicroService 对象的实例，以及处理 MicroService 对象的负载均衡。如果 MicroService 对象的 Spec 发生变化，这个方法还会更新 MicroService 对象。

// Reconcile reads that state of the cluster for a MicroService object and makes changes based on the state read
// and what is in the MicroService.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=app.o0w0o.cn,resources=microservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=app.o0w0o.cn,resources=microservices/status,verbs=get;update;patch
func (r *ReconcileMicroService) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the MicroService instance
	instance := &appv1.MicroService{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	if instance.DeletionTimestamp != nil {
		log.Info("Get deleted MicroService, and do nothing.")
		return reconcile.Result{}, nil
	}

	if err := r.syncMicroServiceStatus(instance); err != nil {
		log.Info("Sync MicroServiceStatus error", err)
		return reconcile.Result{}, err
	}

	if err := r.reconcileInstance(instance); err != nil {
		log.Info("Reconcile Instance Versions error", err)
		return reconcile.Result{}, err
	}

	if err := r.reconcileLoadBalance(instance); err != nil {
		log.Info("Reconcile LoadBalance error", err)
		return reconcile.Result{}, err
	}

	oldMS := &appv1.MicroService{}
	if err := r.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, oldMS); err != nil {
		return reconcile.Result{}, err
	}
	if !reflect.DeepEqual(oldMS.Spec, instance.Spec) {
		oldMS.Spec = instance.Spec
		if err := r.Update(context.TODO(), oldMS); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}
