package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/essentialcontacts/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestEssentialContactsContact(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	contactID := "test-contact"

	contact := &essentialcontacts.GoogleCloudEssentialcontactsV1Contact{
		Name:  fmt.Sprintf("projects/%s/contacts/%s", projectID, contactID),
		Email: "admin@example.com",
	}

	contactList := &essentialcontacts.GoogleCloudEssentialcontactsV1ListContactsResponse{
		Contacts: []*essentialcontacts.GoogleCloudEssentialcontactsV1Contact{contact},
	}

	sdpItemType := gcpshared.EssentialContactsContact

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://essentialcontacts.googleapis.com/v1/projects/%s/contacts/%s", projectID, contactID): {
			StatusCode: http.StatusOK,
			Body:       contact,
		},
		fmt.Sprintf("https://essentialcontacts.googleapis.com/v1/projects/%s/contacts", projectID): {
			StatusCode: http.StatusOK,
			Body:       contactList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, contactID, true)
		if err != nil {
			t.Fatalf("Failed to get contact: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
	})

	t.Run("List", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list contacts: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 contact, got %d", len(sdpItems))
		}
	})

	t.Run("Search with Terraform format", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		// Test Terraform format: projects/[project]/contacts/[contact]
		terraformQuery := fmt.Sprintf("projects/%s/contacts/%s", projectID, contactID)
		sdpItems, err := searchable.Search(ctx, projectID, terraformQuery, true)
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
		if firstItem.GetScope() != projectID {
			t.Errorf("Expected first item scope '%s', got %s", projectID, firstItem.GetScope())
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://essentialcontacts.googleapis.com/v1/projects/%s/contacts/%s", projectID, contactID): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Contact not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, contactID, true)
		if err == nil {
			t.Error("Expected error when getting non-existent contact, but got nil")
		}
	})
}
