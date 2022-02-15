package servicebinding

import (
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/anynines/a8s-deployment/test/integration/framework"
	"github.com/anynines/a8s-deployment/test/integration/framework/dsi"
	"github.com/anynines/a8s-deployment/test/integration/framework/secret"
	"github.com/anynines/a8s-deployment/test/integration/framework/servicebinding"
	sbv1alpha1 "github.com/anynines/a8s-service-binding-controller/api/v1alpha1"
)

const (
	instancePort = 5432
	replicas     = 1
	suffixLength = 5

	DbAdminUsernameKey = "username"
	DbAdminPasswordKey = "password"
)

var (
	portForwardStopCh chan struct{}

	sb             *sbv1alpha1.ServiceBinding
	instance       dsi.Object
	dsiAdminClient dsi.DSIClient
)

var _ = Describe("service binding", func() {
	BeforeEach(func() {
		// Create Dataservice instance and wait for instance readiness
		instance, err = dsi.New(
			dataservice,
			testingNamespace,
			framework.GenerateName(instanceNamePrefix, GinkgoParallelProcess(), suffixLength),
			replicas,
		)
		Expect(err).To(BeNil())

		Expect(k8sClient.Create(ctx, instance.GetClientObject())).
			To(Succeed(), "failed to create DSI")

		dsi.WaitForReadiness(ctx, instance.GetClientObject(), k8sClient)

		// Portforward to access DSI from outside cluster.
		portForwardStopCh, localPort, err = framework.PortForward(
			ctx, instancePort, kubeconfigPath, instance, k8sClient)
		Expect(err).To(BeNil(),
			fmt.Sprintf("failed to establish portforward to DSI %s/%s",
				instance.GetNamespace(), instance.GetName()))

		adminSecret, err := secret.AdminSecretData(ctx, k8sClient,
			instance.GetName(), testingNamespace)
		Expect(err).To(BeNil())

		// Create DSIClient for interacting with the new DSI.
		dsiAdminClient, err = dsi.NewClient(dataservice, strconv.Itoa(localPort), adminSecret)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, instance.GetClientObject())).To(Succeed(),
			"failed to delete DSI")
		close(portForwardStopCh)
	})

	It("Performs basic service binding lifecycle operations", func() {
		// Create service binding for DSI.
		sb = servicebinding.New(
			servicebinding.SetNamespacedName(instance.GetClientObject()),
			servicebinding.SetInstanceRef(instance.GetClientObject()),
		)

		By("Creating the service binding CR", func() {
			Expect(k8sClient.Create(ctx, sb)).To(Succeed(), "failed to create new servicebinding")
			servicebinding.WaitForReadiness(ctx, sb, k8sClient)
		})

		var serviceBindingData secret.SecretData
		By("Creating the service binding secret", func() {
			serviceBindingData, err = secret.Data(ctx, k8sClient,
				servicebinding.SecretName(sb.Name), testingNamespace)
			Expect(err).To(BeNil(), fmt.Sprintf("unable to parse secret data for sb: %s", err))
		})

		By("Creating a user in the Database", func() {
			exists := dsiAdminClient.UserExists(ctx,
				serviceBindingData[DbAdminUsernameKey],
				serviceBindingData[DbAdminPasswordKey])

			Expect(exists).To(BeTrue(), "unable to get DSI users")
		})

		By("Deleting the service binding", func() {
			Expect(k8sClient.Delete(ctx, sb)).To(Succeed(),
				"failed to delete service binding resource")
		})

		By("Deleting the user in the Database", func() {
			EventuallyWithOffset(1, func() bool {
				exists := dsiAdminClient.UserExists(ctx,
					serviceBindingData[DbAdminUsernameKey],
					serviceBindingData[DbAdminPasswordKey])
				return exists
			}, framework.AsyncOpsTimeoutMins, 1*time.Second).
				Should(BeFalse(),
					fmt.Sprintf("timeout reached waiting for instance readiness: %s", err))
		})

		By("Deleting the service binding secret", func() {
			EventuallyWithOffset(1, func() secret.SecretData {
				s, _ := secret.Data(ctx, k8sClient, sb.Name, testingNamespace)
				return s
			}, framework.AsyncOpsTimeoutMins, 1*time.Second).
				Should(BeNil(),
					fmt.Sprintf("timeout reached waiting for instance readiness: %s", err))
		})
	})
})
