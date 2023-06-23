package redis

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/domenicbove/angi/api/v1alpha1"
)

const (
	RedisPort = 6379
)

func GetDeploymentName(myAppResourceName string) string {
	return fmt.Sprintf("%s-redis", myAppResourceName)
}

func GetEndpoint(myAppResourceName, namespace string) string {
	return fmt.Sprintf("tcp://%s.%s.svc.cluster.local:%d", GetDeploymentName(myAppResourceName), namespace, RedisPort)
}

func ConstructRedisDeployment(myAppResource v1alpha1.MyAppResource) *appsv1.Deployment {

	replicas := int32(1)
	name := GetDeploymentName(myAppResource.Name)

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{APIVersion: appsv1.SchemeGroupVersion.String(), Kind: "Deployment"},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       myAppResource.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(&myAppResource, v1alpha1.GroupVersion.WithKind("MyAppResource"))},
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
								{ContainerPort: RedisPort, Name: "redis", Protocol: "TCP"},
							},
						},
					},
				},
			},
		},
	}

	return deployment
}

func ConstructRedisService(myAppResource v1alpha1.MyAppResource) *corev1.Service {
	name := GetDeploymentName(myAppResource.Name)

	targetPort := intstr.IntOrString{
		IntVal: RedisPort,
	}

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{APIVersion: corev1.SchemeGroupVersion.String(), Kind: "Service"},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       myAppResource.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(&myAppResource, v1alpha1.GroupVersion.WithKind("MyAppResource"))},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{Name: "redis", Port: RedisPort, TargetPort: targetPort},
			},
			Selector: map[string]string{
				"app": name,
			},
		},
	}

	return service
}
