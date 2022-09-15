/*
Copyright 2021.

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

package gateway

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	securityv1 "github.com/Layer7-Community/layer7-operator/api/v1"
	"github.com/Layer7-Community/layer7-operator/pkg/gateway"
	"github.com/Layer7-Community/layer7-operator/pkg/gateway/config"
	"github.com/Layer7-Community/layer7-operator/pkg/gateway/hpa"
	"github.com/Layer7-Community/layer7-operator/pkg/gateway/ingress"
	"github.com/Layer7-Community/layer7-operator/pkg/gateway/secrets"
	"github.com/Layer7-Community/layer7-operator/pkg/gateway/service"
	"github.com/Layer7-Community/layer7-operator/pkg/gateway/util"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GatewayReconciler reconciles a Gateway object
type GatewayReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// //+kubebuilder:rbac:groups=security.brcmlabs.com,namespace=default,resources=gateways,verbs=get;list;watch;create;update;patch;delete
// //+kubebuilder:rbac:groups=security.brcmlabs.com,namespace=default,resources=gateways/status,verbs=get;update;patch
// //+kubebuilder:rbac:groups=security.brcmlabs.com,namespace=default,resources=gateways/finalizers,verbs=update
// //+kubebuilder:rbac:groups=apps,namespace=default,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// //+kubebuilder:rbac:groups=core,namespace=default,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// //+kubebuilder:rbac:groups=core,namespace=default,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// //+kubebuilder:rbac:groups=core,namespace=default,resources=services,verbs=get;list;watch;create;update;patch;delete
// //+kubebuilder:rbac:groups=core,namespace=default,resources=pods,verbs=get;list;watch;create;update;patch;delete
// //+kubebuilder:rbac:groups=batch,namespace=default,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// //+kubebuilder:rbac:groups=networking.k8s.io,namespace=default,resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// //+kubebuilder:rbac:groups=autoscaling,namespace=default,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Gateway object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile

func (r *GatewayReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	_ = r.Log.WithValues("gateway", req.NamespacedName)

	gw := &securityv1.Gateway{}

	err := r.Get(ctx, req.NamespacedName, gw)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	gatewayLicense := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: gw.Spec.License.SecretName, Namespace: gw.Namespace}, gatewayLicense)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			r.Log.Error(err, "License not found", "Name", gw.Name, "Namespace", gw.Namespace)
			if err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

	err = reconcileConfigMap(r, gw.Name, ctx, gw)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = reconcileConfigMap(r, gw.Name+"-system", ctx, gw)
	if err != nil {
		return ctrl.Result{}, err
	}

	if gw.Spec.App.ClusterProperties.Enabled {
		err = reconcileConfigMap(r, gw.Name+"-cwp-bundle", ctx, gw)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if gw.Spec.App.ListenPorts.Harden {
		err = reconcileConfigMap(r, gw.Name+"-listen-port-bundle", ctx, gw)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if gw.Spec.App.Management.SecretName != "" {
		gatewaySecret := &corev1.Secret{}
		err = r.Get(ctx, types.NamespacedName{Name: gw.Spec.App.Management.SecretName, Namespace: gw.Namespace}, gatewaySecret)
		if err != nil {
			if k8serrors.IsNotFound(err) {
				r.Log.Error(err, "Secret not found", "Name", gw.Name, "Namespace", gw.Namespace)

				if err != nil {
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, nil
			}
		}
	} else {
		err = reconcileSecret(r, ctx, gw)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	err = reconcileService(r, ctx, gw)
	if err != nil {
		return ctrl.Result{}, err
	}

	if gw.Spec.App.Management.Service.Enabled {
		err = reconcileManagementService(r, ctx, gw)
		if err != nil {
			r.Log.Error(err, "Failed creating Management Service", "Name", gw.Name, "Namespace", gw.Namespace)
			return ctrl.Result{}, err
		}

		err = tagManagementPod(r, ctx, gw)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if gw.Spec.App.Ingress.Enabled {
		err = reconcileIngress(r, ctx, gw)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if gw.Spec.App.Autoscaling.Enabled {
		err = reconcileHPA(r, ctx, gw)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	err = reconcileDeployment(r, ctx, gw)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = updateGatewayStatus(r, ctx, gw)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Second * 10}, err
	}

	if gw.Spec.App.Repository.Enabled {
		err = reconcileBundles(r, ctx, gw)
		if err != nil {
			return ctrl.Result{RequeueAfter: time.Second * 10}, err
		}
	}

	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

func reconcileBundles(r *GatewayReconciler, ctx context.Context, gw *securityv1.Gateway) error {
	commit, err := util.GetLatestCommit(gw.Spec.App.Repository.URL)
	if err != nil {
		return err
	}
	if gw.Status.CommitID == commit {
		return nil
	}

	gw.Status.CommitID = commit
	if err := r.Client.Status().Update(ctx, gw); err != nil {
		r.Log.Error(err, "Failed to update commit id", "Namespace", gw.Namespace, "Name", gw.Name)
		return err
	}

	return nil
}

func reconcileHPA(r *GatewayReconciler, ctx context.Context, gw *securityv1.Gateway) error {
	currHPA := &autoscalingv2.HorizontalPodAutoscaler{}

	err := r.Get(ctx, types.NamespacedName{Name: gw.Name, Namespace: gw.Namespace}, currHPA)
	newHpa := hpa.NewHPA(gw)
	if err != nil && k8serrors.IsNotFound(err) {
		r.Log.Info("Creating HPA", "Name", gw.Name, "Namespace", gw.Namespace)
		ctrl.SetControllerReference(gw, newHpa, r.Scheme)
		err = r.Create(ctx, newHpa)
		if err != nil {
			r.Log.Error(err, "Failed creating HPA", "Name", gw.Name, "Namespace", gw.Namespace)
			return err
		}
		return nil
	}

	if !reflect.DeepEqual(currHPA, newHpa) {
		ctrl.SetControllerReference(gw, newHpa, r.Scheme)
		return r.Update(ctx, newHpa)
	}

	return nil
}

func reconcileConfigMap(r *GatewayReconciler, name string, ctx context.Context, gw *securityv1.Gateway) error {
	currMap := &corev1.ConfigMap{}
	err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: gw.Namespace}, currMap)
	cm := config.NewConfigMap(gw, name)

	if err != nil && k8serrors.IsNotFound(err) {
		r.Log.Info("Creating ConfigMap", "Name", name, "Namespace", gw.Namespace)
		ctrl.SetControllerReference(gw, cm, r.Scheme)
		err = r.Create(ctx, cm)
		if err != nil {
			r.Log.Error(err, "Failed creating ConfigMap", "Name", gw.Name, "Namespace", gw.Namespace)
			return err
		}
		return nil
	}

	if !reflect.DeepEqual(currMap.Data, cm.Data) {
		ctrl.SetControllerReference(gw, cm, r.Scheme)
		return r.Update(ctx, cm)
	}

	return nil
}

func reconcileSecret(r *GatewayReconciler, ctx context.Context, gw *securityv1.Gateway) error {
	currSecret := &corev1.Secret{}
	secret := secrets.NewSecret(gw)
	err := r.Get(ctx, types.NamespacedName{Name: gw.Name, Namespace: gw.Namespace}, currSecret)
	if err != nil && k8serrors.IsNotFound(err) {
		r.Log.Info("Creating Secret", "Name", gw.Name, "Namespace", gw.Namespace)

		ctrl.SetControllerReference(gw, secret, r.Scheme)
		err = r.Create(ctx, secret)
		if err != nil {
			r.Log.Error(err, "Failed creating Secret", "Name", gw.Name, "Namespace", gw.Namespace)
			return err
		}
		return nil
	}

	if !reflect.DeepEqual(currSecret.Data, secret.Data) {
		ctrl.SetControllerReference(gw, secret, r.Scheme)
		return r.Update(ctx, secret)
	}
	return nil
}

func reconcileService(r *GatewayReconciler, ctx context.Context, gw *securityv1.Gateway) error {
	currService := &corev1.Service{}
	svc := service.NewService(gw)
	err := r.Get(ctx, types.NamespacedName{Name: gw.Name, Namespace: gw.Namespace}, currService)
	if err != nil && k8serrors.IsNotFound(err) {
		r.Log.Info("Creating Service", "Name", gw.Name, "Namespace", gw.Namespace)
		ctrl.SetControllerReference(gw, svc, r.Scheme)
		err = r.Create(ctx, svc)
		if err != nil {
			r.Log.Error(err, "Failed creating Service", "Name", gw.Name, "Namespace", gw.Namespace)
			return err
		}
		return nil
	}
	return nil
}

func reconcileManagementService(r *GatewayReconciler, ctx context.Context, gw *securityv1.Gateway) error {
	currService := &corev1.Service{}
	svc := service.NewManagementService(gw)
	err := r.Get(ctx, types.NamespacedName{Name: gw.Name + "-management-service", Namespace: gw.Namespace}, currService)
	if err != nil && k8serrors.IsNotFound(err) {
		r.Log.Info("Creating Management Service", "Name", gw.Name, "Namespace", gw.Namespace)
		ctrl.SetControllerReference(gw, svc, r.Scheme)
		err = r.Create(ctx, svc)
		if err != nil {
			r.Log.Error(err, "Failed creating Management Service", "Name", gw.Name, "Namespace", gw.Namespace)
			return err
		}
		return nil
	}
	return nil
}

func reconcileIngress(r *GatewayReconciler, ctx context.Context, gw *securityv1.Gateway) error {
	currIngress := &networkingv1.Ingress{}
	ingress := ingress.NewIngress(gw)
	err := r.Get(ctx, types.NamespacedName{Name: gw.Name, Namespace: gw.Namespace}, currIngress)
	if err != nil && k8serrors.IsNotFound(err) {
		r.Log.Info("Creating Ingress", "Name", gw.Name, "Namespace", gw.Namespace)
		ctrl.SetControllerReference(gw, ingress, r.Scheme)
		err = r.Create(ctx, ingress)
		if err != nil {
			r.Log.Error(err, "Failed creating Ingress", "Name", gw.Name, "Namespace", gw.Namespace)
			return err
		}
		return nil
	}

	if !reflect.DeepEqual(currIngress.Spec, ingress.Spec) {
		ctrl.SetControllerReference(gw, ingress, r.Scheme)
		return r.Update(ctx, ingress)
	}

	return nil
}

func reconcileDeployment(r *GatewayReconciler, ctx context.Context, gw *securityv1.Gateway) error {
	currDeployment := &appsv1.Deployment{}
	dep := gateway.NewDeployment(gw)
	err := r.Get(ctx, types.NamespacedName{Name: gw.Name, Namespace: gw.Namespace}, currDeployment)
	if err != nil && k8serrors.IsNotFound(err) {
		r.Log.Info("Creating Deployment", "Name", gw.Name, "Namespace", gw.Namespace)
		ctrl.SetControllerReference(gw, dep, r.Scheme)
		err = r.Create(ctx, dep)
		if err != nil {
			r.Log.Error(err, "Failed creating Deployment", "Name", gw.Name, "Namespace", gw.Namespace)
			return err
		}
		return nil
	}

	if gw.Spec.App.Autoscaling.Enabled {
		dep.Spec.Replicas = currDeployment.Spec.Replicas
	}

	update := false

	//TODO: Revisit this
	currContainerBytes, _ := json.Marshal(currDeployment.Spec.Template.Spec.Containers)
	newContainerBytes, _ := json.Marshal(dep.Spec.Template.Spec.Containers)

	if string(currContainerBytes) != string(newContainerBytes) {
		update = true
	}

	if currDeployment.Spec.Template.Spec.ServiceAccountName != dep.Spec.Template.Spec.ServiceAccountName {
		update = true
	}

	if update {
		r.Log.Info("Updating Deployment", "Name", gw.Name, "Namespace", gw.Namespace)
		ctrl.SetControllerReference(gw, dep, r.Scheme)
		return r.Update(ctx, dep)
	}
	return nil
}

func updateGatewayStatus(r *GatewayReconciler, ctx context.Context, gw *securityv1.Gateway) error {
	gatewayStatus := gw.Status

	dep := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: gw.Name, Namespace: gw.Namespace}, dep)
	if err != nil && k8serrors.IsNotFound(err) {
		return err
	}

	gatewayStatus.Host = gw.Spec.App.Management.Cluster.Hostname
	gatewayStatus.Image = gw.Spec.App.Image
	gatewayStatus.Version = gw.Spec.Version
	gatewayStatus.Gateway = []securityv1.GatewayState{}
	gatewayStatus.Replicas = dep.Status.Replicas
	gatewayStatus.Ready = dep.Status.ReadyReplicas
	gatewayStatus.State = "initializing"

	if dep.Status.ReadyReplicas == dep.Status.Replicas {
		gatewayStatus.State = "ready"
	}

	gatewayStatus.Conditions = dep.Status.Conditions

	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(gw.Namespace),
		client.MatchingLabels(util.DefaultLabels(gw)),
	}

	if err := r.List(ctx, podList, listOpts...); err != nil {
		r.Log.Error(err, "Failed to list pods", "Namespace", gw.Namespace, "Name", gw.Name)
		return err
	}
	ready := false
	for p := range podList.Items {

		for cs := range podList.Items[p].Status.ContainerStatuses {
			if podList.Items[p].Status.ContainerStatuses[cs].Image == gw.Spec.App.Image {
				ready = podList.Items[p].Status.ContainerStatuses[cs].Ready
				// if ready {
				// 	util.RestCall("GET", podList.Items[p].Spec.)
				// }
			}
		}

		gatewayStatus.Gateway = append(gatewayStatus.Gateway, securityv1.GatewayState{
			Name:      podList.Items[p].Name,
			Phase:     podList.Items[p].Status.Phase,
			Ready:     ready,
			StartTime: podList.Items[p].Status.StartTime.String(),
		})
	}

	if !reflect.DeepEqual(gatewayStatus, gw.Status) {
		gw.Status = gatewayStatus
		return r.Client.Status().Update(ctx, gw)
	}

	return nil
}

func tagManagementPod(r *GatewayReconciler, ctx context.Context, gw *securityv1.Gateway) error {
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(gw.Namespace),
		client.MatchingLabels(util.DefaultLabels(gw)),
	}

	if err := r.List(ctx, podList, listOpts...); err != nil {
		r.Log.Error(err, "Failed to list pods", "Namespace", gw.Namespace, "Name", gw.Name)
		return err
	}
	podNames := getPodNames(podList.Items)
	if gw.Status.ManagementPod != "" {
		if util.Contains(podNames, gw.Status.ManagementPod) {
			return nil
		}
	}
	for p := range podList.Items {
		if p == 0 {
			patch := []byte(`{"metadata":{"labels":{"management-access": "leader"}}}`)
			if err := r.Client.Patch(context.Background(), &podList.Items[p],
				client.RawPatch(types.StrategicMergePatchType, patch)); err != nil {
				r.Log.Error(err, "Failed to update pod label", "Namespace", gw.Namespace, "Name", gw.Name)
				return err
			}

			gw.Status.ManagementPod = podList.Items[0].Name
			if err := r.Client.Status().Update(ctx, gw); err != nil {
				r.Log.Error(err, "Failed to update pod label", "Namespace", gw.Namespace, "Name", gw.Name)
				return err
			}
		}
	}
	return nil
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

// SetupWithManager sets up the controller with the Manager.
func (r *GatewayReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		//Watches(&source.Kind{Type: &securityv1.Gateway{}}, &handler.EnqueueRequestForObject{}).
		For(&securityv1.Gateway{}).
		Complete(r)
}
