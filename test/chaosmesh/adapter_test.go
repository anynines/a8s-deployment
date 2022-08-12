package chaosmesh_test

import (
	"context"
	"testing"

	"github.com/anynines/a8s-deployment/test/chaosmesh"
)

func TestNilObj(t *testing.T) {
	a := chaosmesh.FaultInjector{}

	a.IsolatePrimary(context.TODO(), nil)
}
