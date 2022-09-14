package networkchaos

import (
	"context"
	"fmt"

	chmv1alpha1 "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type networkChaos = *chmv1alpha1.NetworkChaos
type PodSelector = *chmv1alpha1.PodSelector

type NetworkChaos struct {
	networkChaos
}

// NetworkChaos Actions
const (
	// partitionAction represents the chaos action of network partition of pods.
	partitionAction string = "partition"
)

// NetworkChaosModes
const (
	// allMode represents that the system will do the chaos action on all objects
	// regardless of status (not ready or not running pods includes).
	// Use this label carefully.
	allMode string = "all"
)

func New(namespace string, selector chmv1alpha1.PodSelector, opts ...func(networkChaos)) NetworkChaos {
	networkChaos := &chmv1alpha1.NetworkChaos{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "network-partition",
			Namespace: namespace,
		},
		Spec: chmv1alpha1.NetworkChaosSpec{
			Action: chmv1alpha1.PartitionAction,
			PodSelector: chmv1alpha1.PodSelector{
				Selector: chmv1alpha1.PodSelectorSpec{
					GenericSelectorSpec: chmv1alpha1.GenericSelectorSpec{
						Namespaces:     []string{namespace},
						LabelSelectors: selector.Selector.LabelSelectors,
					},
				},
			},
			Direction: chmv1alpha1.To,
		},
	}

	for _, lambda := range opts {
		lambda(networkChaos)
	}

	return NetworkChaos{networkChaos}
}

func WithName(name string) func(networkChaos) {
	return func(c networkChaos) {
		c.ObjectMeta.Name = name
	}
}

func WithAction(action string) func(networkChaos) {
	var a chmv1alpha1.NetworkChaosAction
	switch action {
	case partitionAction:
		a = chmv1alpha1.PartitionAction
	default:
		panic("Invalid NetworkChaosAction : " + action)
	}

	return func(c networkChaos) {
		c.Spec.Action = a
	}
}

func WithMode(mode string) func(networkChaos) {
	var m chmv1alpha1.SelectorMode
	switch mode {
	case allMode:
		m = chmv1alpha1.AllMode
	default:
		panic("Invalid NetworkChaos mode : " + mode)
	}

	return func(nc networkChaos) {
		nc.Spec.Mode = m
	}
}

func (nc NetworkChaos) CheckChaosActive(ctx context.Context, c runtimeClient.Client) (bool, error) {
	networkChaos := &chmv1alpha1.NetworkChaos{}
	err := c.Get(ctx, types.NamespacedName{Name: nc.Name, Namespace: nc.Namespace}, networkChaos)
	if err != nil {
		return false, fmt.Errorf("failed getting NetworkChaos %s: %w", networkChaos.Name, err)
	}

	for _, cond := range networkChaos.Status.Conditions {
		if cond.Type == chmv1alpha1.ConditionAllInjected {
			if cond.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
	}
	return false, nil
}

func (nc NetworkChaos) Delete(ctx context.Context, c runtimeClient.Client) error {
	if err := c.Delete(ctx, nc.networkChaos); err != nil {
		return fmt.Errorf("failed to delete NetworkChaos %s: %w", nc.Name, err)
	}
	return nil
}

func (nc NetworkChaos) GetObject() networkChaos {
	return nc.networkChaos
}

func NewPodLabelSelector(labels map[string]string,
	opts ...func(PodSelector)) PodSelector {

	podSelector := &chmv1alpha1.PodSelector{
		Selector: chmv1alpha1.PodSelectorSpec{
			GenericSelectorSpec: chmv1alpha1.GenericSelectorSpec{
				LabelSelectors: labels,
			},
		},
		Mode: chmv1alpha1.AllMode,
	}

	for _, lambda := range opts {
		lambda(podSelector)
	}

	return podSelector
}

func WithSelectorMode(mode string) func(PodSelector) {
	var m chmv1alpha1.SelectorMode
	switch mode {
	case "one":
		m = chmv1alpha1.OneMode
	case "all":
		m = chmv1alpha1.AllMode
	case "fixed":
		m = chmv1alpha1.FixedMode
	case "fixed-percent":
		m = chmv1alpha1.FixedPercentMode
	case "random-max-percent":
		m = chmv1alpha1.RandomMaxPercentMode
	}

	return func(s PodSelector) {
		s.Mode = m
	}
}

func WithSelectorNamespace(namespaces []string) func(PodSelector) {
	return func(s PodSelector) {
		s.Selector.Namespaces = namespaces
	}
}

func WithExternalTargets(targets []string) func(networkChaos) {
	return func(nc networkChaos) {
		nc.Spec.ExternalTargets = targets
	}
}
