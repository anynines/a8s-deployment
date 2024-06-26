package podchaos

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

type (
	PodChaos    chmv1alpha1.PodChaos
	PodSelector = chmv1alpha1.PodSelector
)

const (
	CRDName         string = "podchaos.chaos-mesh.org"
	RequiredVersion string = "v1alpha1"

	PodKillAction       string = "pod-kill"
	PodFailureAction    string = "pod-failure"
	ContainerKillAction string = "container-kill"
)

// New returns a PodChaos object configured with a selector and provided options.
func New(namespace string, selector *PodSelector, opts ...func(*PodChaos)) PodChaos {
	podChaos := chmv1alpha1.PodChaos{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-failure",
			Namespace: namespace,
		},
		Spec: chmv1alpha1.PodChaosSpec{
			Action: chmv1alpha1.PodFailureAction,
			ContainerSelector: chmv1alpha1.ContainerSelector{
				PodSelector: *selector,
			},
		},
	}

	podChaosObj := PodChaos(podChaos)
	for _, lambda := range opts {
		lambda(&podChaosObj)
	}

	return podChaosObj
}

// WithName overrides the Name field for a PodChaos object.
func WithName(name string) func(*PodChaos) {
	return func(c *PodChaos) {
		c.ObjectMeta.Name = name
	}
}

// WithAction overrides the PodChaos Action field.
func WithAction(action string) func(*PodChaos) {
	var a chmv1alpha1.PodChaosAction
	switch action {
	case PodKillAction:
		a = chmv1alpha1.PodKillAction
	case PodFailureAction:
		a = chmv1alpha1.PodFailureAction
	case ContainerKillAction:
		a = chmv1alpha1.ContainerKillAction
	default:
		panic("Invalid PodChaosAction : " + action)
	}

	return func(c *PodChaos) {
		c.Spec.Action = a
	}
}

// CheckChaosActive checks if a PodChaos object indicates a successful injection of Chaos action.
func (pc PodChaos) CheckChaosActive(ctx context.Context, c runtimeClient.Client) (bool, error) {
	podChaos := &chmv1alpha1.PodChaos{}
	err := c.Get(ctx, types.NamespacedName{Name: pc.Name, Namespace: pc.Namespace}, podChaos)
	if err != nil {
		return false, fmt.Errorf("failed getting PodChaos %s: %w", podChaos.Name, err)
	}

	for _, cond := range podChaos.Status.Conditions {
		if cond.Type == chmv1alpha1.ConditionAllInjected {
			if cond.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
	}
	return false, nil
}

// GetObject returns the actual PodChaos object
func (pc PodChaos) KubernetesObject() client.Object {
	chaosObj := chmv1alpha1.PodChaos(pc)
	return &chaosObj
}

// NewPodLabelSelector returns a new PodSelector configured using labels and provided options.
func NewPodLabelSelector(labels map[string]string,
	opts ...func(*PodSelector),
) *PodSelector {
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
