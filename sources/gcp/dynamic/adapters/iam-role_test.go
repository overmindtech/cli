package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/iam/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestIAMRole(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	roleName := "customRole"

	role := &iam.Role{
		Name:  fmt.Sprintf("projects/%s/roles/%s", projectID, roleName),
		Title: "Custom Role",
	}

	roleList := &iam.ListRolesResponse{
		Roles: []*iam.Role{role},
	}

	sdpItemType := gcpshared.IAMRole

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://iam.googleapis.com/v1/projects/%s/roles/%s", projectID, roleName): {
			StatusCode: http.StatusOK,
			Body:       role,
		},
		fmt.Sprintf("https://iam.googleapis.com/v1/projects/%s/roles", projectID): {
			StatusCode: http.StatusOK,
			Body:       roleList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, roleName, true)
		if err != nil {
			t.Fatalf("Failed to get IAM role: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
	})

	t.Run("List", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list IAM roles: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 IAM role, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://iam.googleapis.com/v1/projects/%s/roles/%s", projectID, roleName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Role not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, roleName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent IAM role, but got nil")
		}
	})
}
