package topology_awareness

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/anynines/a8s-deployment/test/integration/framework"
	"github.com/anynines/a8s-deployment/test/integration/framework/dsi"
	"github.com/anynines/a8s-deployment/test/integration/framework/namespace"
	"github.com/anynines/a8s-deployment/test/integration/framework/node"
)

var (
	ctx                                                               context.Context
	cancel                                                            context.CancelFunc
	testingNamespace, kubeconfigPath, dataservice, instanceNamePrefix string

	k8sClient runtimeClient.Client
	nodes     NodesClient
)

const minNbrWorkerNodes = 3

func TestTopologyAwareness(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Topology Awareness Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.Background())

	// Parse environmental variable configuration
	config, err := framework.ParseEnv()
	Expect(err).To(BeNil(), "failed to parse environmental variables as configuration")
	kubeconfigPath, instanceNamePrefix, dataservice, testingNamespace =
		framework.ConfigToVars(config)

	// Generate a convenience object for dealing with K8s cluster nodes
	nodes, err = node.NewClientFromKubecfg(kubeconfigPath)
	Expect(err).To(BeNil())

	// This test suite requires a minimum number of K8s worker nodes to run, so we verify this
	// before running the tests to fail fast if the requirement isn't met.
	Expect(verifyK8SClusterHasEnoughWorkerNodes(nodes, minNbrWorkerNodes)).To(Succeed())

	// Create Kubernetes client for interacting with the Kubernetes API
	k8sClient, err = dsi.NewK8sClient(dataservice, kubeconfigPath)
	Expect(err).To(BeNil(),
		fmt.Sprintf("error creating Kubernetes client for dataservice %s", dataservice))

	Expect(namespace.CreateIfNotExists(ctx, testingNamespace, k8sClient)).
		To(Succeed(), "failed to create testing namespace")
})

var _ = AfterSuite(func() {
	defer cancel()
	Expect(namespace.DeleteIfAllowed(ctx, testingNamespace, k8sClient)).
		To(Succeed(), "failed to delete testing namespace")
})

func verifyK8SClusterHasEnoughWorkerNodes(nodes NodesClient, minNbrWorkerNodes int) error {
	workers, err := nodes.ListWorkers(ctx)
	if err != nil {
		return fmt.Errorf("failed to verify that test cluster has enough worker nodes: %w", err)
	}

	if len(workers) < minNbrWorkerNodes {
		return fmt.Errorf("this test suite needs at least %d worker nodes in the test cluster, "+
			"but only %d were found", minNbrWorkerNodes, len(workers))
	}

	return nil
}
