package dynamic_test

import (
	"context"
	"net/http"
	"testing"

	"google.golang.org/api/compute/v1"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

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

		templates := &compute.InstanceTemplateList{
			Items: []*compute.InstanceTemplate{template},
		}

		mockTransport := shared.NewMockRoundTripper(map[string]*http.Response{
			"https://compute.googleapis.com/compute/v1/projects/test-project/global/instanceTemplates/test-instance-template": {
				StatusCode: http.StatusOK,
				Body:       shared.MockHTTPResponse(template),
				Header:     make(http.Header),
			},
			"https://compute.googleapis.com/compute/v1/projects/test-project/global/instanceTemplates": {
				StatusCode: http.StatusOK,
				Body:       shared.MockHTTPResponse(templates),
				Header:     make(http.Header),
			},
		})

		mockHTTPClient := &http.Client{
			Transport: mockTransport,
		}

		computeInstanceTempMeta := gcpshared.SDPAssetTypeToAdapterMeta[gcpshared.ComputeInstanceTemplate]

		getEndpointBaseURLFunc, err := computeInstanceTempMeta.GetEndpointBaseURLFunc(projectID)
		if err != nil {
			t.Fatalf("Failed to get endpoint base URL: %v", err)
		}

		listEndpoint, err := computeInstanceTempMeta.ListEndpointFunc(projectID)
		if err != nil {
			t.Fatalf("Failed to get list endpoint: %v", err)
		}

		cfg := &dynamic.AdapterConfig{
			ProjectID:           projectID,
			Scope:               projectID,
			GetURLFunc:          getEndpointBaseURLFunc,
			SDPAssetType:        gcpshared.ComputeInstanceTemplate,
			SDPAdapterCategory:  computeInstanceTempMeta.SDPAdapterCategory,
			TerraformMappings:   dynamic.SDPAssetTypeToTerraformMappings[gcpshared.ComputeInstanceTemplate].Mappings,
			Linker:              linker,
			HTTPClient:          mockHTTPClient,
			UniqueAttributeKeys: computeInstanceTempMeta.UniqueAttributeKeys,
		}

		adapter, err := dynamic.NewListableAdapter(listEndpoint, cfg)
		if err != nil {
			t.Fatalf("Failed to create adapter: %v", err)
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

		sdpItems, err := adapter.List(ctx, projectID, true)
		if err != nil {
			t.Fatalf("Failed to list instance templates: %v", err)
		}

		if len(sdpItems) != 1 {
			t.Errorf("Expected 1 instance template, got %d", len(sdpItems))
		}

		if len(mockTransport.GetRequestURLs()) != len(mockTransport.GetResponses()) {
			var missingURLs []string
			for url := range mockTransport.GetResponses() {
				if _, found := mockTransport.GetRequestURLs()[url]; !found {
					missingURLs = append(missingURLs, url)
				}
			}

			t.Errorf("Expected %d HTTP request, but got %d, missing calls: %v", len(mockTransport.GetRequestURLs()), len(mockTransport.GetResponses()), missingURLs)
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
}
