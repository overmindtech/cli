package adapters_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"cloud.google.com/go/logging/apiv2/loggingpb"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestLoggingBucket(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	location := "global"
	linker := gcpshared.NewLinker()
	bucketName := "test-bucket"

	bucket := &loggingpb.LogBucket{
		Name: fmt.Sprintf("projects/%s/locations/%s/buckets/%s", projectID, location, bucketName),
		CmekSettings: &loggingpb.CmekSettings{
			KmsKeyName:        "projects/test-project/locations/global/keyRings/my-keyring/cryptoKeys/my-key",
			KmsKeyVersionName: "projects/test-project/locations/global/keyRings/my-keyring/cryptoKeys/my-key/cryptoKeyVersions/1",
			ServiceAccountId:  "cmek-p123456789@gcp-sa-logging.iam.gserviceaccount.com",
		},
	}

	bucketList := &loggingpb.ListBucketsResponse{
		Buckets: []*loggingpb.LogBucket{bucket},
	}

	sdpItemType := gcpshared.LoggingBucket

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets/%s", projectID, location, bucketName): {
			StatusCode: http.StatusOK,
			Body:       bucket,
		},
		fmt.Sprintf("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets", projectID, location): {
			StatusCode: http.StatusOK,
			Body:       bucketList,
		},
	}

	t.Run("Get", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(location, bucketName)
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get logging bucket: %v", err)
		}

		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					// cmekSettings.kmsKeyName
					ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "my-keyring", "my-key"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// cmekSettings.kmsKeyVersionName
					ExpectedType:   gcpshared.CloudKMSCryptoKeyVersion.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey("global", "my-keyring", "my-key", "1"),
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
				{
					// cmekSettings.serviceAccountId
					ExpectedType:   gcpshared.IAMServiceAccount.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  "cmek-p123456789@gcp-sa-logging.iam.gserviceaccount.com",
					ExpectedScope:  projectID,
					ExpectedBlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}

			shared.RunStaticTests(t, adapter, sdpItem, queryTests)
		})
	})

	t.Run("Search", func(t *testing.T) {
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		sdpItems, err := searchable.Search(ctx, projectID, location, true)
		if err != nil {
			t.Fatalf("Failed to search logging buckets: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 logging bucket, got %d", len(sdpItems))
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		errorResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://logging.googleapis.com/v2/projects/%s/locations/%s/buckets/%s", projectID, location, bucketName): {
				StatusCode: http.StatusNotFound,
				Body:       map[string]interface{}{"error": "Bucket not found"},
			},
		}

		httpCli := shared.NewMockHTTPClientProvider(errorResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(location, bucketName)
		_, err = adapter.Get(ctx, projectID, getQuery, true)
		if err == nil {
			t.Error("Expected error when getting non-existent logging bucket, but got nil")
		}
	})
}
