package manual

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var ComputeVirtualMachineScaleSetLookupByName = shared.NewItemTypeLookup("name", azureshared.ComputeVirtualMachineScaleSet)

type computeVirtualMachineScaleSetWrapper struct {
	client clients.VirtualMachineScaleSetsClient

	*azureshared.ResourceGroupBase
}

func NewComputeVirtualMachineScaleSet(client clients.VirtualMachineScaleSetsClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &computeVirtualMachineScaleSetWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			azureshared.ComputeVirtualMachineScaleSet,
		),
	}
}

// ref: https://linear.app/overmind/issue/ENG-2114/create-microsoftcomputevirtualmachinescalesets-adapter
func (c computeVirtualMachineScaleSetWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	pager := c.client.NewListPager(c.ResourceGroup(), nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, c.DefaultScope(), c.Type())
		}
		for _, scaleSet := range page.Value {
			item, sdpErr := c.azureVirtualMachineScaleSetToSDPItem(scaleSet)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func (c computeVirtualMachineScaleSetWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	pager := c.client.NewListPager(c.ResourceGroup(), nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, c.DefaultScope(), c.Type()))
			return
		}
		for _, scaleSet := range page.Value {
			item, sdpErr := c.azureVirtualMachineScaleSetToSDPItem(scaleSet)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// ref: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-scale-sets/get?view=rest-compute-2025-04-01&tabs=HTTP
func (c computeVirtualMachineScaleSetWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) < 1 {
		return nil, azureshared.QueryError(errors.New("queryParts must be at least 1"), c.DefaultScope(), c.Type())
	}
	scaleSetName := queryParts[0]
	if scaleSetName == "" {
		return nil, azureshared.QueryError(errors.New("scaleSetName cannot be empty"), c.DefaultScope(), c.Type())
	}
	scaleSet, err := c.client.Get(ctx, c.ResourceGroup(), scaleSetName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, c.DefaultScope(), c.Type())
	}

	return c.azureVirtualMachineScaleSetToSDPItem(&scaleSet.VirtualMachineScaleSet)
}

func (c computeVirtualMachineScaleSetWrapper) azureVirtualMachineScaleSetToSDPItem(scaleSet *armcompute.VirtualMachineScaleSet) (*sdp.Item, *sdp.QueryError) {
	if scaleSet.Name == nil {
		return nil, azureshared.QueryError(errors.New("scaleSetName is nil"), c.DefaultScope(), c.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(scaleSet, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, c.DefaultScope(), c.Type())
	}

	sdpItem := &sdp.Item{
		Type:            azureshared.ComputeVirtualMachineScaleSet.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),
		Tags:            azureshared.ConvertAzureTags(scaleSet.Tags),
	}

	scaleSetName := *scaleSet.Name

	// Track added links to prevent duplicates (key: type:query:scope)
	addedLinks := make(map[string]bool)
	addLink := func(link *sdp.LinkedItemQuery) {
		key := fmt.Sprintf("%s:%s:%s", link.GetQuery().GetType(), link.GetQuery().GetQuery(), link.GetQuery().GetScope())
		if !addedLinks[key] {
			addedLinks[key] = true
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, link)
		}
	}

	// Link to extensions (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-scale-set-extensions/get?view=rest-compute-2025-04-01&tabs=HTTP
	if scaleSet.Properties != nil && scaleSet.Properties.VirtualMachineProfile != nil &&
		scaleSet.Properties.VirtualMachineProfile.ExtensionProfile != nil &&
		scaleSet.Properties.VirtualMachineProfile.ExtensionProfile.Extensions != nil {
		for _, extension := range scaleSet.Properties.VirtualMachineProfile.ExtensionProfile.Extensions {
			if extension.Name != nil && scaleSetName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ComputeVirtualMachineExtension.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(scaleSetName, *extension.Name),
						Scope:  c.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  false, // If Extensions are deleted → VMSS remains functional (In: false)
						Out: true,  // If VMSS is deleted → Extensions become invalid/unusable (Out: true)
					},
				})
			}
		}
	}

	// Link to VM instances (child resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machine-scale-set-vms/list?view=rest-compute-2025-04-01&tabs=HTTP
	// Note: VM instances are listed via SEARCH method since we can list all instances for a VMSS
	if scaleSetName != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   azureshared.ComputeVirtualMachine.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  scaleSetName,
				Scope:  c.DefaultScope(),
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  false, // If VM instances are deleted → VMSS remains functional (In: false)
				Out: true,  // If VMSS is deleted → VM instances become invalid/unusable (Out: true)
			},
		})
	}

	// Link to network resources
	if scaleSet.Properties != nil && scaleSet.Properties.VirtualMachineProfile != nil &&
		scaleSet.Properties.VirtualMachineProfile.NetworkProfile != nil &&
		scaleSet.Properties.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations != nil {
		for _, nicConfig := range scaleSet.Properties.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations {
			if nicConfig.Properties != nil {
				// Link to Network Security Group
				// Reference: https://learn.microsoft.com/en-us/rest/api/virtual-network/network-security-groups/get
				if nicConfig.Properties.NetworkSecurityGroup != nil && nicConfig.Properties.NetworkSecurityGroup.ID != nil {
					nsgName := azureshared.ExtractResourceName(*nicConfig.Properties.NetworkSecurityGroup.ID)
					if nsgName != "" {
						scope := c.DefaultScope()
						// Check if NSG is in a different resource group
						if extractedScope := azureshared.ExtractScopeFromResourceID(*nicConfig.Properties.NetworkSecurityGroup.ID); extractedScope != "" {
							scope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkNetworkSecurityGroup.String(),
								Method: sdp.QueryMethod_GET,
								Query:  nsgName,
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // If NSG changes → VMSS network behavior changes (In: true)
								Out: false, // If VMSS is deleted → NSG remains (Out: false)
							},
						})
					}
				}

				// Link to IP configurations
				if nicConfig.Properties.IPConfigurations != nil {
					for _, ipConfig := range nicConfig.Properties.IPConfigurations {
						if ipConfig.Properties != nil {
							// Link to Subnet
							// Reference: https://learn.microsoft.com/en-us/rest/api/virtual-network/subnets/get
							if ipConfig.Properties.Subnet != nil && ipConfig.Properties.Subnet.ID != nil {
								subnetID := *ipConfig.Properties.Subnet.ID
								// Extract virtual network name and subnet name from ID
								// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks/{vnetName}/subnets/{subnetName}
								parts := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"virtualNetworks", "subnets"})
								if len(parts) >= 2 {
									vnetName := parts[0]
									subnetName := parts[1]
									scope := c.DefaultScope()
									// Check if subnet is in a different resource group
									if extractedScope := azureshared.ExtractScopeFromResourceID(subnetID); extractedScope != "" {
										scope = extractedScope
									}
									// Link to Virtual Network
									// Reference: https://learn.microsoft.com/en-us/rest/api/virtual-network/virtual-networks/get
									sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
										Query: &sdp.Query{
											Type:   azureshared.NetworkVirtualNetwork.String(),
											Method: sdp.QueryMethod_GET,
											Query:  vnetName,
											Scope:  scope,
										},
										BlastPropagation: &sdp.BlastPropagation{
											In:  true,  // If Virtual Network changes → VMSS network behavior changes (In: true)
											Out: false, // If VMSS is deleted → Virtual Network remains (Out: false)
										},
									})
									// Link to Subnet
									sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
										Query: &sdp.Query{
											Type:   azureshared.NetworkSubnet.String(),
											Method: sdp.QueryMethod_GET,
											Query:  shared.CompositeLookupKey(vnetName, subnetName),
											Scope:  scope,
										},
										BlastPropagation: &sdp.BlastPropagation{
											In:  true,  // If Subnet changes → VMSS network behavior changes (In: true)
											Out: false, // If VMSS is deleted → Subnet remains (Out: false)
										},
									})
								}
							}

							// Link to Public IP Address Configuration
							// Reference: https://learn.microsoft.com/en-us/rest/api/virtual-network/public-ip-addresses/get
							if ipConfig.Properties.PublicIPAddressConfiguration != nil &&
								ipConfig.Properties.PublicIPAddressConfiguration.Properties != nil &&
								ipConfig.Properties.PublicIPAddressConfiguration.Properties.PublicIPPrefix != nil &&
								ipConfig.Properties.PublicIPAddressConfiguration.Properties.PublicIPPrefix.ID != nil {
								publicIPPrefixName := azureshared.ExtractResourceName(*ipConfig.Properties.PublicIPAddressConfiguration.Properties.PublicIPPrefix.ID)
								if publicIPPrefixName != "" {
									scope := c.DefaultScope()
									// Check if Public IP Prefix is in a different resource group
									if extractedScope := azureshared.ExtractScopeFromResourceID(*ipConfig.Properties.PublicIPAddressConfiguration.Properties.PublicIPPrefix.ID); extractedScope != "" {
										scope = extractedScope
									}
									sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
										Query: &sdp.Query{
											Type:   azureshared.NetworkPublicIPPrefix.String(),
											Method: sdp.QueryMethod_GET,
											Query:  publicIPPrefixName,
											Scope:  scope,
										},
										BlastPropagation: &sdp.BlastPropagation{
											In:  true,  // If Public IP Prefix changes → VMSS public IP allocation changes (In: true)
											Out: false, // If VMSS is deleted → Public IP Prefix remains (Out: false)
										},
									})
								}
							}

							// Link to Load Balancer Backend Address Pools
							// Reference: https://learn.microsoft.com/en-us/rest/api/load-balancer/backend-address-pools/get
							if ipConfig.Properties.LoadBalancerBackendAddressPools != nil {
								for _, poolRef := range ipConfig.Properties.LoadBalancerBackendAddressPools {
									if poolRef.ID != nil {
										poolID := *poolRef.ID
										// Extract load balancer name and pool name from ID
										// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/loadBalancers/{lbName}/backendAddressPools/{poolName}
										parts := azureshared.ExtractPathParamsFromResourceID(poolID, []string{"loadBalancers", "backendAddressPools"})
										if len(parts) >= 2 {
											lbName := parts[0]
											poolName := parts[1]
											scope := c.DefaultScope()
											// Check if Load Balancer is in a different resource group
											if extractedScope := azureshared.ExtractScopeFromResourceID(poolID); extractedScope != "" {
												scope = extractedScope
											}
											// Link to Load Balancer (deduplicated - same LB may be referenced by multiple child resources)
											// Reference: https://learn.microsoft.com/en-us/rest/api/load-balancer/load-balancers/get
											addLink(&sdp.LinkedItemQuery{
												Query: &sdp.Query{
													Type:   azureshared.NetworkLoadBalancer.String(),
													Method: sdp.QueryMethod_GET,
													Query:  lbName,
													Scope:  scope,
												},
												BlastPropagation: &sdp.BlastPropagation{
													In:  true,  // If Load Balancer changes → VMSS load balancing changes (In: true)
													Out: false, // If VMSS is deleted → Load Balancer remains (Out: false)
												},
											})
											// Link to Backend Address Pool
											sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
												Query: &sdp.Query{
													Type:   azureshared.NetworkLoadBalancerBackendAddressPool.String(),
													Method: sdp.QueryMethod_GET,
													Query:  shared.CompositeLookupKey(lbName, poolName),
													Scope:  scope,
												},
												BlastPropagation: &sdp.BlastPropagation{
													In:  true, // If Backend Pool changes → VMSS load balancing changes (In: true)
													Out: true, // If VMSS is deleted → Backend Pool loses members (Out: true)
												},
											})
										}
									}
								}
							}

							// Link to Load Balancer Inbound NAT Pools
							// Reference: https://learn.microsoft.com/en-us/rest/api/load-balancer/inbound-nat-pools/get
							if ipConfig.Properties.LoadBalancerInboundNatPools != nil {
								for _, natPoolRef := range ipConfig.Properties.LoadBalancerInboundNatPools {
									if natPoolRef.ID != nil {
										natPoolID := *natPoolRef.ID
										// Extract load balancer name and NAT pool name from ID
										// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/loadBalancers/{lbName}/inboundNatPools/{poolName}
										parts := azureshared.ExtractPathParamsFromResourceID(natPoolID, []string{"loadBalancers", "inboundNatPools"})
										if len(parts) >= 2 {
											lbName := parts[0]
											poolName := parts[1]
											scope := c.DefaultScope()
											// Check if Load Balancer is in a different resource group
											if extractedScope := azureshared.ExtractScopeFromResourceID(natPoolID); extractedScope != "" {
												scope = extractedScope
											}
											// Link to Load Balancer (deduplicated - same LB may be referenced by multiple child resources)
											// Reference: https://learn.microsoft.com/en-us/rest/api/load-balancer/load-balancers/get
											addLink(&sdp.LinkedItemQuery{
												Query: &sdp.Query{
													Type:   azureshared.NetworkLoadBalancer.String(),
													Method: sdp.QueryMethod_GET,
													Query:  lbName,
													Scope:  scope,
												},
												BlastPropagation: &sdp.BlastPropagation{
													In:  true,  // If Load Balancer changes → VMSS load balancing changes (In: true)
													Out: false, // If VMSS is deleted → Load Balancer remains (Out: false)
												},
											})
											// Link to Inbound NAT Pool
											sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
												Query: &sdp.Query{
													Type:   azureshared.NetworkLoadBalancerInboundNatPool.String(),
													Method: sdp.QueryMethod_GET,
													Query:  shared.CompositeLookupKey(lbName, poolName),
													Scope:  scope,
												},
												BlastPropagation: &sdp.BlastPropagation{
													In:  true, // If NAT Pool changes → VMSS NAT behavior changes (In: true)
													Out: true, // If VMSS is deleted → NAT Pool loses members (Out: true)
												},
											})
										}
									}
								}
							}

							// Link to Application Gateway Backend Address Pools
							// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/backend-address-pools/get
							if ipConfig.Properties.ApplicationGatewayBackendAddressPools != nil {
								for _, agPoolRef := range ipConfig.Properties.ApplicationGatewayBackendAddressPools {
									if agPoolRef.ID != nil {
										agPoolID := *agPoolRef.ID
										// Extract application gateway name and pool name from ID
										// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/applicationGateways/{agName}/backendAddressPools/{poolName}
										parts := azureshared.ExtractPathParamsFromResourceID(agPoolID, []string{"applicationGateways", "backendAddressPools"})
										if len(parts) >= 2 {
											agName := parts[0]
											poolName := parts[1]
											scope := c.DefaultScope()
											// Check if Application Gateway is in a different resource group
											if extractedScope := azureshared.ExtractScopeFromResourceID(agPoolID); extractedScope != "" {
												scope = extractedScope
											}
											// Link to Application Gateway (deduplicated - same AG may be referenced by multiple child resources)
											// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateways/get
											addLink(&sdp.LinkedItemQuery{
												Query: &sdp.Query{
													Type:   azureshared.NetworkApplicationGateway.String(),
													Method: sdp.QueryMethod_GET,
													Query:  agName,
													Scope:  scope,
												},
												BlastPropagation: &sdp.BlastPropagation{
													In:  true,  // If Application Gateway changes → VMSS routing changes (In: true)
													Out: false, // If VMSS is deleted → Application Gateway remains (Out: false)
												},
											})
											// Link to Backend Address Pool
											sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
												Query: &sdp.Query{
													Type:   azureshared.NetworkApplicationGatewayBackendAddressPool.String(),
													Method: sdp.QueryMethod_GET,
													Query:  shared.CompositeLookupKey(agName, poolName),
													Scope:  scope,
												},
												BlastPropagation: &sdp.BlastPropagation{
													In:  true, // If Backend Pool changes → VMSS routing changes (In: true)
													Out: true, // If VMSS is deleted → Backend Pool loses members (Out: true)
												},
											})
										}
									}
								}
							}

							// Link to Application Security Groups
							// Reference: https://learn.microsoft.com/en-us/rest/api/virtual-network/application-security-groups/get
							if ipConfig.Properties.ApplicationSecurityGroups != nil {
								for _, asgRef := range ipConfig.Properties.ApplicationSecurityGroups {
									if asgRef.ID != nil {
										asgName := azureshared.ExtractResourceName(*asgRef.ID)
										if asgName != "" {
											scope := c.DefaultScope()
											// Check if Application Security Group is in a different resource group
											if extractedScope := azureshared.ExtractScopeFromResourceID(*asgRef.ID); extractedScope != "" {
												scope = extractedScope
											}
											sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
												Query: &sdp.Query{
													Type:   azureshared.NetworkApplicationSecurityGroup.String(),
													Method: sdp.QueryMethod_GET,
													Query:  asgName,
													Scope:  scope,
												},
												BlastPropagation: &sdp.BlastPropagation{
													In:  true,  // If ASG changes → VMSS network rules change (In: true)
													Out: false, // If VMSS is deleted → ASG remains (Out: false)
												},
											})
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Link to Load Balancer Health Probe
	// Reference: https://learn.microsoft.com/en-us/rest/api/load-balancer/load-balancer-probes/get
	// Note: Health probe is at NetworkProfile level and doesn't require NetworkInterfaceConfigurations
	if scaleSet.Properties != nil && scaleSet.Properties.VirtualMachineProfile != nil &&
		scaleSet.Properties.VirtualMachineProfile.NetworkProfile != nil &&
		scaleSet.Properties.VirtualMachineProfile.NetworkProfile.HealthProbe != nil &&
		scaleSet.Properties.VirtualMachineProfile.NetworkProfile.HealthProbe.ID != nil {
		probeID := *scaleSet.Properties.VirtualMachineProfile.NetworkProfile.HealthProbe.ID
		// Extract load balancer name and probe name from ID
		// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/loadBalancers/{lbName}/probes/{probeName}
		parts := azureshared.ExtractPathParamsFromResourceID(probeID, []string{"loadBalancers", "probes"})
		if len(parts) >= 2 {
			lbName := parts[0]
			probeName := parts[1]
			scope := c.DefaultScope()
			// Check if Load Balancer is in a different resource group
			if extractedScope := azureshared.ExtractScopeFromResourceID(probeID); extractedScope != "" {
				scope = extractedScope
			}
			// Link to Load Balancer (deduplicated - same LB may be referenced by multiple child resources)
			// Reference: https://learn.microsoft.com/en-us/rest/api/load-balancer/load-balancers/get
			addLink(&sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkLoadBalancer.String(),
					Method: sdp.QueryMethod_GET,
					Query:  lbName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If Load Balancer changes → VMSS load balancing changes (In: true)
					Out: false, // If VMSS is deleted → Load Balancer remains (Out: false)
				},
			})
			// Link to Health Probe
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkLoadBalancerProbe.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(lbName, probeName),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If Health Probe changes → VMSS health checks change (In: true)
					Out: false, // If VMSS is deleted → Health Probe remains (Out: false)
				},
			})
		}
	}

	// Link to storage resources
	if scaleSet.Properties != nil && scaleSet.Properties.VirtualMachineProfile != nil &&
		scaleSet.Properties.VirtualMachineProfile.StorageProfile != nil {
		// Link to OS Disk Encryption Set
		// Reference: https://learn.microsoft.com/en-us/rest/api/compute/disk-encryption-sets/get
		if scaleSet.Properties.VirtualMachineProfile.StorageProfile.OSDisk != nil &&
			scaleSet.Properties.VirtualMachineProfile.StorageProfile.OSDisk.ManagedDisk != nil &&
			scaleSet.Properties.VirtualMachineProfile.StorageProfile.OSDisk.ManagedDisk.DiskEncryptionSet != nil &&
			scaleSet.Properties.VirtualMachineProfile.StorageProfile.OSDisk.ManagedDisk.DiskEncryptionSet.ID != nil {
			encryptionSetName := azureshared.ExtractResourceName(*scaleSet.Properties.VirtualMachineProfile.StorageProfile.OSDisk.ManagedDisk.DiskEncryptionSet.ID)
			if encryptionSetName != "" {
				scope := c.DefaultScope()
				// Check if Disk Encryption Set is in a different resource group
				if extractedScope := azureshared.ExtractScopeFromResourceID(*scaleSet.Properties.VirtualMachineProfile.StorageProfile.OSDisk.ManagedDisk.DiskEncryptionSet.ID); extractedScope != "" {
					scope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ComputeDiskEncryptionSet.String(),
						Method: sdp.QueryMethod_GET,
						Query:  encryptionSetName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // If Disk Encryption Set changes → VMSS disk encryption changes (In: true)
						Out: false, // If VMSS is deleted → Disk Encryption Set remains (Out: false)
					},
				})
			}
		}

		// Link to Data Disk Encryption Sets
		if scaleSet.Properties.VirtualMachineProfile.StorageProfile.DataDisks != nil {
			for _, dataDisk := range scaleSet.Properties.VirtualMachineProfile.StorageProfile.DataDisks {
				if dataDisk.ManagedDisk != nil && dataDisk.ManagedDisk.DiskEncryptionSet != nil &&
					dataDisk.ManagedDisk.DiskEncryptionSet.ID != nil {
					encryptionSetName := azureshared.ExtractResourceName(*dataDisk.ManagedDisk.DiskEncryptionSet.ID)
					if encryptionSetName != "" {
						scope := c.DefaultScope()
						// Check if Disk Encryption Set is in a different resource group
						if extractedScope := azureshared.ExtractScopeFromResourceID(*dataDisk.ManagedDisk.DiskEncryptionSet.ID); extractedScope != "" {
							scope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.ComputeDiskEncryptionSet.String(),
								Method: sdp.QueryMethod_GET,
								Query:  encryptionSetName,
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // If Disk Encryption Set changes → VMSS disk encryption changes (In: true)
								Out: false, // If VMSS is deleted → Disk Encryption Set remains (Out: false)
							},
						})
					}
				}
			}
		}

		// Link to Image (if custom image with ID)
		// Reference: https://learn.microsoft.com/en-us/rest/api/compute/images/get
		if scaleSet.Properties.VirtualMachineProfile.StorageProfile.ImageReference != nil {
			imageRef := scaleSet.Properties.VirtualMachineProfile.StorageProfile.ImageReference
			// ImageReference can have:
			// 1. ID field for custom images: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Compute/images/{imageName}
			// 2. SharedGalleryImageID for shared gallery images: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Compute/galleries/{galleryName}/images/{imageName}/versions/{version}
			// 3. CommunityGalleryImageID for community gallery images: /CommunityGalleries/{communityGalleryName}/Images/{imageName}/Versions/{version}
			// 4. Publisher/Offer/Sku for marketplace images (no ID, so we can't link to them)

			// Link to custom image
			if imageRef.ID != nil {
				imageID := *imageRef.ID
				imageName := azureshared.ExtractResourceName(imageID)
				if imageName != "" {
					scope := c.DefaultScope()
					// Check if Image is in a different resource group
					if extractedScope := azureshared.ExtractScopeFromResourceID(imageID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ComputeImage.String(),
							Method: sdp.QueryMethod_GET,
							Query:  imageName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Image changes → VMSS VM configuration changes (In: true)
							Out: false, // If VMSS is deleted → Image remains (Out: false)
						},
					})
				}
			}

			// Link to Shared Gallery Image
			// Reference: https://learn.microsoft.com/en-us/rest/api/compute/shared-gallery-images/get
			if imageRef.SharedGalleryImageID != nil {
				sharedGalleryImageID := *imageRef.SharedGalleryImageID
				if sharedGalleryImageID != "" {
					// Extract gallery name, image name, and version from the ID
					// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Compute/galleries/{galleryName}/images/{imageName}/versions/{version}
					parts := azureshared.ExtractPathParamsFromResourceID(sharedGalleryImageID, []string{"galleries", "images", "versions"})
					if len(parts) >= 3 {
						galleryName := parts[0]
						imageName := parts[1]
						version := parts[2]
						scope := c.DefaultScope()
						// Check if gallery is in a different resource group
						if extractedScope := azureshared.ExtractScopeFromResourceID(sharedGalleryImageID); extractedScope != "" {
							scope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.ComputeSharedGalleryImage.String(),
								Method: sdp.QueryMethod_GET,
								Query:  shared.CompositeLookupKey(galleryName, imageName, version),
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // If Gallery Image changes → VMSS VM configuration changes (In: true)
								Out: false, // If VMSS is deleted → Gallery Image remains (Out: false)
							},
						})
					}
				}
			}

			// Link to Community Gallery Image
			// Reference: https://learn.microsoft.com/en-us/rest/api/compute/community-gallery-images/get
			if imageRef.CommunityGalleryImageID != nil {
				communityGalleryImageID := *imageRef.CommunityGalleryImageID
				if communityGalleryImageID != "" {
					// Extract community gallery name, image name, and version from the ID
					// Format: /CommunityGalleries/{communityGalleryName}/Images/{imageName}/Versions/{version}
					// Note: Community gallery IDs don't follow standard Azure resource ID format
					parts := azureshared.ExtractPathParamsFromResourceID(communityGalleryImageID, []string{"CommunityGalleries", "Images", "Versions"})
					if len(parts) >= 3 {
						communityGalleryName := parts[0]
						imageName := parts[1]
						version := parts[2]
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.ComputeCommunityGalleryImage.String(),
								Method: sdp.QueryMethod_GET,
								Query:  shared.CompositeLookupKey(communityGalleryName, imageName, version),
								Scope:  c.DefaultScope(), // Community galleries are subscription-level
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // If Gallery Image changes → VMSS VM configuration changes (In: true)
								Out: false, // If VMSS is deleted → Gallery Image remains (Out: false)
							},
						})
					}
				}
			}
		}
	}

	// Link to Gallery Application Versions
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/gallery-application-versions/get
	if scaleSet.Properties != nil && scaleSet.Properties.VirtualMachineProfile != nil &&
		scaleSet.Properties.VirtualMachineProfile.ApplicationProfile != nil &&
		scaleSet.Properties.VirtualMachineProfile.ApplicationProfile.GalleryApplications != nil {
		for _, galleryApp := range scaleSet.Properties.VirtualMachineProfile.ApplicationProfile.GalleryApplications {
			if galleryApp.PackageReferenceID != nil {
				packageRefID := *galleryApp.PackageReferenceID
				if packageRefID != "" {
					// Extract gallery name, application name, and version from the ID
					// Format: /subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Compute/galleries/{galleryName}/applications/{application}/versions/{version}
					parts := azureshared.ExtractPathParamsFromResourceID(packageRefID, []string{"galleries", "applications", "versions"})
					if len(parts) >= 3 {
						galleryName := parts[0]
						applicationName := parts[1]
						version := parts[2]
						scope := c.DefaultScope()
						// Check if gallery is in a different resource group
						if extractedScope := azureshared.ExtractScopeFromResourceID(packageRefID); extractedScope != "" {
							scope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.ComputeSharedGalleryApplicationVersion.String(),
								Method: sdp.QueryMethod_GET,
								Query:  shared.CompositeLookupKey(galleryName, applicationName, version),
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // If Gallery Application Version changes → VMSS application configuration changes (In: true)
								Out: false, // If VMSS is deleted → Gallery Application Version remains (Out: false)
							},
						})
					}
				}
			}
		}
	}

	// Link to compute resources
	if scaleSet.Properties != nil {
		// Link to Proximity Placement Group
		// Reference: https://learn.microsoft.com/en-us/rest/api/compute/proximity-placement-groups/get
		if scaleSet.Properties.ProximityPlacementGroup != nil && scaleSet.Properties.ProximityPlacementGroup.ID != nil {
			ppgName := azureshared.ExtractResourceName(*scaleSet.Properties.ProximityPlacementGroup.ID)
			if ppgName != "" {
				scope := c.DefaultScope()
				// Check if Proximity Placement Group is in a different resource group
				if extractedScope := azureshared.ExtractScopeFromResourceID(*scaleSet.Properties.ProximityPlacementGroup.ID); extractedScope != "" {
					scope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ComputeProximityPlacementGroup.String(),
						Method: sdp.QueryMethod_GET,
						Query:  ppgName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // If PPG changes → VMSS placement changes (In: true)
						Out: false, // If VMSS is deleted → PPG remains (Out: false)
					},
				})
			}
		}

		// Link to Dedicated Host Group
		// Reference: https://learn.microsoft.com/en-us/rest/api/compute/dedicated-host-groups/get
		if scaleSet.Properties.HostGroup != nil && scaleSet.Properties.HostGroup.ID != nil {
			hostGroupName := azureshared.ExtractResourceName(*scaleSet.Properties.HostGroup.ID)
			if hostGroupName != "" {
				scope := c.DefaultScope()
				// Check if Dedicated Host Group is in a different resource group
				if extractedScope := azureshared.ExtractScopeFromResourceID(*scaleSet.Properties.HostGroup.ID); extractedScope != "" {
					scope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ComputeDedicatedHostGroup.String(),
						Method: sdp.QueryMethod_GET,
						Query:  hostGroupName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // If Host Group changes → VMSS host placement changes (In: true)
						Out: false, // If VMSS is deleted → Host Group remains (Out: false)
					},
				})
			}
		}
	}

	// Link to Capacity Reservation Group
	// Reference: https://learn.microsoft.com/en-us/rest/api/compute/capacity-reservation-groups/get
	if scaleSet.Properties != nil && scaleSet.Properties.VirtualMachineProfile != nil &&
		scaleSet.Properties.VirtualMachineProfile.CapacityReservation != nil &&
		scaleSet.Properties.VirtualMachineProfile.CapacityReservation.CapacityReservationGroup != nil &&
		scaleSet.Properties.VirtualMachineProfile.CapacityReservation.CapacityReservationGroup.ID != nil {
		capacityReservationGroupName := azureshared.ExtractResourceName(*scaleSet.Properties.VirtualMachineProfile.CapacityReservation.CapacityReservationGroup.ID)
		if capacityReservationGroupName != "" {
			scope := c.DefaultScope()
			// Check if Capacity Reservation Group is in a different resource group
			if extractedScope := azureshared.ExtractScopeFromResourceID(*scaleSet.Properties.VirtualMachineProfile.CapacityReservation.CapacityReservationGroup.ID); extractedScope != "" {
				scope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.ComputeCapacityReservationGroup.String(),
					Method: sdp.QueryMethod_GET,
					Query:  capacityReservationGroupName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // If Capacity Reservation Group changes → VMSS capacity reservation changes (In: true)
					Out: false, // If VMSS is deleted → Capacity Reservation Group remains (Out: false)
				},
			})
		}
	}

	// Link to identity resources
	// Reference: https://learn.microsoft.com/en-us/rest/api/msi/user-assigned-identities/get
	if scaleSet.Identity != nil && scaleSet.Identity.UserAssignedIdentities != nil {
		for identityID := range scaleSet.Identity.UserAssignedIdentities {
			if identityID != "" {
				identityName := azureshared.ExtractResourceName(identityID)
				if identityName != "" {
					scope := c.DefaultScope()
					// Check if identity is in a different resource group
					if extractedScope := azureshared.ExtractScopeFromResourceID(identityID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
							Method: sdp.QueryMethod_GET,
							Query:  identityName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Identity changes → VMSS access changes (In: true)
							Out: false, // If VMSS is deleted → Identity remains (Out: false)
						},
					})
				}
			}
		}
	}

	// Link to storage account for boot diagnostics
	// Reference: https://learn.microsoft.com/en-us/rest/api/storagerp/storage-accounts/get-properties
	if scaleSet.Properties != nil && scaleSet.Properties.VirtualMachineProfile != nil &&
		scaleSet.Properties.VirtualMachineProfile.DiagnosticsProfile != nil &&
		scaleSet.Properties.VirtualMachineProfile.DiagnosticsProfile.BootDiagnostics != nil &&
		scaleSet.Properties.VirtualMachineProfile.DiagnosticsProfile.BootDiagnostics.StorageURI != nil {
		storageURI := *scaleSet.Properties.VirtualMachineProfile.DiagnosticsProfile.BootDiagnostics.StorageURI
		// Extract storage account name from URI
		// Format: https://{accountName}.blob.core.windows.net/
		if storageURI != "" {
			// Parse the storage account name from the URI
			// The URI format is: https://{accountName}.blob.core.windows.net/
			dnsName := azureshared.ExtractDNSFromURL(storageURI)
			if dnsName != "" {
				// Link to DNS name (standard library)
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   "dns",
						Method: sdp.QueryMethod_SEARCH,
						Query:  dnsName,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// DNS names are always linked
						In:  true,
						Out: true,
					},
				})

				// Extract account name (everything before the first dot)
				// dnsName format: accountname.blob.core.windows.net
				accountName := ""
				for i := range len(dnsName) {
					if dnsName[i] == '.' {
						accountName = dnsName[:i]
						break
					}
				}
				if accountName != "" {
					// Storage accounts are typically in the same resource group, but we use DefaultScope
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.StorageAccount.String(),
							Method: sdp.QueryMethod_GET,
							Query:  accountName,
							Scope:  c.DefaultScope(),
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Storage Account changes → VMSS boot diagnostics affected (In: true)
							Out: false, // If VMSS is deleted → Storage Account remains (Out: false)
						},
					})
				}
			}
		}
	}

	// Link to Key Vault for secrets
	// Reference: https://learn.microsoft.com/en-us/rest/api/keyvault/vaults/get
	if scaleSet.Properties != nil && scaleSet.Properties.VirtualMachineProfile != nil &&
		scaleSet.Properties.VirtualMachineProfile.OSProfile != nil &&
		scaleSet.Properties.VirtualMachineProfile.OSProfile.Secrets != nil {
		for _, secret := range scaleSet.Properties.VirtualMachineProfile.OSProfile.Secrets {
			if secret.SourceVault != nil && secret.SourceVault.ID != nil {
				vaultName := azureshared.ExtractResourceName(*secret.SourceVault.ID)
				if vaultName != "" {
					scope := c.DefaultScope()
					// Check if Key Vault is in a different resource group
					if extractedScope := azureshared.ExtractScopeFromResourceID(*secret.SourceVault.ID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.KeyVaultVault.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vaultName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Key Vault changes → VMSS secrets access changes (In: true)
							Out: false, // If VMSS is deleted → Key Vault remains (Out: false)
						},
					})
				}
			}
		}
	}

	// Link to Key Vault for extension protected settings
	if scaleSet.Properties != nil && scaleSet.Properties.VirtualMachineProfile != nil &&
		scaleSet.Properties.VirtualMachineProfile.ExtensionProfile != nil &&
		scaleSet.Properties.VirtualMachineProfile.ExtensionProfile.Extensions != nil {
		for _, extension := range scaleSet.Properties.VirtualMachineProfile.ExtensionProfile.Extensions {
			if extension.Properties != nil && extension.Properties.ProtectedSettingsFromKeyVault != nil &&
				extension.Properties.ProtectedSettingsFromKeyVault.SourceVault != nil &&
				extension.Properties.ProtectedSettingsFromKeyVault.SourceVault.ID != nil {
				vaultName := azureshared.ExtractResourceName(*extension.Properties.ProtectedSettingsFromKeyVault.SourceVault.ID)
				if vaultName != "" {
					scope := c.DefaultScope()
					// Check if Key Vault is in a different resource group
					if extractedScope := azureshared.ExtractScopeFromResourceID(*extension.Properties.ProtectedSettingsFromKeyVault.SourceVault.ID); extractedScope != "" {
						scope = extractedScope
					}
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   azureshared.KeyVaultVault.String(),
							Method: sdp.QueryMethod_GET,
							Query:  vaultName,
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If Key Vault changes → VMSS extension settings access changes (In: true)
							Out: false, // If VMSS is deleted → Key Vault remains (Out: false)
						},
					})
				}
			}
		}
	}

	// Map provisioning state to health status
	if scaleSet.Properties != nil && scaleSet.Properties.ProvisioningState != nil {
		switch *scaleSet.Properties.ProvisioningState {
		case "Succeeded":
			sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
		case "Creating", "Updating", "Migrating":
			sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
		case "Failed", "Deleting":
			sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
		}
	}

	return sdpItem, nil
}

func (c computeVirtualMachineScaleSetWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeVirtualMachineScaleSetLookupByName,
	}
}

func (c computeVirtualMachineScaleSetWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		// Child resources
		azureshared.ComputeVirtualMachineExtension,
		azureshared.ComputeVirtualMachine,
		// Network resources
		azureshared.NetworkVirtualNetwork,
		azureshared.NetworkSubnet,
		azureshared.NetworkPublicIPPrefix,
		azureshared.NetworkNetworkSecurityGroup,
		azureshared.NetworkLoadBalancer,
		azureshared.NetworkLoadBalancerBackendAddressPool,
		azureshared.NetworkLoadBalancerInboundNatPool,
		azureshared.NetworkLoadBalancerProbe,
		azureshared.NetworkApplicationGateway,
		azureshared.NetworkApplicationGatewayBackendAddressPool,
		azureshared.NetworkApplicationSecurityGroup,
		// Compute resources
		azureshared.ComputeDiskEncryptionSet,
		azureshared.ComputeProximityPlacementGroup,
		azureshared.ComputeDedicatedHostGroup,
		azureshared.ComputeCapacityReservationGroup,
		azureshared.ComputeImage,
		azureshared.ComputeSharedGalleryImage,
		azureshared.ComputeCommunityGalleryImage,
		azureshared.ComputeSharedGalleryApplicationVersion,
		// Storage resources
		azureshared.StorageAccount,
		// Identity resources
		azureshared.ManagedIdentityUserAssignedIdentity,
		// Key Vault resources
		azureshared.KeyVaultVault,
		// Standard library types
		stdlib.NetworkDNS,
	)
}

// ref: https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/virtual_machine_scale_set
func (c computeVirtualMachineScaleSetWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_virtual_machine_scale_set.name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_linux_virtual_machine_scale_set.name",
		},
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_windows_virtual_machine_scale_set.name",
		},
	}
}
