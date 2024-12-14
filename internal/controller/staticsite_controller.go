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

// +kubebuilder:rbac:groups=cv.good-coffee-lover.io,resources=staticsites;deployments;configmaps;services,verbs=get;list;watch;create;update;patch;delete
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
	ss := &cvv1alpha1.StaticSite{}
	if err := r.Get(ctx, req.NamespacedName, ss); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return ctrl.Result{}, nil
		}

		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	err := r.synsStatus(ctx, ss)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("presync status: %w", err)
	}

	err = r.ensureResources(ctx, ss)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("ensure resources: %w", err)
	}

	err = r.synsStatus(ctx, ss)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("sync status: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *StaticSiteReconciler) ensureResources(ctx context.Context, ss *cvv1alpha1.StaticSite) error {
	resorces := []client.Object{
		r.siteConfigMap(ctx, ss),
		r.siteDeployment(ctx, ss),
		r.siteService(ctx, ss),
	}
	for _, res := range resorces {
		gvk, unversioned, err := r.Scheme.ObjectKinds(res)
		if err != nil {
			return err
		}
		if unversioned || len(gvk) == 0 {
			return fmt.Errorf("failed to get versions of %s/%s", res.GetNamespace(), res.GetName())
		}
		res.GetObjectKind().SetGroupVersionKind(gvk[0])

		l := log.FromContext(ctx, "object.kind", res.GetObjectKind(), "object.name", res.GetName())
		l.Info("creating resource")
		err = ctrl.SetControllerReference(ss, res, r.Scheme)
		if err != nil {
			return fmt.Errorf("set controller reference: %w", err)
		}

		err = r.Patch(ctx, res, client.Apply,
			client.ForceOwnership, client.FieldOwner("static-site-controller"),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *StaticSiteReconciler) synsStatus(ctx context.Context, ss *cvv1alpha1.StaticSite) error {
	deployment := r.siteDeployment(ctx, ss)
	err := r.Get(ctx, client.ObjectKeyFromObject(deployment), deployment)
	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get deployment %v/%v: %w", deployment.Namespace, deployment.Name, err)
	}
	ss.ManagedFields = nil
	ss.Status.Replicas = deployment.Status.Replicas
	// ss.Status.Conditions = append(ss.Status.Conditions, metav1.Condition{})
	return r.Status().Patch(ctx, ss, client.Apply,
		client.ForceOwnership, client.FieldOwner("static-site-controller"),
	)
}

func (r *StaticSiteReconciler) siteDeployment(ctx context.Context, ss *cvv1alpha1.StaticSite) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ss.Name + "-dpl",
			Namespace: ss.Namespace,
			Labels:    r.siteLabels(ctx, ss),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &ss.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: r.siteLabels(ctx, ss),
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: r.siteLabels(ctx, ss),
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
										Name: r.siteConfigMap(ctx, ss).Name,
									},
									Items: []v1.KeyToPath{
										// {
										// 	Key:  "index",
										// 	Path: "index.html",
										// },
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, page := range ss.Spec.Pages {
		deployment.Spec.Template.Spec.Volumes[0].ConfigMap.Items = append(deployment.Spec.Template.Spec.Volumes[0].ConfigMap.Items, v1.KeyToPath{
			Key:  page.Path,
			Path: page.Path,
		})
	}
	return deployment
}

func (r *StaticSiteReconciler) siteConfigMap(ctx context.Context, ss *cvv1alpha1.StaticSite) *v1.ConfigMap {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ss.Name + "-cm",
			Namespace: ss.Namespace,
			Labels:    r.siteLabels(ctx, ss),
		},
		Data: map[string]string{},
	}
	for _, page := range ss.Spec.Pages {
		cm.Data[page.Path] = page.Content
	}
	return cm
}

func (r *StaticSiteReconciler) siteService(ctx context.Context, ss *cvv1alpha1.StaticSite) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ss.Name + "-svc",
			Namespace: ss.Namespace,
			Labels:    r.siteLabels(ctx, ss),
		},
		Spec: v1.ServiceSpec{
			Selector: r.siteLabels(ctx, ss),
			Type:     v1.ServiceTypeNodePort,
			Ports: []v1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.IntOrString{IntVal: 80},
					// FIXME: remove this field
					NodePort: 32000,
					Protocol: v1.ProtocolTCP,
				},
			},
		},
	}
}

func (r *StaticSiteReconciler) siteLabels(ctx context.Context, ss *cvv1alpha1.StaticSite) map[string]string {
	labels := make(map[string]string, len(ss.Labels))
	for k, v := range ss.Labels {
		labels[k] = v
	}

	if _, exists := labels["site"]; !exists {
		labels["site"] = ss.Name
	}
	return labels
}

// SetupWithManager sets up the controller with the Manager.
func (r *StaticSiteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cvv1alpha1.StaticSite{}).
		Named("staticsite").
		Owns(&v1.ConfigMap{}).
		Owns(&v1.Service{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
