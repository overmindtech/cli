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
	"go.yaml.in/yaml/v3"
)

// KnowledgeFile represents a discovered and validated knowledge file
type KnowledgeFile struct {
	Name        string
	Description string
	Content     string // markdown body only (excluding frontmatter)
	FileName    string // path relative to .overmind/knowledge/
	SourceDir   string // absolute path to the knowledge directory this file came from
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

// FindKnowledgeDir walks up from startDir looking for a .overmind/knowledge/
// directory. Returns the absolute path if found, or empty string if not.
// Stops at the repository root (.git boundary) or filesystem root to avoid
// picking up knowledge files from unrelated parent projects.
func FindKnowledgeDir(startDir string) string {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, ".overmind", "knowledge")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// ResolveKnowledgeDirs returns the list of knowledge directories to use.
// If explicitDirs is non-empty, returns those directories (warning about any that don't exist).
// If explicitDirs is empty, falls back to FindKnowledgeDir(startDir) for backward compatibility.
// Returns an empty slice if no directories are found or specified.
func ResolveKnowledgeDirs(startDir string, explicitDirs []string) []string {
	if len(explicitDirs) == 0 {
		// Fallback to auto-discovery for backward compatibility
		dir := FindKnowledgeDir(startDir)
		if dir != "" {
			return []string{dir}
		}
		return []string{}
	}

	// Use explicit directories, warning about missing ones but tolerating them
	var resolved []string
	for _, dir := range explicitDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			log.WithField("dir", dir).Warn("Failed to resolve absolute path for knowledge directory, skipping")
			continue
		}
		if _, err := os.Stat(absDir); err != nil {
			log.WithField("dir", absDir).WithError(err).Warn("Cannot access knowledge directory, skipping")
			continue
		}
		resolved = append(resolved, absDir)
	}
	return resolved
}

// Discover walks the knowledge directories and discovers all valid knowledge files.
// Accepts a list of knowledge directories to search. Later directories in the list
// override earlier ones when the same knowledge file name appears in multiple directories
// (emits a warning when this happens).
// Returns valid files and any warnings encountered during discovery.
func Discover(knowledgeDirs ...string) ([]KnowledgeFile, []Warning) {
	// Handle legacy single-directory signature for backward compatibility
	if len(knowledgeDirs) == 1 && knowledgeDirs[0] == "" {
		return []KnowledgeFile{}, []Warning{}
	}

	var allFiles []KnowledgeFile
	var allWarnings []Warning

	// Track seen names across all directories for cross-directory deduplication
	// Maps name -> {sourceDir, relPath} of the file that won
	type nameOwner struct {
		sourceDir string
		relPath   string
	}
	seenNames := make(map[string]nameOwner)

	// Process each directory in order
	for _, knowledgeDir := range knowledgeDirs {
		if knowledgeDir == "" {
			continue
		}

		files, warnings := discoverOne(knowledgeDir)
		allWarnings = append(allWarnings, warnings...)

		// Apply cross-directory deduplication: later directories override earlier ones
		for _, kf := range files {
			if owner, exists := seenNames[kf.Name]; exists {
				// Name collision across directories: later wins, emit warning log only
				log.WithField("name", kf.Name).
					WithField("earlier", filepath.Join(owner.sourceDir, owner.relPath)).
					WithField("later", filepath.Join(kf.SourceDir, kf.FileName)).
					Warn("Knowledge file name collision across directories, using later directory")

				// Remove the earlier file from allFiles and replace with the new one
				for i, f := range allFiles {
					if f.Name == kf.Name {
						allFiles = append(allFiles[:i], allFiles[i+1:]...)
						break
					}
				}
			}

			seenNames[kf.Name] = nameOwner{
				sourceDir: kf.SourceDir,
				relPath:   kf.FileName,
			}
			allFiles = append(allFiles, kf)
		}
	}

	return allFiles, allWarnings
}

// discoverOne walks a single knowledge directory and discovers valid knowledge files.
// This is the internal implementation that processes one directory.
// Returns valid files and any warnings encountered during discovery.
func discoverOne(knowledgeDir string) ([]KnowledgeFile, []Warning) {
	var files []KnowledgeFile
	var warnings []Warning

	// Check if directory exists
	if _, err := os.Stat(knowledgeDir); os.IsNotExist(err) {
		return files, warnings
	}

	// Make knowledgeDir absolute for consistent SourceDir tracking
	absKnowledgeDir, err := filepath.Abs(knowledgeDir)
	if err != nil {
		warnings = append(warnings, Warning{
			Path:   knowledgeDir,
			Reason: fmt.Sprintf("failed to resolve absolute path: %v", err),
		})
		return files, warnings
	}

	// Collect all markdown files first for deterministic ordering
	type fileInfo struct {
		path    string
		relPath string
	}
	var mdFiles []fileInfo

	err = filepath.WalkDir(absKnowledgeDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Warn about directories/files we can't access
			relPath, _ := filepath.Rel(absKnowledgeDir, path)
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

		relPath, err := filepath.Rel(absKnowledgeDir, path)
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

	// Track seen names within this directory for intra-directory deduplication
	seenNames := make(map[string]string) // name -> first file path

	// Process each file
	for _, f := range mdFiles {
		kf, warn := processFile(f.path, f.relPath, absKnowledgeDir)
		if warn != nil {
			warnings = append(warnings, *warn)
			continue
		}

		// Check for duplicate names within this directory
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
func processFile(path, relPath, sourceDir string) (*KnowledgeFile, *Warning) {
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
		SourceDir:   sourceDir,
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
	bodyStartIdx := min(startIdx+endIdx+closingDelimLen, len(content))
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
// Accepts a variable number of knowledge directories to search.
func DiscoverAndConvert(ctx context.Context, knowledgeDirs ...string) []*sdp.Knowledge {
	if len(knowledgeDirs) > 0 {
		log.WithContext(ctx).WithField("knowledgeDirs", knowledgeDirs).Debug("Resolved knowledge directories")
	}

	knowledgeFiles, warnings := Discover(knowledgeDirs...)

	// Log warnings
	for _, w := range warnings {
		log.WithContext(ctx).WithField("path", w.Path).WithField("reason", w.Reason).Warn("Skipping knowledge file")
	}

	// Convert to SDP Knowledge messages
	sdpKnowledge := make([]*sdp.Knowledge, 0, len(knowledgeFiles))
	for _, kf := range knowledgeFiles {
		sdpKnowledge = append(sdpKnowledge, &sdp.Knowledge{
			Name:        kf.Name,
			Description: kf.Description,
			Content:     kf.Content,
			FileName:    kf.FileName,
		})
	}

	// Log when knowledge files are loaded
	if len(knowledgeFiles) > 0 {
		log.WithContext(ctx).WithField("knowledgeCount", len(knowledgeFiles)).Info("Loaded knowledge files")
	}

	return sdpKnowledge
}
