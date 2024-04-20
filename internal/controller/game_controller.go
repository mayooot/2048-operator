/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"reflect"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"

	myappv1 "mayooot.github.io/2048-operator/api/v1"
)

// GameReconciler reconciles a Game object
type GameReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// 添加 RBAC 规则
//+kubebuilder:rbac:groups=myapp.mayooot.github.io,resources=games,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=myapp.mayooot.github.io,resources=games/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=myapp.mayooot.github.io,resources=games/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Game object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.2/pkg/reconcile
func (r *GameReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	defer utilruntime.HandleCrash()

	logger := log.FromContext(ctx)
	logger.Info("receive reconcile event", "name", req.String())

	// 获取 game 对象
	game := &myappv1.Game{}
	if err := r.Get(ctx, req.NamespacedName, game); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 如果处在正常删除的状态，直接跳过
	if game.DeletionTimestamp != nil {
		logger.Info("game in deleting", "name", req.String())
		return ctrl.Result{}, nil
	}

	// 同步资源，如果资源不存在，创建 Deployment、Service、Ingress，并更新 status
	if err := r.syncGame(ctx, game); err != nil {
		logger.Error(err, "failed to sync game", "name", req.String())
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

const (
	gameLabelName = "mayooot.github.io/2048-operator"
	port          = 80
)

func (r *GameReconciler) syncGame(ctx context.Context, obj *myappv1.Game) error {
	logger := log.FromContext(ctx)

	// 深拷贝一份 game 对象
	game := obj.DeepCopy()
	name := types.NamespacedName{
		Namespace: game.Namespace,
		Name:      game.Name,
	}

	// 构造 owner，通过 kubectl get xx -o yaml 时可以在 metadata.ownerReferences 字段看到
	owner := []metav1.OwnerReference{
		{
			APIVersion:         game.APIVersion,
			Kind:               game.Kind,
			Name:               game.Name,
			Controller:         ptr.To(true),
			BlockOwnerDeletion: ptr.To(true),
			UID:                game.UID,
		},
	}

	labels := map[string]string{
		gameLabelName: game.Name,
	}

	meta := metav1.ObjectMeta{
		Name:            game.Name,
		Namespace:       game.Namespace,
		Labels:          labels,
		OwnerReferences: owner,
	}

	// 获取对应的 Deployment，如果不存在就创建
	deploy := &appsv1.Deployment{}
	if err := r.Get(ctx, name, deploy); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		deploy = &appsv1.Deployment{
			ObjectMeta: meta,
			Spec:       getDeploymentSpec(game, labels),
		}
		if err = r.Create(ctx, deploy); err != nil {
			return err
		}
		logger.Info("create deployment success", "name", name.String())
	} else {
		// 存在，那么就对比一下 期望状态和实际状态
		want := getDeploymentSpec(game, labels)
		get := getSpecFromDeployment(deploy)

		if !reflect.DeepEqual(want, get) {
			// 调谐，将当前 Deployment 的 Spec 改成期望状态。然后让 Deployment Controller 去操作
			new := deploy.DeepCopy()
			new.Spec = want
			if err := r.Update(ctx, new); err != nil {
				return err
			}
			logger.Info("update deployment success", "name", name.String())
		}
	}

	// 获取对应的 Service，如果不存在就创建
	svc := &corev1.Service{}
	if err := r.Get(ctx, name, svc); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		svc = &corev1.Service{
			ObjectMeta: meta,
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{
						Port:       int32(port),
						TargetPort: intstr.FromInt32(port),
						Protocol:   corev1.ProtocolTCP,
					},
				},
				Selector: labels,
			},
		}
		if err := r.Create(ctx, svc); err != nil {
			return err
		}
		logger.Info("create service success", "name", name.String())
	}

	// 获取对应的 Ingress，如果不存在就创建
	ing := &networkingv1.Ingress{}
	if err := r.Get(ctx, name, ing); err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		ing = &networkingv1.Ingress{
			ObjectMeta: meta,
			Spec:       getIngressSpec(game, labels),
		}
		if err := r.Create(ctx, ing); err != nil {
			return err
		}
		logger.Info("create ingress success", "name", name.String())
	}

	newStatus := myappv1.GameStatus{
		Replicas:      *game.Spec.Replicas,
		ReadyReplicas: deploy.Status.ReadyReplicas,
	}

	// 如果 Ready 的副本数和期望的一样，标记 Game 资源的状态为 Running
	// k get statefulset redis-c-3pe4jn-redis-cluster -n redis-cluster
	// NAME                           READY   AGE
	// redis-c-3pe4jn-redis-cluster   6/6     222d
	if newStatus.Replicas == newStatus.ReadyReplicas {
		newStatus.Phase = myappv1.Running
	} else {
		newStatus.Phase = myappv1.NotReady
	}

	// 最后对比一下 Status，如果状态发生变化，也会触发调谐
	if !reflect.DeepEqual(game.Status, newStatus) {
		game.Status = newStatus
		logger.Info("update game status", "name", name.String())
		return r.Client.Status().Update(ctx, game)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GameReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 3,
		}).
		For(&myappv1.Game{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

// 获取 Deployment 的期望状态
func getDeploymentSpec(game *myappv1.Game, labels map[string]string) appsv1.DeploymentSpec {
	return appsv1.DeploymentSpec{
		Replicas: game.Spec.Replicas,
		Selector: metav1.SetAsLabelSelector(labels),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "main",
						Image: game.Spec.Image,
					},
				},
			},
		},
	}
}

// 获取 Deployment 的实际运行状态
func getSpecFromDeployment(deploy *appsv1.Deployment) appsv1.DeploymentSpec {
	container := deploy.Spec.Template.Spec.Containers[0]
	return appsv1.DeploymentSpec{
		Replicas: deploy.Spec.Replicas,
		Selector: deploy.Spec.Selector,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: deploy.Spec.Template.Labels,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  container.Name,
						Image: container.Image,
					},
				},
			},
		},
	}
}

// 获取 Ingress 的期望状态
func getIngressSpec(game *myappv1.Game, labels map[string]string) networkingv1.IngressSpec {
	pathType := networkingv1.PathTypePrefix
	return networkingv1.IngressSpec{
		Rules: []networkingv1.IngressRule{
			{
				Host: game.Spec.Host,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								PathType: &pathType,
								Path:     "/",
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: game.Name,
										Port: networkingv1.ServiceBackendPort{
											Number: int32(port),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
