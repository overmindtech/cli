# Azure Source Adapters

## Scope Design

Azure adapters use a two-level scope hierarchy based on how Azure SDK clients uniquely identify resources:

### Scope Levels

1. **Subscription ID** (highest level)
   - The primary container for Azure resources
   - Provides billing and access control boundary
   - Required to initialize all Azure SDK clients

2. **Resource Group**
   - Logical container within a subscription
   - Resources are uniquely identified by: `subscription + resourceGroup + resourceName`
   - Resource names must be unique within a resource group

Resource name is unique within the resource group. Resources in the same group can't share the same name, but identical names can exist in different resource groups.
For example, a virtual network named `vnet-prod-westus-001` can exist in multiple resource groups, but only once within a single resource group.

## Naming Convention for Adapters

Azure adapter names follow a structured pattern derived from the official Azure REST API documentation:

### Pattern: `azure-{api}-{resource}`

1. **Source**: Always `azure`
2. **API**: Derived from the second part of the REST API provider namespace
3. **Resource**: Singular form of the resource type from the REST API path

### How to Determine the Naming

**Step 1: Find the REST API Documentation**

Locate the official Azure REST API documentation for the resource. It can be searched from this reference:
https://learn.microsoft.com/en-us/rest/api/compute/operation-groups?view=rest-compute-2025-04-01

**Step 2: Extract the API Name**

From the REST API path, identify the resource provider namespace. The API name is derived from the second part after `Microsoft.`:

```
REST API Path: /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/virtualMachines/{vmName}
                                                                                          ^^^^^^^^
Provider Namespace: Microsoft.Compute
API Name: compute (lowercase, singular)
```

**Step 3: Extract the Resource Name**

The resource name comes from the resource type in the REST API path, converted to singular form and kebab-case:

```
REST API Path: /providers/Microsoft.Compute/virtualMachines/{vmName}
                                            ^^^^^^^^^^^^^^
Resource Type: virtualMachines
Resource Name: virtual-machine (singular, kebab-case)
```

### Examples

**Virtual Machine:**
- REST API: https://learn.microsoft.com/en-us/rest/api/compute/virtual-machines/get
- Provider: `Microsoft.Compute`
- Resource Type: `virtualMachines`
- **Item Type**: `azure-compute-virtual-machine`

**Virtual Network:**
- REST API: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/virtual-networks/get
- Provider: `Microsoft.Network`
- Resource Type: `virtualNetworks`
- **Item Type**: `azure-network-virtual-network`

### Special Cases

- If the API name and resource name would result in stuttering (e.g., `storage-storage`), keep both parts for clarity
- Always use singular form for the resource name (e.g., `virtual-machine` not `virtual-machines`)
- Convert PascalCase to kebab-case (e.g., `virtualMachines` → `virtual-machine`)
- Keep compound words together (e.g., `publicIPAddress` → `public-ip-address`)

## Code Structure

### Models (`shared/models.go`)

Defines constants for building item types:

- **Source**: `Azure` - identifies the cloud provider
- **API**: Resource provider namespaces (e.g., `Compute` → `Microsoft.Compute`)
- **Resource**: Specific resource types within each provider (e.g., `VirtualMachine`, `Disk`)


### Item Types (`shared/item-types.go`)

Pre-defined item type variables combining source + API + resource:

```go
// Example: azure-compute-virtual-machine
ComputeVirtualMachine = shared.NewItemType(Azure, Compute, VirtualMachine)

// Example: azure-network-virtual-network
NetworkVirtualNetwork = shared.NewItemType(Azure, Network, VirtualNetwork)
```

### Base Structs (`shared/base.go`)

Provides foundation structs for adapters based on scope:

- **`ResourceGroupBase`**: For resources scoped to a resource group (most common)
- **`SubscriptionBase`**: For subscription-level resources
- **`AzureBase`**: Common base providing `SubscriptionID()` method

## Official References

- [Azure REST API Reference](https://learn.microsoft.com/en-us/rest/api/azure/)
- [Azure Resource Providers and Types](https://learn.microsoft.com/en-us/azure/azure-resource-manager/management/resource-providers-and-types)
- [Azure Resource Naming Guidelines](https://learn.microsoft.com/azure/cloud-adoption-framework/ready/azure-best-practices/resource-naming-and-tagging-decision-guide)
- [Azure Resource Name Rules](https://learn.microsoft.com/en-us/azure/azure-resource-manager/management/resource-name-rules)