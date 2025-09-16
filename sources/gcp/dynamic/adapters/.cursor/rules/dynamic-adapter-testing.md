# Dynamic Adapter Unit Testing Rules
## Overview

When writing unit tests for GCP dynamic adapters in the Overmind codebase, follow these patterns and requirements to ensure consistency and correctness.
## Package and Imports
- **Package**: Always use `package adapters_test` (never `package adapters` or `package main`)
- **Required Imports**:
  ```go
  import (
      "context"
      "fmt"
      "net/http"
      "testing"

      "cloud.google.com/go/compute/apiv1/computepb" // or relevant protobuf package

      "github.com/overmindtech/cli/discovery"
      "github.com/overmindtech/cli/sdp-go"
      "github.com/overmindtech/cli/sources/gcp/dynamic"
      gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
      "github.com/overmindtech/cli/sources/shared"
      "github.com/overmindtech/cli/sources/stdlib"
  )
  ```
## Protobuf Types
- **Verify Types**: Check available types with `go doc cloud.google.com/go/compute/apiv1/computepb | grep -i "type.*YourResource"`
- **Common Mistake**: Use `computepb.Address` and `computepb.AddressList` (NOT `GlobalAddress`)
- **Always verify**: Don't assume protobuf types exist - check the actual API
## Test Structure Template

### Single Query Parameter Resources
```go
func TestYourResource(t *testing.T) {
    ctx := context.Background()
    projectID := "test-project"
    linker := gcpshared.NewLinker()
    resourceName := "test-resource"

    // Create mock protobuf object
    resource := &computepb.YourResource{
        Name: &resourceName,
        // ... other fields using pointer helpers
    }

    // Create second resource for list testing
    resourceName2 := "test-resource-2"
    resource2 := &computepb.YourResource{
        Name: &resourceName2,
        // ... other fields using pointer helpers
    }

    // Create list response with multiple items
    resourceList := &computepb.YourResourceList{
        Items: []*computepb.YourResource{resource, resource2},
    }

    sdpItemType := gcpshared.YourItemType

    // Mock HTTP responses
    expectedCallAndResponses := map[string]shared.MockResponse{
        fmt.Sprintf("https://api.googleapis.com/v1/projects/%s/resources/%s", projectID, resourceName): {
            StatusCode: http.StatusOK,
            Body:       resource,
        },
        fmt.Sprintf("https://api.googleapis.com/v1/projects/%s/resources/%s", projectID, resourceName2): {
            StatusCode: http.StatusOK,
            Body:       resource2,
        },
        fmt.Sprintf("https://api.googleapis.com/v1/projects/%s/resources", projectID): {
            StatusCode: http.StatusOK,
            Body:       resourceList,
        },
    }

    t.Run("Get", func(t *testing.T) {
        // Test Get functionality
    })

    t.Run("List", func(t *testing.T) {
        // Test List functionality
    })

    t.Run("Search", func(t *testing.T) {
        // Test Search functionality (if supported)
    })
}
```

### Multiple Query Parameter Resources (e.g., location + resource)
```go
func TestLocationBasedResource(t *testing.T) {
    ctx := context.Background()
    projectID := "test-project"
    linker := gcpshared.NewLinker()
    location := "us-central1"
    resourceName := "test-resource"

    // Create mock protobuf object
    resource := &computepb.YourResource{
        Name: &resourceName,
        // ... other fields using pointer helpers
    }

    // Create second resource for list testing
    resourceName2 := "test-resource-2"
    resource2 := &computepb.YourResource{
        Name: &resourceName2,
        // ... other fields using pointer helpers
    }

    // Create list response with multiple items
    resourceList := &computepb.YourResourceList{
        Items: []*computepb.YourResource{resource, resource2},
    }

    sdpItemType := gcpshared.YourItemType

    // Mock HTTP responses for location-based resources
    expectedCallAndResponses := map[string]shared.MockResponse{
        fmt.Sprintf("https://api.googleapis.com/v1/projects/%s/locations/%s/resources/%s", projectID, location, resourceName): {
            StatusCode: http.StatusOK,
            Body:       resource,
        },
        fmt.Sprintf("https://api.googleapis.com/v1/projects/%s/locations/%s/resources/%s", projectID, location, resourceName2): {
            StatusCode: http.StatusOK,
            Body:       resource2,
        },
        fmt.Sprintf("https://api.googleapis.com/v1/projects/%s/locations/%s/resources", projectID, location): {
            StatusCode: http.StatusOK,
            Body:       resourceList,
        },
    }

    // Test Get with location + resource name
    t.Run("Get", func(t *testing.T) {
        httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
        adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
        if err != nil {
            t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
        }

        // For multiple query parameters, use the combined query format
        combinedQuery := fmt.Sprintf("%s/%s", location, resourceName)
        sdpItem, err := adapter.Get(ctx, projectID, combinedQuery, true)
        if err != nil {
            t.Fatalf("Failed to get resource: %v", err)
        }

        // Validate SDP item properties
        if sdpItem.GetType() != sdpItemType.String() {
            t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
        }
        if sdpItem.UniqueAttributeValue() != combinedQuery {
            t.Errorf("Expected unique attribute value '%s', got %s", combinedQuery, sdpItem.UniqueAttributeValue())
        }
    })

    // Test Search (location-based resources typically use Search instead of List)
    t.Run("Search", func(t *testing.T) {
        httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
        adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
        if err != nil {
            t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
        }

        searchable, ok := adapter.(discovery.SearchableAdapter)
        if !ok {
            t.Skipf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
        }

        // Test location-based search
        sdpItems, err := searchable.Search(ctx, projectID, location, true)
        if err != nil {
            t.Fatalf("Failed to search resources: %v", err)
        }

        if len(sdpItems) != 2 {
            t.Errorf("Expected 2 resources, got %d", len(sdpItems))
        }
    })
}
```
## Required Test Functions
### 1. Get Test
```go
t.Run("Get", func(t *testing.T) {
    httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
    adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
    if err != nil {
        t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
    }

    sdpItem, err := adapter.Get(ctx, projectID, resourceName, true)
    if err != nil {
        t.Fatalf("Failed to get resource: %v", err)
    }

    // Validate SDP item properties
    if sdpItem.GetType() != sdpItemType.String() {
        t.Errorf("Expected type %s, got %s", sdpItemType.String(), sdpItem.GetType())
    }
    if sdpItem.UniqueAttributeValue() != resourceName {
        t.Errorf("Expected unique attribute value '%s', got %s", resourceName, sdpItem.UniqueAttributeValue())
    }
    if sdpItem.GetScope() != projectID {
        t.Errorf("Expected scope '%s', got %s", projectID, sdpItem.GetScope())
    }

    // Validate specific attributes
    val, err := sdpItem.GetAttributes().Get("name")
    if err != nil {
        t.Fatalf("Failed to get 'name' attribute: %v", err)
    }
    if val != resourceName {
        t.Errorf("Expected name field to be '%s', got %s", resourceName, val)
    }

    // Include static tests - MUST cover ALL blast propagation links
    t.Run("StaticTests", func(t *testing.T) {
        // CRITICAL: Review the adapter's blast propagation configuration and to create
        // test cases for EVERY linked resource defined in the adapter's blastPropagation map
        // Check the adapter file (e.g., compute-global-address.go) for all blast propagation entries
        queryTests := shared.QueryTests{
            {
                ExpectedType:   gcpshared.LinkedResourceType.String(),
                ExpectedMethod: sdp.QueryMethod_GET,
                ExpectedQuery:  "linked-resource-name",
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
```
### 2. List Test
```go
t.Run("List", func(t *testing.T) {
    httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
    adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
    if err != nil {
        t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
    }

    listable, ok := adapter.(discovery.ListableAdapter)
    if !ok {
        t.Fatalf("Adapter is not a ListableAdapter")
    }

    sdpItems, err := listable.List(ctx, projectID, true)
    if err != nil {
        t.Fatalf("Failed to list resources: %v", err)
    }

    if len(sdpItems) != 2 {
        t.Errorf("Expected 2 resources, got %d", len(sdpItems))
    }
})
```
### 3. Search Test (if supported)
```go
t.Run("Search", func(t *testing.T) {
    httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
    adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
    if err != nil {
        t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
    }

    searchable, ok := adapter.(discovery.SearchableAdapter)
    if !ok {
        t.Skipf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
    }

    // Test different search query types
    searchQueries := []string{
        "us-central1",                    // Location-based search
        "projects/test-project/locations/us-central1/resources", // Full resource name
        "test-resource",                  // Resource name search
    }

    for _, searchQuery := range searchQueries {
        t.Run(fmt.Sprintf("Search_%s", searchQuery), func(t *testing.T) {
            sdpItems, err := searchable.Search(ctx, projectID, searchQuery, true)
            if err != nil {
                t.Fatalf("Failed to search resources with query '%s': %v", searchQuery, err)
            }

            if len(sdpItems) != 2 {
                t.Errorf("Expected 2 resources for query '%s', got %d", searchQuery, len(sdpItems))
            }
        })
    }
})
```
## Pointer Helper Functions
Always define local helper functions for creating pointers:
```go
func stringPtr(s string) *string {
    return &s
}

func uint64Ptr(u uint64) *uint64 {
    return &u
}
```
## Common Mistakes to Avoid
1. **Wrong Package**: Don't use `package main` or `package adapters`
2. **Wrong Protobuf Types**: Check actual available types, don't assume `GlobalAddress` exists
3. **Missing Pointer Helpers**: Always define local pointer helper functions
4. **Incorrect HTTP URLs**: Match the exact API endpoint format from the adapter metadata
5. **Missing Static Tests**: Always include blast propagation tests for linked resources
6. **Missing Search Tests**: Include Search tests only if the adapter implements `SearchableAdapter` - use `t.Skipf()` if not supported


## Blast Propagation Testing Requirements

**CRITICAL**: Every adapter test SHOULD include comprehensive StaticTests that cover ALL blast propagation links defined in the adapter configuration.

### Steps to Ensure Complete Coverage:
1. **Review Adapter File**: Open the corresponding adapter file (e.g., `compute-global-address.go`) 
2. **Find blastPropagation Map**: Locate the `blastPropagation` field in the adapter configuration
3. **Create Test Cases**: Write a QueryTest for EVERY entry in the blastPropagation map
4. **Handle TODOs**: Note any blast propagation entries marked with TODO comments - these may not work yet but should be documented in test comments

### Example Complete StaticTests:
```go
t.Run("StaticTests", func(t *testing.T) {
    queryTests := shared.QueryTests{
        // Network link
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
        // IP address link
        {
            ExpectedType:   "ip",
            ExpectedMethod: sdp.QueryMethod_GET,
            ExpectedQuery:  "203.0.113.1",
            ExpectedScope:  "global",
            ExpectedBlastPropagation: &sdp.BlastPropagation{
                In:  true,
                Out: true,
            },
        },
        // Backend service link
        {
            ExpectedType:   gcpshared.ComputeBackendService.String(),
            ExpectedMethod: sdp.QueryMethod_GET,
            ExpectedQuery:  "test-backend-service",
            ExpectedScope:  projectID,
            ExpectedBlastPropagation: &sdp.BlastPropagation{
                In:  true,
                Out: true,
            },
        },
        // Subnetwork link (note special scope format)
        {
            ExpectedType:   gcpshared.ComputeSubnetwork.String(),
            ExpectedMethod: sdp.QueryMethod_GET,
            ExpectedQuery:  "test-subnet",
            ExpectedScope:  fmt.Sprintf("%s.us-central1", projectID),
            ExpectedBlastPropagation: &sdp.BlastPropagation{
                In:  true,
                Out: false,
            },
        },
    }
    shared.RunStaticTests(t, adapter, sdpItem, queryTests)
})
```

## Validation Checklist
- [ ] Package is `adapters_test`
- [ ] All required imports are present
- [ ] Protobuf types are correct and available
- [ ] Mock HTTP responses match actual API endpoints
- [ ] Get test validates all SDP item properties
- [ ] List test validates item count (expect 2+ items) and properties
- [ ] Search test validates item count (expect 2+ items) and properties (if adapter supports Search)
- [ ] Multiple query parameter resources test combined query format (e.g., "location/resource")
- [ ] **CRITICAL**: Static tests include blast propagation for ALL linked resources in the adapter's blastPropagation map
- [ ] Static test queries use correct scope formats (especially for subnetworks: "projectID.region")
- [ ] Static test queries use correct query formats (especially for KMS keys: use `shared.CompositeLookupKey()`)
- [ ] Pointer helper functions are defined locally
- [ ] Test compiles without errors
- [ ] Test runs successfully
## Key Patterns
- Use `dynamic.MakeAdapter()` to create adapters
- Always validate SDP item type, scope, and unique attribute
- Include comprehensive attribute validation in Get tests
- Use `t.Skipf()` for optional functionality (like Search)
- Always include static tests with blast propagation
- Mock HTTP responses must match actual API endpoints exactly
