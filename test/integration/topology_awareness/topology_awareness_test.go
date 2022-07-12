package topology_awareness

import (
	"context"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/anynines/a8s-deployment/test/integration/framework"
	"github.com/anynines/a8s-deployment/test/integration/framework/dsi"
	"github.com/anynines/a8s-deployment/test/integration/framework/secret"
	"github.com/anynines/a8s-deployment/test/integration/framework/servicebinding"
	sbv1alpha1 "github.com/anynines/a8s-service-binding-controller/api/v1alpha1"
)

// TODO: Test broken cases where the DSI has no tolerations: 1 taint - 0 tolerations;
// 	     2 taints - 1 toleration; 1 taint - 1 toleration that doesn't match it.
// TODO: Test removing tolerations from an existing DSI.
// TODO: Test adding tolerations to an existing DSI.
// TODO: Test horizontal scale up.
// TODO: Test cases where only a subset of nodes is tainted.

const (
	suffixLength = 5

	appsDefaultDB = "a9s_apps_default_db"

	instancePort = 5432
)

var _ = Describe("DSI tolerations to K8s nodes taints", func() {
	Context("DSI has tolerations to node taints", func() {
		var (
			err error

			instance       Object
			instanceNSN    string
			instanceClient dsi.DSIClient

			sb            *sbv1alpha1.ServiceBinding
			sbCredentials secret.SecretData

			portForwardStopCh chan struct{}
			localPort         int

			taints      []corev1.Taint
			tolerations []corev1.Toleration
		)

		BeforeEach(func() {
			ctx = context.Background()
		})

		AfterEach(func() {
			close(portForwardStopCh)

			Expect(nodes.UntaintAll(ctx, taints)).To(Succeed(), "failed to untaint nodes")

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

				Expect(nodes.TaintAll(ctx, taints)).To(Succeed(), "failed to taint nodes")
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

				Expect(nodes.TaintAll(ctx, taints)).To(Succeed(), "failed to taint nodes")
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
})
