package chaos

import (
	"context"
	"fmt"

	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/anynines/a8s-deployment/test/framework/chaos/networkchaos"
	"github.com/anynines/a8s-deployment/test/framework/chaos/podchaos"
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

	podChaos := podchaos.New(
		pg.Instance.GetNamespace(),
		*podchaos.NewPodLabelSelector(pg.Instance.GetReplicaLabels(),
			podchaos.WithSelectorMode("all"),
			podchaos.WithSelectorNamespace([]string{pg.Instance.GetNamespace()}),
		),
		podchaos.WithName(fmt.Sprintf("replica-failure-%s", pg.Instance.GetName())),
		podchaos.WithPodFailureAction(podchaos.PodFailureAction),
	)

	if err := c.Create(ctx, podChaos); err != nil {
		return nil, err
	}

	return podChaos, nil
}

func (pg PgInjector) StopMaster(ctx context.Context, c runtimeClient.Client) (ChaosObject,
	error) {

	podChaos := podchaos.New(
		pg.Instance.GetNamespace(),
		*podchaos.NewPodLabelSelector(
			pg.Instance.GetMasterLabels(),
			podchaos.WithSelectorMode("all"),
			podchaos.WithSelectorNamespace([]string{pg.Instance.GetNamespace()}),
		),
		podchaos.WithName("master-failure"),
		podchaos.WithPodFailureAction(podchaos.PodFailureAction),
	)

	if err := c.Create(ctx, podChaos.GetObject()); err != nil {
		return nil, err
	}

	return podChaos, nil
}

func (pg PgInjector) PartitionMaster(ctx context.Context, c runtimeClient.Client, t []string) (
	ChaosObject, error) {

	nc := networkchaos.New(pg.Instance.GetNamespace(),
		*networkchaos.NewPodLabelSelector(pg.Instance.GetMasterLabels(),
			networkchaos.WithSelectorMode("all"),
			networkchaos.WithSelectorNamespace([]string{pg.Instance.GetNamespace()})),
		networkchaos.WithName("partition-master"),
		networkchaos.WithAction("partition"),
		networkchaos.WithExternalTargets(t),
		networkchaos.WithMode("all"),
	)

	if err := c.Create(ctx, nc.GetObject()); err != nil {
		return nil, err
	}

	return nc, nil
}
