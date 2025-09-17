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
- **CRITICAL: Match API Version**: Always use the same protobuf API version as specified in the adapter's Get function endpoint
  - **Step 1**: Check the adapter file (e.g., `cloudfunctions-function.go`) for the API endpoint URL
  - **Step 2**: Extract the API version from the URL (e.g., `/v2/` means use `apiv2`)
  - **Step 3**: Import the matching protobuf package (e.g., `cloud.google.com/go/functions/apiv2/functionspb`)
  - **Example**: If adapter uses `https://cloudfunctions.googleapis.com/v2/...`, use `apiv2/functionspb`, NOT `apiv1` or `apiv2beta`
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
        // Test Get functionality with StaticTests
    })

    t.Run("Search", func(t *testing.T) {
        // Test Search functionality (for location-based resources)
        // OR use List for project-level resources
    })

    t.Run("ErrorHandling", func(t *testing.T) {
        // Test error responses (e.g., 404 Not Found)
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

    t.Run("ErrorHandling", func(t *testing.T) {
        // Test with error responses to simulate API errors
        errorResponses := map[string]shared.MockResponse{
            fmt.Sprintf("https://api.googleapis.com/v1/projects/%s/locations/%s/resources/%s", projectID, location, resourceName): {
                StatusCode: http.StatusNotFound,
                Body:       map[string]interface{}{"error": "Resource not found"},
            },
        }

        httpCli := shared.NewMockHTTPClientProvider(errorResponses)
        adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
        if err != nil {
            t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
        }

        combinedQuery := shared.CompositeLookupKey(location, resourceName)
        _, err = adapter.Get(ctx, projectID, combinedQuery, true)
        if err == nil {
            t.Error("Expected error when getting non-existent resource, but got nil")
        }
    })
}
```
## Required Test Functions (Limit to These 3 Only)
### 1. Get Test (MUST include StaticTests)
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
### 2. List or Search Test (Choose based on adapter type)
```go
// For location-based resources (use Search)
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

    sdpItems, err := searchable.Search(ctx, projectID, location, true)
    if err != nil {
        t.Fatalf("Failed to search resources: %v", err)
    }

    if len(sdpItems) != 2 {
        t.Errorf("Expected 2 resources, got %d", len(sdpItems))
    }
})

// OR for project-level resources (use List)
t.Run("List", func(t *testing.T) {
    httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
    adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
    if err != nil {
        t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
    }

    listable, ok := adapter.(discovery.ListableAdapter)
    if !ok {
        t.Skipf("Adapter for %s does not implement ListableAdapter", sdpItemType)
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

### 3. ErrorHandling Test
```go
t.Run("ErrorHandling", func(t *testing.T) {
    // Test with error responses to simulate API errors
    errorResponses := map[string]shared.MockResponse{
        fmt.Sprintf("https://api.googleapis.com/v1/projects/%s/resources/%s", projectID, resourceName): {
            StatusCode: http.StatusNotFound,
            Body:       map[string]interface{}{"error": "Resource not found"},
        },
    }

    httpCli := shared.NewMockHTTPClientProvider(errorResponses)
    adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
    if err != nil {
        t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
    }

    _, err = adapter.Get(ctx, projectID, resourceName, true)
    if err == nil {
        t.Error("Expected error when getting non-existent resource, but got nil")
    }
})
```

### 4. Search with Terraform Format Test (if adapter has terraform mappings with Search method)
```go
t.Run("Search with Terraform format", func(t *testing.T) {
    httpCli := shared.NewMockHTTPClientProvider(expectedCallAndResponses)
    adapter, err := dynamic.MakeAdapter(sdpItemType, linker, httpCli, projectID)
    if err != nil {
        t.Fatalf("Failed to create adapter for %s: %v", sdpItemType, err)
    }

    searchable, ok := adapter.(discovery.SearchableAdapter)
    if !ok {
        t.Skipf("Adapter for %s does not implement SearchableAdapter", sdpItemType)
    }

    // Test Terraform format: projects/[project_id]/locations/[location]/resourceType/[resource_id]
    // The adapter should extract the location from this format and search in that location
    terraformQuery := fmt.Sprintf("projects/%s/locations/%s/resourceType/%s", projectID, location, resourceName)
    sdpItems, err := searchable.Search(ctx, projectID, terraformQuery, true)
    if err != nil {
        t.Fatalf("Failed to search resources with Terraform format: %v", err)
    }

    // The search should return all resources in the location extracted from the Terraform format
    // Verify both items directly without length check
    firstItem := sdpItems[0]
    if firstItem.GetType() != sdpItemType.String() {
        t.Errorf("Expected first item type %s, got %s", sdpItemType.String(), firstItem.GetType())
    }
    if firstItem.GetScope() != projectID {
        t.Errorf("Expected first item scope '%s', got %s", projectID, firstItem.GetScope())
    }

    secondItem := sdpItems[1]
    if secondItem.GetType() != sdpItemType.String() {
        t.Errorf("Expected second item type %s, got %s", sdpItemType.String(), secondItem.GetType())
    }
    if secondItem.GetScope() != projectID {
        t.Errorf("Expected second item scope '%s', got %s", projectID, secondItem.GetScope())
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
- [ ] **CRITICAL**: Protobuf API version matches the adapter's Get function endpoint (e.g., `/v2/` → `apiv2/functionspb`)
- [ ] Protobuf types are correct and available
- [ ] Mock HTTP responses match actual API endpoints
- [ ] Get test validates all SDP item properties
- [ ] List test validates item count (expect 2+ items) and properties
- [ ] Search test validates item count (expect 2+ items) and properties (if adapter supports Search)
- [ ] Search with Terraform format test included (if adapter has terraform mappings with Search method)
- [ ] **ErrorHandling test**: Tests error responses (e.g., 404 Not Found)
- [ ] **LIMIT TO 3/4 TEST CASES ONLY**: Get, List/Search, Search with Terraform, ErrorHandling - no additional tests needed
- [ ] Multiple query parameter resources test combined query format (e.g., "location/resource")
- [ ] **CRITICAL**: Static tests include blast propagation for ALL linked resources in the adapter's blastPropagation map
- [ ] Static test queries use correct scope formats (especially for subnetworks: "projectID.region")
- [ ] Static test queries use correct query formats (especially for KMS keys: use `shared.CompositeLookupKey()`)
- [ ] Pointer helper functions are defined locally
- [ ] Test compiles without errors
- [ ] Test runs successfully
## Key Patterns
- **SIMPLIFIED TEST STRUCTURE**: Only 3-4 test cases - Get (with StaticTests), List/Search, Search with Terraform format (if applicable), ErrorHandling
- Use `dynamic.MakeAdapter()` to create adapters
- Always validate SDP item type, scope, and unique attribute
- Include comprehensive attribute validation in Get tests using **camelCase** for attribute names
- Use `t.Skipf()` for optional functionality (like Search)
- Always include static tests with blast propagation
- Mock HTTP responses must match actual API endpoints exactly
- **No length checks**: Assert items directly with `sdpItems[0]` and `sdpItems[1]`

## Post-Implementation Validation

After completing the adapter and test implementation, you MUST run the following validation steps:

### 1. Run golangci-lint
Execute golangci-lint on the sources/gcp directory to check for code quality issues:

```bash
golangci-lint run ./sources/gcp/...
```

**If golangci-lint fails:**
- Analyze the reported issues carefully
- Fix all linting errors and warnings
- Common issues include:
  - Unused variables or imports
  - Missing error handling
  - Code formatting issues
  - Inefficient code patterns
- Re-run golangci-lint until all issues are resolved

### 2. Run Unit Tests
Execute the full test suite for the sources/gcp directory:

```bash
go test -race ./sources/gcp/... -v
```

**If tests fail:**
- Analyze the test failures and error messages
- Common issues include:
  - Missing mock responses for HTTP calls
  - Incorrect protobuf type usage
  - Wrong API endpoint URLs
  - Missing or incorrect blast propagation configurations
  - Scope format issues (especially for regional resources)
- Fix the underlying issues in the adapter or test code
- Re-run tests until all tests pass

### 3. Automatic Issue Resolution
When either golangci-lint or tests fail:
1. **Analyze the root cause** of each failure
2. **Fix the issues systematically** - don't just suppress warnings
3. **Verify the fix** by re-running the failing command
4. **Repeat until both commands pass successfully**

### 4. Final Validation
Both commands must pass before considering the implementation complete:
- ✅ `golangci-lint run ./sources/gcp/...` (no errors or warnings)
- ✅ `go test -race ./sources/gcp/... -v` (all tests pass)

**Do not proceed** with any pull request or consider the task complete until both validation steps pass successfully.
## Test Structure Summary
```
TestYourAdapter(t *testing.T) {
    // Setup mock data and responses

    t.Run("Get", func(t *testing.T) {
        // Test Get functionality
        t.Run("StaticTests", func(t *testing.T) {
            // Test ALL blast propagation links
        })
    })

    t.Run("Search", func(t *testing.T) {  // OR "List" for project-level
        // Test Search/List functionality
    })

    t.Run("ErrorHandling", func(t *testing.T) {
        // Test error responses
    })
}
```
