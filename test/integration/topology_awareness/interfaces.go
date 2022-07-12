package topology_awareness

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

type NodesTainter interface {
	TaintAll(context.Context, []corev1.Taint) error
	UntaintAll(context.Context, []corev1.Taint) error
}
