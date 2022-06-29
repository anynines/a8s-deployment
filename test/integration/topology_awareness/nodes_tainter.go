package topology_awareness

import (
	"context"
	"fmt"
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

// This file contains some utils to fiddle with K8s nodes taints. We initially tried using the
// upstream utils in https://pkg.go.dev/k8s.io/kubernetes/pkg/util/taints . We gave up because
// of this: https://github.com/kubernetes/kubernetes/issues/79384 .

// TODO: Replace  stdlib log package. Either use what Ginkgo recommends or what we use in other
// components (e.g. the controllers).

var (
	// Well known taints for master nodes, see
	// https://kubernetes.io/docs/reference/labels-annotations-taints/.
	// Tainting such nodes might result in a broken cluster should one of the
	// control plane components fail during the test runs.
	masterNodeTaintKeys = map[string]struct{}{
		"node-role.kubernetes.io/master":        {},
		"node-role.kubernetes.io/control-plane": {},
	}
)

// TODO: Consider whether it's worth it to factor this out to a dedicated package
type nodesTainter struct {
	nodes corev1client.NodeInterface
}

func newNodesTainter(nodes corev1client.NodeInterface) nodesTainter {
	return nodesTainter{nodes: nodes}
}

func (nt nodesTainter) taintAllNodes(ctx context.Context, t []corev1.Taint) error {
	nodes, err := nt.nodes.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all K8s nodes to taint: %w", err)
	}

	for _, n := range nodes.Items {
		if hasMasterNodeTaints(n.Spec.Taints) {
			// TODO: stop relying on std logging library, and fix logging all over the code that
			// supports tests (either use what Ginkgo recommends or what we use in controllers).
			log.Printf("Warning: Did not taint node %s as it has a well known master taint", n.Name)
			continue
		}
		if len(n.Spec.Taints) > 0 {
			log.Printf("Warning: Node %s already is tainted with taints %v. This might break the "+
				"tolerations tests", n.Name, n.Spec.Taints)
		}
		if newTaints := nt.union(n.Spec.Taints, t); len(newTaints) != len(n.Spec.Taints) {
			n.Spec.Taints = newTaints
			if _, err := nt.nodes.Update(ctx, &n, metav1.UpdateOptions{}); err != nil {
				return fmt.Errorf("update of node %s to add taints %v failed: %w", n.Name, t, err)
			}
		}
	}

	return nil
}

func (nt nodesTainter) untaintAllNodes(ctx context.Context, t []corev1.Taint) error {
	nodes, err := nt.nodes.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all K8s nodes to untaint: %w", err)
	}

	for _, n := range nodes.Items {
		if newTaints := nt.diff(n.Spec.Taints, t); len(newTaints) != len(n.Spec.Taints) {
			n.Spec.Taints = newTaints
			if _, err := nt.nodes.Update(ctx, &n, metav1.UpdateOptions{}); err != nil {
				return fmt.Errorf("update of node %s to remove taints %v failed: %w", n.Name, t, err)
			}
		}
	}

	return nil
}

func (nodesTainter) union(x, y []corev1.Taint) []corev1.Taint {
	taintKeyToTaint := make(map[string]corev1.Taint, len(x)+len(y))

	for _, t := range x {
		taintKeyToTaint[t.Key] = t
	}

	for _, t1 := range y {
		t2, keyAlreadyPresent := taintKeyToTaint[t1.Key]
		if keyAlreadyPresent && (t1.Value != t2.Value || t1.Effect != t2.Effect) {
			// TODO: return an error rather than panicing here - if we need to panic let the caller
			// do that.
			panic(fmt.Sprintf("can't taint node: found taint %s with a8s test key %s but "+
				"(value, effect)=(%s, %s); (value, effect) must be equal to (%s, %s)", t2, t2.Key,
				t2.Value, t2.Effect, t1.Value, t1.Effect))
		}
		taintKeyToTaint[t1.Key] = t1
	}

	union := make([]corev1.Taint, 0, len(taintKeyToTaint))
	for _, t := range taintKeyToTaint {
		union = append(union, t)
	}
	return union
}

func (nodesTainter) diff(x, y []corev1.Taint) []corev1.Taint {
	taintKeyToTaint := make(map[string]corev1.Taint, len(x))

	for _, t := range x {
		taintKeyToTaint[t.Key] = t
	}

	for _, t1 := range y {
		t2, foundKey := taintKeyToTaint[t1.Key]
		if foundKey && (t1.Value != t2.Value || t1.Effect != t2.Effect) {
			// TODO: return an error rather than panicing here - if we need to panic let the caller
			// do that.
			panic(fmt.Sprintf("can't untaint node: found taint %s with a8s test key %s but "+
				"(value, effect)=(%s, %s); (value, effect) must be equal to (%s, %s)", t2, t2.Key,
				t2.Value, t2.Effect, t1.Value, t1.Effect))
		}
		delete(taintKeyToTaint, t1.Key)
	}

	diff := make([]corev1.Taint, 0, len(taintKeyToTaint))
	for _, t := range taintKeyToTaint {
		diff = append(diff, t)
	}
	return diff
}

func hasMasterNodeTaints(taints []corev1.Taint) bool {
	for _, t := range taints {
		if _, found := masterNodeTaintKeys[t.Key]; found {
			return true
		}
	}
	return false
}
