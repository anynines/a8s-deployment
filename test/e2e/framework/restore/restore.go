package restore

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/anynines/a8s-backup-manager/api/v1alpha1"
	"github.com/anynines/a8s-deployment/test/e2e/framework"
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
type Option func(*v1alpha1.Recovery)

func SetInstanceRef(dsi runtimeClient.Object) Option {
	return func(rcv *v1alpha1.Recovery) {
		rcv.Spec.ServiceInstance.APIGroup = dsi.GetObjectKind().GroupVersionKind().Group
		rcv.Spec.ServiceInstance.Kind = dsi.GetObjectKind().GroupVersionKind().Kind
		rcv.Spec.ServiceInstance.Name = dsi.GetName()
	}
}

// TODO: Make two separate options for name and namespace. We only need to pass string as
// parameters
func SetNamespacedName(dsi runtimeClient.Object) Option {
	return func(rcv *v1alpha1.Recovery) {
		rcv.Name = framework.UniqueName(recoveryPrefix(dsi.GetName()), suffixLength)
		rcv.Namespace = dsi.GetNamespace()
	}
}

func SetBackupName(backupName string) Option {
	return func(rcv *v1alpha1.Recovery) {
		rcv.Spec.BackupName = backupName
	}
}

func New(opts ...Option) *v1alpha1.Recovery {
	rcv := &v1alpha1.Recovery{}
	for _, opt := range opts {
		opt(rcv)
	}
	return rcv
}

func recoveryPrefix(dsiName string) string {
	return fmt.Sprintf("%s-recovery", dsiName)
}

func WaitForReadiness(ctx context.Context, recovery *v1alpha1.Recovery, c runtimeClient.Client) {
	var err error
	EventuallyWithOffset(1, func() string {
		recoveryCreated := New()
		if err = c.Get(
			ctx,
			types.NamespacedName{
				Name:      recovery.GetName(),
				Namespace: recovery.GetNamespace(),
			},
			recoveryCreated,
		); err != nil {
			return fmt.Sprintf("%v+", err)
		}
		return string(recoveryCreated.Status.Condition.Type)
	}, asyncOpsTimeoutMins, 1*time.Second).Should(Equal(recoverySucceeded),
		fmt.Sprintf("timeout reached waiting for recovery %s/%s readiness at %s: %s",
			recovery.GetNamespace(),
			recovery.GetName(),
			recovery.GetCreationTimestamp().String(),
			err,
		),
	)
}
