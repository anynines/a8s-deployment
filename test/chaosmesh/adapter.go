package chaosmesh

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/anynines/a8s-deployment/test/e2e/framework"
	"github.com/anynines/a8s-deployment/test/e2e/framework/dsi"
	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type FaultInjector struct {
	Client    runtimeClient.Client
	Namespace string
}

// Disallow all outgoing traffic from the primary to simulate a network partition.
func (a FaultInjector) IsolatePrimary(ctx context.Context, o dsi.Object,
) (func(context.Context) error, error) {

	p, err := framework.GetPrimaryPodUsingServiceSelector(ctx, o, a.Client)
	if err != nil {
		return nil, fmt.Errorf("unable to get primary %w", err)
	}

	nsn := v1.ObjectMeta{
		GenerateName: "chaos",
		Namespace:    a.Namespace,
		Labels: map[string]string{
			"instance": o.GetName(),
		},
	}

	targetSelector := v1alpha1.PodSelector{
		Mode: v1alpha1.AllMode,
		Selector: v1alpha1.PodSelectorSpec{
			Pods: map[string][]string{
				p.Namespace: {p.Name},
			},
		},
	}

	fault := v1alpha1.NetworkChaos{
		ObjectMeta: nsn,
		Spec: v1alpha1.NetworkChaosSpec{
			Direction: v1alpha1.To,
			Action:    v1alpha1.LossAction,
			TcParameter: v1alpha1.TcParameter{
				Loss: &v1alpha1.LossSpec{Loss: "100"},
			},
			PodSelector: targetSelector,
		},
	}

	if err := a.Client.Create(ctx, &fault); err != nil {
		return nil, fmt.Errorf("failed to create chaos object: %w", err)
	}

	undo := func(ctx context.Context) error {
		return a.Client.Delete(ctx, &fault)
	}

	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("timeout")
		default:
			var fetchedFault v1alpha1.NetworkChaos
			err := a.Client.Get(ctx, runtimeClient.ObjectKeyFromObject(&fault), &fetchedFault)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			for _, c := range fetchedFault.Status.Conditions {
				if c.Type == v1alpha1.ConditionAllInjected && c.Status == corev1.ConditionTrue {
					return undo, nil
				}
			}
		}
	}
}
