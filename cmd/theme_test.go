package cmd

import (
	"strings"
	"testing"
)

func TestMarkdownToString(t *testing.T) {
	InitPalette()

	markdown := `# some random markdown`
	got := markdownToString(0, markdown)

	if !strings.Contains(got, "some random") || !strings.Contains(got, "markdown") {
		t.Fatalf("expected rendered markdown to contain heading text, got %q", got)
	}

	// glamour v2 uses SGR reset (\x1b[m) instead of v1's explicit reset (\x1b[0m).
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("expected ANSI styling sequences in rendered output, got %q", got)
	}
}
