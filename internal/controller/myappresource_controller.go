package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	myv1alpha1 "github.com/domenicbove/angi/api/v1alpha1"
)

// MyAppResourceReconciler reconciles a MyAppResource object
type MyAppResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=my.api.group,resources=myappresources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=my.api.group,resources=myappresources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=my.api.group,resources=myappresources/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=list;watch;get;patch;create;update
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=services,verbs=list;watch;get;patch;create;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *MyAppResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// get myappresource cr
	var myAppResource myv1alpha1.MyAppResource
	if err := r.Get(ctx, req.NamespacedName, &myAppResource); err != nil {
		log.Error(err, "unable to fetch MyAppResource")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// get existing podinfo deployment
	deployment := appsv1.Deployment{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: myAppResource.Namespace, Name: myAppResource.Name}, &deployment)
	if errors.IsNotFound(err) {
		// if it does not exist, create in next step
		deployment = *constructPodInfoDeployment(myAppResource)
	}
	if client.IgnoreNotFound(err) != nil {
		log.Error(err, "failed to get PodInfo Deployment for MyAppResource")
		return ctrl.Result{}, err
	}

	// get deployment spec updates
	newDeployment := *constructPodInfoDeployment(myAppResource)
	specr := deploymentSpecr(&deployment, newDeployment.Spec)

	if operation, err := controllerutil.CreateOrUpdate(ctx, r.Client, &deployment, specr); err != nil {
		log.Error(err, "unable to create or update PodInfo Deployment for MyAppResource", "deployment", deployment.Name)
		return ctrl.Result{}, err
	} else {
		log.V(1).Info(fmt.Sprintf("%s PodInfo Deployment for MyAppResource", operation), "deployment", deployment.Name)
	}

	// update the CR status
	if myAppResource.Status.PodInfoReadyReplicas != deployment.Status.ReadyReplicas {

		log.V(1).Info("updating MyAppResource status", "myappresource", myAppResource.Name)

		myAppResource.Status.PodInfoReadyReplicas = deployment.Status.ReadyReplicas
		if r.Client.Status().Update(ctx, &myAppResource); err != nil {
			log.Error(err, "failed to update MyAppResource status", "myappresource", myAppResource.Name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func deploymentSpecr(deploy *appsv1.Deployment, spec appsv1.DeploymentSpec) controllerutil.MutateFn {
	return func() error {
		deploy.Spec = spec
		return nil
	}
}

func constructPodInfoDeployment(myAppResource myv1alpha1.MyAppResource) *appsv1.Deployment {
	image := fmt.Sprintf("%s:%s", myAppResource.Spec.Image.Repository, myAppResource.Spec.Image.Tag)

	depl := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:            myAppResource.Name,
			Namespace:       myAppResource.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(&myAppResource, myv1alpha1.GroupVersion.WithKind("MyAppResource"))},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: myAppResource.Spec.ReplicaCount,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": myAppResource.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": myAppResource.Name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "podinfo",
							Image: image,
							Env: []corev1.EnvVar{
								// {Name: "PODINFO_CACHE_SERVER", Value: redis.Status.RedisServiceName},
								{Name: "PODINFO_UI_COLOR", Value: myAppResource.Spec.UI.Color},
								{Name: "PODINFO_UI_MESSAGE", Value: myAppResource.Spec.UI.Message},
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 9898, Name: "http", Protocol: "TCP"},
							},
							// Resources: *watchlist.Spec.Frontend.Resources.DeepCopy(),
						},
					},
				},
			},
		},
	}

	// if err := ctrl.SetControllerReference(&myAppResource, depl, r.Scheme); err != nil {
	// 	return nil, err
	// }

	return depl
}

var (
	jobOwnerKey = ".metadata.controller"
	apiGVStr    = myv1alpha1.GroupVersion.String()
)

// SetupWithManager sets up the controller with the Manager.
func (r *MyAppResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &appsv1.Deployment{}, jobOwnerKey, func(rawObj client.Object) []string {
		// grab the deployment object, extract the owner...
		deployment := rawObj.(*appsv1.Deployment)
		owner := metav1.GetControllerOf(deployment)
		if owner == nil {
			return nil
		}
		// ...make sure it's a MyAppResource...
		if owner.APIVersion != apiGVStr || owner.Kind != "MyAppResource" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&myv1alpha1.MyAppResource{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
