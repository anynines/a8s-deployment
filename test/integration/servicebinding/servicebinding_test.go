package servicebinding

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"

	"github.com/anynines/a8s-deployment/test/integration/framework"
	"github.com/anynines/a8s-deployment/test/integration/framework/dsi"
	"github.com/anynines/a8s-deployment/test/integration/framework/secret"
	"github.com/anynines/a8s-deployment/test/integration/framework/servicebinding"
	sbv1alpha1 "github.com/anynines/a8s-service-binding-controller/api/v1alpha1"
)

const (
	instancePort = 5432
	replicas     = 1
	suffixLength = 5
	sbAmount     = 5
	dsiAmount    = 3

	DbAdminUsernameKey = "username"
	DbAdminPasswordKey = "password"
	AppsDefaultDb      = "a9s_apps_default_db"
)

var (
	portForwardStopCh chan struct{}

	sb             *sbv1alpha1.ServiceBinding
	instance       dsi.Object
	dsiAdminClient dsi.DSIClient
	dsiSbClient    dsi.DSIClient
	sbClientMap    map[*sbv1alpha1.ServiceBinding]dsi.DSIClient
)

var _ = Describe("Service binding", func() {
	Context("Single ServiceBinding for a single DSI", func() {
		BeforeEach(func() {
			// Create Dataservice instance and wait for instance readiness
			instance, err = dsi.New(
				dataservice,
				testingNamespace,
				framework.GenerateName(instanceNamePrefix,
					GinkgoParallelProcess(),
					suffixLength),
				replicas,
			)
			Expect(err).To(BeNil(), "failed to generate DSI object")

			Expect(k8sClient.Create(ctx, instance.GetClientObject())).
				To(Succeed(), "failed to create DSI")

			dsi.WaitForReadiness(ctx, instance.GetClientObject(), k8sClient)

			// Portforward to access DSI from outside cluster.
			portForwardStopCh, localPort, err = framework.PortForward(
				ctx, instancePort, kubeconfigPath, instance, k8sClient)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to establish portforward to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()))

			adminSecret, err := secret.AdminSecretData(ctx,
				k8sClient,
				instance.GetName(),
				testingNamespace)

			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to parse secret data of admin credentials for DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()))

			// Create DSIClient for interacting with the new DSI.
			dsiAdminClient, err = dsi.NewClient(dataservice,
				strconv.Itoa(localPort),
				adminSecret)

			Expect(err).To(BeNil(), "failed to create DSI client")
		})

		AfterEach(func() {
			defer func() { close(portForwardStopCh) }()
			Expect(k8sClient.Delete(ctx, instance.GetClientObject())).To(Succeed(),
				fmt.Sprintf("failed to delete DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()))
		})

		It("Performs basic service binding lifecycle operations", func() {
			// Create service binding for DSI.
			sb = servicebinding.New(
				servicebinding.SetNamespacedName(instance.GetClientObject()),
				servicebinding.SetInstanceRef(instance.GetClientObject()),
			)

			By("Creating the service binding CR", func() {
				Expect(k8sClient.Create(ctx, sb)).To(Succeed(),
					fmt.Sprintf("failed to create new servicebinding for DSI %s/%s",
						instance.GetNamespace(),
						instance.GetName()))

				servicebinding.WaitForReadiness(ctx, sb, k8sClient)
			})

			var serviceBindingSecret v1.Secret
			var serviceBindingData secret.SecretData
			By("Creating the service binding secret", func() {
				serviceBindingSecret, err = secret.Get(ctx, k8sClient,
					servicebinding.SecretName(sb.Name), testingNamespace)

				Expect(err).To(BeNil(),
					fmt.Sprintf("unable to get service binding secret for DSI %s/%s",
						instance.GetNamespace(),
						instance.GetName()))

				serviceBindingData = secret.ParseRawSecretData(serviceBindingSecret.Data)
				Expect(serviceBindingData).To(Not(BeNil()),
					"unable to parse secret data")
			})

			By("Creating a user in the Database", func() {
				exists, err := dsiAdminClient.UserExists(ctx,
					serviceBindingData[DbAdminUsernameKey])

				Expect(err).To(BeNil(),
					fmt.Sprintf("unable to get service binding user for DSI %s/%s",
						instance.GetNamespace(),
						instance.GetName()))

				Expect(exists).To(BeTrue(),
					fmt.Sprintf("service binding user is missing for DSI %s/%s",
						instance.GetNamespace(),
						instance.GetName()))
			})

			// Create DSIClient for interacting with the DSI using the SB user
			dsiSbClient, err = dsi.NewClient(dataservice,
				strconv.Itoa(localPort),
				serviceBindingData)
			Expect(err).To(BeNil(), "failed to create Service Binding client")

			By("Ensuring that SB user can write data in the a9s_apps_default_db database", func() {
				err := dsiSbClient.Write(ctx, AppsDefaultDb, "sample data")
				Expect(err).To(BeNil(),
					fmt.Sprintf("unable to insert data into the %s for DSI %s/%s",
						AppsDefaultDb,
						instance.GetNamespace(),
						instance.GetName()))
			})

			By("Ensuring that SB user can read data from the a9s_apps_default_db database", func() {
				readData, err := dsiSbClient.Read(ctx, AppsDefaultDb)
				Expect(err).To(BeNil(),
					fmt.Sprintf("unable to read data from the %s for DSI %s/%s",
						AppsDefaultDb,
						instance.GetNamespace(),
						instance.GetName()))

				Expect(readData).To(Equal("sample data"),
					fmt.Sprintf("data read from %s doesn't match the written data for DSI %s/%s",
						AppsDefaultDb,
						instance.GetNamespace(),
						instance.GetName()))
			})

			By("Deleting the service binding", func() {
				Expect(k8sClient.Delete(ctx, sb)).To(Succeed(),
					fmt.Sprintf("failed to delete service binding resource for DSI %s/%s",
						instance.GetNamespace(),
						instance.GetName()))
			})

			By("Deleting the user in the Database", func() {
				EventuallyWithOffset(1, func() bool {
					userExists, _ := dsiAdminClient.UserExists(ctx,
						serviceBindingData[DbAdminUsernameKey])
					return userExists
				}, framework.AsyncOpsTimeoutMins, 1*time.Second).
					Should(BeFalse(),
						fmt.Sprintf("timeout reached waiting for deletion of the SB user for DSI %s/%s",
							instance.GetNamespace(),
							instance.GetName()))
			})

			By("Deleting the service binding secret", func() {
				EventuallyWithOffset(1, func() v1.Secret {
					s, _ := secret.Get(ctx,
						k8sClient,
						servicebinding.SecretName(sb.Name),
						testingNamespace)

					return s
				}, framework.AsyncOpsTimeoutMins, 1*time.Second).
					Should(Equal(v1.Secret{}),
						"timeout reached waiting for deletion of the SB secret")
			})
		})
	})

	Context("Multiple ServiceBindings for a single DSI", func() {
		BeforeEach(func() {
			// Create Dataservice instance and wait for instance readiness
			instance, err = dsi.New(
				dataservice,
				testingNamespace,
				framework.GenerateName(instanceNamePrefix,
					GinkgoParallelProcess(),
					suffixLength),
				replicas,
			)
			Expect(err).To(BeNil(), "failed to generate DSI object")

			Expect(k8sClient.Create(ctx, instance.GetClientObject())).
				To(Succeed(), "failed to create DSI")

			dsi.WaitForReadiness(ctx, instance.GetClientObject(), k8sClient)

			// Portforward to access DSI from outside cluster.
			portForwardStopCh, localPort, err = framework.PortForward(
				ctx, instancePort, kubeconfigPath, instance, k8sClient)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to establish portforward to DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()))

			adminSecret, err := secret.AdminSecretData(ctx,
				k8sClient,
				instance.GetName(),
				testingNamespace)
			Expect(err).To(BeNil(),
				fmt.Sprintf("failed to parse secret data of admin credentials for DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()))

			// Create DSIClient for interacting with the new DSI.
			dsiAdminClient, err = dsi.NewClient(dataservice,
				strconv.Itoa(localPort),
				adminSecret)

			Expect(err).To(BeNil(), "failed to create DSI client")
		})

		AfterEach(func() {
			defer func() { close(portForwardStopCh) }()
			Expect(k8sClient.Delete(ctx, instance.GetClientObject())).To(Succeed(),
				fmt.Sprintf("failed to delete DSI %s/%s",
					instance.GetNamespace(),
					instance.GetName()))
		})

		It("Performs basic ServiceBinding lifecycle operations", func() {
			sbs := make([]*sbv1alpha1.ServiceBinding, sbAmount)

			By("Creating the ServiceBinding CR", func() {
				for i := 0; i < sbAmount; i++ {
					// Create multiple service bindings for one DSI.
					sbs[i] = servicebinding.New(
						servicebinding.SetNamespacedName(instance.GetClientObject()),
						servicebinding.SetInstanceRef(instance.GetClientObject()),
					)

					Expect(k8sClient.Create(ctx, sbs[i])).To(Succeed(),
						fmt.Sprintf("failed to create new service binding for DSI %s/%s",
							instance.GetNamespace(),
							instance.GetName()))
				}

				for _, sb := range sbs {
					servicebinding.WaitForReadiness(ctx, sb, k8sClient)
				}
			})

			serviceBindingData := make(map[*sbv1alpha1.ServiceBinding]secret.SecretData)
			var serviceBindingSecret v1.Secret
			By("Creating the service binding secret", func() {
				for _, sb := range sbs {
					serviceBindingSecret, err = secret.Get(ctx,
						k8sClient,
						servicebinding.SecretName(sb.Name),
						testingNamespace)

					Expect(err).To(BeNil(),
						"unable to get service binding secret")

					sbData := secret.ParseRawSecretData(serviceBindingSecret.Data)
					Expect(sbData).To(Not(BeNil()),
						"unable to parse secret data for sb")

					serviceBindingData[sb] = sbData
				}
			})

			// Create one DSIClient per ServiceBinding user for interacting with the
			// DSI.
			sbClientMap = make(map[*sbv1alpha1.ServiceBinding]dsi.DSIClient)
			for _, sb := range sbs {
				dsiSbClient, err = dsi.NewClient(dataservice,
					strconv.Itoa(localPort),
					serviceBindingData[sb])

				Expect(err).To(BeNil(), "failed to create service binding client")
				sbClientMap[sb] = dsiSbClient
			}

			By("Creating one user per service binding in the Database", func() {
				for _, sbData := range serviceBindingData {
					exists, err := dsiAdminClient.UserExists(ctx,
						sbData[DbAdminUsernameKey])

					Expect(err).To(BeNil(),
						fmt.Sprintf("unable to get service binding user for DSI %s/%s",
							instance.GetNamespace(),
							instance.GetName()))

					Expect(exists).To(BeTrue(),
						fmt.Sprintf("service binding user is missing for DSI %s/%s",
							instance.GetNamespace(),
							instance.GetName()))
				}
			})

			var writtenData string

			By("Ensuring that SB user can write data in the a9s_apps_default_db database", func() {
				for _, sb := range sbs {
					err := sbClientMap[sb].Write(ctx,
						AppsDefaultDb,
						"sample data")

					Expect(err).To(BeNil(),
						fmt.Sprintf("unable to insert data into the %s for DSI %s/%s",
							AppsDefaultDb,
							instance.GetNamespace(),
							instance.GetName()))

					writtenData = writtenData + "sample data\n"
				}

				// When reading data from the database a newline is appended after
				// each entry but the last one from the table. Therefore we need
				// also to get rid of the newline at the end of the writtenData
				// variable.
				writtenData = strings.TrimSuffix(writtenData, "\n")
			})

			By("Ensuring that SB user can read data from the a9s_apps_default_db database", func() {
				for _, sb := range sbs {
					readData, err := sbClientMap[sb].Read(ctx, AppsDefaultDb)
					Expect(err).To(BeNil(),
						fmt.Sprintf("unable to read data from the %s for DSI %s/%s",
							AppsDefaultDb,
							instance.GetNamespace(),
							instance.GetName()))

					Expect(readData).To(Equal(writtenData),
						fmt.Sprintf("data read from %s doesn't match the written data for DSI %s/%s",
							AppsDefaultDb,
							instance.GetNamespace(),
							instance.GetName()))
				}
			})

			By("Deleting the ServiceBinding", func() {
				for _, sb := range sbs {
					Expect(k8sClient.Delete(ctx, sb)).To(Succeed(),
						fmt.Sprintf("failed to delete sb resource for DSI %s/%s",
							instance.GetNamespace(),
							instance.GetName()))
				}
			})

			By("Deleting the user in the Database", func() {
				for _, sbData := range serviceBindingData {
					EventuallyWithOffset(1, func() bool {
						userExists, _ := dsiAdminClient.UserExists(ctx,
							sbData[DbAdminUsernameKey])

						return userExists
					}, framework.AsyncOpsTimeoutMins, 1*time.Second).
						Should(BeFalse(),
							fmt.Sprintf("timeout reached waiting for deletion of the SB user for DSI %s/%s",
								instance.GetNamespace(),
								instance.GetName()))
				}
			})

			By("Deleting the service binding secret", func() {
				for _, sb := range sbs {
					EventuallyWithOffset(1, func() v1.Secret {
						s, _ := secret.Get(ctx,
							k8sClient,
							servicebinding.SecretName(sb.Name),
							testingNamespace)

						return s
					}, framework.AsyncOpsTimeoutMins, 1*time.Second).
						Should(Equal(v1.Secret{}),
							"timeout reached waiting for deletion of the SB secret")
				}
			})
		})
	})

	Context("Multiple ServiceBindings for multiple DSIs", func() {
		var serviceBindingSecret v1.Secret
		type dataServiceInstance struct {
			adminClient       dsi.DSIClient
			portForward       int
			portForwardStopCh chan struct{}
		}
		dataServiceInstanceMap := make(map[dsi.Object]dataServiceInstance)
		instances := make([]dsi.Object, dsiAmount)

		BeforeEach(func() {
			for i := 0; i < dsiAmount; i++ {
				// Create Dataservice instance and wait for instance readiness
				instance, err = dsi.New(
					dataservice,
					testingNamespace,
					framework.GenerateName(instanceNamePrefix,
						GinkgoParallelProcess(),
						suffixLength),
					replicas,
				)
				Expect(err).To(BeNil(), "failed to generate DSI object")
				instances[i] = instance
				Expect(k8sClient.Create(ctx, instance.GetClientObject())).
					To(Succeed(), "failed to create DSI")
			}

			for _, instance := range instances {
				dsi.WaitForReadiness(ctx, instance.GetClientObject(), k8sClient)

				// Portforward to access DSI from outside cluster.
				portForwardStopCh, localPort, err = framework.PortForward(
					ctx, instancePort, kubeconfigPath, instance, k8sClient)
				Expect(err).To(BeNil(),
					fmt.Sprintf("failed to establish portforward to DSI %s/%s",
						instance.GetNamespace(),
						instance.GetName()))

				adminSecret, err := secret.AdminSecretData(ctx, k8sClient,
					instance.GetName(), testingNamespace)
				Expect(err).To(BeNil(),
					fmt.Sprintf("failed to parse secret data of admin credentials for DSI %s/%s",
						instance.GetNamespace(),
						instance.GetName()))

				// Create DSIClient for interacting with the new DSI.
				dsiAdminClient, err = dsi.NewClient(dataservice,
					strconv.Itoa(localPort),
					adminSecret)
				Expect(err).To(BeNil(), "failed to create DSI client")

				// In case we test multiple DSIs we need to map the adminClient,
				// localPort and portForwardStopCh to the DSI they belong to.
				dataServiceInstanceMap[instance] = dataServiceInstance{
					adminClient:       dsiAdminClient,
					portForward:       localPort,
					portForwardStopCh: portForwardStopCh,
				}
			}
		})

		AfterEach(func() {
			for _, instance := range instances {
				defer close(dataServiceInstanceMap[instance].portForwardStopCh)
				Expect(k8sClient.Delete(ctx, instance.GetClientObject())).To(Succeed(),
					fmt.Sprintf("failed to delete DSI %s/%s",
						instance.GetNamespace(),
						instance.GetName()))
			}
		})

		It("Performs basic ServiceBinding lifecycle operations", func() {
			for _, instance := range instances {
				serviceBindingData := make(map[*sbv1alpha1.ServiceBinding]secret.SecretData)
				sbs := make([]*sbv1alpha1.ServiceBinding, sbAmount)

				By("Creating the ServiceBinding CR", func() {
					for i := 0; i < sbAmount; i++ {
						sbs[i] = servicebinding.New(
							servicebinding.SetNamespacedName(instance.GetClientObject()),
							servicebinding.SetInstanceRef(instance.GetClientObject()),
						)

						Expect(k8sClient.Create(ctx, sbs[i])).To(Succeed(),
							"failed to create new servicebinding")
					}

					for _, sb := range sbs {
						servicebinding.WaitForReadiness(ctx, sb, k8sClient)
						sbData, err := secret.Data(ctx,
							k8sClient,
							servicebinding.SecretName(sb.Name),
							testingNamespace)

						Expect(err).To(BeNil(),
							"failed to get SB secret and parse the Secret data")

						serviceBindingData[sb] = sbData
					}
				})

				By("Creating the service binding secret", func() {
					for _, sb := range sbs {
						serviceBindingSecret, err = secret.Get(ctx,
							k8sClient,
							servicebinding.SecretName(sb.Name),
							testingNamespace)

						Expect(err).To(BeNil(),
							"unable to get service binding secret")

						sbData := secret.ParseRawSecretData(serviceBindingSecret.Data)
						Expect(serviceBindingData).To(Not(BeNil()),
							"unable to parse secret data for sb")

						serviceBindingData[sb] = sbData
					}
				})

				// Create one DSIClient per ServiceBinding user for interacting
				// with the DSI.
				sbClientMap = make(map[*sbv1alpha1.ServiceBinding]dsi.DSIClient)
				for _, sb := range sbs {
					dsiSbClient, err = dsi.NewClient(dataservice,
						strconv.Itoa(dataServiceInstanceMap[instance].portForward),
						serviceBindingData[sb])
					Expect(err).To(BeNil(),
						"failed to create Service Binding client")

					sbClientMap[sb] = dsiSbClient
				}

				By("Creating one user per service binding in the Database", func() {
					for _, sbData := range serviceBindingData {
						exists, err := dataServiceInstanceMap[instance].adminClient.UserExists(ctx,
							sbData[DbAdminUsernameKey])

						Expect(err).To(BeNil(),
							fmt.Sprintf("unable to get service binding user for DSI %s/%s",
								instance.GetNamespace(),
								instance.GetName()))

						Expect(exists).To(BeTrue(),
							fmt.Sprintf("service binding user is missing for DSI %s/%s",
								instance.GetNamespace(),
								instance.GetName()))
					}
				})

				var writtenData string

				By("Ensuring that the SB user can write data in the a9s_apps_default_db database",
					func() {
						for _, sb := range sbs {
							err := sbClientMap[sb].Write(ctx,
								AppsDefaultDb,
								"sample data")

							Expect(err).To(BeNil(),
								fmt.Sprintf("unable to insert data into the %s for DSI %s/%s",
									AppsDefaultDb,
									instance.GetNamespace(),
									instance.GetName()))

							writtenData = writtenData + "sample data\n"
						}

						writtenData = strings.TrimSuffix(writtenData, "\n")
					})

				By("Ensuring that the SB user can read data from the a9s_apps_default_db database",
					func() {
						for _, sb := range sbs {
							readData, err := sbClientMap[sb].Read(ctx,
								AppsDefaultDb)

							Expect(err).To(BeNil(),
								fmt.Sprintf("unable to read data from the %s for DSI %s/%s",
									AppsDefaultDb,
									instance.GetNamespace(),
									instance.GetName()))

							Expect(readData).To(Equal(writtenData),
								fmt.Sprintf("data read from %s doesn't match the written data for DSI %s/%s",
									AppsDefaultDb,
									instance.GetNamespace(),
									instance.GetName()))
						}
					})

				By("Deleting the ServiceBinding", func() {
					for _, sb := range sbs {
						Expect(k8sClient.Delete(ctx, sb)).To(Succeed(),
							fmt.Sprintf("failed to delete sb resource for DSI %s/%s",
								instance.GetNamespace(),
								instance.GetName()))
					}
				})

				By("Deleting the user in the Database", func() {
					for _, sbData := range serviceBindingData {
						EventuallyWithOffset(1, func() bool {
							userExists, _ := dsiAdminClient.UserExists(ctx,
								sbData[DbAdminUsernameKey])
							return userExists
						}, framework.AsyncOpsTimeoutMins, 1*time.Second).
							Should(BeFalse(),
								fmt.Sprintf("timeout reached waiting for deletion of the SB user for DSI %s/%s",
									instance.GetNamespace(),
									instance.GetName()))
					}
				})

				By("Deleting the service binding secret", func() {
					for _, sb := range sbs {
						EventuallyWithOffset(1, func() v1.Secret {
							s, _ := secret.Get(ctx,
								k8sClient,
								servicebinding.SecretName(sb.Name),
								testingNamespace)

							return s
						}, framework.AsyncOpsTimeoutMins, 1*time.Second).
							Should(Equal(v1.Secret{}),
								"timeout reached waiting for deletion of the SB secret")
					}
				})
			}
		})
	})
})
