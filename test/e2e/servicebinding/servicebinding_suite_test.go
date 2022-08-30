package servicebinding

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/anynines/a8s-deployment/test/e2e/framework"
	"github.com/anynines/a8s-deployment/test/e2e/framework/dsi"
	"github.com/anynines/a8s-deployment/test/e2e/framework/namespace"
)

var (
	ctx                                                               context.Context
	cancel                                                            context.CancelFunc
	err                                                               error
	testingNamespace, kubeconfigPath, dataservice, instanceNamePrefix string
	localPort                                                         int

	k8sClient runtimeClient.Client
)

func TestServiceBinding(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ServiceBinding Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.Background())

	// Parse environmental variable configuration
	config, err := framework.ParseEnv()
	Expect(err).To(BeNil(), "failed to parse environmental variables as configuration")

	kubeconfigPath, instanceNamePrefix, dataservice, testingNamespace =
		framework.ConfigToVars(config)

	// Create kubernetes client for interacting with the Kubernetes API
	k8sClient, err = dsi.NewK8sClient(dataservice, kubeconfigPath)
	Expect(err).To(BeNil(),
		fmt.Sprintf("error creating Kubernetes client for dataservice %s", dataservice))

	Expect(namespace.CreateIfNotExists(ctx, testingNamespace, k8sClient)).
		To(Succeed(), "failed to create testing namespace")
})

var _ = AfterSuite(func() {
	Expect(namespace.DeleteIfAllowed(ctx, testingNamespace, k8sClient)).
		To(Succeed(), "failed to delete testing namespace")

	cancel()
})
