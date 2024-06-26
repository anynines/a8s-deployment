package dsi

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/anynines/a8s-deployment/test/framework/postgresql"
)

const (
	clusterStatusRunning = "Running"
	// asyncOpsTimeoutMins is the amount of minutes after which assertions fail if the condition
	// they check has not become true. Needed because some conditions might become true only after
	// some time, so we need to check them asynchronously.
	// TODO: Make asyncOpsTimeoutMins an invocation parameter.
	asyncOpsTimeoutMins = time.Minute * 5
)

type Object interface {
	ClusterStatus() string
	runtimeClient.Object
	GetClientObject() runtimeClient.Object
}

type TolerationsSetter interface {
	SetTolerations(...corev1.Toleration)
}

type WithPodAntiAffinity interface {
	AddRequiredPodAntiAffinityTerm(antiAffinityTerm corev1.PodAffinityTerm)
	AddPreferredPodAntiAffinityTerm(weight int, antiAffinityTerm corev1.PodAffinityTerm)
}

type StatefulSetGetter interface {
	StatefulSet(context.Context, runtimeClient.Client) (*appsv1.StatefulSet, error)
}

type PodsGetter interface {
	Pods(context.Context, runtimeClient.Client) ([]corev1.Pod, error)
}

// This package does not use functional options like others in the framework since we need to
// access the properties of structs. We would need to implement methods to expose these properties
// which would negate some of the value of functional options.

func New(ds, namespace, name string, replicas int32) (Object, error) {
	switch strings.ToLower(ds) {
	case "postgresql":
		return postgresql.New(namespace, name, replicas), nil
	}
	return nil, fmt.Errorf(
		"dsi factory received request to create dsi for unknown data service %s; only supported data services are %s",
		ds,
		supportedDataServices(),
	)
}

func newEmpty(ds string) (Object, error) {
	switch strings.ToLower(ds) {
	case "postgresql":
		return postgresql.NewEmpty(), nil
	}
	return nil, fmt.Errorf(
		"dsi factory received request to create empty dsi for unknown data service %s; only supported data services are %s",
		ds,
		supportedDataServices(),
	)
}

func supportedDataServices() string {
	return "PostgreSQL"
}

// TODO: rather than having all these functions here, consider switching to an OOP approach where
// each instance object exposes these functions for itself as methods.
// This function does not check if replicas have properly started and have
// their replication role defined in the labels, thus it might not enough for
// all test cases, especially chaos tests.

func WaitForReadiness(ctx context.Context, instance runtimeClient.Object, c runtimeClient.Client) {
	var err error
	EventuallyWithOffset(1, func() string {
		instanceCreated, err := newEmpty(instance.GetObjectKind().GroupVersionKind().Kind)
		if err != nil {
			return fmt.Sprintf("%v+", err)
		}
		if err = c.Get(
			ctx,
			types.NamespacedName{
				Name: instance.GetName(), Namespace: instance.GetNamespace()},
			instanceCreated.GetClientObject(),
		); err != nil {
			return fmt.Sprintf("%v+", err)
		}
		return instanceCreated.ClusterStatus()
	}, asyncOpsTimeoutMins, 1*time.Second).Should(Equal(clusterStatusRunning),
		fmt.Sprintf("timeout reached waiting for instance %s/%s readiness: %s",
			instance.GetNamespace(),
			instance.GetName(),
			err,
		),
	)
}

// WaitForReplicaReadiness waits until the given number of ReplicaPods report as ready.
func WaitForReplicaReadiness(ctx context.Context, instance runtimeClient.Object,
	c runtimeClient.Client, replicas int) {

	var err error
	EventuallyWithOffset(1, func() bool {
		dsiPods, err := GetPodsWithLabels(ctx, c, instance.GetNamespace(),
			map[string]string{
				"a8s.a9s/dsi-name": instance.GetName(),
			})
		if err != nil {
			return false
		}

		return NPodsReady(dsiPods) == replicas
	}, asyncOpsTimeoutMins, 1*time.Second).Should(BeTrue(),
		fmt.Sprintf("timeout reached waiting for instance %s/%s readiness: %s",
			instance.GetNamespace(),
			instance.GetName(),
			err,
		),
	)
}

func WaitForDeletion(ctx context.Context, instance runtimeClient.Object, c runtimeClient.Client) {
	var err error
	EventuallyWithOffset(1, func() bool {
		instanceCreated, err := newEmpty(instance.GetObjectKind().GroupVersionKind().Kind)
		if err != nil {
			log.Println("failed to generate empty instance for dataservice while waiting for deletion")
			// Return early since this err is unrecoverable.
			return true
		}
		err = c.Get(
			ctx,
			types.NamespacedName{
				Name: instance.GetName(), Namespace: instance.GetNamespace()},
			instanceCreated.GetClientObject(),
		)
		return err != nil && errors.IsNotFound(err)
	}, asyncOpsTimeoutMins, 1*time.Second).Should(BeTrue(),
		fmt.Sprintf("timeout reached waiting for instance %s/%s deletion: %s",
			instance.GetNamespace(),
			instance.GetName(),
			err,
		),
	)
}

func WaitForPodDeletion(ctx context.Context, pod *corev1.Pod, c runtimeClient.Client) {
	var err error
	EventuallyWithOffset(1, func() bool {
		podCreated := &corev1.Pod{}
		if err = c.Get(
			ctx,
			types.NamespacedName{
				Name: pod.GetName(), Namespace: pod.GetNamespace()},
			podCreated,
		); err != nil {
			log.Println("failed to wait for pod to be deleted")
			return false
		}
		return podCreated.DeletionTimestamp == nil
	}, asyncOpsTimeoutMins, 1*time.Second).Should(BeTrue(),
		fmt.Sprintf("timeout reached waiting for pod %s/%s deletion: %s",
			pod.GetNamespace(),
			pod.GetName(),
			err,
		),
	)
}

func GetPodsWithLabels(ctx context.Context, c runtimeClient.Client,
	namespace string, label map[string]string) (*corev1.PodList, error) {

	selector, err := labels.Set(label).AsValidatedSelector()
	if err != nil {
		return nil, err
	}

	podList := &corev1.PodList{}

	err = c.List(ctx, podList, &runtimeClient.ListOptions{
		Namespace:     namespace,
		LabelSelector: selector,
	})

	return podList, err
}

// NPodsReady returns number ready pods in the given PodList.
func NPodsReady(l *corev1.PodList) int {
	podsRunning := 0
	for _, pod := range l.Items {
		if IsPodReady(&pod) {
			podsRunning++
		}
	}

	return podsRunning
}

// Determines readiness of pods by looking at container statuses
func IsPodReady(pod *corev1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if !containerStatus.Ready {
			return false
		}
	}

	return true
}
