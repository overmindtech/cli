package example_test

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/example"
	"github.com/overmindtech/cli/sources/example/mocks"
	"github.com/overmindtech/cli/sources/shared"
)

func TestStandardSearchableListable(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	defer ctrl.Finish()

	mockExternalAPIClient := mocks.NewMockExternalAPIClient(ctrl)

	projectID := "test-project-id"
	zone := "us-central1-a"

	t.Run("Get", func(t *testing.T) {
		searchableListable := example.NewStandardSearchableListable(mockExternalAPIClient, projectID, zone)

		// Mock the Get method to return a specific ExternalType
		mockExternalAPIClient.EXPECT().Get(ctx, "test-id").Return(&example.ExternalType{
			Type:            "test-type",
			UniqueAttribute: "test-unique-attribute",
			Tags:            map[string]string{"address": "test-address"},
			LinkedItemID:    "test-link-me",
		}, nil)

		item, err := searchableListable.Get(ctx, "test-id")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if item == nil {
			t.Fatalf("Expected item, got nil")
		}

		if item.GetType() != "test-type" {
			t.Fatalf("Expected type 'test-type', got: %s", item.GetType())
		}

		if item.GetUniqueAttribute() != "test-unique-attribute" {
			t.Fatalf("Expected unique attribute 'test-unique-attribute', got: %s", item.GetUniqueAttribute())
		}

		if item.GetTags()["address"] != "test-address" {
			t.Fatalf("Expected address 'test-address', got: %s", item.GetTags()["address"])
		}

		linkedItemQuery := item.GetLinkedItemQueries()[0].GetQuery()
		var potentialLinkedItem shared.ItemType
		for v := range searchableListable.PotentialLinks() {
			potentialLinkedItem = v
		}

		if linkedItemQuery.GetType() != potentialLinkedItem.String() {
			t.Fatalf("Expected linked item type '%s', got: %s", potentialLinkedItem.String(), linkedItemQuery.GetType())
		}
	})

	t.Run("GetNotFound", func(t *testing.T) {
		searchableListable := example.NewStandardSearchableListable(mockExternalAPIClient, projectID, zone)

		// Mock the Get method to return a NotFoundError
		mockExternalAPIClient.EXPECT().Get(ctx, "test-id").Return(nil, example.NotFoundError{})

		item, err := searchableListable.Get(ctx, "test-id")
		if err == nil {
			t.Fatalf("Expected error, got: %v", item)
		}

		if err.GetErrorString() != new(example.NotFoundError).Error() {
			t.Fatalf("Expected NotFoundError, got: %v", err)
		}

		if err.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Fatalf("Expected error type NOT_FOUND, got: %v", err.GetErrorType())
		}

		if item != nil {
			t.Fatalf("Expected nil item, got: %v", item)
		}
	})

	t.Run("List", func(t *testing.T) {
		searchableListable := example.NewStandardSearchableListable(mockExternalAPIClient, projectID, zone)

		// Mock the List method to return a list of ExternalType
		mockExternalAPIClient.EXPECT().List(ctx).Return([]*example.ExternalType{
			{
				Type:            "test-type",
				UniqueAttribute: "test-unique-attribute",
				Tags:            map[string]string{"address": "test-address"},
				LinkedItemID:    "test-link-me",
			},
		}, nil)

		items, err := searchableListable.List(ctx)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(items) == 0 {
			t.Fatalf("Expected items, got empty list")
		}

		if items[0].GetType() != "test-type" {
			t.Fatalf("Expected type 'test-type', got: %s", items[0].GetType())
		}

		if items[0].GetUniqueAttribute() != "test-unique-attribute" {
			t.Fatalf("Expected unique attribute 'test-unique-attribute', got: %s", items[0].GetUniqueAttribute())
		}

		if items[0].GetTags()["address"] != "test-address" {
			t.Fatalf("Expected address 'test-address', got: %s", items[0].GetTags()["address"])
		}
	})

	t.Run("ListNotFound", func(t *testing.T) {
		searchableListable := example.NewStandardSearchableListable(mockExternalAPIClient, projectID, zone)

		// Mock the List method to return a NotFoundError
		mockExternalAPIClient.EXPECT().List(ctx).Return(nil, example.NotFoundError{})

		items, err := searchableListable.List(ctx)
		if err == nil {
			t.Fatalf("Expected error, got: %v", items)
		}

		if err.GetErrorString() != new(example.NotFoundError).Error() {
			t.Fatalf("Expected NotFoundError, got: %v", err)
		}

		if err.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Fatalf("Expected error type NOT_FOUND, got: %v", err.GetErrorType())
		}

		if items != nil {
			t.Fatalf("Expected nil items, got: %v", items)
		}
	})

	t.Run("Search", func(t *testing.T) {
		searchableListable := example.NewStandardSearchableListable(mockExternalAPIClient, projectID, zone)
		// Mock the Search method to return a list of ExternalType
		mockExternalAPIClient.EXPECT().Search(ctx, "test-query").Return([]*example.ExternalType{
			{
				Type:            "test-type",
				UniqueAttribute: "test-unique-attribute",
				Tags:            map[string]string{"address": "test-address"},
				LinkedItemID:    "test-link-me",
			},
		}, nil)

		items, err := searchableListable.Search(ctx, "test-query")
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(items) == 0 {
			t.Fatalf("Expected items, got empty list")
		}

		if items[0].GetType() != "test-type" {
			t.Fatalf("Expected type 'test-type', got: %s", items[0].GetType())
		}

		if items[0].GetUniqueAttribute() != "test-unique-attribute" {
			t.Fatalf("Expected unique attribute 'test-unique-attribute', got: %s", items[0].GetUniqueAttribute())
		}

		if items[0].GetTags()["address"] != "test-address" {
			t.Fatalf("Expected address 'test-address', got: %s", items[0].GetTags()["address"])
		}
	})

	t.Run("SearchNotFound", func(t *testing.T) {
		searchableListable := example.NewStandardSearchableListable(mockExternalAPIClient, projectID, zone)

		// Mock the Search method to return a NotFoundError
		mockExternalAPIClient.EXPECT().Search(ctx, "test-query").Return(nil, example.NotFoundError{})

		items, err := searchableListable.Search(ctx, "test-query")
		if err == nil {
			t.Fatalf("Expected error, got: %v", items)
		}

		if err.GetErrorString() != new(example.NotFoundError).Error() {
			t.Fatalf("Expected NotFoundError, got: %v", err)
		}

		if err.GetErrorType() != sdp.QueryError_NOTFOUND {
			t.Fatalf("Expected error type NOT_FOUND, got: %v", err.GetErrorType())
		}

		if items != nil {
			t.Fatalf("Expected nil items, got: %v", items)
		}
	})
}
