package backup

import (
	"context"
	"fmt"
	"strings"
	"testing"

	chmv1alpha1 "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/anynines/a8s-deployment/test/framework"
	"github.com/anynines/a8s-deployment/test/framework/chaos"
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

func TestChaos(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Backup Chaos Test Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.Background())

	// Parse environmental variable configuration
	config, err := framework.ParseEnv()
	Expect(err).To(BeNil(), "failed to parse environmental variables as configuration")
	kubeconfigPath, instanceNamePrefix, dataservice, testingNamespace =
		framework.ConfigToVars(config)

	Expect(strings.ToLower(dataservice) == "postgresql").To(BeTrue(),
		"This test suite only supports PostgreSQL")

	// Add ChaosMesh definitions
	Expect(chmv1alpha1.AddToScheme(scheme.Scheme)).To(Succeed())

	// Create Kubernetes client for interacting with the Kubernetes API
	k8sClient, err = dsi.NewK8sClient(dataservice, kubeconfigPath)
	Expect(err).To(BeNil(),
		fmt.Sprintf("error creating Kubernetes client for dataservice %s", dataservice))

	Expect(chaos.VerifyChaosMeshPresent(ctx, k8sClient)).To(Succeed(),
		"ChaosMesh needs to be installed to run this test suite")

	Expect(namespace.CreateIfNotExists(ctx, testingNamespace, k8sClient)).
		To(Succeed(), "failed to create testing namespace")

})

var _ = AfterSuite(func() {
	Expect(namespace.DeleteIfAllowed(ctx, testingNamespace, k8sClient)).
		To(Succeed(), "failed to delete testing namespace")
	cancel()
})
