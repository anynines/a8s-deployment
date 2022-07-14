package topology_awareness

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

type NodesClient interface {
	NodesTainter
	NodesLabeler
}

type NodesTainter interface {
	TaintAll(context.Context, []corev1.Taint) error
	UntaintAll(context.Context, []corev1.Taint) error
}

type NodesLabeler interface {
	LabelAll(context.Context, map[string]string) error
	UnlabelAll(context.Context, map[string]string) error
}
