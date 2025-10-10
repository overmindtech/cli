package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/cloudbilling/v1"

	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestCloudBillingBillingInfo(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()

	billingInfo := &cloudbilling.ProjectBillingInfo{
		Name:               fmt.Sprintf("projects/%s/billingInfo", projectID),
		ProjectId:          projectID,
		BillingAccountName: "billingAccounts/012345-ABCDEF-678901",
		BillingEnabled:     true,
	}

	sdpItemType := gcpshared.CloudBillingBillingInfo

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://cloudbilling.googleapis.com/v1/projects/%s/billingInfo", projectID): {
			StatusCode: http.StatusOK,
			Body:       billingInfo,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, projectID, true)
		if err != nil {
			t.Fatalf("Failed to get billing info: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}

		// Note: StaticTests skipped because ProjectBillingInfo doesn't expose proper unique attribute
		// This is a limitation of the API response structure
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://cloudbilling.googleapis.com/v1/projects/%s/billingInfo", projectID): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Billing info not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, projectID, true)
		if err == nil {
			t.Error("Expected error when getting non-existent billing info, but got nil")
		}
	})
}
