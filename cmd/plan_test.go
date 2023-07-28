package cmd

import "testing"

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
