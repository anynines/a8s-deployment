package chaos

import (
	"context"
	"errors"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	apixv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/anynines/a8s-deployment/test/framework/chaos/podchaos"
)

var (
	chaosControllerLabels = labels.Set{
		"app.kubernetes.io/component": "controller-manager",
		"app.kubernetes.io/instance":  "chaos-mesh",
	}
	chaosDaemonSetLabels = labels.Set{
		"app.kubernetes.io/component": "chaos-daemon",
		"app.kubernetes.io/instance":  "chaos-mesh",
	}
)

type ChaosObject interface {
	// CheckChaosActive checks whether the effect of the applied chaos is already active
	CheckChaosActive(ctx context.Context, c runtimeClient.Client) (bool, error)
	KubernetesObject() runtimeClient.Object
}

func VerifyChaosMeshPresent(ctx context.Context, c runtimeClient.Client) error {
	// Add CRD definition
	apixv1.AddToScheme(scheme.Scheme)

	err := verifyChaosMeshCRDsInstalled(ctx, c)
	if err != nil {
		return err
	}

	return verifyChaosMeshControllersRunning(ctx, c)
}

// TODO: Verify all chaos CRDs are installed
func verifyChaosMeshCRDsInstalled(ctx context.Context, c runtimeClient.Client) error {
	crd := apixv1.CustomResourceDefinition{}
	err := c.Get(ctx, types.NamespacedName{Name: podchaos.CRDName}, &crd)
	if k8serrors.IsNotFound(err) || k8serrors.IsGone(err) {
		return fmt.Errorf("missing ChaosMesh CRD %s", podchaos.CRDName)
	}
	if err != nil {
		return fmt.Errorf("failed to verify presence of ChaosMesh CRD %s: %w", podchaos.CRDName, err)
	}

	for _, version := range crd.Spec.Versions {
		if version.Name == podchaos.RequiredVersion {
			return nil
		}
	}

	return fmt.Errorf("required version of ChaosMesh CRD %s not installed: needs %s",
		podchaos.CRDName, podchaos.RequiredVersion)
}

func verifyChaosMeshControllersRunning(ctx context.Context, c runtimeClient.Client) error {
	deploymentList := appsv1.DeploymentList{}

	labelSelector, err := chaosControllerLabels.AsValidatedSelector()
	if err != nil {
		panic("ChaosMesh controller Deployment labels invalid")
	}

	err = c.List(ctx, &deploymentList, &runtimeClient.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return fmt.Errorf("failed to get ChaosMesh Deployments: %w", err)
	}

	if len(deploymentList.Items) == 0 {
		return errors.New("could not find ChaosMesh controller Deployment")
	}

	daemonSetList := appsv1.DaemonSetList{}
	labelSelector, err = chaosDaemonSetLabels.AsValidatedSelector()
	if err != nil {
		panic("ChaosMesh controller DaemonSet labels invalid")
	}

	err = c.List(ctx, &daemonSetList, &runtimeClient.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return fmt.Errorf("failed to get ChaosMesh DaemonSet: %w", err)
	}

	if len(daemonSetList.Items) == 0 {
		return errors.New("could not find ChaosMesh DaemonSet")
	}

	return nil
}
