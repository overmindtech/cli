package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"

	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestComputeTargetPool(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	region := "us-central1"
	linker := gcpshared.NewLinker()
	poolName := "test-target-pool"
	zone := "us-central1-a"

	instance1URL := fmt.Sprintf("projects/%s/zones/%s/instances/instance-1", projectID, zone)
	instance2URL := fmt.Sprintf("projects/%s/zones/%s/instances/instance-2", projectID, zone)
	healthCheck1URL := fmt.Sprintf("projects/%s/global/healthChecks/health-check-1", projectID)
	healthCheck2URL := fmt.Sprintf("projects/%s/global/healthChecks/health-check-2", projectID)
	backupPoolURL := fmt.Sprintf("projects/%s/regions/%s/targetPools/backup-pool", projectID, region)

	pool := &computepb.TargetPool{
		Name: &poolName,
		Instances: []string{
			instance1URL,
			instance2URL,
		},
		HealthChecks: []string{
			healthCheck1URL,
			healthCheck2URL,
		},
		BackupPool: &backupPoolURL,
	}

	poolName2 := "test-target-pool-2"
	pool2 := &computepb.TargetPool{
		Name: &poolName2,
	}

	poolList := &computepb.TargetPoolList{
		Items: []*computepb.TargetPool{pool, pool2},
	}

	sdpItemType := gcpshared.ComputeTargetPool

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/targetPools/%s", projectID, region, poolName): {
			StatusCode: http.StatusOK,
			Body:       pool,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/targetPools/%s", projectID, region, poolName2): {
			StatusCode: http.StatusOK,
			Body:       pool2,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/targetPools", projectID, region): {
			StatusCode: http.StatusOK,
			Body:       poolList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, fmt.Sprintf("%s.%s", projectID, region), poolName, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != poolName {
			t.Errorf("Expected unique attribute value '%s', got %s", poolName, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// Instance 1 link
				{
					ExpectedType:   gcpshared.ComputeInstance.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "instance-1",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Instance 2 link
				{
					ExpectedType:   gcpshared.ComputeInstance.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "instance-2",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, zone),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Health check 1 link
				{
					ExpectedType:   gcpshared.ComputeHealthCheck.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "health-check-1",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Health check 2 link
				{
					ExpectedType:   gcpshared.ComputeHealthCheck.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "health-check-2",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Backup pool link
				{
					ExpectedType:   gcpshared.ComputeTargetPool.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "backup-pool",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}
			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("List", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID, region)
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

	t.Run("Search with Terraform format", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		// Test Terraform format: projects/[project]/regions/[region]/targetPools/[name]
		terraformQuery := fmt.Sprintf("projects/%s/regions/%s/targetPools/%s", projectID, region, poolName)
		sdpItems, err := searchable.Search(ctx, fmt.Sprintf("%s.%s", projectID, region), terraformQuery, true)
		if err != nil {
			t.Fatalf("Failed to search resources with Terraform format: %v", err)
		}

		// The search should return only the specific resource matching the Terraform format
		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 resource, got %d", len(sdpItems))
			return
		}

		// Verify the single item returned
		firstItem := sdpItems[0]
		if firstItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected first item type %s, got %s", sdpItemType.String(), firstItem.GetType())
		}
		if firstItem.GetScope() != fmt.Sprintf("%s.%s", projectID, region) {
			t.Errorf("Expected first item scope '%s.%s', got %s", projectID, region, firstItem.GetScope())
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/targetPools/%s", projectID, region, poolName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Target pool not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, fmt.Sprintf("%s.%s", projectID, region), poolName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
