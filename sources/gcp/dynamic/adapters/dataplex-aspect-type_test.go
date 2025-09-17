package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"cloud.google.com/go/dataplex/apiv1/dataplexpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestDataplexAspectType(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	location := "us-central1"
	linker := gcpshared.NewLinker()
	aspectTypeName := "test-aspect-type"

	// Mock AspectType using proper GCP SDK types
	aspectType := &dataplexpb.AspectType{
		Name:        fmt.Sprintf("projects/%s/locations/%s/aspectTypes/%s", projectID, location, aspectTypeName),
		Uid:         "12345678-1234-1234-1234-123456789012",
		CreateTime:  timestamppb.New(mustParseTime("2023-01-15T10:30:00.000Z")),
		UpdateTime:  timestamppb.New(mustParseTime("2023-01-16T14:20:00.000Z")),
		DisplayName: "Test Aspect Type",
		Description: "A test aspect type for unit testing",
		Labels: map[string]string{
			"env":  "test",
			"team": "data-platform",
		},
		Etag: "BwWWja0YfJA=",
	}

	// Create a second aspect type for list testing
	aspectTypeName2 := "test-aspect-type-2"
	aspectType2 := &dataplexpb.AspectType{
		Name:        fmt.Sprintf("projects/%s/locations/%s/aspectTypes/%s", projectID, location, aspectTypeName2),
		Uid:         "87654321-4321-4321-4321-210987654321",
		CreateTime:  timestamppb.New(mustParseTime("2023-01-17T09:15:00.000Z")),
		UpdateTime:  timestamppb.New(mustParseTime("2023-01-17T16:45:00.000Z")),
		DisplayName: "Second Test Aspect Type",
		Description: "A second test aspect type for list testing",
		Labels: map[string]string{
			"env":  "prod",
			"team": "analytics",
		},
		Etag: "CwXXkb1ZgKB=",
	}

	// Create the list response using a map structure instead of the protobuf ListAspectTypesResponse
	// This is necessary because the dynamic adapter expects JSON-serializable structures
	// Individual items use proper SDK types, but the list wrapper uses a simple map
	aspectTypesList := map[string]interface{}{
		"aspectTypes": []interface{}{aspectType, aspectType2},
	}

	sdpItemType := gcpshared.DataplexAspectType

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://dataplex.googleapis.com/v1/projects/%s/locations/%s/aspectTypes/%s", projectID, location, aspectTypeName): {
			StatusCode: http.StatusOK,
			Body:       aspectType,
		},
		fmt.Sprintf("https://dataplex.googleapis.com/v1/projects/%s/locations/%s/aspectTypes", projectID, location): {
			StatusCode: http.StatusOK,
			Body:       aspectTypesList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := fmt.Sprintf("%s|%s", location, aspectTypeName)
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get Dataplex aspect type: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != getQuery {
			t.Errorf("Expected unique attribute value '%s', got %s", getQuery, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetScope() != projectID {
			t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
		}

		// Test key attributes (using snake_case as shown in debug output)
		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		expectedName := fmt.Sprintf("projects/%s/locations/%s/aspectTypes/%s", projectID, location, aspectTypeName)
		if val != expectedName {
			t.Errorf("Expected name field to be '%s', got %s", expectedName, val)
		}

		val, err = sdpItem.GetAttributes().Get("displayName")
		if err != nil {
			t.Fatalf("Failed to get 'displayName' attribute: %v", err)
		}
		if val != "Test Aspect Type" {
			t.Errorf("Expected displayName field to be 'Test Aspect Type', got %s", val)
		}

		val, err = sdpItem.GetAttributes().Get("description")
		if err != nil {
			t.Fatalf("Failed to get 'description' attribute: %v", err)
		}
		if val != "A test aspect type for unit testing" {
			t.Errorf("Expected description field to be 'A test aspect type for unit testing', got %s", val)
		}

		val, err = sdpItem.GetAttributes().Get("uid")
		if err != nil {
			t.Fatalf("Failed to get 'uid' attribute: %v", err)
		}
		if val != "12345678-1234-1234-1234-123456789012" {
			t.Errorf("Expected uid field to be '12345678-1234-1234-1234-123456789012', got %s", val)
		}

		// Note: createTime and updateTime are struct values (timestamps), not simple strings
		// Testing their presence rather than exact format
		_, err = sdpItem.GetAttributes().Get("createTime")
		if err != nil {
			t.Fatalf("Failed to get 'createTime' attribute: %v", err)
		}

		_, err = sdpItem.GetAttributes().Get("updateTime")
		if err != nil {
			t.Fatalf("Failed to get 'updateTime' attribute: %v", err)
		}

		val, err = sdpItem.GetAttributes().Get("etag")
		if err != nil {
			t.Fatalf("Failed to get 'etag' attribute: %v", err)
		}
		if val != "BwWWja0YfJA=" {
			t.Errorf("Expected etag field to be 'BwWWja0YfJA=', got %s", val)
		}

		// Note: Since this adapter doesn't define blast propagation relationships,
		// we don't run StaticTests here. The adapter's blastPropagation map is empty,
		// which is correct as AspectTypes are schema definitions rather than runtime resources.
	})

	t.Run("Search", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(gcpshared.DataplexAspectType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter is not a SearchableAdapter")
		}

		sdpItems, err := searchable.Search(ctx, projectID, location, true)
		if err != nil {
			t.Fatalf("Failed to search Dataplex aspect types: %v", err)
		}

		// Verify the first item
		firstItem := sdpItems[0]
		if firstItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected first item type %s, got %s", sdpItemType.String(), firstItem.GetType())
		}
		if firstItem.GetScope() != projectID {
			t.Errorf("Expected first item scope '%s', got %s", projectID, firstItem.GetScope())
		}

		// Verify the second item
		secondItem := sdpItems[1]
		if secondItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected second item type %s, got %s", sdpItemType.String(), secondItem.GetType())
		}
		if secondItem.GetScope() != projectID {
			t.Errorf("Expected second item scope '%s', got %s", projectID, secondItem.GetScope())
		}
	})

	t.Run("Search with Terraform format", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(gcpshared.DataplexAspectType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter is not a SearchableAdapter")
		}

		// Test Terraform format: projects/[project_id]/locations/[location]/aspectTypes/[aspect_type_id]
		// The adapter should extract the location from this format and search in that location
		terraformQuery := fmt.Sprintf("projects/%s/locations/%s/aspectTypes/%s", projectID, location, aspectTypeName)
		sdpItems, err := searchable.Search(ctx, projectID, terraformQuery, true)
		if err != nil {
			t.Fatalf("Failed to search Dataplex aspect types with Terraform format: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 Dataplex aspect types with Terraform format, got %d", len(sdpItems))
		}

		// Verify the first item
		firstItem := sdpItems[0]
		if firstItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected first item type %s, got %s", sdpItemType.String(), firstItem.GetType())
		}
		if firstItem.GetScope() != projectID {
			t.Errorf("Expected first item scope '%s', got %s", projectID, firstItem.GetScope())
		}
		expectedFirstUniqueAttr := fmt.Sprintf("%s|%s", location, aspectTypeName)
		if firstItem.UniqueAttributeValue() != expectedFirstUniqueAttr {
			t.Errorf("Expected first item unique attribute '%s', got %s", expectedFirstUniqueAttr, firstItem.UniqueAttributeValue())
		}
	})

	t.Run("Error handling", func(t *testing.T) {
		// Test 404 error handling
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://dataplex.googleapis.com/v1/projects/%s/locations/%s/aspectTypes/nonexistent", projectID, location): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": map[string]interface{}{"code": 404, "message": "AspectType not found"}},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := fmt.Sprintf("%s|nonexistent", location)
		_, err = adapter.Get(ctx, projectID, getQuery, true)
		if err == nil {
			t.Error("Expected error when getting nonexistent aspect type, got nil")
		}
	})
}

// Helper function to parse time strings
func mustParseTime(timeStr string) time.Time {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse time %s: %v", timeStr, err))
	}
	return t
}
