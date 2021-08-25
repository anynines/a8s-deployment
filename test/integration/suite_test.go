package integration

import (
	"log"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// We start with the simple approach of simply deploying the a8s  framework via yaml manifests
// contained in this directory.

const (
	deploymentPath = "../../deploy/a8s"
)

func TestTest(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Test Suite")
}

var _ = BeforeSuite(func() {
	bytes, err := kubectl("apply", "--kustomize", deploymentPath)
	if err != nil {
		log.Println(err)
	}
	log.Println(string(bytes))
})

var _ = AfterSuite(func() {
})
