package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	resourcemanagerpb "cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestCloudResourceManagerTagKey(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	tagKeyID := "123456789"

	// Mock TagKey response using protobuf types from GCP Go SDK
	// Reference: https://cloud.google.com/resource-manager/reference/rest/v3/tagKeys#TagKey
	tagKey := &resourcemanagerpb.TagKey{
		Name:           fmt.Sprintf("tagKeys/%s", tagKeyID),
		Parent:         fmt.Sprintf("projects/%s", projectID),
		ShortName:      "environment",
		NamespacedName: fmt.Sprintf("%s/environment", projectID),
		Description:    "Environment classification for resources",
		CreateTime:     timestamppb.New(mustParseTime("2023-01-15T10:30:00.000Z")),
		UpdateTime:     timestamppb.New(mustParseTime("2023-01-15T10:30:00.000Z")),
		Etag:           "BwXhqhCKJvM=",
		Purpose:        resourcemanagerpb.Purpose_GCE_FIREWALL,
		PurposeData: map[string]string{
			"network": fmt.Sprintf("projects/%s/global/networks/default", projectID),
		},
	}

	// Create a second TagKey for list testing
	tagKeyID2 := "987654321"
	tagKey2 := &resourcemanagerpb.TagKey{
		Name:           fmt.Sprintf("tagKeys/%s", tagKeyID2),
		Parent:         fmt.Sprintf("projects/%s", projectID),
		ShortName:      "team",
		NamespacedName: fmt.Sprintf("%s/team", projectID),
		Description:    "Team ownership for resources",
		CreateTime:     timestamppb.New(mustParseTime("2023-01-16T11:45:00.000Z")),
		UpdateTime:     timestamppb.New(mustParseTime("2023-01-16T11:45:00.000Z")),
		Etag:           "BwXhqhCKJvN=",
	}

	// Mock list response structure using protobuf types
	tagKeys := &resourcemanagerpb.ListTagKeysResponse{
		TagKeys: []*resourcemanagerpb.TagKey{tagKey, tagKey2},
	}

	sdpItemType := gcpshared.CloudResourceManagerTagKey

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/tagKeys/%s", tagKeyID): {
			StatusCode: http.StatusOK,
			Body:       tagKey,
		},
		fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/tagKeys?parent=projects/%s", projectID): {
			StatusCode: http.StatusOK,
			Body:       tagKeys,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := tagKeyID
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get TagKey: %v", err)
		}

		// Validate basic SDP item properties
		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != getQuery {
			t.Errorf("Expected unique attribute value '%s', got %s", tagKeyID, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetScope() != projectID {
			t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
		}

		// Validate specific TagKey attributes
		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		expectedName := fmt.Sprintf("tagKeys/%s", tagKeyID)
		if val != expectedName {
			t.Errorf("Expected name field to be '%s', got %s", expectedName, val)
		}

		val, err = sdpItem.GetAttributes().Get("parent")
		if err != nil {
			t.Fatalf("Failed to get 'parent' attribute: %v", err)
		}
		expectedParent := fmt.Sprintf("projects/%s", projectID)
		if val != expectedParent {
			t.Errorf("Expected parent field to be '%s', got %s", expectedParent, val)
		}

		val, err = sdpItem.GetAttributes().Get("shortName")
		if err != nil {
			t.Fatalf("Failed to get 'shortName' attribute: %v", err)
		}
		if val != "environment" {
			t.Errorf("Expected shortName field to be 'environment', got %s", val)
		}

		val, err = sdpItem.GetAttributes().Get("namespacedName")
		if err != nil {
			t.Fatalf("Failed to get 'namespacedName' attribute: %v", err)
		}
		expectedNamespacedName := fmt.Sprintf("%s/environment", projectID)
		if val != expectedNamespacedName {
			t.Errorf("Expected namespacedName field to be '%s', got %s", expectedNamespacedName, val)
		}

		val, err = sdpItem.GetAttributes().Get("description")
		if err != nil {
			t.Fatalf("Failed to get 'description' attribute: %v", err)
		}
		if val != "Environment classification for resources" {
			t.Errorf("Expected description field to be 'Environment classification for resources', got %s", val)
		}

		val, err = sdpItem.GetAttributes().Get("createTime")
		if err != nil {
			t.Fatalf("Failed to get 'createTime' attribute: %v", err)
		}
		if val != "2023-01-15T10:30:00Z" {
			t.Errorf("Expected createTime field to be '2023-01-15T10:30:00Z', got %s", val)
		}

		val, err = sdpItem.GetAttributes().Get("updateTime")
		if err != nil {
			t.Fatalf("Failed to get 'updateTime' attribute: %v", err)
		}
		if val != "2023-01-15T10:30:00Z" {
			t.Errorf("Expected updateTime field to be '2023-01-15T10:30:00Z', got %s", val)
		}

		val, err = sdpItem.GetAttributes().Get("etag")
		if err != nil {
			t.Fatalf("Failed to get 'etag' attribute: %v", err)
		}
		if val != "BwXhqhCKJvM=" {
			t.Errorf("Expected etag field to be 'BwXhqhCKJvM=', got %s", val)
		}

		val, err = sdpItem.GetAttributes().Get("purpose")
		if err != nil {
			t.Fatalf("Failed to get 'purpose' attribute: %v", err)
		}
		if val != "GCE_FIREWALL" {
			t.Errorf("Expected purpose field to be 'GCE_FIREWALL', got %s", val)
		}

		// Test nested purposeData structure
		val, err = sdpItem.GetAttributes().Get("purposeData")
		if err != nil {
			t.Fatalf("Failed to get 'purposeData' attribute: %v", err)
		}
		purposeData, ok := val.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected purposeData to be a map, got %T", val)
		}
		networkVal, exists := purposeData["network"]
		if !exists {
			t.Errorf("Expected purposeData to contain 'network' field")
		} else {
			expectedNetwork := fmt.Sprintf("projects/%s/global/networks/default", projectID)
			if networkVal != expectedNetwork {
				t.Errorf("Expected purposeData.network to be '%s', got %s", expectedNetwork, networkVal)
			}
		}

		// Note: Since this adapter doesn't define blast propagation relationships,
		// we don't run StaticTests here. The adapter's blastPropagation map is empty,
		// which is correct as TagKeys are configuration resources rather than runtime resources.
	})

	t.Run("List", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(gcpshared.CloudResourceManagerTagKey, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter is not a ListableAdapter")
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list TagKeys: %v", err)
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

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test error handling for HTTP errors
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://cloudresourcemanager.googleapis.com/v3/tagKeys/%s", tagKeyID): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": map[string]interface{}{"code": 404, "message": "TagKey not found"}},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, tagKeyID, true)
		if err == nil {
			t.Errorf("Expected error for 404 response, got nil")
		}
	})
}
