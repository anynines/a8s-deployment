package chaos

import (
	"context"
	"fmt"

	"github.com/anynines/a8s-deployment/test/framework/postgresql"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type PgInjector struct {
	Instance *postgresql.Postgresql
}

type PgChaosHelper interface {
	StopReplicas(ctx context.Context, c runtimeClient.Client) (ChaosObject, error)
	StopMaster(ctx context.Context, c runtimeClient.Client) (ChaosObject, error)
}

func (pg PgInjector) StopReplicas(ctx context.Context, c runtimeClient.Client) (ChaosObject,
	error) {

	podChaos := newPodChaos(
		pg.Instance.GetNamespace(),
		*newPodLabelSelector(pg.Instance.GetReplicaLabels(),
			withSelectorMode("all"),
			withSelectorNamespace([]string{pg.Instance.GetNamespace()}),
		),
		withName(fmt.Sprintf("replica-failure-%s", pg.Instance.GetName())),
		withPodFailureAction(PodFailureAction),
	)

	if err := c.Create(ctx, podChaos.podChaos); err != nil {
		return nil, err
	}

	return podChaos, nil
}

func (pg PgInjector) StopMaster(ctx context.Context, c runtimeClient.Client) (ChaosObject,
	error) {

	podChaos := newPodChaos(
		pg.Instance.GetNamespace(),
		*newPodLabelSelector(
			pg.Instance.GetMasterLabels(),
			withSelectorMode("all"),
			withSelectorNamespace([]string{pg.Instance.GetNamespace()}),
		),
		withName("master-failure"),
		withPodFailureAction(PodFailureAction),
	)

	if err := c.Create(ctx, podChaos.podChaos); err != nil {
		return nil, err
	}

	return podChaos, nil
}
