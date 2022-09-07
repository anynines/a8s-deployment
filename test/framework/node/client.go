package node

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/anynines/a8s-deployment/test/framework/log"
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
	Log              logr.Logger
}

func NewClientFromKubecfg(kubecfg string) (Client, error) {
	cfg, err := clientcmd.BuildConfigFromFlags("", kubecfg)
	if err != nil {
		return Client{}, fmt.Errorf("failed to create client config for K8s nodes client from "+
			"kubeconig %s: %w", kubecfg, err)
	}

	cv1Client, err := corev1client.NewForConfig(cfg)
	if err != nil {
		return Client{},
			fmt.Errorf("failed to create client for K8s cluster nodes from config %v: %w", cfg, err)
	}

	return Client{
		Nodes:            cv1Client.Nodes(),
		MasterNodeTaints: MasterTaintKeys,
		Log:              log.NewWithNames("Node", "Client"),
	}, nil
}

func (c Client) GetLabels(ctx context.Context, nodeName string) (map[string]string, error) {
	node, err := c.Nodes.Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get labels for node %s: failed to get node: %w",
			nodeName, err)
	}

	if node == nil {
		return nil, nil
	}

	return node.Labels, nil
}

// Get returns the node named `nodeName`.
// In case of error, it returns the error and an empty v1.Node struct.
func (c Client) Get(ctx context.Context, nodeName string) (v1.Node, error) {
	n, err := c.Nodes.Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return v1.Node{}, fmt.Errorf("failed to get node %s: %w", nodeName, err)
	}
	return *n, nil
}

// ListAll returns a slice with all the nodes in the K8s cluster.
// ListAll returns an empty slice and an error if a failure occurs.
func (c Client) ListAll(ctx context.Context) ([]v1.Node, error) {
	nodes, err := c.Nodes.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list K8s cluster nodes: %w", err)
	}
	return nodes.Items, nil
}

// ListWorkers returns a slice with all the worker nodes in the K8s cluster.
// Worker nodes are nodes that DO NOT have the taints with the keys contained in
// `c.MasterNodeTaints`.
// ListWorkers returns an empty slice and an error if a failure occurs.
func (c Client) ListWorkers(ctx context.Context) ([]v1.Node, error) {
	allNodes, err := c.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	workerNodes := make([]v1.Node, 0, len(allNodes))
	for _, n := range allNodes {
		if !c.hasMasterNodeTaints(n.Spec.Taints) {
			workerNodes = append(workerNodes, n)
		}
	}

	return workerNodes, nil
}

// TaintWorkers adds the taints `t` to all the worker nodes in the K8s cluster.
// Worker nodes are nodes that DO NOT have the taints with the keys contained in
// `c.MasterNodeTaints`.
// TaintWorkers is idempotent:
//   - if a worker already has a subset of the taints in `t`, only the missing ones are added.
//   - if a worker already has all the taints in `t`, it's left unchanged.
//
// If a worker has one (or more) taint(s) with the same key as one of the taints in `t` but
// different value or effect, TaintWorkers panics.
// TaintWorkers returns an error if a failure occurs.
func (c Client) TaintWorkers(ctx context.Context, t []v1.Taint) error {
	workers, err := c.ListWorkers(ctx)
	if err != nil {
		return fmt.Errorf("failed to taint K8s worker nodes: %w", err)
	}

	// Here we don't fail fast. Rather than returning an error on the first failure, we try to
	// taint as many nodes as possible, i.e., even if tainting a node fails we try to taint
	// the remaining nodes. This is done because users of this library are e2e and integration tests
	// that require tainting to succeed and will retry tainting multiple
	// times before giving up, so it's faster to try to always tainting as many
	// nodes as possible.
	var errs []error
	for _, w := range workers {
		if err := c.taint(ctx, w, t); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("tainting some nodes failed: %v", errors.NewAggregate(errs))
	}

	return nil
}

// UntaintAll removes the taints `t` from all the nodes (worker and master ones) in the K8s cluster.
// UntaintAll is idempotent and safe to retry:
//   - if a node has only a subset of the taints in `t`, only that subset is removed.
//   - if a node doesn't have any of the taints in `t`, it's left unchanged.
//
// If a node has one (or more) taint(s) with the same key as one of the taints in `t` but different
// value or effect, UntaintAll panics.
// UntaintAll returns an error if a failure occurs.
func (c Client) UntaintAll(ctx context.Context, t []v1.Taint) error {
	nodes, err := c.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to untaint K8s nodes: %w", err)
	}

	// Here we don't fail fast. Rather than returning an error on the first failure, we try to
	// untaint as many nodes as possible, i.e., even if untainting a node fails we try to untaint
	// the remaining nodes. This is done because users of this library are e2e and integration tests
	// that require untainting to succeed and will retry untainting multiple
	// times before giving up, so it's faster to try to always untaint as many
	// nodes as possible.
	var errs []error
	for _, n := range nodes {
		if err := c.untaint(ctx, n, t); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("removing taints from some nodes failed: %v", errors.NewAggregate(errs))
	}

	return nil
}

// UnlabelAll removes the labels with the keys in `labelsKeys` from all the nodes (worker and master
// ones) in the K8s cluster, regardless of the labels values.
// UnlabelAll is idempotent and safe to retry:
//   - if a node has only a subset of the labels in `labelsKeys`, only that subset is removed.
//   - if a node doesn't have any of the labels in `labelsKeys`, it's left unchanged.
//
// UnlabelAll returns an error if a failure occurs.
func (c Client) UnlabelAll(ctx context.Context, labelsKeys []string) error {
	nodes, err := c.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to unlabel K8s nodes: %w", err)
	}

	// Here we don't fail fast. Rather than returning an error on the first failure, we try to
	// unlabel as many nodes as possible, i.e., even if unlabeling a node fails we try to unlabel
	// the remaining nodes. This is done because users of this library are e2e and integration tests
	// that require unlabeling to succeed, and will retry it if it fails, so it's actually faster
	// to always try to unlabel as many nodes as possible.
	var errs []error
	for _, n := range nodes {
		if err := c.unlabel(ctx, n, labelsKeys); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("unlabeling all or some nodes failed: %v", errors.NewAggregate(errs))
	}

	return nil
}

func (c Client) taint(ctx context.Context, n v1.Node, t []v1.Taint) error {
	if len(n.Spec.Taints) > 0 {
		c.Log.Info(
			"Warning: Node is already tainted with taints. This might break the tolerations tests.",
			"node", n.Name, "taints", n.Spec.Taints,
		)
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

func (c Client) Label(ctx context.Context, n v1.Node, labels map[string]string) error {
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

func (c Client) unlabel(ctx context.Context, n v1.Node, labelsKeysToRemove []string) error {
	labelsChanged := false
	for _, key := range labelsKeysToRemove {
		if _, found := n.Labels[key]; found {
			delete(n.Labels, key)
			labelsChanged = true
		}
	}

	if !labelsChanged {
		return nil
	}

	if _, err := c.Nodes.Update(ctx, &n, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update of node %s to remove labels with keys %v failed: %w",
			n.Name, labelsKeysToRemove, err)
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
		union[key] = val
	}

	return union
}

func (c Client) hasMasterNodeTaints(taints []v1.Taint) bool {
	for _, t := range taints {
		if _, found := c.MasterNodeTaints[t.Key]; found {
			return true
		}
	}
	return false
}
