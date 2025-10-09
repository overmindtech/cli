package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/compute/apiv1/computepb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestComputePublicDelegatedPrefix(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	region := "us-central1"
	linker := gcpshared.NewLinker()
	prefixName := "test-prefix"

	parentPrefixURL := fmt.Sprintf("projects/%s/global/publicAdvertisedPrefixes/test-parent-prefix", projectID)
	subPrefixName1 := fmt.Sprintf("projects/%s/regions/%s/publicDelegatedPrefixes/test-sub-prefix-1", projectID, region)
	subPrefixName2 := fmt.Sprintf("projects/%s/regions/%s/publicDelegatedPrefixes/test-sub-prefix-2", projectID, region)
	delegateeProject1 := "projects/delegatee-project-1"
	delegateeProject2 := "projects/delegatee-project-2"

	prefix := &computepb.PublicDelegatedPrefix{
		Name:         &prefixName,
		ParentPrefix: &parentPrefixURL,
		PublicDelegatedSubPrefixs: []*computepb.PublicDelegatedPrefixPublicDelegatedSubPrefix{
			{
				Name:             &subPrefixName1,
				DelegateeProject: &delegateeProject1,
			},
			{
				Name:             &subPrefixName2,
				DelegateeProject: &delegateeProject2,
			},
		},
	}

	prefixName2 := "test-prefix-2"
	prefix2 := &computepb.PublicDelegatedPrefix{
		Name: &prefixName2,
	}

	prefixList := &computepb.PublicDelegatedPrefixList{
		Items: []*computepb.PublicDelegatedPrefix{prefix, prefix2},
	}

	sdpItemType := gcpshared.ComputePublicDelegatedPrefix

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/publicDelegatedPrefixes/%s", projectID, region, prefixName): {
			StatusCode: http.StatusOK,
			Body:       prefix,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/publicDelegatedPrefixes/%s", projectID, region, prefixName2): {
			StatusCode: http.StatusOK,
			Body:       prefix2,
		},
		fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/publicDelegatedPrefixes", projectID, region): {
			StatusCode: http.StatusOK,
			Body:       prefixList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		sdpItem, err := adapter.Get(ctx, fmt.Sprintf("%s.%s", projectID, region), prefixName, true)
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}
		if sdpItem.UniqueAttributeValue() != prefixName {
			t.Errorf("Expected unique attribute value '%s', got %s", prefixName, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				// Note: Parent prefix link test omitted because gcp-compute-public-advertised-prefix adapter doesn't exist yet
				// Delegatee project 1 link
				{
					ExpectedType:   gcpshared.CloudResourceManagerProject.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "delegatee-project-1",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Delegatee project 2 link
				{
					ExpectedType:   gcpshared.CloudResourceManagerProject.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "delegatee-project-2",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Sub-prefix 1 link
				{
					ExpectedType:   gcpshared.ComputePublicDelegatedPrefix.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-sub-prefix-1",
					ExpectedScope:  fmt.Sprintf("%s.%s", projectID, region),
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				// Sub-prefix 2 link
				{
					ExpectedType:   gcpshared.ComputePublicDelegatedPrefix.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "test-sub-prefix-2",
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
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, region)
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

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://compute.googleapis.com/compute/v1/projects/%s/regions/%s/publicDelegatedPrefixes/%s", projectID, region, prefixName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Prefix not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID, region)
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		_, err = adapter.Get(ctx, fmt.Sprintf("%s.%s", projectID, region), prefixName, true)
		if err == nil {
			t.Error("Expected error when getting non-existent resource, but got nil")
		}
	})
}
