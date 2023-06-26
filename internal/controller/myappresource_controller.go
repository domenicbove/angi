package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
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

	"github.com/domenicbove/angi/api/v1alpha1"
	"github.com/domenicbove/angi/internal/podinfo"
	"github.com/domenicbove/angi/internal/redis"
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
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=list;watch;get;patch;create;update
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups=core,resources=services,verbs=list;watch;get;patch;create;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *MyAppResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// get myappresource cr
	var myAppResource v1alpha1.MyAppResource
	if err := r.Get(ctx, req.NamespacedName, &myAppResource); err != nil {
		log.Error(err, "unable to fetch MyAppResource")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// in the case someone disables redis after enabling it, it should be cleaned up
	if myAppResource.Spec.Redis == nil || !myAppResource.Spec.Redis.Enabled {
		name := redis.GetDeploymentName(myAppResource.Name)
		lookupKey := client.ObjectKey{Namespace: myAppResource.Namespace, Name: name}

		redisDeployment := appsv1.Deployment{}
		err := r.Client.Get(ctx, lookupKey, &redisDeployment)
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "unable to fetch Redis Deployment")
		}
		if err == nil {
			// deployment was fetched successfully, should be deleted
			if err := r.Delete(ctx, &redisDeployment); err != nil {
				return ctrl.Result{}, err
			}
			log.V(1).Info("deleted Deployment for MyAppResource", "myappresource", myAppResource.Name, "deployment", name)
		}

		redisService := corev1.Service{}
		err = r.Client.Get(ctx, lookupKey, &redisService)
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "unable to fetch Redis Service")
		}
		if err == nil {
			// service was fetched successfully, should be deleted
			if err := r.Delete(ctx, &redisService); err != nil {
				return ctrl.Result{}, err
			}
			log.V(1).Info("deleted Service for MyAppResource", "myappresource", myAppResource.Name, "service", name)
		}
	}

	// create or update the redis deployment and service
	var redisDeployment *appsv1.Deployment
	if myAppResource.Spec.Redis != nil && myAppResource.Spec.Redis.Enabled {

		redisName := redis.GetDeploymentName(myAppResource.Name)

		var err error
		redisDeployment, err = r.createOrUpdateDeployment(ctx, redisName, myAppResource.Namespace,
			redis.ConstructRedisDeployment(myAppResource), log)

		if err != nil {
			return ctrl.Result{}, err
		}

		if err := r.createOrUpdateService(ctx, redisName, myAppResource.Namespace,
			redis.ConstructRedisService(myAppResource), log); err != nil {

			return ctrl.Result{}, err
		}
	}

	// create or update the podInfo deployment
	podInfoDeployment, err := r.createOrUpdateDeployment(ctx, myAppResource.Name,
		myAppResource.Namespace, podinfo.ConstructPodInfoDeployment(myAppResource), log)

	if err != nil {
		return ctrl.Result{}, err
	}

	// update the CR status
	updateStatus := false
	if redisDeployment != nil && redisDeployment.Status.ReadyReplicas != myAppResource.Status.RedisReadyReplicas {
		updateStatus = true
		myAppResource.Status.RedisReadyReplicas = redisDeployment.Status.ReadyReplicas
	}
	if myAppResource.Status.PodInfoReadyReplicas != podInfoDeployment.Status.ReadyReplicas {
		updateStatus = true
		myAppResource.Status.PodInfoReadyReplicas = podInfoDeployment.Status.ReadyReplicas
	}

	if updateStatus {
		log.V(1).Info("updating MyAppResource status", "myappresource", myAppResource.Name)

		if r.Client.Status().Update(ctx, &myAppResource); err != nil {
			log.Error(err, "failed to update MyAppResource status", "myappresource", myAppResource.Name)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *MyAppResourceReconciler) createOrUpdateDeployment(ctx context.Context, name, namespace string, updatedDeployment *appsv1.Deployment, log logr.Logger) (*appsv1.Deployment, error) {
	// get existing deployment
	deployment := appsv1.Deployment{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &deployment)
	if errors.IsNotFound(err) {
		// if it does not exist, create in next step
		deployment = *updatedDeployment
	}
	if client.IgnoreNotFound(err) != nil {
		log.Error(err, "failed to get Deployment for MyAppResource", "myappresource", name, "deployment", deployment.Name)
		return nil, err
	}

	specr := deploymentSpecr(&deployment, updatedDeployment.Spec)

	if operation, err := controllerutil.CreateOrUpdate(ctx, r.Client, &deployment, specr); err != nil {
		log.Error(err, "unable to create or update Deployment for MyAppResource", "myappresource", name, "deployment", deployment.Name)
		return nil, err
	} else {
		log.V(1).Info(fmt.Sprintf("%s Deployment for MyAppResource", operation), "myappresource", name, "deployment", deployment.Name)
	}

	return &deployment, nil
}

func deploymentSpecr(deploy *appsv1.Deployment, spec appsv1.DeploymentSpec) controllerutil.MutateFn {
	return func() error {
		deploy.Spec = spec
		return nil
	}
}

func (r *MyAppResourceReconciler) createOrUpdateService(ctx context.Context, name, namespace string, updatedService *corev1.Service, log logr.Logger) error {
	// get existing service
	service := corev1.Service{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &service)
	if errors.IsNotFound(err) {
		// if it does not exist, create in next step
		service = *updatedService
	}
	if client.IgnoreNotFound(err) != nil {
		log.Error(err, "failed to get Service for MyAppResource", "myappresource", name, "service", service.Name)
		return err
	}

	specr := serviceSpecr(&service, updatedService.Spec)

	if operation, err := controllerutil.CreateOrUpdate(ctx, r.Client, &service, specr); err != nil {
		log.Error(err, "unable to create or update Service for MyAppResource", "myappresource", name, "service", service.Name)
		return err
	} else {
		log.V(1).Info(fmt.Sprintf("%s Service for MyAppResource", operation), "myappresource", name, "service", service.Name)
	}

	return nil
}

func serviceSpecr(service *corev1.Service, spec corev1.ServiceSpec) controllerutil.MutateFn {
	return func() error {
		service.Spec = spec
		return nil
	}
}

var (
	jobOwnerKey = ".metadata.controller"
	apiGVStr    = v1alpha1.GroupVersion.String()
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
		For(&v1alpha1.MyAppResource{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
