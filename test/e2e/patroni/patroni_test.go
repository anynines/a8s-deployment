package patroni

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/anynines/a8s-deployment/test/framework"
	"github.com/anynines/a8s-deployment/test/framework/dsi"
	"github.com/anynines/a8s-deployment/test/framework/postgresql"
	"github.com/anynines/a8s-deployment/test/framework/secret"
	"github.com/anynines/postgresql-operator/api/v1alpha1"
)

const (
	instancePort       = 5432
	replicas           = 1
	suffixLength       = 5
	patroniMonitorPort = 8008

	// PostgreSQL configuration naming style
	ArchiveTimeout        = "archive_timeout"
	TempFileLimit         = "temp_file_limit"
	TrackIOTiming         = "track_io_timing"
	StatementTimeout      = "statement_timeout"
	ClientMinMessages     = "client_min_messages"
	LogMinMessages        = "log_min_messages"
	LogMinErrorStatement  = "log_min_error_statement"
	LogStatement          = "log_statement"
	LogErrorVerbosity     = "log_error_verbosity"
	SSLCiphers            = "ssl_ciphers"
	SSLMinProtocolVersion = "ssl_min_protocol_version"
	WALWriterDelay        = "wal_writer_delay"
	SynchronousCommit     = "synchronous_commit"
	MaxConnections        = "max_connections"
	// SharedBuffers is not being set or updated.
	// https://github.com/anynines/postgresql-operator/issues/75
	SharedBuffers          = "shared_buffers"
	MaxReplicationSlots    = "max_replication_slots"
	MaxWALSenders          = "max_wal_senders"
	MaxLocksPerTransaction = "max_locks_per_transaction"
)

var (
	// portForwardStopCh is the channel used to manage the lifecycle of a port forward.
	portForwardStopCh chan struct{}
	localPort         int
	ok                bool

	instance        dsi.Object
	client          dsi.DSIClient
	pg              *v1alpha1.Postgresql
	adminSecretData secret.SecretData
)

var _ = Describe("Patroni end-to-end Tests", func() {
	Context("Patroni Configuration", func() {
		AfterEach(func() {
			defer func() { close(portForwardStopCh) }()
			Expect(k8sClient.Delete(ctx, instance.GetClientObject())).To(Succeed(),
				fmt.Sprintf("failed to delete instance %s/%s",
					instance.GetNamespace(), instance.GetName()))
			dsi.WaitForDeletion(ctx, instance.GetClientObject(), k8sClient)
		})

		It("Sets default configuration when deploying a PostgreSQL instance without explicit configuration", func() {
			const (
				// The representations between values given to fields
				// in the PostgreSQL CR and values of these parameters in PostgreSQL
				// itself do not always match. So we provide here what we expect in
				// PostgreSQL itself and not in the PostgreSQL CR.
				defaultArchiveTimeoutSeconds  = 0
				defaultClientMinMessages      = "notice"
				defaultLogErrorVerbosity      = "default"
				defaultLogMinErrorStatement   = "error"
				defaultLogMinMessages         = "warning"
				defaultLogStatement           = "none"
				defaultMaxConnections         = 100
				defaultMaxReplicationSlots    = 10
				defaultMaxWALSenders          = 10
				defaultSharedBuffers          = "100MB" // 1024 is converted to 100MB.
				defaultSSLCiphers             = "HIGH:MEDIUM:+3DES:!aNULL"
				defaultSSLMinProtocolVersion  = "TLSv1.2"
				defaultStatementTimeoutMillis = 0
				defaultSynchronousCommit      = "on"
				defaultTempFileLimitKiloBytes = -1
				defaultTrackIOTiming          = "off"
				defaultWalWriterDelayMillis   = 200 // Needs ms
				defaultMaxLocksPerTransaction = 64
			)

			By("creating a PostgreSQL instance with implicit defaults", func() {
				instance, err = dsi.New(
					dataservice,
					testingNamespace,
					framework.GenerateName(instanceNamePrefix,
						GinkgoParallelProcess(),
						suffixLength),
					replicas,
				)
				Expect(err).To(BeNil(), "failed to generate new DSI resource")

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

				// Fetch admin secret for privileged DSIClient. We need a
				// privileged client since some config parameters such as
				// SSLCiphers can not be fetched by service binding users.
				adminSecretData, err = secret.AdminSecretData(
					ctx, k8sClient, instance.GetName(), instance.GetNamespace())
				Expect(err).To(BeNil(),
					fmt.Sprintf("failed to parse secret data of admin credentials for %s/%s",
						instance.GetNamespace(), instance.GetName()))

				// Create client for interacting with the new instance.
				client, err = dsi.NewClient(
					dataservice, strconv.Itoa(localPort), adminSecretData)
				Expect(err).To(BeNil(), "failed to create new dsi client")
			})

			By("checking that the defaults are set correctly", func() {
				var expectedConfig = []struct {
					parameter, value string
				}{
					{ArchiveTimeout, strconv.Itoa(defaultArchiveTimeoutSeconds)},
					{TempFileLimit, strconv.Itoa(defaultTempFileLimitKiloBytes)},
					{TrackIOTiming, defaultTrackIOTiming},
					{StatementTimeout, strconv.Itoa(defaultStatementTimeoutMillis)},
					{ClientMinMessages, defaultClientMinMessages},
					{LogMinMessages, defaultLogMinMessages},
					{LogMinErrorStatement, defaultLogMinErrorStatement},
					{LogStatement, defaultLogStatement},
					{LogErrorVerbosity, defaultLogErrorVerbosity},
					{SSLCiphers, defaultSSLCiphers},
					{SSLMinProtocolVersion, defaultSSLMinProtocolVersion},
					{WALWriterDelay, strconv.Itoa(defaultWalWriterDelayMillis) + "ms"},
					{SynchronousCommit, defaultSynchronousCommit},
					{MaxConnections, strconv.Itoa(defaultMaxConnections)},
					// SharedBuffers is not being set or updated.
					// https://github.com/anynines/postgresql-operator/issues/75
					// {SharedBuffers, defaultSharedBuffers},
					{MaxReplicationSlots, strconv.Itoa(defaultMaxReplicationSlots)},
					{MaxWALSenders, strconv.Itoa(defaultMaxWALSenders)},
					{MaxLocksPerTransaction, strconv.Itoa(defaultMaxLocksPerTransaction)},
				}

				for _, setting := range expectedConfig {
					Expect(client.CheckParameter(
						ctx,
						setting.parameter,
						setting.value,
					)).To(Succeed(),
						fmt.Sprintf("the default configuration was not what we expected for %s/%s",
							instance.GetNamespace(), instance.GetName()))
				}
			})
		})

		It("Sets configuration when creating an instance with an explicit custom PostgreSQL configuration", func() {
			By("applying custom configuration to PostgreSQL resource", func() {
				instance, err = dsi.New(
					dataservice,
					testingNamespace,
					framework.GenerateName(instanceNamePrefix,
						GinkgoParallelProcess(),
						suffixLength),
					replicas,
				)
				Expect(err).To(BeNil(), "failed to generate new DSI resource")

				// Cast interface to concrete struct so that we can access fields
				// directly
				pg, ok = instance.GetClientObject().(*v1alpha1.Postgresql)
				Expect(ok).To(BeTrue(),
					"failed to cast object interface to PostgreSQL struct")

				setCustomPostgresConfig(pg)
			})

			By("creating a PostgreSQL instance with custom configuration", func() {
				Expect(k8sClient.Create(ctx, pg)).
					To(Succeed(), fmt.Sprintf("failed to create instance %s/%s",
						instance.GetNamespace(), instance.GetName()))
				dsi.WaitForReadiness(ctx, instance.GetClientObject(), k8sClient)

				// Portforward to access instance from outside cluster.
				portForwardStopCh, localPort, err = framework.PortForward(
					ctx, instancePort, kubeconfigPath, instance, k8sClient)
				Expect(err).To(BeNil(),
					fmt.Sprintf("failed to establish portforward to DSI %s/%s",
						instance.GetNamespace(), instance.GetName()))

				// Fetch admin secret for privileged DSIClient. We need a
				// privileged client since some parameters such as SSLCiphers can
				// not be fetched by service binding users.
				adminSecretData, err = secret.AdminSecretData(
					ctx, k8sClient, instance.GetName(), instance.GetNamespace())
				Expect(err).To(BeNil(),
					fmt.Sprintf("failed to parse secret data of admin credentials for %s/%s",
						instance.GetNamespace(), instance.GetName()))

				// Create client for interacting with the new instance.
				client, err = dsi.NewClient(
					dataservice, strconv.Itoa(localPort), adminSecretData)
				Expect(err).To(BeNil(), "failed to create new dsi client")
			})

			By("checking that the custom configuration is set correctly", func() {
				var expectedConfig = []struct {
					parameter, value string
				}{
					{ArchiveTimeout, strconv.Itoa(pg.Spec.PostgresConfiguration.ArchiveTimeoutSeconds) + "s"},
					{TempFileLimit, strconv.Itoa(pg.Spec.PostgresConfiguration.TempFileLimitKiloBytes)},
					{TrackIOTiming, pg.Spec.PostgresConfiguration.TrackIOTiming},
					{StatementTimeout, strconv.Itoa(pg.Spec.PostgresConfiguration.StatementTimeoutMillis) + "ms"},
					{ClientMinMessages, pg.Spec.PostgresConfiguration.ClientMinMessages},
					{LogMinMessages, pg.Spec.PostgresConfiguration.LogMinMessages},
					{LogMinErrorStatement, pg.Spec.PostgresConfiguration.LogMinErrorStatement},
					{LogStatement, pg.Spec.PostgresConfiguration.LogStatement},
					{LogErrorVerbosity, strings.ToLower(pg.Spec.PostgresConfiguration.LogErrorVerbosity)},
					{SSLCiphers, pg.Spec.PostgresConfiguration.SSLCiphers},
					{SSLMinProtocolVersion, pg.Spec.PostgresConfiguration.SSLMinProtocolVersion},
					{WALWriterDelay, strconv.Itoa(pg.Spec.PostgresConfiguration.WALWriterDelayMillis) + "ms"},
					{SynchronousCommit, pg.Spec.PostgresConfiguration.SynchronousCommit},
					{MaxConnections, strconv.Itoa(pg.Spec.PostgresConfiguration.MaxConnections)},
					// SharedBuffers is not being set or updated.
					// https://github.com/anynines/postgresql-operator/issues/75
					// {SharedBuffers, "200MB"}, // 2024 is converted to 200MB
					{MaxReplicationSlots, strconv.Itoa(pg.Spec.PostgresConfiguration.MaxReplicationSlots)},
					{MaxWALSenders, strconv.Itoa(pg.Spec.PostgresConfiguration.MaxWALSenders)},
					{MaxLocksPerTransaction, strconv.Itoa(*pg.Spec.PostgresConfiguration.MaxLocksPerTransaction)},
				}

				for _, setting := range expectedConfig {
					Expect(client.CheckParameter(
						ctx,
						setting.parameter,
						setting.value,
					)).To(Succeed(),
						fmt.Sprintf("the custom parameter was not what we expected for %s/%s",
							instance.GetNamespace(), instance.GetName()))
				}
			})
		})

		It("Custom configuration can be updated on a running PostgreSQL instance", func() {
			By("creating a PostgreSQL instance with implicit defaults", func() {
				instance, err = dsi.New(
					dataservice,
					testingNamespace,
					framework.GenerateName(instanceNamePrefix,
						GinkgoParallelProcess(),
						suffixLength),
					replicas,
				)
				Expect(err).To(BeNil(), "failed to generate new DSI resource")

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

				// Fetch admin secret for privileged DSIClient. We need a
				// privileged client since some parameters such as SSLCiphers can
				// not be fetched by service binding users.
				adminSecretData, err = secret.AdminSecretData(
					ctx, k8sClient, instance.GetName(), instance.GetNamespace())
				Expect(err).To(BeNil(),
					fmt.Sprintf("failed to parse secret data of admin credentials for %s/%s",
						instance.GetNamespace(), instance.GetName()))

				// Create client for interacting with the new instance.
				client, err = dsi.NewClient(
					dataservice, strconv.Itoa(localPort), adminSecretData)
				Expect(err).To(BeNil(), "failed to create new dsi client")
			})

			By("setting custom parameters of the retrieved PostgreSQL object", func() {
				newInstance := postgresql.NewEmpty()
				Expect(k8sClient.Get(ctx, types.NamespacedName{
					Name:      instance.GetName(),
					Namespace: instance.GetNamespace(),
				},
					newInstance.GetClientObject())).
					To(Succeed(), fmt.Sprintf("failed to get instance %s/%s",
						instance.GetNamespace(), instance.GetName()))

				pg, ok = newInstance.GetClientObject().(*v1alpha1.Postgresql)
				Expect(ok).To(BeTrue(),
					"failed to cast object interface to PostgreSQL struct")

				setCustomPostgresConfig(pg)
			})

			By("updating the live PostgreSQL instance with custom configuration", func() {
				Expect(k8sClient.Update(ctx, pg)).
					To(Succeed(), fmt.Sprintf("failed to update instance %s/%s",
						instance.GetNamespace(), instance.GetName()))
			})

			// Parameters such as max_connections, shared_buffers, max_replication_slots,
			// max_locks_per_transaction and max_wal_senders will require Patroni to restart the
			// PostgreSQL process. We need to ensure that PostgreSQL has been successfully
			// restarted by Patroni before we continue with our assertions. If we are
			// unlucky PostgreSQL can be down at the time we make our first parameter
			// assertion. In this case the portforward will break and not recover.
			// Therefore we introduce this retry logic to reduce the
			// possibility of flaky tests.
			By("ensuring the PostgreSQL process has restarted", func() {
				Eventually(func() error {
					portForwardStopCh, localPort, err = framework.PortForward(
						ctx, instancePort, kubeconfigPath, instance, k8sClient)
					Expect(err).To(BeNil(),
						fmt.Sprintf("failed to establish portforward to DSI %s/%s",
							instance.GetNamespace(), instance.GetName()))

					// Create client for interacting with the new instance.
					client, err = dsi.NewClient(
						dataservice, strconv.Itoa(localPort), adminSecretData)
					Expect(err).To(BeNil(), "failed to create new dsi client")

					// This check is simply a probe to ensure that the PostgreSQL
					// process has restarted. We still explicitly check this
					// parameter in the table driven tests below for the sake
					// of verbosity.
					probeErr := client.CheckParameter(ctx, ArchiveTimeout,
						strconv.Itoa(pg.Spec.PostgresConfiguration.ArchiveTimeoutSeconds)+"s")

					if probeErr != nil {
						close(portForwardStopCh)
						return probeErr
					}
					return probeErr
				}, framework.AsyncOpsTimeoutMins, 1*time.Second).Should(Succeed(),
					fmt.Sprintf("unable to wait for PostgreSQL process restart for %s/%s",
						instance.GetNamespace(), instance.GetName()))
			})

			By("checking that the custom config is set correctly", func() {
				var expectedConfig = []struct {
					parameter, value string
				}{
					{ArchiveTimeout, strconv.Itoa(pg.Spec.PostgresConfiguration.ArchiveTimeoutSeconds) + "s"},
					{TempFileLimit, strconv.Itoa(pg.Spec.PostgresConfiguration.TempFileLimitKiloBytes)},
					{TrackIOTiming, pg.Spec.PostgresConfiguration.TrackIOTiming},
					{StatementTimeout, strconv.Itoa(pg.Spec.PostgresConfiguration.StatementTimeoutMillis) + "ms"},
					{ClientMinMessages, pg.Spec.PostgresConfiguration.ClientMinMessages},
					{LogMinMessages, pg.Spec.PostgresConfiguration.LogMinMessages},
					{LogMinErrorStatement, pg.Spec.PostgresConfiguration.LogMinErrorStatement},
					{LogStatement, pg.Spec.PostgresConfiguration.LogStatement},
					{LogErrorVerbosity, strings.ToLower(pg.Spec.PostgresConfiguration.LogErrorVerbosity)},
					{SSLCiphers, pg.Spec.PostgresConfiguration.SSLCiphers},
					{SSLMinProtocolVersion, pg.Spec.PostgresConfiguration.SSLMinProtocolVersion},
					{WALWriterDelay, strconv.Itoa(pg.Spec.PostgresConfiguration.WALWriterDelayMillis) + "ms"},
					{SynchronousCommit, pg.Spec.PostgresConfiguration.SynchronousCommit},
					{MaxConnections, strconv.Itoa(pg.Spec.PostgresConfiguration.MaxConnections)},
					// SharedBuffers is not being set or updated.
					// https://github.com/anynines/postgresql-operator/issues/75
					// {SharedBuffers, "200MB"}, // 2024 is converted to 200MB
					{MaxReplicationSlots, strconv.Itoa(pg.Spec.PostgresConfiguration.MaxReplicationSlots)},
					{MaxWALSenders, strconv.Itoa(pg.Spec.PostgresConfiguration.MaxWALSenders)},
					{MaxLocksPerTransaction, strconv.Itoa(*pg.Spec.PostgresConfiguration.MaxLocksPerTransaction)},
				}

				for _, setting := range expectedConfig {
					// Eventually is used to avoid failing when PostgreSQL is
					// still in the process of restarting due to parameters that
					// require restart.
					Eventually(func() error {
						return client.CheckParameter(
							ctx,
							setting.parameter,
							setting.value)
					}, framework.AsyncOpsTimeoutMins, 1*time.Second).Should(Succeed(),
						fmt.Sprintf("unable to check custom config is set correctly on update for %s/%s",
							instance.GetNamespace(), instance.GetName()))

				}
			})

			By("emitting an event about the configuration update", func() {
				// We want to ensure that eventually the events for the updated object are present
				// in the Kubernetes API. Expecting the events to be available on the first list
				// operation is overly optimistic so we wait for them to become available before
				// making further assertions in order to reduce flakiness.
				events := &corev1.EventList{}
				Eventually(func() bool {
					Expect(k8sClient.List(ctx, events, &ctrlruntimeclient.ListOptions{
						FieldSelector: fields.AndSelectors(
							fields.OneTermEqualSelector("reason", "Updated"),
							fields.OneTermEqualSelector("involvedObject.uid",
								string(instance.GetUID())))})).To(Succeed(),
						"failed to list events emitted for the config update of the DSI")

					return len(events.Items) > 0
				}, framework.AsyncOpsTimeoutMins, 1*time.Second).Should(BeTrue(),
					fmt.Sprintf("failed to list events emitted for the config update of %s/%s",
						instance.GetNamespace(), instance.GetName()))

				Expect(len(events.Items)).To(Equal(1), "exactly one event should be emitted for "+
					"the config update of a DSI, found more than one")

				event := events.Items[0]
				Expect(event.Message).To(Equal("Successfully updated the PostgreSQL configuration"),
					"wrong event message")
				Expect(event.Type).To(Equal(corev1.EventTypeNormal), "wrong event type")
				Expect(event.Count).To(Equal(int32(1)), "wrong event count")
				Expect(event.Source.Component).To(Equal("postgresql-controller"),
					"wrong event source.component")
				Expect(event.InvolvedObject.Kind).To(Equal("Postgresql"),
					"wrong event involvedObject.kind")
				Expect(event.InvolvedObject.APIVersion).
					To(Equal("postgresql.anynines.com/v1alpha1"),
						"wrong event involvedObject.apiVersion")
			})
		})
	})
})

func setCustomPostgresConfig(pg *v1alpha1.Postgresql) {
	maxLocksPerTransaction := 120

	pg.Spec.PostgresConfiguration.MaxConnections = 101
	pg.Spec.PostgresConfiguration.MaxLocksPerTransaction = &maxLocksPerTransaction
	// SharedBuffers is not being set or updated.
	// https://github.com/anynines/postgresql-operator/issues/75
	pg.Spec.PostgresConfiguration.SharedBuffers = 200
	pg.Spec.PostgresConfiguration.MaxReplicationSlots = 11
	pg.Spec.PostgresConfiguration.MaxWALSenders = 11
	pg.Spec.PostgresConfiguration.StatementTimeoutMillis = 2147483647
	pg.Spec.PostgresConfiguration.SSLCiphers = "high:medium:+3des:!anull"
	pg.Spec.PostgresConfiguration.SSLMinProtocolVersion = "TLSv1.2"
	pg.Spec.PostgresConfiguration.TempFileLimitKiloBytes = 0
	pg.Spec.PostgresConfiguration.WALWriterDelayMillis = 201
	pg.Spec.PostgresConfiguration.SynchronousCommit = "off"
	pg.Spec.PostgresConfiguration.TrackIOTiming = "on"
	pg.Spec.PostgresConfiguration.ArchiveTimeoutSeconds = 10
	pg.Spec.PostgresConfiguration.ClientMinMessages = "warning"
	pg.Spec.PostgresConfiguration.LogMinMessages = "notice"
	pg.Spec.PostgresConfiguration.LogMinErrorStatement = "warning"
	pg.Spec.PostgresConfiguration.LogStatement = "all"
	pg.Spec.PostgresConfiguration.LogErrorVerbosity = "DEFAULT"
}
