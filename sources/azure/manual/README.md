# Azure Manual Adapters

This directory contains manually implemented Azure adapters that cannot be generated using the dynamic adapter framework due to their complex API response patterns or resource relationships.

## When to Use Manual Adapters

**Prefer Dynamic Adapters**: Always use the [dynamic adapter framework](../../dynamic/adapters/README.md) when possible. Dynamic adapters can leverage the [Azure Resource List API](https://learn.microsoft.com/en-us/rest/api/resources/resources/list?view=rest-resources-2021-04-01) which lists all resources in a subscription, similar to how GCP dynamic adapters work. This makes dynamic adapters easier to maintain and automatically generated from Azure API specifications.

**Create Manual Adapters Only When**:

1. **Non-standard API Response Format**: The Azure API response doesn't follow the general pattern where resource names or attributes reference different types of resources that require manual handling for linked item queries.

2. **Complex Resource Relationships**: The adapter needs to manually parse and link to multiple different resource types based on the API response content.

## Examples of Manual Adapter Use Cases

### Non-standard API Response Format

**Compute Virtual Machine** (`compute-virtual-machine.go`):
- Complex resource ID parsing from Azure resource manager format
- Requires manual extraction of resource names from full resource IDs (`/subscriptions/{subscription}/resourceGroups/{resourceGroup}/providers/Microsoft.Compute/virtualMachines/{vmName}`)
- Multiple disk and network interface references need manual parsing

### Attributes Referencing Different Resource Types

**Virtual Machine with Multiple Linked Resources**:
- The `Properties` field contains references to multiple different resource types:
  - Managed Disks: `/subscriptions/{subscription}/resourceGroups/{resourceGroup}/providers/Microsoft.Compute/disks/{diskName}`
  - Network Interfaces: `/subscriptions/{subscription}/resourceGroups/{resourceGroup}/providers/Microsoft.Network/networkInterfaces/{nicName}`
  - Availability Sets: `/subscriptions/{subscription}/resourceGroups/{resourceGroup}/providers/Microsoft.Compute/availabilitySets/{availabilitySetName}`
  - Public IP Addresses: Referenced through network interfaces
  - Network Security Groups: Referenced through network interfaces
- Requires manual parsing and conditional linking based on the resource ID format and provider namespace

## Implementation Guidelines

### For Detailed Implementation Rules
Refer to the [cursor rules](.cursor/rules/azure-manual-adapter-creation.mdc) for comprehensive implementation patterns, examples, and best practices.

### Key Implementation Requirements

1. **Follow Naming Conventions**:
   - File names: `{api}-{resource}.go` (e.g., `compute-virtual-machine.go`, `network-virtual-network.go`)
   - Struct names: `{resourceName}Wrapper` (e.g., `computeVirtualMachineWrapper`, `networkVirtualNetworkWrapper`)
   - Constructor: `New{ResourceName}` (e.g., `NewComputeVirtualMachine`, `NewNetworkVirtualNetwork`)

2. **Implement Required Methods**:
   - `IAMPermissions()` - List specific Azure RBAC permissions (e.g., `Microsoft.Compute/virtualMachines/read`)
   - `PredefinedRole()` - Most restrictive Azure built-in role (e.g., `Reader`, `Virtual Machine Contributor`)
   - `PotentialLinks()` - All possible linked resource types
   - `TerraformMappings()` - Terraform registry mappings (using `azurerm_` provider)
   - `GetLookups()` / `SearchLookups()` - Query parameter definitions

3. **Handle Complex Resource Linking**:
   - Parse Azure resource IDs to extract resource names and types
   - Extract resource identifiers from Azure resource manager format
   - Create appropriate linked item queries with correct blast propagation

4. **Include Comprehensive Tests**:
   - Unit tests for all methods
   - Static tests for linked item queries
   - Mock-based testing with gomock
   - Interface compliance tests

## Code Review Checklist

When reviewing PRs for manual adapters, ensure:

### ✅ Fundamentals Coverage
- [ ] Unit tests cover all adapter methods (Get, List, Search if applicable)
- [ ] Static tests validate linked item queries using `shared.RunStaticTests`
- [ ] Mock expectations are properly set up with gomock
- [ ] Interface compliance is tested (ListableWrapper, SearchableWrapper, etc.)

### ✅ Terraform Integration
- [ ] Terraform mappings reference official Terraform registry URLs
- [ ] Terraform method (GET vs SEARCH) matches adapter capabilities
- [ ] Terraform query map uses correct resource attribute names

### ✅ Naming and Structure
- [ ] File name follows `{api}-{resource}.go` convention (e.g., `compute-subnetwork.go`)
- [ ] Struct and function names follow Go conventions
- [ ] Package imports are properly organized

### ✅ Linked Item Queries
- [ ] Example values in tests match actual Azure resource formats
- [ ] Scopes for linked item queries are correct (verify with linked resource documentation)
- [ ] Blast propagation rules are appropriate for resource relationships
- [ ] All possible resource references are handled (no missing cases)

### ✅ Documentation and References
- [ ] Azure REST API documentation URLs are included in comments
- [ ] Resource relationship explanations are documented
- [ ] Complex parsing logic is well-commented
- [ ] Official Azure reference links are provided for linked resources

### ✅ Error Handling
- [ ] Proper error wrapping with `azureshared.QueryError`
- [ ] Input validation for parsed values
- [ ] Graceful handling of malformed API responses

## Testing Examples

### Static Tests for Linked Item Queries
```go
t.Run("StaticTests", func(t *testing.T) {
    queryTests := shared.QueryTests{
        {
            ExpectedType:   azureshared.ComputeDisk.String(),
            ExpectedMethod: sdp.QueryMethod_GET,
            ExpectedQuery:  "test-disk",
            ExpectedScope:  fmt.Sprintf("%s.%s", subscriptionID, resourceGroup),
            ExpectedBlastPropagation: &sdp.BlastPropagation{
                In:  true,
                Out: true,
            },
        },
        // ... more test cases
    }
    shared.RunStaticTests(t, adapter, sdpItem, queryTests)
})
```

### Mock Setup for Complex APIs
```go
mockClient := mocks.NewMockVirtualMachinesClient(ctrl)
vm := createAzureVirtualMachine("test-vm", "Succeeded")
mockClient.EXPECT().Get(ctx, resourceGroup, vmName, nil).Return(
    armcompute.VirtualMachinesClientGetResponse{VirtualMachine: *vm}, nil)
```

## Common Patterns

### Parsing Azure Resource IDs
```go
// Azure resource ID format: /subscriptions/{subscription}/resourceGroups/{resourceGroup}/providers/Microsoft.Compute/disks/{diskName}
diskName := azureshared.ExtractResourceName(*vm.Properties.StorageProfile.OSDisk.ManagedDisk.ID)
if diskName == "" {
    return nil, azureshared.QueryError(fmt.Errorf("invalid disk resource ID: %s", *vm.Properties.StorageProfile.OSDisk.ManagedDisk.ID), c.DefaultScope(), c.Type())
}
```

### Conditional Resource Linking
```go
if vm.Properties.NetworkProfile != nil && len(vm.Properties.NetworkProfile.NetworkInterfaces) > 0 {
    for _, nicRef := range vm.Properties.NetworkProfile.NetworkInterfaces {
        if nicRef.ID != nil {
            nicName := azureshared.ExtractResourceName(*nicRef.ID)
            // Determine resource type from provider namespace in ID
            if strings.Contains(*nicRef.ID, "Microsoft.Network/networkInterfaces") {
                // Handle network interface linking
            }
        }
    }
}
```

### Resource ID Extraction
```go
// Extract resource name from Azure resource ID
// ID: /subscriptions/{subscription}/resourceGroups/{resourceGroup}/providers/Microsoft.Compute/virtualMachines/{vmName}
resourceName := azureshared.ExtractResourceName(resourceID)
if resourceName != "" {
    // Use extracted resource name for linking
}
```

## Getting Help

- **Implementation Details**: See [cursor rules](.cursor/rules/azure-manual-adapter-creation.mdc)
- **Dynamic Adapters**: See [dynamic adapter README](../../dynamic/adapters/README.md) - Note: Azure dynamic adapters can leverage the [Azure Resource List API](https://learn.microsoft.com/en-us/rest/api/resources/resources/list?view=rest-resources-2021-04-01) to list all resources in a subscription
- **General Source Adapters**: See [sources README](../../README.md)
- **Azure API Documentation**: Always reference official Azure REST API documentation for API specifics

## Related Files

- **Cursor Rules**: `.cursor/rules/azure-manual-adapter-creation.mdc` - Comprehensive implementation guide
- **Shared Utilities**: `../../shared/` - Common utilities and patterns
- **Azure Shared**: `../shared/` - Azure-specific utilities and base structs
- **Test Utilities**: `../../shared/testing.go` - Testing helpers and patterns
