package topology_awareness

import (
	"context"
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/anynines/a8s-deployment/test/framework"
	"github.com/anynines/a8s-deployment/test/framework/dsi"
	"github.com/anynines/a8s-deployment/test/framework/secret"
	"github.com/anynines/a8s-deployment/test/framework/servicebinding"
	sbv1alpha1 "github.com/anynines/a8s-service-binding-controller/api/v1alpha1"
)

// TODO: Test broken cases where the DSI has no tolerations: 1 taint - 0 tolerations;
// 	     2 taints - 1 toleration; 1 taint - 1 toleration that doesn't match it.
// TODO: Test removing tolerations from an existing DSI.
// TODO: Test adding tolerations to an existing DSI.
// TODO: Test horizontal scale up.
// TODO: Test cases where only a subset of nodes is tainted.
// TODO: Test cases where we update affinity and anti-affinity.
// TODO: Test cases with anti-affinity and upscaling.
// TODO: Test cases with pod-affinity and upscaling.

const (
	suffixLength = 5

	appsDefaultDB = "a9s_apps_default_db"

	instancePort = 5432

	taintingTimeout = 15 * time.Second
	labelingTimeout = 15 * time.Second
)

var _ = Describe("DSIs topology awareness", func() {
	var (
		err error
		ctx context.Context

		instance       Object
		instanceNSN    string
		instanceClient dsi.DSIClient

		sb            *sbv1alpha1.ServiceBinding
		sbCredentials secret.SecretData

		portForwardStopCh chan struct{}
		localPort         int
	)

	Context("DSI has tolerations to node taints", func() {
		var (
			taints      []corev1.Taint
			tolerations []corev1.Toleration
		)

		BeforeEach(func() {
			ctx = context.Background()
		})

		AfterEach(func() {
			close(portForwardStopCh)

			Eventually(func() error { return nodes.UntaintAll(ctx, taints) }, taintingTimeout).
				Should(Succeed(), "failed to untaint nodes")

			Expect(k8sClient.Delete(ctx, sb)).To(Succeed(),
				"failed to delete DSI "+instanceNSN+"'s ServiceBinding")

			Expect(k8sClient.Delete(ctx, instance.GetClientObject())).To(Succeed(),
				"failed to delete DSI "+instanceNSN)
		})

		Context("One taint - one toleration", func() {
			BeforeEach(func() {
				taints = []corev1.Taint{
					{
						Key:    "a8s-test-taint-1",
						Value:  "dummy-val-1",
						Effect: "NoSchedule",
					},
				}
				tolerations = []corev1.Toleration{
					{
						Key:      "a8s-test-taint-1",
						Operator: corev1.TolerationOpEqual,
						Value:    "dummy-val-1",
						Effect:   "NoSchedule",
					},
				}

				Eventually(func() error {
					return nodes.TaintWorkers(ctx, taints)
				}, taintingTimeout).Should(Succeed())
			})

			It("Implements a 1-replica DSI that tolerates the node taint", func() {
				replicas := int32(1)

				By("Accepting the creation of the DSI API object", func() {
					// Create the DSI K8s API object
					instance, err = newDSI(dataservice, testingNamespace,
						framework.GenerateName(instanceNamePrefix, GinkgoParallelProcess(),
							suffixLength), replicas)
					Expect(err).To(BeNil(), "failed to generate DSI object")
					instanceNSN = instance.GetNamespace() + "/" + instance.GetName()
					instance.SetTolerations(tolerations...)
					Expect(k8sClient.Create(ctx, instance.GetClientObject())).To(Succeed(),
						"failed to create DSI "+instanceNSN)
				})

				By("Creating a healthy StatefulSet for the DSI", func() {
					Eventually(func(g Gomega) {
						sset, err := instance.StatefulSet(ctx, k8sClient)
						g.Expect(err).To(BeNil(), "failed to get the DSI "+instanceNSN+
							"'s StatefulSet")
						g.Expect(sset.Status.ReadyReplicas).To(Equal(replicas),
							"ready replicas of DSI "+instanceNSN+
								"'s StatefulSet don't match DSI's desired replicas")
					}, 5*time.Minute).Should(Succeed(),
						"failed to verify that the DSI's StatefulSet gets up and running")
				})

				By("Accepting a ServiceBinding to the DSI", func() {
					sb = servicebinding.New(
						servicebinding.SetNamespacedName(instance.GetClientObject()),
						servicebinding.SetInstanceRef(instance.GetClientObject()),
					)
					Expect(k8sClient.Create(ctx, sb)).To(Succeed(),
						"failed to create ServiceBinding for DSI "+instanceNSN)
				})

				By("Marking the ServiceBinding as implemented", func() {
					Eventually(func(g Gomega) {
						sbNSN := types.NamespacedName{Namespace: sb.Namespace, Name: sb.Name}
						g.Expect(k8sClient.Get(ctx, sbNSN, sb)).
							To(Succeed(), "failed to get DSI "+instanceNSN+"'s ServiceBinding")
						g.Expect(sb.Status.Implemented).
							To(BeTrue(), "ServiceBinding of DSI "+instanceNSN+" isn't implemented")
					}, 10*time.Second).Should(Succeed(),
						"failed to verify that the ServiceBinding of DSI "+instanceNSN+
							" gets implemented")
				})

				By("Generating a client to the DSI to read/write to/from it", func() {
					// Get credentials to write to the DSI
					sbCredentials, err = secret.Data(
						ctx, k8sClient, servicebinding.SecretName(sb.Name), testingNamespace)
					Expect(err).To(BeNil(), "failed to parse secret data for ServiceBinding "+
						sb.GetNamespace()+"/"+sb.GetName())

					// Create a portforwarding to write to the DSI from out of cluster
					portForwardStopCh, localPort, err = framework.PortForward(
						ctx, instancePort, kubeconfigPath, instance, k8sClient)
					Expect(err).To(BeNil(), "failed to establish portforward to DSI "+instanceNSN)

					// Generate a client to the DSI using the credentials and the portforwarding
					instanceClient, err = dsi.NewClient(dataservice,
						strconv.Itoa(localPort),
						sbCredentials)
					Expect(err).To(BeNil(), "failed to create client to DSI "+instanceNSN)
				})

				By("Setting up the DSI so that clients can write/read data into/from it", func() {
					Expect(instanceClient.Write(ctx, appsDefaultDB, "sample data")).To(Succeed(),
						"failed to write into database "+appsDefaultDB+" in DSI "+instanceNSN)
					readData, err := instanceClient.Read(ctx, appsDefaultDB)
					Expect(err).To(BeNil(),
						"failed to read data from database "+appsDefaultDB+" in DSI "+instanceNSN)
					Expect(readData).To(Equal("sample data"),
						"data read from database "+appsDefaultDB+" in DSI "+instanceNSN+
							" doesn't match previously written data")
				})
			})

			It("Implements a 3-replica DSI that tolerates the node taint", func() {
				replicas := int32(3)

				By("Accepting the creation of the DSI API object", func() {
					// Create the DSI K8s API object
					instance, err = newDSI(dataservice, testingNamespace,
						framework.GenerateName(instanceNamePrefix, GinkgoParallelProcess(),
							suffixLength), replicas)
					Expect(err).To(BeNil(), "failed to generate DSI object")
					instanceNSN = instance.GetNamespace() + "/" + instance.GetName()
					instance.SetTolerations(tolerations...)
					Expect(k8sClient.Create(ctx, instance.GetClientObject())).To(Succeed(),
						"failed to create DSI "+instanceNSN)
				})

				By("Creating a healthy StatefulSet for the DSI", func() {
					Eventually(func(g Gomega) {
						sset, err := instance.StatefulSet(ctx, k8sClient)
						g.Expect(err).To(BeNil(), "failed to get the DSI "+instanceNSN+
							"'s StatefulSet")
						g.Expect(sset.Status.ReadyReplicas).To(Equal(replicas),
							"ready replicas of DSI "+instanceNSN+
								"'s StatefulSet don't match DSI's desired replicas")
					}, 5*time.Minute).Should(Succeed(),
						"failed to verify that the DSI's StatefulSet gets up and running")
				})

				By("Accepting a ServiceBinding to the DSI", func() {
					sb = servicebinding.New(
						servicebinding.SetNamespacedName(instance.GetClientObject()),
						servicebinding.SetInstanceRef(instance.GetClientObject()),
					)
					Expect(k8sClient.Create(ctx, sb)).To(Succeed(),
						"failed to create ServiceBinding for DSI "+instanceNSN)
				})

				By("Marking the ServiceBinding as implemented", func() {
					Eventually(func(g Gomega) {
						sbNSN := types.NamespacedName{Namespace: sb.Namespace, Name: sb.Name}
						g.Expect(k8sClient.Get(ctx, sbNSN, sb)).
							To(Succeed(), "failed to get DSI "+instanceNSN+"'s ServiceBinding")
						g.Expect(sb.Status.Implemented).
							To(BeTrue(), "ServiceBinding of DSI "+instanceNSN+" isn't implemented")
					}, 10*time.Second).Should(Succeed(),
						"failed to verify that the ServiceBinding of DSI "+instanceNSN+
							" gets implemented")
				})

				By("Generating a client to the DSI to read/write to/from it", func() {
					// Get credentials to write to the DSI
					sbCredentials, err = secret.Data(
						ctx, k8sClient, servicebinding.SecretName(sb.Name), testingNamespace)
					Expect(err).To(BeNil(), "failed to parse secret data for ServiceBinding "+
						sb.GetNamespace()+"/"+sb.GetName())

					// Create a portforwarding to write to the DSI from out of cluster
					portForwardStopCh, localPort, err = framework.PortForward(
						ctx, instancePort, kubeconfigPath, instance, k8sClient)
					Expect(err).To(BeNil(), "failed to establish portforward to DSI "+instanceNSN)

					// Generate a client to the DSI using the credentials and the portforwarding
					instanceClient, err = dsi.NewClient(dataservice,
						strconv.Itoa(localPort),
						sbCredentials)
					Expect(err).To(BeNil(), "failed to create client to DSI "+instanceNSN)
				})

				By("Setting up the DSI so that clients can write/read data into/from it", func() {
					Expect(instanceClient.Write(ctx, appsDefaultDB, "sample data")).To(Succeed(),
						"failed to write into database "+appsDefaultDB+" in DSI "+instanceNSN)
					readData, err := instanceClient.Read(ctx, appsDefaultDB)
					Expect(err).To(BeNil(),
						"failed to read data from database "+appsDefaultDB+" in DSI "+instanceNSN)
					Expect(readData).To(Equal("sample data"),
						"data read from database "+appsDefaultDB+" in DSI "+instanceNSN+
							" doesn't match previously written data")
				})
			})
		})

		Context("Two taints - two tolerations", func() {
			BeforeEach(func() {
				taints = []corev1.Taint{
					{
						Key:    "a8s-test-taint-1",
						Value:  "dummy-val-1",
						Effect: "NoSchedule",
					},
					{
						Key:    "a8s-test-taint-2",
						Value:  "dummy-val-2",
						Effect: "NoSchedule",
					},
				}
				tolerations = []corev1.Toleration{
					{
						Key:      "a8s-test-taint-1",
						Operator: corev1.TolerationOpEqual,
						Value:    "dummy-val-1",
						Effect:   "NoSchedule",
					},
					{
						Key:      "a8s-test-taint-2",
						Operator: corev1.TolerationOpExists,
						Effect:   "NoSchedule",
					},
				}

				Eventually(func() error {
					return nodes.TaintWorkers(ctx, taints)
				}, taintingTimeout).Should(Succeed())
			})

			It("Implements a 1-replica DSI that tolerates the node taints", func() {
				replicas := int32(1)

				By("Accepting the creation of the DSI API object", func() {
					// Create the DSI K8s API object
					instance, err = newDSI(dataservice, testingNamespace,
						framework.GenerateName(instanceNamePrefix, GinkgoParallelProcess(),
							suffixLength), replicas)
					Expect(err).To(BeNil(), "failed to generate DSI object")
					instanceNSN = instance.GetNamespace() + "/" + instance.GetName()
					instance.SetTolerations(tolerations...)
					Expect(k8sClient.Create(ctx, instance.GetClientObject())).To(Succeed(),
						"failed to create DSI "+instanceNSN)
				})

				By("Creating a healthy StatefulSet for the DSI", func() {
					Eventually(func(g Gomega) {
						sset, err := instance.StatefulSet(ctx, k8sClient)
						g.Expect(err).To(BeNil(), "failed to get the DSI "+instanceNSN+
							"'s StatefulSet")
						g.Expect(sset.Status.ReadyReplicas).To(Equal(replicas),
							"ready replicas of DSI "+instanceNSN+
								"'s StatefulSet don't match DSI's desired replicas")
					}, 5*time.Minute).Should(Succeed(),
						"failed to verify that the DSI's StatefulSet gets up and running")
				})

				By("Accepting a ServiceBinding to the DSI", func() {
					sb = servicebinding.New(
						servicebinding.SetNamespacedName(instance.GetClientObject()),
						servicebinding.SetInstanceRef(instance.GetClientObject()),
					)
					Expect(k8sClient.Create(ctx, sb)).To(Succeed(),
						"failed to create ServiceBinding for DSI "+instanceNSN)
				})

				By("Marking the ServiceBinding as implemented", func() {
					Eventually(func(g Gomega) {
						sbNSN := types.NamespacedName{Namespace: sb.Namespace, Name: sb.Name}
						g.Expect(k8sClient.Get(ctx, sbNSN, sb)).
							To(Succeed(), "failed to get DSI "+instanceNSN+"'s ServiceBinding")
						g.Expect(sb.Status.Implemented).
							To(BeTrue(), "ServiceBinding of DSI "+instanceNSN+" isn't implemented")
					}, 10*time.Second).Should(Succeed(),
						"failed to verify that the ServiceBinding of DSI "+instanceNSN+
							" gets implemented")
				})

				By("Generating a client to the DSI to read/write to/from it", func() {
					// Get credentials to write to the DSI
					sbCredentials, err = secret.Data(
						ctx, k8sClient, servicebinding.SecretName(sb.Name), testingNamespace)
					Expect(err).To(BeNil(), "failed to parse secret data for ServiceBinding "+
						sb.GetNamespace()+"/"+sb.GetName())

					// Create a portforwarding to write to the DSI from out of cluster
					portForwardStopCh, localPort, err = framework.PortForward(
						ctx, instancePort, kubeconfigPath, instance, k8sClient)
					Expect(err).To(BeNil(), "failed to establish portforward to DSI "+instanceNSN)

					// Generate a client to the DSI using the credentials and the portforwarding
					instanceClient, err = dsi.NewClient(dataservice,
						strconv.Itoa(localPort),
						sbCredentials)
					Expect(err).To(BeNil(), "failed to create client to DSI "+instanceNSN)
				})

				By("Setting up the DSI so that clients can write/read data into/from it", func() {
					Expect(instanceClient.Write(ctx, appsDefaultDB, "sample data")).To(Succeed(),
						"failed to write into database "+appsDefaultDB+" in DSI "+instanceNSN)
					readData, err := instanceClient.Read(ctx, appsDefaultDB)
					Expect(err).To(BeNil(),
						"failed to read data from database "+appsDefaultDB+" in DSI "+instanceNSN)
					Expect(readData).To(Equal("sample data"),
						"data read from database "+appsDefaultDB+" in DSI "+instanceNSN+
							" doesn't match previously written data")
				})
			})

			It("Implements a 3-replica DSI that tolerates the node taints", func() {
				replicas := int32(3)

				By("Accepting the creation of the DSI API object", func() {
					// Create the DSI K8s API object
					instance, err = newDSI(dataservice, testingNamespace,
						framework.GenerateName(instanceNamePrefix, GinkgoParallelProcess(),
							suffixLength), replicas)
					Expect(err).To(BeNil(), "failed to generate DSI object")
					instanceNSN = instance.GetNamespace() + "/" + instance.GetName()
					instance.SetTolerations(tolerations...)
					Expect(k8sClient.Create(ctx, instance.GetClientObject())).To(Succeed(),
						"failed to create DSI "+instanceNSN)
				})

				By("Creating a healthy StatefulSet for the DSI", func() {
					Eventually(func(g Gomega) {
						sset, err := instance.StatefulSet(ctx, k8sClient)
						g.Expect(err).To(BeNil(), "failed to get the DSI "+instanceNSN+
							"'s StatefulSet")
						g.Expect(sset.Status.ReadyReplicas).To(Equal(replicas),
							"ready replicas of DSI "+instanceNSN+
								"'s StatefulSet don't match DSI's desired replicas")
					}, 5*time.Minute).Should(Succeed(),
						"failed to verify that the DSI's StatefulSet gets up and running")
				})

				By("Accepting a ServiceBinding to the DSI", func() {
					sb = servicebinding.New(
						servicebinding.SetNamespacedName(instance.GetClientObject()),
						servicebinding.SetInstanceRef(instance.GetClientObject()),
					)
					Expect(k8sClient.Create(ctx, sb)).To(Succeed(),
						"failed to create ServiceBinding for DSI "+instanceNSN)
				})

				By("Marking the ServiceBinding as implemented", func() {
					Eventually(func(g Gomega) {
						sbNSN := types.NamespacedName{Namespace: sb.Namespace, Name: sb.Name}
						g.Expect(k8sClient.Get(ctx, sbNSN, sb)).
							To(Succeed(), "failed to get DSI "+instanceNSN+"'s ServiceBinding")
						g.Expect(sb.Status.Implemented).
							To(BeTrue(), "ServiceBinding of DSI "+instanceNSN+" isn't implemented")
					}, 10*time.Second).Should(Succeed(),
						"failed to verify that the ServiceBinding of DSI "+instanceNSN+
							" gets implemented")
				})

				By("Generating a client to the DSI to read/write to/from it", func() {
					// Get credentials to write to the DSI
					sbCredentials, err = secret.Data(
						ctx, k8sClient, servicebinding.SecretName(sb.Name), testingNamespace)
					Expect(err).To(BeNil(), "failed to parse secret data for ServiceBinding "+
						sb.GetNamespace()+"/"+sb.GetName())

					// Create a portforwarding to write to the DSI from out of cluster
					portForwardStopCh, localPort, err = framework.PortForward(
						ctx, instancePort, kubeconfigPath, instance, k8sClient)
					Expect(err).To(BeNil(), "failed to establish portforward to DSI "+instanceNSN)

					// Generate a client to the DSI using the credentials and the portforwarding
					instanceClient, err = dsi.NewClient(dataservice,
						strconv.Itoa(localPort),
						sbCredentials)
					Expect(err).To(BeNil(), "failed to create client to DSI "+instanceNSN)
				})

				By("Setting up the DSI so that clients can write/read data into/from it", func() {
					Expect(instanceClient.Write(ctx, appsDefaultDB, "sample data")).To(Succeed(),
						"failed to write into database "+appsDefaultDB+" in DSI "+instanceNSN)
					readData, err := instanceClient.Read(ctx, appsDefaultDB)
					Expect(err).To(BeNil(),
						"failed to read data from database "+appsDefaultDB+" in DSI "+instanceNSN)
					Expect(readData).To(Equal("sample data"),
						"data read from database "+appsDefaultDB+" in DSI "+instanceNSN+
							" doesn't match previously written data")
				})
			})
		})
	})

	Context("DSI pods have anti-affinity rules to repel each other", func() {
		const (
			a8sTestNodeLabelKey = "a8s-test-host"
			a8sTestAZLabelKey   = "a8s-test-az"
		)

		BeforeEach(func() {
			ctx = context.Background()
		})

		AfterEach(func() {
			Eventually(func() error {
				return nodes.UnlabelAll(ctx, []string{a8sTestNodeLabelKey, a8sTestAZLabelKey})
			}).Should(Succeed())

			close(portForwardStopCh)

			Expect(k8sClient.Delete(ctx, sb)).To(Succeed(),
				"failed to delete DSI "+instanceNSN+"'s ServiceBinding")

			Expect(k8sClient.Delete(ctx, instance.GetClientObject())).To(Succeed(),
				"failed to delete DSI "+instanceNSN)
		})

		Context("3-replica DSI with each replica in a different AZ", func() {
			numAZs := 3

			BeforeEach(func() {
				Eventually(func(g Gomega) {
					var k8sNodes []corev1.Node
					k8sNodes, err = nodes.ListWorkers(ctx)
					g.Expect(err).To(BeNil())

					for i, n := range k8sNodes {
						az := "az-" + strconv.Itoa(i%numAZs)
						labelToAdd := map[string]string{a8sTestAZLabelKey: az}
						g.Expect(nodes.Label(ctx, n, labelToAdd)).To(Succeed())
					}
				}).Should(Succeed(), labelingTimeout)
			})

			It("Implements a 3-replica DSI where each replica is in a different AZ", func() {
				replicas := int32(3)

				By("Accepting the creation of the DSI API object", func() {
					// Create the DSI K8s API object
					instance, err = newDSI(dataservice, testingNamespace,
						framework.GenerateName(instanceNamePrefix, GinkgoParallelProcess(),
							suffixLength), replicas)
					Expect(err).To(BeNil(), "failed to generate DSI object")

					instance.AddRequiredPodAntiAffinityTerm(corev1.PodAffinityTerm{
						TopologyKey: a8sTestAZLabelKey,
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"a8s.a9s/dsi-name": instance.GetName(),
								"a8s.a9s/dsi-kind": instance.
									GetObjectKind().
									GroupVersionKind().
									Kind,
							},
						},
					})

					instanceNSN = instance.GetNamespace() + "/" + instance.GetName()
					Expect(k8sClient.Create(ctx, instance.GetClientObject())).To(Succeed(),
						"failed to create DSI "+instanceNSN)
				})

				By("Getting the DSI up and running", func() {
					dsi.WaitForReadiness(ctx, instance.GetClientObject(), k8sClient)
				})

				By("Placing the Pods on nodes in different AZs", func() {
					pods, err := instance.Pods(ctx, k8sClient)
					Expect(err).To(BeNil())
					Expect(verifyDSIPodsAreInDifferentAZs(ctx, pods, a8sTestAZLabelKey)).
						To(Succeed())
				})

				By("Accepting a ServiceBinding to the DSI", func() {
					sb = servicebinding.New(
						servicebinding.SetNamespacedName(instance.GetClientObject()),
						servicebinding.SetInstanceRef(instance.GetClientObject()),
					)
					Expect(k8sClient.Create(ctx, sb)).To(Succeed(),
						"failed to create ServiceBinding for DSI "+instanceNSN)
				})

				By("Marking the ServiceBinding as implemented", func() {
					Eventually(func(g Gomega) {
						sbNSN := types.NamespacedName{Namespace: sb.Namespace, Name: sb.Name}
						g.Expect(k8sClient.Get(ctx, sbNSN, sb)).
							To(Succeed(), "failed to get DSI "+instanceNSN+"'s ServiceBinding")
						g.Expect(sb.Status.Implemented).
							To(BeTrue(), "ServiceBinding of DSI "+instanceNSN+" isn't implemented")
					}, 10*time.Second).Should(Succeed(),
						"failed to verify that the ServiceBinding of DSI "+instanceNSN+
							" gets implemented")
				})

				By("Generating a client to the DSI to read/write to/from it", func() {
					// Get credentials to write to the DSI
					sbCredentials, err = secret.Data(
						ctx, k8sClient, servicebinding.SecretName(sb.Name), testingNamespace)
					Expect(err).To(BeNil(), "failed to parse secret data for ServiceBinding "+
						sb.GetNamespace()+"/"+sb.GetName())

					// Create a portforwarding to write to the DSI from out of cluster
					portForwardStopCh, localPort, err = framework.PortForward(
						ctx, instancePort, kubeconfigPath, instance, k8sClient)
					Expect(err).To(BeNil(), "failed to establish portforward to DSI "+instanceNSN)

					// Generate a client to the DSI using the credentials and the portforwarding
					instanceClient, err = dsi.NewClient(dataservice,
						strconv.Itoa(localPort),
						sbCredentials)
					Expect(err).To(BeNil(), "failed to create client to DSI "+instanceNSN)
				})

				By("Setting up the DSI so that clients can write/read data into/from it", func() {
					Expect(instanceClient.Write(ctx, appsDefaultDB, "sample data")).To(Succeed(),
						"failed to write into database "+appsDefaultDB+" in DSI "+instanceNSN)
					readData, err := instanceClient.Read(ctx, appsDefaultDB)
					Expect(err).To(BeNil(),
						"failed to read data from database "+appsDefaultDB+" in DSI "+instanceNSN)
					Expect(readData).To(Equal("sample data"),
						"data read from database "+appsDefaultDB+" in DSI "+instanceNSN+
							" doesn't match previously written data")
				})
			})
		})

		Context("3-replica DSI in a cluster with 3 hosts but only 2 AZs", func() {
			numAZs := 2
			numNodes := 3

			BeforeEach(func() {
				Eventually(func(g Gomega) {
					var k8sNodes []corev1.Node
					k8sNodes, err = nodes.ListWorkers(ctx)
					g.Expect(err).To(BeNil())

					for i, n := range k8sNodes {
						az := "az-" + strconv.Itoa(i%numAZs)
						node := "node-" + strconv.Itoa(i%numNodes)
						labelsToAdd := map[string]string{
							a8sTestAZLabelKey:   az,
							a8sTestNodeLabelKey: node,
						}
						g.Expect(nodes.Label(ctx, n, labelsToAdd)).To(Succeed())
					}
				}).Should(Succeed(), labelingTimeout)
			})

			It("Implements a 3-replica DSI with replicas spread across 3 hosts and 2 AZs", func() {
				replicas := int32(3)

				By("Accepting the creation of the DSI API object", func() {
					// Create the DSI K8s API object
					instance, err = newDSI(dataservice, testingNamespace,
						framework.GenerateName(instanceNamePrefix, GinkgoParallelProcess(),
							suffixLength), replicas)
					Expect(err).To(BeNil(), "failed to generate DSI object")

					instance.AddRequiredPodAntiAffinityTerm(corev1.PodAffinityTerm{
						TopologyKey: a8sTestNodeLabelKey,
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"a8s.a9s/dsi-name": instance.GetName(),
								"a8s.a9s/dsi-kind": instance.
									GetObjectKind().
									GroupVersionKind().
									Kind,
							},
						},
					})

					instance.AddPreferredPodAntiAffinityTerm(100, corev1.PodAffinityTerm{
						TopologyKey: a8sTestAZLabelKey,
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"a8s.a9s/dsi-name": instance.GetName(),
								"a8s.a9s/dsi-kind": instance.
									GetObjectKind().
									GroupVersionKind().
									Kind,
							},
						},
					})

					instanceNSN = instance.GetNamespace() + "/" + instance.GetName()
					Expect(k8sClient.Create(ctx, instance.GetClientObject())).To(Succeed(),
						"failed to create DSI "+instanceNSN)
				})

				By("Getting the DSI up and running", func() {
					dsi.WaitForReadiness(ctx, instance.GetClientObject(), k8sClient)
				})

				By("Placing the Pods on different nodes and not in just one AZ", func() {
					pods, err := instance.Pods(ctx, k8sClient)
					Expect(err).To(BeNil())
					Expect(verifyDSIPodsAreOnDifferentNodes(ctx, pods, a8sTestNodeLabelKey)).
						To(Succeed())
					Expect(verifyDSIPodsAreInMoreThanOneAZ(ctx, pods, a8sTestAZLabelKey)).
						To(Succeed())
				})

				By("Accepting a ServiceBinding to the DSI", func() {
					sb = servicebinding.New(
						servicebinding.SetNamespacedName(instance.GetClientObject()),
						servicebinding.SetInstanceRef(instance.GetClientObject()),
					)
					Expect(k8sClient.Create(ctx, sb)).To(Succeed(),
						"failed to create ServiceBinding for DSI "+instanceNSN)
				})

				By("Marking the ServiceBinding as implemented", func() {
					Eventually(func(g Gomega) {
						sbNSN := types.NamespacedName{Namespace: sb.Namespace, Name: sb.Name}
						g.Expect(k8sClient.Get(ctx, sbNSN, sb)).
							To(Succeed(), "failed to get DSI "+instanceNSN+"'s ServiceBinding")
						g.Expect(sb.Status.Implemented).
							To(BeTrue(), "ServiceBinding of DSI "+instanceNSN+" isn't implemented")
					}, 10*time.Second).Should(Succeed(),
						"failed to verify that the ServiceBinding of DSI "+instanceNSN+
							" gets implemented")
				})

				By("Generating a client to the DSI to read/write to/from it", func() {
					// Get credentials to write to the DSI
					sbCredentials, err = secret.Data(
						ctx, k8sClient, servicebinding.SecretName(sb.Name), testingNamespace)
					Expect(err).To(BeNil(), "failed to parse secret data for ServiceBinding "+
						sb.GetNamespace()+"/"+sb.GetName())

					// Create a portforwarding to write to the DSI from out of cluster
					portForwardStopCh, localPort, err = framework.PortForward(
						ctx, instancePort, kubeconfigPath, instance, k8sClient)
					Expect(err).To(BeNil(), "failed to establish portforward to DSI "+instanceNSN)

					// Generate a client to the DSI using the credentials and the portforwarding
					instanceClient, err = dsi.NewClient(dataservice,
						strconv.Itoa(localPort),
						sbCredentials)
					Expect(err).To(BeNil(), "failed to create client to DSI "+instanceNSN)
				})

				By("Setting up the DSI so that clients can write/read data into/from it", func() {
					Expect(instanceClient.Write(ctx, appsDefaultDB, "sample data")).To(Succeed(),
						"failed to write into database "+appsDefaultDB+" in DSI "+instanceNSN)
					readData, err := instanceClient.Read(ctx, appsDefaultDB)
					Expect(err).To(BeNil(),
						"failed to read data from database "+appsDefaultDB+" in DSI "+instanceNSN)
					Expect(readData).To(Equal("sample data"),
						"data read from database "+appsDefaultDB+" in DSI "+instanceNSN+
							" doesn't match previously written data")
				})
			})
		})
	})

})

func verifyDSIPodsAreInDifferentAZs(ctx context.Context,
	dsiPods []corev1.Pod,
	azLabelKey string) error {

	return verifyDSIPodsAreInDifferentTopologyDomains(ctx, dsiPods, "AZ", azLabelKey)
}

func verifyDSIPodsAreOnDifferentNodes(ctx context.Context,
	dsiPods []corev1.Pod,
	nodeLabelKey string) error {

	return verifyDSIPodsAreInDifferentTopologyDomains(ctx, dsiPods, "node", nodeLabelKey)
}

func verifyDSIPodsAreInDifferentTopologyDomains(ctx context.Context,
	dsiPods []corev1.Pod,
	topologyDomainName, topologyDomainKey string) error {

	domainToPod := map[string]string{}
	for _, p := range dsiPods {
		podNodeLabels, err := nodes.GetLabels(ctx, p.Spec.NodeName)
		if err != nil {
			return fmt.Errorf("failed to verify that all dsi pods are in different %ss: %w",
				topologyDomainName, err)
		}

		podDomain := podNodeLabels[topologyDomainKey]
		if otherPodInDomain := domainToPod[podDomain]; otherPodInDomain != "" {
			return fmt.Errorf("all dsi pods should be in different %ss but pods %s and %s are "+
				"both in %s %s", topologyDomainName, p.Name, otherPodInDomain, topologyDomainName,
				podDomain)
		}
		domainToPod[podDomain] = p.Name
	}

	return nil
}

func verifyDSIPodsAreInMoreThanOneAZ(ctx context.Context,
	dsiPods []corev1.Pod,
	azLabelKey string) error {

	seenAZs := map[string]struct{}{}
	podAZ := ""
	for _, p := range dsiPods {
		podNodeLabels, err := nodes.GetLabels(ctx, p.Spec.NodeName)
		if err != nil {
			return fmt.Errorf("failed to verify that dsi pods are in more than one AZ: %w", err)
		}

		podAZ = podNodeLabels[azLabelKey]
		seenAZs[podAZ] = struct{}{}
		if len(seenAZs) > 1 {
			return nil
		}
	}

	return fmt.Errorf("all dsi pods are in AZ %s when they should be in more than one AZ", podAZ)
}
