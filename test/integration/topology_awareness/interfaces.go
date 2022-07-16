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
	LabelWorkers(context.Context, map[string]string) error
	UnlabelAll(context.Context, map[string]string) error
}
