package chaos

import (
	"context"
	"fmt"

	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/anynines/a8s-deployment/test/framework/chaos/network"
	"github.com/anynines/a8s-deployment/test/framework/chaos/pod"
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

	podChaos := pod.NewChaos(
		pg.Instance.GetNamespace(),
		*pod.NewPodLabelSelector(pg.Instance.GetReplicaLabels(),
			pod.WithSelectorMode("all"),
			pod.WithSelectorNamespace([]string{pg.Instance.GetNamespace()}),
		),
		pod.WithName(fmt.Sprintf("replica-failure-%s", pg.Instance.GetName())),
		pod.WithPodFailureAction(pod.PodFailureAction),
	)

	if err := c.Create(ctx, podChaos); err != nil {
		return nil, err
	}

	return podChaos, nil
}

func (pg PgInjector) StopMaster(ctx context.Context, c runtimeClient.Client) (ChaosObject,
	error) {

	podChaos := pod.NewChaos(
		pg.Instance.GetNamespace(),
		*pod.NewPodLabelSelector(
			pg.Instance.GetMasterLabels(),
			pod.WithSelectorMode("all"),
			pod.WithSelectorNamespace([]string{pg.Instance.GetNamespace()}),
		),
		pod.WithName("master-failure"),
		pod.WithPodFailureAction(pod.PodFailureAction),
	)

	if err := c.Create(ctx, podChaos.GetObject()); err != nil {
		return nil, err
	}

	return podChaos, nil
}

func (pg PgInjector) PartitionMaster(ctx context.Context, c runtimeClient.Client, t []string) (
	ChaosObject, error) {

	nc := network.NewChaos(pg.Instance.GetNamespace(),
		*network.NewPodLabelSelector(pg.Instance.GetMasterLabels(), network.WithSelectorMode("all"),
			network.WithSelectorNamespace([]string{pg.Instance.GetNamespace()})),
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
