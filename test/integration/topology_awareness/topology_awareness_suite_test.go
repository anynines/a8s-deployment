package topology_awareness

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"
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
	nodes     NodesTainter
)

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

	// Create Kubernetes client for interacting with the Kubernetes API
	k8sClient, err = dsi.NewK8sClient(dataservice, kubeconfigPath)
	Expect(err).To(BeNil(),
		fmt.Sprintf("error creating Kubernetes client for dataservice %s", dataservice))

	Expect(namespace.CreateIfNotExists(ctx, testingNamespace, k8sClient)).
		To(Succeed(), "failed to create testing namespace")

	// Generate a convenience object for tainting K8s nodes
	c, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	Expect(err).To(BeNil(),
		"failed to create client config for nodes tainter from kubeconig "+kubeconfigPath)

	cv1Client, err := corev1client.NewForConfig(c)
	Expect(err).To(BeNil(),
		fmt.Sprintf("failed to create client for nodes tainter from config %v", c))

	nodes = node.Client{
		Nodes:            cv1Client.Nodes(),
		MasterNodeTaints: node.MasterTaintKeys,
	}
})

var _ = AfterSuite(func() {
	defer cancel()
	Expect(namespace.DeleteIfAllowed(ctx, testingNamespace, k8sClient)).
		To(Succeed(), "failed to delete testing namespace")
})
