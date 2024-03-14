package postgresql

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	pgv1beta3 "github.com/anynines/postgresql-operator/api/v1beta3"

	"github.com/anynines/a8s-deployment/test/framework"
	"github.com/anynines/a8s-deployment/test/framework/dsi"
	"github.com/anynines/a8s-deployment/test/framework/secret"
	"github.com/anynines/a8s-deployment/test/framework/servicebinding"
)

const (
	SSLModeRequired = "require"
)

var _ = Describe("end-to-end tests for exposed instances", Label("ExternalLoadbalancer", "KindIncompatible"), func() {

	Context("Instance exposed via Load Balancer", Ordered, func() {
		AfterAll(func() {
			Expect(k8sClient.Delete(ctx, instance.GetClientObject())).To(
				Succeed(), "failed to delete instance")
		})

		It("Provisions the exposed PostgreSQL instance", func() {
			By("Accepting instance creation")
			instance, err = dsi.New(
				dataservice,
				testingNamespace,
				framework.GenerateName(instanceNamePrefix,
					GinkgoParallelProcess(), suffixLength),
				3,
			)
			Expect(err).To(BeNil(), "failed to generate new DSI resource")

			// Cast interface to concrete struct so that we can access fields
			// directly
			pg, ok = instance.GetClientObject().(*pgv1beta3.Postgresql)
			Expect(ok).To(BeTrue(),
				"failed to cast object interface to PostgreSQL struct")

			// We break the framework abstraction here to access a feature
			// that is available in the pg data service. If we make access from outside the cluster a
			// feature throughout the framework, we should adjust our testing framework instead
			pg.Spec.Expose = "LoadBalancer"
			pg.Spec.EnableReadOnlyService = true

			Expect(k8sClient.Create(ctx, instance.GetClientObject())).
				To(Succeed(), fmt.Sprintf("failed to create instance %s/%s",
					instance.GetNamespace(), instance.GetName()))

			By("Setting the DSI status to Running")

			dsi.WaitForReadiness(ctx, instance.GetClientObject(), k8sClient)
		})

		var serviceBindingData secret.SecretData
		It("supports service bindings", func() {
			By("accepting service binding creation")
			// Create service binding for instance and get secret data
			sb = servicebinding.New(
				servicebinding.SetNamespacedName(instance.GetClientObject()),
				servicebinding.SetInstanceRef(instance.GetClientObject()),
			)
			Expect(k8sClient.Create(ctx, sb)).To(Succeed(),
				fmt.Sprintf("failed to create new servicebinding for DSI %s/%s",
					instance.GetNamespace(), instance.GetName()))

			By("Setting the service binding status to ready")
			servicebinding.WaitForReadiness(ctx, sb, k8sClient)

			By("Creating a service binding secret")
			serviceBindingData, err = secret.Data(
				ctx, k8sClient, servicebinding.SecretName(sb.Name), testingNamespace)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to parse secret data for service binding %s/%s",
					sb.GetNamespace(), sb.GetName()))

		})

		var connInfo v1.ConfigMap
		It("Creates a connection ConfigMap", func() {
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Namespace: instance.GetNamespace(),
				Name:      dsi.ConnectionInfoName(instance.GetName()),
			}, &connInfo)).To(Succeed())

			Expect(connInfo.Data).To(HaveKey("primary"),
				"connInfo does not contain information about primary database")

			Expect(connInfo.Data).To(HaveKey("port"),
				"connInfo does not contain information about port")

			Expect(connInfo.Data).To(HaveKey("readOnly"),
				"connInfo does not contain information about read only service")

		})

		It("can write data via public address", func() {
			By("allowing client creation")
			// Create client for interacting with the new instance.
			client, err = dsi.NewClientForURL(
				dataservice, connInfo.Data["primary"], connInfo.Data["port"],
				SSLModeRequired, // use ssl, as unencrypted connections from outside the cluster
				// are not allowed
				serviceBindingData)
			Expect(err).To(BeNil(), "failed to create new dsi client")

			By("accepting writes")
			// we need retry logic, as could provider load balancers may take some time
			// to become available from the internet
			Eventually(func() error {
				return client.Write(ctx, entity, testInput)
			}, 5*time.Minute).Should(Succeed())
		})

		It("can read data via public address", func() {
			By("allowing reading from primary service")
			// Create client for interacting with the new instance.
			client, err = dsi.NewClientForURL(
				dataservice, connInfo.Data["primary"], connInfo.Data["port"],
				SSLModeRequired, // use ssl, as unencrypted connections from outside the cluster
				// are not allowed
				serviceBindingData)
			Expect(err).To(BeNil(), "failed to create new dsi client")

			By("accepting reads")
			// we need retry logic, as cloud provider load balancers may take some time
			// to become available from the internet
			Eventually(func(g Gomega) {
				entityData, err := client.Read(ctx, entity)
				g.Expect(err).To(BeNil(), "error reading from the instances primary service")
				g.Expect(entityData).To(Equal(testInput), "data service returned unexpected entry")
			}, 5*time.Minute).Should(Succeed())

			By("allowing reading from read-only service")

			// Create client for interacting with the new instance.
			client, err = dsi.NewClientForURL(
				dataservice, connInfo.Data["readOnly"], connInfo.Data["port"],
				"require", // use ssl, as unencrypted connections from outside the cluster
				// are not allowed
				serviceBindingData)
			Expect(err).To(BeNil(), "failed to create new dsi client")

			By("accepting reads via the read-only service")
			// we need retry logic, as cloud provider load balancers may take some time
			// to become available from the internet
			Eventually(func(g Gomega) {
				entityData, err := client.Read(ctx, entity)
				g.Expect(err).To(BeNil(), "error reading from the instances read-only service")
				g.Expect(entityData).To(Equal(testInput), "data service returned unexpected entry")
			}, 5*time.Minute).Should(Succeed())
		})
	})
})
