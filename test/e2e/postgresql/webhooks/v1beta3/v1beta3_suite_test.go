package v1beta3

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	//+kubebuilder:scaffold:imports
	"github.com/anynines/a8s-deployment/test/framework"
	"github.com/anynines/a8s-deployment/test/framework/dsi"
	"github.com/anynines/a8s-deployment/test/framework/namespace"
	pgv1beta3 "github.com/anynines/postgresql-operator/api/v1beta3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	runtimeClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.
const (
	suffixLength        = 5
	asyncOpsTimeoutMins = time.Minute * 5
)

var (
	ctx                                                               context.Context
	cancel                                                            context.CancelFunc
	testingNamespace, kubeconfigPath, dataservice, instanceNamePrefix string

	k8sClient runtimeClient.Client

	reservedLabelsKeys []string = []string{
		pgv1beta3.DSINameLabelKey,
		pgv1beta3.DSIGroupLabelKey,
		pgv1beta3.DSIKindLabelKey,
		pgv1beta3.ReplicationRoleLabelKey,
	}
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Validating Webhook Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.Background())

	// Parse environmental variable configuration
	config, err := framework.ParseEnv()
	Expect(err).To(BeNil(), "failed to parse environmental variables as configuration")
	kubeconfigPath, instanceNamePrefix, dataservice, testingNamespace = framework.ConfigToVars(config)

	// Create Kubernetes client for interacting with the Kubernetes API
	k8sClient, err = dsi.NewK8sClient(dataservice, kubeconfigPath)
	Expect(err).To(BeNil(),
		fmt.Sprintf("error creating Kubernetes client for dataservice %s", dataservice))

	Expect(namespace.CreateIfNotExists(ctx, testingNamespace, k8sClient)).
		To(Succeed(), "failed to create testing namespace")
})

var _ = Describe("Validating webhook", func() {
	Context("DSI name length validation on creation", func() {
		It("Allows a DSI with name of only one character", func() {
			dsi := newDSI(withNameOfLength(1))
			err := k8sClient.Create(ctx, dsi)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Allows a DSI with name half the maximum length", func() {
			dsi := newDSI(withNameOfLength(pgv1beta3.MaxNameLengthChars / 2))
			err := k8sClient.Create(ctx, dsi)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Allows a DSI with name shorter than the maximum by one", func() {
			dsi := newDSI(withNameOfLength(pgv1beta3.MaxNameLengthChars - 1))
			err := k8sClient.Create(ctx, dsi)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Allows a DSI with name as long as the maximum", func() {
			dsi := newDSI(withNameOfLength(pgv1beta3.MaxNameLengthChars))
			err := k8sClient.Create(ctx, dsi)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Rejects a DSI with name longer than the maximum by one", func() {
			dsi := newDSI(withNameOfLength(pgv1beta3.MaxNameLengthChars + 1))
			err := k8sClient.Create(ctx, dsi)
			Expect(errors.IsInvalid(err)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("metadata.name"),
				"error message doesn't mention invalid field name")
			Expect(err.Error()).To(ContainSubstring(fmt.Sprint(pgv1beta3.MaxNameLengthChars+1)),
				"error message doesn't mention the actual name length")
			Expect(err.Error()).To(ContainSubstring(fmt.Sprint(pgv1beta3.MaxNameLengthChars)),
				"error message doesn't mention the maximum name length")
		})

		It("Rejects a DSI with name twice the maximum length", func() {
			dsi := newDSI(withNameOfLength(2 * pgv1beta3.MaxNameLengthChars))
			err := k8sClient.Create(ctx, dsi)
			Expect(errors.IsInvalid(err)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("metadata.name"),
				"error message doesn't mention invalid field name")
			Expect(err.Error()).To(ContainSubstring(fmt.Sprint(2*pgv1beta3.MaxNameLengthChars)),
				"error message doesn't mention the actual name length")
			Expect(err.Error()).To(ContainSubstring(fmt.Sprint(pgv1beta3.MaxNameLengthChars)),
				"error message doesn't mention the maximum name length")
		})
	})

	Context("Storage size validation on creation", func() {
		It("Allows a DSI with storage size of 1Gi", func() {
			dsi := newDSI(withName("dsi-1gi-pass"), withStorageSize("1Gi"))
			err := k8sClient.Create(ctx, dsi)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Allows a DSI with storage size of 42Gi", func() {
			dsi := newDSI(withName("dsi-42gi-pass"), withStorageSize("42Gi"))
			err := k8sClient.Create(ctx, dsi)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Allows a DSI with storage size of 2000M", func() {
			dsi := newDSI(withName("dsi-2000m-pass"), withStorageSize("2000M"))
			err := k8sClient.Create(ctx, dsi)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Allows a DSI with storage size of 0.5Gi", func() {
			dsi := newDSI(withName("dsi-0.5gi-pass"), withStorageSize("0.5Gi"))
			err := k8sClient.Create(ctx, dsi)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Rejects a DSI with storage size of 1Mi", func() {
			dsi := newDSI(withName("dsi-1mi-fail"), withStorageSize("1Mi"))
			err := k8sClient.Create(ctx, dsi)
			Expect(errors.IsInvalid(err)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("spec.volumeSize"),
				"error message doesn't mention name of the invalid field")
			Expect(err.Error()).To(ContainSubstring(pgv1beta3.MinVolumeSize),
				"error message doesn't mention the minimum storage size")
			Expect(err.Error()).To(ContainSubstring("1Mi"),
				"error message doesn't mention the specified storage size")
		})

		It("Rejects a DSI with storage size of 1k", func() {
			dsi := newDSI(withName("dsi-1k-fail"), withStorageSize("1k"))
			err := k8sClient.Create(ctx, dsi)
			Expect(errors.IsInvalid(err)).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("spec.volumeSize"),
				"error message doesn't mention name of the invalid field")
			Expect(err.Error()).To(ContainSubstring(pgv1beta3.MinVolumeSize),
				"error message doesn't mention the minimum storage size")
			Expect(err.Error()).To(ContainSubstring("1k"),
				"error message doesn't mention the specified storage size")
		})
	})

	Context("Labels validation on creation", func() {
		var dsi *pgv1beta3.Postgresql

		Context("Valid labels", func() {
			AfterEach(func() {
				Expect(k8sClient.Delete(ctx, dsi)).To(Succeed(), "failed to delete DSI after test")
			})

			It("Allows a DSI with nil labels", func() {
				dsi = newDSI(withLabels(nil))
				Expect(k8sClient.Create(ctx, dsi)).To(Succeed(),
					"failed to create DSI with nil labels even if it's allowed")
			})

			It("Allows a DSI with empty labels", func() {
				dsi = newDSI(withLabels(map[string]string{}))
				Expect(k8sClient.Create(ctx, dsi)).To(Succeed(),
					"failed to create DSI with empty labels even if it's allowed")
			})

			It("Allows a DSI with only allowed labels", func() {
				labels := map[string]string{
					"allowed-label-1": "val1",
					"allowed-label-2": "val2",
				}
				dsi = newDSI(withLabels(labels))
				Expect(k8sClient.Create(ctx, dsi)).To(Succeed(),
					fmt.Sprintf("failed to create DSI with allowed labels %s", labels))
			})

			It("Allows a DSI with only allowed labels which are similar to the reserved ones",
				func() {
					reservedLabelKeyWithExtraCharAtTheBeginning := "x" + reservedLabelsKeys[0]
					reservedLabelKeyWithExtraCharAtTheEnd := reservedLabelsKeys[1] + "a"
					reservedLabelKeyWithoutFirstChar := reservedLabelsKeys[2][1:]
					reservedLabelKeyWithoutLastChar := reservedLabelsKeys[3][:len(reservedLabelsKeys[3])-1]
					reservedLabelKeyWithoutMiddleChar := reservedLabelsKeys[0][:len(reservedLabelsKeys[0])/2] +
						reservedLabelsKeys[0][1+len(reservedLabelsKeys[0])/2:]

					labels := map[string]string{
						reservedLabelKeyWithExtraCharAtTheBeginning: "val1",
						reservedLabelKeyWithExtraCharAtTheEnd:       "val2",
						reservedLabelKeyWithoutFirstChar:            "val3",
						reservedLabelKeyWithoutLastChar:             "val4",
						reservedLabelKeyWithoutMiddleChar:           "val5",
					}
					dsi = newDSI(withLabels(labels))
					Expect(k8sClient.Create(ctx, dsi)).To(Succeed(),
						fmt.Sprintf("failed to create DSI with allowed labels %s", labels))
				})
		})

		Context("Invalid labels", func() {
			It("Rejects a DSI with just one reserved label", func() {
				labels := map[string]string{
					reservedLabelsKeys[0]: "val1",
				}
				dsi = newDSI(withLabels(labels))

				err := k8sClient.Create(ctx, dsi)

				Expect(errors.IsInvalid(err)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring(reservedLabelsKeys[0]),
					"got error that doesn't mention the reserved labels while it should")
				Expect(strings.ToLower(err.Error())).To(ContainSubstring("reserved"),
					"got error that doesn't mention the word \"reserved\" while it should")
				Expect(err.Error()).To(ContainSubstring("metadata.labels"),
					"got error that doesn't mention the name of the invalid field while it should")
			})

			It("Rejects a DSI with two reserved labels", func() {
				labels := map[string]string{
					reservedLabelsKeys[1]: "val1",
					reservedLabelsKeys[2]: "val2",
				}
				dsi = newDSI(withLabels(labels))

				err := k8sClient.Create(ctx, dsi)

				Expect(errors.IsInvalid(err)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring(reservedLabelsKeys[1]),
					"got error that doesn't mention the reserved labels while it should")
				Expect(err.Error()).To(ContainSubstring(reservedLabelsKeys[2]),
					"got error that doesn't mention the reserved labels while it should")
				Expect(strings.ToLower(err.Error())).To(ContainSubstring("reserved"),
					"got error that doesn't mention the word \"reserved\" while it should")
				Expect(err.Error()).To(ContainSubstring("metadata.labels"),
					"got error that doesn't mention the name of the invalid field while it should")
			})

			It("Rejects a DSI whose labels exactly match the reserved ones", func() {
				labels := make(map[string]string, len(reservedLabelsKeys))
				for i, k := range reservedLabelsKeys {
					labels[k] = "val" + strconv.Itoa(i)
				}
				dsi = newDSI(withLabels(labels))

				err := k8sClient.Create(ctx, dsi)

				Expect(errors.IsInvalid(err)).To(BeTrue())
				for _, k := range reservedLabelsKeys {
					Expect(err.Error()).To(ContainSubstring(k),
						"got error that doesn't mention the reserved labels while it should")
				}
				Expect(strings.ToLower(err.Error())).To(ContainSubstring("reserved"),
					"got error that doesn't mention the word \"reserved\" while it should")
				Expect(err.Error()).To(ContainSubstring("metadata.labels"),
					"got error that doesn't mention the name of the invalid field while it should")
			})

			It("Rejects a DSI with one reserved label plus some allowed ones", func() {
				labels := map[string]string{
					"allowed-label-1":     "val1",
					"allowed-label-2":     "val2",
					reservedLabelsKeys[3]: "val3",
				}
				dsi = newDSI(withLabels(labels))

				err := k8sClient.Create(ctx, dsi)

				Expect(errors.IsInvalid(err)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring(reservedLabelsKeys[3]),
					"got error that doesn't mention the reserved labels while it should")
				Expect(strings.ToLower(err.Error())).To(ContainSubstring("reserved"),
					"got error that doesn't mention the word \"reserved\" while it should")
				Expect(err.Error()).To(ContainSubstring("metadata.labels"),
					"got error that doesn't mention the name of the invalid field while it should")
			})

			It("Rejects a DSI with all reserved labels plus some allowed ones", func() {
				labels := make(map[string]string, len(reservedLabelsKeys)+3)
				labels["allowed-label-1"] = "val100"
				labels["allowed-label-2"] = "val101"
				labels["allowed-label-3"] = "val102"
				for i, k := range reservedLabelsKeys {
					labels[k] = "val" + strconv.Itoa(i)
				}
				dsi = newDSI(withLabels(labels))

				err := k8sClient.Create(ctx, dsi)

				Expect(errors.IsInvalid(err)).To(BeTrue())
				for _, k := range reservedLabelsKeys {
					Expect(err.Error()).To(ContainSubstring(k),
						"got error that doesn't mention the reserved labels while it should")
				}
				Expect(strings.ToLower(err.Error())).To(ContainSubstring("reserved"),
					"got error that doesn't mention the word \"reserved\" while it should")
				Expect(err.Error()).To(ContainSubstring("metadata.labels"),
					"got error that doesn't mention the name of the invalid field while it should")
			})
		})
	})

	Context("Labels validation on update", func() {
		var dsi *pgv1beta3.Postgresql

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, dsi)).To(Succeed(), "failed to delete DSI after test")
		})

		Context("Valid labels", func() {
			It("Allows update from valid labels to nil ones", func() {
				labels := map[string]string{
					"allowed-label-1": "val1",
					"allowed-label-2": "val2",
				}
				dsi = newDSI(withLabels(labels))
				Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

				Eventually(func() error {
					var currDSI pgv1beta3.Postgresql
					if err := k8sClient.Get(ctx, types.NamespacedName{
						Namespace: dsi.GetNamespace(),
						Name:      dsi.GetName(),
					}, &currDSI); err != nil {
						return err
					}

					currDSI.Labels = nil

					return k8sClient.Update(ctx, &currDSI)
				}, asyncOpsTimeoutMins, 1*time.Second).Should(BeNil())
			})

			It("Allows update from valid labels to empty ones", func() {
				labels := map[string]string{
					"allowed-label-1": "val1",
				}
				dsi = newDSI(withLabels(labels))
				Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

				Eventually(func() error {
					var currDSI pgv1beta3.Postgresql
					if err := k8sClient.Get(ctx, types.NamespacedName{
						Namespace: dsi.GetNamespace(),
						Name:      dsi.GetName(),
					}, &currDSI); err != nil {
						return err
					}

					currDSI.Labels = map[string]string{}

					return k8sClient.Update(ctx, &currDSI)
				}, asyncOpsTimeoutMins, 1*time.Second).Should(BeNil())
			})

			It("Allows update from nil labels to valid ones", func() {
				dsi = newDSI(withLabels(nil))
				Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

				Eventually(func() error {
					var currDSI pgv1beta3.Postgresql
					if err := k8sClient.Get(ctx, types.NamespacedName{
						Namespace: dsi.GetNamespace(),
						Name:      dsi.GetName(),
					}, &currDSI); err != nil {
						return err
					}

					currDSI.Labels = map[string]string{
						"allowed-label-1": "val1",
					}

					return k8sClient.Update(ctx, &currDSI)
				}, asyncOpsTimeoutMins, 1*time.Second).Should(BeNil())
			})

			It("Allows update from empty labels to valid ones", func() {
				dsi = newDSI(withLabels(map[string]string{}))
				Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

				Eventually(func() error {
					var currDSI pgv1beta3.Postgresql
					if err := k8sClient.Get(ctx, types.NamespacedName{
						Namespace: dsi.GetNamespace(),
						Name:      dsi.GetName(),
					}, &currDSI); err != nil {
						return err
					}

					currDSI.Labels = map[string]string{
						"allowed-label-1": "val1",
						"allowed-label-2": "val2",
					}

					return k8sClient.Update(ctx, &currDSI)
				}, asyncOpsTimeoutMins, 1*time.Second).Should(BeNil())
			})

			It("Allows addition of valid labels", func() {
				labels := map[string]string{
					"allowed-label-1": "val1",
				}
				dsi = newDSI(withLabels(labels))
				Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

				Eventually(func() error {
					var currDSI pgv1beta3.Postgresql
					if err := k8sClient.Get(ctx, types.NamespacedName{
						Namespace: dsi.GetNamespace(),
						Name:      dsi.GetName(),
					}, &currDSI); err != nil {
						return err
					}

					currDSI.Labels = map[string]string{
						"allowed-label-1": "val1",
						"allowed-label-2": "val2",
						"allowed-label-3": "val3",
					}

					return k8sClient.Update(ctx, &currDSI)
				}, asyncOpsTimeoutMins, 1*time.Second).Should(BeNil())
			})

			It("Allows removal of labels", func() {
				labels := map[string]string{
					"allowed-label-1": "val1",
					"allowed-label-2": "val2",
					"allowed-label-3": "val3",
				}
				dsi = newDSI(withLabels(labels))
				Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

				Eventually(func() error {
					var currDSI pgv1beta3.Postgresql
					if err := k8sClient.Get(ctx, types.NamespacedName{
						Namespace: dsi.GetNamespace(),
						Name:      dsi.GetName(),
					}, &currDSI); err != nil {
						return err
					}

					currDSI.Labels = map[string]string{
						"allowed-label-1": "val1",
					}

					return k8sClient.Update(ctx, &currDSI)
				}, asyncOpsTimeoutMins, 1*time.Second).Should(BeNil())
			})
		})

		Context("Invalid labels", func() {
			It("Rejects update from nil labels to reserved only labels", func() {
				dsi = newDSI(withLabels(nil))
				Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

				var err error
				Eventually(func(g Gomega) {
					var currDSI pgv1beta3.Postgresql
					err = k8sClient.Get(ctx, types.NamespacedName{
						Namespace: dsi.GetNamespace(),
						Name:      dsi.GetName(),
					}, &currDSI)
					g.Expect(err).To(BeNil(), "failed to get DSI object")

					currDSI.Labels = map[string]string{
						reservedLabelsKeys[0]: "val1",
					}

					err = k8sClient.Update(ctx, &currDSI)
					g.Expect(errors.IsInvalid(err)).To(BeTrue())
				}, asyncOpsTimeoutMins, 1*time.Second).Should(Succeed())

				Expect(err.Error()).To(ContainSubstring(reservedLabelsKeys[0]),
					"got error that doesn't mention the reserved labels while it should")
				Expect(strings.ToLower(err.Error())).To(ContainSubstring("reserved"),
					"got error that doesn't mention the word \"reserved\" while it should")
				Expect(err.Error()).To(ContainSubstring("metadata.labels"),
					"got error that doesn't mention the name of the invalid field while it should")
			})

			It("Rejects update from nil labels to reserved and valid labels", func() {
				dsi = newDSI(withLabels(nil))
				Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

				var err error
				Eventually(func(g Gomega) {
					var currDSI pgv1beta3.Postgresql
					err = k8sClient.Get(ctx, types.NamespacedName{
						Namespace: dsi.GetNamespace(),
						Name:      dsi.GetName(),
					}, &currDSI)
					g.Expect(err).To(BeNil(), "failed to get DSI object")

					currDSI.Labels = map[string]string{
						reservedLabelsKeys[1]: "val1",
						"allowed-label":       "val2",
					}

					err = k8sClient.Update(ctx, &currDSI)
					g.Expect(errors.IsInvalid(err)).To(BeTrue())
				}, asyncOpsTimeoutMins, 1*time.Second).Should(Succeed())

				Expect(err.Error()).To(ContainSubstring(reservedLabelsKeys[1]),
					"got error that doesn't mention the reserved labels while it should")
				Expect(strings.ToLower(err.Error())).To(ContainSubstring("reserved"),
					"got error that doesn't mention the word \"reserved\" while it should")
				Expect(err.Error()).To(ContainSubstring("metadata.labels"),
					"got error that doesn't mention the name of the invalid field while it should")
			})

			It("Rejects update from empty labels to reserved only labels", func() {
				dsi = newDSI(withLabels(map[string]string{}))
				Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

				var err error
				Eventually(func(g Gomega) {
					var currDSI pgv1beta3.Postgresql
					err = k8sClient.Get(ctx, types.NamespacedName{
						Namespace: dsi.GetNamespace(),
						Name:      dsi.GetName(),
					}, &currDSI)
					g.Expect(err).To(BeNil(), "failed to get DSI object")

					currDSI.Labels = map[string]string{
						reservedLabelsKeys[2]: "val1",
						reservedLabelsKeys[3]: "val1",
					}

					err = k8sClient.Update(ctx, &currDSI)
					g.Expect(errors.IsInvalid(err)).To(BeTrue())
				}, asyncOpsTimeoutMins, 1*time.Second).Should(Succeed())

				Expect(err.Error()).To(ContainSubstring(reservedLabelsKeys[2]),
					"got error that doesn't mention the reserved labels while it should")
				Expect(err.Error()).To(ContainSubstring(reservedLabelsKeys[3]),
					"got error that doesn't mention the reserved labels while it should")
				Expect(strings.ToLower(err.Error())).To(ContainSubstring("reserved"),
					"got error that doesn't mention the word \"reserved\" while it should")
				Expect(err.Error()).To(ContainSubstring("metadata.labels"),
					"got error that doesn't mention the name of the invalid field while it should")
			})

			It("Rejects update from empty labels to reserved and valid labels", func() {
				dsi = newDSI(withLabels(map[string]string{}))
				Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

				var err error
				Eventually(func(g Gomega) {
					var currDSI pgv1beta3.Postgresql
					err = k8sClient.Get(ctx, types.NamespacedName{
						Namespace: dsi.GetNamespace(),
						Name:      dsi.GetName(),
					}, &currDSI)
					g.Expect(err).To(BeNil(), "failed to get DSI object")

					currDSI.Labels = map[string]string{
						reservedLabelsKeys[1]: "val1",
						"allowed-label-1":     "val2",
						"allowed-label-2":     "val3",
					}

					err = k8sClient.Update(ctx, &currDSI)
					g.Expect(errors.IsInvalid(err)).To(BeTrue())
				}, asyncOpsTimeoutMins, 1*time.Second).Should(Succeed())

				Expect(err.Error()).To(ContainSubstring(reservedLabelsKeys[1]),
					"got error that doesn't mention the reserved labels while it should")
				Expect(strings.ToLower(err.Error())).To(ContainSubstring("reserved"),
					"got error that doesn't mention the word \"reserved\" while it should")
				Expect(err.Error()).To(ContainSubstring("metadata.labels"),
					"got error that doesn't mention the name of the invalid field while it should")
			})

			It("Rejects update from valid labels to reserved labels only", func() {
				labels := map[string]string{
					"allowed-label-1": "val1",
					"allowed-label-2": "val2",
				}
				dsi = newDSI(withLabels(labels))
				Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

				var err error
				Eventually(func(g Gomega) {
					var currDSI pgv1beta3.Postgresql
					err = k8sClient.Get(ctx, types.NamespacedName{
						Namespace: dsi.GetNamespace(),
						Name:      dsi.GetName(),
					}, &currDSI)
					g.Expect(err).To(BeNil(), "failed to get DSI object")

					currDSI.Labels = map[string]string{
						reservedLabelsKeys[0]: "val2",
						reservedLabelsKeys[2]: "val2",
					}

					err = k8sClient.Update(ctx, &currDSI)
					g.Expect(errors.IsInvalid(err)).To(BeTrue())
				}, asyncOpsTimeoutMins, 1*time.Second).Should(Succeed())

				Expect(err.Error()).To(ContainSubstring(reservedLabelsKeys[0]),
					"got error that doesn't mention the reserved labels while it should")
				Expect(err.Error()).To(ContainSubstring(reservedLabelsKeys[2]),
					"got error that doesn't mention the reserved labels while it should")
				Expect(strings.ToLower(err.Error())).To(ContainSubstring("reserved"),
					"got error that doesn't mention the word \"reserved\" while it should")
				Expect(err.Error()).To(ContainSubstring("metadata.labels"),
					"got error that doesn't mention the name of the invalid field while it should")
			})

			It("Rejects update from valid labels to reserved and valid labels", func() {
				labels := map[string]string{
					"allowed-label-1": "val1",
				}
				dsi = newDSI(withLabels(labels))
				Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

				var err error
				Eventually(func(g Gomega) {
					var currDSI pgv1beta3.Postgresql
					err = k8sClient.Get(ctx, types.NamespacedName{
						Namespace: dsi.GetNamespace(),
						Name:      dsi.GetName(),
					}, &currDSI)
					g.Expect(err).To(BeNil(), "failed to get DSI object")

					currDSI.Labels = map[string]string{
						"allowed-label-1":     "val1",
						reservedLabelsKeys[0]: "val2",
					}

					err = k8sClient.Update(ctx, &currDSI)
					g.Expect(errors.IsInvalid(err)).To(BeTrue())
				}, asyncOpsTimeoutMins, 1*time.Second).Should(Succeed())

				Expect(err.Error()).To(ContainSubstring(reservedLabelsKeys[0]),
					"got error that doesn't mention the reserved labels while it should")
				Expect(strings.ToLower(err.Error())).To(ContainSubstring("reserved"),
					"got error that doesn't mention the word \"reserved\" while it should")
				Expect(err.Error()).To(ContainSubstring("metadata.labels"),
					"got error that doesn't mention the name of the invalid field while it should")
			})
		})
	})

	Context("Test validation of creation with invalid storage size, name length and labels "+
		"together", func() {
		It("Rejects a DSI with storage of 1k, name longer than the max by two and a reserved label",
			func() {
				labels := map[string]string{
					reservedLabelsKeys[3]: "val1",
					"allowed-label":       "val2",
				}
				dsi := newDSI(withNameOfLength(pgv1beta3.MaxNameLengthChars+2),
					withStorageSize("1k"),
					withLabels(labels))
				err := k8sClient.Create(ctx, dsi)
				Expect(errors.IsInvalid(err)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("metadata.name"),
					"error message doesn't mention invalid field name")
				Expect(err.Error()).To(ContainSubstring(fmt.Sprint(pgv1beta3.MaxNameLengthChars+2)),
					"error message doesn't mention the actual name length")
				Expect(err.Error()).To(ContainSubstring(fmt.Sprint(pgv1beta3.MaxNameLengthChars)),
					"error message doesn't mention the maximum name length")
				Expect(err.Error()).To(ContainSubstring("spec.volumeSize"),
					"error message doesn't mention name of the invalid field")
				Expect(err.Error()).To(ContainSubstring(pgv1beta3.MinVolumeSize),
					"error message doesn't mention the minimum storage size")
				Expect(err.Error()).To(ContainSubstring("1k"),
					"error message doesn't mention the specified storage size")
				Expect(err.Error()).To(ContainSubstring(reservedLabelsKeys[3]),
					"error message doesn't mention the reserved labels")
				Expect(strings.ToLower(err.Error())).To(ContainSubstring("reserved"),
					"error message doesn't mention the word \"reserved\"")
				Expect(err.Error()).To(ContainSubstring("metadata.labels"),
					"error message doesn't mention invalid field name")
			})
	})

	Context("Storage size update validation", func() {
		It("Allows update with no adjustment to volume size", func() {
			dsi := newDSI(withName("dsi-1gi-nochange-pass"),
				withStorageSize("1Gi"),
				withReplicas(1))
			err := k8sClient.Create(ctx, dsi)
			Expect(err).NotTo(HaveOccurred())

			// Update the replicas from 1 to 3 because we just want to update something different
			// than the resources.
			Eventually(func() error {
				var currDSI pgv1beta3.Postgresql
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: dsi.GetNamespace(),
					Name:      dsi.GetName(),
				}, &currDSI); err != nil {
					return err
				}

				currDSI.Spec.Replicas = pointer.Int32(3)

				return k8sClient.Update(ctx, &currDSI)
			}, asyncOpsTimeoutMins, 1*time.Second).Should(BeNil())
			Expect(err).NotTo(HaveOccurred())
		})

		It("Rejects scaling a DSI volume from 2Gi to 1Gi", func() {
			var err error
			dsi := newDSI(withName("dsi-2-to-1gi-fail"), withStorageSize("2Gi"))
			err = k8sClient.Create(ctx, dsi)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func(g Gomega) {
				var currDSI pgv1beta3.Postgresql
				err = k8sClient.Get(ctx, types.NamespacedName{
					Namespace: dsi.GetNamespace(),
					Name:      dsi.GetName(),
				}, &currDSI)
				g.Expect(err).To(BeNil(), "failed to get DSI object")

				currDSI.Spec.VolumeSize = resource.MustParse("1Gi")

				err = k8sClient.Update(ctx, &currDSI)
				g.Expect(errors.IsInvalid(err)).To(BeTrue())
			}, asyncOpsTimeoutMins, 1*time.Second).Should(Succeed())

			Expect(err.Error()).To(ContainSubstring("spec.volumeSize"),
				"error message doesn't mention name of the invalid field")
			Expect(err.Error()).To(ContainSubstring("2Gi"),
				"error message doesn't mention the current volume size")
			Expect(err.Error()).To(ContainSubstring("1Gi"),
				"error message doesn't mention the new volume size")
		})

		It("Rejects scaling a DSI volume from 2Gi to 3k", func() {
			var err error
			dsi := newDSI(withName("dsi-2-to-3k-fail"), withStorageSize("2Gi"))
			err = k8sClient.Create(ctx, dsi)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func(g Gomega) {
				var currDSI pgv1beta3.Postgresql
				err = k8sClient.Get(ctx, types.NamespacedName{
					Namespace: dsi.GetNamespace(),
					Name:      dsi.GetName(),
				}, &currDSI)
				g.Expect(err).To(BeNil(), "failed to get DSI object")

				currDSI.Spec.VolumeSize = resource.MustParse("3k")

				err = k8sClient.Update(ctx, &currDSI)
				g.Expect(errors.IsInvalid(err)).To(BeTrue())
			}, asyncOpsTimeoutMins, 1*time.Second).Should(Succeed())

			Expect(err.Error()).To(ContainSubstring("spec.volumeSize"),
				"error message doesn't mention name of the invalid field")
			Expect(err.Error()).To(ContainSubstring("2Gi"),
				"error message doesn't mention the current volume size")
			Expect(err.Error()).To(ContainSubstring("3k"),
				"error message doesn't mention the new volume size")
		})
	})

	Context("Update validation with storage size and labels both invalid", func() {
		var dsi *pgv1beta3.Postgresql

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, dsi)).To(Succeed(), "failed to delete DSI after test")
		})

		It("Rejects update of DSI to reserved labels and smaller volume", func() {
			labels := map[string]string{
				"allowed-label-1": "val1",
				"allowed-label-2": "val2",
			}
			dsi = newDSI(withStorageSize("4Gi"), withLabels(labels))
			Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

			var err error
			Eventually(func(g Gomega) {
				var currDSI pgv1beta3.Postgresql
				err = k8sClient.Get(ctx, types.NamespacedName{
					Namespace: dsi.GetNamespace(),
					Name:      dsi.GetName(),
				}, &currDSI)
				g.Expect(err).To(BeNil(), "failed to get DSI object")

				currDSI.Spec.VolumeSize = resource.MustParse("1Gi")
				currDSI.Labels = map[string]string{
					"allowed-label-1":     "val1",
					reservedLabelsKeys[2]: "val2",
				}

				err = k8sClient.Update(ctx, &currDSI)
				g.Expect(errors.IsInvalid(err)).To(BeTrue())
			}, asyncOpsTimeoutMins, 1*time.Second).Should(Succeed())

			Expect(err.Error()).To(ContainSubstring("spec.volumeSize"),
				"error message doesn't mention name of the invalid field")
			Expect(err.Error()).To(ContainSubstring("4Gi"),
				"error message doesn't mention the current volume size")
			Expect(err.Error()).To(ContainSubstring("1Gi"),
				"error message doesn't mention the new volume size")
			Expect(err.Error()).To(ContainSubstring(reservedLabelsKeys[2]),
				"error message doesn't mention the reserved labels")
			Expect(strings.ToLower(err.Error())).To(ContainSubstring("reserved"),
				"error message doesn't mention the word \"reserved\"")
			Expect(err.Error()).To(ContainSubstring("metadata.labels"),
				"error message doesn't mention name of the invalid field")
		})
	})
})

const (
	maxLocksPerTransactionDefault               int = 64
	maxLocksPerTransactionDefaultWithMobilityDB int = 100
)

var _ = Describe("Defaulting webhook", func() {
	Context("Defaulting of configuration", func() {
		var dsi *pgv1beta3.Postgresql

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, dsi)).To(Succeed(), "failed to delete DSI after test")
		})

		It("Applies defaulting when maxLocksPerTransaction is not set", func() {
			dsi = newDSI(withName("dsi-defaulting-with-default-config"))
			Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

			dsiNSN := types.NamespacedName{Namespace: dsi.Namespace, Name: dsi.Name}
			Expect(k8sClient.Get(ctx, dsiNSN, dsi)).To(Succeed())
			Expect(*dsi.Spec.Parameters.MaxLocksPerTransaction).
				To(Equal(maxLocksPerTransactionDefault))
		})

		It("Does not apply defaulting when maxLocksPerTransaction is set", func() {
			maxLocksPerTransaction := 150
			customConfig := pgv1beta3.PostgresqlParameters{
				MaxLocksPerTransaction: &maxLocksPerTransaction,
			}

			dsi = newDSI(withName("dsi-defaulting-with-custom-config"),
				withCustomConfiguration(customConfig))

			Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

			dsiNSN := types.NamespacedName{Namespace: dsi.Namespace, Name: dsi.Name}
			Expect(k8sClient.Get(ctx, dsiNSN, dsi)).To(Succeed())
			Expect(*dsi.Spec.Parameters.MaxLocksPerTransaction).
				To(Equal(maxLocksPerTransaction))
		})

		It("Applies defaulting when maxLocksPerTransaction is not set and MobilityDB is defined", func() {
			dsi = newDSI(withName("dsi-defaulting-with-mobilitydb"),
				withExtensions("MobilityDB"))

			Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

			dsiNSN := types.NamespacedName{Namespace: dsi.Namespace, Name: dsi.Name}
			Expect(k8sClient.Get(ctx, dsiNSN, dsi)).To(Succeed())
			Expect(*dsi.Spec.Parameters.MaxLocksPerTransaction).
				To(Equal(maxLocksPerTransactionDefaultWithMobilityDB))
		})

		It("Does not apply defaulting when maxLocksPerTransaction is set and MobilityDB is defined", func() {
			dsi = newDSI(withName("dsi-defaulting-with-custom-config-and-mobilitydb"),
				withExtensions("MobilityDB"))

			Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

			dsiNSN := types.NamespacedName{Namespace: dsi.Namespace, Name: dsi.Name}
			Expect(k8sClient.Get(ctx, dsiNSN, dsi)).To(Succeed())
			Expect(*dsi.Spec.Parameters.MaxLocksPerTransaction).
				To(Equal(maxLocksPerTransactionDefaultWithMobilityDB))
		})
	})

	Context("Creation of an invalid PostgreSQL instance", func() {
		It("Fails if the custom configuration is out of range", func() {
			maxLocksPerTransaction := 5
			customConfig := pgv1beta3.PostgresqlParameters{
				MaxLocksPerTransaction: &maxLocksPerTransaction,
			}

			dsi := newDSI(withName("dsi-with-custom-config-fail"),
				withCustomConfiguration(customConfig))

			err := k8sClient.Create(ctx, dsi)
			Expect(errors.IsInvalid(err)).To(BeTrue())
		})
	})

	Context("Update of custom configuration", func() {
		var dsi *pgv1beta3.Postgresql

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, dsi)).To(Succeed(), "failed to delete DSI after test")
		})

		It("Allows update with no adjustments to the PostgreSQL configuration", func() {
			dsi = newDSI(withName("dsi-default-nochange-pass"))
			Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

			Eventually(func() error {
				var currDSI pgv1beta3.Postgresql
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: dsi.GetNamespace(),
					Name:      dsi.GetName(),
				}, &currDSI); err != nil {
					return err
				}

				// Update the replicas from 1 to 3 because we just want to update something different
				// than the resources.
				currDSI.Spec.Replicas = pointer.Int32(3)
				return k8sClient.Update(ctx, &currDSI)
			}, asyncOpsTimeoutMins, 1*time.Second).Should(BeNil())
		})

		It("Allows a DSI without default configuration being updated", func() {
			dsi = newDSI(withName("dsi-default-to-custom-pass"))
			Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

			maxLocksPerTransaction := 200
			Eventually(func() error {
				var currDSI pgv1beta3.Postgresql
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: dsi.GetNamespace(),
					Name:      dsi.GetName(),
				}, &currDSI); err != nil {
					return err
				}

				currDSI.Spec.Parameters.MaxLocksPerTransaction = &maxLocksPerTransaction

				return k8sClient.Update(ctx, &currDSI)
			}, asyncOpsTimeoutMins, 1*time.Second).Should(BeNil())

			dsiNSN := types.NamespacedName{Namespace: dsi.Namespace, Name: dsi.Name}
			Expect(k8sClient.Get(ctx, dsiNSN, dsi)).To(Succeed())
			Expect(*dsi.Spec.Parameters.MaxLocksPerTransaction).
				To(Equal(maxLocksPerTransaction))
		})

		It("Allows a DSI with MaxLocksPerTransaction being updated", func() {
			maxLocksPerTransaction := 150
			customConfig := pgv1beta3.PostgresqlParameters{
				MaxLocksPerTransaction: &maxLocksPerTransaction,
			}

			dsi = newDSI(withName("dsi-150-to-200-pass"), withCustomConfiguration(customConfig))
			Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

			Eventually(func() error {
				var currDSI pgv1beta3.Postgresql
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: dsi.GetNamespace(),
					Name:      dsi.GetName(),
				}, &currDSI); err != nil {
					return err
				}

				maxLocksPerTransaction = 200
				currDSI.Spec.Parameters.MaxLocksPerTransaction = &maxLocksPerTransaction
				return k8sClient.Update(ctx, &currDSI)
			}, asyncOpsTimeoutMins, 1*time.Second).Should(BeNil())

			dsiNSN := types.NamespacedName{Namespace: dsi.Namespace, Name: dsi.Name}
			Expect(k8sClient.Get(ctx, dsiNSN, dsi)).To(Succeed())
			Expect(*dsi.Spec.Parameters.MaxLocksPerTransaction).
				To(Equal(maxLocksPerTransaction))
		})

		It("Fails if the updated custom configuration is out of range", func() {
			maxLocksPerTransaction := 150
			customConfig := pgv1beta3.PostgresqlParameters{
				MaxLocksPerTransaction: &maxLocksPerTransaction,
			}

			dsi = newDSI(withName("dsi-150-to-5-fail"), withCustomConfiguration(customConfig))
			Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

			// Minimum value for MaxLocksPerTransaction is 10
			maxLocksPerTransaction = 5

			Eventually(func(g Gomega) {
				var currDSI pgv1beta3.Postgresql
				err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: dsi.GetNamespace(),
					Name:      dsi.GetName(),
				}, &currDSI)
				g.Expect(err).To(BeNil(), "failed to get DSI object")

				currDSI.Spec.Parameters.MaxLocksPerTransaction = &maxLocksPerTransaction

				err = k8sClient.Update(ctx, &currDSI)
				g.Expect(errors.IsInvalid(err)).To(BeTrue())
			}, asyncOpsTimeoutMins, 1*time.Second).Should(Succeed())
		})

		It("Succeeds if the custom configuration is removed from the CR", func() {
			maxLocksPerTransaction := 150
			customConfig := pgv1beta3.PostgresqlParameters{
				MaxLocksPerTransaction: &maxLocksPerTransaction,
			}

			dsi = newDSI(withName("dsi-150-to-default-pass"), withCustomConfiguration(customConfig))
			Expect(k8sClient.Create(ctx, dsi)).To(Succeed())

			Eventually(func() error {
				var currDSI pgv1beta3.Postgresql
				if err := k8sClient.Get(ctx, types.NamespacedName{
					Namespace: dsi.GetNamespace(),
					Name:      dsi.GetName(),
				}, &currDSI); err != nil {
					return err
				}

				currDSI.Spec.Parameters.MaxLocksPerTransaction = nil
				return k8sClient.Update(ctx, &currDSI)
			}, asyncOpsTimeoutMins, 1*time.Second).Should(BeNil())

			dsiNSN := types.NamespacedName{Namespace: dsi.Namespace, Name: dsi.Name}
			Expect(k8sClient.Get(ctx, dsiNSN, dsi)).To(Succeed())
			Expect(*dsi.Spec.Parameters.MaxLocksPerTransaction).
				To(Equal(maxLocksPerTransactionDefault))
		})
	})
})

func withCustomConfiguration(config pgv1beta3.PostgresqlParameters) func(*pgv1beta3.Postgresql) {
	return func(dsi *pgv1beta3.Postgresql) {
		dsi.Spec.Parameters = config
	}
}

func withExtensions(extensions ...string) func(*pgv1beta3.Postgresql) {
	return func(dsi *pgv1beta3.Postgresql) {
		dsi.Spec.Extensions = extensions
	}
}

func newDSI(opts ...func(*pgv1beta3.Postgresql)) *pgv1beta3.Postgresql {
	instance, err := dsi.New(
		dataservice,
		testingNamespace,
		framework.GenerateName(
			instanceNamePrefix, GinkgoParallelProcess(), suffixLength),
		1,
	)
	Expect(err).To(BeNil(), "failed to generate new DSI resource")

	dsi, ok := instance.GetClientObject().(*pgv1beta3.Postgresql)
	Expect(ok).To(BeTrue(),
		"failed to cast object interface to PostgreSQL struct")

	for _, applyOption := range opts {
		applyOption(dsi)
	}
	return dsi
}

func withCPURequest(cpu string) func(*pgv1beta3.Postgresql) {
	return func(pg *pgv1beta3.Postgresql) {
		initResourcesStructs(pg)
		pg.Spec.Resources.Requests["cpu"] = resource.MustParse(cpu)
	}
}

func withCPULimit(cpu string) func(*pgv1beta3.Postgresql) {
	return func(pg *pgv1beta3.Postgresql) {
		initResourcesStructs(pg)
		pg.Spec.Resources.Limits["cpu"] = resource.MustParse(cpu)
	}
}

func withMemoryRequest(mem string) func(*pgv1beta3.Postgresql) {
	return func(pg *pgv1beta3.Postgresql) {
		initResourcesStructs(pg)
		pg.Spec.Resources.Requests["memory"] = resource.MustParse(mem)
	}
}

func withMemoryLimit(mem string) func(*pgv1beta3.Postgresql) {
	return func(pg *pgv1beta3.Postgresql) {
		initResourcesStructs(pg)
		pg.Spec.Resources.Limits["memory"] = resource.MustParse(mem)
	}
}

func initResourcesStructs(pg *pgv1beta3.Postgresql) {
	if pg.Spec.Resources == nil {
		pg.Spec.Resources = &corev1.ResourceRequirements{}
	}
	if pg.Spec.Resources.Requests == nil {
		pg.Spec.Resources.Requests = corev1.ResourceList{}
	}
	if pg.Spec.Resources.Limits == nil {
		pg.Spec.Resources.Limits = corev1.ResourceList{}
	}
}

func withName(name string) func(*pgv1beta3.Postgresql) {
	return func(dsi *pgv1beta3.Postgresql) {
		dsi.Name = name
	}
}

func withNameOfLength(length int) func(*pgv1beta3.Postgresql) {
	return func(dsi *pgv1beta3.Postgresql) {
		dsi.Name = nameOfLength(length)
	}
}

func withStorageSize(size string) func(*pgv1beta3.Postgresql) {
	return func(dsi *pgv1beta3.Postgresql) {
		dsi.Spec.VolumeSize = resource.MustParse(size)
	}
}

func withReplicas(r int32) func(*pgv1beta3.Postgresql) {
	return func(dsi *pgv1beta3.Postgresql) {
		dsi.Spec.Replicas = &r
	}
}

func withLabels(l map[string]string) func(*pgv1beta3.Postgresql) {
	return func(dsi *pgv1beta3.Postgresql) {
		dsi.Labels = l
	}
}

// nameOfLength is a testing helper that concatenates the letter 'a' `length` times to produce a
// string of length `length`. Used to produce names of arbitrary length for PostgreSQL instances in
// tests where the length of the name matters.
// Example: nameOfLength(2) will return "aa".
// TODO: Make the generated name random.
func nameOfLength(length int) string {
	name := make([]rune, length)
	for i := 0; i < length; i++ {
		name[i] = 'a'
	}
	return string(name)
}

// now returns a string representing the date and time at which its invocation occurs, sanitized by
// removing the characters that make it unusable in an API object name (so that it can be used
// in an API object name).
func now() string {
	now := time.Now().Format(time.RFC3339)

	// Replace ":" and "+" with "-" as the former are not allowed in API object names.
	now = strings.Replace(now, ":", "-", -1)
	now = strings.Replace(now, "+", "-", -1)

	// Only lowercase letters are allowed in API object names.
	return strings.ToLower(now)
}

var _ = AfterSuite(func() {
	Expect(namespace.DeleteIfAllowed(ctx, testingNamespace, k8sClient)).
		To(Succeed(), "failed to delete testing namespace")
	cancel()
})
