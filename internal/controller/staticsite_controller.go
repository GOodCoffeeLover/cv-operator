/*
Copyright 2024 GoodCoffeeLover.

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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cvv1alpha1 "github.com/GoodCoffeeLover/cv-operator/api/v1alpha1"
)

// StaticSiteReconciler reconciles a StaticSite object
type StaticSiteReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cv.good-coffee-lover.io,resources=staticsites,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cv.good-coffee-lover.io,resources=staticsites/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cv.good-coffee-lover.io,resources=staticsites/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the StaticSite object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *StaticSiteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx = ctrl.LoggerInto(ctx, log.FromContext(ctx, "name", req.Name, "namespace", req.Namespace))

	log.FromContext(ctx).Info("got new object for reconcile")
	sp := &cvv1alpha1.StaticSite{}
	if err := r.Get(ctx, req.NamespacedName, sp); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	labels := map[string]string{"app": req.Name}
	err := r.Create(ctx, &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       sp.Kind,
					APIVersion: sp.APIVersion,
					Name:       sp.Name,
					UID:        sp.UID,
				},
			},
		},
		Spec: v1.ServiceSpec{
			Selector: labels,
			Type:     v1.ServiceTypeNodePort,
			Ports: []v1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.IntOrString{IntVal: 80},
					NodePort:   32000,
					Protocol:   v1.ProtocolTCP,
				},
			},
		},
	})
	if client.IgnoreAlreadyExists(err) != nil {
		return ctrl.Result{}, fmt.Errorf("create service: %w", err)
	}
	log.FromContext(ctx).Info("service created")

	err = r.Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       sp.Kind,
					APIVersion: sp.APIVersion,
					Name:       sp.Name,
					UID:        sp.UID,
				},
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32(3),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            "nginx",
							Image:           "nginx:latest",
							ImagePullPolicy: v1.PullIfNotPresent,
							Ports: []v1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 80,
									Protocol:      v1.ProtocolTCP,
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									MountPath: "/usr/share/nginx/html",
									ReadOnly:  true,
									Name:      "pages",
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "pages",
							VolumeSource: v1.VolumeSource{
								ConfigMap: &v1.ConfigMapVolumeSource{
									LocalObjectReference: v1.LocalObjectReference{
										Name: req.Name,
									},
									Items: []v1.KeyToPath{
										{
											Key:  "index",
											Path: "index.html",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	if client.IgnoreAlreadyExists(err) != nil {
		return ctrl.Result{}, fmt.Errorf("create deployment: %w", err)
	}
	log.FromContext(ctx).Info("deployment created", "name", req.Name)

	err = r.Create(ctx, &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind:       sp.Kind,
					APIVersion: sp.APIVersion,
					Name:       sp.Name,
					UID:        sp.UID,
				},
			},
		},
		Data: map[string]string{
			"index": sp.Spec.Content,
		},
	})
	if client.IgnoreAlreadyExists(err) != nil {
		return ctrl.Result{}, fmt.Errorf("create configmap: %w", err)
	}
	log.FromContext(ctx).Info("configmap created", "name", req.Name)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StaticSiteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cvv1alpha1.StaticSite{}).
		Named("staticsite").
		Complete(r)
}
