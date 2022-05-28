package postgresql

import (
	"fmt"
	"log"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	"github.com/anynines/a8s-deployment/test/integration/framework"
	"github.com/anynines/a8s-deployment/test/integration/framework/dsi"
	"github.com/anynines/a8s-deployment/test/integration/framework/postgresql"
	"github.com/anynines/a8s-deployment/test/integration/framework/secret"
	"github.com/anynines/a8s-deployment/test/integration/framework/servicebinding"
	sbv1alpha1 "github.com/anynines/a8s-service-binding-controller/api/v1alpha1"
	pgv1alpha1 "github.com/anynines/postgresql-operator/api/v1alpha1"
)

const (
	instancePort = 5432
	replicas     = 1
	suffixLength = 5

	roleKey            = "spilo-role"
	primaryRoleValue   = "master"
	secondaryRoleValue = "replica"
	databaseKey        = "database"
	DbAdminUsernameKey = "username"
	DbAdminPasswordKey = "password"

	// TODO: Make configurable and generalizable using Data interface
	// testInput is data input used for testing data service functionality.
	testInput = "test_input"
	// entity is a generic term to describe where data services store their data.
	entity = "test_entity"
	// asyncOpsTimeoutMins...
	asyncOpsTimeoutMins = time.Minute * 5
)

var (
	// portForwardStopCh is the channel used to manage the lifecycle of a port forward.
	portForwardStopCh chan struct{}
	localPort         int
	ok                bool

	sb       *sbv1alpha1.ServiceBinding
	instance dsi.Object
	client   dsi.DSIClient
	pg       *pgv1alpha1.Postgresql
)

var _ = Describe("PostgreSQL Operator integration tests", func() {
	Context("PostgreSQL Instance Creation", func() {
		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, instance.GetClientObject())).To(
				Succeed(), "failed to delete instance")
		})

		It("Provisions the PostgreSQL instance", func() {
			By("creating a dataservice instance", func() {
				instance, err = dsi.New(
					dataservice,
					testingNamespace,
					framework.GenerateName(instanceNamePrefix,
						GinkgoParallelProcess(), suffixLength),
					replicas,
				)
				Expect(err).To(BeNil(), "failed to generate new DSI resource")

				// Cast interface to concrete struct so that we can access fields
				// directly
				pg, ok = instance.GetClientObject().(*pgv1alpha1.Postgresql)
				Expect(ok).To(BeTrue(),
					"failed to cast object interface to PostgreSQL struct")

				Expect(k8sClient.Create(ctx, instance.GetClientObject())).
					To(Succeed(), fmt.Sprintf("failed to create instance %s/%s",
						instance.GetNamespace(), instance.GetName()))
				dsi.WaitForReadiness(ctx, instance.GetClientObject(), k8sClient)
			})

			By("creating a StatefulSet", func() {
				sts := &appsv1.StatefulSet{}
				Expect(k8sClient.Get(ctx,
					types.NamespacedName{Name: instance.GetName(),
						Namespace: instance.GetNamespace()},
					sts)).To(Succeed(), "failed to get statefulset")

				Expect(*sts.Spec.Replicas).To(Equal(*pg.Spec.Replicas))
				Expect(sts.Status.ReadyReplicas).To(Equal(*pg.Spec.Replicas))

				// Labels and Annotations are tested since other a8s framework
				// components rely on them.
				Expect(sts.Spec.Template.Labels).
					To(HaveKeyWithValue("application", "spilo"))
				Expect(sts.Spec.Template.Labels).
					To(HaveKeyWithValue("cluster-name", pg.Name))
				Expect(sts.Spec.Template.Labels).
					To(HaveKeyWithValue("dsi-group", "postgresql.anynines.com"))
				Expect(sts.Spec.Template.Labels).
					To(HaveKeyWithValue("dsi-kind", "Postgresql"))

				Expect(sts.Spec.Template.Annotations).
					To(HaveKeyWithValue("prometheus.io/port", "9187"))
				Expect(sts.Spec.Template.Annotations).
					To(HaveKeyWithValue("prometheus.io/scrape", "true"))

				Expect(sts.Spec.Template.Spec.Containers[0].Name).
					To(Equal("postgres"))
				Expect(sts.Spec.Template.Spec.Containers[1].Name).
					To(Equal("backup-agent"))

				Expect(sts.Spec.Template.Spec.ServiceAccountName).
					To(Equal(pg.Name))
			})

			By("creating a Service that points to the primary for writes", func() {
				svc := &corev1.Service{}
				Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Name: postgresql.MasterService(
							instance.GetName()),
						Namespace: instance.GetNamespace()},
					svc)).To(Succeed())

				Expect(svc.Spec.Selector).
					To(HaveKeyWithValue("cluster-name", instance.GetName()))
				Expect(svc.Spec.Selector).
					To(HaveKeyWithValue("spilo-role", "master"))

				Expect(svc.Spec.Type).To(Equal(corev1.ServiceTypeClusterIP))
				Expect(svc.Spec.Ports[0].Name).To(Equal("postgresql"))
				Expect(svc.Spec.Ports[0].Port).To(Equal(int32(5432)))
				Expect(svc.Spec.Ports[0].Protocol).To(Equal(corev1.ProtocolTCP))
			})

			By("creating the ServiceAccount", func() {
				Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{Name: instance.GetName(),
						Namespace: instance.GetNamespace()},
					&corev1.ServiceAccount{},
				)).To(Succeed(), "failed to get serviceaccount")
			})

			By("creating a RoleBinding between the PostgreSQL instance service account and the Spilo role", func() {
				rolebinding := &rbacv1.RoleBinding{}
				Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{Name: instance.GetName(),
						Namespace: instance.GetNamespace()},
					rolebinding,
				)).To(Succeed(), "failed to get rolebinding")

				Expect(rolebinding.RoleRef.Name).To(Equal("postgresql-spilo-role"))
				Expect(rolebinding.RoleRef.Kind).To(Equal("ClusterRole"))
				Expect(rolebinding.RoleRef.APIGroup).To(Equal(rbacv1.GroupName))

				Expect(rolebinding.Subjects[0].Name).To(Equal(instance.GetName()))
				Expect(rolebinding.Subjects[0].Kind).To(Equal("ServiceAccount"))
				Expect(rolebinding.Subjects[0].APIGroup).To(Equal(corev1.GroupName))
			})

			By("creating a Secret with the credentials of the admin role", func() {
				adminRoleSecret := &corev1.Secret{}
				Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Name:      postgresql.AdminRoleSecretName(instance.GetName()),
						Namespace: instance.GetNamespace()},
					adminRoleSecret,
				)).To(Succeed(), "failed to get admin role secret")

				Expect(adminRoleSecret.Data).To(HaveKey("password"))
				Expect(adminRoleSecret.Data["password"]).NotTo(BeEmpty())

				Expect(adminRoleSecret.Data).To(HaveKey("username"))
				Expect(adminRoleSecret.Data["username"]).NotTo(BeEmpty())

				Expect(adminRoleSecret.Labels).
					To(HaveKeyWithValue("application", "spilo"))
				Expect(adminRoleSecret.Labels).
					To(HaveKeyWithValue("cluster-name", instance.GetName()))
			})

			By("creating a Secret with the credentials of the Standby role for streaming replication", func() {
				standbyRoleSecret := &corev1.Secret{}
				Expect(k8sClient.Get(
					ctx,
					types.NamespacedName{
						Name: postgresql.StandbyRoleSecretName(
							instance.GetName()),
						Namespace: instance.GetNamespace()},
					standbyRoleSecret,
				)).To(Succeed(), "failed to get standby role secret")

				Expect(standbyRoleSecret.Data).To(HaveKey("password"))
				Expect(standbyRoleSecret.Data["password"]).NotTo(BeEmpty())

				Expect(standbyRoleSecret.Data).To(HaveKey("username"))
				Expect(standbyRoleSecret.Data["username"]).NotTo(BeEmpty())

				Expect(standbyRoleSecret.Labels).
					To(HaveKeyWithValue("application", "spilo"))
				Expect(standbyRoleSecret.Labels).
					To(HaveKeyWithValue("cluster-name", instance.GetName()))
			})

			By("creating PersistentVolumeClaims for each of the replicas", func() {
				for i := 0; i < int(*pg.Spec.Replicas); i++ {
					pvc := &corev1.PersistentVolumeClaim{}
					Expect(k8sClient.Get(
						ctx,
						types.NamespacedName{
							Name: postgresql.PvcName(
								instance.GetName(), i),
							Namespace: instance.GetNamespace()}, pvc,
					)).To(Succeed(), "failed to get pvc")

					Expect(pvc.Status.Phase).To(Equal(corev1.ClaimBound))
				}
			})
		})
	})

	Context("PostgreSQL API Object spec can be updated", func() {
		BeforeEach(func() {
			instance, err = dsi.New(
				dataservice,
				testingNamespace,
				framework.GenerateName(
					instanceNamePrefix, GinkgoParallelProcess(), suffixLength),
				replicas,
			)
			Expect(err).To(BeNil(), "failed to generate new DSI resource")
			Expect(k8sClient.Create(ctx, instance.GetClientObject())).
				To(Succeed(), fmt.Sprintf("failed to create instance %s/%s",
					instance.GetNamespace(), instance.GetName()))
			dsi.WaitForReadiness(ctx, instance.GetClientObject(), k8sClient)
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, instance.GetClientObject())).To(
				Succeed(), "failed to delete instance")
		})

		It("Updates cpu and memory requirements and limits", func() {
			var old pgv1alpha1.Postgresql
			err := k8sClient.Get(ctx, types.NamespacedName{
				Namespace: instance.GetNamespace(),
				Name:      instance.GetName(),
			},
				&old,
			)
			Expect(err).To(BeNil(), "failed to fetch instance resource")

			old.Spec.Resources = &corev1.ResourceRequirements{
				Limits: map[corev1.ResourceName]k8sresource.Quantity{
					corev1.ResourceCPU:    k8sresource.MustParse("200m"),
					corev1.ResourceMemory: k8sresource.MustParse("200Mi"),
				},
				Requests: map[corev1.ResourceName]k8sresource.Quantity{
					corev1.ResourceCPU:    k8sresource.MustParse("200m"),
					corev1.ResourceMemory: k8sresource.MustParse("200Mi"),
				},
			}
			Expect(k8sClient.Update(ctx, &old)).To(Succeed())

			Eventually(func() *corev1.ResourceRequirements {
				sts := &appsv1.StatefulSet{}
				err = k8sClient.Get(
					ctx,
					types.NamespacedName{Name: instance.GetName(),
						Namespace: instance.GetNamespace()},
					sts,
				)
				if err != nil {
					return nil
				}
				return &sts.Spec.Template.Spec.Containers[0].Resources
			}, asyncOpsTimeoutMins, 1*time.Second).Should(Equal(old.Spec.Resources))
		})

		It("Updates replicas", func() {
			var old pgv1alpha1.Postgresql
			err := k8sClient.Get(ctx, types.NamespacedName{
				Namespace: instance.GetNamespace(),
				Name:      instance.GetName(),
			},
				&old,
			)
			Expect(err).To(BeNil(), "failed to fetch instance resource")

			old.Spec.Replicas = pointer.Int32(3)
			Expect(k8sClient.Update(ctx, &old)).To(Succeed())

			Eventually(func() *int32 {
				sts := &appsv1.StatefulSet{}
				err = k8sClient.Get(
					ctx,
					types.NamespacedName{Name: instance.GetName(),
						Namespace: instance.GetNamespace()},
					sts,
				)
				if err != nil {
					return nil
				}
				return sts.Spec.Replicas
			}, asyncOpsTimeoutMins, 1*time.Second).Should(Equal(pointer.Int32(3)))
		})
	})

	Context("PostgreSQL Instance deletion", func() {
		BeforeEach(func() {
			// Create Dataservice instance and wait for instance readiness
			instance, err = dsi.New(
				dataservice,
				testingNamespace,
				framework.GenerateName(
					instanceNamePrefix, GinkgoParallelProcess(), suffixLength),
				replicas,
			)
			Expect(err).To(BeNil(), "failed to generate new DSI resource")

			pg, ok = instance.GetClientObject().(*pgv1alpha1.Postgresql)
			Expect(ok).To(BeTrue(),
				"failed to cast instance object interface to PostgreSQL struct")

			Expect(k8sClient.Create(ctx, instance.GetClientObject())).
				To(Succeed(), fmt.Sprintf("failed to create instance %s/%s",
					instance.GetNamespace(), instance.GetName()))
			dsi.WaitForReadiness(ctx, instance.GetClientObject(), k8sClient)
		})

		It("Deprovisions the PostgreSQL instance", func() {
			By("deleting the PostgreSQL API object", func() {
				Expect(k8sClient.Delete(ctx, instance.GetClientObject())).
					To(Succeed(), "failed to delete PostgreSQL instance")
				dsi.WaitForDeletion(ctx, instance.GetClientObject(), k8sClient)
			})

			By("removing the StatefulSet", func() {
				Eventually(func() bool {
					sts := &appsv1.StatefulSet{}
					err := k8sClient.Get(ctx,
						types.NamespacedName{Name: instance.GetName(),
							Namespace: instance.GetNamespace()},
						sts)
					return err != nil && k8serrors.IsNotFound(err)
				}, asyncOpsTimeoutMins).Should(BeTrue())
			})

			By("removing the service that points to the primary for writes", func() {
				Eventually(func() bool {
					err := k8sClient.Get(ctx,
						types.NamespacedName{
							Name:      postgresql.MasterService(instance.GetName()),
							Namespace: instance.GetNamespace()},
						&corev1.Service{})
					return err != nil && k8serrors.IsNotFound(err)
				}, asyncOpsTimeoutMins).Should(BeTrue())
			})

			By("removing the RoleBinding between the PostgreSQL instance service account and the Spilo role", func() {
				Eventually(func() bool {
					err := k8sClient.Get(
						ctx,
						types.NamespacedName{Name: instance.GetName(),
							Namespace: instance.GetNamespace()},
						&rbacv1.RoleBinding{},
					)
					return err != nil && k8serrors.IsNotFound(err)
				}, asyncOpsTimeoutMins).Should(BeTrue())
			})

			By("removing the ServiceAccount", func() {
				Eventually(func() bool {
					err := k8sClient.Get(
						ctx,
						types.NamespacedName{Name: instance.GetName(),
							Namespace: instance.GetNamespace()},
						&corev1.ServiceAccount{})
					return err != nil && k8serrors.IsNotFound(err)
				}, asyncOpsTimeoutMins).Should(BeTrue())
			})

			By("removing the Secret with the credentials of the admin role", func() {
				Eventually(func() bool {
					err := k8sClient.Get(
						ctx,
						types.NamespacedName{
							Name: postgresql.AdminRoleSecretName(
								instance.GetName()),
							Namespace: instance.GetNamespace()},
						&corev1.Secret{},
					)
					return err != nil && k8serrors.IsNotFound(err)
				}, asyncOpsTimeoutMins).Should(BeTrue())
			})

			By("removing the Secret with the credentials of the Standby role for streaming replication", func() {
				Eventually(func() bool {
					err := k8sClient.Get(
						ctx,
						types.NamespacedName{
							Name: postgresql.StandbyRoleSecretName(
								instance.GetName()),
							Namespace: instance.GetNamespace()},
						&corev1.Secret{},
					)
					return err != nil && k8serrors.IsNotFound(err)
				}, asyncOpsTimeoutMins).Should(BeTrue())
			})

			By("removing the PersistentVolumeClaims of the replicas", func() {
				Eventually(func() bool {
					for i := 0; i < int(*pg.Spec.Replicas); i++ {
						err := k8sClient.Get(
							ctx,
							types.NamespacedName{
								Name: postgresql.PvcName(
									instance.GetName(), i),
								Namespace: instance.GetNamespace()},
							&corev1.PersistentVolumeClaim{},
						)
						if err == nil || !k8serrors.IsNotFound(err) {
							return false
						}
					}
					return true
				}, asyncOpsTimeoutMins).Should(BeTrue())
			})
		})
	})

	Context("PostgreSQL database operations", func() {
		var serviceBindingData secret.SecretData
		BeforeEach(func() {
			// Create Dataservice instance and wait for instance readiness
			singleReplica := int32(1)
			instance, err = dsi.New(
				dataservice,
				testingNamespace,
				framework.GenerateName(
					instanceNamePrefix, GinkgoParallelProcess(), suffixLength),
				singleReplica,
			)
			Expect(err).To(BeNil(), "failed to generate new DSI resource")

			Expect(k8sClient.Create(ctx, instance.GetClientObject())).
				To(Succeed(), fmt.Sprintf("failed to create instance %s/%s",
					instance.GetNamespace(), instance.GetName()))
			dsi.WaitForReadiness(ctx, instance.GetClientObject(), k8sClient)

			// Portforward to access instance from outside cluster.
			portForwardStopCh, localPort, err = framework.PortForward(
				ctx, instancePort, kubeconfigPath, instance, k8sClient)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to establish portforward to DSI %s/%s",
					instance.GetNamespace(), instance.GetName()))

			// Create service binding for instance and get secret data
			sb = servicebinding.New(
				servicebinding.SetNamespacedName(instance.GetClientObject()),
				servicebinding.SetInstanceRef(instance.GetClientObject()),
			)
			Expect(k8sClient.Create(ctx, sb)).To(
				Succeed(),
				fmt.Sprintf("failed to create new servicebinding for DSI %s/%s",
					instance.GetNamespace(), instance.GetName()))
			servicebinding.WaitForReadiness(ctx, sb, k8sClient)
			serviceBindingData, err = secret.Data(
				ctx, k8sClient, servicebinding.SecretName(sb.Name), testingNamespace)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to parse secret data for service binding %s/%s",
					sb.GetNamespace(), sb.GetName()))

			// Create client for interacting with the new instance.
			client, err = dsi.NewClient(
				dataservice, strconv.Itoa(localPort), serviceBindingData)
			Expect(err).To(BeNil(), "failed to create new dsi client")
		})

		AfterEach(func() {
			defer func() { close(portForwardStopCh) }()
			Expect(k8sClient.Delete(ctx, instance.GetClientObject())).To(Succeed(),
				fmt.Sprintf("failed to delete instance %s/%s",
					instance.GetNamespace(), instance.GetName()))
			Expect(k8sClient.Delete(ctx, sb)).To(Succeed(),
				fmt.Sprintf("failed to delete service binding %s/%s",
					sb.GetNamespace(), sb.GetName()))
			dsi.WaitForDeletion(ctx, instance.GetClientObject(), k8sClient)
		})

		It("Data can be written to and read from database even after primary pod deletion", func() {
			var readData string
			By("writing data", func() {
				Expect(client.Write(ctx, entity, testInput)).To(
					BeNil(), fmt.Sprintln("failed to insert data"))
			})

			By("ensuring data was written successfully", func() {
				readData, err = client.Read(ctx, entity)
				Expect(err).To(BeNil(), "failed to read data")
				Expect(readData).To(Equal(testInput), "read data does not match test input")
			})

			By("testing whether data persists after primary pod deletion", func() {
				// Fetch and delete the primary pod
				pod, err := framework.GetPrimaryPodUsingServiceSelector(
					ctx, instance.GetClientObject(), k8sClient)
				Expect(err).To(BeNil(), fmt.Sprintf(
					"failed to get primary pod using service selector for %s/%s",
					instance.GetNamespace(), instance.GetName()))
				Expect(k8sClient.Delete(ctx, pod)).
					To(Succeed(), fmt.Sprintf("failed to delete pod %s/%s",
						pod.GetNamespace(), pod.GetName()))
				dsi.WaitForPodDeletion(ctx, pod, k8sClient)

				// Portforward to access new primary pod from outside cluster.
				portForwardStopCh, localPort, err = framework.PortForward(
					ctx, instancePort, kubeconfigPath,
					instance, k8sClient)
				Expect(err).To(BeNil(),
					fmt.Sprintf("failed to establish portforward to DSI %s/%s",
						instance.GetNamespace(), instance.GetName()))

				// Create client for interacting with the new PostgreSQL primary
				// node
				client, err = dsi.NewClient(dataservice,
					strconv.Itoa(localPort), serviceBindingData)
				Expect(err).To(BeNil(), "failed to create new dsi client")

				// Ensure that newly read data matches our original test input
				readData, err = client.Read(ctx, entity)
				Expect(err).To(BeNil(), "failed to read data")
				Expect(readData).To(Equal(testInput), "read data does not match test input")
			})
		})

		It("The default database and non-login role exist as required by service bindings", func() {
			By("Creating a admin client", func() {
				adminSecretData, err := secret.AdminSecretData(ctx,
					k8sClient,
					instance.GetName(),
					instance.GetNamespace())
				Expect(err).To(BeNil(),
					fmt.Sprintf("failed to parse secret data of admin credentials for %s/%s",
						instance.GetNamespace(), instance.GetName()))

				client, err = dsi.NewClient(dataservice,
					strconv.Itoa(localPort), adminSecretData)
				Expect(err).To(BeNil(), "failed to create new dsi client")

			})

			By("ensuring that the default database exists", func() {
				collection := serviceBindingData[databaseKey]
				Expect(client.CollectionExists(ctx, collection)).To(BeTrue(),
					fmt.Sprintf("failed to find existing colletion %s",
						collection))
			})

			By("ensuring that the non-login user role exists", func() {
				user := serviceBindingData[DbAdminUsernameKey]
				Expect(client.UserExists(ctx, user)).To(BeTrue(),
					fmt.Sprintf("failed to find existing user %s", user))
			})
		})
	})

	Context("PostgreSQL high availability", func() {
		var serviceBindingData secret.SecretData
		BeforeEach(func() {
			// Create high availability instance and wait for instance readiness
			haReplicas := int32(3)
			instance, err = dsi.New(
				dataservice,
				testingNamespace,
				framework.GenerateName(
					instanceNamePrefix, GinkgoParallelProcess(), suffixLength),
				haReplicas,
			)
			Expect(err).To(BeNil(), "failed to generate new DSI resource")
			Expect(k8sClient.Create(ctx, instance.GetClientObject())).
				To(Succeed(), fmt.Sprintf("failed to create instance %s/%s",
					instance.GetNamespace(), instance.GetName()))
			dsi.WaitForReadiness(ctx, instance.GetClientObject(), k8sClient)

			// Portforward to access instance from outside cluster.
			portForwardStopCh, localPort, err = framework.PortForward(
				ctx, instancePort, kubeconfigPath, instance, k8sClient)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to establish portforward to DSI %s/%s",
					instance.GetNamespace(), instance.GetName()))

			// Create service binding for instance and fetch secret data
			sb = servicebinding.New(
				servicebinding.SetNamespacedName(instance.GetClientObject()),
				servicebinding.SetInstanceRef(instance.GetClientObject()),
			)
			Expect(k8sClient.Create(ctx, sb)).To(Succeed(),
				fmt.Sprintf("failed to create new servicebinding for DSI %s/%s",
					instance.GetNamespace(), instance.GetName()))
			servicebinding.WaitForReadiness(ctx, sb, k8sClient)
			serviceBindingData, err = secret.Data(
				ctx, k8sClient, servicebinding.SecretName(sb.Name), testingNamespace)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to parse secret data for service binding %s/%s",
					sb.GetNamespace(), sb.GetName()))

			// Create client for interacting with the new instance.
			client, err = dsi.NewClient(
				dataservice, strconv.Itoa(localPort), serviceBindingData)
			Expect(err).To(BeNil(), "failed to create new dsi client")
		})

		AfterEach(func() {
			defer func() { close(portForwardStopCh) }()
			Expect(k8sClient.Delete(ctx, instance.GetClientObject())).To(Succeed(),
				fmt.Sprintf("failed to delete instance %s/%s",
					instance.GetNamespace(), instance.GetName()))
			Expect(k8sClient.Delete(ctx, sb)).To(Succeed(),
				fmt.Sprintf("failed to delete service binding %s/%s",
					sb.GetNamespace(), sb.GetName()))
			dsi.WaitForDeletion(ctx, instance.GetClientObject(), k8sClient)
		})

		It("Failover occurs when primary pod is gone without data loss", func() {
			pod := &corev1.Pod{}
			var readData string
			By("checking if we have a primary pod", func() {
				pod, err = framework.GetPrimaryPodUsingServiceSelector(
					ctx, instance, k8sClient)
				Expect(err).To(BeNil())
				Expect(pod.Labels[roleKey]).To(Equal(primaryRoleValue))
			})

			By("inserting data", func() {
				Expect(client.Write(ctx, entity, testInput)).To(
					BeNil(), fmt.Sprintln("failed to insert data"))
			})

			By("ensuring that the data exists", func() {
				readData, err = client.Read(ctx, entity)
				Expect(err).To(BeNil(), "failed to read data")
				Expect(readData).To(Equal(testInput), "read data does not match test input")
			})

			By("deleting the primary pod to prompt a fail over", func() {
				Expect(k8sClient.Delete(ctx, pod)).To(Succeed(),
					fmt.Sprintf("failed to delete pod %s/%s",
						pod.GetNamespace(), pod.GetName()))
				dsi.WaitForPodDeletion(ctx, pod, k8sClient)
			})

			By("checking that we a new pod that assumes the primary role", func() {
				newPod, err := framework.GetPrimaryPodUsingServiceSelector(
					ctx, instance, k8sClient)
				Expect(err).To(BeNil())
				Expect(newPod.Labels[roleKey]).To(Equal(primaryRoleValue))
				Expect(newPod.GetUID()).ToNot(Equal(pod.GetUID()),
					"pod UIDs should not be equal after fail over")
				// Checking that the new pod and the deleted pod have different
				// names behaves non-deterministically which is likely a result of
				// how Patroni manages leader election. Therefore, the assertion
				// that the new leader must be an old follower is not true in a
				// subset of cases since leader election can be slower than deletion
				// and readiness of the new pod.
				if pod.GetName() == newPod.GetName() {
					log.Println("The new leader pod name is the same after failover:",
						pod.GetName())
				}
			})

			By("ensuring that the data was replicated to the new primary", func() {
				// Portforward to access new primary pod from outside cluster.
				portForwardStopCh, localPort, err = framework.PortForward(
					ctx, instancePort, kubeconfigPath, instance, k8sClient)
				Expect(err).To(BeNil(),
					fmt.Sprintf("failed to establish portforward to DSI %s/%s",
						instance.GetNamespace(), instance.GetName()))

				// Create client for interacting with the new instance.
				client, err = dsi.NewClient(
					dataservice, strconv.Itoa(localPort), serviceBindingData)
				Expect(err).To(BeNil(), "failed to create new dsi client")

				// Ensure that the replicated data is equal to our previously read
				// data
				replicatedData, err := client.Read(ctx, entity)
				Expect(err).To(BeNil(), "failed to read data")
				Expect(readData).To(Equal(replicatedData),
					"read data does not match data replicated in new primary")
			})
		})
	})
})
