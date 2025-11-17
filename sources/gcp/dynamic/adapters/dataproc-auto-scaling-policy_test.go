package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/dataproc/v2/apiv1/dataprocpb"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestDataprocAutoscalingPolicy(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	region := "us-central1"
	linker := gcpshared.NewLinker()
	policyName := "test-policy"

	policy := &dataprocpb.AutoscalingPolicy{
		Id:   policyName,
		Name: fmt.Sprintf("projects/%s/regions/%s/autoscalingPolicies/%s", projectID, region, policyName),
		Algorithm: &dataprocpb.AutoscalingPolicy_BasicAlgorithm{
			BasicAlgorithm: &dataprocpb.BasicAutoscalingAlgorithm{
				Config: &dataprocpb.BasicAutoscalingAlgorithm_YarnConfig{
					YarnConfig: &dataprocpb.BasicYarnAutoscalingConfig{
						GracefulDecommissionTimeout: durationpb.New(300_000_000_000), // 300s
						ScaleUpFactor:               0.8,
						ScaleDownFactor:             0.5,
						ScaleUpMinWorkerFraction:    0.1,
						ScaleDownMinWorkerFraction:  0.05,
					},
				},
				CooldownPeriod: durationpb.New(120_000_000_000), // 120s
			},
		},
		WorkerConfig: &dataprocpb.InstanceGroupAutoscalingPolicyConfig{
			MinInstances: 2,
			MaxInstances: 10,
			Weight:       2,
		},
		SecondaryWorkerConfig: &dataprocpb.InstanceGroupAutoscalingPolicyConfig{
			MinInstances: 0,
			MaxInstances: 5,
			Weight:       1,
		},
		Labels: map[string]string{
			"environment": "test",
			"team":        "engineering",
		},
	}

	policyName2 := "test-policy-2"
	policy2 := &dataprocpb.AutoscalingPolicy{
		Id:   policyName2,
		Name: fmt.Sprintf("projects/%s/regions/%s/autoscalingPolicies/%s", projectID, region, policyName2),
		Algorithm: &dataprocpb.AutoscalingPolicy_BasicAlgorithm{
			BasicAlgorithm: &dataprocpb.BasicAutoscalingAlgorithm{
				Config: &dataprocpb.BasicAutoscalingAlgorithm_YarnConfig{
					YarnConfig: &dataprocpb.BasicYarnAutoscalingConfig{
						GracefulDecommissionTimeout: durationpb.New(600_000_000_000), // 600s
						ScaleUpFactor:               1.0,
						ScaleDownFactor:             0.3,
					},
				},
				CooldownPeriod: durationpb.New(180_000_000_000), // 180s
			},
		},
		WorkerConfig: &dataprocpb.InstanceGroupAutoscalingPolicyConfig{
			MinInstances: 3,
			MaxInstances: 15,
			Weight:       1,
		},
	}

	policyList := &dataprocpb.ListAutoscalingPoliciesResponse{
		Policies: []*dataprocpb.AutoscalingPolicy{policy, policy2},
	}

	sdpItemType := gcpshared.DataprocAutoscalingPolicy

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://dataproc.googleapis.com/v1/projects/%s/regions/%s/autoscalingPolicies/%s", projectID, region, policyName): {
			StatusCode: http.StatusOK,
			Body:       policy,
		},
		fmt.Sprintf("https://dataproc.googleapis.com/v1/projects/%s/regions/%s/autoscalingPolicies/%s", projectID, region, policyName2): {
			StatusCode: http.StatusOK,
			Body:       policy2,
		},
		fmt.Sprintf("https://dataproc.googleapis.com/v1/projects/%s/regions/%s/autoscalingPolicies", projectID, region): {
			StatusCode: http.StatusOK,
			Body:       policyList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, fmt.Sprintf("%s.%s", projectID, region), policyName, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != policyName {
			t.Errorf("Expected unique attribute value '%s', got %s", policyName, sdpItem.UniqueAttributeValue())
		}

		// Skip static tests - no blast propagations for this adapter
	})

	t.Run("List", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := listable.List(ctx, fmt.Sprintf("%s.%s", projectID, region), true)
		if err != nil {
			t.Fatalf("Failed to list resources: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://dataproc.googleapis.com/v1/projects/%s/regions/%s/autoscalingPolicies/%s", projectID, region, policyName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Policy not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, fmt.Sprintf("%s.%s", projectID, region), policyName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
