package networkchaos

import (
	"context"
	"fmt"

	chmv1alpha1 "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type NetworkChaos chmv1alpha1.NetworkChaos
type PodSelector = chmv1alpha1.PodSelector

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

// New returns a NetworkChaos object configured with a selector and provided options.
func New(namespace string, selector *PodSelector, opts ...func(*NetworkChaos)) NetworkChaos {
	networkChaos := chmv1alpha1.NetworkChaos{
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

	networkChaosObj := NetworkChaos(networkChaos)
	for _, lambda := range opts {
		lambda(&networkChaosObj)
	}

	return networkChaosObj
}

// WithName overrides the Name field for a NetworkChaos object.
func WithName(name string) func(*NetworkChaos) {
	return func(c *NetworkChaos) {
		c.ObjectMeta.Name = name
	}
}

// WithAction overrides the NetworkChaos Action field.
func WithAction(action string) func(*NetworkChaos) {
	var a chmv1alpha1.NetworkChaosAction
	switch action {
	case partitionAction:
		a = chmv1alpha1.PartitionAction
	default:
		panic("Invalid NetworkChaosAction : " + action)
	}

	return func(c *NetworkChaos) {
		c.Spec.Action = a
	}
}

// WithAction overrides the NetworkChaos Mode field.
func WithMode(mode string) func(*NetworkChaos) {
	var m chmv1alpha1.SelectorMode
	switch mode {
	case allMode:
		m = chmv1alpha1.AllMode
	default:
		panic("Invalid NetworkChaos mode : " + mode)
	}

	return func(nc *NetworkChaos) {
		nc.Spec.Mode = m
	}
}

// CheckChaosActive checks if a NetworkChaos object indicates a successful injection of Chaos action.
func (nc NetworkChaos) CheckChaosActive(ctx context.Context, c runtimeClient.Client) (bool, error) {
	networkChaos := &chmv1alpha1.NetworkChaos{}
	err := c.Get(ctx, types.NamespacedName{Name: nc.Name, Namespace: nc.Namespace}, networkChaos)
	if err != nil {
		return false, fmt.Errorf("failed getting NetworkChaos %s: %w", networkChaos.Name, err)
	}

	for _, cond := range networkChaos.Status.Conditions {
		if cond.Type == chmv1alpha1.ConditionAllInjected && cond.Status == corev1.ConditionTrue {
			return true, nil
		}
	}
	return false, nil
}

// GetObject returns the actual NetworkChaos object
func (nc NetworkChaos) GetObject() client.Object {
	networkChaosObj := chmv1alpha1.NetworkChaos(nc)
	return &networkChaosObj
}

// NewPodLabelSelector returns a new PodSelector configured using labels and provided options.
func NewPodLabelSelector(labels map[string]string,
	opts ...func(*PodSelector)) *PodSelector {

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

// WithSelectorMode overrides the SelectorMode for a PodSelector.
func WithSelectorMode(mode string) func(*PodSelector) {
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

	return func(s *PodSelector) {
		s.Mode = m
	}
}

// WithSelectorNamespace overrides the namespaces for a PodSelector.
func WithSelectorNamespace(namespaces []string) func(*PodSelector) {
	return func(s *PodSelector) {
		s.Selector.Namespaces = namespaces
	}
}

// WithExternalTargets overrides the ExternalTargets for a NetworkChaos object.
func WithExternalTargets(targets []string) func(*NetworkChaos) {
	return func(nc *NetworkChaos) {
		nc.Spec.ExternalTargets = targets
	}
}
