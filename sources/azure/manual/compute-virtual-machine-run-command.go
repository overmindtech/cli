package manual

import (
	"context"
	"errors"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var ComputeVirtualMachineRunCommandLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeVirtualMachineRunCommand)

type computeVirtualMachineRunCommandWrapper struct {
	client clients.VirtualMachineRunCommandsClient

	*azureshared.ResourceGroupBase
}

func NewComputeVirtualMachineRunCommand(client clients.VirtualMachineRunCommandsClient, subscriptionID, resourceGroup string) sources.SearchableWrapper {
	return &computeVirtualMachineRunCommandWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.ComputeVirtualMachineRunCommand,
		),
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-run-commands/get-by-virtual-machine?view=rest-compute-2025-04-01&tabs=HTTP
func (s computeVirtualMachineRunCommandWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if scope == "" {
		return nil, azureshared.QueryError(errors.New("scope cannot be empty"), scope, s.Type())
	}
	if len(queryParts) != 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires exactly 2 query parts: virtualMachineName and runCommandName",
			Scope:       scope,
			ItemType:    s.Type(),
		}
	}
	virtualMachineName := queryParts[0]
	runCommandName := queryParts[1]
	if virtualMachineName == "" {
		return nil, azureshared.QueryError(errors.New("virtualMachineName cannot be empty"), scope, s.Type())
	}
	if runCommandName == "" {
		return nil, azureshared.QueryError(errors.New("runCommandName cannot be empty"), scope, s.Type())
	}

	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = s.ResourceGroup()
	}

	resp, err := s.client.GetByVirtualMachine(ctx, resourceGroup, virtualMachineName, runCommandName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	return s.azureVirtualMachineRunCommandToSDPItem(&resp.VirtualMachineRunCommand, virtualMachineName, scope)
}

func (s computeVirtualMachineRunCommandWrapper) azureVirtualMachineRunCommandToSDPItem(runCommand *armcompute.VirtualMachineRunCommand, virtualMachineName, scope string) (*sdp.Item, *sdp.QueryError) {
	if runCommand == nil {
		return nil, azureshared.QueryError(errors.New("runCommand is nil"), scope, s.Type())
	}
	if runCommand.Name == nil {
		return nil, azureshared.QueryError(errors.New("runCommand name is nil"), scope, s.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(runCommand, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(virtualMachineName, *runCommand.Name))
	if err != nil {
		return nil, azureshared.QueryError(err, scope, s.Type())
	}

	sdpItem := &sdp.Item{
		Type:            s.Type(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            azureshared.ConvertAzureTags(runCommand.Tags),
	}

	// Link to Virtual Machine (parent resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/get?view=rest-compute-2025-04-01&tabs=HTTP
	// GET https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/virtualMachines/{vmName}?api-version=2025-04-01
	if virtualMachineName != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.ComputeVirtualMachine.String(),
				Method: sdp.QueryMethod_GET,
				Query:  virtualMachineName,
				Scope:  scope,
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,  // If VM is deleted/modified → Run Command becomes invalid (In: true)
				Out: false, // If Run Command is deleted → VM remains functional (Out: false)
			}, // Run Command is a child resource of VM
		})
	}

	// Process properties for blob URIs and script URIs
	if runCommand.Properties != nil {
		// Helper function to process managed identity and create links to User Assigned Managed Identity
		// Reference: https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/get?view=rest-managedidentity-2024-11-30&tabs=HTTP
		// GET /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.ManagedIdentity/userAssignedIdentities/{identityName}
		processManagedIdentity := func(managedIdentity *armcompute.RunCommandManagedIdentity) {
			if managedIdentity == nil {
				return
			}
			// Managed identity can be referenced by ClientID or ObjectID
			// Since we don't have the resource name, we use SEARCH method with ClientID/ObjectID
			var identityQuery string
			if managedIdentity.ClientID != nil && *managedIdentity.ClientID != "" {
				identityQuery = *managedIdentity.ClientID
			} else if managedIdentity.ObjectID != nil && *managedIdentity.ObjectID != "" {
				identityQuery = *managedIdentity.ObjectID
			} else {
				// System-assigned identity (empty object) - no link needed
				return
			}

			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
					Method: sdp.QueryMethod_SEARCH,
					Query:  identityQuery,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If Managed Identity is deleted/modified → Run Command cannot access blob/script (In: true)
					Out: false, // If Run Command is deleted → Managed Identity remains (Out: false)
				}, // Run Command depends on Managed Identity for blob/script access
			})
		}

		// Helper function to process blob URI and create links to Storage Account and Blob Container
		processBlobURI := func(blobURI *string) {
			if blobURI == nil || *blobURI == "" {
				return
			}

			uri := *blobURI
			isBlobURI := strings.Contains(uri, ".blob.core.windows.net")

			// Check if it's an Azure blob URI (contains .blob.core.windows.net)
			if isBlobURI {
				storageAccountName := azureshared.ExtractStorageAccountNameFromBlobURI(uri)
				if storageAccountName != "" {
					// Link to Storage Account
					// Reference: https://learn.microsoft.com/en-us/rest/api/storagerp/storage-accounts/get-properties?view=rest-storagerp-2025-06-01&tabs=HTTP
					// GET https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Storage/storageAccounts/{accountName}?api-version=2025-06-01
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.StorageAccount.String(),
							Method: sdp.QueryMethod_GET,
							Query:  storageAccountName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Storage Account is deleted/modified → blob becomes inaccessible (In: true)
							Out: false, // If Run Command is deleted → Storage Account remains (Out: false)
						}, // Run Command depends on Storage Account for blob access
					})

					// Extract container name and link to Blob Container
					containerName := azureshared.ExtractContainerNameFromBlobURI(uri)
					if containerName != "" {
						// Link to Blob Container
						// Reference: https://learn.microsoft.com/en-us/rest/api/storagerp/blob-containers/get?view=rest-storagerp-2025-06-01&tabs=HTTP
						// GET https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Storage/storageAccounts/{accountName}/blobServices/default/containers/{containerName}?api-version=2025-06-01
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.StorageBlobContainer.String(),
								Method: sdp.QueryMethod_GET,
								Query:  shared.CompositeLookupKey(storageAccountName, containerName),
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // If Blob Container is deleted/modified → blob becomes inaccessible (In: true)
								Out: false, // If Run Command is deleted → Blob Container remains (Out: false)
							}, // Run Command depends on Blob Container for blob access
						})
					}
				}
			}

			// Link to stdlib.NetworkHTTP and DNS only for non-blob URIs
			// For blob URIs, the StorageBlobContainer already has these links
			if !isBlobURI && (strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://")) {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkHTTP.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  uri,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // If HTTP endpoint is unavailable → Run Command cannot access script/blob (In: true)
						Out: true, // If Run Command is deleted → HTTP endpoint may still be used by other resources (Out: true)
					}, // Run Command depends on HTTP endpoint for script/blob access
				})

				// Link to DNS name (standard library) from URI
				dnsName := azureshared.ExtractDNSFromURL(uri)
				if dnsName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   stdlib.NetworkDNS.String(),
							Method: sdp.QueryMethod_SEARCH,
							Query:  dnsName,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true, // If DNS name is unavailable → Run Command cannot resolve endpoint (In: true)
							Out: true, // If Run Command is deleted → DNS name may still be used by other resources (Out: true)
						}, // Run Command depends on DNS name for endpoint resolution
					})
				}
			}
		}

		// Link to Storage Account and Blob Container from outputBlobUri
		// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-run-commands/get-by-virtual-machine?view=rest-compute-2025-04-01&tabs=HTTP
		if runCommand.Properties.OutputBlobURI != nil {
			processBlobURI(runCommand.Properties.OutputBlobURI)
		}

		// Link to Managed Identity from outputBlobManagedIdentity
		if runCommand.Properties.OutputBlobManagedIdentity != nil {
			processManagedIdentity(runCommand.Properties.OutputBlobManagedIdentity)
		}

		// Link to Storage Account and Blob Container from errorBlobUri
		if runCommand.Properties.ErrorBlobURI != nil {
			processBlobURI(runCommand.Properties.ErrorBlobURI)
		}

		// Link to Managed Identity from errorBlobManagedIdentity
		if runCommand.Properties.ErrorBlobManagedIdentity != nil {
			processManagedIdentity(runCommand.Properties.ErrorBlobManagedIdentity)
		}

		// Link to Storage Account, Blob Container, HTTP, and DNS from source.scriptUri
		if runCommand.Properties.Source != nil && runCommand.Properties.Source.ScriptURI != nil {
			processBlobURI(runCommand.Properties.Source.ScriptURI)
		}

		// Link to Managed Identity from source.scriptUriManagedIdentity
		if runCommand.Properties.Source != nil && runCommand.Properties.Source.ScriptURIManagedIdentity != nil {
			processManagedIdentity(runCommand.Properties.Source.ScriptURIManagedIdentity)
		}
	}

	return sdpItem, nil
}

func (s computeVirtualMachineRunCommandWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeVirtualMachineLookupByName,
		ComputeVirtualMachineRunCommandLookupByName,
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-run-commands/list-by-virtual-machine?view=rest-compute-2025-04-01&tabs=HTTP
func (s computeVirtualMachineRunCommandWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("search requires exactly 1 query part: virtualMachineName"), scope, s.Type())
	}
	virtualMachineName := queryParts[0]
	if virtualMachineName == "" {
		return nil, azureshared.QueryError(errors.New("virtualMachineName cannot be empty"), scope, s.Type())
	}

	resourceGroup := azureshared.ResourceGroupFromScope(scope)
	if resourceGroup == "" {
		resourceGroup = s.ResourceGroup()
	}
	pager := s.client.NewListByVirtualMachinePager(resourceGroup, virtualMachineName, nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, scope, s.Type())
		}
		for _, runCommand := range page.Value {
			if runCommand.Name == nil {
				continue
			}
			item, sdpErr := s.azureVirtualMachineRunCommandToSDPItem(runCommand, virtualMachineName, scope)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (s computeVirtualMachineRunCommandWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			ComputeVirtualMachineLookupByName,
		},
	}
}

func (s computeVirtualMachineRunCommandWrapper) PotentialLinks() map[shared.ItemType]bool {
	return map[shared.ItemType]bool{
		azureshared.ComputeVirtualMachine:               true,
		azureshared.StorageAccount:                      true,
		azureshared.StorageBlobContainer:                true,
		azureshared.ManagedIdentityUserAssignedIdentity: true,
		stdlib.NetworkHTTP:                              true,
		stdlib.NetworkDNS:                               true,
	}
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/virtual_machine_run_command#attributes-reference
func (s computeVirtualMachineRunCommandWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_SEARCH,
			TerraformQueryMap: "azurerm_virtual_machine_run_command.id",
		},
	}
}

// ref: https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/compute#microsoftcompute
func (s computeVirtualMachineRunCommandWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Compute/virtualMachines/runCommands/read",
	}
}

func (s computeVirtualMachineRunCommandWrapper) PredefinedRole() string {
	return "Reader"
}
