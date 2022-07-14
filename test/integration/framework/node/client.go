package node

import (
	"context"
	"fmt"
	"log"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

var (
	// Well known taints for master nodes, see
	// https://kubernetes.io/docs/reference/labels-annotations-taints/.
	// Tainting such nodes might result in a broken cluster should one of the
	// control plane components fail during the test runs.
	MasterTaintKeys = map[string]struct{}{
		"node-role.kubernetes.io/master":        {},
		"node-role.kubernetes.io/control-plane": {},
	}
)

type Client struct {
	Nodes            corev1client.NodeInterface
	MasterNodeTaints map[string]struct{}
}

func (c Client) TaintAll(ctx context.Context, t []v1.Taint) error {
	nodes, err := c.Nodes.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all K8s nodes to taint: %w", err)
	}

	for _, n := range nodes.Items {
		// TODO: consider moving on to taint other nodes when an error occurs here (while keeping
		// the error).
		if err := c.taint(ctx, n, t); err != nil {
			return err
		}
	}

	return nil
}

func (c Client) UntaintAll(ctx context.Context, t []v1.Taint) error {
	nodes, err := c.Nodes.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all K8s nodes to untaint: %w", err)
	}

	for _, n := range nodes.Items {
		// TODO: consider moving on to untaint other nodes when an error occurs here (while keeping
		// the error).
		if err := c.untaint(ctx, n, t); err != nil {
			return err
		}
	}

	return nil
}

func (c Client) LabelAll(ctx context.Context, labels map[string]string) error {
	nodes, err := c.Nodes.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all K8s nodes to label: %w", err)
	}

	// Here we don't fail fast. Rather than returning an error on the first failure, we try to
	// label as many nodes as possible, i.e., even if labeling a node fails we try to label
	// the remaining nodes. This is done because users of this library are e2e and integration tests
	// that require labeling to succeed, and will retry it if it fails, so it's actually faster
	// to always try to label as many nodes as possible. This is safe because labeling is
	// idempotent.
	var errs []error
	for _, n := range nodes.Items {
		if err := c.label(ctx, n, labels); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("labeling all or some nodes failed: %v", errors.NewAggregate(errs))
	}

	return nil
}

func (c Client) UnlabelAll(ctx context.Context, labels map[string]string) error {
	nodes, err := c.Nodes.List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list all K8s nodes to unlabel: %w", err)
	}

	// Here we don't fail fast. Rather than returning an error on the first failure, we try to
	// unlabel as many nodes as possible, i.e., even if unlabeling a node fails we try to unlabel
	// the remaining nodes. This is done because users of this library are e2e and integration tests
	// that require unlabeling to succeed, and will retry it if it fails, so it's actually faster
	// to always try to unlabel as many nodes as possible. This is safe because unlabeling is
	// idempotent.
	var errs []error
	for _, n := range nodes.Items {
		if err := c.unlabel(ctx, n, labels); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("unlabeling all or some nodes failed: %v", errors.NewAggregate(errs))
	}

	return nil
}

func (c Client) taint(ctx context.Context, n v1.Node, t []v1.Taint) error {
	if c.hasMasterNodeTaints(n.Spec.Taints) {
		// TODO: stop relying on std logging library, and fix logging all over the code that
		// supports tests (either use what Ginkgo recommends or what we use in controllers).
		log.Printf("Warning: Did not taint node %s as it has a well known master taint", n.Name)
		return nil
	}

	if len(n.Spec.Taints) > 0 {
		log.Printf("Warning: Node %s already is tainted with taints %v. This might break the "+
			"tolerations tests", n.Name, n.Spec.Taints)
	}

	newTaints := taintsUnion(n.Spec.Taints, t)
	if len(newTaints) == len(n.Spec.Taints) {
		return nil
	}

	n.Spec.Taints = newTaints
	if _, err := c.Nodes.Update(ctx, &n, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update of node %s to add taints %v failed: %w", n.Name, t, err)
	}

	return nil
}

func (c Client) untaint(ctx context.Context, n v1.Node, t []v1.Taint) error {
	newTaints := taintsDiff(n.Spec.Taints, t)
	if len(newTaints) == len(n.Spec.Taints) {
		return nil
	}

	n.Spec.Taints = newTaints
	if _, err := c.Nodes.Update(ctx, &n, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update of node %s to remove taints %v failed: %w", n.Name, t, err)
	}

	return nil
}

func (c Client) label(ctx context.Context, n v1.Node, labels map[string]string) error {
	if c.hasMasterNodeTaints(n.Spec.Taints) {
		// TODO: stop relying on std logging library, and fix logging all over the code that
		// supports tests (either use what Ginkgo recommends or what we use in controllers).
		log.Printf("Warning: did not label node %s as it has a well known master taint", n.Name)
		return nil
	}

	newLabels := labelsUnion(n.Labels, labels)
	if len(newLabels) == len(n.Labels) {
		return nil
	}

	n.Labels = newLabels
	if _, err := c.Nodes.Update(ctx, &n, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update of node %s to add labels %v failed: %w", n.Name, labels, err)
	}

	return nil
}

func (c Client) unlabel(ctx context.Context, n v1.Node, labels map[string]string) error {
	newLabels := labelsDiff(n.Labels, labels)
	if len(newLabels) == len(n.Labels) {
		return nil
	}

	n.Labels = newLabels
	if _, err := c.Nodes.Update(ctx, &n, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update of node %s to remove labels %v failed: %w", n.Name, labels, err)
	}

	return nil
}

func taintsUnion(x, y []v1.Taint) []v1.Taint {
	taintKeyToTaint := make(map[string]v1.Taint, len(x)+len(y))

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

	union := make([]v1.Taint, 0, len(taintKeyToTaint))
	for _, t := range taintKeyToTaint {
		union = append(union, t)
	}
	return union
}

func taintsDiff(x, y []v1.Taint) []v1.Taint {
	taintKeyToTaint := make(map[string]v1.Taint, len(x))

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

	diff := make([]v1.Taint, 0, len(taintKeyToTaint))
	for _, t := range taintKeyToTaint {
		diff = append(diff, t)
	}
	return diff
}

func labelsUnion(x, y map[string]string) map[string]string {
	union := make(map[string]string, len(x)+len(y))

	for key, val := range x {
		union[key] = val
	}

	for key, val := range y {
		if oldVal, keyAlreadyPresent := union[key]; keyAlreadyPresent && oldVal != val {
			panic(fmt.Sprintf("can't label node: found a8s test label with key %s and value %s, "+
				"value for that key must be equal to %s", key, oldVal, val))
		}
		union[key] = val
	}

	return union
}

func labelsDiff(x, y map[string]string) map[string]string {
	diff := make(map[string]string)

	for key, xVal := range x {
		yVal, foundKey := y[key]
		if !foundKey {
			diff[key] = xVal
			continue
		}
		if xVal != yVal {
			panic(fmt.Sprintf("can't unlabel node: found a8s test label with key %s and value %s, "+
				"value for that key must be equal to %s", key, xVal, yVal))
		}
	}

	return diff
}

func (c Client) hasMasterNodeTaints(taints []v1.Taint) bool {
	for _, t := range taints {
		if _, found := c.MasterNodeTaints[t.Key]; found {
			return true
		}
	}
	return false
}
