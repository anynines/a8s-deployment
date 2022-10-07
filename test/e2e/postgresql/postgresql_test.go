package postgresql

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/anynines/a8s-deployment/test/framework"
	"github.com/anynines/a8s-deployment/test/framework/dsi"
	"github.com/anynines/a8s-deployment/test/framework/postgresql"
	"github.com/anynines/a8s-deployment/test/framework/secret"
	"github.com/anynines/a8s-deployment/test/framework/servicebinding"
	sbv1alpha1 "github.com/anynines/a8s-service-binding-controller/api/v1alpha1"
	pgv1alpha1 "github.com/anynines/postgresql-operator/api/v1alpha1"
)

const (
	instancePort = 5432
	replicas     = 1
	suffixLength = 5

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

var _ = Describe("PostgreSQL Operator end-to-end tests", func() {
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
				Expect(sts.Spec.Template.Labels).To(HaveKeyWithValue("a8s.a9s/dsi-name", pg.Name))
				// TODO: find a way to avoid hardcoding
				Expect(sts.Spec.Template.Labels).
					To(HaveKeyWithValue("a8s.a9s/dsi-group", "postgresql.anynines.com"))
				Expect(sts.Spec.Template.Labels).
					To(HaveKeyWithValue("a8s.a9s/dsi-kind", "Postgresql"))
				Expect(len(sts.Spec.Template.Labels)).To(Equal(3))

				Expect(sts.Spec.Template.Annotations).
					To(HaveKeyWithValue("prometheus.io/port", "9187"))
				Expect(sts.Spec.Template.Annotations).
					To(HaveKeyWithValue("prometheus.io/scrape", "true"))

				Expect(sts.Spec.Template.Spec.Containers[0].Name).To(Equal("postgres"))
				Expect(sts.Spec.Template.Spec.Containers[1].Name).To(Equal("backup-agent"))

				Expect(sts.Spec.Template.Spec.ServiceAccountName).To(Equal(pg.Name))
			})

			By("creating a Service that points to the primary for writes", func() {
				svc := &corev1.Service{}
				Expect(k8sClient.Get(ctx,
					types.NamespacedName{
						Name: postgresql.MasterService(
							instance.GetName()),
						Namespace: instance.GetNamespace()},
					svc)).To(Succeed())

				Expect(svc.Spec.Selector).To(HaveKeyWithValue("a8s.a9s/dsi-name", pg.Name))
				Expect(svc.Spec.Selector).
					To(HaveKeyWithValue("a8s.a9s/dsi-group", "postgresql.anynines.com"))
				Expect(svc.Spec.Selector).To(HaveKeyWithValue("a8s.a9s/dsi-kind", "Postgresql"))
				Expect(svc.Spec.Selector).To(HaveKeyWithValue("a8s.a9s/replication-role", "master"))
				Expect(len(svc.Spec.Selector)).To(Equal(4))

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
					To(HaveKeyWithValue("a8s.a9s/dsi-name", instance.GetName()))
				Expect(adminRoleSecret.Labels).
					To(HaveKeyWithValue("a8s.a9s/dsi-group", "postgresql.anynines.com"))
				Expect(adminRoleSecret.Labels).
					To(HaveKeyWithValue("a8s.a9s/dsi-kind", "Postgresql"))
				Expect(len(adminRoleSecret.Labels)).To(Equal(3))
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
					To(HaveKeyWithValue("a8s.a9s/dsi-name", instance.GetName()))
				Expect(standbyRoleSecret.Labels).
					To(HaveKeyWithValue("a8s.a9s/dsi-group", "postgresql.anynines.com"))
				Expect(standbyRoleSecret.Labels).
					To(HaveKeyWithValue("a8s.a9s/dsi-kind", "Postgresql"))
				Expect(len(standbyRoleSecret.Labels)).To(Equal(3))
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

			// TODO: test that events are emitted in failure cases
			// TODO: after the switch to ginkgo v2 (or the decision to get rid of gingko), find a
			// way to handle the flakiness of the checks on events. Test on events are flaky because
			// events creation is best-effort by design. If we stay with ginkgo we wait until the
			// switch to v2 because there are new decorators that explicitly handle flaky tests:
			// https://pkg.go.dev/github.com/onsi/ginkgo/v2#FlakeAttempts .
			By("emitting exactly one event for each secondary API object that is directly "+
				"created by the operator, and no more", func() {

				instanceEvents := &corev1.EventList{}
				Expect(k8sClient.List(ctx, instanceEvents, &ctrlruntimeclient.ListOptions{
					FieldSelector: ctrlruntimeclient.MatchingFieldsSelector{
						Selector: fields.OneTermEqualSelector("involvedObject.uid",
							string(instance.GetUID())),
					},
				})).To(Succeed(), "failed to list events emitted for test DSI")

				Expect(len(instanceEvents.Items)).To(Equal(6), "found more events than expected, "+
					"there should be one for every secondary API object that the Operator "+
					"directly creates (ServiceAccount, RoleBinding, master Service, StatefulSet, "+
					"two Secrets)")

				// Sort events by message so that we know for which secondary API object each event
				// is created w/o having to inspect the event first. *This is a hack that makes the
				// tests brittle*, but doing otherwise would require more code and more logic, which
				// results in less readable (and potentially buggy) tests.
				sort.Slice(instanceEvents.Items, func(i, j int) bool {
					return instanceEvents.Items[i].Message <= instanceEvents.Items[j].Message
				})
				masterSvcEvent := instanceEvents.Items[0]
				roleBindingEvent := instanceEvents.Items[1]
				adminSecretEvent := instanceEvents.Items[2]
				standbySecretsEvent := instanceEvents.Items[3]
				svcAccountEvent := instanceEvents.Items[4]
				ssetEvent := instanceEvents.Items[5]

				By("emitting an event for the creation of the master service", func() {
					Expect(masterSvcEvent.Message).To(Equal("Successfully created master service"),
						"wrong event message")
					Expect(masterSvcEvent.Type).To(Equal(corev1.EventTypeNormal),
						"wrong event type")
					Expect(masterSvcEvent.Reason).To(Equal("Created"), "wrong event reason")
					Expect(masterSvcEvent.Count).To(Equal(int32(1)), "wrong event count")
					Expect(masterSvcEvent.Source.Component).To(Equal("postgresql-controller"),
						"wrong event source.component")
					Expect(masterSvcEvent.InvolvedObject.Kind).To(Equal("Postgresql"),
						"wrong event involvedObject.kind")
					Expect(masterSvcEvent.InvolvedObject.APIVersion).
						To(Equal("postgresql.anynines.com/v1beta3"),
							"wrong event involvedObject.apiVersion")
				})

				By("emitting an event for the creation of the roleBinding", func() {
					Expect(roleBindingEvent.Message).To(Equal("Successfully created roleBinding"),
						"wrong event message")
					Expect(roleBindingEvent.Type).To(Equal(corev1.EventTypeNormal),
						"wrong event type")
					Expect(roleBindingEvent.Reason).To(Equal("Created"), "wrong event reason")
					Expect(roleBindingEvent.Count).To(Equal(int32(1)), "wrong event count")
					Expect(roleBindingEvent.Source.Component).To(Equal("postgresql-controller"),
						"wrong event source.component")
					Expect(roleBindingEvent.InvolvedObject.Kind).To(Equal("Postgresql"),
						"wrong event involvedObject.kind")
					Expect(roleBindingEvent.InvolvedObject.APIVersion).
						To(Equal("postgresql.anynines.com/v1beta3"),
							"wrong event involvedObject.apiVersion")
				})

				By("emitting an event for the creation of the admin secret", func() {
					adminSecretNSN := types.NamespacedName{
						Namespace: instance.GetNamespace(),
						Name:      postgresql.AdminRoleSecretName(instance.GetName())}

					Expect(adminSecretEvent.Message).
						To(Equal(fmt.Sprintf("Successfully created secret %s", adminSecretNSN)),
							"wrong event message")
					Expect(adminSecretEvent.Type).To(Equal(corev1.EventTypeNormal), "wrong event type")
					Expect(adminSecretEvent.Reason).To(Equal("Created"), "wrong event reason")
					Expect(adminSecretEvent.Count).To(Equal(int32(1)), "wrong event count")
					Expect(adminSecretEvent.Source.Component).To(Equal("postgresql-controller"),
						"wrong event source.component")
					Expect(adminSecretEvent.InvolvedObject.Kind).To(Equal("Postgresql"),
						"wrong event involvedObject.kind")
					Expect(adminSecretEvent.InvolvedObject.APIVersion).
						To(Equal("postgresql.anynines.com/v1beta3"),
							"wrong event involvedObject.apiVersion")
				})

				By("emitting an event for the creation of the standby secret", func() {
					standbySecretNSN := types.NamespacedName{
						Namespace: instance.GetNamespace(),
						Name:      postgresql.StandbyRoleSecretName(instance.GetName())}

					Expect(standbySecretsEvent.Message).
						To(Equal(fmt.Sprintf("Successfully created secret %s", standbySecretNSN)),
							"wrong event message")
					Expect(standbySecretsEvent.Type).To(Equal(corev1.EventTypeNormal), "wrong event type")
					Expect(standbySecretsEvent.Reason).To(Equal("Created"), "wrong event reason")
					Expect(standbySecretsEvent.Count).To(Equal(int32(1)), "wrong event count")
					Expect(standbySecretsEvent.Source.Component).To(Equal("postgresql-controller"),
						"wrong event source.component")
					Expect(standbySecretsEvent.InvolvedObject.Kind).To(Equal("Postgresql"),
						"wrong event involvedObject.kind")
					Expect(standbySecretsEvent.InvolvedObject.APIVersion).
						To(Equal("postgresql.anynines.com/v1beta3"),
							"wrong event involvedObject.apiVersion")
				})

				By("emitting an event for the creation of the serviceAcccount", func() {
					Expect(svcAccountEvent.Message).To(Equal("Successfully created serviceAccount"),
						"wrong event message")
					Expect(svcAccountEvent.Type).To(Equal(corev1.EventTypeNormal),
						"wrong event type")
					Expect(svcAccountEvent.Reason).To(Equal("Created"), "wrong event reason")
					Expect(svcAccountEvent.Count).To(Equal(int32(1)), "wrong event count")
					Expect(svcAccountEvent.Source.Component).To(Equal("postgresql-controller"),
						"wrong event source.component")
					Expect(svcAccountEvent.InvolvedObject.Kind).To(Equal("Postgresql"),
						"wrong event involvedObject.kind")
					Expect(svcAccountEvent.InvolvedObject.APIVersion).
						To(Equal("postgresql.anynines.com/v1beta3"),
							"wrong event involvedObject.apiVersion")
				})

				By("emitting an event for the creation of the statefulSet", func() {
					Expect(ssetEvent.Message).To(Equal("Successfully created statefulSet"),
						"wrong event message")
					Expect(ssetEvent.Type).To(Equal(corev1.EventTypeNormal), "wrong event type")
					Expect(ssetEvent.Reason).To(Equal("Created"), "wrong event reason")
					Expect(ssetEvent.Count).To(Equal(int32(1)), "wrong event count")
					Expect(ssetEvent.Source.Component).To(Equal("postgresql-controller"),
						"wrong event source.component")
					Expect(ssetEvent.InvolvedObject.Kind).To(Equal("Postgresql"),
						"wrong event involvedObject.kind")
					Expect(ssetEvent.InvolvedObject.APIVersion).
						To(Equal("postgresql.anynines.com/v1beta3"),
							"wrong event involvedObject.apiVersion")
				})
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

			By("emitting an event about the instance deletion", func() {
				events := &corev1.EventList{}
				Expect(k8sClient.List(ctx, events, &ctrlruntimeclient.ListOptions{
					FieldSelector: fields.AndSelectors(
						fields.OneTermEqualSelector("reason", "Deleted"),
						fields.OneTermEqualSelector("involvedObject.uid",
							string(instance.GetUID()))),
				})).To(Succeed(), "failed to list events emitted for deletion of the DSI")

				Expect(len(events.Items)).To(Equal(1),
					"exactly one event should be emitted for the deletion of a DSI")

				event := events.Items[0]
				Expect(event.Message).To(Equal("Successfully deleted Instance"),
					"wrong event message")
				Expect(event.Type).To(Equal(corev1.EventTypeNormal), "wrong event type")
				Expect(event.Count).To(Equal(int32(1)), "wrong number of events")
				Expect(event.Source.Component).To(Equal("postgresql-controller"),
					"wrong event source.component")
				Expect(event.InvolvedObject.Kind).To(Equal("Postgresql"),
					"wrong event involvedObject.kind")
				Expect(event.InvolvedObject.APIVersion).
					To(Equal("postgresql.anynines.com/v1beta3"),
						"wrong event involvedObject.apiVersion")
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

				// TODO: This is only a temporary solution to an issue that was introduced
				// by the PostgreSQL extensions feature. In order to install PostgreSQL extensions
				// the PostgreSQL-Operator executes an installation script. This overwrites the
				// default Spilo entrypoint command of the container image and introduces
				// additional latency to the initialization of a PostgreSQL instance. This means
				// that the port-forward the tests are using might be opened while Patroni
				// is not fully initialized. As a result the port-forward breaks and is closed.
				// Wrapping everything into an Eventually solves this issue as opening a
				// port-forward and using the service binding to read data from the DSI is
				// retried in case of a failure.
				Eventually(func(g Gomega) {
					// Portforward to access new primary pod from outside cluster.
					portForwardStopCh, localPort, err = framework.PortForward(
						ctx, instancePort, kubeconfigPath,
						instance, k8sClient)
					g.Expect(err).To(BeNil(),
						fmt.Sprintf("failed to establish portforward to DSI %s/%s",
							instance.GetNamespace(), instance.GetName()))

					// Create client for interacting with the new PostgreSQL primary
					// node
					client, err = dsi.NewClient(dataservice,
						strconv.Itoa(localPort), serviceBindingData)
					g.Expect(err).To(BeNil(), "failed to create new dsi client")

					// Ensure that newly read data matches our original test input
					readData, err = client.Read(ctx, entity)
					g.Expect(err).To(BeNil(), "failed to read data")
					g.Expect(readData).To(Equal(testInput), "read data does not match test input")
				}, 60*time.Second).Should(Succeed())
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
				Expect(pod.Labels["a8s.a9s/replication-role"]).To(Equal("master"))
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
				Expect(newPod.Labels["a8s.a9s/replication-role"]).To(Equal("master"))
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
				// TODO: This is only a temporary solution to an issue that was introduced
				// by the PostgreSQL extensions feature. In order to install PostgreSQL extensions
				// the PostgreSQL-Operator executes an installation script. This overwrites the
				// default Spilo entrypoint command of the container image and introduces
				// additional latency to the initialization of a PostgreSQL instance. This means
				// that the port-forward the tests are using might be opened while Patroni
				// is not fully initialized. As a result the port-forward breaks and is closed.
				// Wrapping everything into an Eventually solves this issue as opening a
				// port-forward and using the service binding to read data from the DSI is
				// retried in case of a failure.
				Eventually(func(g Gomega) {
					// Portforward to access new primary pod from outside cluster.
					portForwardStopCh, localPort, err = framework.PortForward(
						ctx, instancePort, kubeconfigPath, instance, k8sClient)
					g.Expect(err).To(BeNil(),
						fmt.Sprintf("failed to establish portforward to DSI %s/%s",
							instance.GetNamespace(), instance.GetName()))

					// Create client for interacting with the new instance.
					client, err = dsi.NewClient(
						dataservice, strconv.Itoa(localPort), serviceBindingData)
					g.Expect(err).To(BeNil(), "failed to create new dsi client")

					// Ensure that the replicated data is equal to our previously read
					// data
					replicatedData, err := client.Read(ctx, entity)
					g.Expect(err).To(BeNil(), "failed to read data")
					g.Expect(readData).To(Equal(replicatedData),
						"read data does not match data replicated in new primary")
				}, 60*time.Second).Should(Succeed())
			})
		})
	})
})
