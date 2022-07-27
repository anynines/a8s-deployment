package topology_awareness

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

type NodesClient interface {
	NodesLister
	NodesTainter
	NodesLabeler
}

type NodesLister interface {
	ListAll(context.Context) ([]corev1.Node, error)
	ListWorkers(context.Context) ([]corev1.Node, error)
}

type NodesTainter interface {
	TaintWorkers(context.Context, []corev1.Taint) error
	UntaintAll(context.Context, []corev1.Taint) error
}

type NodesLabeler interface {
	Label(ctx context.Context, node corev1.Node, labels map[string]string) error
	UnlabelAll(ctx context.Context, keysOfLabelsToRemove []string) error
	GetLabels(ctx context.Context, node string) (map[string]string, error)
}
