package knowledge

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/overmindtech/cli/go/sdp-go"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// KnowledgeFile represents a discovered and validated knowledge file
type KnowledgeFile struct {
	Name        string
	Description string
	Content     string // markdown body only (excluding frontmatter)
	FileName    string // path relative to .overmind/knowledge/
}

// Warning represents a validation or parsing issue with a knowledge file
type Warning struct {
	Path   string // relative path within .overmind/knowledge/
	Reason string
}

// frontmatter represents the YAML frontmatter structure
type frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// nameRegex validates knowledge file names (kebab-case: lowercase letters, digits, hyphens)
// Must start with a letter, end with letter or digit, 1-64 chars total
var nameRegex = regexp.MustCompile(`^[a-z]([a-z0-9-]*[a-z0-9])?$`)

const (
	// maxFileSize is the maximum allowed size for a knowledge file (10MB)
	// This prevents memory exhaustion and excessive API payload sizes
	maxFileSize = 10 * 1024 * 1024 // 10MB
)

// Discover walks the knowledge directory and discovers all valid knowledge files
// Returns valid files and any warnings encountered during discovery
func Discover(knowledgeDir string) ([]KnowledgeFile, []Warning) {
	var files []KnowledgeFile
	var warnings []Warning

	// Check if directory exists
	if _, err := os.Stat(knowledgeDir); os.IsNotExist(err) {
		return files, warnings
	}

	// Collect all markdown files first for deterministic ordering
	type fileInfo struct {
		path    string
		relPath string
	}
	var mdFiles []fileInfo

	err := filepath.WalkDir(knowledgeDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Warn about directories/files we can't access
			relPath, _ := filepath.Rel(knowledgeDir, path)
			warnings = append(warnings, Warning{
				Path:   relPath,
				Reason: fmt.Sprintf("cannot access: %v", err),
			})
			return nil // Continue walking
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process .md files
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		relPath, err := filepath.Rel(knowledgeDir, path)
		if err != nil {
			return err
		}

		mdFiles = append(mdFiles, fileInfo{
			path:    path,
			relPath: relPath,
		})

		return nil
	})

	if err != nil {
		warnings = append(warnings, Warning{
			Path:   "",
			Reason: fmt.Sprintf("error walking directory: %v", err),
		})
		return files, warnings
	}

	// Sort files lexicographically for deterministic processing
	sort.Slice(mdFiles, func(i, j int) bool {
		return mdFiles[i].relPath < mdFiles[j].relPath
	})

	// Track seen names for deduplication
	seenNames := make(map[string]string) // name -> first file path

	// Process each file
	for _, f := range mdFiles {
		kf, warn := processFile(f.path, f.relPath)
		if warn != nil {
			warnings = append(warnings, *warn)
			continue
		}

		// Check for duplicate names
		if firstPath, exists := seenNames[kf.Name]; exists {
			warnings = append(warnings, Warning{
				Path:   f.relPath,
				Reason: fmt.Sprintf("duplicate name %q (already loaded from %q)", kf.Name, firstPath),
			})
			continue
		}

		seenNames[kf.Name] = f.relPath
		files = append(files, *kf)
	}

	return files, warnings
}

// processFile reads and validates a single knowledge file
func processFile(path, relPath string) (*KnowledgeFile, *Warning) {
	// Check file size before reading
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, &Warning{
			Path:   relPath,
			Reason: fmt.Sprintf("cannot stat file: %v", err),
		}
	}
	
	if fileInfo.Size() > maxFileSize {
		return nil, &Warning{
			Path:   relPath,
			Reason: fmt.Sprintf("file size %d bytes exceeds maximum allowed size of %d bytes", fileInfo.Size(), maxFileSize),
		}
	}
	
	// Read file content
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, &Warning{
			Path:   relPath,
			Reason: fmt.Sprintf("cannot read file: %v", err),
		}
	}

	// Parse frontmatter
	name, description, body, err := parseFrontmatter(string(content))
	if err != nil {
		return nil, &Warning{
			Path:   relPath,
			Reason: err.Error(),
		}
	}

	// Validate name
	if err := validateName(name); err != nil {
		return nil, &Warning{
			Path:   relPath,
			Reason: err.Error(),
		}
	}

	// Validate description
	if err := validateDescription(description); err != nil {
		return nil, &Warning{
			Path:   relPath,
			Reason: err.Error(),
		}
	}

	return &KnowledgeFile{
		Name:        name,
		Description: description,
		Content:     body,
		FileName:    relPath,
	}, nil
}

// parseFrontmatter extracts YAML frontmatter from markdown content
// Returns name, description, body (without frontmatter), and any error
func parseFrontmatter(content string) (string, string, string, error) {
	// Frontmatter must start at the beginning of the file
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		return "", "", "", fmt.Errorf("frontmatter is required (must start with ---)")
	}

	// Determine opening delimiter length
	startIdx := 4 // "---\n"
	if strings.HasPrefix(content, "---\r\n") {
		startIdx = 5 // "---\r\n"
	}

	// Find the closing delimiter
	remaining := content[startIdx:]
	
	// Handle edge case: empty frontmatter where second --- is immediately after first
	if strings.HasPrefix(remaining, "---\n") || strings.HasPrefix(remaining, "---\r\n") {
		bodyStartIdx := startIdx + 4 // "---\n"
		if strings.HasPrefix(remaining, "---\r\n") {
			bodyStartIdx = startIdx + 5 // "---\r\n"
		}
		body := strings.TrimLeft(content[bodyStartIdx:], "\n\r")
		
		// Empty frontmatter will result in empty name/description which will fail validation
		var fm frontmatter
		return fm.Name, fm.Description, body, nil
	}
	
	// Find closing delimiter and track which type we found
	var endIdx int
	var closingDelimLen int
	
	// Try CRLF first (more specific), then LF
	endIdx = strings.Index(remaining, "\n---\r\n")
	if endIdx != -1 {
		closingDelimLen = 6 // "\n---\r\n"
	} else {
		endIdx = strings.Index(remaining, "\n---\n")
		if endIdx != -1 {
			closingDelimLen = 5 // "\n---\n"
		} else {
			// Check for closing delimiter at end of file (more specific first)
			if strings.HasSuffix(remaining, "\r\n---") {
				endIdx = len(remaining) - 5
				closingDelimLen = 5 // "\r\n---" (no trailing newline)
			} else if strings.HasSuffix(remaining, "\n---") {
				endIdx = len(remaining) - 4
				closingDelimLen = 4 // "\n---" (no trailing newline)
			} else {
				return "", "", "", fmt.Errorf("frontmatter closing delimiter (---) not found")
			}
		}
	}

	// Extract YAML content
	yamlContent := remaining[:endIdx]

	// Parse YAML with strict mode (unknown fields will cause error)
	var fm frontmatter
	decoder := yaml.NewDecoder(strings.NewReader(yamlContent))
	decoder.KnownFields(true) // Reject unknown fields
	if err := decoder.Decode(&fm); err != nil {
		if strings.Contains(err.Error(), "field") && strings.Contains(err.Error(), "not found") {
			return "", "", "", fmt.Errorf("only 'name' and 'description' fields are allowed in frontmatter")
		}
		return "", "", "", fmt.Errorf("invalid YAML in frontmatter: %w", err)
	}

	// Extract body using the correct offset for the delimiter type found
	bodyStartIdx := startIdx + endIdx + closingDelimLen
	if bodyStartIdx > len(content) {
		bodyStartIdx = len(content)
	}
	body := strings.TrimLeft(content[bodyStartIdx:], "\n\r")

	// Trim whitespace from name and description as per validation
	return strings.TrimSpace(fm.Name), strings.TrimSpace(fm.Description), body, nil
}

// validateName checks if the name meets the specification requirements
func validateName(name string) error {
	name = strings.TrimSpace(name)

	if name == "" {
		return fmt.Errorf("name is required")
	}

	if len(name) > 64 {
		return fmt.Errorf("name must be 64 characters or less")
	}

	if !nameRegex.MatchString(name) {
		return fmt.Errorf("name must use kebab-case (lowercase letters, digits, hyphens; start with letter, end with letter or digit)")
	}

	return nil
}

// validateDescription checks if the description meets the specification requirements
func validateDescription(description string) error {
	description = strings.TrimSpace(description)

	if description == "" {
		return fmt.Errorf("description is required")
	}

	if len(description) > 1024 {
		return fmt.Errorf("description must be 1024 characters or less")
	}

	return nil
}

// DiscoverAndConvert discovers knowledge files and converts them to SDP Knowledge messages.
// This is a convenience function that combines discovery, warning logging, and conversion
// to reduce code duplication across commands.
func DiscoverAndConvert(ctx context.Context, knowledgeDir string) []*sdp.Knowledge {
	knowledgeFiles, warnings := Discover(knowledgeDir)
	
	// Log warnings
	for _, w := range warnings {
		log.WithContext(ctx).Warnf("Warning: skipping knowledge file %q: %s", w.Path, w.Reason)
	}
	
	// Convert to SDP Knowledge messages
	sdpKnowledge := make([]*sdp.Knowledge, len(knowledgeFiles))
	for i, kf := range knowledgeFiles {
		sdpKnowledge[i] = &sdp.Knowledge{
			Name:        kf.Name,
			Description: kf.Description,
			Content:     kf.Content,
			FileName:    kf.FileName,
		}
	}
	
	// Log when knowledge files are loaded
	if len(knowledgeFiles) > 0 {
		log.WithContext(ctx).WithField("knowledgeCount", len(knowledgeFiles)).Info("Loaded knowledge files")
	}
	
	return sdpKnowledge
}
