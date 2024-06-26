package backup

import (
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	backupv1beta3 "github.com/anynines/a8s-backup-manager/api/v1beta3"
	sbv1beta3 "github.com/anynines/a8s-service-binding-controller/api/v1beta3"

	"github.com/anynines/a8s-deployment/test/framework"
	bkp "github.com/anynines/a8s-deployment/test/framework/backup"
	"github.com/anynines/a8s-deployment/test/framework/chaos"
	"github.com/anynines/a8s-deployment/test/framework/dsi"
	"github.com/anynines/a8s-deployment/test/framework/postgresql"
	"github.com/anynines/a8s-deployment/test/framework/secret"
	"github.com/anynines/a8s-deployment/test/framework/servicebinding"
)

const (
	instancePort = 5432
	// We use a single replica since we are only interested in testing the behaviour of the
	// backup-agent.
	replicas     = 1
	suffixLength = 5

	// entity is a generic term to describe where data services store their data (e.g., a table in
	// a PostgreSQL database).
	entity = "test_entity"

	// backupPollingInterval specifies the polling interval for a backup for s3.
	s3PollingInterval = 100 * time.Millisecond

	// asyncOpsTimeout is the amount of minutes after which assertions fail if the condition
	// they check has not become true. Needed because some conditions might become true only after
	// some time, so we need to check them asynchronously.
	// TODO: Make asyncOpsTimeout an invocation parameter.
	asyncOpsTimeout = time.Minute * 5
	// backupTimeout is the amount of minutes after which assertions fail waiting for a backup
	// to complete. This should be adjusted once we have backups capable of recovering from crashes.
	backupTimeout = time.Minute * 10
)

var (
	// portForwardStopCh is the channel to close to terminate a port forward.
	portForwardStopCh chan struct{}
	localPort         int

	sb                 *sbv1beta3.ServiceBinding
	backup             *backupv1beta3.Backup
	serviceBindingData secret.SecretData
	instance           *postgresql.Postgresql
	client             dsi.DSIClient
)

var _ = Describe("Backup Chaos Tests", func() {
	BeforeEach(func() {
		// Create Dataservice instance and wait for instance readiness
		instance = postgresql.New(
			testingNamespace,
			framework.GenerateName(instanceNamePrefix, GinkgoParallelProcess(), suffixLength),
			replicas)

		Expect(k8sClient.Create(ctx, instance.GetClientObject())).
			To(Succeed(), fmt.Sprintf("failed to create instance %s/%s",
				instance.GetNamespace(), instance.GetName()))
		dsi.WaitForReadiness(ctx, instance.GetClientObject(), k8sClient)

		// Portforward to access instance from outside cluster.
		portForwardStopCh, localPort, err = framework.PortForward(
			ctx, instancePort, kubeconfigPath, instance, k8sClient)
		Expect(err).To(BeNil(),
			fmt.Sprintf("failed to establish portforward to DSI %s/%s",
				instance.GetNamespace(), instance.GetName()))

		// Create service binding for instance.
		sb = servicebinding.New(
			servicebinding.SetNamespacedName(instance.GetClientObject()),
			servicebinding.SetInstanceRef(instance.GetClientObject()),
		)
		Expect(k8sClient.Create(ctx, sb)).
			To(Succeed(), fmt.Sprintf("failed to create new servicebinding for DSI %s/%s",
				instance.GetNamespace(), instance.GetName()))
		servicebinding.WaitForReadiness(ctx, sb, k8sClient)
		serviceBindingData, err = secret.Data(
			ctx, k8sClient, servicebinding.SecretName(sb.Name), testingNamespace)
		Expect(err).To(BeNil(),
			fmt.Sprintf("failed to parse secret data for service binding %s/%s",
				sb.GetNamespace(), sb.GetName()))

		// Create client for interacting with the new instance.
		client, err = dsi.NewClient(dataservice, strconv.Itoa(localPort), serviceBindingData)
		Expect(err).To(BeNil(), "failed to create new dsi client")
	})

	AfterEach(func() {
		defer func() { close(portForwardStopCh) }()

		Expect(k8sClient.Delete(ctx, backup)).To(Succeed(),
			fmt.Sprintf("failed to delete backup %s/%s",
				backup.GetNamespace(), backup.GetName()))
		bkp.WaitForDeletion(ctx, backup, k8sClient)

		Expect(k8sClient.Delete(ctx, instance.GetClientObject())).To(Succeed(),
			fmt.Sprintf("failed to delete instance %s/%s",
				instance.GetNamespace(), instance.GetName()))

		Expect(k8sClient.Delete(ctx, sb)).To(Succeed(),
			fmt.Sprintf("failed to delete service binding %s/%s",
				sb.GetNamespace(), sb.GetName()))
		dsi.WaitForDeletion(ctx, instance.GetClientObject(), k8sClient)
		// TODO: Wait for deletion for all secondary objects
	})

	It("Backup agent crashes while processing a backup", func() {
		pgChaosInjector := chaos.PgInjector{Instance: instance}

		// TODO: Move this to beforeach. This is not a test spec.
		var writtenData string
		By("Bulk inserting data", func() {
			for i := 0; i < 10; i++ { // 10 * ~10MB ~= 100MB
				if i != 0 {
					writtenData += "\n"
				}
				randString := framework.GenerateRandString(10000000) // ~10 MB
				Expect(client.Write(ctx, entity, randString)).To(Succeed(),
					fmt.Sprintf("failed to insert data in DSI %s/%s",
						instance.GetNamespace(),
						instance.GetName()),
				)
				writtenData += randString
			}
		})

		By("Ensuring data was written successfully to master", func() {
			readData, err := client.Read(ctx, entity)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to read data from DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)

			Expect(readData).To(Equal(writtenData),
				fmt.Sprintf("read data does not match data written to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		var partitionMaster chaos.ChaosObject
		By("Stop all outgoing connections to AWS for S3 with a network partition", func() {
			partitionMaster, err = pgChaosInjector.PartitionMaster(
				ctx, k8sClient, []string{"amazonaws.com"})

			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to create network partition for DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		By("Wait for network partition chaos to apply", func() {
			var err error
			Eventually(func() bool {
				var ready bool
				ready, err = partitionMaster.CheckChaosActive(ctx, k8sClient)
				if err != nil {
					return false
				}

				return ready
			}, asyncOpsTimeout).Should(BeTrue(),
				fmt.Sprintf("timeout reached waiting for chaos to apply to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		By("Requesting a backup from the backup agent", func() {
			backup = bkp.New(
				bkp.SetNamespacedName(instance),
				bkp.SetInstanceRef(instance.GetClientObject()),
			)
			Expect(k8sClient.Create(ctx, backup)).To(Succeed(),
				fmt.Sprintf("failed to create backup for DSI %s/%s",
					instance.GetNamespace(), instance.GetName()))
		})

		var masterStop chaos.ChaosObject
		By("Crash master by applying PodChaos while processing backup", func() {
			var err error
			masterStop, err = pgChaosInjector.StopMaster(ctx, k8sClient)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to create Pod Chaos for DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		By("Wait for PodChaos to apply", func() {
			var err error
			Eventually(func() bool {
				var ready bool
				ready, err = masterStop.CheckChaosActive(ctx, k8sClient)
				if err != nil {
					return false
				}

				return ready
			}, asyncOpsTimeout).Should(BeTrue(),
				fmt.Sprintf("timeout reached waiting for Pod Chaos to apply to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		// Sleep to ensure the backup fails.
		time.Sleep(time.Second * 10)

		By("Restart master by deleting PodChaos", func() {
			Expect(k8sClient.Delete(ctx, masterStop.KubernetesObject())).To(Succeed(),
				fmt.Sprintf("failed to delete PodChaos on DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		By("Delete network partition on master", func() {
			Expect(k8sClient.Delete(ctx, partitionMaster.KubernetesObject())).To(Succeed(),
				fmt.Sprintf("failed to delete network partition on DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		By("Ensure the backup is eventually successful", func() {
			bkp.WaitForReadiness(ctx, backup, backupTimeout, k8sClient)
		})
	})

	// Chaos mesh creates chaos asynchronously and cannot be relied upon when precise timings of
	// chaos injection is required. It introduces flakiness that is hard to deal with when making
	// assertions due to delicate timing constraints. This test spec introduces a significant amount
	// of flakiness, so we may want to skip these tests using Ginkgo Labels on specs. Unit and
	// integration tests are likely to provide us with more reliable insights for tests when
	// dealing with delicate and precise timings of assertions.
	It("removes leftovers from crashed backups", Label("chaos", "backup", "flaky"), func() {
		s3Client, err := bkp.NewS3Client(k8sClient)
		if err != nil {
			Skip("Could not locate backup credentials or backup config for S3 client")
		}

		pgChaosInjector := chaos.PgInjector{Instance: instance}

		// TODO: Move this to beforeach. This is not a test spec.
		// Bulk insert data into instance.
		var writtenData string
		for i := 0; i < 10; i++ { // 10 * ~10MB ~= 100MB
			if i != 0 {
				writtenData += "\n"
			}
			randString := framework.GenerateRandString(10000000) // ~10 MB
			Expect(client.Write(ctx, entity, randString)).To(Succeed(),
				fmt.Sprintf("failed to insert data in DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
			writtenData += randString
		}

		// Ensure data was written successfully to master.
		readData, err := client.Read(ctx, entity)
		Expect(err).To(BeNil(),
			fmt.Sprintf("failed to read data from DSI %s/%s",
				instance.GetNamespace(),
				instance.GetName()),
		)

		Expect(readData).To(Equal(writtenData),
			fmt.Sprintf("read data does not match data written to DSI %s/%s",
				instance.GetNamespace(),
				instance.GetName()),
		)

		// Request a backup from the backup agent.
		backup = bkp.New(
			bkp.SetNamespacedName(instance),
			bkp.SetInstanceRef(instance.GetClientObject()),
			// Reduces flakiness by ensuring the backup is not retried which would lead to the files
			// being uploaded to S3 again. This would reduce the timing window we use to check
			// whether a failed backup has been cleaned up.
			bkp.MaxRetries("0"),
		)
		Expect(k8sClient.Create(ctx, backup)).To(Succeed(),
			fmt.Sprintf("failed to create backup for DSI %s/%s",
				instance.GetNamespace(), instance.GetName()))

		// Wait for backup to begin before going further.
		Eventually(func() bool {
			hasPartialData, err := s3Client.HasPartialBackupData(ctx, *backup)
			if err != nil {
				return false
			}

			return hasPartialData
		}, asyncOpsTimeout, s3PollingInterval).Should(BeTrue(),
			"timeout reached waiting for backup to begin")

		// Chaos Mesh takes awhile to apply the Chaos objects in which the backup could have already
		// been completed. This can increase flakiness.
		masterStop := applyPodChaos(pgChaosInjector)

		// Restart master by deleting PodChaos.
		Expect(k8sClient.Delete(ctx, masterStop.KubernetesObject())).To(Succeed(),
			fmt.Sprintf("failed to delete PodChaos on DSI %s/%s",
				instance.GetNamespace(),
				instance.GetName()),
		)

		// Asynchronous assertion cannot be relied upon due to flakiness.
		By("Deleting data from S3", func() {
			Eventually(func() bool {
				hasPartialData, err := s3Client.HasPartialBackupData(ctx, *backup)
				if err != nil {
					return false
				}

				return !hasPartialData
			}, asyncOpsTimeout, s3PollingInterval).Should(BeTrue(),
				"timeout reached waiting for cleanup of partial data")
		})
	})
})

// TODO: Move to dedicated package and call in chaos test. This could be a method on
// pgChaosInjector.
func applyPodChaos(pgChaosInjector chaos.PgInjector) chaos.ChaosObject { //nolint:ireturn
	// Crash master by applying PodChaos while processing backup.
	// This only works for single node DSIs.
	var masterStop chaos.ChaosObject
	masterStop, err = pgChaosInjector.StopMaster(ctx, k8sClient)
	Expect(err).To(BeNil(),
		fmt.Sprintf("failed to create Pod Chaos for DSI %s/%s",
			instance.GetNamespace(),
			instance.GetName()),
	)

	// Wait for PodChaos to apply
	Eventually(func() bool {
		var ready bool
		ready, err = masterStop.CheckChaosActive(ctx, k8sClient)
		if err != nil {
			return false
		}

		return ready
	}, asyncOpsTimeout).Should(BeTrue(),
		fmt.Sprintf("timeout reached waiting for Pod Chaos to apply to DSI %s/%s",
			instance.GetNamespace(),
			instance.GetName()),
	)

	return masterStop
}
