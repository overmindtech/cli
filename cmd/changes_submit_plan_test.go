package cmd

import (
	"context"
	"testing"

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
