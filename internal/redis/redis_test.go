package redis

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBooks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Redis Suite")
}

var _ = Describe("Redis", func() {

	Context("When constructing endpoint", func() {
		It("Should build strings correctly", func() {
			Expect(GetDeploymentName("whatever")).Should(Equal("whatever-redis"))

			Expect(GetEndpoint("whatever", "default")).Should(Equal("tcp://whatever-redis.default.svc.cluster.local:6379"))
		})
	})
})
