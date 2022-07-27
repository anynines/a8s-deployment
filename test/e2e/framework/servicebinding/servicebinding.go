package servicebinding

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/anynines/a8s-deployment/test/e2e/framework"
	"github.com/anynines/a8s-service-binding-controller/api/v1alpha1"
)

const (
	asyncOpsTimeoutMins = time.Minute * 5
	suffixLength        = 6
)

// Option represents a functional option for service binding objects. To learn what a functional
// option is, read here: https://dave.cheney.net/2014/10/17/functional-options-for-friendly-apis
type Option func(*v1alpha1.ServiceBinding)

func SetInstanceRef(dsi runtimeclient.Object) Option {
	return func(sb *v1alpha1.ServiceBinding) {
		sb.Spec.Instance.APIVersion = dsi.
			GetObjectKind().GroupVersionKind().GroupVersion().String()
		sb.Spec.Instance.Kind = dsi.GetObjectKind().GroupVersionKind().Kind
		sb.Spec.Instance.NamespacedName = v1alpha1.NamespacedName{
			Name:      dsi.GetName(),
			Namespace: dsi.GetNamespace(),
		}
	}
}

// TODO: Make two separate options for name and namespace. We only need to pass string as
// parameters
func SetNamespacedName(dsi runtimeclient.Object) Option {
	return func(sb *v1alpha1.ServiceBinding) {
		sb.Name = framework.UniqueName(sbPrefix(dsi.GetName()), suffixLength)
		sb.Namespace = dsi.GetNamespace()
	}
}

func sbPrefix(dsiName string) string {
	return fmt.Sprintf("%s-sb", dsiName)
}

func New(opts ...Option) *v1alpha1.ServiceBinding {
	sb := &v1alpha1.ServiceBinding{}
	for _, opt := range opts {
		opt(sb)
	}
	return sb
}

func SecretName(sbName string) string {
	return fmt.Sprintf("%s-%s", sbName, "service-binding")
}

func WaitForReadiness(ctx context.Context, sb *v1alpha1.ServiceBinding, c runtimeclient.Client) {
	var err error
	EventuallyWithOffset(1, func() bool {
		sbCreated := New()
		if err = c.Get(
			ctx,
			types.NamespacedName{Name: sb.GetName(), Namespace: sb.GetNamespace()},
			sbCreated,
		); err != nil {
			return false
		}
		return sbCreated.Status.Implemented
	}, asyncOpsTimeoutMins, 1*time.Second).Should(BeTrue(),
		fmt.Sprintf("timeout reached waiting for servicebinding %s/%s readiness: %s",
			sb.GetNamespace(),
			sb.GetName(),
			err,
		),
	)
}
