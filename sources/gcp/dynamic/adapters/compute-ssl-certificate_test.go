package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestComputeSSLCertificate(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()
	certificateName := "test-ssl-certificate"

	// Create mock protobuf object
	certificate := &computepb.SslCertificate{
		Name:        stringPtr(certificateName),
		Description: stringPtr("Test SSL Certificate"),
		Certificate: stringPtr("-----BEGIN CERTIFICATE-----\nMIIC...test certificate data...\n-----END CERTIFICATE-----"),
		PrivateKey:  stringPtr("-----BEGIN PRIVATE KEY-----\nMIIE...test private key data...\n-----END PRIVATE KEY-----"),
		SelfLink:    stringPtr(fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/sslCertificates/%s", projectID, certificateName)),
	}

	// Create second certificate for list testing
	certificateName2 := "test-ssl-certificate-2"
	certificate2 := &computepb.SslCertificate{
		Name:        stringPtr(certificateName2),
		Description: stringPtr("Test SSL Certificate 2"),
		Certificate: stringPtr("-----BEGIN CERTIFICATE-----\nMIIC...test certificate data 2...\n-----END CERTIFICATE-----"),
		SelfLink:    stringPtr(fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/global/sslCertificates/%s", projectID, certificateName2)),
	}

	// Create list response with multiple items
	certificateList := &computepb.SslCertificateList{
		Items: []*computepb.SslCertificate{certificate, certificate2},
	}

	sdpItemType := gcpshared.ComputeSSLCertificate

	// Mock HTTP responses
	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/sslCertificates/%s", projectID, certificateName): {
			StatusCode: http.StatusOK,
			Body:       certificate,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/sslCertificates/%s", projectID, certificateName2): {
			StatusCode: http.StatusOK,
			Body:       certificate2,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/sslCertificates", projectID): {
			StatusCode: http.StatusOK,
			Body:       certificateList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, projectID, certificateName, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		// Validate SDP item properties
		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != certificateName {
			t.Errorf("Expected unique attribute value '%s', got %s", certificateName, sdpItem.UniqueAttributeValue())
		}
		if sdpItem.GetScope() != projectID {
			t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
		}

		// Validate specific attributes
		val, err := sdpItem.GetAttributes().Get("name")
		if err != nil {
			t.Fatalf("Failed to get 'name' attribute: %v", err)
		}
		if val != certificateName {
			t.Errorf("Expected name field to be '%s', got %s", certificateName, val)
		}
	})

	t.Run("List", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		listable, ok := adapter.(discovery.ListableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := listable.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list resources: %v", err)
		}

		if len(sdpItems) != 2 {
			t.Errorf("Expected 2 resources, got %d", len(sdpItems))
		}

		// Validate first item
		if len(sdpItems) > 0 {
			firstItem := sdpItems[0]
			if firstItem.GetType() != sdpItemType.String() {
				t.Errorf("Expected first item type %s, got %s", sdpItemType.String(), firstItem.GetType())
			}
			if firstItem.GetScope() != projectID {
				t.Errorf("Expected first item scope '%s', got %s", projectID, firstItem.GetScope())
			}
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test with error responses to simulate API errors
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/global/sslCertificates/%s", projectID, certificateName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "SSL Certificate not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, projectID, certificateName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
