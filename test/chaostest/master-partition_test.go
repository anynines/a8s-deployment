package chaostest

import (
	"context"
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/anynines/a8s-deployment/test/e2e/framework"
	"github.com/anynines/a8s-deployment/test/e2e/framework/dsi"
	"github.com/anynines/a8s-deployment/test/e2e/framework/postgresql"
	"github.com/anynines/a8s-deployment/test/e2e/framework/secret"
)

const (
	instancePort = 5432
	replicas     = 3
	suffixLength = 5
	sbAmount     = 5
	dsiAmount    = 3

	DbAdminUsernameKey = "username"
	DbAdminPasswordKey = "password"
	AppsDefaultDb      = "a9s_apps_default_db"
)

var _ = Describe("Replication Manager", func() {

	var (
		portForwardStopCh chan struct{}

		// sb             *sbv1alpha1.ServiceBinding
		instance       dsi.Object
		dsiAdminClient dsi.DSIClient
		// dsiSbClient    dsi.DSIClient
		// sbClientMap    map[*sbv1alpha1.ServiceBinding]dsi.DSIClient
	)

	Context("The master is in a Network Partition and cannot reach the Kubernetes API, or replicas",
		func() {
			BeforeEach(func() {
				// Create Dataservice instance and wait for instance readiness
				instance = postgresql.New(
					testingNamespace,
					framework.GenerateName(instanceNamePrefix,
						GinkgoParallelProcess(),
						suffixLength),
					replicas,
				)

				Expect(k8sClient.Create(ctx, instance.GetClientObject())).
					To(Succeed(), "failed to create DSI")

				dsi.WaitForReadiness(ctx, instance.GetClientObject(), k8sClient)

				// Portforward to access DSI from outside cluster.
				portForwardStopCh, localPort, err = framework.PortForward(
					ctx, instancePort, kubeconfigPath, instance, k8sClient)
				Expect(err).To(BeNil(),
					fmt.Sprintf("failed to establish portforward to DSI %s/%s",
						instance.GetNamespace(),
						instance.GetName()))

				adminSecret, err := secret.AdminSecretData(ctx,
					k8sClient,
					instance.GetName(),
					testingNamespace)

				Expect(err).To(BeNil(),
					fmt.Sprintf("failed to parse secret data of admin credentials for DSI %s/%s",
						instance.GetNamespace(),
						instance.GetName()))

				// Create DSIClient for interacting with the new DSI.
				dsiAdminClient = postgresql.NewClient(
					adminSecret,
					strconv.Itoa(localPort))

				Expect(err).To(BeNil(), "failed to create DSI client")
			})

			AfterEach(func() {
				defer func() { close(portForwardStopCh) }()
				Expect(k8sClient.Delete(ctx, instance.GetClientObject())).To(Succeed(),
					fmt.Sprintf("failed to delete DSI %s/%s",
						instance.GetNamespace(),
						instance.GetName()))
			})

			It("Elects a new leader when the old one is in a Network partition", func() {

				Expect(dsiAdminClient.Write(ctx, "test", "hello")).To(Succeed())

				oldMaster, err := framework.GetPrimaryPodUsingServiceSelector(ctx, instance, k8sClient)
				Expect(err).To(BeNil())

				c, cf := context.WithTimeout(ctx, framework.AsyncOpsTimeoutMins)
				defer cf()

				Expect(chaos.IsolatePrimary(c, instance)).To(Succeed())
				defer chaos.Undo(instance)

				Eventually(func() bool {
					newMaster, err := framework.GetPrimaryPodUsingServiceSelector(ctx, instance, k8sClient)
					if err != nil {
						return false
					}
					return oldMaster.UID != newMaster.UID
				}, framework.AsyncOpsTimeoutMins, 1*time.Second).Should(BeTrue(),
					"No new master was elected after old master was partitioned")

			})

			It("Stops Accepting Writes to an Isolated Master", func() {

				Expect(dsiAdminClient.Write(ctx, "test", "hello")).To(Succeed())

				Expect(err).To(BeNil())

				c, cf := context.WithTimeout(ctx, framework.AsyncOpsTimeoutMins)
				defer cf()

				Expect(chaos.IsolatePrimary(c, instance)).To(Succeed())
				defer chaos.Undo(instance)

				Expect(dsiAdminClient.Write(ctx, "test", "123")).NotTo(Succeed(),
					"Isolated Master still accepts writes")

			})
		})
})
