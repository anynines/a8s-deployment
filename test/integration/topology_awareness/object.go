package topology_awareness

import (
	"fmt"

	"github.com/anynines/a8s-deployment/test/integration/framework/dsi"
)

type Object interface {
	dsi.Object
	dsi.StatefulSetGetter
	dsi.PodsGetter
	dsi.TolerationsSetter
}

func newDSI(ds, namespace, name string, replicas int32) (Object, error) {
	baseObj, err := dsi.New(ds, namespace, name, replicas)
	if err != nil {
		return nil, err
	}

	// Convert baseObj to a topology_awareness.Object
	taObj, ok := baseObj.(Object)
	if !ok {
		return nil, fmt.Errorf("can't create topology-aware DSI for data service %s because "+
			"the data service doesn't implement interface \"topology_awareness.Object\"", ds)
	}

	return taObj, nil
}
