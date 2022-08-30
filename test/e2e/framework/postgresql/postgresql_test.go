package postgresql_test

import (
	"context"
	"fmt"
	"sort"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/anynines/a8s-deployment/test/e2e/framework/postgresql"
	"github.com/anynines/postgresql-operator/api/v1alpha1"
)

// TODO: Test failure cases (e.g. the K8s API calls return an error). Not already done because the
// fake client doesn't support injection of errors.

func TestPodsRetrievalHappyPaths(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		dsi        *postgresql.Postgresql
		dsiPods    []client.Object
		nonDSIPods []client.Object
	}{
		"return_nil_when_there_are_0_pods": {
			dsi:        postgresql.New("ns0", "pg0", 3),
			dsiPods:    nil,
			nonDSIPods: nil,
		},

		"return_nil_when_all_pods_are_not_of_a_dsi": {
			dsi:     postgresql.New("ns0", "pg0", 3),
			dsiPods: nil,
			nonDSIPods: []client.Object{
				newPod(
					withName("p0"),
					withNamespace("ns0"),
					withLabels(map[string]string{"foo": "bar"}),
				),
			},
		},

		"return_nil_when_all_pods_are_of_a_dsi_with_same_kind_and_name_but_different_namespace": {
			dsi:     postgresql.New("ns0", "pg0", 3),
			dsiPods: nil,
			nonDSIPods: []client.Object{
				newPod(
					withName("p0"),
					withNamespace("ns1"),
					withLabels(map[string]string{
						v1alpha1.DSINameLabelKey:         "pg0",
						v1alpha1.DSIKindLabelKey:         "Postgresql",
						v1alpha1.DSIGroupLabelKey:        "postgresql.anynines.com",
						v1alpha1.ReplicationRoleLabelKey: "master",
					}),
				),
			},
		},

		"return_nil_when_all_pods_are_of_a_dsi_with_same_kind_and_namespace_but_different_name": {
			dsi:     postgresql.New("ns0", "pg0", 3),
			dsiPods: nil,
			nonDSIPods: []client.Object{
				newPod(
					withName("p0"),
					withNamespace("ns0"),
					withLabels(map[string]string{
						v1alpha1.DSINameLabelKey:         "pg1",
						v1alpha1.DSIKindLabelKey:         "Postgresql",
						v1alpha1.DSIGroupLabelKey:        "postgresql.anynines.com",
						v1alpha1.ReplicationRoleLabelKey: "replica",
					}),
				),
			},
		},

		"return_nil_when_all_pods_are_of_a_dsi_of_another_kind": {
			dsi:     postgresql.New("ns0", "pg0", 3),
			dsiPods: nil,
			nonDSIPods: []client.Object{
				newPod(
					withName("p0"),
					withNamespace("ns0"),
					withLabels(map[string]string{
						v1alpha1.DSINameLabelKey:  "r1",
						v1alpha1.DSIKindLabelKey:  "Redis",
						v1alpha1.DSIGroupLabelKey: "redis.anynines.com",
					}),
				),
			},
		},

		"return_the_dsi_pods_when_all_pods_belong_to_the_dsi": {
			dsi: postgresql.New("ns0", "pg0", 3),
			dsiPods: []client.Object{
				newPod(
					withName("p0"),
					withNamespace("ns0"),
					withLabels(map[string]string{
						v1alpha1.DSINameLabelKey:         "pg0",
						v1alpha1.DSIKindLabelKey:         "Postgresql",
						v1alpha1.DSIGroupLabelKey:        "postgresql.anynines.com",
						v1alpha1.ReplicationRoleLabelKey: "master",
					}),
				),
				newPod(
					withName("p1"),
					withNamespace("ns0"),
					withLabels(map[string]string{
						v1alpha1.DSINameLabelKey:         "pg0",
						v1alpha1.DSIKindLabelKey:         "Postgresql",
						v1alpha1.DSIGroupLabelKey:        "postgresql.anynines.com",
						v1alpha1.ReplicationRoleLabelKey: "replica",
					}),
				),
				newPod(
					withName("p2"),
					withNamespace("ns0"),
					withLabels(map[string]string{
						v1alpha1.DSINameLabelKey:         "pg0",
						v1alpha1.DSIKindLabelKey:         "Postgresql",
						v1alpha1.DSIGroupLabelKey:        "postgresql.anynines.com",
						v1alpha1.ReplicationRoleLabelKey: "replica",
					}),
				),
			},
			nonDSIPods: nil,
		},

		"return_the_dsi_pods_when_there_are_less_pods_than_replicas": {
			dsi: postgresql.New("ns0", "pg0", 3),
			dsiPods: []client.Object{
				newPod(
					withName("p0"),
					withNamespace("ns0"),
					withLabels(map[string]string{
						v1alpha1.DSINameLabelKey:         "pg0",
						v1alpha1.DSIKindLabelKey:         "Postgresql",
						v1alpha1.DSIGroupLabelKey:        "postgresql.anynines.com",
						v1alpha1.ReplicationRoleLabelKey: "master",
					}),
				),
				newPod(
					withName("p1"),
					withNamespace("ns0"),
					withLabels(map[string]string{
						v1alpha1.DSINameLabelKey:         "pg0",
						v1alpha1.DSIKindLabelKey:         "Postgresql",
						v1alpha1.DSIGroupLabelKey:        "postgresql.anynines.com",
						v1alpha1.ReplicationRoleLabelKey: "replica",
					}),
				),
			},
			nonDSIPods: nil,
		},

		"return_only_the_dsi_pods_when_some_pods_are_not_of_a_dsi": {
			dsi: postgresql.New("ns0", "pg0", 3),
			dsiPods: []client.Object{
				newPod(
					withName("p0"),
					withNamespace("ns0"),
					withLabels(map[string]string{
						v1alpha1.DSINameLabelKey:         "pg0",
						v1alpha1.DSIKindLabelKey:         "Postgresql",
						v1alpha1.DSIGroupLabelKey:        "postgresql.anynines.com",
						v1alpha1.ReplicationRoleLabelKey: "master",
					}),
				),
			},
			nonDSIPods: []client.Object{
				newPod(
					withName("p1"),
					withNamespace("ns0"),
					withLabels(map[string]string{"foo": "bar"}),
				),
			},
		},

		"return_only_the_dsi_pods_when_some_pods_are_of_a_dsi_of_a_different_kind": {
			dsi: postgresql.New("ns0", "pg0", 3),
			dsiPods: []client.Object{
				newPod(
					withName("p0"),
					withNamespace("ns0"),
					withLabels(map[string]string{
						v1alpha1.DSINameLabelKey:         "pg0",
						v1alpha1.DSIKindLabelKey:         "Postgresql",
						v1alpha1.DSIGroupLabelKey:        "postgresql.anynines.com",
						v1alpha1.ReplicationRoleLabelKey: "master",
					}),
				),
			},
			nonDSIPods: []client.Object{
				newPod(
					withName("p1"),
					withNamespace("ns0"),
					withLabels(map[string]string{
						v1alpha1.DSINameLabelKey:  "r1",
						v1alpha1.DSIKindLabelKey:  "Redis",
						v1alpha1.DSIGroupLabelKey: "redis.anynines.com",
					}),
				),
			},
		},

		"return_only_the_dsi_pods_when_some_pods_are_of_another_dsi_with_same_kind_and_namespace": {
			dsi: postgresql.New("ns0", "pg0", 3),
			dsiPods: []client.Object{
				newPod(
					withName("p0"),
					withNamespace("ns0"),
					withLabels(map[string]string{
						v1alpha1.DSINameLabelKey:         "pg0",
						v1alpha1.DSIKindLabelKey:         "Postgresql",
						v1alpha1.DSIGroupLabelKey:        "postgresql.anynines.com",
						v1alpha1.ReplicationRoleLabelKey: "master",
					}),
				),
			},
			nonDSIPods: []client.Object{
				newPod(
					withName("p1"),
					withNamespace("ns0"),
					withLabels(map[string]string{
						v1alpha1.DSINameLabelKey:         "pg1",
						v1alpha1.DSIKindLabelKey:         "Postgresql",
						v1alpha1.DSIGroupLabelKey:        "postgresql.anynines.com",
						v1alpha1.ReplicationRoleLabelKey: "master",
					}),
				),
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Rebind tc into this lexical scope. Details on the why at
			// https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
			tc := tc

			t.Parallel()

			// Generate a fake K8s client pre-populated with the pods of the test case.
			k8sClient := fake.NewClientBuilder().
				WithObjects(tc.dsiPods...).
				WithObjects(tc.nonDSIPods...).
				Build()

			// Invoke the method under test
			gotPods, err := tc.dsi.Pods(context.Background(), k8sClient)

			if err != nil {
				t.Fatalf("Expected no error when listing DSI Pods, got: \"%v\"", err)
			}

			if !podsEqual(gotPods, tc.dsiPods) {
				t.Fatalf("Got pods don't match those that belong to the DSI"+
					"\n\n\tgot pods: %#+v\n\n\tdsi pods:  %#+v\n\n", gotPods, tc.dsiPods)
			}
		})
	}
}

func newPod(opts ...func(*corev1.Pod)) *corev1.Pod {
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod0",
			Namespace: "test-ns",
		},
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

func withName(name string) func(*corev1.Pod) {
	return func(p *corev1.Pod) {
		p.Name = name
	}
}

func withNamespace(ns string) func(*corev1.Pod) {
	return func(p *corev1.Pod) {
		p.Namespace = ns
	}
}

func withLabels(l map[string]string) func(*corev1.Pod) {
	return func(p *corev1.Pod) {
		p.Labels = l
	}
}

func podsEqual(p1 []corev1.Pod, p2 []client.Object) bool {
	if len(p1) != len(p2) {
		return false
	}

	sort.Slice(p1, func(i, j int) bool { return p1[i].Name < p1[j].Name })
	sort.Slice(p2, func(i, j int) bool { return p2[i].GetName() < p2[j].GetName() })

	for i, pod1 := range p1 {
		pod2, ok := p2[i].(*corev1.Pod)
		if !ok {
			panic(fmt.Sprintf("podsEqual invoked with element of 2nd input argument p2 of type %T "+
				", it MUST be a *corev1.Pod", p2[i]))
		}

		if !equality.Semantic.DeepEqual(pod1, *pod2) {
			return false
		}
	}

	return true
}
