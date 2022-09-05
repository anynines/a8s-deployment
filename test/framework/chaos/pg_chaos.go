package chaos

import (
	"context"
	"fmt"

	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/anynines/a8s-deployment/test/framework/chaos/network"
	"github.com/anynines/a8s-deployment/test/framework/postgresql"
)

type PgInjector struct {
	Instance *postgresql.Postgresql
}

type PgChaosHelper interface {
	StopReplicas(ctx context.Context, c runtimeClient.Client) (ChaosObject, error)
	StopMaster(ctx context.Context, c runtimeClient.Client) (ChaosObject, error)
	PartitionMaster(ctx, c runtimeClient.Client, t []string) (ChaosObject, error)
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

func (pg PgInjector) PartitionMaster(ctx context.Context, c runtimeClient.Client, t []string) (
	ChaosObject, error) {

	nc := network.NewChaos(pg.Instance.GetNamespace(),
		*newPodLabelSelector(pg.Instance.GetMasterLabels(), withSelectorMode("all"),
			withSelectorNamespace([]string{pg.Instance.GetNamespace()})),
		network.WithName("partition-master"),
		network.WithAction("partition"),
		network.WithExternalTargets(t),
		network.WithMode("all"),
	)

	if err := c.Create(ctx, nc.GetObject()); err != nil {
		return nil, err
	}

	return nc, nil
}
