package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

	Context("When creating MyAppResourceName", func() {
		It("Should create MyAppResourceName", func() {
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

			myAppResourceLookupKey := types.NamespacedName{Name: MyAppResourceName, Namespace: MyAppResourceNamespace}
			createdMyAppResource := &myv1alpha1.MyAppResource{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, myAppResourceLookupKey, createdMyAppResource)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(createdMyAppResource.Spec.UI.Color).Should(Equal("#34577c"))

			By("Expecting to delete successfully")
			Eventually(func() error {
				myApp := &myv1alpha1.MyAppResource{}
				k8sClient.Get(context.Background(), myAppResourceLookupKey, myApp)
				return k8sClient.Delete(context.Background(), myApp)
			}, timeout, interval).Should(Succeed())

			By("Expecting to delete finish")
			Eventually(func() error {
				myApp := &myv1alpha1.MyAppResource{}
				return k8sClient.Get(context.Background(), myAppResourceLookupKey, myApp)
			}, timeout, interval).ShouldNot(Succeed())
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

			myAppResourceLookupKey := types.NamespacedName{Name: MyAppResourceName, Namespace: MyAppResourceNamespace}
			createdMyAppResource := &myv1alpha1.MyAppResource{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, myAppResourceLookupKey, createdMyAppResource)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			Expect(createdMyAppResource.Spec.UI.Color).Should(Equal("#34577c"))

			Expect(*createdMyAppResource.Spec.ReplicaCount).Should(Equal(int32(1)))
			Expect(createdMyAppResource.Spec.Image.Repository).Should(Equal("ghcr.io/stefanprodan/podinfo"))
			Expect(createdMyAppResource.Spec.Image.Tag).Should(Equal("latest"))

			By("Expecting to delete successfully")
			Eventually(func() error {
				myApp := &myv1alpha1.MyAppResource{}
				k8sClient.Get(context.Background(), myAppResourceLookupKey, myApp)
				return k8sClient.Delete(context.Background(), myApp)
			}, timeout, interval).Should(Succeed())

			By("Expecting to delete finish")
			Eventually(func() error {
				myApp := &myv1alpha1.MyAppResource{}
				return k8sClient.Get(context.Background(), myAppResourceLookupKey, myApp)
			}, timeout, interval).ShouldNot(Succeed())
		})
	})
})
