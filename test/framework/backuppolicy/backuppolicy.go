package backuppolicy

import (
	"context"
	"fmt"
	"time"

	"github.com/anynines/a8s-backup-manager/api/v1beta3"
	"github.com/anynines/a8s-deployment/test/framework"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// asyncOpsTimeoutMins is the amount of minutes after which assertions fail if the condition
	// they check has not become true. Needed because some conditions might become true only
	// after some time, so we need to check them asynchronously.
	// TODO: Make asyncOpsTimeoutMins an invocation parameter.
	asyncOpsTimeoutMins = time.Minute * 5
	suffixLength        = 6
	pollingPeriod       = 1 * time.Second
)

// Option represents a functional option for backup objects. To learn what a functional option is,
// read here: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
type Option func(*v1beta3.BackupPolicy)

func SetNamespacedName(namespace string) Option {
	return func(p *v1beta3.BackupPolicy) {
		p.Name = framework.UniqueName("backup-policy", suffixLength)
		p.Namespace = namespace
	}
}

func SetInstanceRef(dsi runtimeClient.Object) Option {
	return func(p *v1beta3.BackupPolicy) {
		p.Spec.ServiceInstance.APIGroup = dsi.GetObjectKind().GroupVersionKind().Group
		p.Spec.ServiceInstance.Kind = dsi.GetObjectKind().GroupVersionKind().Kind
		p.Spec.ServiceInstance.Name = dsi.GetName()
	}
}

func SetEnabled(enabled bool) Option {
	return func(p *v1beta3.BackupPolicy) {
		p.Spec.Enabled = enabled
	}
}

func ScheduleConfiguration(scheduleConfig v1beta3.ScheduleConfiguration) Option {
	return func(p *v1beta3.BackupPolicy) {
		p.Spec.ScheduleConfig = scheduleConfig
	}
}

func RetentionConfiguration(retentionConfig *v1beta3.RetentionConfiguration) Option {
	return func(p *v1beta3.BackupPolicy) {
		p.Spec.RetentionConfig = retentionConfig
	}
}

func New(opts ...Option) *v1beta3.BackupPolicy {
	p := &v1beta3.BackupPolicy{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func GenerateBackups(p *v1beta3.BackupPolicy, dsi runtimeClient.Object) []v1beta3.Backup {
	existingBackups := []v1beta3.Backup{
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              framework.UniqueName("backup", suffixLength),
				Namespace:         p.Namespace,
				CreationTimestamp: v1.Time{Time: mustParseTime("2024-07-03T00:00:00Z")},
				Labels: map[string]string{
					v1beta3.BackupPolicyNameLabelKey: p.Name,
				},
			},
			Spec: v1beta3.BackupSpec{
				ServiceInstance: p.Spec.ServiceInstance,
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              framework.UniqueName("backup", suffixLength),
				Namespace:         p.Namespace,
				CreationTimestamp: v1.Time{Time: mustParseTime("2024-07-02T00:00:00Z")},
				Labels: map[string]string{
					v1beta3.BackupPolicyNameLabelKey: p.Name,
				},
			},
			Spec: v1beta3.BackupSpec{
				ServiceInstance: p.Spec.ServiceInstance,
			},
		},
		{
			ObjectMeta: v1.ObjectMeta{
				Name:              framework.UniqueName("backup", suffixLength),
				Namespace:         p.Namespace,
				CreationTimestamp: v1.Time{Time: mustParseTime("2024-07-01T00:00:00Z")},
				Labels: map[string]string{
					v1beta3.BackupPolicyNameLabelKey: p.Name,
				},
			},
			Spec: v1beta3.BackupSpec{
				ServiceInstance: p.Spec.ServiceInstance,
			},
		},
	}
	return existingBackups
}

func CreateBackup(ctx context.Context, k8sClient runtimeClient.Client, namespace, creationTime string) {
	backup := &v1beta3.Backup{
		ObjectMeta: v1.ObjectMeta{
			Name:              framework.UniqueName("backup", suffixLength),
			Namespace:         namespace,
			CreationTimestamp: v1.Time{Time: mustParseTime(creationTime)},
		},
	}
	k8sClient.Create(ctx, backup)
}

func mustParseTime(timeStr string) time.Time {
	parsedTime, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		panic(fmt.Errorf("error parsing time %w", err))
	}
	return parsedTime
}

func CleanupBackups(ctx context.Context, p *v1beta3.BackupPolicy, c runtimeClient.Client) error {
	var err error
	backups := &v1beta3.BackupList{}
	lisOptions := []client.ListOption{
		client.InNamespace(p.Namespace),
		client.MatchingLabels{v1beta3.BackupPolicyNameLabelKey: p.Name},
	}

	err = c.List(ctx, backups, lisOptions...)
	if err != nil {
		return err
	}
	for _, backup := range backups.Items {
		err = c.Delete(ctx, &backup)
		if err != nil {
			return err
		}
	}
	return nil
}
