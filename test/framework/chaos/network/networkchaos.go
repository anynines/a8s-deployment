package network

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

const (
	// Actions
	NetemAction     string = "netem"
	DelayAction     string = "delay"
	LossAction      string = "loss"
	DuplicateAction string = "duplicate"
	CorruptAction   string = "corrupt"
	PartitionAction string = "partition"
	BandwidthAction string = "bandwidth"

	// Modes
	OneMode              string = "one"
	AllMode              string = "all"
	FixedMode            string = "fixed"
	FixedPercentMode     string = "fixed-percent"
	RandomMaxPercentMode string = "random-max-percent"
)

func NewChaos(namespace string, selector chmv1alpha1.PodSelector, opts ...func(networkChaos)) NetworkChaos {
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
	case NetemAction:
		a = chmv1alpha1.NetemAction
	case DelayAction:
		a = chmv1alpha1.DelayAction
	case LossAction:
		a = chmv1alpha1.LossAction
	case DuplicateAction:
		a = chmv1alpha1.DuplicateAction
	case CorruptAction:
		a = chmv1alpha1.CorruptAction
	case PartitionAction:
		a = chmv1alpha1.PartitionAction
	case BandwidthAction:
		a = chmv1alpha1.BandwidthAction
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
	case OneMode:
		m = chmv1alpha1.OneMode
	case AllMode:
		m = chmv1alpha1.AllMode
	case FixedMode:
		m = chmv1alpha1.FixedMode
	case FixedPercentMode:
		m = chmv1alpha1.FixedPercentMode
	case RandomMaxPercentMode:
		m = chmv1alpha1.RandomMaxPercentMode
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
