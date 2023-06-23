package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
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
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=list;watch;get;patch;create;update
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups=core,resources=services,verbs=list;watch;get;patch;create;update

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

	var redisDeployment *appsv1.Deployment
	if myAppResource.Spec.Redis != nil && myAppResource.Spec.Redis.Enabled {
		redis, err := r.createOrUpdateDeployment(ctx, fmt.Sprintf("%s-redis", myAppResource.Name),
			myAppResource.Namespace, constructRedisDeployment(myAppResource))

		if err != nil {
			return ctrl.Result{}, err
		}
		redisDeployment = redis

		if err := r.createOrUpdateService(ctx, fmt.Sprintf("%s-redis", myAppResource.Name),
			myAppResource.Namespace, constructRedisService(myAppResource)); err != nil {

			return ctrl.Result{}, err
		}

	}

	// create or update the podInfoDeployment
	podInfoDeployment, err := r.createOrUpdateDeployment(ctx, myAppResource.Name,
		myAppResource.Namespace, constructPodInfoDeployment(myAppResource))

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

func (r *MyAppResourceReconciler) createOrUpdateDeployment(ctx context.Context, name, namespace string, updatedDeployment *appsv1.Deployment) (*appsv1.Deployment, error) {
	log := log.FromContext(ctx)

	// get existing podinfo deployment
	deployment := appsv1.Deployment{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &deployment)
	if errors.IsNotFound(err) {
		// if it does not exist, create in next step
		deployment = *updatedDeployment
	}
	if client.IgnoreNotFound(err) != nil {
		log.Error(err, "failed to get Deployment for MyAppResource", "deployment", deployment.Name)
		return nil, err
	}

	specr := deploymentSpecr(&deployment, updatedDeployment.Spec)

	if operation, err := controllerutil.CreateOrUpdate(ctx, r.Client, &deployment, specr); err != nil {
		log.Error(err, "unable to create or update Deployment for MyAppResource", "deployment", deployment.Name)
		return nil, err
	} else {
		log.V(1).Info(fmt.Sprintf("%s Deployment for MyAppResource", operation), "deployment", deployment.Name)
	}

	return &deployment, nil
}

func deploymentSpecr(deploy *appsv1.Deployment, spec appsv1.DeploymentSpec) controllerutil.MutateFn {
	return func() error {
		deploy.Spec = spec
		return nil
	}
}

func (r *MyAppResourceReconciler) createOrUpdateService(ctx context.Context, name, namespace string, updatedService *corev1.Service) error {
	log := log.FromContext(ctx)

	// get existing service
	service := corev1.Service{}
	err := r.Client.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &service)
	if errors.IsNotFound(err) {
		// if it does not exist, create in next step
		service = *updatedService
	}
	if client.IgnoreNotFound(err) != nil {
		log.Error(err, "failed to get service", "service", service.Name)
		return err
	}

	specr := serviceSpecr(&service, updatedService.Spec)

	if operation, err := controllerutil.CreateOrUpdate(ctx, r.Client, &service, specr); err != nil {
		log.Error(err, "unable to create or update Service for MyAppResource", "service", service.Name)
		return err
	} else {
		log.V(1).Info(fmt.Sprintf("%s Service for MyAppResource", operation), "service", service.Name)
	}

	return nil
}

// TODO try in code func definition
func serviceSpecr(service *corev1.Service, spec corev1.ServiceSpec) controllerutil.MutateFn {
	return func() error {
		service.Spec = spec
		return nil
	}
}

func constructPodInfoDeployment(myAppResource myv1alpha1.MyAppResource) *appsv1.Deployment {
	image := fmt.Sprintf("%s:%s", myAppResource.Spec.Image.Repository, myAppResource.Spec.Image.Tag)

	deployment := &appsv1.Deployment{
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

	return deployment
}

func constructRedisDeployment(myAppResource myv1alpha1.MyAppResource) *appsv1.Deployment {

	replicas := int32(1)
	name := fmt.Sprintf("%s-redis", myAppResource.Name)

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       myAppResource.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(&myAppResource, myv1alpha1.GroupVersion.WithKind("MyAppResource"))},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "redis",
							Image: "redis/redis-stack:latest",
							Ports: []corev1.ContainerPort{
								{ContainerPort: 6379, Name: "redis", Protocol: "TCP"},
							},
						},
					},
				},
			},
		},
	}

	return deployment
}

func constructRedisService(myAppResource myv1alpha1.MyAppResource) *corev1.Service {
	name := fmt.Sprintf("%s-redis", myAppResource.Name)

	targetPort := intstr.IntOrString{
		IntVal: 6379,
	}

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{APIVersion: corev1.SchemeGroupVersion.String(), Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       myAppResource.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(&myAppResource, myv1alpha1.GroupVersion.WithKind("MyAppResource"))},
		},
		// 		spec:
		//   type: ClusterIP
		//   ports:
		//   - name: redis
		//     port: 6379
		//     targetPort: 6379
		//   selector:
		//     app: redis
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Name: "redis", Port: 6379, TargetPort: targetPort},
			},
			Selector: map[string]string{
				"app": name,
			},
		},
	}

	return service
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
