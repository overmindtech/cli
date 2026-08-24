package tracing

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestNoLegacyBrentTelemetryLiterals ensures that new brent.* or ovm.brent.*
// telemetry identifiers are not introduced in Go source after the dual-write
// wrapper removal (ENG-6186).
//
// This test scans Go source for string literals matching those patterns and
// checks whether they appear in telemetry contexts:
//   - Span names: tracer.Start(ctx, "name")
//   - Attribute keys: attribute.String("key", ...)
//   - Log field keys: WithField("key", ...)
//   - Span attributes: span.SetAttributes(attribute.String("key", ...))
//
// Non-telemetry identifiers (URLs, file paths, package references, proto
// field comments) are explicitly allowlisted.
//
// See: ENG-6197, ENG-6186
func TestNoLegacyBrentTelemetryLiterals(t *testing.T) {
	t.Parallel()

	// Pattern matching string literals containing "brent." or "ovm.brent."
	brentPattern := regexp.MustCompile(`"[^"]*\b(ovm\.)?brent\.[^"]*"`)

	// Telemetry context patterns - if a match appears near these, it's telemetry
	telemetryPatterns := []string{
		"tracer.Start",
		"Tracer().Start",
		"attribute.String",
		"attribute.Int",
		"attribute.Bool",
		"attribute.StringSlice",
		"WithField",
		"WithFields",
		"span.SetAttributes",
		"SetAttributes",
		"span.AddEvent",
		"AddEvent",
	}

	// Allowlist of non-telemetry brent.* identifiers that may remain.
	allowlist := []string{
		// Test fixture URLs (not actual telemetry emission)
		`"https://until.example`,
		`"http://until.example`,

		// Proto field comments describing legacy attribute keys
		`"ovm.brent.llm.keySource"`,

		// File paths and directory references
		`".brent/workflows"`,
		`".brent/"`,
		`"/brent/"`,

		// Function/method names in strings (not telemetry keys)
		`"brent.validateBYOConnectEndpoint"`,
		`"brent.MapByoIntakeError"`,

		// Auth0 identifiers, package names, and other non-telemetry uses
		`"brent.Auth0ManagementClient"`,
		`"brent.proto"`,
		`"brent-backend"`,
		`"project-brent"`,

		// Legacy plan ID prefixes in tests/docs (not telemetry)
		`"BRENT-"`,

		// Test assertions checking for absence of brent prefix
		`"brent."`, // Used in strings.HasPrefix checks in tests
	}

	var failures []string

	// Walk all Go source files
	err := filepath.Walk("../..", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-Go files, generated files, and vendor directories
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == ".git" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip generated protobuf files - they contain proto field comments
		if strings.HasSuffix(path, ".pb.go") {
			return nil
		}

		// Read and scan the file
		content, err := os.ReadFile(path)
		if err != nil {
			t.Logf("warning: could not read %s: %v", path, err)
			return nil
		}

		scanner := bufio.NewScanner(strings.NewReader(string(content)))
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Find all matches of brent.* or ovm.brent.* in string literals
			matches := brentPattern.FindAllString(line, -1)
			if len(matches) == 0 {
				continue
			}

			// Check each match
			for _, match := range matches {
				// Check if this match is allowlisted
				if isAllowlisted(match, allowlist) {
					continue
				}

				// Check if this match is in a telemetry context
				if isTelemetryContext(line, telemetryPatterns) {
					relPath, _ := filepath.Rel("../..", path)
					failures = append(failures,
						formatViolation(relPath, lineNum, match, line))
				}
			}
		}

		return nil
	})

	if err != nil {
		t.Fatalf("failed to walk source tree: %v", err)
	}

	if len(failures) > 0 {
		t.Errorf("Found %d new brent.*/ovm.brent.* telemetry literal(s):\n\n%s\n\n"+
			"After ENG-6186, all telemetry must use until.*/ovm.until.* naming.\n"+
			"Non-telemetry identifiers (test URLs, file paths) "+
			"should be added to the allowlist in go/tracing/telemetry_naming_test.go "+
			"with a comment explaining why they are not telemetry.",
			len(failures),
			strings.Join(failures, "\n"))
	}
}

// isAllowlisted checks if a match is in the explicit allowlist
func isAllowlisted(match string, allowlist []string) bool {
	matchLower := strings.ToLower(match)
	for _, allowed := range allowlist {
		if strings.Contains(matchLower, strings.ToLower(allowed)) {
			return true
		}
	}
	return false
}

// isTelemetryContext checks if the line contains telemetry-related function calls
func isTelemetryContext(line string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}

// formatViolation formats a violation for the test error message
func formatViolation(path string, lineNum int, match, line string) string {
	// Format: file:line: match in "context"
	trimmed := strings.TrimSpace(line)
	// Remove inline comments for cleaner output
	if idx := strings.Index(trimmed, "//"); idx > 0 {
		trimmed = strings.TrimSpace(trimmed[:idx])
	}

	return trimmed +
		"\n    → " + path + ":" + formatLineNum(lineNum) +
		"\n    → Found disallowed telemetry literal: " + match
}

func formatLineNum(n int) string {
	return strings.TrimSpace(strings.Fields(fmt.Sprintf("%d", n))[0])
}
