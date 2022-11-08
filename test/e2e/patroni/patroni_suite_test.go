package patroni

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/anynines/a8s-deployment/test/framework"
	"github.com/anynines/a8s-deployment/test/framework/dsi"
	"github.com/anynines/a8s-deployment/test/framework/namespace"
)

var (
	ctx                                                               context.Context
	cancel                                                            context.CancelFunc
	err                                                               error
	testingNamespace, kubeconfigPath, dataservice, instanceNamePrefix string

	k8sClient runtimeClient.Client
)

const expectedDataservice = "PostgreSQL"

func TestPatroni(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Patroni Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.Background())

	// Parse environmental variable configuration
	config, err := framework.ParseEnv()
	Expect(err).To(BeNil(), "failed to parse environmental variables as configuration")
	kubeconfigPath, instanceNamePrefix, dataservice, testingNamespace =
		framework.ConfigToVars(config)

	// TODO: We could use AbortSuite to safely exit the test suite rather than the approach here.
	// We may want to bump Ginkgo soon to keep it up to date and get new features.
	// https://pkg.go.dev/github.com/onsi/ginkgo/v2#AbortSuite
	// Provides information on failure to help the user identify the issue. When we use
	// AbortSuite we can simply write out the error as a message.
	Expect(dataservice).To(Equal(expectedDataservice), "this suite can run only for dataservice "+
		expectedDataservice)

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
