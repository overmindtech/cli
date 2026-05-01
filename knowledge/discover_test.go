package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscover_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, "knowledge")
	err := os.Mkdir(knowledgeDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	files, warnings := Discover(knowledgeDir)

	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
}

func TestDiscover_DirectoryDoesNotExist(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, "nonexistent")

	files, warnings := Discover(knowledgeDir)

	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
}

func TestDiscover_ValidFiles(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, "knowledge")
	err := os.Mkdir(knowledgeDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Create valid files at root
	writeFile(t, filepath.Join(knowledgeDir, "aws-s3.md"), `---
name: aws-s3-security
description: Security best practices for S3 buckets
---
# AWS S3 Security
Content here.
`)

	// Create valid file in subfolder
	subdir := filepath.Join(knowledgeDir, "cloud")
	err = os.Mkdir(subdir, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(subdir, "gcp.md"), `---
name: gcp-compute
description: GCP Compute Engine guidelines
---
# GCP Compute
Content here.
`)

	files, warnings := Discover(knowledgeDir)

	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d: %v", len(warnings), warnings)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	// Check first file (lexicographic order)
	if files[0].Name != "aws-s3-security" {
		t.Errorf("expected name 'aws-s3-security', got %q", files[0].Name)
	}
	if files[0].Description != "Security best practices for S3 buckets" {
		t.Errorf("unexpected description: %q", files[0].Description)
	}
	if files[0].FileName != "aws-s3.md" {
		t.Errorf("expected fileName 'aws-s3.md', got %q", files[0].FileName)
	}
	if files[0].Content != "# AWS S3 Security\nContent here.\n" {
		t.Errorf("unexpected content: %q", files[0].Content)
	}

	// Check second file
	if files[1].Name != "gcp-compute" {
		t.Errorf("expected name 'gcp-compute', got %q", files[1].Name)
	}
	if files[1].FileName != filepath.Join("cloud", "gcp.md") {
		t.Errorf("expected fileName 'cloud/gcp.md', got %q", files[1].FileName)
	}
}

func TestDiscover_NonMarkdownFilesSkipped(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, "knowledge")
	err := os.Mkdir(knowledgeDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Create non-markdown files
	writeFile(t, filepath.Join(knowledgeDir, "readme.txt"), "This is a text file")
	writeFile(t, filepath.Join(knowledgeDir, "config.yaml"), "key: value")
	writeFile(t, filepath.Join(knowledgeDir, "script.sh"), "#!/bin/bash")

	// Create one valid markdown file
	writeFile(t, filepath.Join(knowledgeDir, "valid.md"), `---
name: valid-file
description: A valid knowledge file
---
Content
`)

	files, warnings := Discover(knowledgeDir)

	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d: %v", len(warnings), warnings)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}
}

func TestDiscover_NestedSubfolders(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, "knowledge")

	// Create nested directory structure
	deepDir := filepath.Join(knowledgeDir, "cloud", "aws", "services")
	err := os.MkdirAll(deepDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(deepDir, "s3.md"), `---
name: deep-s3
description: Deeply nested file
---
Content
`)

	files, warnings := Discover(knowledgeDir)

	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d: %v", len(warnings), warnings)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	expectedPath := filepath.Join("cloud", "aws", "services", "s3.md")
	if files[0].FileName != expectedPath {
		t.Errorf("expected fileName %q, got %q", expectedPath, files[0].FileName)
	}
}

func TestParseFrontmatter_Valid(t *testing.T) {
	content := `---
name: test-file
description: Test description
---
# Markdown content
Here is some content.
`

	name, desc, body, err := parseFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "test-file" {
		t.Errorf("expected name 'test-file', got %q", name)
	}
	if desc != "Test description" {
		t.Errorf("expected description 'Test description', got %q", desc)
	}
	if body != "# Markdown content\nHere is some content.\n" {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestParseFrontmatter_CRLF(t *testing.T) {
	// Test with Windows-style CRLF line endings
	content := "---\r\nname: windows-file\r\ndescription: File with CRLF endings\r\n---\r\n# Windows content\r\nWith CRLF.\r\n"

	name, desc, body, err := parseFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "windows-file" {
		t.Errorf("expected name 'windows-file', got %q", name)
	}
	if desc != "File with CRLF endings" {
		t.Errorf("expected description 'File with CRLF endings', got %q", desc)
	}
	// Body should have CRLF stripped by TrimLeft
	if !strings.Contains(body, "Windows content") {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestParseFrontmatter_CRLFAtEOF(t *testing.T) {
	// Test CRLF with frontmatter at end of file (no trailing content)
	content := "---\r\nname: eof-test\r\ndescription: Frontmatter at EOF\r\n---"

	name, desc, _, err := parseFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "eof-test" {
		t.Errorf("expected name 'eof-test', got %q", name)
	}
	if desc != "Frontmatter at EOF" {
		t.Errorf("expected description 'Frontmatter at EOF', got %q", desc)
	}
}

func TestParseFrontmatter_MixedLineEndings(t *testing.T) {
	// Test with LF in frontmatter but CRLF in closing delimiter
	content := "---\nname: mixed-file\ndescription: Mixed line endings\n---\r\n# Content\nHere.\n"

	name, desc, body, err := parseFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "mixed-file" {
		t.Errorf("expected name 'mixed-file', got %q", name)
	}
	if desc != "Mixed line endings" {
		t.Errorf("expected description 'Mixed line endings', got %q", desc)
	}
	if !strings.Contains(body, "Content") {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestParseFrontmatter_Whitespace(t *testing.T) {
	// Test that whitespace is trimmed from name and description
	content := `---
name:   whitespace-name  
description:   Lots of whitespace   
---
Content
`

	name, desc, _, err := parseFrontmatter(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if name != "whitespace-name" {
		t.Errorf("expected trimmed name 'whitespace-name', got %q", name)
	}
	if desc != "Lots of whitespace" {
		t.Errorf("expected trimmed description 'Lots of whitespace', got %q", desc)
	}
}

func TestParseFrontmatter_MissingFrontmatter(t *testing.T) {
	content := `# Just markdown content
No frontmatter here.
`

	_, _, _, err := parseFrontmatter(content)

	if err == nil {
		t.Error("expected error for missing frontmatter")
	}
}

func TestParseFrontmatter_EmptyFrontmatter(t *testing.T) {
	content := `---
---
Content
`

	name, desc, _, err := parseFrontmatter(content)
	// Empty frontmatter parses successfully but will fail validation
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if name != "" || desc != "" {
		t.Error("expected empty name and description")
	}
}

func TestParseFrontmatter_UnknownFields(t *testing.T) {
	content := `---
name: test
description: Test
license: MIT
author: Someone
---
Content
`

	_, _, _, err := parseFrontmatter(content)

	if err == nil {
		t.Error("expected error for unknown fields")
	}
	if err != nil && err.Error() != "only 'name' and 'description' fields are allowed in frontmatter" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestParseFrontmatter_InvalidYAML(t *testing.T) {
	content := `---
name: test
description: [unclosed bracket
---
Content
`

	_, _, _, err := parseFrontmatter(content)

	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestParseFrontmatter_NoClosingDelimiter(t *testing.T) {
	content := `---
name: test
description: No closing delimiter
`

	_, _, _, err := parseFrontmatter(content)

	if err == nil {
		t.Error("expected error for missing closing delimiter")
	}
}

func TestValidateName_Valid(t *testing.T) {
	validNames := []string{
		"a",
		"a1",
		"aws-s3-security",
		"kubernetes-resource-limits",
		"test123",
		"a-b-c-1-2-3",
	}

	for _, name := range validNames {
		err := validateName(name)
		if err != nil {
			t.Errorf("expected %q to be valid, got error: %v", name, err)
		}
	}
}

func TestValidateName_Invalid(t *testing.T) {
	tests := []struct {
		name        string
		expectedErr string
	}{
		{"", "name is required"},
		{"   ", "name is required"},
		{"AWS-S3", "name must use kebab-case"},
		{"-leading-hyphen", "name must use kebab-case"},
		{"trailing-hyphen-", "name must use kebab-case"},
		{"123-starts-with-digit", "name must use kebab-case"},
		{"has_underscores", "name must use kebab-case"},
		{"has spaces", "name must use kebab-case"},
		{"Capital-Letter", "name must use kebab-case"},
		{string(make([]byte, 65)), "name must be 64 characters or less"}, // 65 chars
	}

	for _, tt := range tests {
		err := validateName(tt.name)
		if err == nil {
			t.Errorf("expected %q to be invalid", tt.name)
		} else if !strings.Contains(err.Error(), tt.expectedErr) {
			t.Errorf("for name %q, expected error containing %q, got %q", tt.name, tt.expectedErr, err.Error())
		}
	}
}

func TestValidateDescription_Valid(t *testing.T) {
	validDescs := []string{
		"A",
		"Short description",
		string(make([]byte, 1024)), // exactly 1024 chars
	}

	for _, desc := range validDescs {
		err := validateDescription(desc)
		if err != nil {
			t.Errorf("expected %q to be valid, got error: %v", desc, err)
		}
	}
}

func TestValidateDescription_Invalid(t *testing.T) {
	tests := []struct {
		desc        string
		expectedErr string
	}{
		{"", "description is required"},
		{"   ", "description is required"},
		{string(make([]byte, 1025)), "description must be 1024 characters or less"},
	}

	for _, tt := range tests {
		err := validateDescription(tt.desc)
		if err == nil {
			t.Errorf("expected description to be invalid")
		} else if !strings.Contains(err.Error(), tt.expectedErr) {
			t.Errorf("expected error containing %q, got %q", tt.expectedErr, err.Error())
		}
	}
}

func TestDiscover_Deduplication(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, "knowledge")
	err := os.Mkdir(knowledgeDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Create two files with same name
	writeFile(t, filepath.Join(knowledgeDir, "aws-s3.md"), `---
name: duplicate-name
description: First file
---
First
`)

	writeFile(t, filepath.Join(knowledgeDir, "s3-aws.md"), `---
name: duplicate-name
description: Second file
---
Second
`)

	files, warnings := Discover(knowledgeDir)

	if len(files) != 1 {
		t.Errorf("expected 1 file (first wins), got %d", len(files))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning for duplicate, got %d", len(warnings))
	}

	// First file (lexicographic order) should win
	if files[0].Description != "First file" {
		t.Errorf("expected first file to win, got description: %q", files[0].Description)
	}

	// Check warning message
	if !strings.Contains(warnings[0].Reason, "duplicate name") {
		t.Errorf("expected warning about duplicate name, got: %q", warnings[0].Reason)
	}
	if !strings.Contains(warnings[0].Reason, "aws-s3.md") {
		t.Errorf("expected warning to mention first file, got: %q", warnings[0].Reason)
	}
}

func TestDiscover_DuplicateInSubfolder(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, "knowledge")

	subdir := filepath.Join(knowledgeDir, "cloud")
	err := os.MkdirAll(subdir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Create files with same name in different folders
	writeFile(t, filepath.Join(knowledgeDir, "aws.md"), `---
name: aws-service
description: Root file
---
Root
`)

	writeFile(t, filepath.Join(subdir, "aws.md"), `---
name: aws-service
description: Subfolder file
---
Subfolder
`)

	files, warnings := Discover(knowledgeDir)

	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for duplicate, got %d", len(warnings))
	}
}

func TestDiscover_InvalidFilesProduceWarnings(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, "knowledge")
	err := os.Mkdir(knowledgeDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Invalid name
	writeFile(t, filepath.Join(knowledgeDir, "invalid-name.md"), `---
name: INVALID-NAME
description: Invalid name with uppercase
---
Content
`)

	// Missing description
	writeFile(t, filepath.Join(knowledgeDir, "no-desc.md"), `---
name: no-description
---
Content
`)

	// Invalid frontmatter
	writeFile(t, filepath.Join(knowledgeDir, "bad-yaml.md"), `Not yaml frontmatter
`)

	// Valid file
	writeFile(t, filepath.Join(knowledgeDir, "valid.md"), `---
name: valid-file
description: This one is valid
---
Content
`)

	files, warnings := Discover(knowledgeDir)

	if len(files) != 1 {
		t.Errorf("expected 1 valid file, got %d", len(files))
	}
	if len(warnings) != 3 {
		t.Fatalf("expected 3 warnings, got %d: %v", len(warnings), warnings)
	}

	// Check that all warnings have paths and reasons
	for _, w := range warnings {
		if w.Path == "" {
			t.Error("warning path should not be empty")
		}
		if w.Reason == "" {
			t.Error("warning reason should not be empty")
		}
	}
}

func TestDiscover_FileSizeLimit(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, "knowledge")
	err := os.Mkdir(knowledgeDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Create a file that exceeds the size limit
	// Generate content larger than 10MB
	largeContent := "---\nname: large-file\ndescription: Too large\n---\n"
	largeContent += strings.Repeat("x", 11*1024*1024) // 11MB of content

	writeFile(t, filepath.Join(knowledgeDir, "large.md"), largeContent)

	// Create a valid small file
	writeFile(t, filepath.Join(knowledgeDir, "small.md"), `---
name: small-file
description: Normal size
---
Content
`)

	files, warnings := Discover(knowledgeDir)

	if len(files) != 1 {
		t.Errorf("expected 1 valid file, got %d", len(files))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning for large file, got %d", len(warnings))
	}

	if !strings.Contains(warnings[0].Reason, "exceeds maximum") {
		t.Errorf("expected warning about file size, got: %q", warnings[0].Reason)
	}
}

func TestDiscover_LexicographicOrdering(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, "knowledge")
	err := os.Mkdir(knowledgeDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Create files in non-alphabetical order
	writeFile(t, filepath.Join(knowledgeDir, "zebra.md"), `---
name: z-file
description: Last alphabetically
---
Z
`)

	writeFile(t, filepath.Join(knowledgeDir, "apple.md"), `---
name: a-file
description: First alphabetically
---
A
`)

	writeFile(t, filepath.Join(knowledgeDir, "middle.md"), `---
name: m-file
description: Middle alphabetically
---
M
`)

	files, warnings := Discover(knowledgeDir)

	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files, got %d", len(files))
	}

	// Files should be processed in lexicographic order
	if files[0].Name != "a-file" {
		t.Errorf("expected first file to be 'a-file', got %q", files[0].Name)
	}
	if files[1].Name != "m-file" {
		t.Errorf("expected second file to be 'm-file', got %q", files[1].Name)
	}
	if files[2].Name != "z-file" {
		t.Errorf("expected third file to be 'z-file', got %q", files[2].Name)
	}
}

// FindKnowledgeDir tests

func TestFindKnowledgeDir_InCWD(t *testing.T) {
	root := t.TempDir()
	knowledgeDir := filepath.Join(root, ".overmind", "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result := FindKnowledgeDir(root)

	if result != knowledgeDir {
		t.Errorf("expected %q, got %q", knowledgeDir, result)
	}
}

func TestFindKnowledgeDir_InParent(t *testing.T) {
	root := t.TempDir()
	knowledgeDir := filepath.Join(root, ".overmind", "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	childDir := filepath.Join(root, "environments", "prod")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result := FindKnowledgeDir(childDir)

	if result != knowledgeDir {
		t.Errorf("expected %q, got %q", knowledgeDir, result)
	}
}

func TestFindKnowledgeDir_InGrandparent(t *testing.T) {
	root := t.TempDir()
	knowledgeDir := filepath.Join(root, ".overmind", "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	deepDir := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deepDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result := FindKnowledgeDir(deepDir)

	if result != knowledgeDir {
		t.Errorf("expected %q, got %q", knowledgeDir, result)
	}
}

func TestFindKnowledgeDir_StopsAtGitBoundary(t *testing.T) {
	root := t.TempDir()
	// Knowledge above the git boundary -- should NOT be found
	knowledgeDir := filepath.Join(root, ".overmind", "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Git repo is a subdirectory
	repoDir := filepath.Join(root, "my-repo")
	if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	workDir := filepath.Join(repoDir, "environments", "prod")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result := FindKnowledgeDir(workDir)

	if result != "" {
		t.Errorf("expected empty string (should not escape .git boundary), got %q", result)
	}
}

func TestFindKnowledgeDir_CWDTakesPriority(t *testing.T) {
	root := t.TempDir()
	// Knowledge at root
	rootKnowledge := filepath.Join(root, ".overmind", "knowledge")
	if err := os.MkdirAll(rootKnowledge, 0o755); err != nil {
		t.Fatal(err)
	}
	// Knowledge also in subdirectory
	childDir := filepath.Join(root, "sub")
	childKnowledge := filepath.Join(childDir, ".overmind", "knowledge")
	if err := os.MkdirAll(childKnowledge, 0o755); err != nil {
		t.Fatal(err)
	}

	result := FindKnowledgeDir(childDir)

	if result != childKnowledge {
		t.Errorf("expected CWD knowledge %q to take priority, got %q", childKnowledge, result)
	}
}

func TestFindKnowledgeDir_NotFoundAnywhere(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "some", "dir")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Place .git at root to create a boundary
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	result := FindKnowledgeDir(workDir)

	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestFindKnowledgeDir_GitBoundaryWithKnowledge(t *testing.T) {
	root := t.TempDir()
	// .git and .overmind/knowledge at the same level
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	knowledgeDir := filepath.Join(root, ".overmind", "knowledge")
	if err := os.MkdirAll(knowledgeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	workDir := filepath.Join(root, "environments", "prod")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result := FindKnowledgeDir(workDir)

	// Should find knowledge at repo root before the .git stop triggers
	if result != knowledgeDir {
		t.Errorf("expected %q, got %q", knowledgeDir, result)
	}
}

// Multi-directory tests

func TestResolveKnowledgeDirs_EmptyExplicit(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, ".overmind", "knowledge")
	err := os.MkdirAll(knowledgeDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Empty explicit dirs should fall back to auto-discovery
	result := ResolveKnowledgeDirs(dir, []string{})

	if len(result) != 1 {
		t.Fatalf("expected 1 directory, got %d", len(result))
	}
	if result[0] != knowledgeDir {
		t.Errorf("expected %q, got %q", knowledgeDir, result[0])
	}
}

func TestResolveKnowledgeDirs_ExplicitDirs(t *testing.T) {
	dir := t.TempDir()
	dir1 := filepath.Join(dir, "global", ".overmind", "knowledge")
	dir2 := filepath.Join(dir, "local", ".overmind", "knowledge")
	err := os.MkdirAll(dir1, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.MkdirAll(dir2, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	result := ResolveKnowledgeDirs(".", []string{dir1, dir2})

	if len(result) != 2 {
		t.Fatalf("expected 2 directories, got %d", len(result))
	}
}

func TestResolveKnowledgeDirs_MissingDirTolerated(t *testing.T) {
	dir := t.TempDir()
	existingDir := filepath.Join(dir, "existing")
	missingDir := filepath.Join(dir, "missing")
	err := os.Mkdir(existingDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	result := ResolveKnowledgeDirs(".", []string{existingDir, missingDir})

	if len(result) != 1 {
		t.Fatalf("expected 1 directory (missing should be skipped), got %d", len(result))
	}
	absExisting, _ := filepath.Abs(existingDir)
	if result[0] != absExisting {
		t.Errorf("expected %q, got %q", absExisting, result[0])
	}
}

func TestResolveKnowledgeDirs_AllMissing(t *testing.T) {
	dir := t.TempDir()
	missing1 := filepath.Join(dir, "missing1")
	missing2 := filepath.Join(dir, "missing2")

	result := ResolveKnowledgeDirs(".", []string{missing1, missing2})

	if len(result) != 0 {
		t.Errorf("expected 0 directories, got %d", len(result))
	}
}

func TestDiscover_MultipleDirectories(t *testing.T) {
	dir := t.TempDir()

	// Create global directory with one file
	globalDir := filepath.Join(dir, "global", ".overmind", "knowledge")
	err := os.MkdirAll(globalDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(globalDir, "global.md"), `---
name: global-file
description: Global knowledge file
---
Global content
`)

	// Create local directory with another file
	localDir := filepath.Join(dir, "local", ".overmind", "knowledge")
	err = os.MkdirAll(localDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(localDir, "local.md"), `---
name: local-file
description: Local knowledge file
---
Local content
`)

	files, warnings := Discover(globalDir, localDir)

	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d: %v", len(warnings), warnings)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	// Check both files are present
	names := make(map[string]bool)
	for _, f := range files {
		names[f.Name] = true
	}
	if !names["global-file"] {
		t.Error("expected global-file")
	}
	if !names["local-file"] {
		t.Error("expected local-file")
	}
}

func TestDiscover_CrossDirOverride(t *testing.T) {
	dir := t.TempDir()

	// Create global directory with a file
	globalDir := filepath.Join(dir, "global", ".overmind", "knowledge")
	err := os.MkdirAll(globalDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(globalDir, "shared.md"), `---
name: shared-config
description: Global version
---
Global content
`)

	// Create local directory with file of same name
	localDir := filepath.Join(dir, "local", ".overmind", "knowledge")
	err = os.MkdirAll(localDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(localDir, "shared.md"), `---
name: shared-config
description: Local override
---
Local content
`)

	files, warnings := Discover(globalDir, localDir)

	// Should have exactly 1 file (local overrides global)
	if len(files) != 1 {
		t.Fatalf("expected 1 file (local should override global), got %d", len(files))
	}

	// Cross-directory override is logged but not added to warnings
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings (cross-dir override is logged only), got %d", len(warnings))
	}

	// The local version should win
	if files[0].Description != "Local override" {
		t.Errorf("expected local version to win, got description: %q", files[0].Description)
	}
	if files[0].Content != "Local content\n" {
		t.Errorf("expected local content, got: %q", files[0].Content)
	}

	// Check SourceDir is set correctly
	absLocalDir, _ := filepath.Abs(localDir)
	if files[0].SourceDir != absLocalDir {
		t.Errorf("expected SourceDir %q, got %q", absLocalDir, files[0].SourceDir)
	}
}

func TestDiscover_WithinDirDuplicateStillWarns(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, ".overmind", "knowledge")
	err := os.MkdirAll(knowledgeDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}

	// Create two files with same name in the same directory
	writeFile(t, filepath.Join(knowledgeDir, "file1.md"), `---
name: duplicate-name
description: First
---
First
`)
	writeFile(t, filepath.Join(knowledgeDir, "file2.md"), `---
name: duplicate-name
description: Second
---
Second
`)

	files, warnings := Discover(knowledgeDir)

	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning for within-dir duplicate, got %d", len(warnings))
	}
}

func TestDiscover_MixedExistingAndMissing(t *testing.T) {
	dir := t.TempDir()

	existingDir := filepath.Join(dir, "existing")
	err := os.Mkdir(existingDir, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(existingDir, "test.md"), `---
name: test-file
description: Test
---
Content
`)

	missingDir := filepath.Join(dir, "missing")

	// Should silently skip missing directory
	files, warnings := Discover(existingDir, missingDir)

	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings (missing dir skipped), got %d", len(warnings))
	}
}

func TestDiscover_DeterministicOrdering(t *testing.T) {
	dir := t.TempDir()

	dir1 := filepath.Join(dir, "dir1")
	err := os.Mkdir(dir1, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir1, "a.md"), `---
name: file-a
description: A
---
A
`)

	dir2 := filepath.Join(dir, "dir2")
	err = os.Mkdir(dir2, 0o755)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir2, "b.md"), `---
name: file-b
description: B
---
B
`)

	// Run multiple times to ensure deterministic ordering
	for i := range 3 {
		files, _ := Discover(dir1, dir2)
		if len(files) != 2 {
			t.Fatalf("iteration %d: expected 2 files, got %d", i, len(files))
		}
		// Files from each directory are sorted lexicographically, then combined
		// Since both files are in different directories, they should appear in order
		if files[0].Name != "file-a" || files[1].Name != "file-b" {
			t.Errorf("iteration %d: unexpected order: %s, %s", i, files[0].Name, files[1].Name)
		}
	}
}

func TestDiscover_EmptyList(t *testing.T) {
	files, warnings := Discover()

	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(warnings))
	}
}

// Helper functions

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0o644)
	if err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}
