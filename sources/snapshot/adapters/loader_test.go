package adapters

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/overmindtech/cli/go/sdp-go"
	"google.golang.org/protobuf/proto"
)

func TestLoadSnapshotFromFile(t *testing.T) {
	// Create a test snapshot
	attrs, _ := sdp.ToAttributesViaJson(map[string]interface{}{
		"name": "test-item",
	})

	snapshot := &sdp.Snapshot{
		Properties: &sdp.SnapshotProperties{
			Name: "test-snapshot",
			Items: []*sdp.Item{
				{
					Type:            "test-type",
					UniqueAttribute: "name",
					Attributes:      attrs,
					Scope:           "test-scope",
				},
			},
		},
	}

	// Marshal to bytes
	data, err := proto.Marshal(snapshot)
	if err != nil {
		t.Fatalf("Failed to marshal test snapshot: %v", err)
	}

	// Write to temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-snapshot.pb")
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		t.Fatalf("Failed to write test snapshot file: %v", err)
	}

	// Test loading
	ctx := context.Background()
	loaded, err := LoadSnapshot(ctx, tmpFile)
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}

	if loaded.GetProperties().GetName() != "test-snapshot" {
		t.Errorf("Expected snapshot name 'test-snapshot', got '%s'", loaded.GetProperties().GetName())
	}

	if len(loaded.GetProperties().GetItems()) != 1 {
		t.Errorf("Expected 1 item, got %d", len(loaded.GetProperties().GetItems()))
	}
}

func TestLoadSnapshotFromURL(t *testing.T) {
	// Create a test snapshot
	attrs, _ := sdp.ToAttributesViaJson(map[string]interface{}{
		"name": "test-item",
	})

	snapshot := &sdp.Snapshot{
		Properties: &sdp.SnapshotProperties{
			Name: "test-snapshot-url",
			Items: []*sdp.Item{
				{
					Type:            "test-type",
					UniqueAttribute: "name",
					Attributes:      attrs,
					Scope:           "test-scope",
				},
			},
		},
	}

	// Marshal to bytes
	data, err := proto.Marshal(snapshot)
	if err != nil {
		t.Fatalf("Failed to marshal test snapshot: %v", err)
	}

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}))
	defer server.Close()

	// Test loading from URL
	ctx := context.Background()
	loaded, err := LoadSnapshot(ctx, server.URL)
	if err != nil {
		t.Fatalf("LoadSnapshot from URL failed: %v", err)
	}

	if loaded.GetProperties().GetName() != "test-snapshot-url" {
		t.Errorf("Expected snapshot name 'test-snapshot-url', got '%s'", loaded.GetProperties().GetName())
	}
}

func TestLoadSnapshotEmptyItems(t *testing.T) {
	// Create a snapshot with no items
	snapshot := &sdp.Snapshot{
		Properties: &sdp.SnapshotProperties{
			Name:  "empty-snapshot",
			Items: []*sdp.Item{},
		},
	}

	// Marshal to bytes
	data, err := proto.Marshal(snapshot)
	if err != nil {
		t.Fatalf("Failed to marshal test snapshot: %v", err)
	}

	// Write to temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty-snapshot.pb")
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		t.Fatalf("Failed to write test snapshot file: %v", err)
	}

	// Test loading - should fail validation
	ctx := context.Background()
	_, err = LoadSnapshot(ctx, tmpFile)
	if err == nil {
		t.Error("Expected error for snapshot with no items, got nil")
	}
}

func TestLoadSnapshotFileNotFound(t *testing.T) {
	ctx := context.Background()
	_, err := LoadSnapshot(ctx, "/nonexistent/file.pb")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestLoadSnapshotInvalidProtobuf(t *testing.T) {
	// Write invalid protobuf data
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.pb")
	if err := os.WriteFile(tmpFile, []byte("invalid protobuf data"), 0o644); err != nil {
		t.Fatalf("Failed to write invalid data: %v", err)
	}

	// Test loading - should fail
	ctx := context.Background()
	_, err := LoadSnapshot(ctx, tmpFile)
	if err == nil {
		t.Error("Expected error for invalid protobuf, got nil")
	}
}
