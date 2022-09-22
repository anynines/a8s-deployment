package postgresql

import (
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/anynines/a8s-deployment/test/framework"
	"github.com/anynines/a8s-deployment/test/framework/chaos"
	"github.com/anynines/a8s-deployment/test/framework/dsi"
	"github.com/anynines/a8s-deployment/test/framework/postgresql"
	"github.com/anynines/a8s-deployment/test/framework/secret"
	"github.com/anynines/a8s-deployment/test/framework/servicebinding"
	sbv1alpha1 "github.com/anynines/a8s-service-binding-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	instancePort = 5432
	replicas     = 3
	suffixLength = 5

	// entity is a generic term to describe where data services store their data.
	entity = "test_entity"

	// asyncOpsTimeoutMins is the amount of minutes after which assertions fail if the condition
	// they check has not become true. Needed because some conditions might become true only after
	// some time, so we need to check them asynchronously.
	// TODO: Make asyncOpsTimeoutMins an invocation parameter.
	asyncOpsTimeoutMins = time.Minute * 5
)

var (
	// portForwardStopCh is the channel to close to terminate a port forward
	portForwardStopCh chan struct{}
	localPort         int

	sb                 *sbv1alpha1.ServiceBinding
	serviceBindingData secret.SecretData
	instance           *postgresql.Postgresql
	client             dsi.DSIClient
)

var _ = Describe("PostgreSQL Chaos tests", func() {
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
		Expect(k8sClient.Delete(ctx, instance.GetClientObject())).To(Succeed(),
			fmt.Sprintf("failed to delete instance %s/%s",
				instance.GetNamespace(), instance.GetName()))
		Expect(k8sClient.Delete(ctx, sb)).To(Succeed(),
			fmt.Sprintf("failed to delete service binding %s/%s",
				sb.GetNamespace(), sb.GetName()))
		dsi.WaitForDeletion(ctx, instance.GetClientObject(), k8sClient)
		//TODO: Wait for deletion for all secondary objects
	})

	It("No failover to replica with critical replication lag", func() {
		dsi.WaitForReplicaReadiness(ctx, instance.GetClientObject(), k8sClient, replicas)

		// The chaos operator checks for matching pods only at time of applying
		// chaos, thus all pods need to be running and Patroni needs to have
		// labels assigned to them
		// TODO: As soon as a meaningful startup probe is implemented, this step
		// should become unnecessary.
		By("Waiting for all PostgreSQL pods to get assigned labels by Patroni", func() {
			var err error
			Eventually(func() bool {
				var ready bool
				ready, err = instance.CheckPatroniLabelsAssigned(ctx, k8sClient)
				if err != nil {
					return false
				}
				return ready
			}, asyncOpsTimeoutMins).Should(BeTrue(),
				fmt.Sprintf("timeout reached waiting for labels to be assigned to "+
					"instance %s/%s: %s",
					instance.GetNamespace(),
					instance.GetName(),
					err,
				),
			)
		})

		pgChaosInjector := chaos.PgInjector{Instance: instance}

		var replicaStop chaos.ChaosObject
		By("Stop all replicas by applying PodChaos", func() {
			var err error
			replicaStop, err = pgChaosInjector.StopReplicas(ctx, k8sClient)
			Expect(err).To(BeNil(),
				fmt.Sprintf("timeout reached waiting for chaos to apply to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		By("Wait for PodChaos to apply", func() {
			var err error
			Eventually(func() bool {
				var ready bool
				ready, err = replicaStop.CheckChaosActive(ctx, k8sClient)
				if err != nil {
					return false
				}

				return ready
			}, asyncOpsTimeoutMins).Should(BeTrue(),
				fmt.Sprintf("timeout reached waiting for chaos to apply to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		// Create critical replication lag
		// Spilo image default maximum_lag_on_failover: 33554432
		// Source:
		// https://github.com/zalando/spilo/blob/cdae614e71b04ccbbd9e53f684c8a5a30afd08fa/postgres-appliance/scripts/configure_spilo.py#L195
		// Using a random String of length 100000 (size of 1 char = 1 byte,
		// ignoring length byte here) we need to generate at least 336
		// entries to reach critical replication lag
		// TODO: replace with more efficient insertion strategy when Postgres
		// specific client is accessible in the tests and retrieve the max
		// replication lag dynamically from PG
		var writtenData string
		By("Writing random data", func() {
			// More data can help to ensure critical lag in the next to steps
			for i := 0; i < 500; i++ {
				if i != 0 {
					writtenData += "\n"
				}
				randString := framework.GenerateRandString(100000)
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
				fmt.Sprintf("failed to read data from  DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)

			Expect(readData).To(Equal(writtenData),
				fmt.Sprintf("read data does not match data written to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		// Restart the replicas with available master so that they can pick up
		// on their replication lag
		By("Restart replicas by deleting PodChaos", func() {
			Expect(k8sClient.Delete(ctx, replicaStop.KubernetesObject())).To(Succeed(),
				fmt.Sprintf("failed to delete chaos applied to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		// Wait for replicas to become available
		dsi.WaitForReplicaReadiness(ctx, instance.GetClientObject(), k8sClient, replicas)

		// This timing is critical : We need to ensure the replicas have enough
		// time to connect to the master and get their replication delay while
		// simultaneously not giving them enough time to catch up.
		//
		// If this becomes a source of flakiness, you can increase the amount of
		// data that is written and increase the delay. Otherwise limiting the
		// bandwidth to the replicas with the  help of NetworkChaos would be an
		// option.
		time.Sleep(100 * time.Millisecond)

		// Stop the master before replicas can catch up
		var masterStop chaos.ChaosObject
		By("Stop the master by applying PodChaos", func() {
			masterStop, err = pgChaosInjector.StopMaster(ctx, k8sClient)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to apply chaos to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		// Give replicas time to potentially perform a failover, this should not
		// happen as the replication lag should be to high. This value is set
		// very pessimistically.
		time.Sleep(30 * time.Second)

		// If this check passes, Patroni behaved as expected. A new master was
		// not elected since the replicas had reached critical replication lag

		By("Checking if a new master has been elected", func() {
			masterPods, err := dsi.GetPodsWithLabels(ctx, k8sClient, instance.GetNamespace(),
				instance.GetMasterLabels())

			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to list master pods of DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)

			Expect(dsi.NPodsReady(masterPods)).To(BeZero(),
				fmt.Sprintf("leader election in DSI %s/%s occurred even though"+
					"max_replication_lag exceeded",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		// Ensure recovery as soon as the master comes back online
		By("Restart master by deleting PodChaos", func() {
			Expect(k8sClient.Delete(ctx, masterStop.KubernetesObject())).To(Succeed(),
				fmt.Sprintf("failed to delete chaos applied to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		// Wait for master readiness
		dsi.WaitForReplicaReadiness(ctx, instance.GetClientObject(), k8sClient, replicas)

		// Wait for propagation of data to the replicas
		// TODO alternative: check replication lag in the replica with specific
		// pg client
		time.Sleep(20 * time.Second)

		// Check replica data propagation
		By("Ensuring data was propagated to replicas", func() {

			// TODO: use replica service when implemented
			replicaPods, err := dsi.GetPodsWithLabels(ctx, k8sClient, instance.GetNamespace(),
				instance.GetReplicaLabels())
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to list master pods of DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)

			Expect(len(replicaPods.Items) > 0).To(BeTrue(),
				fmt.Sprintf("no replicas found for DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)

			replicaPod := &replicaPods.Items[0]
			replicaPortForwardStopCh, replicaLocalPort, err := framework.PortForwardPod(
				ctx, instancePort, kubeconfigPath, replicaPod, k8sClient)
			defer func() { close(replicaPortForwardStopCh) }()

			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to open port forward to replica of DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)

			replicaClient, err := dsi.NewClient(dataservice,
				strconv.Itoa(replicaLocalPort), serviceBindingData)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to create client to DSI %s/%s connecting to replica",
					instance.GetNamespace(),
					instance.GetName()),
			)

			readData, err := replicaClient.Read(ctx, entity)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to read replica data of DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
			Expect(readData).To(Equal(writtenData),
				fmt.Sprintf("read data does not match data written to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

	})

	It("Failed Master rejoins as replica", func() {
		dsi.WaitForReplicaReadiness(ctx, instance.GetClientObject(), k8sClient, replicas)

		var masterPod *corev1.Pod
		By("Selecting master Pod", func() {

			masterPodList, err := dsi.GetPodsWithLabels(ctx, k8sClient, instance.GetNamespace(),
				instance.GetMasterLabels())
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to select master pods of DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)

			Expect(len(masterPodList.Items)).To(BeEquivalentTo(1), "invalid number of masters")

			masterPod = &masterPodList.Items[0]
		})

		var writtenData string
		By("Writing random data to master", func() {
			for i := 0; i < 50; i++ {
				if i != 0 {
					writtenData += "\n"
				}
				randString := framework.GenerateRandString(1000)
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
				fmt.Sprintf("failed to read data from  DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)

			Expect(readData).To(Equal(writtenData),
				fmt.Sprintf("read data does not match data written to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		pgChaosInjector := chaos.PgInjector{Instance: instance}

		// ensure data had time to propagate to replicas
		// TODO: This could be removed if we can check for replication lag
		time.Sleep(1 * time.Second)

		var masterStop chaos.ChaosObject
		By("Stop the master by applying PodChaos", func() {
			masterStop, err = pgChaosInjector.StopMaster(ctx, k8sClient)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to apply chaos to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		By("Wait for chaos to apply", func() {
			var err error
			Eventually(func() bool {
				err := k8sClient.Get(ctx,
					types.NamespacedName{Name: masterPod.Name, Namespace: instance.GetNamespace()},
					masterPod)
				Expect(err).To(BeNil(),
					fmt.Sprintf("failed to get pod %s of DSI %s/%s",
						masterPod.Name,
						instance.GetNamespace(),
						instance.GetName()),
				)
				return !dsi.IsPodReady(masterPod)
			}, asyncOpsTimeoutMins).Should(BeTrue(),
				fmt.Sprintf("timeout reached waiting for chaos to apply to DSI %s/%s: %s",
					instance.GetNamespace(),
					instance.GetName(),
					err,
				),
			)
		})

		By("Waiting for failover to happen", func() {
			Eventually(func() int {
				masterPods, err := dsi.GetPodsWithLabels(ctx, k8sClient, instance.GetNamespace(),
					instance.GetMasterLabels())

				Expect(err).To(BeNil(),
					fmt.Sprintf("failed to list master pods of DSI %s/%s",
						instance.GetNamespace(),
						instance.GetName()),
				)

				return dsi.NPodsReady(masterPods)
			}, asyncOpsTimeoutMins).Should(BeEquivalentTo(1),
				fmt.Sprintf("timeout reached while waiting for new master of DSI %s/%s"+
					"to be elected",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		// recreate port forward to new master
		// TODO: After rework of port forwarding logic, this step should be unnecessary
		By("Recreating port forward for new master", func() {
			close(portForwardStopCh)

			masterPods, err := dsi.GetPodsWithLabels(ctx, k8sClient, instance.GetNamespace(),
				instance.GetMasterLabels())
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to list master pods of DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)

			var newMasterPod *corev1.Pod
			for _, pod := range masterPods.Items {
				if dsi.IsPodReady(&pod) {
					newMasterPod = &pod
				}
			}

			portForwardStopCh, localPort, err = framework.PortForwardPod(
				ctx, instancePort, kubeconfigPath, newMasterPod, k8sClient)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to recreate port forward to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)

			client, err = dsi.NewClient(dataservice, strconv.Itoa(localPort), serviceBindingData)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to recreate dsi client to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		By("Writing more random data to new master", func() {
			for i := 0; i < 10; i++ {
				writtenData += "\n"
				randString := framework.GenerateRandString(1000)
				Expect(client.Write(ctx, entity, randString)).To(Succeed(),
					fmt.Sprintf("failed to insert data in DSI %s/%s",
						instance.GetNamespace(),
						instance.GetName()),
				)
				writtenData += randString
			}
		})

		By("Ensuring data was written successfully to new master", func() {
			readData, err := client.Read(ctx, entity)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to read data from  DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)

			Expect(readData).To(Equal(writtenData),
				fmt.Sprintf("read data does not match data written to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		By("Restart old master by deleting chaos", func() {
			Expect(k8sClient.Delete(ctx, masterStop.KubernetesObject())).To(Succeed(),
				fmt.Sprintf("failed to delete chaos applied to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		dsi.WaitForReplicaReadiness(ctx, instance.GetClientObject(), k8sClient, replicas)

		By("Ensure old master returns as replica", func() {
			Eventually(func() bool {
				err := k8sClient.Get(ctx,
					types.NamespacedName{Name: masterPod.Name, Namespace: instance.GetNamespace()},
					masterPod)
				Expect(err).To(BeNil(),
					fmt.Sprintf("failed to get pod %s of DSI %s/%s",
						masterPod.Name,
						instance.GetNamespace(),
						instance.GetName()),
				)
				return !postgresql.IsMaster(masterPod)
			}, asyncOpsTimeoutMins).Should(BeTrue(),
				fmt.Sprintf("timed out while waiting for former master pod %s of DSI %s/%s "+
					"to rejoin the cluster as replica",
					masterPod.Name,
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

		// wait for data to be propagated to old master
		// TODO: This could be removed if we can check for replication lag
		time.Sleep(5 * time.Second)

		By("Ensuring data is readable from master", func() {
			masterPortForwardStopCh, masterLocalPort, err := framework.PortForwardPod(
				ctx, instancePort, kubeconfigPath, masterPod, k8sClient)
			defer func() { close(masterPortForwardStopCh) }()

			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to open port forward to replica pod %s of DSI %s/%s",
					masterPod.Name,
					instance.GetNamespace(),
					instance.GetName()),
			)

			replicaClient, err := dsi.NewClient(dataservice, strconv.Itoa(masterLocalPort),
				serviceBindingData)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to create DSI client for pod %s of DSI %s/%s",
					masterPod.Name,
					instance.GetNamespace(),
					instance.GetName()),
			)

			readData, err := replicaClient.Read(ctx, entity)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to read data from replica pod %s of DSI %s/%s",
					masterPod.Name,
					instance.GetNamespace(),
					instance.GetName()),
			)
			Expect(readData).To(Equal(writtenData),
				fmt.Sprintf("read data from replica pod %s of DSI %s/%s doesn't match written data",
					masterPod.Name,
					instance.GetNamespace(),
					instance.GetName()),
			)
		})

	})
})
