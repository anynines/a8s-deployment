package backuppolicy

import (
	"fmt"

	backupv1beta3 "github.com/anynines/a8s-backup-manager/api/v1beta3"
	"github.com/anynines/a8s-deployment/test/framework"
	policy "github.com/anynines/a8s-deployment/test/framework/backuppolicy"
	"github.com/anynines/a8s-deployment/test/framework/dsi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"
)

const (
	replicas     = 1
	suffixLength = 5
)

var (
	backupPolicy *backupv1beta3.BackupPolicy
	instance     dsi.Object
)

var _ = Describe("BackupPolicy", func() {
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
	})

	AfterEach(func() {

		Expect(k8sClient.Delete(ctx, backupPolicy)).To(Succeed(),
			fmt.Sprintf("failed to delete backup policy %s/%s",
				backupPolicy.GetNamespace(), backupPolicy.GetName()))
		Expect(k8sClient.Delete(ctx, instance.GetClientObject())).To(Succeed(),
			fmt.Sprintf("failed to delete instance %s/%s",
				instance.GetNamespace(), instance.GetName()))
		Expect(policy.CleanupBackups(ctx, backupPolicy, k8sClient)).To(Succeed(), "failed to delete backups")
		dsi.WaitForDeletion(ctx, instance.GetClientObject(), k8sClient)
	})

	Context("Backup Policy", func() {
		It("Should Create BackupPolicy", func() {
			By("Create a policy", func() {
				backupPolicy = policy.New(
					policy.SetNamespacedName(testingNamespace),
					policy.SetEnabled(false),
					policy.SetInstanceRef(instance),
					policy.ScheduleConfiguration(backupv1beta3.ScheduleConfiguration{
						Type:     backupv1beta3.ScheduleTypeCron,
						TimeZone: "UTC",
						Cron: backupv1beta3.Cron{
							Expression: "2 2 * * *",
						},
					}))
				Expect(k8sClient.Create(ctx, backupPolicy)).To(Succeed(),
					"failed to create backup policy")
			})
		})
	})

	Context("Retention Configuration", func() {
		It("Should retain backups based on count", func() {
			By("Create a policy", func() {
				backupPolicy = policy.New(
					policy.SetNamespacedName(testingNamespace),
					policy.SetEnabled(true),
					policy.ScheduleConfiguration(backupv1beta3.ScheduleConfiguration{
						Type:     backupv1beta3.ScheduleTypeCron,
						TimeZone: "UTC",
						Cron: backupv1beta3.Cron{
							Expression: "*/1 * * * *",
						},
					}),
					policy.RetentionConfiguration(&backupv1beta3.RetentionConfiguration{
						Count: ptr.To(2),
					}))
				Expect(k8sClient.Create(ctx, backupPolicy)).To(Succeed(),
					"failed to create backup policy")
			})

			By("Create existing policies", func() {
				for _, backup := range policy.GetExistingBackups(backupPolicy, instance) {
					Expect(k8sClient.Create(ctx, &backup)).To(Succeed(),
						"failed to create existing backups")
				}
			})

			By("Backups should be deleted based on count", func() {
				backups := policy.CheckBackups(ctx, backupPolicy, k8sClient, 2)
				Expect(backups.Items).To(HaveLen(2), "backups cleanup is unsuccessful")
			})
		})
	})
})
