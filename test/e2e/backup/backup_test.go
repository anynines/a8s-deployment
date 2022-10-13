package backup

import (
	"fmt"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	backupv1alpha1 "github.com/anynines/a8s-backup-manager/api/v1alpha1"
	"github.com/anynines/a8s-deployment/test/framework"
	bkp "github.com/anynines/a8s-deployment/test/framework/backup"
	"github.com/anynines/a8s-deployment/test/framework/dsi"
	rst "github.com/anynines/a8s-deployment/test/framework/restore"
	"github.com/anynines/a8s-deployment/test/framework/secret"
	"github.com/anynines/a8s-deployment/test/framework/servicebinding"
	sbv1alpha1 "github.com/anynines/a8s-service-binding-controller/api/v1alpha1"
)

const (
	instancePort = 5432
	replicas     = 1
	suffixLength = 5

	// TODO: Make configurable and generalizable using Data interface
	// testInput is data input used for testing data service functionality.
	testInput = "test_input"
	// entity is a generic term to decribe where data services store their data.
	entity = "test_entity"
)

var (
	// portForwardStopCh is the channel to close to terminate a port forward
	portForwardStopCh chan struct{}
	localPort         int

	sb       *sbv1alpha1.ServiceBinding
	backup   *backupv1alpha1.Backup
	restore  *backupv1alpha1.Restore
	instance dsi.Object
	client   dsi.DSIClient
)

var _ = Describe("Backup", func() {
	BeforeEach(func() {
		// Create Dataservice instance and wait for instance readiness
		instance, err = dsi.New(
			dataservice,
			testingNamespace,
			framework.GenerateName(instanceNamePrefix, GinkgoParallelProcess(), suffixLength),
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

		// Create service binding for instance.
		sb = servicebinding.New(
			servicebinding.SetNamespacedName(instance.GetClientObject()),
			servicebinding.SetInstanceRef(instance.GetClientObject()),
		)
		Expect(k8sClient.Create(ctx, sb)).
			To(Succeed(), fmt.Sprintf("failed to create new servicebinding for DSI %s/%s",
				instance.GetNamespace(), instance.GetName()))
		servicebinding.WaitForReadiness(ctx, sb, k8sClient)
		serviceBindingData, err := secret.Data(
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
		Expect(k8sClient.Delete(ctx, backup)).To(Succeed(),
			fmt.Sprintf("failed to delete backup %s/%s",
				backup.GetNamespace(), backup.GetName()))
		Expect(k8sClient.Delete(ctx, restore)).To(Succeed(),
			fmt.Sprintf("failed to delete restore %s/%s",
				restore.GetNamespace(), restore.GetName()))
		dsi.WaitForDeletion(ctx, instance.GetClientObject(), k8sClient)
		//TODO: Wait for deletion for all secondary objects
	})

	It("Performs backup and restore of instance", func() {
		var beforeBackup string
		By("Writing data", func() {
			Expect(client.Write(ctx, entity, testInput)).
				To(Succeed(), "failed to insert data")
		})

		By("Ensuring data was written succesfully", func() {
			beforeBackup, err = client.Read(ctx, entity)
			Expect(err).To(BeNil(), "failed to read data")
			Expect(beforeBackup).To(Equal(testInput),
				"read data does not match test input")
		})

		By("Taking a backup", func() {
			backup = bkp.New(
				bkp.SetNamespacedName(instance),
				bkp.SetInstanceRef(instance.GetClientObject()),
			)
			Expect(k8sClient.Create(ctx, backup)).To(Succeed(),
				fmt.Sprintf("failed to create backup for DSI %s/%s",
					instance.GetNamespace(), instance.GetName()))
			bkp.WaitForReadiness(ctx, backup, framework.AsyncOpsTimeoutMins, k8sClient)
		})

		By("Writing more data", func() {
			Expect(client.Write(ctx, entity, testInput)).
				To(Succeed(), "failed to insert data")
		})

		By("Ensuring we have written another entry correctly", func() {
			data, err := client.Read(ctx, entity)
			Expect(err).To(BeNil(), "failed to read data")

			entries := strings.Split(data, "\n")
			Expect(len(entries)).To(Equal(2),
				"The database entity does not have the entries we expect")
			// Can we just compare a single big string representation of the table
			for _, s := range entries {
				Expect(s).To(Equal(testInput), "data read is not of the expected form")
			}
		})

		By("Restoring the instance from a backup", func() {
			restore = rst.New(
				rst.SetInstanceRef(instance.GetClientObject()),
				rst.SetNamespacedName(instance),
				rst.SetBackupName(backup.GetName()),
			)
			Expect(k8sClient.Create(ctx, restore)).To(Succeed(),
				fmt.Sprintf("failed to create restore for %s/%s",
					restore.GetNamespace(), restore.GetName()))
			rst.WaitForReadiness(ctx, restore, k8sClient)
		})

		By("Ensuring that the original data from the backup matches the data restored", func() {
			afterRestore, err := client.Read(ctx, entity)
			Expect(err).To(BeNil(), "failed to read data")
			Expect(beforeBackup).To(Equal(afterRestore),
				"restored data does not match data taken at backup")
		})
	})
})
