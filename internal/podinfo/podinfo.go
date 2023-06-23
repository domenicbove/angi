package podinfo

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/domenicbove/angi/api/v1alpha1"
	"github.com/domenicbove/angi/internal/redis"
)

const (
	Port            = 9898
	UIColorEnvVar   = "PODINFO_UI_COLOR"
	UIMessageEnvVar = "PODINFO_UI_MESSAGE"
	CachEnvVar      = "PODINFO_CACHE_SERVER"
)

func ConstructPodInfoDeployment(myAppResource v1alpha1.MyAppResource) *appsv1.Deployment {
	image := fmt.Sprintf("%s:%s", myAppResource.Spec.Image.Repository, myAppResource.Spec.Image.Tag)

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:            myAppResource.Name,
			Namespace:       myAppResource.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(&myAppResource, v1alpha1.GroupVersion.WithKind("MyAppResource"))},
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
								{Name: UIColorEnvVar, Value: myAppResource.Spec.UI.Color},
								{Name: UIMessageEnvVar, Value: myAppResource.Spec.UI.Message},
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: Port, Name: "http", Protocol: "TCP"},
							},
							// Resources: *watchlist.Spec.Frontend.Resources.DeepCopy(),
						},
					},
				},
			},
		},
	}

	// add the redis env var if redis enabled
	if myAppResource.Spec.Redis != nil && myAppResource.Spec.Redis.Enabled {
		deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env,
			corev1.EnvVar{Name: CachEnvVar, Value: redis.GetEndpoint(myAppResource.Name, myAppResource.Namespace)})
	}

	return deployment
}
