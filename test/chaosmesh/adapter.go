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

func (a FaultInjector) IsolatePrimary(ctx context.Context, o dsi.Object) error {
	p, err := framework.GetPrimaryPodUsingServiceSelector(ctx, o, a.Client)
	if err != nil {
		return fmt.Errorf("unable to get primary %w", err)
	}
	name := framework.GenerateName("chaos", 0, 5)
	nsn := v1.ObjectMeta{
		GenerateName: name,
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
			// Target: &v1alpha1.PodSelector{
			// 	Mode: v1alpha1.AllMode,
			// 	Selector: v1alpha1.PodSelectorSpec{
			// 		GenericSelectorSpec: v1alpha1.GenericSelectorSpec{
			// 			Namespaces: []string{a.Namespace},
			// 			LabelSelectors: map[string]string{
			// 				"a8s.a9s/dsi-name":         o.GetName(),
			// 				"a8s.a9s/replication-role": "replica",
			// 			},
			// 		},
			// 	},
			// },
			ExternalTargets: []string{
				"10.0.0.0/8", // services on most kubernetes clusters
				"100.64.0.1", // proxy on gardener
			},
		},
	}

	if err := a.Client.Create(ctx, &fault); err != nil {
		return fmt.Errorf("failed to create chaos object: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return errors.New("timeout")
		default:
			err := a.Client.Get(ctx, runtimeClient.ObjectKeyFromObject(&fault), &fault)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			for _, c := range fault.Status.Conditions {
				if c.Type == v1alpha1.ConditionAllInjected && c.Status == corev1.ConditionTrue {
					return nil
				}
			}
		}
	}
}
