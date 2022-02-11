package backup

import (
	"context"
	"fmt"
	"log"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/anynines/a8s-deployment/test/integration/framework"
	"github.com/anynines/a8s-deployment/test/integration/framework/dsi"
)

var (
	ctx                                                               context.Context
	cancel                                                            context.CancelFunc
	err                                                               error
	testingNamespace, kubeconfigPath, dataservice, instanceNamePrefix string

	k8sClient runtimeClient.Client
)

func TestBackup(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Backup Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.Background())

	// Parse environmental variable configuration
	config, err := framework.ParseEnv()
	Expect(err).To(BeNil(), "failed to parse environmental variables as configuration")
	kubeconfigPath, instanceNamePrefix, dataservice, testingNamespace = framework.ConfigToVars(config)

	// Create kubernetes client for interacting with the Kubernetes API
	k8sClient, err = dsi.NewK8sClient(dataservice, kubeconfigPath)
	Expect(err).To(BeNil(),
		fmt.Sprintf("error creating Kubernetes client for dataservice %s", dataservice))

	Expect(createNamespaceIfNotExists(ctx, testingNamespace, k8sClient)).
		To(Succeed(), "failed to create testing namespace")
})

var _ = AfterSuite(func() {
	Expect(deleteNamespaceIfAllowed(ctx, testingNamespace, k8sClient)).
		To(Succeed(), "failed to delete testing namespace")
	cancel()
})

func createNamespaceIfNotExists(ctx context.Context,
	testingNamespace string,
	c runtimeClient.Client) error {

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testingNamespace,
		},
	}

	err := c.Create(ctx, ns)
	if k8serrors.IsAlreadyExists(err) {
		log.Println("The namespace already exists. Skipping namespace creation for: ",
			testingNamespace)
		return nil
	}
	return err
}

func deleteNamespaceIfAllowed(ctx context.Context,
	testingNamespace string,
	c runtimeClient.Client) error {

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testingNamespace,
		},
	}

	err := c.Delete(ctx, ns)
	if err != nil && k8serrors.IsForbidden(err) {
		// TODO: Use structured logging using context where stdlib log or fmt is used for
		// logging.
		log.Println("The namespace is forbidden from deletion: ", testingNamespace)
		return nil
	}
	return err
}
