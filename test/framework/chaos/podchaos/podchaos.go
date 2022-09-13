package podchaos

import (
	"context"
	"fmt"

	chmv1alpha1 "github.com/chaos-mesh/chaos-mesh/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type podChaos = *chmv1alpha1.PodChaos
type PodSelector = *chmv1alpha1.PodSelector

type PodChaos struct {
	podChaos
}

const (
	ChaosCRDName         string = "podchaos.chaos-mesh.org"
	ChaosRequiredVersion string = "v1alpha1"

	PodKillAction       string = "pod-kill"
	PodFailureAction    string = "pod-failure"
	ContainerKillAction string = "container-kill"
)

func New(namespace string, selector chmv1alpha1.PodSelector, opts ...func(podChaos)) PodChaos {
	podChaos := &chmv1alpha1.PodChaos{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-failure",
			Namespace: namespace,
		},
		Spec: chmv1alpha1.PodChaosSpec{
			Action: chmv1alpha1.PodFailureAction,
			ContainerSelector: chmv1alpha1.ContainerSelector{
				PodSelector: selector,
			},
		},
	}

	for _, lambda := range opts {
		lambda(podChaos)
	}

	return PodChaos{podChaos}
}

func WithName(name string) func(podChaos) {
	return func(c podChaos) {
		c.ObjectMeta.Name = name
	}
}

func WithPodFailureAction(action string) func(podChaos) {
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

	return func(c podChaos) {
		c.Spec.Action = a
	}
}

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

func (pc PodChaos) Delete(ctx context.Context, c runtimeClient.Client) error {
	if err := c.Delete(ctx, pc.podChaos); err != nil {
		return fmt.Errorf("failed to delete PodChaos %s: %w", pc.Name, err)
	}
	return nil
}

func (nc PodChaos) GetObject() podChaos {
	return nc.podChaos
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
