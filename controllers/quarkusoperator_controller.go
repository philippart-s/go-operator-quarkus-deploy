/*
Copyright 2023.

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

package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	wildafrv1 "github.com/philippart-s/go-operator-quarkus-deploy/api/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// QuarkusOperatorReconciler reconciles a QuarkusOperator object
type QuarkusOperatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=wilda.fr,resources=quarkusoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=wilda.fr,resources=quarkusoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=wilda.fr,resources=quarkusoperators/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the QuarkusOperator object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *QuarkusOperatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	const quarkusOperatorFinalizer = "wilda.fr/finalizer"
	quarkusDeployment := &appsv1.Deployment{}
	quarkusService := &corev1.Service{}
	customResource := &wildafrv1.QuarkusOperator{}

	err := r.Get(ctx, req.NamespacedName, customResource)
	if err != nil {
		if errors.IsNotFound(err) {
			// CR deleted, nothing to do
			log.Info("No CR found, nothing to do 🧐.")
		} else {
			// Error reading the object - requeue the request.
			log.Error(err, "Failed to get CR QuarkusOperator")
			return ctrl.Result{}, err
		}
	} else {
		// Add finalizer for this CR
		if !controllerutil.ContainsFinalizer(customResource, quarkusOperatorFinalizer) {
			controllerutil.AddFinalizer(customResource, quarkusOperatorFinalizer)
			err = r.Update(ctx, customResource)
			if err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}

		// Check if the CR is marked for deletion
		if customResource.GetDeletionTimestamp() != nil {
			// CR marked for deletion ➡️ Goodbye
			log.Info("Undeploy Quarkus application 🗑.")
			controllerutil.RemoveFinalizer(customResource, quarkusOperatorFinalizer)
			err := r.Update(ctx, customResource)
			if err != nil {
				log.Info("Error during deletion")
				return ctrl.Result{}, err
			}

			// Get the deployment to delete it
			err = r.Get(ctx, types.NamespacedName{Name: "quarkus-deployment", Namespace: customResource.Namespace}, quarkusDeployment)
			if err != nil {
				log.Error(err, "Failed to get Deployment")
				return ctrl.Result{}, err
			} else {
				// If a Deployment is found -> delete it
				log.Info("Delete Deployment 🗑.")
				err = r.Delete(ctx, quarkusDeployment)
				if err != nil {
					log.Error(err, "Failed to delete Deployment")
					return ctrl.Result{}, err
				}
			}

			// Get the service to delete it
			err = r.Get(ctx, types.NamespacedName{Name: "quarkus-service", Namespace: customResource.Namespace}, quarkusService)
			if err != nil {
				log.Error(err, "Failed to get Service")
				log.Info("Delete Service 🗑.")
				return ctrl.Result{}, err
			} else {
				// If a Service is found -> delete it
				err = r.Delete(ctx, quarkusService)
				if err != nil {
					log.Error(err, "Failed to delete Service")
					return ctrl.Result{}, err
				}
			}
		} else {
			// CR created -> deployQuarkus application
			log.Info(fmt.Sprintf("🚀 Deploy Quarkus application %s on port %d in namespace %s!", customResource.Spec.ImageVersion, customResource.Spec.Port, customResource.Namespace))
			err = r.Create(ctx, r.createDeployment(customResource, customResource.Namespace))
			if err != nil {
				log.Error(err, "❌ Failed to create new Deployment")
				return ctrl.Result{}, err
			}

			err = r.Create(ctx, r.createService(customResource, customResource.Namespace))
			if err != nil {
				log.Error(err, "❌ Failed to create new Service")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// Create a Deployment for the Quarkus Hello, World! application.
func (r *QuarkusOperatorReconciler) createDeployment(quarkusCR *wildafrv1.QuarkusOperator, namespace string) *appsv1.Deployment {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "quarkus-deployment",
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "quarkus"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "quarkus"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "wilda/hello-world-from-quarkus:" + quarkusCR.Spec.ImageVersion,
						Name:  "quarkus",
						Ports: []corev1.ContainerPort{{
							ContainerPort: 80,
							Name:          "http",
							Protocol:      "TCP",
						}},
					}},
				},
			},
		},
	}

	return deployment
}

// Create a Service for the hello world quarkus application.
func (r *QuarkusOperatorReconciler) createService(quarkusCR *wildafrv1.QuarkusOperator, namespace string) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "quarkus-service",
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "quarkus",
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					NodePort:   quarkusCR.Spec.Port,
					Port:       80,
					TargetPort: intstr.FromInt(8080),
				},
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	return service
}

// SetupWithManager sets up the controller with the Manager.
func (r *QuarkusOperatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wildafrv1.QuarkusOperator{}).
		Complete(r)
}
