package integrationtests

import (
	"fmt"
	"os"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/go/discovery"
	"github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	"github.com/overmindtech/cli/sources/azure/manual"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
)

// Gallery application version integration tests require pre-existing Azure resources
// (gallery, gallery application, and gallery application version) because creating
// a version requires a source blob URL. Set these env vars to run the tests:
//
//	AZURE_TEST_GALLERY_NAME                 - name of the gallery
//	AZURE_TEST_GALLERY_APPLICATION_NAME     - name of the gallery application
//	AZURE_TEST_GALLERY_APPLICATION_VERSION  - name of the gallery application version
//
// Optional: AZURE_TEST_GALLERY_RESOURCE_GROUP (defaults to overmind-integration-tests)
func getGalleryApplicationVersionTestConfig(t *testing.T) (resourceGroup, galleryName, applicationName, versionName string, skip bool) {
	galleryName = os.Getenv("AZURE_TEST_GALLERY_NAME")
	applicationName = os.Getenv("AZURE_TEST_GALLERY_APPLICATION_NAME")
	versionName = os.Getenv("AZURE_TEST_GALLERY_APPLICATION_VERSION")
	resourceGroup = os.Getenv("AZURE_TEST_GALLERY_RESOURCE_GROUP")
	if resourceGroup == "" {
		resourceGroup = integrationTestResourceGroup
	}
	if galleryName == "" || applicationName == "" || versionName == "" {
		t.Skip("Skipping gallery application version integration test: set AZURE_TEST_GALLERY_NAME, AZURE_TEST_GALLERY_APPLICATION_NAME, and AZURE_TEST_GALLERY_APPLICATION_VERSION to run")
		return "", "", "", "", true
	}
	return resourceGroup, galleryName, applicationName, versionName, false
}

func TestComputeGalleryApplicationVersionIntegration(t *testing.T) {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		t.Skip("AZURE_SUBSCRIPTION_ID environment variable not set")
	}

	resourceGroup, galleryName, applicationName, versionName, skip := getGalleryApplicationVersionTestConfig(t)
	if skip {
		return
	}

	cred, err := azureshared.NewAzureCredential(t.Context())
	if err != nil {
		t.Fatalf("Failed to create Azure credential: %v", err)
	}

	galleryApplicationVersionsClient, err := armcompute.NewGalleryApplicationVersionsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Gallery Application Versions client: %v", err)
	}

	rgClient, err := armresources.NewResourceGroupsClient(subscriptionID, cred, nil)
	if err != nil {
		t.Fatalf("Failed to create Resource Groups client: %v", err)
	}

	t.Run("Run", func(t *testing.T) {
		ctx := t.Context()

		// Ensure resource group exists (may be used for pre-created gallery)
		err := createResourceGroup(ctx, rgClient, resourceGroup, integrationTestLocation)
		if err != nil {
			t.Fatalf("Failed to create/verify resource group: %v", err)
		}

		t.Run("GetGalleryApplicationVersion", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Retrieving gallery application version %s/%s/%s in subscription %s, resource group %s",
				galleryName, applicationName, versionName, subscriptionID, resourceGroup)

			wrapper := manual.NewComputeGalleryApplicationVersion(
				clients.NewGalleryApplicationVersionsClient(galleryApplicationVersionsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(galleryName, applicationName, versionName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem == nil {
				t.Fatalf("Expected sdpItem to be non-nil")
			}

			uniqueAttrKey := sdpItem.GetUniqueAttribute()
			uniqueAttrValue, err := sdpItem.GetAttributes().Get(uniqueAttrKey)
			if err != nil {
				t.Fatalf("Failed to get unique attribute: %v", err)
			}

			expectedUniqueAttr := shared.CompositeLookupKey(galleryName, applicationName, versionName)
			if uniqueAttrValue != expectedUniqueAttr {
				t.Fatalf("Expected unique attribute value to be %s, got %s", expectedUniqueAttr, uniqueAttrValue)
			}

			log.Printf("Successfully retrieved gallery application version %s", versionName)
		})

		t.Run("SearchGalleryApplicationVersions", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Searching gallery application versions for gallery %s, application %s in subscription %s, resource group %s",
				galleryName, applicationName, subscriptionID, resourceGroup)

			wrapper := manual.NewComputeGalleryApplicationVersion(
				clients.NewGalleryApplicationVersionsClient(galleryApplicationVersionsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter does not support Search operation")
			}

			searchQuery := galleryName + shared.QuerySeparator + applicationName
			sdpItems, err := searchable.Search(ctx, scope, searchQuery, true)
			if err != nil {
				t.Fatalf("Failed to search gallery application versions: %v", err)
			}

			if len(sdpItems) < 1 {
				t.Fatalf("Expected at least one gallery application version, got %d", len(sdpItems))
			}

			var found bool
			expectedUniqueAttr := shared.CompositeLookupKey(galleryName, applicationName, versionName)
			for _, item := range sdpItems {
				uniqueAttrKey := item.GetUniqueAttribute()
				if v, err := item.GetAttributes().Get(uniqueAttrKey); err == nil && v == expectedUniqueAttr {
					found = true
					break
				}
			}

			if !found {
				t.Fatalf("Expected to find gallery application version %s in the search results", versionName)
			}

			log.Printf("Found %d gallery application versions in resource group %s", len(sdpItems), resourceGroup)
		})

		t.Run("VerifyItemAttributes", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying item attributes for gallery application version %s", versionName)

			wrapper := manual.NewComputeGalleryApplicationVersion(
				clients.NewGalleryApplicationVersionsClient(galleryApplicationVersionsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(galleryName, applicationName, versionName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			if sdpItem.GetType() != azureshared.ComputeGalleryApplicationVersion.String() {
				t.Errorf("Expected item type %s, got %s", azureshared.ComputeGalleryApplicationVersion.String(), sdpItem.GetType())
			}

			expectedScope := fmt.Sprintf("%s.%s", subscriptionID, resourceGroup)
			if sdpItem.GetScope() != expectedScope {
				t.Errorf("Expected scope %s, got %s", expectedScope, sdpItem.GetScope())
			}

			if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
				t.Errorf("Expected unique attribute 'uniqueAttr', got %s", sdpItem.GetUniqueAttribute())
			}

			if err := sdpItem.Validate(); err != nil {
				t.Fatalf("Item validation failed: %v", err)
			}

			log.Printf("Verified item attributes for gallery application version %s", versionName)
		})

		t.Run("VerifyLinkedItems", func(t *testing.T) {
			ctx := t.Context()

			log.Printf("Verifying linked items for gallery application version %s", versionName)

			wrapper := manual.NewComputeGalleryApplicationVersion(
				clients.NewGalleryApplicationVersionsClient(galleryApplicationVersionsClient),
				[]azureshared.ResourceGroupScope{azureshared.NewResourceGroupScope(subscriptionID, resourceGroup)},
			)
			scope := wrapper.Scopes()[0]

			adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())
			query := shared.CompositeLookupKey(galleryName, applicationName, versionName)
			sdpItem, qErr := adapter.Get(ctx, scope, query, true)
			if qErr != nil {
				t.Fatalf("Expected no error, got: %v", qErr)
			}

			linkedQueries := sdpItem.GetLinkedItemQueries()
			log.Printf("Found %d linked item queries for gallery application version %s", len(linkedQueries), versionName)

			// Should have at least Gallery and Gallery Application parent links
			if len(linkedQueries) < 2 {
				t.Fatalf("Expected at least 2 linked item queries (Gallery, Gallery Application), got %d", len(linkedQueries))
			}

			for _, liq := range linkedQueries {
				query := liq.GetQuery()
				if query == nil {
					t.Error("Linked item query has nil Query")
					continue
				}

				if query.GetType() == "" {
					t.Error("Linked item query has empty Type")
				}
				if query.GetMethod() != sdp.QueryMethod_GET && query.GetMethod() != sdp.QueryMethod_SEARCH {
					t.Errorf("Linked item query has unexpected Method: %v", query.GetMethod())
				}
				if query.GetQuery() == "" {
					t.Error("Linked item query has empty Query")
				}
				if query.GetScope() == "" {
					t.Error("Linked item query has empty Scope")
				}
			}
		})
	})
}
