package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// This executable produces an adapter authoring prompt by filling a template with
// user-provided parameters.
// Usage:
//   go run ./sources/gcp/dynamic/prompter -name monitoring-alert-policy -api https://... -type https://...
// -type is optional.

const baseTemplate = `## Task
Create a new dynamic adapter for GCP {{NAME}} resource.

## Context
- **Adapter File**: ` + "`sources/gcp/dynamic/adapters/{{NAME}}.go`" + ` (to be created)
- **API Reference**: {{API_REF}}
{{TYPE_LINE}}

## Files to Create
- ` + "`sources/gcp/dynamic/adapters/{{NAME}}.go`" + `
- ` + "`sources/gcp/shared/item-types.go`" + ` (if new SDP item type needed)
- ` + "`sources/gcp/shared/models.go`" + ` (if new SDP item type needed)

## Instructions
Follow the dynamic adapter creation rules in ` + "`.cursor/rules/dynamic-adapter-creation.md`" + ` for comprehensive implementation guidance.`

func main() {
	name := flag.String("name", "", "(required) adapter name, e.g. monitoring-alert-policy")
	api := flag.String("api-ref", "", "(required) GCP reference for API Call structure")
	typeRef := flag.String("type-ref", "", "(optional) GCP reference for Type Definition")
	verbose := flag.Bool("verbose", false, "print ticket content instead of copying to clipboard")
	flag.Parse()

	missing := []string{}
	if *name == "" {
		missing = append(missing, "-name")
	}
	if *api == "" {
		missing = append(missing, "-api-ref")
	}
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "Missing required flags: %s\n", strings.Join(missing, ", "))
		flag.Usage()
		os.Exit(2)
	}

	// Generate adapter creation description
	adapterDescription := baseTemplate
	adapterDescription = strings.ReplaceAll(adapterDescription, "{{NAME}}", *name)
	adapterDescription = strings.ReplaceAll(adapterDescription, "{{API_REF}}", *api)

	if *typeRef != "" {
		adapterDescription = strings.ReplaceAll(adapterDescription, "{{TYPE_LINE}}", "- **Type Reference**: "+*typeRef+"\n")
	} else {
		adapterDescription = strings.ReplaceAll(adapterDescription, "{{TYPE_LINE}}", "")
	}

	// Generate test ticket description
	testDescription, err := generateTestTicketDescription(*name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not generate test ticket description: %v\n", err)
		testDescription = fmt.Sprintf("## Test Ticket\nWrite unit test for %s dynamic adapter (test ticket generation failed)", *name)
	}

	// Combine both descriptions with workflow instructions
	combinedDescription := fmt.Sprintf(`%s

---

%s

---

## Workflow

### Phase 1: Adapter Implementation
1. Create the adapter by following the relevant rule in `+"`.cursor/rules/dynamic-adapter-creation.md`"+`
2. Open PR with adapter implementation

### Phase 2: Unit Tests (After Reviewer Tag)
1. Wait for reviewer to add the `+"`adapter-is-approved`"+` tag to the PR
2. Once tagged, add unit tests to the same PR following `+"`.cursor/rules/dynamic-adapter-testing.md`"+`
3. Update the existing PR with test implementation`, adapterDescription, testDescription)

	// Generate Linear URL
	url := generateLinearURL(*name)

	fmt.Printf("Generated Linear issue URL:\n%s\n\n", url)

	if err := copyToClipboard(combinedDescription); err != nil {
		fmt.Println("💡 Tip: Copy the description below to paste into the Linear issue")
	} else {
		fmt.Println("✅ Combined description copied to clipboard!")
	}

	fmt.Printf("\nClick the URL above to create a new Linear issue with:\n")
	fmt.Printf("- Title: Create %s dynamic adapter\n", *name)
	fmt.Printf("- Assignee: cursor\n")
	fmt.Printf("- Project: GCP Source Improvements\n")
	fmt.Printf("- Cycle: This\n")
	fmt.Printf("- Size: Small (2 points)\n")
	fmt.Printf("- Status: Todo\n")
	fmt.Printf("- Milestone: Quality Improvements\n\n")

	if *verbose {
		fmt.Println("Combined description is already copied to clipboard - paste it into the issue:")
		fmt.Println("==========================================")
		fmt.Println(combinedDescription)
		fmt.Println("==========================================")
	} else {
		fmt.Println("Combined description is already copied to clipboard - paste it into the issue.")
	}
}

func generateTestTicketDescription(adapterName string) (string, error) {
	// Minimal test ticket description - let Cursor rule handle the details
	return fmt.Sprintf(`## Test Ticket
Write unit tests for the `+"`%s`"+` dynamic adapter.

## Files to Create
- `+"`sources/gcp/dynamic/adapters/%s_test.go`"+`

## Instructions
Follow the dynamic adapter testing rules in `+"`.cursor/rules/dynamic-adapter-testing.md`"+` for comprehensive test implementation.`, adapterName, adapterName), nil
}

func generateLinearURL(adapterName string) string {
	title := fmt.Sprintf("Create %s dynamic adapter", adapterName)
	titleEncoded := strings.ReplaceAll(title, " ", "+")

	return fmt.Sprintf("https://linear.new?title=%s&assignee=cursor&project=GCP+Source+Improvements&cycle=This&estimate=2&status=Todo&projectMilestone=Quantity+Improvements",
		titleEncoded)
}

func copyToClipboard(text string) error {
	// Define allowed clipboard commands for security
	allowedCommands := map[string][]string{
		"pbcopy": {},
		"xclip":  {"-selection", "clipboard"},
		"wl-copy": {},
	}

	// Try different clipboard commands based on OS
	commandOrder := []string{"pbcopy", "xclip", "wl-copy"}

	for _, cmdName := range commandOrder {
		args := allowedCommands[cmdName]

		// Check if command is available with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if exec.CommandContext(ctx, cmdName).Run() != nil {
			cancel()
			continue // Command not available
		}
		cancel()

		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		process := exec.CommandContext(ctx, cmdName, args...)
		stdin, err := process.StdinPipe()
		if err != nil {
			cancel()
			continue
		}

		if err := process.Start(); err != nil {
			cancel()
			continue
		}

		writer := bufio.NewWriter(stdin)
		if _, err := writer.WriteString(text); err != nil {
			stdin.Close()
			cancel()
			continue
		}
		writer.Flush()
		stdin.Close()

		if err := process.Wait(); err != nil {
			cancel()
			continue
		}

		cancel()
		return nil // Success
	}

	return fmt.Errorf("no clipboard command available")
}
