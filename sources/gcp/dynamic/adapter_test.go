package dynamic_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	instancepb "cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/artifactregistry/v1"
	"google.golang.org/api/compute/v1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

type SearchStreamAdapter interface {
	SearchStream(ctx context.Context, scope string, query string, ignoreCache bool, stream discovery.QueryResultStream)
}

type ListStreamAdapter interface {
	ListStream(ctx context.Context, scope string, ignoreCache bool, stream discovery.QueryResultStream)
}

// TODO: Possible improvements:
// - Create a helper function that does some of the common assertions for the adapter tests
func TestAdapter(t *testing.T) {
	ctx := context.Background()
	projectID := "test-project"
	linker := gcpshared.NewLinker()

	t.Run("ComputeInstanceTemplate", func(t *testing.T) {
		// Create a template object
		template := &compute.InstanceTemplate{
			Id:          123456789,
			Name:        "test-instance-template",
			Description: "Test instance template",
			Properties: &compute.InstanceProperties{
				MachineType: "e2-medium",
				Disks: []*compute.AttachedDisk{
					{
						Boot:       true,
						DeviceName: "boot-disk",
						InitializeParams: &compute.AttachedDiskInitializeParams{
							DiskName:         "projects/test-project/zones/us-central1-a/disks/disk-name",
							DiskType:         "projects/test-project/zones/us-central1-a/diskTypes/pd-standard",
							SourceImage:      "projects/debian-cloud/global/images/family/debian-11",
							SourceSnapshot:   "projects/test-project/global/snapshots/my-snapshot",
							ResourcePolicies: []string{"projects/test-project/regions/us-central1/resourcePolicies/my-resource-policy"},
							StoragePool:      "projects/test-project/zones/us-central1-a/storagePools/my-storage-pool",
							Licenses:         []string{"https://www.googleapis.com/compute/v1/projects/test-project/global/licenses/debian-11-bullseye-init-param"},
							SourceImageEncryptionKey: &compute.CustomerEncryptionKey{
								KmsKeyName:           "projects/test-project/locations/global/keyRings/my-keyring/cryptoKeys/source-image-encryption-key",
								KmsKeyServiceAccount: "source-image-encryption-key-service-account@test-project.iam.gserviceaccount.com",
							},
							SourceSnapshotEncryptionKey: &compute.CustomerEncryptionKey{
								KmsKeyName:           "projects/test-project/locations/global/keyRings/my-keyring/cryptoKeys/source-snapshot-encryption-key",
								KmsKeyServiceAccount: "source-snapshot-encryption-key-service-account@test-project.iam.gserviceaccount.com",
							},
						},

						Source:   "projects/test-project/zones/us-central1-a/disks/source",
						Licenses: []string{"https://www.googleapis.com/compute/v1/projects/test-project/global/licenses/debian-11-bullseye-disk"},
						DiskEncryptionKey: &compute.CustomerEncryptionKey{
							KmsKeyName:           "projects/test-project/locations/global/keyRings/my-keyring/cryptoKeys/disk-encryption-key",
							KmsKeyServiceAccount: "disk-encryption-key-service-account@test-project.iam.gserviceaccount.com",
						},
					},
				},
				NetworkInterfaces: []*compute.NetworkInterface{
					{
						Network:     "global/networks/default",
						Subnetwork:  "regions/us-central1/subnetworks/default",
						NetworkIP:   "10.240.17.92",
						Ipv6Address: "2600:1901:0:1234::1",
						AccessConfigs: []*compute.AccessConfig{
							{
								NatIP:          "10.240.17.93",
								ExternalIpv6:   "2600:1901:0:1234::2",
								SecurityPolicy: "projects/test-project/global/securityPolicies/test-security-policy",
							},
						},
						Ipv6AccessConfigs: []*compute.AccessConfig{
							{
								NatIP:          "10.240.17.94",
								ExternalIpv6:   "2600:1901:0:1234::3",
								SecurityPolicy: "projects/test-project/global/securityPolicies/test-security-policy-ipv6",
							},
						},
					},
				},
				GuestAccelerators: []*compute.AcceleratorConfig{
					{
						AcceleratorType:  "projects/test-project/zones/us-central1-a/acceleratorTypes/nvidia-tesla-t4",
						AcceleratorCount: 1,
					},
				},
				Scheduling: &compute.Scheduling{
					NodeAffinities: []*compute.SchedulingNodeAffinity{
						{
							Key:      "compute.googleapis.com/node-group-name",
							Operator: "IN",
							Values:   []string{"projects/test-project/zones/us-central1-a/nodeGroups/my-node-group"}},
					},
				},
				ReservationAffinity: &compute.ReservationAffinity{
					ConsumeReservationType: "SPECIFIC_RESERVATION",
					Key:                    "compute.googleapis.com/reservation-name",
					Values:                 []string{"my-reservation"},
				},
			},
			SelfLink: "https://compute.googleapis.com/compute/v1/projects/test-project/global/instanceTemplates/test-instance-template",
		}

		sizeOfFirstPage := 100
		sizeOfLastPage := 1

		templatesWithNextPage := &compute.InstanceTemplateList{
			Items:         dynamic.Multiply(template, sizeOfFirstPage),
			NextPageToken: "next-page-token",
		}

		templates := &compute.InstanceTemplateList{
			Items: dynamic.Multiply(template, sizeOfLastPage),
		}

		expectedCallAndResponses := map[string]shared.MockResponse{
			"https://compute.googleapis.com/compute/v1/projects/test-project/global/instanceTemplates/test-instance-template": {
				StatusCode: http.StatusOK,
				Body:       template,
			},
			"https://compute.googleapis.com/compute/v1/projects/test-project/global/instanceTemplates": {
				StatusCode: http.StatusOK,
				Body:       templatesWithNextPage,
			},
			"https://compute.googleapis.com/compute/v1/projects/test-project/global/instanceTemplates?pageToken=next-page-token": {
				StatusCode: http.StatusOK,
				Body:       templates,
			},
		}

		meta := gcpshared.SDPAssetTypeToAdapterMeta[gcpshared.ComputeInstanceTemplate]

		t.Run("Get", func(t *testing.T) {
			adapter, err := dynamic.MakeAdapter(gcpshared.ComputeInstanceTemplate, meta, linker, shared.NewMockHTTPClientProvider(expectedCallAndResponses), projectID)
			if err != nil {
				t.Fatalf("Failed to create adapter for ComputeInstanceTemplate: %v", err)
			}

			sdpItem, err := adapter.Get(ctx, projectID, "test-instance-template", true)
			if err != nil {
				t.Fatalf("Failed to get instance template: %v", err)
			}

			// Verify the returned item
			if sdpItem.GetType() != gcpshared.ComputeInstanceTemplate.String() {
				t.Errorf("Expected type %s, got %s", gcpshared.ComputeInstanceTemplate.String(), sdpItem.GetType())
			}

			if sdpItem.UniqueAttributeValue() != "test-instance-template" {
				t.Errorf("Expected unique attribute value 'test-instance-template', got %s", sdpItem.UniqueAttributeValue())
			}

			t.Run("StaticTests", func(t *testing.T) {
				queryTests := shared.QueryTests{
					{
						ExpectedType:   gcpshared.ComputeMachineType.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "e2-medium",
						ExpectedScope:  projectID,
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						ExpectedType:   gcpshared.ComputeNetwork.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "default",
						ExpectedScope:  projectID,
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						// properties.disks.initializeParams.diskName
						ExpectedType:   gcpshared.ComputeDisk.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "disk-name",
						ExpectedScope:  "test-project.us-central1-a",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					},
					{
						// properties.disks.source
						ExpectedType:   gcpshared.ComputeDisk.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "source",
						ExpectedScope:  "test-project.us-central1-a",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					},
					{
						ExpectedType:   gcpshared.ComputeImage.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "debian-11",
						ExpectedScope:  projectID,
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						ExpectedType:   gcpshared.ComputeSubnetwork.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "default",
						ExpectedScope:  "test-project.us-central1",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						// properties.networkInterfaces.networkIP
						ExpectedType:   stdlib.NetworkIP.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "10.240.17.92",
						ExpectedScope:  "global",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					},
					{
						// properties.networkInterfaces.ipv6Address
						ExpectedType:   stdlib.NetworkIP.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "2600:1901:0:1234::1",
						ExpectedScope:  "global",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					},
					{
						// properties.networkInterfaces.accessConfigs.natIP
						ExpectedType:   stdlib.NetworkIP.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "10.240.17.93",
						ExpectedScope:  "global",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					},
					{
						// properties.networkInterfaces.accessConfigs.externalIpv6
						ExpectedType:   stdlib.NetworkIP.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "2600:1901:0:1234::2",
						ExpectedScope:  "global",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					},
					{
						// properties.networkInterfaces.accessConfigs.securityPolicy
						ExpectedType:   gcpshared.ComputeSecurityPolicy.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-security-policy",
						ExpectedScope:  projectID,
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						// properties.networkInterfaces.ipv6AccessConfigs.natIP
						ExpectedType:   stdlib.NetworkIP.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "10.240.17.94",
						ExpectedScope:  "global",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					},
					{
						// properties.networkInterfaces.ipv6AccessConfigs.externalIpv6
						ExpectedType:   stdlib.NetworkIP.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "2600:1901:0:1234::3",
						ExpectedScope:  "global",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					},
					{
						// properties.networkInterfaces.ipv6AccessConfigs.securityPolicy
						ExpectedType:   gcpshared.ComputeSecurityPolicy.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "test-security-policy-ipv6",
						ExpectedScope:  projectID,
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						// properties.disks.initializeParams.sourceSnapshot
						ExpectedType:   gcpshared.ComputeSnapshot.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "my-snapshot",
						ExpectedScope:  projectID,
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						// properties.disks.initializeParams.resourcePolicies
						ExpectedType:   gcpshared.ComputeResourcePolicy.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "my-resource-policy",
						ExpectedScope:  "test-project.us-central1",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						// properties.disks.initializeParams.storagePool
						ExpectedType:   gcpshared.ComputeStoragePool.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "my-storage-pool",
						ExpectedScope:  "test-project.us-central1-a",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						// properties.disks.initializeParams.licenses
						ExpectedType:   gcpshared.ComputeLicense.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "debian-11-bullseye-init-param",
						ExpectedScope:  projectID,
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						// properties.disks.initializeParams.licenses
						ExpectedType:   gcpshared.ComputeLicense.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "debian-11-bullseye-disk",
						ExpectedScope:  projectID,
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						// properties.disks.initializeParams.sourceImageEncryptionKey.kmsKeyName
						ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "global|my-keyring|source-image-encryption-key",
						ExpectedScope:  projectID,
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						// properties.disks.initializeParams.sourceImageEncryptionKey.kmsKeyServiceAccount
						ExpectedType:   gcpshared.IAMServiceAccount.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "source-image-encryption-key-service-account@test-project.iam.gserviceaccount.com",
						ExpectedScope:  projectID,
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						// properties.guestAccelerators.acceleratorType
						ExpectedType:   gcpshared.ComputeAcceleratorType.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "nvidia-tesla-t4",
						ExpectedScope:  "test-project.us-central1-a",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						// properties.scheduling.nodeAffinities.values
						ExpectedType:   gcpshared.ComputeNodeGroup.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "my-node-group",
						ExpectedScope:  "test-project.us-central1-a",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: true,
						},
					},
					{
						// properties.reservationAffinity.values
						ExpectedType:   gcpshared.ComputeReservation.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "my-reservation",
						ExpectedScope:  projectID,
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						// properties.disks.initializeParams.diskType
						ExpectedType:   gcpshared.ComputeDiskType.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "pd-standard",
						ExpectedScope:  "test-project.us-central1-a",
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						// properties.disks.initializeParams.sourceSnapshotEncryptionKey.kmsKeyName
						ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "global|my-keyring|source-snapshot-encryption-key",
						ExpectedScope:  projectID,
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						// properties.disks.initializeParams.sourceSnapshotEncryptionKey.kmsKeyServiceAccount
						ExpectedType:   gcpshared.IAMServiceAccount.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "source-snapshot-encryption-key-service-account@test-project.iam.gserviceaccount.com",
						ExpectedScope:  projectID,
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						// properties.disks.diskEncryptionKey.kmsKeyName
						ExpectedType:   gcpshared.CloudKMSCryptoKey.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "global|my-keyring|disk-encryption-key",
						ExpectedScope:  projectID,
						ExpectedBlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
					{
						// properties.disks.diskEncryptionKey.kmsKeyServiceAccount
						ExpectedType:   gcpshared.IAMServiceAccount.String(),
						ExpectedMethod: sdp.QueryMethod_GET,
						ExpectedQuery:  "disk-encryption-key-service-account@test-project.iam.gserviceaccount.com",
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

		t.Run("List", func(t *testing.T) {
			adapter, err := dynamic.MakeAdapter(gcpshared.ComputeInstanceTemplate, meta, linker, shared.NewMockHTTPClientProvider(expectedCallAndResponses), projectID)
			if err != nil {
				t.Fatalf("Failed to create adapter for ComputeInstanceTemplate: %v", err)
			}

			listable, ok := adapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter is not a ListableAdapter")
			}

			sdpItems, err := listable.List(ctx, projectID, true)
			if err != nil {
				t.Fatalf("Failed to list instance templatesWithNextPage: %v", err)
			}

			expectedItemCount := sizeOfFirstPage + sizeOfLastPage
			if len(sdpItems) != expectedItemCount {
				t.Errorf("Expected %d instance template, got %d", expectedItemCount, len(sdpItems))
			}
		})

		t.Run("ListStream", func(t *testing.T) {
			adapter, err := dynamic.MakeAdapter(gcpshared.ComputeInstanceTemplate, meta, linker, shared.NewMockHTTPClientProvider(expectedCallAndResponses), projectID)
			if err != nil {
				t.Fatalf("Failed to create adapter for ComputeInstanceTemplate: %v", err)
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

			listStreamable, ok := adapter.(ListStreamAdapter)
			if !ok {
				t.Fatalf("Adapter is not a ListStreamAdapter")
			}

			stream := discovery.NewQueryResultStream(itemHandler, errHandler)
			listStreamable.ListStream(ctx, projectID, true, stream)

			assert.Eventually(t, func() bool {
				return len(items) == expectedItemCount
			}, 5*time.Second, 100*time.Millisecond, "Expected to receive all items in the stream")
		})
	})

	t.Run("ArtifactRegistryDockerImage", func(t *testing.T) {
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
		meta := gcpshared.SDPAssetTypeToAdapterMeta[sdpItemType]

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
			adapter, err := dynamic.MakeAdapter(sdpItemType, meta, linker, httpCli, projectID)
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

		t.Run("SearchWithTerraformMapping", func(t *testing.T) {
			// This is a project level adapter, so we pass the project ID
			httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
			adapter, err := dynamic.MakeAdapter(sdpItemType, meta, linker, httpCli, projectID)
			if err != nil {
				t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
			}

			searchable, ok := adapter.(discovery.SearchableAdapter)
			if !ok {
				t.Fatalf("Adapter for %s does not implement ListableAdapter", sdpItemType)
			}

			// projects/{{project}}/locations/{{location}}/repository/{{repository_id}}/dockerImages/{{docker_image}}
			terraformQuery := fmt.Sprintf("projects/%s/locations/%s/repositories/%s/dockerImages/%s", projectID, location, repository, imageName)
			sdpItems, err := searchable.Search(ctx, projectID, terraformQuery, true)
			if err != nil {
				t.Fatalf("Failed to get docker image with terraform query: %v", err)
			}

			if len(sdpItems) != 1 {
				t.Errorf("Unexpected number of docker images: %d", len(sdpItems))
			}

			// Verify the returned item
			if err := sdpItems[0].Validate(); err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})

		t.Run("Search", func(t *testing.T) {
			// This is a project level adapter, so we pass the project
			httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
			adapter, err := dynamic.MakeAdapter(sdpItemType, meta, linker, httpCli, projectID)
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
			adapter, err := dynamic.MakeAdapter(sdpItemType, meta, linker, httpCli, projectID)
			if err != nil {
				t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
			}

			streaming, ok := adapter.(SearchStreamAdapter)
			if !ok {
				t.Fatalf("Adapter for %s does not implement StreamingAdapter", sdpItemType)
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
	})

	t.Run("SpannerInstance", func(t *testing.T) {
		instanceName := "test-instance"
		spannerInstance := &instancepb.Instance{
			Name:        fmt.Sprintf("projects/%s/instances/%s", projectID, instanceName),
			DisplayName: "Test Spanner Instance",
			Config:      "projects/test-project/instanceConfigs/regional-us-central1",
			NodeCount:   3,
			State:       instancepb.Instance_READY,
			Labels: map[string]string{
				"env":  "test",
				"team": "devops",
			},
			ProcessingUnits: 1000,
		}

		spannerInstances := &instancepb.ListInstancesResponse{
			Instances: []*instancepb.Instance{spannerInstance},
		}

		sdpItemType := gcpshared.SpannerInstance
		meta := gcpshared.SDPAssetTypeToAdapterMeta[sdpItemType]

		expectedCallAndResponses := map[string]shared.MockResponse{
			fmt.Sprintf("https://spanner.googleapis.com/v1/projects/%s/instances/%s", projectID, instanceName): {
				StatusCode: http.StatusOK,
				Body:       spannerInstance,
			},
			fmt.Sprintf("https://spanner.googleapis.com/v1/projects/%s/instances", projectID): {
				StatusCode: http.StatusOK,
				Body:       spannerInstances,
			},
		}

		t.Run("Get", func(t *testing.T) {
			httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
			adapter, err := dynamic.MakeAdapter(sdpItemType, meta, linker, httpCli, projectID)
			if err != nil {
				t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
			}

			getQuery := instanceName
			sdpItem, err := adapter.Get(ctx, projectID, getQuery, true)
			if err != nil {
				t.Fatalf("Failed to get Spanner instance: %v", err)
			}

			if sdpItem.GetType() != sdpItemType.String() {
				t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
			}
			if sdpItem.UniqueAttributeValue() != getQuery {
				t.Errorf("Expected unique attribute value '%s', got %s", instanceName, sdpItem.UniqueAttributeValue())
			}
			if sdpItem.GetScope() != projectID {
				t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
			}
			val, err := sdpItem.GetAttributes().Get("name")
			if err != nil {
				t.Fatalf("Failed to get 'name' attribute: %v", err)
			}
			if val != fmt.Sprintf("projects/%s/instances/%s", projectID, instanceName) {
				t.Errorf("Expected name field to be 'projects/%s/instances/%s', got %s", projectID, instanceName, val)
			}
			val, err = sdpItem.GetAttributes().Get("display_name")
			if err != nil {
				t.Fatalf("Failed to get 'display_name' attribute: %v", err)
			}
			if val != "Test Spanner Instance" {
				t.Errorf("Expected display_name field to be 'Test Spanner Instance', got %s", val)
			}
			val, err = sdpItem.GetAttributes().Get("config")
			if err != nil {
				t.Fatalf("Failed to get 'config' attribute: %v", err)
			}
			if val != "projects/test-project/instanceConfigs/regional-us-central1" {
				t.Errorf("Expected config field to be 'projects/test-project/instanceConfigs/regional-us-central1', got %s", val)
			}
			val, err = sdpItem.GetAttributes().Get("node_count")
			if err != nil {
				t.Fatalf("Failed to get 'node_count' attribute: %v", err)
			}
			converted, ok := val.(float64)
			if !ok {
				t.Fatalf("Expected node_count to be a float64, got %T", val)
			}
			if converted != 3 {
				t.Errorf("Expected node_count field to be '3', got %s", val)
			}
			val, err = sdpItem.GetAttributes().Get("state")
			if err != nil {
				t.Fatalf("Failed to get 'state' attribute: %v", err)
			}
			converted, ok = val.(float64)
			if !ok {
				t.Fatalf("Expected state to be a float64, got %T", val)
			}
			if instancepb.Instance_State(converted) != instancepb.Instance_READY {
				t.Errorf("Expected state field to be 'READY', got %s", val)
			}
		})

		t.Run("List", func(t *testing.T) {
			httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
			adapter, err := dynamic.MakeAdapter(gcpshared.SpannerInstance, meta, linker, httpCli, projectID)
			if err != nil {
				t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
			}

			listable, ok := adapter.(discovery.ListableAdapter)
			if !ok {
				t.Fatalf("Adapter is not a ListableAdapter")
			}

			sdpItems, err := listable.List(ctx, projectID, true)
			if err != nil {
				t.Fatalf("Failed to list Spanner instances: %v", err)
			}

			if len(sdpItems) != 1 {
				t.Errorf("Expected 1 Spanner instance, got %d", len(sdpItems))
			}
		})
	})
}
