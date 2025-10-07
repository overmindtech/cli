# GCP Dynamic Adapter Framework

The GCP Dynamic Adapter Framework is a powerful system for automatically generating GCP resource adapters by making simple HTTP requests to GCP APIs instead of using versioned SDKs. This framework eliminates the need to manually implement GET/SEARCH/LIST methods and handles all the complex wiring, validation, and error handling automatically.

## What is a Dynamic Adapter?

Instead of using versioned SDKs for GCP, we make simple HTTP requests and generate resource adapters dynamically. The framework provides several key advantages:

- **No Manual Method Implementation**: Instead of creating GET/SEARCH/LIST methods manually, we define only the endpoints in the adapter type definition
- **Automatic Link Detection**: We identify the linked items, but the framework handles all the wiring depending on the adapter metadata
- **Centralized Framework Logic**: All adapter metadata, query validations, error handling, iterations, and caching are handled by the framework
- **AI-Assisted Development**: With Cursor instructions, all we need to do is provide links for the resource type definition and GET endpoint. Cursor does a good job generating the code, but the output should be thoroughly inspected. The author should not allow good-looking verbose unnecessary code since every line of code is a liability. Focus on concise, essential implementations and comprehensive test coverage

## Why Dynamic?

We don't use fixed SDKs. We always use the dynamic API response. With comprehensive logging in place, we can identify potential links even after creating adapters, which was not possible before. We do this by checking the structure of an attribute - if it looks like a resource name but we don't have a link for it, then we log it as a potential adapter.

This approach provides several benefits:

- **Future-Proof**: No dependency on SDK versions that may change
- **Consistent**: All adapters follow the same patterns and behaviors
- **Discoverable**: Automatic detection of new potential links from API responses
- **Maintainable**: Centralized logic means updates apply to all adapters

## Resource Requirements

For a resource to be compatible with the dynamic adapter framework, it should follow standard naming conventions and API response types. See BigQuery as an example of a non-standard adapter that required a manual implementation due to its unique API response format and naming conventions.

**Standard Requirements:**
- Consistent resource naming in API responses
- Standard REST API patterns (GET, LIST endpoints)
- Predictable response structures
- Standard GCP resource URL patterns

**Non-Standard Examples (Require Manual Adapters):**
- BigQuery resources with composite IDs (`projectID:datasetID.tableID`)
- Resources with attributes referencing multiple resource types
- APIs with non-standard response formats

## Linker: How It Works

The linker is a critical component that finds the adapter metadata for linked items and creates linked item queries by their definition. This standardizes how a certain adapter is linked across the entire source and prevents code duplication.

**Key Benefits:**
- **Standardization**: Ensures consistent linking patterns across all adapters
- **Centralized Updates**: If a linked item adapter changes, the update applies to all existing adapters automatically
- **No Find/Replace**: Eliminates the need to manually update multiple files when linked item logic changes
- **Manual Adapter Compatibility**: It's possible to link manual adapters to dynamic adapters seamlessly

## Flow: GET Request to SDP Adapter

The complete flow from making a GET request to creating an SDP adapter follows these steps:

1. **Adapter Definition**: Define the adapter metadata in the adapter file (see [dynamic-adapter-creation.mdc](adapters/.cursor/rules/dynamic-adapter-creation.mdc))
2. **Adapter Creation**: Framework creates the appropriate adapter type based on metadata configuration
3. **GET Request Processing**: Validate scope, check cache, construct URL, make HTTP request, convert to SDP item
4. **External Response to SDP Conversion**: Extract attributes, apply blast propagation rules, generate linked item queries
5. **Unit Test Coverage**: Test GET functionality and static tests for blast propagation

For detailed implementation patterns and code examples, refer to the [dynamic adapter creation rules](adapters/.cursor/rules/dynamic-adapter-creation.mdc).

## AI Tools Available

We have helper scripts that benefit from Linear and Cursor integration to streamline adapter development:

### Generate Adapter Ticket
```bash
# Generate implementation ticket for new adapter
go run ai-tools/generate-adapter-ticket-cmd/main.go -name compute-subnetwork -api-ref "https://cloud.google.com/compute/docs/reference/rest/v1/subnetworks/get"
```

### Generate Test Ticket
```bash
# Generate test ticket for existing adapter
go run ai-tools/generate-test-ticket-cmd/main.go compute-global-address
```

**Benefits:**
- **Automated Ticket Creation**: Generates Linear tickets with proper context and requirements
- **Cursor Integration**: Works seamlessly with Cursor rules for consistent implementation
- **Comprehensive Context**: Includes API references, implementation checklists, and testing requirements

For detailed usage instructions, see the [AI Tools README](ai-tools/README.md).

## Cursor Integration

It is highly recommended to use Cursor for creating adapters. There are comprehensive rules available that guide the implementation process. After creating an adapter, the author MUST perform the following checks:

### Adapter Validation

1. **Terraform Mappings GET/Search**: Check from Terraform registry that the mappings are correct
2. **Blast Propagations**: Verify they are comprehensive and attribute values follow standards
3. **Item Selector**: If the item identifier in the API response is something other than `name`, define it properly
4. **Unique Attribute Keys**: Investigate the GET endpoint format and ensure it's correct

### Test Completeness

1. **Blast Propagation/Linked Item Queries**: Verify they work as expected
2. **Unique Attribute**: Ensure it matches the GET call response
3. **Terraform Mapping for Search**: Confirm it exists if search is supported

## Post-Implementation Steps

After adding a new adapter, follow the comprehensive post-implementation checklist in the [main adapter documentation](../README.md#post-implementation-steps). This includes updating documentation, IAM permissions, and enabling required APIs.

## Adapter Types

The framework supports four types of adapters based on their capabilities:

- **Standard**: GET only
- **Listable**: GET + LIST
- **Searchable**: GET + SEARCH
- **SearchableListable**: GET + LIST + SEARCH

The adapter type is automatically determined based on the metadata configuration:

```go
func adapterType(meta gcpshared.AdapterMeta) typeOfAdapter {
    if meta.ListEndpointFunc != nil && meta.SearchEndpointFunc == nil {
        return Listable
    }
    if meta.SearchEndpointFunc != nil && meta.ListEndpointFunc == nil {
        return Searchable
    }
    if meta.ListEndpointFunc != nil && meta.SearchEndpointFunc != nil {
        return SearchableListable
    }
    return Standard
}
```

## Benefits of Dynamic Adapters

1. **Consistency**: All adapters follow the same patterns and behaviors
2. **Efficiency**: Reduces boilerplate code and speeds up development
3. **Maintainability**: Centralized logic makes updates and bug fixes easier
4. **Scalability**: Simplifies the process of adding new resources
5. **Quality**: Automatic validation and error handling ensure reliability
6. **Discoverability**: Automatic detection of potential new links from API responses

## Getting Started

1. **Use AI Tools**: Generate tickets using the helper scripts
2. **Follow Cursor Rules**: Apply the comprehensive rules for consistent implementation
3. **Review Thoroughly**: Check all validation points before considering complete
4. **Update Documentation**: Ensure all related documentation is updated
5. **Test Extensively**: Verify all functionality works as expected

The dynamic adapter framework represents a significant advancement in how we handle GCP resource discovery, providing a robust, scalable, and maintainable solution for infrastructure mapping.
