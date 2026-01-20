package adapters

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/api/artifactregistry/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

func TestArtifactRegistryDockerImage(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()

	imageName := "nginx@sha256:e9954c1fc875017be1c3e36eca16be2d9e9bccc4bf072163515467d6a823c7cf"
	location := "us-central1-a"
	repository := "my-repo"
	dockerImage := &artifactregistry.DockerImage{
		Name:           fmt.Sprintf("projects/test-project/locations/%s/repositories/%s/dockerImages/%s", location, repository, imageName),
		Uri:            fmt.Sprintf("%s-docker.pkg.dev/%s/%s/%s", strings.TrimSuffix(location, "-a"), projectID, repository, imageName),
		Tags:           []string{"latest", "v1.2.3", "stable"},
		MediaType:      "application/vnd.docker.distribution.manifest.v2+json",
		BuildTime:      "2023-06-15T10:30:00Z",
		UpdateTime:     "2023-06-15T10:35:00Z",
		UploadTime:     "2023-06-15T10:32:00Z",
		ImageSizeBytes: 75849324,
	}

	sizeOfFirstPage := 100
	sizeOfLastPage := 1

	dockerImagesWithNextPageToken := &artifactregistry.ListDockerImagesResponse{
		DockerImages:  dynamic.Multiply(dockerImage, sizeOfFirstPage),
		NextPageToken: "next-page-token",
	}

	dockerImages := &artifactregistry.ListDockerImagesResponse{
		DockerImages: dynamic.Multiply(dockerImage, sizeOfLastPage),
	}

	sdpItemType := gcpshared.ArtifactRegistryDockerImage

	expectedCallAndResponses := map[string]shared.MockResponse{
		fmt.Sprintf(
			"https://artifactregistry.googleapis.com/v1/projects/test-project/locations/%s/repositories/%s/dockerImages/%s",
			location,
			repository,
			imageName,
		): {
			StatusCode: http.StatusOK,
			Body:       dockerImage,
		},
		fmt.Sprintf(
			"https://artifactregistry.googleapis.com/v1/projects/test-project/locations/%s/repositories/%s/dockerImages",
			location,
			repository,
		): {
			StatusCode: http.StatusOK,
			Body:       dockerImagesWithNextPageToken,
		},
		fmt.Sprintf(
			"https://artifactregistry.googleapis.com/v1/projects/test-project/locations/%s/repositories/%s/dockerImages?pageToken=next-page-token",
			location,
			repository,
		): {
			StatusCode: http.StatusOK,
			Body:       dockerImages,
		},
	}

	t.Run("Get", func(t *testing.T) {
		// This is a project level adapter, so we pass the project ID
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		getQuery := shared.CompositeLookupKey(location, repository, imageName)
		sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
		if err != nil {
			t.Fatalf("Failed to get docker image: %v", err)
		}

		// Verify the returned item
		if sdpItem.GetType() != sdpItemType.String() {
			t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
		}

		if sdpItem.UniqueAttributeValue() != getQuery {
			t.Errorf("Expected unique attribute value '%s', got %s", imageName, sdpItem.UniqueAttributeValue())
		}

		t.Run("StaticTests", func(t *testing.T) {
			queryTests := shared.QueryTests{
				{
					ExpectedType:   gcpshared.ArtifactRegistryRepository.String(),
					ExpectedMethod: sdp.QueryMethod_GET,
					ExpectedQuery:  shared.CompositeLookupKey(location, repository),
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
		// This is a project level adapter, so we pass the project
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := searchable.Search(ctx, projectID, shared.CompositeLookupKey(location, repository), true)
		if err != nil {
			t.Fatalf("Failed to list docker images: %v", err)
		}

		expectedItemCount := sizeOfFirstPage + sizeOfLastPage
		if len(sdpItems) != expectedItemCount {
			t.Errorf("Expected %d docker images, got %d", expectedItemCount, len(sdpItems))
		}
	})

	t.Run("Search with Terraform format", func(t *testing.T) {
		// This is a project level adapter, so we pass the project ID
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Skipf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
		}

		// Test Terraform format: projects/[project]/locations/[location]/repositories/[repository]/dockerImages/[docker_image]
		terraformQuery := fmt.Sprintf("projects/%s/locations/%s/repositories/%s/dockerImages/%s", projectID, location, repository, imageName)
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

	t.Run("SearchStream", func(t *testing.T) {
		// This is a project level adapter, so we pass the project
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		searchable, ok := adapter.(discovery.SearchableAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement ListableAdapter", sdpItemType)
		}

		sdpItems, err := searchable.Search(ctx, projectID, shared.CompositeLookupKey(location, repository), true)
		if err != nil {
			t.Fatalf("Failed to list docker images: %v", err)
		}

		expectedItemCount := sizeOfFirstPage + sizeOfLastPage
		if len(sdpItems) != expectedItemCount {
			t.Errorf("Expected %d docker images, got %d", expectedItemCount, len(sdpItems))
		}
	})

	t.Run("SearchStream", func(t *testing.T) {
		// This is a project level adapter, so we pass the project
		httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
		adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, sdpcache.NewNoOpCache(), []gcpshared.LocationInfo{gcpshared.NewProjectLocation(projectID)})
		if err != nil {
			t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
		}

		streaming, ok := adapter.(SearchStreamAdapter)
		if !ok {
			t.Fatalf("Adapter for %s does not implement SearchStreamableAdapter", sdpItemType)
		}

		expectedItemCount := sizeOfFirstPage + sizeOfLastPage
		items := make(chan *sdp.Item, expectedItemCount)
		t.Cleanup(func() {
			close(items)
		})

		itemHandler := func(item *sdp.Item) {
			time.Sleep(10 * time.Millisecond)
			items <- item
		}

		errHandler := func(err error) {
			if err != nil {
				t.Fatalf("Unexpected error in stream: %v", err)
			}
		}

		stream := discovery.NewQueryResultStream(itemHandler, errHandler)
		streaming.SearchStream(ctx, projectID, shared.CompositeLookupKey(location, repository), true, stream)

		assert.Eventually(t, func() bool {
			return len(items) == expectedItemCount
		}, 5*time.Second, 100*time.Millisecond, "Expected to receive all items in the stream")
	})
}
