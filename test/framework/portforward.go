package framework

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	AsyncOpsTimeoutMins = time.Minute * 5
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
	c runtimeClient.Client,
) (chan struct{}, int, error) {
	pod, err := GetPrimaryPodUsingServiceSelector(ctx, dsi, c)
	if err != nil {
		return nil, -1, err
	}

	return PortForwardPod(ctx, targetPort, pathToKubeConfig, pod, c)
}

// PortForwardPod d establishes a port-forward from a randomly selected local port to port `targetPort`
// of `pod`.
//
//	To terminate the port-forward, close the returned channel.
//	The other return arguments are the selected local port, and an error in case of failure.
func PortForwardPod(ctx context.Context,
	targetPort int,
	pathToKubeConfig string,
	pod *corev1.Pod,
	c runtimeClient.Client,
) (chan struct{}, int, error) {
	config, err := clientcmd.BuildConfigFromFlags("", pathToKubeConfig)
	if err != nil {
		panic(err)
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

	var fw *portforward.PortForwarder

	fw, err = portForwardAPod(portForwardAPodRequest{
		restConfig: config,
		pod:        *pod,
		localPort:  0, // causes a random port to be chosen
		targetPort: targetPort,
		streams:    stream,
		stopCh:     stopCh,
		readyCh:    readyCh,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to configure port forward for pod %s/%s port %d",
			pod.Namespace, pod.Name, targetPort)
	}

	go func() {
		err := fw.ForwardPorts()
		if err != nil {
			panic(fmt.Sprintf("An error occurred during port forwarding for pod %s/%s port %d: %s",
				pod.Namespace, pod.Name, targetPort, err))
		}
	}()

	<-readyCh

	portList, err := fw.GetPorts()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get local port of port forward for pod %s/%s port %d",
			pod.Namespace, pod.Name, targetPort)
	}

	if len(portList) != 1 {
		return nil, 0, fmt.Errorf("unexpected number of forwarded ports %d, only"+
			" one should be forwarded for pod %s/%s",
			len(portList), pod.Namespace, pod.Name)
	}
	localPort := int(portList[0].Local)
	log.Printf("Forwarding pod %s/%s port %d on local port %d",
		pod.Namespace, pod.Name, targetPort, localPort)

	return stopCh, localPort, err
}

func portForwardAPod(req portForwardAPodRequest) (*portforward.PortForwarder, error) {
	path := fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/portforward",
		req.pod.Namespace, req.pod.Name)
	hostIP := strings.TrimLeft(req.restConfig.Host, "htps:/")

	transport, upgrader, err := spdy.RoundTripperFor(req.restConfig)
	if err != nil {
		return nil, err
	}

	dialer := spdy.NewDialer(upgrader,
		&http.Client{Transport: transport},
		http.MethodPost,
		&url.URL{Scheme: "https", Path: path, Host: hostIP})

	fw, err := portforward.New(dialer,
		[]string{fmt.Sprintf("%d:%d", req.localPort, req.targetPort)},
		req.stopCh, req.readyCh, req.streams.Out, req.streams.ErrOut)
	if err != nil {
		return nil, err
	}

	return fw, nil
}

func primarySvcSelector(ctx context.Context,
	dsi runtimeClient.Object,
	c runtimeClient.Client,
) (*labels.Selector, error) {
	svcName := fmt.Sprintf("%s-master", dsi.GetName())
	var svc corev1.Service
	err := c.Get(ctx, types.NamespacedName{Name: svcName, Namespace: dsi.GetNamespace()}, &svc)
	if err != nil {
		return nil, fmt.Errorf("unable to get primary service: %w", err)
	}

	selector, err := labels.ValidatedSelectorFromSet(svc.Spec.Selector)
	if err != nil {
		return nil, fmt.Errorf("failed to create selector for primary pod from primary service's "+
			"spec.selector %s: %w", svc.Spec.Selector, err)
	}
	return &selector, nil
}

// TODO: Wait for primary pod explicitly in test setup rather than here. Asynchronous assertions
// are better exposed at the top level of the setup than hidden within the port forward logic itself.
// GetPrimaryPodUsingServiceSelector requires an eventually assertion since Patroni only applies the master
// label once quorum has been achieved. We must wait for this before we can know which pod to
// portforward to using the service selector.
func GetPrimaryPodUsingServiceSelector(ctx context.Context,
	dsi runtimeClient.Object,
	c runtimeClient.Client,
) (*corev1.Pod, error) {
	svcSelector, err := primarySvcSelector(ctx, dsi, c)
	if err != nil {
		return nil, fmt.Errorf("unable to get selector for service: %w", err)
	}

	var primaryPod *corev1.Pod
	EventuallyWithOffset(1, func() error {
		primaryPodList := &corev1.PodList{}
		if err := c.List(ctx, primaryPodList, &runtimeClient.ListOptions{
			Namespace:     dsi.GetNamespace(),
			LabelSelector: *svcSelector,
		}); err != nil {
			return err
		}
		if len(primaryPodList.Items) != 1 {
			return fmt.Errorf("found %d primary pods, expected only 1",
				len(primaryPodList.Items))
		}
		primaryPod = &primaryPodList.Items[0]
		return nil
	}, AsyncOpsTimeoutMins, 1*time.Second).
		Should(
			BeNil(),
			"timeout reached to get primary pod using service selector for dsi %s/%s",
			dsi.GetNamespace(), dsi.GetName(),
		)
	return primaryPod, nil
}
