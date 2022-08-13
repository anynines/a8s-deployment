package chaostest

import (
	"context"
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"

	"github.com/anynines/a8s-deployment/test/e2e/framework"
	"github.com/anynines/a8s-deployment/test/e2e/framework/dsi"
	"github.com/anynines/a8s-deployment/test/e2e/framework/postgresql"
	"github.com/anynines/a8s-deployment/test/e2e/framework/secret"
)

const (
	instancePort = 5432
	replicas     = 3
	suffixLength = 5

	DbAdminUsernameKey = "username"
	DbAdminPasswordKey = "password"
	AppsDefaultDb      = "a9s_apps_default_db"
)

var _ = Describe("Replication Manager", func() {

	var (
		portForwardStopCh chan struct{}

		instance       dsi.Object
		dsiAdminClient dsi.DSIClient

		oldPrimary []v1.Pod
		teardown   func(context.Context) error
	)

	Context("The Primary is in a Network Partition and cannot reach the Kubernetes API, or replicas",
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

				Expect(dsiAdminClient.Write(ctx, "test", "hello")).To(Succeed())

				oldPrimary, err = framework.GetAllPrimaryPodsUsingServiceSelector(ctx, instance, k8sClient)
				Expect(err).To(BeNil())

				time.Sleep(30 * time.Second) // wait for replica synch

				timeoutCtx, cancel := context.WithTimeout(ctx, framework.AsyncOpsTimeoutMins)
				defer cancel()

				var e error
				teardown, e = chaos.IsolatePrimary(timeoutCtx, instance)
				Expect(e).To(Succeed())
			})

			AfterEach(func() {
				defer func() { close(portForwardStopCh) }()

				Expect(teardown(ctx)).To(Succeed(),
					"Failed to clean up injected fault",
				)
				Expect(k8sClient.Delete(ctx, instance.GetClientObject())).To(Succeed(),
					fmt.Sprintf("failed to delete DSI %s/%s",
						instance.GetNamespace(),
						instance.GetName()))
			})

			It("Elects a new Primary", func() {
				Eventually(func() bool {
					newPrimary, err := framework.GetAllPrimaryPodsUsingServiceSelector(ctx, instance, k8sClient)
					if err != nil {
						return false
					}

					// Old primary will not remove the primary label
					// because patroni, which manages the labels cannot access the k8s api.
					// This means we have handle the old primary still being marked as such
					for _, o := range oldPrimary {
						flag := false
						for _, n := range newPrimary {
							if n.Name == o.Name {
								flag = true
							}
						}
						if !flag {
							return false
						}
					}
					return true
				}, framework.AsyncOpsTimeoutMins, 1*time.Second).Should(BeTrue(),
					"No new master was elected after old primary was partitioned")
			})

			It("Stops Accepting Writes to an Isolated Primary", func() {
				Eventually(func() error {

					// Write to the old Primary
					wErr := dsiAdminClient.Write(ctx, "test", "123")
					canWrite := wErr == nil
					primaries, err := framework.GetAllPrimaryPodsUsingServiceSelector(ctx, instance, k8sClient)
					if err != nil {
						// The write to the master has not actually failed, ignore error
						return nil
					}

					if len(primaries) != 1 {
						Expect(canWrite).To(BeFalse())
					}

					return wErr
				}, framework.AsyncOpsTimeoutMins, 1*time.Second).
					ShouldNot(Succeed(), "Isolated Primary still accepts writes")
			})
		})
})
