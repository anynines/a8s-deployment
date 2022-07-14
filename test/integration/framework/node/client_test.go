package node_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stest "k8s.io/client-go/testing"

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

func TestLabelAllHappyPaths(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		labelsToAdd   map[string]string
		inputNodes    []v1.Node
		expectedNodes []v1.Node
	}{
		"1_label_is_added_to_1_node_with_nil_labels": {
			labelsToAdd: map[string]string{"a8s.key1": "val1"},
			inputNodes:  []v1.Node{newNode(withName("n1"), withLabels(nil))},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{"a8s.key1": "val1"}),
				),
			},
		},

		"2_labels_are_added_to_1_node_with_empty_labels": {
			labelsToAdd: map[string]string{
				"a8s.key1": "val1",
				"a8s.key2": "val2",
			},
			inputNodes: []v1.Node{newNode(withName("n1"), withLabels(map[string]string{}))},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"a8s.key2": "val2",
					}),
				),
			},
		},

		"1_label_is_added_to_3_nodes_with_no_or_different_labels": {
			labelsToAdd: map[string]string{"a8s.key1": "val1"},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{"key10": "val10"}),
				),
				newNode(
					withName("n3"),
					withLabels(map[string]string{
						"key10": "val10",
						"key20": "val20",
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{"a8s.key1": "val1"}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{
						"key10":    "val10",
						"a8s.key1": "val1",
					}),
				),
				newNode(
					withName("n3"),
					withLabels(map[string]string{
						"key10":    "val10",
						"key20":    "val20",
						"a8s.key1": "val1",
					}),
				),
			},
		},

		"2_labels_are_added_to_3_nodes_with_no_or_different_labels": {
			labelsToAdd: map[string]string{
				"a8s.key1": "val1",
				"a8s.key2": "val2",
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(nil),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{"key10": "val10"}),
				),
				newNode(
					withName("n3"),
					withLabels(map[string]string{
						"key10": "val10",
						"key20": "val20",
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"a8s.key2": "val2",
					}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{
						"key10":    "val10",
						"a8s.key1": "val1",
						"a8s.key2": "val2",
					}),
				),
				newNode(
					withName("n3"),
					withLabels(map[string]string{
						"key10":    "val10",
						"key20":    "val20",
						"a8s.key1": "val1",
						"a8s.key2": "val2",
					}),
				),
			},
		},

		"2_labels_to_2_nodes_that_already_have_them": {
			labelsToAdd: map[string]string{
				"a8s.key1": "val1",
				"a8s.key2": "val2",
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"a8s.key2": "val2",
					}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"a8s.key2": "val2",
						"key10":    "val10",
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"a8s.key2": "val2",
					}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"a8s.key2": "val2",
						"key10":    "val10",
					}),
				),
			},
		},

		"2_labels_to_2_nodes_that_have_only_1_of_the_2_labels": {
			labelsToAdd: map[string]string{
				"a8s.key1": "val1",
				"a8s.key2": "val2",
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{"a8s.key1": "val1"}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{
						"a8s.key2": "val2",
						"key10":    "val10",
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"a8s.key2": "val2",
					}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"a8s.key2": "val2",
						"key10":    "val10",
					}),
				),
			},
		},

		"2_labels_are_not_added_to_nodes_with_control_taint": {
			labelsToAdd: map[string]string{
				"a8s.key1": "val1",
				"a8s.key2": "val2",
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(nil),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/control-plane", Effect: "NoSchedule"},
					}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{"key10": "val10"}),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/control-plane", Effect: "NoExecute"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(nil),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/control-plane", Effect: "NoSchedule"},
					}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{"key10": "val10"}),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/control-plane", Effect: "NoExecute"},
					}),
				),
			},
		},

		"1_label_is_not_added_to_nodes_with_master_taint": {
			labelsToAdd: map[string]string{"a8s.key1": "val1"},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(nil),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/master", Effect: "NoSchedule"},
					}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{"key10": "val10"}),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/master", Effect: "NoExecute"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(nil),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/master", Effect: "NoSchedule"},
					}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{"key10": "val10"}),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/master", Effect: "NoExecute"},
					}),
				),
			},
		},

		"1_label_is_added_to_normal_nodes_but_not_to_control_plane_nodes": {
			labelsToAdd: map[string]string{"a8s.key1": "val1"},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{"key10": "val10"}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{"key10": "val10"}),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/control-plane", Effect: "NoExecute"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"key10":    "val10",
						"a8s.key1": "val1",
					}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{"key10": "val10"}),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/control-plane", Effect: "NoExecute"},
					}),
				),
			},
		},

		"1_label_is_added_to_normal_nodes_but_not_to_master_nodes": {
			labelsToAdd: map[string]string{"a8s.key1": "val1"},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{"key10": "val10"}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{"key10": "val10"}),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/master", Effect: "NoExecute"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"key10":    "val10",
						"a8s.key1": "val1",
					}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{"key10": "val10"}),
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
			if err := nodes.LabelAll(context.Background(), tc.labelsToAdd); err != nil {
				t.Fatal("Expected no error when invoking LabelAll, got:", err)
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

func TestLabelAllListFails(t *testing.T) {
	t.Parallel()

	labelsToAdd := map[string]string{"a8s.key1": "val1"}
	inputNode := newNode(withName("n1"), withLabels(nil))

	// Prepare a fake K8s client that is sabotaged to return an error on LIST API calls.
	sabotagedK8sClient := fake.NewSimpleClientset(&v1.NodeList{Items: []v1.Node{inputNode}})
	listSabotager := func(k8stest.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("dummy error")
	}
	sabotagedK8sClient.PrependReactor("list", "nodes", listSabotager)

	// Set up the object under test with the sabotaged K8s client
	nodes := node.Client{
		Nodes:            sabotagedK8sClient.CoreV1().Nodes(),
		MasterNodeTaints: node.MasterTaintKeys,
	}

	// Invoke the method under test
	err := nodes.LabelAll(context.Background(), labelsToAdd)

	if err == nil {
		t.Fatal("LabelAll returned <nil> error, expected non-nil error")
	}

	if !strings.Contains(err.Error(), "dummy error") {
		t.Fatalf("Got error \"%s\" must contain message of injected error "+
			"\"dummy error\" but it doesn't", err.Error())
	}
}

func TestLabelAllUpdateFails(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		labelsToAdd   map[string]string
		inputNodes    []v1.Node
		expectedNodes []v1.Node
		// this is a hash map rather than a slice to make it easy to verify if a node's update
		// should fail with `nodesWhoseUpdateFails[nodeName]`
		nodesWhoseUpdateFails map[string]struct{}
	}{
		"no_node_is_updated_because_updating_fails_for_all_nodes": {
			labelsToAdd: map[string]string{"a8s.key1": "val1"},
			inputNodes: []v1.Node{
				newNode(withName("n1"), withLabels(nil)),
				newNode(withName("n2"), withLabels(nil)),
			},
			expectedNodes: []v1.Node{
				newNode(withName("n1"), withLabels(nil)),
				newNode(withName("n2"), withLabels(nil)),
			},
			nodesWhoseUpdateFails: map[string]struct{}{
				"n1": {},
				"n2": {},
			},
		},

		"only_1_node_out_of_3_is_updated_because_updating_the_others_fails": {
			labelsToAdd: map[string]string{"a8s.key1": "val1"},
			inputNodes: []v1.Node{
				newNode(withName("n1"), withLabels(nil)),
				newNode(withName("n2"), withLabels(nil)),
				newNode(withName("n3"), withLabels(nil)),
			},
			expectedNodes: []v1.Node{
				newNode(withName("n1"), withLabels(map[string]string{"a8s.key1": "val1"})),
				newNode(withName("n2"), withLabels(nil)),
				newNode(withName("n3"), withLabels(nil)),
			},
			nodesWhoseUpdateFails: map[string]struct{}{
				"n2": {},
				"n3": {},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Rebind tc into this lexical scope. Details on the why at
			// https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
			tc := tc

			t.Parallel()

			// Define a function that will intercept the K8s Update API calls on the nodes whose
			// update must fail in the test case and makes them fail by returning an error.
			testErr := errors.New("dummy error")
			updateSabotager := func(apiCall k8stest.Action) (bool, runtime.Object, error) {
				updatedNode := apiCall.(k8stest.UpdateAction).GetObject().(*v1.Node)
				if _, mustFail := tc.nodesWhoseUpdateFails[updatedNode.Name]; mustFail {
					return true, nil, testErr
				}
				return false, nil, nil
			}

			// Prepare a fake K8s client that is sabotaged to return an error on the update of
			// certain nodes via the function defined right above.
			sabotagedK8sClient := fake.NewSimpleClientset(&v1.NodeList{Items: tc.inputNodes})
			sabotagedK8sClient.PrependReactor("update", "nodes", updateSabotager)
			k8sAPINodesClient := sabotagedK8sClient.CoreV1().Nodes()

			// Set up the object under test with the sabotaged K8s client
			nodes := node.Client{
				Nodes:            k8sAPINodesClient,
				MasterNodeTaints: node.MasterTaintKeys,
			}

			// Invoke method under test
			err := nodes.LabelAll(context.Background(), tc.labelsToAdd)

			if err == nil {
				t.Fatal("LabelAll returned <nil> error, expected non-nil error")
			}

			// Verify that the error message of every update that failed is mentioned in the error
			// returned by LabelAll
			individualUpdateErrsCount := strings.Count(err.Error(), testErr.Error())
			if individualUpdateErrsCount != len(tc.nodesWhoseUpdateFails) {
				t.Fatalf("Got error \"%s\" must report the error message of every update that "+
					"failed, but it doesn't: %d updates should have failed but it reports the "+
					"error messages of %d", err.Error(), len(tc.nodesWhoseUpdateFails),
					individualUpdateErrsCount)
			}

			for nodeName := range tc.nodesWhoseUpdateFails {
				if !strings.Contains(err.Error(), nodeName) {
					t.Fatalf("Got error \"%s\" must report the name of each node whose update "+
						"failed and it doesn't: update of %s should have failed but %s is not "+
						"contained in the error message", err.Error(), nodeName, nodeName)
				}
			}

			// Get the nodes after invoking the method under test
			gotNodesList, err := k8sAPINodesClient.List(context.Background(), metav1.ListOptions{})
			if err != nil {
				t.Fatal("Expected no error when listing nodes, got:", err)
			}
			gotNodes := gotNodesList.Items

			// Compare the expected nodes with the got ones to ensure that only those whose update
			// didn't fail changed.
			if !equality.Semantic.DeepEqual(tc.expectedNodes, gotNodes) {
				t.Fatalf("Expected nodes don't match got ones\n\n\texpected: %#+v\n\n\tgot:"+
					" %#+v\n\n", tc.expectedNodes, gotNodes)
			}
		})
	}
}

func TestLabelAllPanicsWhenLabelKeyMatchesAndValDoesNot(t *testing.T) {
	t.Parallel()

	// inputNode has a label with the same key but different value as the label to add, that must
	// trigger a panic.
	labelsToAdd := map[string]string{"a8s.key1": "val1"}
	inputNode := newNode(withName("n1"), withLabels(map[string]string{"a8s.key1": "different-val"}))

	// Set up the object under test.
	k8sAPINodesClient := fake.
		NewSimpleClientset(&v1.NodeList{Items: []v1.Node{inputNode}}).
		CoreV1().
		Nodes()
	nodes := node.Client{
		Nodes:            k8sAPINodesClient,
		MasterNodeTaints: node.MasterTaintKeys,
	}

	// Deferred check that the method under test will panic, produce an informative enough message,
	// and leave the nodes unchanged.
	defer func() {
		panicMsg := recover()
		if panicMsg == nil {
			t.Fatal("Expected LabelAll to panic, but it didn't")
		}
		panicMsgStr := fmt.Sprintf("%#+v", panicMsg)
		if !strings.Contains(panicMsgStr, "a8s.key1") {
			t.Fatal("Expected LabelAll panic message to mention the label key, but it didn't")
		}
		if !strings.Contains(panicMsgStr, "val1") {
			t.Fatal("Expected LabelAll panic message to mention the allowed label value, " +
				"but it didn't")
		}
		if !strings.Contains(panicMsgStr, "different-val") {
			t.Fatal("Expected LabelAll panic message to mention the found label value, " +
				"but it didn't")
		}

		// To make sure that the node wasn't updated anyway, we get the nodes after invoking the
		// method under test.
		gotNodesList, err := k8sAPINodesClient.List(context.Background(), metav1.ListOptions{})
		if err != nil {
			t.Fatal("Expected no error when listing nodes, got:", err)
		}
		// We only had one node so we can get it at index 0.
		gotNode := gotNodesList.Items[0]

		// Compare the input node with the got one to ensure it wasn't updated even if there was a
		// panic.
		if !equality.Semantic.DeepEqual(inputNode, gotNode) {
			t.Fatalf("Expected node doesn't match got one\n\n\texpected: %#+v\n\n\tgot:"+
				" %#+v\n\n", inputNode, gotNode)
		}
	}()

	// Invoke the method under test.
	if err := nodes.LabelAll(context.Background(), labelsToAdd); err != nil {
		t.Fatal("Expected panic when invoking LabelAll, got error instead:", err)
	}
}

func TestUnlabelAllHappyPaths(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		labelsToRemove map[string]string
		inputNodes     []v1.Node
		expectedNodes  []v1.Node
	}{
		"nodes_with_no_labels_are_left_unchanged": {
			labelsToRemove: map[string]string{"a8s.key1": "val1"},
			inputNodes: []v1.Node{
				newNode(withName("n1"), withLabels(nil)),
				newNode(withName("n2"), withLabels(map[string]string{})),
			},
			expectedNodes: []v1.Node{
				newNode(withName("n1"), withLabels(nil)),
				newNode(withName("n2"), withLabels(map[string]string{})),
			},
		},

		"1_label_removed_from_1_node_with_no_other_labels": {
			labelsToRemove: map[string]string{"a8s.key1": "val1"},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{"a8s.key1": "val1"}),
				),
			},
			expectedNodes: []v1.Node{newNode(withName("n1"), withLabels(nil))},
		},

		"2_labels_removed_from_1_node_with_no_other_labels": {
			labelsToRemove: map[string]string{
				"a8s.key1": "val1",
				"a8s.key2": "val2",
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"a8s.key2": "val2",
					}),
				),
			},
			expectedNodes: []v1.Node{newNode(withName("n1"), withLabels(nil))},
		},

		"1_label_removed_from_1_node_that_has_other_labels_as_well": {
			labelsToRemove: map[string]string{"a8s.key1": "val1"},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"key10":    "val10",
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{"key10": "val10"}),
				),
			},
		},

		"2_labels_removed_from_1_node_that_has_other_labels_as_well": {
			labelsToRemove: map[string]string{
				"a8s.key1": "val1",
				"a8s.key2": "val2",
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"key10":    "val10",
						"a8s.key2": "val2",
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{"key10": "val10"}),
				),
			},
		},

		"2_labels_removed_from_2_nodes_that_have_no_other_labels": {
			labelsToRemove: map[string]string{
				"a8s.key1": "val1",
				"a8s.key2": "val2",
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"a8s.key2": "val2",
					}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"a8s.key2": "val2",
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(withName("n1"), withLabels(nil)),
				newNode(withName("n2"), withLabels(nil)),
			},
		},

		"2_labels_removed_from_2_nodes_1_with_other_labels_as_well": {
			labelsToRemove: map[string]string{
				"a8s.key1": "val1",
				"a8s.key2": "val2",
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"a8s.key2": "val2",
					}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"a8s.key2": "val2",
						"key10":    "val10",
						"key20":    "val20",
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(withName("n1"), withLabels(nil)),
				newNode(
					withName("n2"),
					withLabels(map[string]string{
						"key10": "val10",
						"key20": "val20",
					}),
				),
			},
		},

		"1_node_with_labels_but_not_the_ones_to_remove_is_left_unchanged": {
			labelsToRemove: map[string]string{
				"a8s.key1": "val1",
				"a8s.key2": "val2",
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"key10": "val10",
						"key20": "val20",
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"key10": "val10",
						"key20": "val20",
					}),
				),
			},
		},

		"2_labels_are_removed_from_nodes_that_have_them_but_other_nodes_are_unchanged": {
			labelsToRemove: map[string]string{
				"a8s.key1": "val1",
				"a8s.key2": "val2",
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"a8s.key2": "val2",
					}),
				),
				newNode(
					withName("n2"),
					withLabels(map[string]string{
						"key10": "val10",
						"key20": "val20",
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(withName("n1"), withLabels(nil)),
				newNode(
					withName("n2"),
					withLabels(map[string]string{
						"key10": "val10",
						"key20": "val20",
					}),
				),
			},
		},

		"only_the_labels_to_remove_that_a_node_has_are_removed": {
			labelsToRemove: map[string]string{
				"a8s.key1": "val1",
				"a8s.key2": "val2",
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"key10":    "val10",
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{"key10": "val10"}),
				),
			},
		},

		"when_removing_2_labels_from_4_nodes_only_the_labels_that_the_nodes_have_are_removed": {
			labelsToRemove: map[string]string{
				"a8s.key1": "val1",
				"a8s.key2": "val2",
			},
			inputNodes: []v1.Node{
				newNode(withName("n1"), withLabels(nil)),
				newNode(
					withName("n2"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"key10":    "val10",
					}),
				),
				newNode(
					withName("n3"),
					withLabels(map[string]string{"key10": "val10"}),
				),
				newNode(
					withName("n4"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"a8s.key2": "val2",
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(withName("n1"), withLabels(nil)),
				newNode(
					withName("n2"),
					withLabels(map[string]string{"key10": "val10"}),
				),
				newNode(
					withName("n3"),
					withLabels(map[string]string{"key10": "val10"}),
				),
				newNode(withName("n4"), withLabels(nil)),
			},
		},

		"2_labels_are_removed_from_a_node_with_master_taints": {
			labelsToRemove: map[string]string{
				"a8s.key1": "val1",
				"a8s.key2": "val2",
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"a8s.key2": "val2",
					}),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/master", Effect: "NoExecute"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(nil),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/master", Effect: "NoExecute"},
					}),
				),
			},
		},

		"2_labels_are_removed_from_a_node_with_control_plane_taints": {
			labelsToRemove: map[string]string{
				"a8s.key1": "val1",
				"a8s.key2": "val2",
			},
			inputNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(map[string]string{
						"a8s.key1": "val1",
						"a8s.key2": "val2",
					}),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/control-plane", Effect: "NoExecute"},
					}),
				),
			},
			expectedNodes: []v1.Node{
				newNode(
					withName("n1"),
					withLabels(nil),
					withTaints([]v1.Taint{
						{Key: "node-role.kubernetes.io/control-plane", Effect: "NoExecute"},
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
			if err := nodes.UnlabelAll(context.Background(), tc.labelsToRemove); err != nil {
				t.Fatal("Expected no error when invoking UnlabelAll, got:", err)
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

func TestUnlabelAllListFails(t *testing.T) {
	t.Parallel()

	labelsToRemove := map[string]string{"a8s.key1": "val1"}
	inputNode := newNode(withName("n1"), withLabels(map[string]string{"a8s.key1": "val1"}))

	// Prepare a fake K8s client that is sabotaged to return an error on LIST API calls.
	sabotagedK8sClient := fake.NewSimpleClientset(&v1.NodeList{Items: []v1.Node{inputNode}})
	listSabotager := func(k8stest.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("dummy error")
	}
	sabotagedK8sClient.PrependReactor("list", "nodes", listSabotager)

	// Set up the object under test with the sabotaged K8s client
	nodes := node.Client{
		Nodes:            sabotagedK8sClient.CoreV1().Nodes(),
		MasterNodeTaints: node.MasterTaintKeys,
	}

	// Invoke the method under test
	err := nodes.UnlabelAll(context.Background(), labelsToRemove)

	if err == nil {
		t.Fatal("UnlabelAll returned <nil> error, expected non-nil error")
	}

	if !strings.Contains(err.Error(), "dummy error") {
		t.Fatalf("Got error \"%s\" must contain message of injected error "+
			"\"dummy error\" but it doesn't", err.Error())
	}
}

func TestUnlabelAllUpdateFails(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		labelsToRemove map[string]string
		inputNodes     []v1.Node
		expectedNodes  []v1.Node
		// this is a hash map rather than a slice to make it easy to verify if a node's update
		// should fail with `nodesWhoseUpdateFails[nodeName]`
		nodesWhoseUpdateFails map[string]struct{}
	}{
		"no_node_is_updated_because_updating_fails_for_all_nodes": {
			labelsToRemove: map[string]string{"a8s.key1": "val1"},
			inputNodes: []v1.Node{
				newNode(withName("n1"), withLabels(map[string]string{"a8s.key1": "val1"})),
				newNode(withName("n2"), withLabels(map[string]string{"a8s.key1": "val1"})),
			},
			expectedNodes: []v1.Node{
				newNode(withName("n1"), withLabels(map[string]string{"a8s.key1": "val1"})),
				newNode(withName("n2"), withLabels(map[string]string{"a8s.key1": "val1"})),
			},
			nodesWhoseUpdateFails: map[string]struct{}{
				"n1": {},
				"n2": {},
			},
		},

		"only_1_node_out_of_3_is_updated_because_updating_the_others_fails": {
			labelsToRemove: map[string]string{"a8s.key1": "val1"},
			inputNodes: []v1.Node{
				newNode(withName("n1"), withLabels(map[string]string{"a8s.key1": "val1"})),
				newNode(withName("n2"), withLabels(map[string]string{"a8s.key1": "val1"})),
				newNode(withName("n3"), withLabels(map[string]string{"a8s.key1": "val1"})),
			},
			expectedNodes: []v1.Node{
				newNode(withName("n1"), withLabels(nil)),
				newNode(withName("n2"), withLabels(map[string]string{"a8s.key1": "val1"})),
				newNode(withName("n3"), withLabels(map[string]string{"a8s.key1": "val1"})),
			},
			nodesWhoseUpdateFails: map[string]struct{}{
				"n2": {},
				"n3": {},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			// Rebind tc into this lexical scope. Details on the why at
			// https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
			tc := tc

			t.Parallel()

			// Define a function that will intercept the K8s Update API calls on the nodes whose
			// update must fail in the test case and makes them fail by returning an error.
			testErr := errors.New("dummy error")
			updateSabotager := func(apiCall k8stest.Action) (bool, runtime.Object, error) {
				updatedNode := apiCall.(k8stest.UpdateAction).GetObject().(*v1.Node)
				if _, mustFail := tc.nodesWhoseUpdateFails[updatedNode.Name]; mustFail {
					return true, nil, testErr
				}
				return false, nil, nil
			}

			// Prepare a fake K8s client that is sabotaged to return an error on the update of
			// certain nodes via the function defined right above.
			sabotagedK8sClient := fake.NewSimpleClientset(&v1.NodeList{Items: tc.inputNodes})
			sabotagedK8sClient.PrependReactor("update", "nodes", updateSabotager)
			k8sAPINodesClient := sabotagedK8sClient.CoreV1().Nodes()

			// Set up the object under test with the sabotaged K8s client
			nodes := node.Client{
				Nodes:            k8sAPINodesClient,
				MasterNodeTaints: node.MasterTaintKeys,
			}

			// Invoke method under test
			err := nodes.UnlabelAll(context.Background(), tc.labelsToRemove)

			if err == nil {
				t.Fatal("UnlabelAll returned <nil> error, expected non-nil error")
			}

			// Verify that the error message of every update that failed is mentioned in the error
			// returned by UnlabelAll
			individualUpdateErrsCount := strings.Count(err.Error(), testErr.Error())
			if individualUpdateErrsCount != len(tc.nodesWhoseUpdateFails) {
				t.Fatalf("Got error \"%s\" must report the error message of every update that "+
					"failed, but it doesn't: %d updates should have failed but it reports the "+
					"error messages of %d", err.Error(), len(tc.nodesWhoseUpdateFails),
					individualUpdateErrsCount)
			}

			for nodeName := range tc.nodesWhoseUpdateFails {
				if !strings.Contains(err.Error(), nodeName) {
					t.Fatalf("Got error \"%s\" must report the name of each node whose update "+
						"failed and it doesn't: update of %s should have failed but %s is not "+
						"contained in the error message", err.Error(), nodeName, nodeName)
				}
			}

			// Get the nodes after invoking the method under test
			gotNodesList, err := k8sAPINodesClient.List(context.Background(), metav1.ListOptions{})
			if err != nil {
				t.Fatal("Expected no error when listing nodes, got:", err)
			}
			gotNodes := gotNodesList.Items

			// Compare the expected nodes with the got ones to ensure that only those whose update
			// didn't fail changed.
			if !equality.Semantic.DeepEqual(tc.expectedNodes, gotNodes) {
				t.Fatalf("Expected nodes don't match got ones\n\n\texpected: %#+v\n\n\tgot:"+
					" %#+v\n\n", tc.expectedNodes, gotNodes)
			}
		})
	}
}

func TestUnlabelAllPanicsWhenLabelKeyMatchesAndValDoesNot(t *testing.T) {
	t.Parallel()

	// inputNode has a label with the same key but different value as the label to remove, that must
	// trigger a panic.
	labelsToRemove := map[string]string{"a8s.key1": "val1"}
	inputNode := newNode(withName("n1"), withLabels(map[string]string{"a8s.key1": "different-val"}))

	// Set up the object under test.
	k8sAPINodesClient := fake.
		NewSimpleClientset(&v1.NodeList{Items: []v1.Node{inputNode}}).
		CoreV1().
		Nodes()
	nodes := node.Client{
		Nodes:            k8sAPINodesClient,
		MasterNodeTaints: node.MasterTaintKeys,
	}

	// Deferred check that the method under test will panic, produce an informative enough message,
	// and leave the nodes unchanged.
	defer func() {
		panicMsg := recover()
		if panicMsg == nil {
			t.Fatal("Expected UnlabelAll to panic, but it didn't")
		}
		panicMsgStr := fmt.Sprintf("%#+v", panicMsg)
		if !strings.Contains(panicMsgStr, "a8s.key1") {
			t.Fatal("Expected UnlabelAll panic message to mention the label key, but it didn't")
		}
		if !strings.Contains(panicMsgStr, "val1") {
			t.Fatal("Expected UnlabelAll panic message to mention the allowed label value, " +
				"but it didn't")
		}
		if !strings.Contains(panicMsgStr, "different-val") {
			t.Fatal("Expected UnlabelAll panic message to mention the found label value, " +
				"but it didn't")
		}

		// To make sure that the node wasn't updated anyway, we get the nodes after invoking the
		// method under test.
		gotNodesList, err := k8sAPINodesClient.List(context.Background(), metav1.ListOptions{})
		if err != nil {
			t.Fatal("Expected no error when listing nodes, got:", err)
		}
		// We only had one node so we can get it at index 0.
		gotNode := gotNodesList.Items[0]

		// Compare the input node with the got one to ensure it wasn't updated even if there was a
		// panic.
		if !equality.Semantic.DeepEqual(inputNode, gotNode) {
			t.Fatalf("Expected node doesn't match got one\n\n\texpected: %#+v\n\n\tgot:"+
				" %#+v\n\n", inputNode, gotNode)
		}
	}()

	// Invoke the method under test.
	if err := nodes.UnlabelAll(context.Background(), labelsToRemove); err != nil {
		t.Fatal("Expected panic when invoking UnlabelAll, got error instead:", err)
	}
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

func withLabels(labels map[string]string) func(*v1.Node) {
	return func(n *v1.Node) {
		n.Labels = labels
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
