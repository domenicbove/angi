package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	myv1alpha1 "github.com/domenicbove/angi/api/v1alpha1"
)

var _ = Describe("MyAppResource controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
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
			myApp := &myv1alpha1.MyAppResource{}
			k8sClient.Get(context.Background(), lookupKey, myApp)
			return k8sClient.Delete(context.Background(), myApp)
		}, timeout, interval).Should(Succeed())

		Eventually(func() error {
			myApp := &myv1alpha1.MyAppResource{}
			return k8sClient.Get(context.Background(), lookupKey, myApp)
		}, timeout, interval).ShouldNot(Succeed())

		// cleanup podinfo deployment
		Eventually(func() error {
			myApp := &appsv1.Deployment{}
			k8sClient.Get(context.Background(), lookupKey, myApp)
			return k8sClient.Delete(context.Background(), myApp)
		}, timeout, interval).Should(Succeed())

		Eventually(func() error {
			myApp := &appsv1.Deployment{}
			return k8sClient.Get(context.Background(), lookupKey, myApp)
		}, timeout, interval).ShouldNot(Succeed())
	})

	Context("When creating MyAppResourceName", func() {
		It("Should create subresources", func() {
			By("By creating a new MyAppResourceName")
			ctx := context.Background()

			replicas := int32(2)

			myAppResource := &myv1alpha1.MyAppResource{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "my.api.group/v1alpha1",
					Kind:       "MyAppResource",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      MyAppResourceName,
					Namespace: MyAppResourceNamespace,
				},
				Spec: myv1alpha1.MyAppResourceSpec{
					ReplicaCount: &replicas,
					Image: myv1alpha1.Image{
						Repository: "ghcr.io/stefanprodan/podinfo",
						Tag:        "latest",
					},
					UI: myv1alpha1.UI{
						Color:   "#34577c",
						Message: "some message",
					},
				},
			}

			Expect(k8sClient.Create(ctx, myAppResource)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: MyAppResourceName, Namespace: MyAppResourceNamespace}
			createdMyAppResource := &myv1alpha1.MyAppResource{}

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
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Name).Should(Equal("podinfo"))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Env).Should(ContainElement(
				corev1.EnvVar{Name: "PODINFO_UI_COLOR", Value: "#34577c"}))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Env).Should(ContainElement(
				corev1.EnvVar{Name: "PODINFO_UI_MESSAGE", Value: "some message"}))
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

			myAppResource := &myv1alpha1.MyAppResource{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "my.api.group/v1alpha1",
					Kind:       "MyAppResource",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      MyAppResourceName,
					Namespace: MyAppResourceNamespace,
				},
				Spec: myv1alpha1.MyAppResourceSpec{
					UI: myv1alpha1.UI{
						Color:   "#34577c",
						Message: "some message",
					},
				},
			}

			Expect(k8sClient.Create(ctx, myAppResource)).Should(Succeed())

			By("By checking the created MyAppResource fields")
			lookupKey := types.NamespacedName{Name: MyAppResourceName, Namespace: MyAppResourceNamespace}
			createdMyAppResource := &myv1alpha1.MyAppResource{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdMyAppResource)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(createdMyAppResource.Spec.UI.Color).Should(Equal("#34577c"))
			Expect(*createdMyAppResource.Spec.ReplicaCount).Should(Equal(int32(1)))
			Expect(createdMyAppResource.Spec.Image.Repository).Should(Equal("ghcr.io/stefanprodan/podinfo"))
			Expect(createdMyAppResource.Spec.Image.Tag).Should(Equal("latest"))

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
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Name).Should(Equal("podinfo"))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Env).Should(ContainElement(
				corev1.EnvVar{Name: "PODINFO_UI_COLOR", Value: "#34577c"}))
			Expect(podInfoDeployment.Spec.Template.Spec.Containers[0].Env).Should(ContainElement(
				corev1.EnvVar{Name: "PODINFO_UI_MESSAGE", Value: "some message"}))
		})
	})
})
