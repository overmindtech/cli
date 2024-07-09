package cmd

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
	_, err := mappedItemDiffsFromPlanFile(context.Background(), "testdata/state.json", logrus.Fields{})

	if err == nil {
		t.Error("Expected error when running with state file, got none")
	}
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
				"foo": "REDACTED",
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
				"foo": "REDACTED",
			},
			maskSensitiveData(
				map[string]any{
					"foo": map[string]any{"key": "bar"},
				},
				map[string]any{"foo": true}))

		require.Equal(t,
			map[string]any{
				"foo": map[string]any{"key": "REDACTED"},
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
				"foo": "REDACTED",
			},
			maskSensitiveData(
				map[string]any{
					"foo": []any{"one", "two"},
				},
				map[string]any{"foo": true}))

		require.Equal(t,
			map[string]any{
				"foo": []any{"one", "REDACTED"},
			},
			maskSensitiveData(
				map[string]any{
					"foo": []any{"one", "two"},
				},
				map[string]any{"foo": []any{false, true}}))

	})
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
