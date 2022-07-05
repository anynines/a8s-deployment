package node_test

import (
	"context"
	"os"
	"sort"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/anynines/a8s-deployment/test/integration/framework/node"
)

func TestMain(m *testing.M) {
	// This suite needs to compare slices of nodes and taints without caring about the order of
	// the elements in the slices. But default comparators care about ordering. So we register
	// custom comparators that ignore the order. To do that once for the whole suite, we do it in
	// the TestMain function.
	equality.Semantic.AddFuncs(compareNodesIgnoringOrder, compareTaintsIgnoringOrder)

	os.Exit(m.Run())
}

func TestTaintAllHappyPaths(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		taintsToAdd   []v1.Taint
		inputNodes    []v1.Node
		expectedNodes []v1.Node
	}{
		"1_taint_is_added_to_1_node_with_nil_taints": {
			taintsToAdd: []v1.Taint{{Key: "key1", Value: "val1", Effect: "NoSchedule"}},
			inputNodes:  []v1.Node{newNode(withName("n1"), withTaints(nil))},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{{Key: "key1", Value: "val1", Effect: "NoSchedule"}}),
				),
			},
		},

		"3_taints_are_added_to_1_node_with_nil_taints": {
			taintsToAdd: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
				{Key: "key3", Value: "val3", Effect: "NoSchedule"},
			},
			inputNodes: []v1.Node{newNode(withName("n1"), withTaints(nil))},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
						{Key: "key3", Value: "val3", Effect: "NoSchedule"},
					}),
				),
			},
		},

		"1_taint_is_added_to_1_node_with_empty_taints": {
			taintsToAdd: []v1.Taint{{Key: "key1", Value: "val1", Effect: "NoExecute"}},
			inputNodes:  []v1.Node{newNode(withName("n1"), withTaints([]v1.Taint{}))},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{{Key: "key1", Value: "val1", Effect: "NoExecute"}}),
				),
			},
		},

		"3_taints_are_added_to_1_node_with_empty_taints": {
			taintsToAdd: []v1.Taint{
				{Key: "key1", Effect: "NoExecute"},
				{Key: "key2", Effect: "NoExecute"},
				{Key: "key3", Value: "val3", Effect: "NoSchedule"},
			},
			inputNodes: []v1.Node{newNode(withName("n1"), withTaints([]v1.Taint{}))},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Effect: "NoExecute"},
						{Key: "key2", Effect: "NoExecute"},
						{Key: "key3", Value: "val3", Effect: "NoSchedule"},
					}),
				),
			},
		},

		"1_taint_is_added_to_3_nodes_with_no_or_different_taints": {
			taintsToAdd: []v1.Taint{{Key: "key1", Value: "val1", Effect: "NoSchedule"}},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{}),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "key10", Value: "val10", Effect: "NoSchedule"},
					}),
				),
				newNode(
					withName("n3"),
					withTaints([]v1.Taint{
						{Key: "key10", Effect: "NoSchedule"},
						{Key: "key11", Value: "val11", Effect: "NoExecute"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{{Key: "key1", Value: "val1", Effect: "NoSchedule"}}),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "key10", Value: "val10", Effect: "NoSchedule"},
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
					}),
				),
				newNode(
					withName("n3"),
					withTaints([]v1.Taint{
						{Key: "key10", Effect: "NoSchedule"},
						{Key: "key11", Value: "val11", Effect: "NoExecute"},
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
					}),
				),
			},
		},

		"2_taints_are_added_to_3_nodes_with_no_or_different_taints": {
			taintsToAdd: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints(nil),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "key10", Value: "val10", Effect: "NoSchedule"},
					}),
				),
				newNode(
					withName("n3"),
					withTaints([]v1.Taint{
						{Key: "key10", Effect: "NoSchedule"},
						{Key: "key11", Value: "val11", Effect: "NoExecute"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
					}),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "key10", Value: "val10", Effect: "NoSchedule"},
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
					}),
				),
				newNode(
					withName("n3"),
					withTaints([]v1.Taint{
						{Key: "key10", Effect: "NoSchedule"},
						{Key: "key11", Value: "val11", Effect: "NoExecute"},
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
					}),
				),
			},
		},

		"1_taint_to_1_node_that_already_has_it": {
			taintsToAdd: []v1.Taint{{Key: "key1", Value: "val1", Effect: "NoSchedule"}},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"}},
					),
				),
			},
		},

		"2_taints_to_two_nodes_that_already_have_them": {
			taintsToAdd: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
					}),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
						{Key: "key3", Value: "val3", Effect: "NoSchedule"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
					}),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
						{Key: "key3", Value: "val3", Effect: "NoSchedule"},
					}),
				),
			},
		},

		"2_taints_to_2_nodes_that_have_only_one_of_the_two_taints": {
			taintsToAdd: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
					}),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "key2", Effect: "NoExecute"},
						{Key: "key3", Value: "val3", Effect: "NoSchedule"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
					}),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
						{Key: "key3", Value: "val3", Effect: "NoSchedule"},
					}),
				),
			},
		},

		"2_taints_are_not_added_to_nodes_with_control_taint": {
			taintsToAdd: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/control-plane", Effect: "NoSchedule"},
					}),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/control-plane", Effect: "NoExecute"},
						{Key: "key3", Value: "val3", Effect: "NoSchedule"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/control-plane", Effect: "NoSchedule"},
					}),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/control-plane", Effect: "NoExecute"},
						{Key: "key3", Value: "val3", Effect: "NoSchedule"},
					}),
				),
			},
		},

		"2_taints_are_not_added_to_nodes_with_master_taint": {
			taintsToAdd: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/master", Effect: "NoSchedule"},
					}),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/master", Effect: "NoExecute"},
						{Key: "key3", Value: "val3", Effect: "NoSchedule"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/master", Effect: "NoSchedule"},
					}),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/master", Effect: "NoExecute"},
						{Key: "key3", Value: "val3", Effect: "NoSchedule"},
					}),
				),
			},
		},

		"1_taint_is_added_to_normal_nodes_but_not_to_control_plane_nodes": {
			taintsToAdd: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints(nil),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/control-plane", Effect: "NoExecute"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{{Key: "key1", Value: "val1", Effect: "NoSchedule"}}),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/control-plane", Effect: "NoExecute"},
					}),
				),
			},
		},

		"1_taint_is_added_to_normal_nodes_but_not_to_master_nodes": {
			taintsToAdd: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{}),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/master", Effect: "NoExecute"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{{Key: "key1", Value: "val1", Effect: "NoSchedule"}}),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/master", Effect: "NoExecute"},
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

			// Set up object under test
			k8sAPINodesClient := fake.
				NewSimpleClientset(&v1.NodeList{Items: tc.inputNodes}).
				CoreV1().
				Nodes()
			nodes := node.Client{
				Nodes:            k8sAPINodesClient,
				MasterNodeTaints: node.MasterTaintKeys,
			}

			// Invoke method under test
			if err := nodes.TaintAll(context.Background(), tc.taintsToAdd); err != nil {
				t.Fatal("Expected no error when invoking TaintAll, got:", err)
			}

			// Get the nodes after invoking the method under test
			gotNodesList, err := k8sAPINodesClient.List(context.Background(), metav1.ListOptions{})
			if err != nil {
				t.Fatal("Expected no error when listing nodes, got:", err)
			}
			gotNodes := gotNodesList.Items

			// Compare the expected nodes with the got ones to assess the test outcome
			if !equality.Semantic.DeepEqual(tc.expectedNodes, gotNodes) {
				t.Fatalf("Expected nodes don't match got ones\n\n\texpected: %#+v\n\n\tgot:"+
					" %#+v\n\n", tc.expectedNodes, gotNodes)
			}
		})
	}
}

func TestUntaintAllHappyPaths(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		taintsToRemove []v1.Taint
		inputNodes     []v1.Node
		expectedNodes  []v1.Node
	}{
		"1_taint_removed_from_1_node_with_no_other_taints": {
			taintsToRemove: []v1.Taint{{Key: "key1", Value: "val1", Effect: "NoSchedule"}},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{{Key: "key1", Value: "val1", Effect: "NoSchedule"}}),
				),
			},
			expectedNodes: []v1.Node{newNode(withName("n1"), withTaints(nil))},
		},

		"3_taints_removed_from_1_node_with_no_other_taints": {
			taintsToRemove: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
				{Key: "key3", Value: "val3", Effect: "NoSchedule"},
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
						{Key: "key3", Value: "val3", Effect: "NoSchedule"},
					}),
				),
			},
			expectedNodes: []v1.Node{newNode(withName("n1"), withTaints(nil))},
		},

		"1_taint_removed_from_1_node_that_has_other_taints_as_well": {
			taintsToRemove: []v1.Taint{{Key: "key1", Value: "val1", Effect: "NoSchedule"}},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
					},
					),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{{Key: "key2", Effect: "NoExecute"}}),
				),
			},
		},

		"2_taints_removed_from_1_node_that_has_other_taints_as_well": {
			taintsToRemove: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
						{Key: "key3", Effect: "NoSchedule"},
					},
					),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{{Key: "key3", Effect: "NoSchedule"}}),
				),
			},
		},

		"2_taints_removed_from_2_nodes_that_have_no_other_taints": {
			taintsToRemove: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
					}),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(withName("n1"), withTaints(nil)),
				newNode(withName("n2"), withTaints(nil)),
			},
		},

		"2_taints_removed_from_2_nodes_1_with_other_taints_as_well": {
			taintsToRemove: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
					}),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
						{Key: "key3", Effect: "NoSchedule"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(withName("n1"), withTaints(nil)),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{{Key: "key3", Effect: "NoSchedule"}})),
			},
		},

		"1_node_with_taints_but_not_the_ones_to_remove_is_left_unchanged": {
			taintsToRemove: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key10", Value: "val10", Effect: "NoSchedule"},
						{Key: "key20", Effect: "NoExecute"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key10", Value: "val10", Effect: "NoSchedule"},
						{Key: "key20", Effect: "NoExecute"},
					}),
				),
			},
		},

		"2_taints_are_removed_from_nodes_that_have_them_but_other_nodes_are_unchanged": {
			taintsToRemove: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
					}),
				),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{{Key: "key3", Value: "val3", Effect: "NoSchedule"}}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(withName("n1"), withTaints(nil)),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{{Key: "key3", Value: "val3", Effect: "NoSchedule"}}),
				),
			},
		},

		"only_the_taints_to_remove_that_a_node_has_are_removed": {
			taintsToRemove: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key3", Effect: "NoSchedule"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{{Key: "key3", Effect: "NoSchedule"}}),
				),
			},
		},

		"when_removing_2_taints_from_4_nodes_only_the_taints_that_the_nodes_have_are_removed": {
			taintsToRemove: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
			},
			inputNodes: []v1.Node{
				newNode(withName("n1"), withTaints(nil)),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key3", Effect: "NoSchedule"},
					}),
				),
				newNode(
					withName("n3"),
					withTaints([]v1.Taint{{Key: "key3", Effect: "NoSchedule"}}),
				),
				newNode(
					withName("n4"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(withName("n1"), withTaints(nil)),
				newNode(
					withName("n2"),
					withTaints([]v1.Taint{{Key: "key3", Effect: "NoSchedule"}}),
				),
				newNode(
					withName("n3"),
					withTaints([]v1.Taint{{Key: "key3", Effect: "NoSchedule"}}),
				),
				newNode(withName("n4"), withTaints(nil)),
			},
		},

		"2_taints_are_removed_from_a_node_with_master_taints": {
			taintsToRemove: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
						{Key: "node-role.kubernetes.io/master", Effect: "NoExecute"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/master", Effect: "NoExecute"},
					}),
				),
			},
		},

		"2_taints_are_removed_from_a_node_with_control_plane_taints": {
			taintsToRemove: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "key1", Value: "val1", Effect: "NoSchedule"},
						{Key: "key2", Effect: "NoExecute"},
						{Key: "node-role.kubernetes.io/control-plane", Effect: "NoExecute"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/control-plane", Effect: "NoExecute"},
					}),
				),
			},
		},

		"1_node_with_nil_taints_is_left_unchanged": {
			taintsToRemove: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
			},
			inputNodes:    []v1.Node{newNode(withName("n1"), withTaints(nil))},
			expectedNodes: []v1.Node{newNode(withName("n1"), withTaints(nil))},
		},

		"1_node_with_empty_taints_is_left_unchanged": {
			taintsToRemove: []v1.Taint{
				{Key: "key1", Value: "val1", Effect: "NoSchedule"},
				{Key: "key2", Effect: "NoExecute"},
			},
			inputNodes:    []v1.Node{newNode(withName("n1"), withTaints([]v1.Taint{}))},
			expectedNodes: []v1.Node{newNode(withName("n1"), withTaints([]v1.Taint{}))},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Rebind tc into this lexical scope. Details on the why at
			// https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
			tc := tc

			t.Parallel()

			// Set up object under test
			k8sAPINodesClient := fake.
				NewSimpleClientset(&v1.NodeList{Items: tc.inputNodes}).
				CoreV1().
				Nodes()
			nodes := node.Client{
				Nodes:            k8sAPINodesClient,
				MasterNodeTaints: node.MasterTaintKeys,
			}

			// Invoke method under test
			if err := nodes.UntaintAll(context.Background(), tc.taintsToRemove); err != nil {
				t.Fatal("Expected no error when invoking UntaintAll, got:", err)
			}

			// Get the nodes after invoking the method under test
			gotNodesList, err := k8sAPINodesClient.List(context.Background(), metav1.ListOptions{})
			if err != nil {
				t.Fatal("Expected no error when listing nodes, got:", err)
			}
			gotNodes := gotNodesList.Items

			// Compare the expected nodes with the got ones to assess the test outcome
			if !equality.Semantic.DeepEqual(tc.expectedNodes, gotNodes) {
				t.Fatalf("Expected nodes don't match got ones\n\n\texpected: %#+v\n\n\tgot:"+
					" %#+v\n\n", tc.expectedNodes, gotNodes)
			}
		})
	}
}

func newNode(opts ...func(*v1.Node)) v1.Node {
	n := v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
		},
	}

	for _, opt := range opts {
		opt(&n)
	}

	return n
}

func withName(name string) func(*v1.Node) {
	return func(n *v1.Node) {
		n.Name = name
	}
}

func withTaints(t []v1.Taint) func(*v1.Node) {
	return func(n *v1.Node) {
		n.Spec.Taints = t
	}
}

func compareNodesIgnoringOrder(n1, n2 []v1.Node) bool {
	if len(n1) != len(n2) {
		return false
	}

	sort.Slice(n1, cmpByName(n1))
	sort.Slice(n2, cmpByName(n2))

	for i := range n1 {
		if !equality.Semantic.DeepEqual(n1[i], n2[i]) {
			return false
		}
	}

	return true
}

func compareTaintsIgnoringOrder(t1, t2 []v1.Taint) bool {
	if len(t1) != len(t2) {
		return false
	}

	sort.Slice(t1, cmpByKey(t1))
	sort.Slice(t2, cmpByKey(t2))

	for i := range t1 {
		if !equality.Semantic.DeepEqual(t1[i], t2[i]) {
			return false
		}
	}

	return true
}

func cmpByName(n []v1.Node) func(i, j int) bool {
	return func(i, j int) bool {
		return n[i].Name < n[j].Name
	}
}

func cmpByKey(t []v1.Taint) func(i, j int) bool {
	return func(i, j int) bool {
		return t[i].Key < t[j].Key
	}
}
