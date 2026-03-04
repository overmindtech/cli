package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderKnowledgeList_NoKnowledgeDir(t *testing.T) {
	dir := t.TempDir()

	output, err := renderKnowledgeList(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "No .overmind/knowledge/ directory found") {
		t.Errorf("expected message about no directory found, got: %s", output)
	}
	if !strings.Contains(output, "Create a .overmind/knowledge/ directory") {
		t.Errorf("expected helpful message about creating directory, got: %s", output)
	}
	if !strings.Contains(output, "terraform plan") {
		t.Errorf("expected reference to terraform plan, got: %s", output)
	}
}

func TestRenderKnowledgeList_EmptyKnowledgeDir(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, ".overmind", "knowledge")
	err := os.MkdirAll(knowledgeDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	output, err := renderKnowledgeList(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "Knowledge directory:") {
		t.Errorf("expected resolved directory message, got: %s", output)
	}
	if !strings.Contains(output, knowledgeDir) {
		t.Errorf("expected directory path %s in output, got: %s", knowledgeDir, output)
	}
	if !strings.Contains(output, "No knowledge files found") {
		t.Errorf("expected 'No knowledge files found' message, got: %s", output)
	}
}

func TestRenderKnowledgeList_ValidFiles(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, ".overmind", "knowledge")
	err := os.MkdirAll(knowledgeDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create valid knowledge files
	writeTestFile(t, filepath.Join(knowledgeDir, "aws-s3.md"), `---
name: aws-s3-security
description: Security best practices for S3 buckets
---
# AWS S3 Security
Content here.
`)

	subdir := filepath.Join(knowledgeDir, "cloud")
	err = os.Mkdir(subdir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(subdir, "gcp.md"), `---
name: gcp-compute
description: GCP Compute Engine guidelines
---
# GCP Compute
Content here.
`)

	output, err := renderKnowledgeList(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check for resolved directory
	if !strings.Contains(output, "Knowledge directory:") {
		t.Errorf("expected resolved directory message, got: %s", output)
	}
	if !strings.Contains(output, knowledgeDir) {
		t.Errorf("expected directory path in output, got: %s", output)
	}

	// Check for header
	if !strings.Contains(output, "Valid Knowledge Files") {
		t.Errorf("expected 'Valid Knowledge Files' header, got: %s", output)
	}

	// Check for first file details
	if !strings.Contains(output, "aws-s3-security") {
		t.Errorf("expected file name 'aws-s3-security', got: %s", output)
	}
	if !strings.Contains(output, "Security best practices for S3 buckets") {
		t.Errorf("expected file description, got: %s", output)
	}
	if !strings.Contains(output, "aws-s3.md") {
		t.Errorf("expected file path 'aws-s3.md', got: %s", output)
	}

	// Check for second file details
	if !strings.Contains(output, "gcp-compute") {
		t.Errorf("expected file name 'gcp-compute', got: %s", output)
	}
	if !strings.Contains(output, "GCP Compute Engine guidelines") {
		t.Errorf("expected file description, got: %s", output)
	}
	if !strings.Contains(output, filepath.Join("cloud", "gcp.md")) {
		t.Errorf("expected file path 'cloud/gcp.md', got: %s", output)
	}
}

func TestRenderKnowledgeList_InvalidFiles(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, ".overmind", "knowledge")
	err := os.MkdirAll(knowledgeDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create valid file
	writeTestFile(t, filepath.Join(knowledgeDir, "valid.md"), `---
name: valid-file
description: A valid knowledge file
---
Content here.
`)

	// Create invalid file (missing frontmatter)
	writeTestFile(t, filepath.Join(knowledgeDir, "invalid.md"), `# No frontmatter
This file is missing frontmatter.
`)

	output, err := renderKnowledgeList(dir)
	if err == nil {
		t.Fatal("expected error when invalid files present, got nil")
	}
	if !errors.Is(err, ErrInvalidKnowledgeFiles) {
		t.Errorf("expected ErrInvalidKnowledgeFiles, got: %v", err)
	}

	// Check for valid file
	if !strings.Contains(output, "Valid Knowledge Files") {
		t.Errorf("expected 'Valid Knowledge Files' header, got: %s", output)
	}
	if !strings.Contains(output, "valid-file") {
		t.Errorf("expected valid file name, got: %s", output)
	}

	// Check for warnings section
	if !strings.Contains(output, "Invalid/Skipped Files") {
		t.Errorf("expected 'Invalid/Skipped Files' header, got: %s", output)
	}
	if !strings.Contains(output, "invalid.md") {
		t.Errorf("expected invalid file path in warnings, got: %s", output)
	}
	if !strings.Contains(output, "Reason:") {
		t.Errorf("expected reason in warnings, got: %s", output)
	}
}

func TestRenderKnowledgeList_OnlyInvalidFiles(t *testing.T) {
	dir := t.TempDir()
	knowledgeDir := filepath.Join(dir, ".overmind", "knowledge")
	err := os.MkdirAll(knowledgeDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create only invalid files
	writeTestFile(t, filepath.Join(knowledgeDir, "bad1.md"), `# No frontmatter`)
	writeTestFile(t, filepath.Join(knowledgeDir, "bad2.md"), `---
name: invalid name with spaces
description: This has an invalid name
---
Content.
`)

	output, err := renderKnowledgeList(dir)
	if err == nil {
		t.Fatal("expected error when only invalid files present, got nil")
	}
	if !errors.Is(err, ErrInvalidKnowledgeFiles) {
		t.Errorf("expected ErrInvalidKnowledgeFiles, got: %v", err)
	}

	// Should NOT have valid files section
	if strings.Contains(output, "Valid Knowledge Files") {
		t.Errorf("should not have 'Valid Knowledge Files' header when all files are invalid, got: %s", output)
	}

	// Should have warnings
	if !strings.Contains(output, "Invalid/Skipped Files") {
		t.Errorf("expected 'Invalid/Skipped Files' header, got: %s", output)
	}
	if !strings.Contains(output, "bad1.md") {
		t.Errorf("expected bad1.md in warnings, got: %s", output)
	}
	if !strings.Contains(output, "bad2.md") {
		t.Errorf("expected bad2.md in warnings, got: %s", output)
	}
}

func TestRenderKnowledgeList_SubdirectoryUsesLocal(t *testing.T) {
	dir := t.TempDir()

	// Create parent knowledge dir
	parentKnowledgeDir := filepath.Join(dir, ".overmind", "knowledge")
	err := os.MkdirAll(parentKnowledgeDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(parentKnowledgeDir, "parent.md"), `---
name: parent-file
description: Parent knowledge file
---
Content.
`)

	// Create subdirectory with its own knowledge dir
	childDir := filepath.Join(dir, "child")
	childKnowledgeDir := filepath.Join(childDir, ".overmind", "knowledge")
	err = os.MkdirAll(childKnowledgeDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(childKnowledgeDir, "child.md"), `---
name: child-file
description: Child knowledge file
---
Content.
`)

	output, err := renderKnowledgeList(childDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use child knowledge dir
	if !strings.Contains(output, childKnowledgeDir) {
		t.Errorf("expected child knowledge dir %s, got: %s", childKnowledgeDir, output)
	}
	if strings.Contains(output, parentKnowledgeDir) {
		t.Errorf("should not mention parent knowledge dir, got: %s", output)
	}

	// Should show child file, not parent file
	if !strings.Contains(output, "child-file") {
		t.Errorf("expected child file, got: %s", output)
	}
	if strings.Contains(output, "parent-file") {
		t.Errorf("should not show parent file, got: %s", output)
	}
}

func TestRenderKnowledgeList_SubdirectoryUsesParent(t *testing.T) {
	dir := t.TempDir()

	// Create parent knowledge dir
	parentKnowledgeDir := filepath.Join(dir, ".overmind", "knowledge")
	err := os.MkdirAll(parentKnowledgeDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(parentKnowledgeDir, "parent.md"), `---
name: parent-file
description: Parent knowledge file
---
Content.
`)

	// Create subdirectory WITHOUT its own knowledge dir
	childDir := filepath.Join(dir, "child")
	err = os.Mkdir(childDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	output, err := renderKnowledgeList(childDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should use parent knowledge dir
	if !strings.Contains(output, parentKnowledgeDir) {
		t.Errorf("expected parent knowledge dir %s, got: %s", parentKnowledgeDir, output)
	}

	// Should show parent file
	if !strings.Contains(output, "parent-file") {
		t.Errorf("expected parent file, got: %s", output)
	}
}

func TestRenderKnowledgeList_StopsAtGitBoundary(t *testing.T) {
	dir := t.TempDir()

	// Create outer directory with knowledge (outside git repo)
	outerKnowledgeDir := filepath.Join(dir, ".overmind", "knowledge")
	err := os.MkdirAll(outerKnowledgeDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(outerKnowledgeDir, "outer.md"), `---
name: outer-file
description: Knowledge file outside git repo
---
Content.
`)

	// Create a git repo subdirectory
	repoDir := filepath.Join(dir, "my-repo")
	repoGitDir := filepath.Join(repoDir, ".git")
	err = os.MkdirAll(repoGitDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Create a workspace dir inside the repo (without its own knowledge)
	workspaceDir := filepath.Join(repoDir, "workspace")
	err = os.Mkdir(workspaceDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	output, err := renderKnowledgeList(workspaceDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should NOT find outer knowledge dir (stops at .git boundary)
	if !strings.Contains(output, "No .overmind/knowledge/ directory found") {
		t.Errorf("expected no knowledge dir found (should stop at .git), got: %s", output)
	}
	if strings.Contains(output, "outer-file") {
		t.Errorf("should not find knowledge from outside git repo, got: %s", output)
	}
}

func TestTruncateDescription(t *testing.T) {
	tests := []struct {
		name     string
		desc     string
		maxLen   int
		expected string
	}{
		{
			name:     "short description",
			desc:     "Short",
			maxLen:   20,
			expected: "Short",
		},
		{
			name:     "exact length",
			desc:     "Exactly twenty char",
			maxLen:   20,
			expected: "Exactly twenty char",
		},
		{
			name:     "needs truncation",
			desc:     "This is a very long description that needs to be truncated",
			maxLen:   20,
			expected: "This is a very lo...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateDescription(tt.desc, tt.maxLen)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
			if len(result) > tt.maxLen {
				t.Errorf("result length %d exceeds maxLen %d", len(result), tt.maxLen)
			}
		})
	}
}

// Helper function for tests
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}
