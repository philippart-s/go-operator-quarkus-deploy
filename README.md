# go-operator-quarkus-deploy
Operator written in Go to deploy a Quarkus application

## 🎉 Init project
 - la branche `01-init-project` contient le résultat de cette étape
 - [installer / mettre](https://sdk.operatorframework.io/docs/installation/) à jour la dernière version du [Operator SDK](https://sdk.operatorframework.io/) (v1.27.0 au moment de l'écriture du readme)
 - créer le répertoire `go-operator-quarkus-deploy`
 - dans le répertoire `go-operator-quarkus-deploy`, scaffolding du projet : `operator-sdk init --domain wilda.fr --repo github.com/philippart-s/go-operator-quarkus-deploy`
 - A ce stage une arborescence complète a été générée, notamment la partie configuration dans `config` et un `Makefile` permettant le lancement des différentes commandes de build
 - vérification que cela compile : `go build`

 ## 📄 CRD generation
 - la branche `02-crd-generation` contient le résultat de cette étape
 - création de l'API : `operator-sdk create api --version v1 --kind QuarkusOperator --resource --controller`
 - de nouveau, de nombreux fichiers de générés, notamment le controller `./controllers/quarkusoperator_controller.go`
 - ensuite on génère la CRD `./config/crd/bases/wilda.fr_quarkusoperators.yaml` avec la commande `make manifests`
  - puis on peut l'appliquer avec la commande `make install`
  - et vérfier qu'elle a été créée : `kubectl get crd quarkusoperators.wilda.fr`
```bash
$ kubectl get crd quarkusoperators.wilda.fr

NAME                         CREATED AT
quarkusoperators.wilda.fr   2022-09-01T06:43:36Z
```

## 👋  Hello World
 - la branche `03-hello-world` contient le résultat de cette étape
 - modifier le fichier `api/v1/quarkusoperator_types.go`:
```go
package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// QuarkusOperatorSpec defines the desired state of QuarkusOperator
type QuarkusOperatorSpec struct {
	// Image version for the quarkus hello world image
	ImageVersion string `json:"imageVersion"`
	// Exposed port
	Port int32 `json:"port"`
}

// Unchanged code
// ...
```
 - générer la CRD modifiée : `make manifests`
 - deployer la CRD dans Kubernetes : `make install`
 - vérifier que la CRD a bien été mise à jour:
```bash
$ kubectl get crds quarkusoperators.wilda.fr -o json | jq '.spec.versions[0].schema.openAPIV3Schema.properties.spec'
{
  "description": "QuarkusOperatorSpec defines the desired state of QuarkusOperator",
  "properties": {
    "imageVersion": {
      "description": "Image version for the quarkus hello world image",
      "type": "string"
    },
    "port": {
      "description": "Exposed port",
      "format": "int32",
      "type": "integer"
    }
  },
  "required": [
    "imageVersion",
    "port"
  ],
  "type": "object"
}
```
 - modifier le reconciler en ajoutant les fonctions permettant la création d'un Deployment et d'un Service dans `controllers/quarkusoperator_controller.go`:
```go
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
```
 - modifier le reconciler pour la création du Pod et de son Service dans `controllers/quarkusoperator_controller.go`:
```go
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

// Unchanged code
// ...

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

// Unchanged code
// ...
```
 - vérifier que ça compile : `go build`
 - créer le namespace `test-helloworld-operator`: `kubectl create ns test-helloworld-operator`
 - modifier la CR: `config/samples/_v1_quarkusoperator.yaml`:
```yaml
apiVersion: wilda.fr/v1
kind: QuarkusOperator
metadata:
  labels:
    app.kubernetes.io/name: quarkusoperator
    app.kubernetes.io/instance: quarkusoperator-sample
    app.kubernetes.io/part-of: go-operator-quarkus-deploy
    app.kubernetes.io/managed-by: kustomize
    app.kubernetes.io/created-by: go-operator-quarkus-deploy
  name: quarkusoperator-sample
spec:
  imageVersion: 1.0.0
  port: 30080
```
  - lancer l'opérateur en mode dev : `make install run`
	- appliquer la CR : `kubectl apply -f ./config/samples/_v1_quarkusoperator.yaml -n test-helloworld-operator`
  - l'opérateur devrait créer le pod Quarkus et son service:
```bash
INFO    🚀 Deploy Quarkus application 1.0.0 on port 30080 in namespace test-helloworld-operator!        {"controller": "quarkusoperator", "controllerGroup": "wilda.fr", "controllerKind": "QuarkusOperator", "QuarkusOperator": {"name":"quarkusoperator-sample","namespace":"test-helloworld-operator"}, "namespace": "test-helloworld-operator", "name": "quarkusoperator-sample", "reconcileID": "55d63c99-6199-41ca-b574-6a02a5ef7938"}
```
      Dans Kubernetes:
```bash
$  kubectl get pod,svc  -n test-helloworld-operator
NAME                                     READY   STATUS    RESTARTS   AGE
pod/quarkus-deployment-5c56cbc47-6rxg5   1/1     Running   0          70s

NAME                      TYPE       CLUSTER-IP    EXTERNAL-IP   PORT(S)        AGE
service/quarkus-service   NodePort   X.XX.XXX.XX   <none>        80:30080/TCP   70s
```
 - tester dans un navigateur ou par un curl l'accès à `http://<node external ip>:30080/hello`, pour récupérer l'IP externe du node : `kubectl cluster-info`
```bash
$ curl http://ptgtl8.nodes.c1.gra7.k8s.ovh.net:30080/hello
👋  Hello, World ! 🌍
```
 - supprimer la CR : `kubectl delete quarkusoperators.wilda.fr/quarkusoperator-sample -n test-helloworld-operator`
 - verifier que tout a été supprimé:
```bash
kubectl get pod,svc  -n test-helloworld-operator
No resources found in test-helloworld-operator namespace.
```
 - supprimer le namespace `test-helloworld-operator` : `kubectl delete ns test-helloworld-operator`

 ## 🐳 Packaging & deployment to K8s
 - la branche `04-package-deploy` contient le résultat de cette étape
 - modifier le controller `controllers/quarkusoperator_controller.go` pour les droits:
```go
// unmodified code ...

//+kubebuilder:rbac:groups=wilda.fr,resources=quarkusoperators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=wilda.fr,resources=quarkusoperators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=wilda.fr,resources=quarkusoperators/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete

// unmodified code ...
```
 - générer les RBAC dans `config/rbac/role.yaml` : `make manifests`
 - modifier le Makefile:
```makefile
## unmodified code ...

IMAGE_TAG_BASE ?= wilda/go-operator-samples

## unmodified code ...

IMG ?= $(IMAGE_TAG_BASE):$(VERSION)

## unmodified code ...

.PHONY: docker-build
docker-build: #test ## Build docker image with the manager.
	docker build -t ${IMG} .

## unmodified code ...
```
 - lancer la création de l'image: `make docker-build`
 - s'authentifier sur le docker hub : `docker login`
 - push de l'image : `make docker-push`
 - déployer l'opérateur dans Kubernetes : `make deploy`:
```bash
$ kubectl get deployment -n go-operator-quarkus-deploy-system

NAME                                      			 READY   UP-TO-DATE   AVAILABLE   AGE
go-operator-quarkus-deploy-controller-manager     1/1     1            1           79s
```
 - créer le namespace `test-helloworld-operator` : `kubectl create ns test-helloworld-operator`
 - appliquer la CR : `kubectl apply -f ./config/samples/_v1_quarkusoperator.yaml -n test-helloworld-operator`
 - vérifier que l'opérateur a fait le nécessaire: `kubectl get pod,svc  -n test-helloworld-operator`
```bash
$ kubectl get pod,svc  -n test-helloworld-operator
NAME                                     READY   STATUS    RESTARTS   AGE
pod/quarkus-deployment-5c56cbc47-4mqt2   1/1     Running   0          19s

NAME                      TYPE       CLUSTER-IP   EXTERNAL-IP   PORT(S)        AGE
service/quarkus-service   NodePort   X.X.X.X    <none>        80:30080/TCP   19s
```
 - tester dans un navigateur ou par un curl l'accès à `http://<node external ip>:30080/hello`, pour récupérer l'IP externe du node : `kubectl cluster-info`
```bash
$ curl http://xxxx.nodes.c1.xxxx.k8s.ovh.net:30080/hello
👋  Hello, World ! 🌍
```
 - supprimer la CR : `kubectl delete quarkusoperators.wilda.fr/quarkusoperator-sample -n test-helloworld-operator` 
 - undeploy de l'opérateur : `make undeploy`
 - supprimer le namespace `test-helloworld-operator` : `kubectl delete ns test-helloworld-operator`