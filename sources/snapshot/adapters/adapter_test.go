package adapters

import (
	"context"
	"errors"
	"testing"

	"github.com/overmindtech/cli/go/sdp-go"
)

func createTestAdapters(t *testing.T) map[string]*SnapshotAdapter {
	t.Helper()
	snapshot := createTestSnapshot()
	index, err := NewSnapshotIndex(snapshot)
	if err != nil {
		t.Fatalf("Failed to create test index: %v", err)
	}

	adapters := make(map[string]*SnapshotAdapter)
	for _, typ := range index.GetAllTypes() {
		scopes := index.GetScopesForType(typ)
		adapters[typ] = NewSnapshotAdapter(index, typ, scopes)
	}
	return adapters
}

func TestAdapterType(t *testing.T) {
	adapters := createTestAdapters(t)

	ec2 := adapters["ec2-instance"]
	if ec2.Type() != "ec2-instance" {
		t.Errorf("Expected type 'ec2-instance', got '%s'", ec2.Type())
	}

	s3 := adapters["s3-bucket"]
	if s3.Type() != "s3-bucket" {
		t.Errorf("Expected type 's3-bucket', got '%s'", s3.Type())
	}
}

func TestAdapterName(t *testing.T) {
	adapters := createTestAdapters(t)

	if adapters["ec2-instance"].Name() != "snapshot-ec2-instance" {
		t.Errorf("Expected name 'snapshot-ec2-instance', got '%s'", adapters["ec2-instance"].Name())
	}
}

func TestAdapterScopes(t *testing.T) {
	adapters := createTestAdapters(t)

	ec2Scopes := adapters["ec2-instance"].Scopes()
	if len(ec2Scopes) != 2 {
		t.Fatalf("Expected 2 scopes for ec2-instance, got %d: %v", len(ec2Scopes), ec2Scopes)
	}
	scopeSet := map[string]bool{}
	for _, s := range ec2Scopes {
		scopeSet[s] = true
	}
	if !scopeSet["us-east-1"] || !scopeSet["us-west-2"] {
		t.Errorf("Expected scopes [us-east-1, us-west-2], got %v", ec2Scopes)
	}

	s3Scopes := adapters["s3-bucket"].Scopes()
	if len(s3Scopes) != 1 || s3Scopes[0] != "global" {
		t.Errorf("Expected scopes [global], got %v", s3Scopes)
	}
}

func TestAdapterGet(t *testing.T) {
	adapters := createTestAdapters(t)
	ec2 := adapters["ec2-instance"]
	ctx := context.Background()

	// Get by unique attribute value with wildcard scope
	item, err := ec2.Get(ctx, "*", "i-12345", false)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if item == nil || item.UniqueAttributeValue() != "i-12345" {
		t.Errorf("Expected 'i-12345', got '%v'", item)
	}

	// Get by GUN
	item, err = ec2.Get(ctx, "*", "us-east-1.ec2-instance.i-12345", false)
	if err != nil {
		t.Fatalf("Get by GUN failed: %v", err)
	}
	if item == nil {
		t.Fatal("Expected item by GUN, got nil")
	}

	// Get with specific scope
	item, err = ec2.Get(ctx, "us-east-1", "i-12345", false)
	if err != nil {
		t.Fatalf("Get with specific scope failed: %v", err)
	}
	if item == nil {
		t.Fatal("Expected item, got nil")
	}

	// Not found
	_, err = ec2.Get(ctx, "*", "nonexistent", false)
	if err == nil {
		t.Error("Expected error for non-existent item")
	}
	var queryErr *sdp.QueryError
	if !errors.As(err, &queryErr) || queryErr.GetErrorType() != sdp.QueryError_NOTFOUND {
		t.Errorf("Expected NOTFOUND, got %v", err)
	}

	// Scope mismatch: requesting us-west-2 for an item in us-east-1
	_, err = ec2.Get(ctx, "us-west-2", "us-east-1.ec2-instance.i-12345", false)
	if err == nil {
		t.Fatal("Expected error when scope doesn't match GUN scope")
	}
	if !errors.As(err, &queryErr) || queryErr.GetErrorType() != sdp.QueryError_NOTFOUND {
		t.Errorf("Expected NOTFOUND, got %v", err)
	}

	// Same GUN with matching scope works
	item, err = ec2.Get(ctx, "us-east-1", "us-east-1.ec2-instance.i-12345", false)
	if err != nil || item == nil || item.GetScope() != "us-east-1" {
		t.Errorf("Get with matching scope should work: err=%v item=%v", err, item)
	}

	// Cross-type: ec2 adapter should not return s3-bucket items
	_, err = ec2.Get(ctx, "*", "my-test-bucket", false)
	if err == nil {
		t.Error("ec2 adapter should not find s3-bucket items")
	}
}

func TestAdapterList(t *testing.T) {
	adapters := createTestAdapters(t)
	ctx := context.Background()

	// ec2 adapter lists its 2 items
	items, err := adapters["ec2-instance"].List(ctx, "*", false)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("Expected 2 ec2-instance items, got %d", len(items))
	}

	// Verify linked items are preserved
	var ec2East *sdp.Item
	for _, item := range items {
		if item.GloballyUniqueName() == "us-east-1.ec2-instance.i-12345" {
			ec2East = item
			break
		}
	}
	if ec2East == nil {
		t.Fatal("Expected to find ec2 instance i-12345")
	}
	linked := ec2East.GetLinkedItems()
	if len(linked) != 1 {
		t.Fatalf("Expected 1 linked item, got %d", len(linked))
	}
	ref := linked[0].GetItem()
	if ref.GetType() != "s3-bucket" || ref.GetUniqueAttributeValue() != "my-test-bucket" {
		t.Errorf("Unexpected linked item reference: %v", ref)
	}

	// List with specific scope
	items, err = adapters["ec2-instance"].List(ctx, "us-east-1", false)
	if err != nil {
		t.Fatalf("List with specific scope failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("Expected 1 item for us-east-1, got %d", len(items))
	}

	// s3 adapter lists its 1 item
	items, err = adapters["s3-bucket"].List(ctx, "*", false)
	if err != nil {
		t.Fatalf("s3 List failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("Expected 1 s3-bucket item, got %d", len(items))
	}

	// List with nonexistent scope
	items, err = adapters["ec2-instance"].List(ctx, "nonexistent", false)
	if err != nil {
		t.Fatalf("List with nonexistent scope failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(items))
	}
}

func TestAdapterSearch(t *testing.T) {
	adapters := createTestAdapters(t)
	ec2 := adapters["ec2-instance"]
	ctx := context.Background()

	// Search matching both ec2 instances (neighbor s3-bucket is different type, not included)
	items, err := ec2.Search(ctx, "*", ".*ec2-instance.*", false)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("Expected 2 ec2-instance items, got %d", len(items))
	}

	// Search with specific scope
	items, err = ec2.Search(ctx, "us-east-1", ".*ec2-instance.*", false)
	if err != nil {
		t.Fatalf("Search with specific scope failed: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("Expected 1 item in us-east-1, got %d", len(items))
	}

	// Search that matches nothing
	items, err = ec2.Search(ctx, "*", "nonexistent-xyz", false)
	if err != nil {
		t.Fatalf("Search no match failed: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("Expected 0 items, got %d", len(items))
	}

	// Invalid regex
	_, err = ec2.Search(ctx, "*", "[invalid(regex", false)
	if err == nil {
		t.Error("Expected error for invalid regex")
	}
	var queryErr *sdp.QueryError
	if !errors.As(err, &queryErr) || queryErr.GetErrorType() != sdp.QueryError_OTHER {
		t.Errorf("Expected OTHER error, got %v", err)
	}
}

func TestAdapterMetadata(t *testing.T) {
	adapters := createTestAdapters(t)

	// ec2-instance should get metadata from the catalog
	ec2Meta := adapters["ec2-instance"].Metadata()
	if ec2Meta == nil {
		t.Fatal("Expected metadata, got nil")
	}
	if ec2Meta.GetType() != "ec2-instance" {
		t.Errorf("Expected type 'ec2-instance', got '%s'", ec2Meta.GetType())
	}
	if ec2Meta.GetCategory() != sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION {
		t.Errorf("Expected COMPUTE_APPLICATION category, got %v", ec2Meta.GetCategory())
	}
	if ec2Meta.GetDescriptiveName() != "EC2 Instance" {
		t.Errorf("Expected descriptive name 'EC2 Instance', got '%s'", ec2Meta.GetDescriptiveName())
	}

	methods := ec2Meta.GetSupportedQueryMethods()
	if !methods.GetGet() || !methods.GetList() || !methods.GetSearch() {
		t.Error("Expected all query methods to be supported for ec2-instance")
	}

	// s3-bucket should also get catalog metadata
	s3Meta := adapters["s3-bucket"].Metadata()
	if s3Meta.GetType() != "s3-bucket" {
		t.Errorf("Expected type 's3-bucket', got '%s'", s3Meta.GetType())
	}
}

func TestNewSnapshotAdapter(t *testing.T) {
	snapshot := createTestSnapshot()
	index, _ := NewSnapshotIndex(snapshot)

	adapter := NewSnapshotAdapter(index, "ec2-instance", []string{"us-east-1", "us-west-2"})
	if adapter == nil {
		t.Fatal("Expected adapter, got nil")
		return
	}
	if adapter.index != index {
		t.Error("Expected adapter to store index reference")
	}
	if adapter.itemType != "ec2-instance" {
		t.Errorf("Expected type 'ec2-instance', got '%s'", adapter.itemType)
	}
}

func TestAdapterMetadataFallback(t *testing.T) {
	snapshot := createTestSnapshot()
	index, _ := NewSnapshotIndex(snapshot)

	// Use a type not in the catalog to test fallback
	adapter := NewSnapshotAdapter(index, "unknown-type-xyz", []string{"test"})
	meta := adapter.Metadata()
	if meta.GetType() != "unknown-type-xyz" {
		t.Errorf("Expected type 'unknown-type-xyz', got '%s'", meta.GetType())
	}
	if meta.GetCategory() != sdp.AdapterCategory_ADAPTER_CATEGORY_OTHER {
		t.Errorf("Expected OTHER category for unknown type, got %v", meta.GetCategory())
	}
}
