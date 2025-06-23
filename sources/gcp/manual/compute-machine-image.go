package manual

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	ComputeMachineImage = shared.NewItemType(gcpshared.GCP, gcpshared.Compute, gcpshared.MachineImage)

	ComputeMachineImageLookupByName = shared.NewItemTypeLookup("name", ComputeMachineImage)
)

type computeMachineImageWrapper struct {
	client gcpshared.ComputeMachineImageClient
	*gcpshared.ProjectBase
}

// NewComputeMachineImage creates a new computeMachineImageWrapper instance
func NewComputeMachineImage(client gcpshared.ComputeMachineImageClient, projectID string) sources.ListableWrapper {
	return &computeMachineImageWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			ComputeMachineImage,
		),
	}
}

func (c computeMachineImageWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		ComputeNetwork,
		ComputeSubnetwork,
		ComputeDisk,
		gcpshared.CloudKMSCryptoKeyVersion,
		ComputeInstance,
	)
}

// TerraformMappings returns the Terraform mappings for the compute machine image wrapper
func (c computeMachineImageWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_machine_image.name",
		},
	}
}

// GetLookups returns the lookups for the compute machine image wrapper
func (c computeMachineImageWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeMachineImageLookupByName,
	}
}

// Get retrieves a compute machine image by its name
func (c computeMachineImageWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetMachineImageRequest{
		Project:      c.ProjectID(),
		MachineImage: queryParts[0],
	}

	machineImage, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	item, sdpErr := c.gcpComputeMachineImageToSDPItem(machineImage)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// List lists compute machine images and converts them to sdp.Items.
func (c computeMachineImageWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	it := c.client.List(ctx, &computepb.ListMachineImagesRequest{
		Project: c.ProjectID(),
	})

	var items []*sdp.Item
	for {
		machineImage, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err)
		}

		item, sdpErr := c.gcpComputeMachineImageToSDPItem(machineImage)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeMachineImageWrapper) gcpComputeMachineImageToSDPItem(machineImage *computepb.MachineImage) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(machineImage, "labels")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            ComputeMachineImage.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),
		Tags:            machineImage.GetLabels(),
	}

	// The source instance used to create the machine image. You can provide this as a partial or full URL to the resource.
	// For example, the following are valid values:
	// - https://www.googleapis.com/compute/v1/projects/project/zones/zone /instances/instance
	// - projects/project/zones/zone/instances/instance
	if instanceProperties := machineImage.GetInstanceProperties(); instanceProperties != nil {

		for _, networkInterface := range instanceProperties.GetNetworkInterfaces() {
			// Network Interfaces
			// GET https://compute.googleapis.com/compute/v1/projects/{project}/global/networks/{network}
			// https://cloud.google.com/compute/docs/reference/rest/v1/networks/get
			if network := networkInterface.GetNetwork(); network != "" {
				networkName := gcpshared.LastPathComponent(network)
				if networkName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   ComputeNetwork.String(),
							Method: sdp.QueryMethod_GET,
							Query:  networkName,
							Scope:  c.ProjectID(),
						},
						// If the network is no longer valid errors may occur.
						// User will need to override the network interface config when instantiating a VM from the image.
						BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
					})
				}
			}

			// Network Interfaces
			// GET https://compute.googleapis.com/compute/v1/projects/{project}/regions/{region}/subnetworks/{subnetwork}
			// https://cloud.google.com/compute/docs/reference/rest/v1/subnetworks/get
			if subnet := networkInterface.GetSubnetwork(); subnet != "" {
				subnetworkName := gcpshared.LastPathComponent(subnet)
				if subnetworkName != "" {
					region := gcpshared.ExtractPathParam("regions", subnet)
					if region != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   ComputeSubnetwork.String(),
								Method: sdp.QueryMethod_GET,
								Query:  subnetworkName,
								Scope:  gcpshared.RegionalScope(c.ProjectID(), region),
							},
							// If the network is no longer valid errors may occur.
							// User will need to select valid VPC/subnet during creation.
							BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
						})
					}
				}
			}
		}

		//An array of disks that are associated with the instances that are created from this machine image.

		for _, disk := range instanceProperties.GetDisks() {
			diskSource := disk.GetSource()
			if diskSource != "" {
				// Specifies a valid partial or full URL to an existing Persistent Disk resource.
				// "source": "https://www.googleapis.com/compute/v1/projects/project-test/zones/us-central1-c/disks/integration-test-instance"
				// last part is the disk name
				if strings.Contains(diskSource, "/") {
					diskName := gcpshared.LastPathComponent(diskSource)
					if diskName != "" {
						zone := gcpshared.ExtractPathParam("zones", diskSource)
						if zone != "" {
							sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
								Query: &sdp.Query{
									Type:   ComputeDisk.String(),
									Method: sdp.QueryMethod_GET,
									Query:  diskName,
									Scope:  gcpshared.ZonalScope(c.ProjectID(), zone),
								},
								BlastPropagation: &sdp.BlastPropagation{
									In:  true,
									Out: false,
								},
							})
						}

						// The encryption key for the disk; appears in the following format:
						// "sourceDiskEncryptionKey.kmsKeyName": "projects/ kms_project_id/locations/ region/keyRings/ key_region/cryptoKeys/key /cryptoKeyVersions/1
						// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/keyRings/*/cryptoKeys/*/cryptoKeyVersions/*}
						// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions
						// sourceDiskEncryptionKey.kmsKeyName -> CloudKMSCryptoKeyVersion
						if sourceDiskEncryptionKey := disk.GetDiskEncryptionKey(); sourceDiskEncryptionKey != nil {
							if keyName := sourceDiskEncryptionKey.GetKmsKeyName(); keyName != "" {
								// Parsing them all together to improve readability
								location := gcpshared.ExtractPathParam("locations", keyName)
								keyRing := gcpshared.ExtractPathParam("keyRings", keyName)
								cryptoKey := gcpshared.ExtractPathParam("cryptoKeys", keyName)
								cryptoKeyVersion := gcpshared.ExtractPathParam("cryptoKeyVersions", keyName)

								// Validate all parts before proceeding, a bit less performatic if any is missing but readability is improved
								if location != "" && keyRing != "" && cryptoKey != "" && cryptoKeyVersion != "" {
									sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
										Query: &sdp.Query{
											Type:   gcpshared.CloudKMSCryptoKeyVersion.String(),
											Method: sdp.QueryMethod_GET,
											Query:  shared.CompositeLookupKey(location, keyRing, cryptoKey, cryptoKeyVersion),
											Scope:  c.ProjectID(),
										},
										//Deleting a key might impact the users ability to restore the VM from the machine image
										//because the encrypted disk data cannot be decrypted.
										//Deleting a machineImage in GCP does not affect its associated sourceDiskEncryptionKey
										BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
									})
								}
							}
						}
					}
				}
			}
		}

	}

	// Encrypts the machine image using a customer-supplied encryption key.
	// After you encrypt a machine image using a customer-supplied key, you must provide the same key if you use the machine image later.
	// Appears in the following format:
	// "machineImageEncryptionKey.kmsKeyName": "projects/ kms_project_id/locations/ region/keyRings/ key_region/cryptoKeys/key /cryptoKeyVersions/1
	// GET https://cloudkms.googleapis.com/v1/{name=projects/*/locations/*/keyRings/*/cryptoKeys/*/cryptoKeyVersions/*}
	// https://cloud.google.com/kms/docs/reference/rest/v1/projects.locations.keyRings.cryptoKeys.cryptoKeyVersions
	// machineImageEncryptionKey.kmsKeyName -> CloudKMSCryptoKeyVersion
	if machineImageEncryptionKey := machineImage.GetMachineImageEncryptionKey(); machineImageEncryptionKey != nil {
		if keyName := machineImageEncryptionKey.GetKmsKeyName(); keyName != "" {
			// Parsing them all together to improve readability
			location := gcpshared.ExtractPathParam("locations", keyName)
			keyRing := gcpshared.ExtractPathParam("keyRings", keyName)
			cryptoKey := gcpshared.ExtractPathParam("cryptoKeys", keyName)
			cryptoKeyVersion := gcpshared.ExtractPathParam("cryptoKeyVersions", keyName)

			// Validate all parts before proceeding, a bit less performatic if any is missing but readability is improved
			if location != "" && keyRing != "" && cryptoKey != "" && cryptoKeyVersion != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.CloudKMSCryptoKeyVersion.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(location, keyRing, cryptoKey, cryptoKeyVersion),
						Scope:  c.ProjectID(),
					},
					//Deleting the custom-supplied key makes the machine unusable.
					//Deleting a machine in GCP does not affect its associated encryption key
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			}
		}
	}

	// The source instance used to create this machine image.
	// This can be a partial or full URL to the Compute Engine instance resource.
	// Example values:
	// - https://www.googleapis.com/compute/v1/projects/{project}/zones/{zone}/instances/{instance}
	// - projects/{project}/zones/{zone}/instances/{instance}
	if sourceInstance := machineImage.GetSourceInstance(); sourceInstance != "" {
		sourceInstanceName := gcpshared.LastPathComponent(sourceInstance)
		if sourceInstanceName != "" {
			zone := gcpshared.ExtractPathParam("zones", sourceInstance)
			if zone != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   ComputeInstance.String(),
						Method: sdp.QueryMethod_GET,
						Query:  sourceInstanceName,
						Scope:  gcpshared.ZonalScope(c.ProjectID(), zone),
					},
					// If sourceInstance gets deleted and user needs to recreate the machineImage in the future he will not be able to
					// Deleting a machine image does not impact source Instance
					BlastPropagation: &sdp.BlastPropagation{In: true, Out: false},
				})
			}
		}
	}

	// The status of the MachineImage.
	// For more information about the status of the MachineImage, see MachineImage life cycle.
	// Check the Status enum for the list of possible values.
	switch machineImage.GetStatus() {
	case computepb.MachineImage_READY.String():
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	case computepb.MachineImage_CREATING.String(),
		computepb.MachineImage_DELETING.String(),
		computepb.MachineImage_UPLOADING.String():
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	case computepb.MachineImage_INVALID.String():
		sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
	}

	return sdpItem, nil
}
