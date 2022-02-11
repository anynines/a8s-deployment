package dsi

import (
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/anynines/a8s-deployment/test/integration/framework/postgresql"
)

func NewK8sClient(ds, kubeconfig string) (client.Client, error) {
	switch strings.ToLower(ds) {
	case "postgresql":
		return postgresql.NewK8sClient(kubeconfig)
	}
	return nil, fmt.Errorf(
		"kubernetes client factory received request to create kubernetes client for unknown data service %s; only supported data services are %s",
		ds,
		supportedDataServices(),
	)
}
