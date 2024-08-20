package backuppolicy

import (
	"fmt"
	"time"

	backupv1beta3 "github.com/anynines/a8s-backup-manager/api/v1beta3"
	"github.com/anynines/a8s-deployment/test/framework"
	policy "github.com/anynines/a8s-deployment/test/framework/backuppolicy"
	"github.com/anynines/a8s-deployment/test/framework/dsi"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	replicas            = 1
	suffixLength        = 5
	pollingPeriod       = 1 * time.Second
	asyncOpsTimeoutMins = time.Minute * 2
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
					policy.SetEnabled(true),
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

				EventuallyWithOffset(1, func() bool {
					createdBuckup := &backupv1beta3.BackupPolicy{}

					if err = k8sClient.Get(ctx, types.NamespacedName{
						Name:      backupPolicy.GetName(),
						Namespace: backupPolicy.GetNamespace(),
					}, createdBuckup); err != nil {
						return false
					}

					return createdBuckup.Status.BackupsCount > 0
				}, asyncOpsTimeoutMins, pollingPeriod).Should(BeTrue(),
					fmt.Sprintf("timeout reached waiting for fetching backupPolicy %s/%s: %s",
						backupPolicy.GetNamespace(),
						backupPolicy.GetName(),
						err,
					),
				)
			})
		})

		It("Should Create BackupPolicy", func() {
			By("Create Buckups based on Schedule Config", func() {
				backupPolicy = policy.New(
					policy.SetNamespacedName(testingNamespace),
					policy.SetEnabled(true),
					policy.SetInstanceRef(instance),
					policy.ScheduleConfiguration(backupv1beta3.ScheduleConfiguration{
						Type:     backupv1beta3.ScheduleTypeCron,
						TimeZone: "UTC",
						Cron: backupv1beta3.Cron{
							Expression: "0/1 * * * *",
						},
					}))
				Expect(k8sClient.Create(ctx, backupPolicy)).To(Succeed(),
					"failed to create backup policy")

				backups := &backupv1beta3.BackupList{}

				EventuallyWithOffset(1, func() bool {
					createdBuckup := &backupv1beta3.BackupPolicy{}
					if err = k8sClient.Get(ctx, types.NamespacedName{
						Name:      backupPolicy.GetName(),
						Namespace: backupPolicy.GetNamespace(),
					}, createdBuckup); err != nil {
						return false
					}
					lisOptions := []client.ListOption{
						client.InNamespace(backupPolicy.Namespace),
						client.MatchingLabels{backupv1beta3.BackupPolicyNameLabelKey: backupPolicy.Name},
					}

					if err = k8sClient.List(ctx, backups, lisOptions...); err != nil {
						return false
					}

					return len(backups.Items) >= 3
				}, asyncOpsTimeoutMins, pollingPeriod).Should(BeTrue(),
					fmt.Sprintf("timeout reached waiting for fetching backupPolicy %s/%s: %s",
						backupPolicy.GetNamespace(),
						backupPolicy.GetName(),
						err,
					),
				)
			})
		})
	})

	Context("Retention Configuration", func() {
		It("Should retain backups based on count", func() {
			By("Create a policy", func() {
				backupPolicy = policy.New(
					policy.SetNamespacedName(testingNamespace),
					policy.SetEnabled(true),
					policy.SetInstanceRef(instance),
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

			By("Create existing backups", func() {
				for _, backup := range policy.GenerateBackups(backupPolicy, instance) {
					Expect(k8sClient.Create(ctx, &backup)).To(Succeed(),
						"failed to create existing backups")
				}
			})

			By("deleting backups over the requested count", func() {
				var err error
				backups := &backupv1beta3.BackupList{}
				EventuallyWithOffset(1, func() bool {
					lisOptions := []client.ListOption{
						client.InNamespace(backupPolicy.Namespace),
						client.MatchingLabels{backupv1beta3.BackupPolicyNameLabelKey: backupPolicy.Name},
					}

					if err = k8sClient.List(ctx, backups, lisOptions...); err != nil {
						return false
					}

					return len(backups.Items) == 2
				}, asyncOpsTimeoutMins, pollingPeriod).Should(BeTrue(),
					fmt.Sprintf("timeout reached waiting for correct backups count %s/%s: %s",
						backupPolicy.GetNamespace(),
						backupPolicy.GetName(),
						err,
					),
				)
				Expect(backups.Items).To(HaveLen(2), "backups cleanup is unsuccessful")
			})
		})
	})
})
