package manual_test

import (
	"context"
	"testing"

	certificatemanagerpb "cloud.google.com/go/certificatemanager/apiv1/certificatemanagerpb"
	"go.uber.org/mock/gomock"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/overmindtech/workspace/discovery"
	"github.com/overmindtech/workspace/sdp-go"
	"github.com/overmindtech/workspace/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/gcp/shared/mocks"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

func createCertificate(projectID, location, name string) *certificatemanagerpb.Certificate {
	return &certificatemanagerpb.Certificate{
		Name:        "projects/" + projectID + "/locations/" + location + "/certificates/" + name,
		Description: "Test certificate",
		CreateTime:  timestamppb.Now(),
		UpdateTime:  timestamppb.Now(),
		Labels: map[string]string{
			"env": "test",
		},
		Scope: certificatemanagerpb.Certificate_DEFAULT,
	}
}

func TestCertificateManagerCertificate(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockCertificateManagerCertificateClient(ctrl)
	projectID := "test-project-id"
	location := "us-central1"
	certificateName := "test-certificate"

	t.Run("Get", func(t *testing.T) {
		wrapper := manual.NewCertificateManagerCertificate(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		mockClient.EXPECT().GetCertificate(ctx, gomock.Any()).Return(createCertificate(projectID, location, certificateName), nil)

		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		sdpItem, qErr := adapter.Get(ctx, wrapper.Scopes()[0], location+shared.QuerySeparator+certificateName, true)
		if qErr != nil {
			t.Fatalf("Expected no error, got: %v", qErr)
		}

		if sdpItem == nil {
			t.Fatal("Expected item, got nil")
		}

		if err := sdpItem.Validate(); err != nil {
			t.Fatalf("Expected no validation error, got: %v", err)
		}

		// Verify the item type
		if sdpItem.GetType() != gcpshared.CertificateManagerCertificate.String() {
			t.Errorf("Expected type %s, got: %s", gcpshared.CertificateManagerCertificate.String(), sdpItem.GetType())
		}

		// Verify the unique attribute
		if sdpItem.GetUniqueAttribute() != "uniqueAttr" {
			t.Errorf("Expected unique attribute 'uniqueAttr', got: %s", sdpItem.GetUniqueAttribute())
		}

		// Verify the scope
		expectedScope := projectID
		if sdpItem.GetScope() != expectedScope {
			t.Errorf("Expected scope %s, got: %s", expectedScope, sdpItem.GetScope())
		}
	})

	t.Run("Search", func(t *testing.T) {
		wrapper := manual.NewCertificateManagerCertificate(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		adapter := sources.WrapperToAdapter(wrapper, sdpcache.NewNoOpCache())

		mockIterator := mocks.NewMockCertificateIterator(ctrl)

		mockIterator.EXPECT().Next().Return(createCertificate(projectID, location, "cert1"), nil)
		mockIterator.EXPECT().Next().Return(createCertificate(projectID, location, "cert2"), nil)
		mockIterator.EXPECT().Next().Return(nil, iterator.Done)

		mockClient.EXPECT().ListCertificates(ctx, gomock.Any()).Return(mockIterator)

		// Check if adapter supports searching
		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter does not support Search operation")
		}

		sdpItems, err := searchable.Search(ctx, wrapper.Scopes()[0], location, true)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		expectedCount := 2
		actualCount := len(sdpItems)
		if actualCount != expectedCount {
			t.Fatalf("Expected %d items, got: %d", expectedCount, actualCount)
		}

		for _, item := range sdpItems {
			if err := item.Validate(); err != nil {
				t.Fatalf("Expected no validation error, got: %v", err)
			}
		}
	})

	t.Run("GetLookups", func(t *testing.T) {
		wrapper := manual.NewCertificateManagerCertificate(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		lookups := wrapper.GetLookups()
		if len(lookups) != 2 {
			t.Errorf("Expected 2 lookups, got: %d", len(lookups))
		}

		// Verify the lookup types
		expectedTypes := []string{"location", "name"}
		for i, lookup := range lookups {
			if lookup.By != expectedTypes[i] {
				t.Errorf("Expected lookup by %s, got: %s", expectedTypes[i], lookup.By)
			}
		}
	})

	t.Run("SearchLookups", func(t *testing.T) {
		wrapper := manual.NewCertificateManagerCertificate(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		searchLookups := wrapper.SearchLookups()
		if len(searchLookups) != 1 {
			t.Errorf("Expected 1 search lookup, got: %d", len(searchLookups))
		}

		// Verify the search lookup has only location
		if len(searchLookups[0]) != 1 {
			t.Errorf("Expected 1 lookup in search lookup, got: %d", len(searchLookups[0]))
		}
	})

	t.Run("TerraformMappings", func(t *testing.T) {
		wrapper := manual.NewCertificateManagerCertificate(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		mappings := wrapper.TerraformMappings()
		if len(mappings) != 1 {
			t.Errorf("Expected 1 terraform mapping, got: %d", len(mappings))
		}

		mapping := mappings[0]
		if mapping.GetTerraformMethod() != sdp.QueryMethod_SEARCH {
			t.Errorf("Expected SEARCH method, got: %v", mapping.GetTerraformMethod())
		}

		expectedQueryMap := "google_certificate_manager_certificate.id"
		if mapping.GetTerraformQueryMap() != expectedQueryMap {
			t.Errorf("Expected query map %s, got: %s", expectedQueryMap, mapping.GetTerraformQueryMap())
		}
	})

	t.Run("IAMPermissions", func(t *testing.T) {
		wrapper := manual.NewCertificateManagerCertificate(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		permissions := wrapper.IAMPermissions()
		expectedPermissions := []string{
			"certificatemanager.certs.get",
			"certificatemanager.certs.list",
		}

		if len(permissions) != len(expectedPermissions) {
			t.Errorf("Expected %d permissions, got: %d", len(expectedPermissions), len(permissions))
		}

		for i, perm := range permissions {
			if perm != expectedPermissions[i] {
				t.Errorf("Expected permission %s, got: %s", expectedPermissions[i], perm)
			}
		}
	})

	t.Run("PredefinedRole", func(t *testing.T) {
		wrapper := manual.NewCertificateManagerCertificate(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		// PredefinedRole is available on the wrapper, not the adapter
		role := wrapper.(interface{ PredefinedRole() string }).PredefinedRole()
		expectedRole := "roles/certificatemanager.viewer"
		if role != expectedRole {
			t.Errorf("Expected role %s, got: %s", expectedRole, role)
		}
	})

	t.Run("PotentialLinks", func(t *testing.T) {
		wrapper := manual.NewCertificateManagerCertificate(mockClient, []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})

		links := wrapper.PotentialLinks()
		expectedLinks := map[shared.ItemType]bool{
			gcpshared.CertificateManagerDnsAuthorization:          true,
			gcpshared.CertificateManagerCertificateIssuanceConfig: true,
			stdlib.NetworkDNS:                                     true,
		}

		if len(links) != len(expectedLinks) {
			t.Errorf("Expected %d potential links, got: %d", len(expectedLinks), len(links))
		}

		for expectedLink := range expectedLinks {
			if !links[expectedLink] {
				t.Errorf("Expected link to %s", expectedLink)
			}
		}
	})
}
