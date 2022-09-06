package backup

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/anynines/a8s-backup-manager/api/v1alpha1"
	"github.com/anynines/a8s-deployment/test/framework"
)

const (
	backupSucceeded = "Succeeded"
	// asyncOpsTimeoutMins is the amount of minutes after which assertions fail if the condition
	// they check has not become true. Needed because some conditions might become true only
	// after some time, so we need to check them asynchronously.
	// TODO: Make asyncOpsTimeoutMins an invocation parameter.
	asyncOpsTimeoutMins = time.Minute * 5
	suffixLength        = 6
)

// Option represents a functional option for backup objects. To learn what a functional option is,
// read here: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
type Option func(*v1alpha1.Backup)

func SetInstanceRef(dsi runtimeClient.Object) Option {
	return func(b *v1alpha1.Backup) {
		b.Spec.ServiceInstance.APIGroup = dsi.GetObjectKind().GroupVersionKind().Group
		b.Spec.ServiceInstance.Kind = dsi.GetObjectKind().GroupVersionKind().Kind
		b.Spec.ServiceInstance.Name = dsi.GetName()
	}
}

func SetNamespacedName(dsi runtimeClient.Object) Option {
	return func(b *v1alpha1.Backup) {
		b.Name = framework.UniqueName(backupPrefix(dsi.GetName()), suffixLength)
		b.Namespace = dsi.GetNamespace()
	}
}

func New(opts ...Option) *v1alpha1.Backup {
	b := &v1alpha1.Backup{}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

func backupPrefix(dsiName string) string {
	return fmt.Sprintf("%s-backup", dsiName)
}

// TODO: Make the timeout configurable for more advanced backup scenarios
func WaitForReadiness(ctx context.Context, backup *v1alpha1.Backup, c runtimeClient.Client) {
	var err error
	EventuallyWithOffset(1, func() bool {
		backupCreated := New()
		if err = c.Get(
			ctx,
			types.NamespacedName{
				Name:      backup.GetName(),
				Namespace: backup.GetNamespace(),
			},
			backupCreated,
		); err != nil {
			return false
		}

		for _, c := range backupCreated.Status.Conditions {
			if c.Type == "Complete" && c.Status == v1.ConditionTrue {
				return true
			}
		}
		return false
	}, asyncOpsTimeoutMins, 1*time.Second).Should(BeTrue(),
		fmt.Sprintf("timeout reached waiting for backup %s/%s readiness: %s",
			backup.GetNamespace(),
			backup.GetName(),
			err,
		),
	)
}