package controller

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/domenicbove/angi/api/v1alpha1"
	"github.com/domenicbove/angi/internal/podinfo"
	"github.com/domenicbove/angi/internal/redis"
)

var _ = Describe("MyAppResource controller", func() {

	const (
		MyAppResourceName      = "whatever"
		MyAppResourceNamespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	AfterEach(func() {
		lookupKey := types.NamespacedName{Name: MyAppResourceName, Namespace: MyAppResourceNamespace}

		// cleanup myappresource
		Eventually(func() error {
			myApp := &v1alpha1.MyAppResource{}
			k8sClient.Get(context.Background(), lookupKey, myApp)
			return k8sClient.Delete(context.Background(), myApp)
		}, timeout, interval).Should(Succeed())

		Eventually(func() error {
			myApp := &v1alpha1.MyAppResource{}
			return k8sClient.Get(context.Background(), lookupKey, myApp)
		}, timeout, interval).ShouldNot(Succeed())

		// cleanup podinfo deployment
		Eventually(func() error {
			podInfo := &appsv1.Deployment{}
			k8sClient.Get(context.Background(), lookupKey, podInfo)
			return k8sClient.Delete(context.Background(), podInfo)
		}, timeout, interval).Should(Succeed())

		Eventually(func() error {
			podInfo := &appsv1.Deployment{}
			return k8sClient.Get(context.Background(), lookupKey, podInfo)
		}, timeout, interval).ShouldNot(Succeed())
	})

	Context("When creating MyAppResource without Redis", func() {
		It("Should create subresources", func() {
			By("By creating a new MyAppResource")
			ctx := context.Background()

			replicas := int32(2)

			myAppResource := &v1alpha1.MyAppResource{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "my.api.group/v1alpha1",
					Kind:       "MyAppResource",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      MyAppResourceName,
					Namespace: MyAppResourceNamespace,
				},
				Spec: v1alpha1.MyAppResourceSpec{
					ReplicaCount: &replicas,
					Image: &v1alpha1.Image{
						Repository: "ghcr.io/stefanprodan/podinfo",
						Tag:        "latest",
					},
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							"cpu": resource.MustParse("100m"),
						},
						Limits: corev1.ResourceList{
							"memory": resource.MustParse("64Mi"),
						},
					},
					UI: v1alpha1.UI{
						Color:   "#34577c",
						Message: "some message",
					},
				},
			}

			Expect(k8sClient.Create(ctx, myAppResource)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: MyAppResourceName, Namespace: MyAppResourceNamespace}
			createdMyAppResource := &v1alpha1.MyAppResource{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdMyAppResource)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("By checking the created MyAppResource fields")
			Expect(createdMyAppResource.Spec.UI.Color).Should(Equal("#34577c"))
			Expect(createdMyAppResource.Spec.UI.Message).Should(Equal("some message"))

			Consistently(func() (int, error) {
				err := k8sClient.Get(ctx, lookupKey, createdMyAppResource)
				if err != nil {
					return -1, err
				}
				return int(createdMyAppResource.Status.PodInfoReadyReplicas), nil
			}, duration, interval).Should(Equal(0))

			By("By checking the podInfo deployment fields")
			podInfoDeployment := &appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, podInfoDeployment)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			// validate its fields!
			Expect(*podInfoDeployment.Spec.Replicas).Should(Equal(int32(2)))
			Expect(len(podInfoDeployment.Spec.Template.Spec.Containers)).Should(Equal(1))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Name).Should(Equal("podinfo"))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Image).Should(Equal("ghcr.io/stefanprodan/podinfo:latest"))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Resources.Limits.Memory().String()).Should(Equal("64Mi"))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Resources.Requests.Cpu().String()).Should(Equal("100m"))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Env).Should(ContainElement(
				corev1.EnvVar{Name: podinfo.UIColorEnvVar, Value: "#34577c"}))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Env).Should(ContainElement(
				corev1.EnvVar{Name: podinfo.UIMessageEnvVar, Value: "some message"}))
			Expect(podInfoDeployment.Status.ReadyReplicas).Should(Equal(int32(0)))

			By("By updating the podInfo deployment status")
			podInfoDeployment.Status.ReadyReplicas = int32(2)
			podInfoDeployment.Status.Replicas = int32(2)
			Expect(k8sClient.Status().Update(ctx, podInfoDeployment)).Should(Succeed())

			By("By checking the myappresource status updated")
			Eventually(func() (int, error) {
				err := k8sClient.Get(ctx, lookupKey, createdMyAppResource)
				if err != nil {
					return 0, err
				}

				return int(createdMyAppResource.Status.PodInfoReadyReplicas), nil
			}, timeout, interval).Should(Equal(2), "podInfoReadyReplicas in status should match the deployment")

		})

		It("Should create MyAppResourceName with Defaults", func() {
			By("By creating a new MyAppResourceName without optional fields")
			ctx := context.Background()

			myAppResource := &v1alpha1.MyAppResource{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "my.api.group/v1alpha1",
					Kind:       "MyAppResource",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      MyAppResourceName,
					Namespace: MyAppResourceNamespace,
				},
				Spec: v1alpha1.MyAppResourceSpec{
					UI: v1alpha1.UI{
						Color:   "#34577c",
						Message: "some message",
					},
				},
			}

			Expect(k8sClient.Create(ctx, myAppResource)).Should(Succeed())

			By("By checking the created MyAppResource fields")
			lookupKey := types.NamespacedName{Name: MyAppResourceName, Namespace: MyAppResourceNamespace}
			createdMyAppResource := &v1alpha1.MyAppResource{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdMyAppResource)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(createdMyAppResource.Spec.UI.Color).Should(Equal("#34577c"))
			Expect(*createdMyAppResource.Spec.ReplicaCount).Should(Equal(int32(1)))

			By("By checking the podInfo deployment fields")
			podInfoDeployment := &appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, podInfoDeployment)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			// validate its fields!
			Expect(*podInfoDeployment.Spec.Replicas).Should(Equal(int32(1)))
			Expect(len(podInfoDeployment.Spec.Template.Spec.Containers)).Should(Equal(1))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Name).Should(Equal("podinfo"))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Image).Should(Equal("ghcr.io/stefanprodan/podinfo:latest"))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Env).Should(ContainElement(
				corev1.EnvVar{Name: podinfo.UIColorEnvVar, Value: "#34577c"}))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Env).Should(ContainElement(
				corev1.EnvVar{Name: podinfo.UIMessageEnvVar, Value: "some message"}))
		})
	})

})

var _ = Describe("MyAppResource controller - Redis Enabled", func() {

	const (
		MyAppResourceName      = "whatever"
		MyAppResourceNamespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	AfterEach(func() {
		lookupKey := types.NamespacedName{Name: MyAppResourceName, Namespace: MyAppResourceNamespace}

		// cleanup myappresource
		Eventually(func() error {
			myApp := &v1alpha1.MyAppResource{}
			k8sClient.Get(context.Background(), lookupKey, myApp)
			return k8sClient.Delete(context.Background(), myApp)
		}, timeout, interval).Should(Succeed())

		Eventually(func() error {
			myApp := &v1alpha1.MyAppResource{}
			return k8sClient.Get(context.Background(), lookupKey, myApp)
		}, timeout, interval).ShouldNot(Succeed())

		// cleanup podinfo deployment
		Eventually(func() error {
			podInfo := &appsv1.Deployment{}
			k8sClient.Get(context.Background(), lookupKey, podInfo)
			return k8sClient.Delete(context.Background(), podInfo)
		}, timeout, interval).Should(Succeed())

		Eventually(func() error {
			podInfo := &appsv1.Deployment{}
			return k8sClient.Get(context.Background(), lookupKey, podInfo)
		}, timeout, interval).ShouldNot(Succeed())
	})

	Context("When creating MyAppResource with Redis", func() {

		It("Should create subresources", func() {
			By("By creating a new MyAppResourceName with Redis Enabled")
			ctx := context.Background()

			myAppResource := &v1alpha1.MyAppResource{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "my.api.group/v1alpha1",
					Kind:       "MyAppResource",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      MyAppResourceName,
					Namespace: MyAppResourceNamespace,
				},
				Spec: v1alpha1.MyAppResourceSpec{
					UI: v1alpha1.UI{
						Color:   "#34577c",
						Message: "some message",
					},
					Redis: &v1alpha1.Redis{
						Enabled: true,
					},
				},
			}

			Expect(k8sClient.Create(ctx, myAppResource)).Should(Succeed())

			By("By checking the created MyAppResource fields")
			lookupKey := types.NamespacedName{Name: MyAppResourceName, Namespace: MyAppResourceNamespace}
			createdMyAppResource := &v1alpha1.MyAppResource{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdMyAppResource)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(createdMyAppResource.Spec.UI.Color).Should(Equal("#34577c"))
			Expect(*createdMyAppResource.Spec.ReplicaCount).Should(Equal(int32(1)))

			By("By checking the pod info deployment fields")
			podInfoDeployment := &appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, podInfoDeployment)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			// validate its fields!
			Expect(podInfoDeployment.Name).Should(Equal(MyAppResourceName))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Env).Should(ContainElement(
				corev1.EnvVar{Name: podinfo.CacheEnvVar, Value: fmt.Sprintf("tcp://whatever-redis.%s.svc.cluster.local:6379", MyAppResourceNamespace)}))

			By("By checking the redis deployment fields")
			redisName := fmt.Sprintf("%s-redis", MyAppResourceName)
			redisLookupKey := types.NamespacedName{Name: redisName, Namespace: MyAppResourceNamespace}

			redisDeployment := &appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, redisLookupKey, redisDeployment)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			// validate its fields!
			Expect(redisDeployment.Name).Should(Equal(redisName))
			Expect(len(redisDeployment.Spec.Template.Spec.Containers)).Should(Equal(1))
			Expect(redisDeployment.Spec.Template.Spec.Containers[0].Name).Should(Equal("redis"))
			Expect(redisDeployment.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort).Should(Equal(int32(redis.RedisPort)))

			By("By checking the redis service fields")
			redisService := &corev1.Service{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, redisLookupKey, redisService)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			// validate its fields!
			Expect(redisService.Name).Should(Equal(redisName))
			Expect(len(redisService.Spec.Ports)).Should(Equal(1))
			Expect(redisService.Spec.Ports[0].Name).Should(Equal("redis"))
			Expect(redisService.Spec.Ports[0].Port).Should(Equal(int32(redis.RedisPort)))
			Expect(redisService.Spec.Ports[0].TargetPort).Should(Equal(intstr.IntOrString{IntVal: redis.RedisPort}))
			Expect(redisService.Spec.Selector).Should(Equal(map[string]string{"app": redisName}))

			By("By updating the redis deployment status")
			redisDeployment.Status.ReadyReplicas = int32(1)
			redisDeployment.Status.Replicas = int32(1)
			Expect(k8sClient.Status().Update(ctx, redisDeployment)).Should(Succeed())

			By("By checking the myappresource status updated")
			Eventually(func() (int, error) {
				err := k8sClient.Get(ctx, lookupKey, createdMyAppResource)
				if err != nil {
					return 0, err
				}

				return int(createdMyAppResource.Status.RedisReadyReplicas), nil
			}, timeout, interval).Should(Equal(1), "podInfoReadyReplicas in status should match the redis deployment")

			By("By disabling the redis")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdMyAppResource)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			createdMyAppResource.Spec.Redis.Enabled = false
			Expect(k8sClient.Update(ctx, createdMyAppResource)).Should(Succeed())

			By("By checking the redis gets deleted")
			Eventually(func() error {
				dep := &appsv1.Deployment{}
				return k8sClient.Get(context.Background(), redisLookupKey, dep)
			}, timeout, interval).ShouldNot(Succeed())

			Eventually(func() error {
				svc := &corev1.Service{}
				return k8sClient.Get(context.Background(), redisLookupKey, svc)
			}, timeout, interval).ShouldNot(Succeed())

		})
	})
})

var _ = Describe("MyAppResource controller - error cases", func() {

	It("Should error MyAppResourceName without required fields", func() {
		By("By creating a new MyAppResourceName without required ui fields")
		ctx := context.Background()

		replicas := int32(2)

		myAppResource := &v1alpha1.MyAppResource{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "my.api.group/v1alpha1",
				Kind:       "MyAppResource",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "whatever",
				Namespace: "default",
			},
			Spec: v1alpha1.MyAppResourceSpec{
				ReplicaCount: &replicas,
				//UI is missing
			},
		}

		Expect(k8sClient.Create(ctx, myAppResource)).ShouldNot(Succeed())

		By("By creating a new MyAppResourceName with invalid ui fields")
		myAppResource.Spec.UI = v1alpha1.UI{
			Color:   "#ddd", // should be six characters
			Message: "whatever",
		}
		Expect(k8sClient.Create(ctx, myAppResource)).ShouldNot(Succeed())

	})

})
