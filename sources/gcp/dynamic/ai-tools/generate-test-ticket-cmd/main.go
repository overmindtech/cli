package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Verbose     bool
	AdapterName string
}

type AdapterInfo struct {
	Name string
}

func main() {
	config := parseArgs()

	adapterInfo, err := extractAdapterInfo(config.AdapterName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	description := generateDescription(adapterInfo)
	url := generateLinearURL(adapterInfo.Name)

	fmt.Printf("Generated Linear issue URL:\n%s\n\n", url)

	if err := copyToClipboard(description); err != nil {
		fmt.Println("💡 Tip: Copy the description below to paste into the Linear issue")
	} else {
		fmt.Println("✅ Description copied to clipboard!")
	}

	fmt.Printf("\nClick the URL above to create a new Linear issue with:\n")
	fmt.Printf("- Title: Write unit test for %s dynamic adapter\n", adapterInfo.Name)
	fmt.Printf("- Assignee: cursor\n")
	fmt.Printf("- Project: GCP Source Improvements\n")
	fmt.Printf("- Cycle: This\n")
	fmt.Printf("- Size: Small (2 points)\n")
	fmt.Printf("- Status: Todo\n")
	fmt.Printf("- Milestone: Quality Improvements\n\n")

	if config.Verbose {
		fmt.Println("Description is already copied to clipboard - paste it into the issue:")
		fmt.Println("==========================================")
		fmt.Println(description)
		fmt.Println("==========================================")
	} else {
		fmt.Println("Description is already copied to clipboard - paste it into the issue.")
	}
}

func parseArgs() Config {
	config := Config{}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	args := os.Args[1:]
	for _, arg := range args {
		switch arg {
		case "--verbose", "-v":
			config.Verbose = true
		default:
			if config.AdapterName == "" {
				config.AdapterName = arg
			} else {
				fmt.Fprintf(os.Stderr, "Error: Multiple adapter names provided\n")
				os.Exit(1)
			}
		}
	}

	if config.AdapterName == "" {
		printUsage()
		os.Exit(1)
	}

	return config
}

func printUsage() {
	fmt.Printf("Usage: %s [--verbose|-v] <adapter-file-name>\n", os.Args[0])
	fmt.Printf("Example: %s compute-global-forwarding-rule\n", os.Args[0])
	fmt.Printf("Example: %s --verbose compute-global-forwarding-rule\n", os.Args[0])
}

func extractAdapterInfo(adapterName string) (*AdapterInfo, error) {
	adapterFile := adapterName + ".go"
	adapterPath := filepath.Join("..", "adapters", adapterFile)

	// Check if adapter file exists
	if _, err := os.Stat(adapterPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("adapter file '%s' not found", adapterPath)
	}

	// For simplified version, we just need the adapter name
	info := &AdapterInfo{Name: adapterName}
	return info, nil
}

func generateDescription(info *AdapterInfo) string {
	return fmt.Sprintf(`## Task
Write unit tests for the `+"`%s`"+` dynamic adapter.

## Context
- **Adapter File**: `+"`sources/gcp/dynamic/adapters/%s.go`"+`
- **Test File**: `+"`sources/gcp/dynamic/adapters/%s_test.go`"+` (to be created)

## Files to Create
- `+"`sources/gcp/dynamic/adapters/%s_test.go`"+`

## Instructions
Follow the dynamic adapter testing rules in `+"`.cursor/rules/dynamic-adapter-testing.md`"+` for comprehensive test implementation.`,
		info.Name,
		info.Name,
		info.Name,
		info.Name)
}

func generateLinearURL(adapterName string) string {
	title := fmt.Sprintf("Write unit test for %s dynamic adapter", adapterName)
	titleEncoded := strings.ReplaceAll(title, " ", "+")

	return fmt.Sprintf("https://linear.new?title=%s&assignee=cursor&project=GCP+Source+Improvements&cycle=This&estimate=2&status=Todo&projectMilestone=Quality+Improvements",
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
