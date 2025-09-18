# Dynamic Adapter AI Tools

This directory contains tools for generating prompts and tickets for dynamic adapter development and testing.

## Files

- `generate-test-ticket-cmd/` - Go implementation for generating Linear ticket content for dynamic adapter unit tests
- `generate-adapter-ticket-cmd/` - Go implementation for generating Linear ticket content for creating new dynamic adapters
- `build.sh` - Build script for both tools
- `README.md` - This documentation

## Related Files

- `../adapters/.cursor/rules/dynamic-adapter-testing.md` - Cursor agent rules for writing adapter tests
- `../adapters/.cursor/rules/dynamic-adapter-creation.md` - Cursor agent rules for creating new adapters
- `../adapters/` - Directory containing dynamic adapter implementations

## generate-adapter-ticket

### Purpose
Generates complete Linear ticket content for creating new dynamic adapters. This tool helps create comprehensive tickets for implementing new GCP resource adapters with proper context and requirements.

### Usage
```bash
# Run directly with go run
go run generate-adapter-ticket-cmd/main.go -name <adapter-name> -api-ref <api-reference-url> [-type-ref <type-reference-url>] [--verbose]

# Or build and run
./build.sh
./generate-adapter-ticket -name <adapter-name> -api-ref <api-reference-url> [-type-ref <type-reference-url>] [--verbose]

# Build for specific platform
./build.sh linux/amd64
./build.sh darwin/arm64
```

### Examples
```bash
# Generate ticket for monitoring alert policy adapter
go run generate-adapter-ticket-cmd/main.go -name monitoring-alert-policy -api-ref "https://cloud.google.com/monitoring/api/ref_v3/rest/v3/projects.alertPolicies"

# Generate ticket with type reference
go run generate-adapter-ticket-cmd/main.go -name compute-instance-template -api-ref "https://cloud.google.com/compute/docs/reference/rest/v1/instanceTemplates" -type-ref "https://cloud.google.com/compute/docs/reference/rest/v1/instanceTemplates#InstanceTemplate"

# Generate with verbose output
go run generate-adapter-ticket-cmd/main.go --verbose -name storage-bucket -api-ref "https://cloud.google.com/storage/docs/json_api/v1/buckets"
```

### What it does
1. **Create Linear ticket** for new adapter implementation
2. **Generate comprehensive context** with API references
3. **Include implementation checklist** following dynamic adapter patterns
4. **Reference Cursor rules** for consistent implementation
5. **Copy description to clipboard** and optionally print it

### Output
The tool generates a Linear URL with pre-filled fields and copies the description to clipboard. The description includes:
- Task overview
- API references
- Files to create
- Implementation instructions referencing Cursor rules

## generate-test-ticket

### Purpose
Generates complete Linear ticket content for creating unit tests for dynamic adapters. The Go implementation provides better maintainability, type safety, and cross-platform compatibility.

### Usage
```bash
# Run directly with go run
go run generate-test-ticket-cmd/main.go [--verbose|-v] <adapter-name>

# Or build and run
./build.sh
./generate-test-ticket [--verbose|-v] <adapter-name>

# Build for specific platform
./build.sh linux/amd64
./build.sh darwin/arm64

# Build specific tool only
./build.sh "" generate-test-ticket
./build.sh linux/amd64 generate-test-ticket
```

### Examples
```bash
# Generate ticket for compute global forwarding rule (quiet mode)
go run generate-test-ticket-cmd/main.go compute-global-forwarding-rule

# Generate ticket with verbose output (shows description)
go run generate-test-ticket-cmd/main.go --verbose compute-global-forwarding-rule

# Short form of verbose flag
go run generate-test-ticket-cmd/main.go -v compute-global-address
```

### What it does
1. **Extract adapter information** from the adapter file in `../adapters/`
2. **Determine protobuf types** based on adapter name patterns
3. **Extract blast propagation** configuration from the adapter
4. **Generate a Linear URL** with basic fields pre-filled:
   - Title: "Write unit test for {adapter-name} dynamic adapter"
   - Assignee: Cursor Agent
   - Project: GCP Source Improvements
   - Cycle: This
   - Size: Small (2 points)
   - Status: Todo
   - Milestone: Quality Improvements
5. **Copy description to clipboard** and optionally print it

### Output
The tool generates a Linear URL with basic fields and copies the description to clipboard. In verbose mode (`--verbose` or `-v`), it also prints the complete description for review.

### Requirements
- Must be run from the `prompter` directory
- Adapter file must exist in `../adapters/`
- Adapter file must contain valid SDP item type and blast propagation configuration
- Go 1.19+ required

## Integration with Cursor Agents

The generated tickets work seamlessly with:
- **Cursor rules** in `../adapters/.cursor/rules/dynamic-adapter-testing.md`
- **Existing test patterns** from `../adapters/compute-global-address_test.go`
- **Comprehensive testing requirements** for Get, List, and Search functionality

## Workflow

### Creating New Adapters

#### Quick Mode (default)
1. **Generate Linear URL** using `generate-adapter-ticket`
2. **Click the URL** to create a new Linear issue with basic fields pre-filled
3. **Paste the description** (already copied to clipboard) into the issue
4. **Save the issue** - it's ready for implementation

#### Review Mode (verbose)
1. **Generate Linear URL** using `generate-adapter-ticket --verbose`
2. **Review the description** printed in the output
3. **Click the URL** to create a new Linear issue with basic fields pre-filled
4. **Paste the description** (already copied to clipboard) into the issue
5. **Save the issue** - it's ready for implementation

### Creating Tests for Existing Adapters

#### Quick Mode (default)
1. **Generate Linear URL** using `generate-test-ticket`
2. **Click the URL** to create a new Linear issue with basic fields pre-filled
3. **Paste the description** (already copied to clipboard) into the issue
4. **Save the issue** - it's already assigned to Cursor Agent

#### Review Mode (verbose)
1. **Generate Linear URL** using `generate-test-ticket --verbose` or `-v` flag
2. **Review the description** printed in the output
3. **Click the URL** to create a new Linear issue with basic fields pre-filled
4. **Paste the description** (already copied to clipboard) into the issue
5. **Save the issue** - it's already assigned to Cursor Agent

### Cursor Agent Execution
When a Cursor agent picks up the ticket:
1. It will automatically apply the rules from `../adapters/.cursor/rules/dynamic-adapter-testing.md`
2. Follow the comprehensive testing patterns
3. Create the test file with proper structure
4. Include all required test cases (Get, List, Search if supported)
5. Add proper blast propagation tests

## Example Ticket Content

For `compute-global-forwarding-rule`:

**Title**: `Write unit test for compute-global-forwarding-rule dynamic adapter`

**Key Details**:
- **SDP Item Type**: `gcpshared.ComputeGlobalForwardingRule`
- **Protobuf Types**: `computepb.ForwardingRule` and `computepb.ForwardingRuleList`
- **API Endpoints**:
  - GET: `https://compute.googleapis.com/compute/v1/projects/{project}/global/forwardingRules/{forwardingRule}`
  - LIST: `https://compute.googleapis.com/compute/v1/projects/{project}/global/forwardingRules`
- **Blast Propagation**: network (InOnly), subnetwork (InOnly), IPAddress (BothWays), backendService (BothWays)

## Benefits

1. **Consistency**: All tests follow the same patterns and structure
2. **Completeness**: Comprehensive coverage of Get, List, and Search functionality
3. **Automation**: Cursor agents can automatically generate high-quality tests
4. **Documentation**: Clear requirements and acceptance criteria
5. **Maintainability**: Standardized approach makes tests easier to maintain

## Adding New Adapters

### Complete Workflow for New Adapters

#### Step 1: Create Implementation Ticket
1. Run `go run generate-adapter-ticket-cmd/main.go -name my-new-adapter -api-ref "https://api-reference-url"`
2. Click the generated URL to create Linear issue with pre-filled fields
3. Paste the description (copied to clipboard) into the issue
4. Save the issue - it's ready for implementation

#### Step 2: Implement the Adapter
The Cursor agent (or developer) will:
1. Follow the rules in `../adapters/.cursor/rules/dynamic-adapter-creation.md`
2. Create the adapter file (e.g., `my-new-adapter.go`)
3. Add any necessary SDP item types to `../shared/item-types.go` and `../shared/models.go`

#### Step 3: Create Test Ticket
1. Run `go run generate-test-ticket-cmd/main.go my-new-adapter` to generate test ticket content
2. Click the generated URL to create Linear issue with pre-filled fields
3. Paste the description (copied to clipboard) into the issue
4. Save the issue - it's already assigned to Cursor Agent

### Quick Testing for Existing Adapters

When you just need tests for an existing adapter:
1. Run `go run generate-test-ticket-cmd/main.go existing-adapter-name`
2. Click the generated URL to create Linear issue with pre-filled fields
3. Paste the description (copied to clipboard) into the issue
4. Save the issue - it's already assigned to Cursor Agent

## Rules Application

### For Adapter Creation
The `../adapters/.cursor/rules/dynamic-adapter-creation.md` file ensures that:
- Proper adapter structure and patterns are followed
- Correct SDP item types and metadata are defined
- Appropriate blast propagation is configured
- Terraform mappings are included when applicable
- IAM permissions are properly defined

### For Test Creation
The `../adapters/.cursor/rules/dynamic-adapter-testing.md` file ensures that:
- All tests use the correct package (`adapters_test`)
- Proper imports are included
- Correct protobuf types are used
- Comprehensive test coverage is provided
- Static tests with blast propagation are included
- Common mistakes are avoided

This ensures consistent, high-quality implementations and unit tests for all dynamic adapters.

## Quick Reference

### Building Tools
```bash
# Build both tools for current platform
./build.sh

# Build for specific platform
./build.sh linux/amd64

# Build specific tool only
./build.sh "" generate-adapter-ticket
./build.sh "" generate-test-ticket
```

### Creating New Adapter
```bash
# Generate implementation ticket
go run generate-adapter-ticket-cmd/main.go -name my-adapter -api-ref "https://api-url"

# After implementation, generate test ticket
go run generate-test-ticket-cmd/main.go my-adapter
```

### Testing Existing Adapter
```bash
# Generate test ticket
go run generate-test-ticket-cmd/main.go existing-adapter-name
```

Both tools support `--verbose` flag to preview the description before creating tickets.