package chaosmesh

import (
	"context"
	"fmt"
	"time"

	"github.com/anynines/a8s-deployment/test/e2e/framework"
	"github.com/anynines/a8s-deployment/test/e2e/framework/dsi"
	"github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Adapter struct {
	Client runtimeClient.Client
}

func (a Adapter) IsolatePrimary(ctx context.Context, o dsi.Object) error {
	p, err := framework.GetPrimaryPodUsingServiceSelector(ctx, o, a.Client)
	if err != nil {
		return fmt.Errorf("unable to get primary %w", err)
	}
	name := framework.GenerateName("chaos", 0, 5)
	fault := v1alpha1.NetworkChaos{
		ObjectMeta: v1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.NetworkChaosSpec{
			Target: &v1alpha1.PodSelector{
				Mode: v1alpha1.AllMode,
				Selector: v1alpha1.PodSelectorSpec{
					Pods: map[string][]string{
						p.Namespace: {p.Name},
					},
				},
			},
		},
	}

	if err := a.Client.Create(ctx, &fault); err != nil {
		return fmt.Errorf("failed to create chaos object: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			err := a.Client.Get(ctx, runtimeClient.ObjectKeyFromObject(&fault), &fault)
			if err != nil {
				time.Sleep(1 * time.Second)
				continue
			}
			for _, c := range fault.Status.Conditions {
				if c.Type == v1alpha1.ConditionAllInjected {
					return nil
				}
			}
		}
	}
}
func (Adapter) Undo(dsi.Object) {

}
