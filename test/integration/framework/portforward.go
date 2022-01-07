package framework

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	minPort             = 1024
	maxPort             = 65535
	asyncOpsTimeoutMins = time.Minute * 5
)

// TODO: This portforward logic contains some data service specific implementation details such as the
// name of the service, ect. It would make sense for port forward logic to be implemented for each
// data service so that we can hide these implementation details beneath abstraction.

type portForwardAPodRequest struct {
	// restConfig is the kubernetes config
	restConfig *rest.Config
	// pod is the selected pod for this port forwarding
	pod corev1.Pod
	// localPort is the local port that will be selected to expose the TargetPort
	localPort int
	// targetPort is the target port for the pod
	targetPort int
	// streams configures where to write or read input from
	streams genericclioptions.IOStreams
	// StopCh is the channel to close to terminate the port-forwarding
	stopCh <-chan struct{}
	// readyCh communicates when the tunnel is ready to receive traffic
	readyCh chan struct{}
}

// PortForward establishes a port-forward from a randomly selected local port to port `targetPort`
// of `dsi`.
// To terminate the port-forward, close the returned channel.
// The other return arguments are the selected local port, and an error in case of failure.
func PortForward(ctx context.Context,
	targetPort int,
	pathToKubeConfig string,
	dsi runtimeClient.Object,
	c runtimeClient.Client) (chan struct{}, int, error) {

	config, err := clientcmd.BuildConfigFromFlags("", pathToKubeConfig)
	if err != nil {
		panic(err)
	}

	pod, err := getPrimaryPodUsingServiceSelector(ctx, dsi, c)
	if err != nil {
		return nil, -1, err
	}

	// stopCh control the port forwarding lifecycle. When it gets closed the
	// port forward will terminate
	stopCh := make(chan struct{})
	// readyCh communicate when the port forward is ready to get traffic
	readyCh := make(chan struct{})
	// stream is used to tell the port forwarder where to place its output or
	// where to expect input if needed.
	stream := genericclioptions.IOStreams{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
	}

	var localPort int
	go func() {
		var err error
		for i := 0; i < 10; i++ {
			localPort = pickPort()
			err = portForwardAPod(portForwardAPodRequest{
				restConfig: config,
				pod:        *pod,
				localPort:  localPort,
				targetPort: targetPort,
				streams:    stream,
				stopCh:     stopCh,
				readyCh:    readyCh,
			})
			if err == nil {
				break
			}
			log.Printf("Attempt number %d failed to bind to port %d: %s", i, localPort, err)
		}
		if err != nil {
			panic("we ran out of retries to find a port to portforward")
		}
	}()

	<-readyCh
	return stopCh, localPort, nil
}

func portForwardAPod(req portForwardAPodRequest) error {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward",
		req.pod.Namespace, req.pod.Name)
	hostIP := strings.TrimLeft(req.restConfig.Host, "htps:/")

	transport, upgrader, err := spdy.RoundTripperFor(req.restConfig)
	if err != nil {
		return err
	}

	dialer := spdy.NewDialer(upgrader,
		&http.Client{Transport: transport},
		http.MethodPost,
		&url.URL{Scheme: "https", Path: path, Host: hostIP})

	fw, err := portforward.New(dialer,
		[]string{fmt.Sprintf("%d:%d", req.localPort, req.targetPort)},
		req.stopCh, req.readyCh, req.streams.Out, req.streams.ErrOut)
	if err != nil {
		return err
	}
	return fw.ForwardPorts()
}

func primarySvcSelector(ctx context.Context,
	dsi runtimeClient.Object,
	c runtimeClient.Client) (*labels.Selector, error) {

	svcName := fmt.Sprintf("%s-master", dsi.GetName())
	var svc corev1.Service
	err := c.Get(ctx, types.NamespacedName{
		Name:      svcName,
		Namespace: dsi.GetNamespace(),
	}, &svc)
	if err != nil {
		return nil, fmt.Errorf("unable to get primary service: %w", err)
	}

	selector := labels.NewSelector()
	for key, val := range svc.Spec.Selector {
		newRequirement, err := labels.NewRequirement(key, selection.Equals, []string{val})
		if err != nil {
			return nil, fmt.Errorf("unable to create requirement from label %s=%s: %w",
				key, val, err)
		}
		selector = selector.Add(*newRequirement)
	}
	return &selector, nil
}

func pickPort() int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return minPort + r.Intn(maxPort-minPort+1)
}

// TODO: Wait for primary pod explicitly in test setup rather than here. Asynchronous assertions
// are better exposed at the top level of the setup than hidden within the port forward logic itself.
// getPrimaryPodUsingServiceSelector requires an eventually assertion since Patroni only applies the master
// label once quorum has been achieved. We must wait for this before we can know which pod to
// portforward to using the service selector.
func getPrimaryPodUsingServiceSelector(ctx context.Context,
	dsi runtimeClient.Object,
	c runtimeClient.Client) (*corev1.Pod, error) {

	svcSelector, err := primarySvcSelector(ctx, dsi, c)
	if err != nil {
		return nil, fmt.Errorf("unable to get selector for service: %w", err)
	}

	var primaryPod *corev1.Pod
	EventuallyWithOffset(1, func() error {
		primaryPodList := &corev1.PodList{}
		if err := c.List(ctx, primaryPodList, &runtimeClient.ListOptions{
			Namespace:     dsi.GetNamespace(),
			LabelSelector: *svcSelector}); err != nil {

			return err
		}
		if len(primaryPodList.Items) != 1 {
			return fmt.Errorf("found %d primary pods, expected only 1",
				len(primaryPodList.Items))
		}
		primaryPod = &primaryPodList.Items[0]
		return nil
	}, asyncOpsTimeoutMins, 1*time.Second).
		Should(
			BeNil(),
			"timeout reached to get primary pod using service selector for dsi %s/%s",
			dsi.GetNamespace(), dsi.GetName(),
		)
	return primaryPod, nil
}
