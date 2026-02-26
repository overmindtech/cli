package adapters

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/overmindtech/cli/go/sdp-go"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

// LoadSnapshot loads a snapshot from a URL or local file path
func LoadSnapshot(ctx context.Context, source string) (*sdp.Snapshot, error) {
	var data []byte
	var err error

	// Determine if source is a URL or file path
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		log.WithField("url", source).Info("Loading snapshot from URL")
		data, err = loadSnapshotFromURL(ctx, source)
		if err != nil {
			return nil, fmt.Errorf("failed to load snapshot from URL: %w", err)
		}
	} else {
		log.WithField("path", source).Info("Loading snapshot from file")
		data, err = loadSnapshotFromFile(source)
		if err != nil {
			return nil, fmt.Errorf("failed to load snapshot from file: %w", err)
		}
	}

	// Unmarshal the protobuf data
	snapshot := &sdp.Snapshot{}
	if err := proto.Unmarshal(data, snapshot); err != nil {
		return nil, fmt.Errorf("failed to unmarshal snapshot protobuf: %w", err)
	}

	// Validate snapshot has items
	if snapshot.GetProperties() == nil || len(snapshot.GetProperties().GetItems()) == 0 {
		return nil, fmt.Errorf("snapshot has no items")
	}

	log.WithFields(log.Fields{
		"items": len(snapshot.GetProperties().GetItems()),
		"edges": len(snapshot.GetProperties().GetEdges()),
	}).Info("Snapshot loaded successfully")

	return snapshot, nil
}

// loadSnapshotFromURL loads snapshot data from an HTTP(S) URL
func loadSnapshotFromURL(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req) //nolint:gosec // G107 (SSRF): URL comes from operator-supplied snapshot source config, not from untrusted network input
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// loadSnapshotFromFile loads snapshot data from a local file
func loadSnapshotFromFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}
