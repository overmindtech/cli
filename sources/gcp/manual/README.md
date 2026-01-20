# GCP Manual Adapters

This directory contains manually implemented GCP adapters that cannot be generated using the dynamic adapter framework due to their complex API response patterns or resource relationships.

## When to Use Manual Adapters

**Prefer Dynamic Adapters**: Always use the [dynamic adapter framework](../../dynamic/adapters/README.md) when possible. Dynamic adapters are automatically generated from GCP API specifications and are easier to maintain.

**Create Manual Adapters Only When**:

1. **Non-standard API Response Format**: The GCP API response doesn't follow the general pattern where resource names or attributes reference different types of resources that require manual handling for linked item queries.

2. **Complex Resource Relationships**: The adapter needs to manually parse and link to multiple different resource types based on the API response content.

## Examples of Manual Adapter Use Cases

### Non-standard API Response Format

**BigQuery Dataset** (`big-query-dataset.go`):
- Uses dot notation for resource references (`projectID:datasetID`)
- Requires manual parsing of the `FullID` field to extract dataset ID
- Complex access control parsing with multiple entity types

**BigQuery Table** (`big-query-table.go`):
- Uses dot notation for composite keys (`projectID:datasetID.tableID`)
- Requires manual parsing and splitting of the `FullID` field
- Multiple connection ID formats need manual parsing (`projectId.locationId;connectionId` vs `projects/projectId/locations/locationId/connections/connectionId`)

### Attributes Referencing Different Resource Types

**Logging Sink** (`logging-sink.go`):
- The `destination` field can reference multiple different resource types:
  - Storage buckets: `storage.googleapis.com/[BUCKET]`
  - BigQuery datasets: `bigquery.googleapis.com/projects/[PROJECT]/datasets/[DATASET]`
  - Pub/Sub topics: `pubsub.googleapis.com/projects/[PROJECT]/topics/[TOPIC]`
  - Logging buckets: `logging.googleapis.com/projects/[PROJECT]/locations/[LOCATION]/buckets/[BUCKET]`
- Requires manual parsing and conditional linking based on the destination format

## Implementation Guidelines

### For Detailed Implementation Rules
Refer to the [cursor rules](.cursor/rules/gcp-manual-adapter-creation.mdc) for comprehensive implementation patterns, examples, and best practices.

### Key Implementation Requirements

1. **Follow Naming Conventions**:
   - File names: `{api}-{resource}.go` (e.g., `compute-subnetwork.go`, `bigquery-table.go`, `logging-sink.go`)
   - Struct names: `{resourceName}Wrapper` (e.g., `computeSubnetworkWrapper`, `bigQueryTableWrapper`)
   - Constructor: `New{ResourceName}` (e.g., `NewComputeSubnetwork`, `NewBigQueryTable`)

2. **Implement Required Methods**:
   - `IAMPermissions()` - List specific GCP API permissions
   - `PredefinedRole()` - Most restrictive GCP predefined role
   - `PotentialLinks()` - All possible linked resource types
   - `TerraformMappings()` - Terraform registry mappings
   - `GetLookups()` / `SearchLookups()` - Query parameter definitions

3. **Handle Complex Resource Linking**:
   - Parse non-standard API response formats
   - Extract resource identifiers from various formats
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
- [ ] Example values in tests match actual GCP resource formats
- [ ] Scopes for linked item queries are correct (verify with linked resource documentation)
- [ ] Blast propagation rules are appropriate for resource relationships
- [ ] All possible resource references are handled (no missing cases)

### ✅ Documentation and References
- [ ] GCP API documentation URLs are included in comments
- [ ] Resource relationship explanations are documented
- [ ] Complex parsing logic is well-commented
- [ ] Official GCP reference links are provided for linked resources

### ✅ Error Handling
- [ ] Proper error wrapping with `gcpshared.QueryError`
- [ ] Input validation for parsed values
- [ ] Graceful handling of malformed API responses

## Testing Examples

### Static Tests for Linked Item Queries
```go
t.Run("StaticTests", func(t *testing.T) {
    queryTests := shared.QueryTests{
        {
            ExpectedType:   gcpshared.BigQueryDataset.String(),
            ExpectedMethod: sdp.QueryMethod_GET,
            ExpectedQuery:  "test-dataset",
            ExpectedScope:  "test-project-id",
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
mockClient := mocks.NewMockBigQueryTableClient(ctrl)
mockClient.EXPECT().Get(ctx, projectID, datasetID, tableID).Return(
    createTableMetadata(projectID, datasetID, tableID, connectionID), nil)
```

## Common Patterns

### Parsing Composite IDs
```go
// BigQuery format: projectID:datasetID.tableID
parts := strings.Split(strings.TrimPrefix(metadata.FullID, b.ProjectID()+":"), ".")
if len(parts) != 2 {
    return nil, gcpshared.QueryError(fmt.Errorf("invalid table full ID: %s", metadata.FullID), scope, b.Type())
}
```

### Conditional Resource Linking
```go
if sink.GetDestination() != "" {
    switch {
    case strings.HasPrefix(sink.GetDestination(), "storage.googleapis.com"):
        // Handle storage bucket linking
    case strings.HasPrefix(sink.GetDestination(), "bigquery.googleapis.com"):
        // Handle BigQuery dataset linking
    // ... more cases
    }
}
```

### Path Parameter Extraction
```go
values := gcpshared.ExtractPathParams(keyName, "locations", "keyRings", "cryptoKeys")
if len(values) == 3 && values[0] != "" && values[1] != "" && values[2] != "" {
    // Use extracted values for linking
}
```

## Getting Help

- **Implementation Details**: See [cursor rules](.cursor/rules/gcp-manual-adapter-creation.mdc)
- **Dynamic Adapters**: See [dynamic adapter README](../../dynamic/adapters/README.md)
- **General Source Adapters**: See [sources README](../../README.md)
- **GCP API Documentation**: Always reference official GCP documentation for API specifics

## Related Files

- **Cursor Rules**: `.cursor/rules/gcp-manual-adapter-creation.mdc` - Comprehensive implementation guide
- **Shared Utilities**: `../../shared/` - Common utilities and patterns
- **GCP Shared**: `../shared/` - GCP-specific utilities and base structs
- **Test Utilities**: `../../shared/testing.go` - Testing helpers and patterns
