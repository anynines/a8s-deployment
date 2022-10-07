package restore

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
	recoverySucceeded = "Succeeded"
	// asyncOpsTimeoutMins is the amount of minutes after which assertions fail if the condition
	// they check has not become true. Needed because some conditions might become true only after
	// some time, so we need to check them asynchronously.
	// TODO: Make asyncOpsTimeoutMins an invocation parameter.
	asyncOpsTimeoutMins = time.Minute * 5
	suffixLength        = 6
)

// Option represents a functional option for restore objects. To learn what a functional option is,
// read here: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
type Option func(*v1alpha1.Restore)

func SetInstanceRef(dsi runtimeClient.Object) Option {
	return func(rcv *v1alpha1.Restore) {
		rcv.Spec.ServiceInstance.APIGroup = dsi.GetObjectKind().GroupVersionKind().Group
		rcv.Spec.ServiceInstance.Kind = dsi.GetObjectKind().GroupVersionKind().Kind
		rcv.Spec.ServiceInstance.Name = dsi.GetName()
	}
}

// TODO: Make two separate options for name and namespace. We only need to pass string as
// parameters
func SetNamespacedName(dsi runtimeClient.Object) Option {
	return func(rcv *v1alpha1.Restore) {
		rcv.Name = framework.UniqueName(recoveryPrefix(dsi.GetName()), suffixLength)
		rcv.Namespace = dsi.GetNamespace()
	}
}

func SetBackupName(backupName string) Option {
	return func(rcv *v1alpha1.Restore) {
		rcv.Spec.BackupName = backupName
	}
}

func New(opts ...Option) *v1alpha1.Restore {
	rcv := &v1alpha1.Restore{}
	for _, opt := range opts {
		opt(rcv)
	}
	return rcv
}

func recoveryPrefix(dsiName string) string {
	return fmt.Sprintf("%s-recovery", dsiName)
}

func WaitForReadiness(ctx context.Context, recovery *v1alpha1.Restore, c runtimeClient.Client) {
	var err error
	EventuallyWithOffset(1, func() bool {
		recoveryCreated := New()
		if err = c.Get(
			ctx,
			types.NamespacedName{
				Name:      recovery.GetName(),
				Namespace: recovery.GetNamespace(),
			},
			recoveryCreated,
		); err != nil {
			return false
		}

		for _, c := range recoveryCreated.Status.Conditions {
			if c.Type == "Complete" && c.Status == v1.ConditionTrue {
				return true
			}
		}

		return false
	}, asyncOpsTimeoutMins, 1*time.Second).Should(BeTrue(),
		fmt.Sprintf("timeout reached waiting for recovery %s/%s readiness at %s: %s",
			recovery.GetNamespace(),
			recovery.GetName(),
			recovery.GetCreationTimestamp().String(),
			err,
		),
	)
}
