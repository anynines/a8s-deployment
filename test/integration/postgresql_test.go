package integration

import (
	//"log"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// THIS WILL BE GENERALIZED DURING REFACTORING
var _ = Describe("PostgreSQL integration tests", func() {
		BeforeEach(func() {
			serviceInstance = newDSI(PostgresNamespace, PostgresName, replicas)
			Expect(k8sClient.Create(ctx, serviceInstance)).Should(Succeed())
			waitForClusterCreation(ctx, serviceInstance, k8sClient)
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, serviceInstance)).Should(Succeed())
			waitForClusterDeletion(ctx, serviceInstance, k8sClient)
		})

})
