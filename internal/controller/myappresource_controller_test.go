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
		It("Should create PodInfo Deployment", func() {
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

			// We'll need to retry getting this newly created CronJob, given that creation may not immediately happen.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, myAppResourceLookupKey, createdMyAppResource)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			// Let's make sure our Schedule string value was properly converted/handled.
			Expect(createdMyAppResource.Spec.UI.Color).Should(Equal("#34577c"))

			// Delete
			By("Expecting to delete successfully")
			Eventually(func() error {
				f := &myv1alpha1.MyAppResource{}
				k8sClient.Get(context.Background(), myAppResourceLookupKey, f)
				return k8sClient.Delete(context.Background(), f)
			}, timeout, interval).Should(Succeed())

			By("Expecting to delete finish")
			Eventually(func() error {
				f := &myv1alpha1.MyAppResource{}
				return k8sClient.Get(context.Background(), myAppResourceLookupKey, f)
			}, timeout, interval).ShouldNot(Succeed())
		})
	})
})
