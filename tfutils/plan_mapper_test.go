package tfutils

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/overmindtech/sdp-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestWithStateFile(t *testing.T) {
	_, err := MappedItemDiffsFromPlanFile(context.Background(), "testdata/state.json", logrus.Fields{})

	if err == nil {
		t.Error("Expected error when running with state file, got none")
	}
}

func TestExtractProviderNameFromConfigKey(t *testing.T) {
	tests := []struct {
		ConfigKey string
		Expected  string
	}{
		{
			ConfigKey: "kubernetes",
			Expected:  "kubernetes",
		},
		{
			ConfigKey: "module.core:kubernetes",
			Expected:  "kubernetes",
		},
	}

	for _, test := range tests {
		t.Run(test.ConfigKey, func(t *testing.T) {
			actual := extractProviderNameFromConfigKey(test.ConfigKey)
			if actual != test.Expected {
				t.Errorf("Expected %v, got %v", test.Expected, actual)
			}
		})
	}
}

func TestMappedItemDiffsFromPlan(t *testing.T) {
	results, err := MappedItemDiffsFromPlanFile(context.Background(), "testdata/plan.json", logrus.Fields{})
	if err != nil {
		t.Error(err)
	}

	if results.RemovedSecrets != 16 {
		t.Errorf("Expected 16 secrets, got %v", results.RemovedSecrets)
	}

	if len(results.Results) != 6 {
		t.Errorf("Expected 6 changes, got %v:", len(results.Results))
		for _, diff := range results.Results {
			t.Errorf("  %v", diff)
		}
	}

	var nats_box_deployment *sdp.MappedItemDiff
	var api_server_deployment *sdp.MappedItemDiff
	var aws_iam_policy *sdp.MappedItemDiff
	var secret *sdp.MappedItemDiff

	for _, result := range results.Results {
		item := result.GetItem().GetBefore()
		if item == nil && result.GetItem().GetAfter() != nil {
			item = result.GetItem().GetAfter()
		}
		if item == nil {
			t.Errorf("Expected any of before/after items to be set, but there's nothing: %v", result)
			continue
		}

		// t.Logf("item: %v", item.Attributes.AttrStruct.Fields["terraform_address"].GetStringValue())
		if item.GetAttributes().GetAttrStruct().GetFields()["terraform_address"].GetStringValue() == "kubernetes_deployment.nats_box" {
			if nats_box_deployment != nil {
				t.Errorf("Found multiple nats_box_deployment: %v, %v", nats_box_deployment, result)
			}
			nats_box_deployment = result.MappedItemDiff
		} else if item.GetAttributes().GetAttrStruct().GetFields()["terraform_address"].GetStringValue() == "kubernetes_deployment.api_server" {
			if api_server_deployment != nil {
				t.Errorf("Found multiple api_server_deployment: %v, %v", api_server_deployment, result)
			}
			api_server_deployment = result.MappedItemDiff
		} else if item.GetType() == "iam-policy" {
			if aws_iam_policy != nil {
				t.Errorf("Found multiple aws_iam_policy: %v, %v", aws_iam_policy, result)
			}
			aws_iam_policy = result.MappedItemDiff
		} else if item.GetType() == "Secret" {
			if secret != nil {
				t.Errorf("Found multiple secrets: %v, %v", secret, result)
			}
			secret = result.MappedItemDiff
		}
	}

	// check nats_box_deployment
	t.Logf("nats_box_deployment: %v", nats_box_deployment)
	if nats_box_deployment == nil {
		t.Fatalf("Expected nats_box_deployment to be set, but it's not")
	}
	if nats_box_deployment.GetItem().GetStatus() != sdp.ItemDiffStatus_ITEM_DIFF_STATUS_DELETED {
		t.Errorf("Expected nats_box_deployment status to be 'deleted', but it's '%v'", nats_box_deployment.GetItem().GetStatus())
	}
	if nats_box_deployment.GetMappingQuery().GetType() != "Deployment" {
		t.Errorf("Expected nats_box_deployment query type to be 'Deployment', got '%v'", nats_box_deployment.GetMappingQuery().GetType())
	}
	if nats_box_deployment.GetMappingQuery().GetQuery() != "nats-box" {
		t.Errorf("Expected nats_box_deployment query to be 'nats-box', got '%v'", nats_box_deployment.GetMappingQuery().GetQuery())
	}
	if nats_box_deployment.GetMappingQuery().GetScope() != "*" {
		t.Errorf("Expected nats_box_deployment query scope to be '*', got '%v'", nats_box_deployment.GetMappingQuery().GetScope())
	}
	if nats_box_deployment.GetItem().GetBefore().GetScope() != "terraform_plan" {
		t.Errorf("Expected nats_box_deployment before item scope to be 'terraform_plan', got '%v'", nats_box_deployment.GetItem().GetBefore().GetScope())
	}
	if nats_box_deployment.GetMappingQuery().GetType() != "Deployment" {
		t.Errorf("Expected nats_box_deployment query type to be 'Deployment', got '%v'", nats_box_deployment.GetMappingQuery().GetType())
	}
	if nats_box_deployment.GetItem().GetBefore().GetType() != "Deployment" {
		t.Errorf("Expected nats_box_deployment before item type to be 'Deployment', got '%v'", nats_box_deployment.GetItem().GetBefore().GetType())
	}
	if nats_box_deployment.GetMappingQuery().GetQuery() != "nats-box" {
		t.Errorf("Expected nats_box_deployment query query to be 'nats-box', got '%v'", nats_box_deployment.GetMappingQuery().GetQuery())
	}

	// check api_server_deployment
	t.Logf("api_server_deployment: %v", api_server_deployment)
	if api_server_deployment == nil {
		t.Fatalf("Expected api_server_deployment to be set, but it's not")
	}
	if api_server_deployment.GetItem().GetStatus() != sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UPDATED {
		t.Errorf("Expected api_server_deployment status to be 'updated', but it's '%v'", api_server_deployment.GetItem().GetStatus())
	}
	if api_server_deployment.GetMappingQuery().GetType() != "Deployment" {
		t.Errorf("Expected api_server_deployment query type to be 'Deployment', got '%v'", api_server_deployment.GetMappingQuery().GetType())
	}
	if api_server_deployment.GetMappingQuery().GetQuery() != "api-server" {
		t.Errorf("Expected api_server_deployment query to be 'api-server', got '%v'", api_server_deployment.GetMappingQuery().GetQuery())
	}
	if api_server_deployment.GetMappingQuery().GetScope() != "dogfood.default" {
		t.Errorf("Expected api_server_deployment query scope to be 'dogfood.default', got '%v'", api_server_deployment.GetMappingQuery().GetScope())
	}
	if api_server_deployment.GetItem().GetBefore().GetScope() != "dogfood.default" {
		t.Errorf("Expected api_server_deployment before item scope to be 'dogfood.default', got '%v'", api_server_deployment.GetItem().GetBefore().GetScope())
	}
	if api_server_deployment.GetMappingQuery().GetType() != "Deployment" {
		t.Errorf("Expected api_server_deployment query type to be 'Deployment', got '%v'", api_server_deployment.GetMappingQuery().GetType())
	}
	if api_server_deployment.GetItem().GetBefore().GetType() != "Deployment" {
		t.Errorf("Expected api_server_deployment before item type to be 'Deployment', got '%v'", api_server_deployment.GetItem().GetBefore().GetType())
	}
	if api_server_deployment.GetMappingQuery().GetQuery() != "api-server" {
		t.Errorf("Expected api_server_deployment query query to be 'api-server', got '%v'", api_server_deployment.GetMappingQuery().GetQuery())
	}

	// check aws_iam_policy
	t.Logf("aws_iam_policy: %v", aws_iam_policy)
	if aws_iam_policy == nil {
		t.Fatalf("Expected aws_iam_policy to be set, but it's not")
	}
	if aws_iam_policy.GetItem().GetStatus() != sdp.ItemDiffStatus_ITEM_DIFF_STATUS_UPDATED {
		t.Errorf("Expected aws_iam_policy status to be 'updated', but it's %v", aws_iam_policy.GetItem().GetStatus())
	}
	if aws_iam_policy.GetMappingQuery().GetType() != "iam-policy" {
		t.Errorf("Expected aws_iam_policy query type to be 'iam-policy', got '%v'", aws_iam_policy.GetMappingQuery().GetType())
	}
	if aws_iam_policy.GetMappingQuery().GetQuery() != "arn:aws:iam::123456789012:policy/test-alb-ingress" {
		t.Errorf("Expected aws_iam_policy query to be 'arn:aws:iam::123456789012:policy/test-alb-ingress', got '%v'", aws_iam_policy.GetMappingQuery().GetQuery())
	}
	if aws_iam_policy.GetMappingQuery().GetScope() != "*" {
		t.Errorf("Expected aws_iam_policy query scope to be '*', got '%v'", aws_iam_policy.GetMappingQuery().GetScope())
	}
	if aws_iam_policy.GetItem().GetBefore().GetScope() != "terraform_plan" {
		t.Errorf("Expected aws_iam_policy before item scope to be 'terraform_plan', got '%v'", aws_iam_policy.GetItem().GetBefore().GetScope())
	}
	if aws_iam_policy.GetMappingQuery().GetType() != "iam-policy" {
		t.Errorf("Expected aws_iam_policy query type to be 'iam-policy', got '%v'", aws_iam_policy.GetMappingQuery().GetType())
	}
	if aws_iam_policy.GetItem().GetBefore().GetType() != "iam-policy" {
		t.Errorf("Expected aws_iam_policy before item type to be 'iam-policy', got '%v'", aws_iam_policy.GetItem().GetBefore().GetType())
	}
	if aws_iam_policy.GetMappingQuery().GetQuery() != "arn:aws:iam::123456789012:policy/test-alb-ingress" {
		t.Errorf("Expected aws_iam_policy query query to be 'arn:aws:iam::123456789012:policy/test-alb-ingress', got '%v'", aws_iam_policy.GetMappingQuery().GetQuery())
	}

	// check secret
	t.Logf("secret: %v", secret)
	if secret == nil {
		t.Fatalf("Expected secret to be set, but it's not")
	}
	if secret.GetMappingQuery().GetScope() != "dogfood.default" {
		t.Errorf("Expected secret query scope to be 'dogfood.default', got '%v'", secret.GetMappingQuery().GetScope())
	}

	// In a secret the "data" field is known after apply, but we don't *know*
	// that it's definitely going to change, so this should be (known after apply)
	dataVal, _ := secret.GetItem().GetAfter().GetAttributes().Get("data")
	if dataVal != KnownAfterApply {
		t.Errorf("Expected secret data to be known after apply, got '%v'", dataVal)
	}
}

func TestPlanMappingResultNumFuncs(t *testing.T) {
	result := PlanMappingResult{
		Results: []PlannedChangeMapResult{
			{
				Status: MapStatusSuccess,
			},
			{
				Status: MapStatusSuccess,
			},
			{
				Status: MapStatusNotEnoughInfo,
			},
			{
				Status: MapStatusUnsupported,
			},
		},
	}

	if result.NumSuccess() != 2 {
		t.Errorf("Expected 2 success, got %v", result.NumSuccess())
	}

	if result.NumNotEnoughInfo() != 1 {
		t.Errorf("Expected 1 not enough info, got %v", result.NumNotEnoughInfo())
	}

	if result.NumUnsupported() != 1 {
		t.Errorf("Expected 1 unsupported, got %v", result.NumUnsupported())
	}
}

func TestInterpolateScope(t *testing.T) {
	t.Run("with no interpolation", func(t *testing.T) {
		t.Parallel()

		result, err := InterpolateScope("foo", map[string]any{})

		if err != nil {
			t.Error(err)
		}

		if result != "foo" {
			t.Errorf("Expected result to be foo, got %s", result)
		}
	})

	t.Run("with a single variable", func(t *testing.T) {
		t.Parallel()

		result, err := InterpolateScope("${outputs.overmind_kubernetes_cluster_name}", map[string]any{
			"outputs": map[string]any{
				"overmind_kubernetes_cluster_name": "foo",
			},
		})

		if err != nil {
			t.Error(err)
		}

		if result != "foo" {
			t.Errorf("Expected result to be foo, got %s", result)
		}
	})

	t.Run("with multiple variables", func(t *testing.T) {
		t.Parallel()

		result, err := InterpolateScope("${outputs.overmind_kubernetes_cluster_name}.${values.metadata.namespace}", map[string]any{
			"outputs": map[string]any{
				"overmind_kubernetes_cluster_name": "foo",
			},
			"values": map[string]any{
				"metadata": map[string]any{
					"namespace": "bar",
				},
			},
		})

		if err != nil {
			t.Error(err)
		}

		if result != "foo.bar" {
			t.Errorf("Expected result to be foo.bar, got %s", result)
		}
	})

	t.Run("with a variable that doesn't exist", func(t *testing.T) {
		t.Parallel()

		_, err := InterpolateScope("${outputs.overmind_kubernetes_cluster_name}", map[string]any{})

		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

// note that these tests need to allocate the input map for every test to avoid
// false positives from maskSensitiveData mutating the data
func TestMaskSensitiveData(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		got := maskSensitiveData(map[string]any{}, map[string]any{})
		require.Equal(t, map[string]any{}, got)
	})

	t.Run("easy", func(t *testing.T) {
		t.Parallel()
		require.Equal(t,
			map[string]any{
				"foo": "bar",
			},
			maskSensitiveData(
				map[string]any{
					"foo": "bar",
				},
				map[string]any{}))

		require.Equal(t,
			map[string]any{
				"foo": "(sensitive value)",
			},
			maskSensitiveData(
				map[string]any{
					"foo": "bar",
				},
				map[string]any{"foo": true}))

	})

	t.Run("deep", func(t *testing.T) {
		t.Parallel()
		require.Equal(t,
			map[string]any{
				"foo": map[string]any{"key": "bar"},
			},
			maskSensitiveData(
				map[string]any{
					"foo": map[string]any{"key": "bar"},
				},
				map[string]any{}))

		require.Equal(t,
			map[string]any{
				"foo": "(sensitive value)",
			},
			maskSensitiveData(
				map[string]any{
					"foo": map[string]any{"key": "bar"},
				},
				map[string]any{"foo": true}))

		require.Equal(t,
			map[string]any{
				"foo": map[string]any{"key": "(sensitive value)"},
			},
			maskSensitiveData(
				map[string]any{
					"foo": map[string]any{"key": "bar"},
				},
				map[string]any{"foo": map[string]any{"key": true}}))

	})

	t.Run("arrays", func(t *testing.T) {
		t.Parallel()
		require.Equal(t,
			map[string]any{
				"foo": []any{"one", "two"},
			},
			maskSensitiveData(
				map[string]any{
					"foo": []any{"one", "two"},
				},
				map[string]any{}))

		require.Equal(t,
			map[string]any{
				"foo": "(sensitive value)",
			},
			maskSensitiveData(
				map[string]any{
					"foo": []any{"one", "two"},
				},
				map[string]any{"foo": true}))

		require.Equal(t,
			map[string]any{
				"foo": []any{"one", "(sensitive value)"},
			},
			maskSensitiveData(
				map[string]any{
					"foo": []any{"one", "two"},
				},
				map[string]any{"foo": []any{false, true}}))

	})
}

func TestHandleKnownAfterApply(t *testing.T) {
	before, err := sdp.ToAttributes(map[string]interface{}{
		"string_value": "foo",
		"int_value":    42,
		"bool_value":   true,
		"float_value":  3.14,
		"data":         "secret", // Known after apply but doesn't exist in the "after" map, this happens sometimes
		"list_value": []interface{}{
			"foo",
			"bar",
		},
		"map_value": map[string]interface{}{
			"foo": "bar",
			"bar": "baz",
		},
		"map_value2": map[string]interface{}{
			"ding": map[string]interface{}{
				"foo": "bar",
			},
		},
		"nested_list": []interface{}{
			[]interface{}{},
			[]interface{}{
				"foo",
				"bar",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	after, err := sdp.ToAttributes(map[string]interface{}{
		"string_value": "bar", // I want to see a diff here
		"int_value":    nil,   // These are going to be known after apply
		"bool_value":   nil,   // These are going to be known after apply
		"float_value":  3.14,
		"list_value": []interface{}{
			"foo",
			"bar",
			"baz", // So is this one
		},
		"map_value": map[string]interface{}{ // This whole thing will be known after apply
			"foo": "bar",
		},
		"map_value2": map[string]interface{}{
			"ding": map[string]interface{}{
				"foo": nil, // This will be known after apply
			},
		},
		"nested_list": []interface{}{
			[]interface{}{
				"foo",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	afterUnknown := json.RawMessage(`{
		"int_value": true,
		"bool_value": true,
		"float_value": false,
		"data": true,
		"list_value": [
			false,
			false,
			true
		],
		"map_value": true,
		"map_value2": {
			"ding": {
				"foo": true
			}
		},
		"nested_list": [
			[
				false,
				true
			],
			[
				false,
				true
			]
		]
	}`)

	err = handleKnownAfterApply(before, after, afterUnknown)
	if err != nil {
		t.Fatal(err)
	}

	beforeJSON, err := json.MarshalIndent(before, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	afterJSON, err := json.MarshalIndent(after, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("BEFORE:")
	fmt.Println(string(beforeJSON))
	fmt.Println("\n\nAFTER:")
	fmt.Println(string(afterJSON))

	if val, _ := after.Get("int_value"); val != KnownAfterApply {
		t.Errorf("expected int_value to be %v, got %v", KnownAfterApply, val)
	}

	if val, _ := after.Get("bool_value"); val != KnownAfterApply {
		t.Errorf("expected bool_value to be %v, got %v", KnownAfterApply, val)
	}

	i, err := after.Get("list_value")
	if err != nil {
		t.Error(err)
	}

	if list, ok := i.([]interface{}); ok {
		if list[2] != KnownAfterApply {
			t.Errorf("expected third string_value to be %v, got %v", KnownAfterApply, list[2])
		}
	} else {
		t.Error("list_value is not a string slice")
	}

	if val, _ := after.Get("data"); val != KnownAfterApply {
		t.Errorf("expected data to be %v, got %v", KnownAfterApply, val)
	}
}
