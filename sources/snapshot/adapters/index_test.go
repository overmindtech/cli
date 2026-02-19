package adapters

import (
	"testing"

	"github.com/overmindtech/cli/go/sdp-go"
)

func createTestSnapshot() *sdp.Snapshot {
	attrs1, _ := sdp.ToAttributesViaJson(map[string]interface{}{
		"instanceId": "i-12345",
		"name":       "test-instance",
	})
	attrs2, _ := sdp.ToAttributesViaJson(map[string]interface{}{
		"instanceId": "i-67890",
		"name":       "test-instance-2",
	})
	attrs3, _ := sdp.ToAttributesViaJson(map[string]interface{}{
		"bucketName": "my-test-bucket",
	})

	return &sdp.Snapshot{
		Properties: &sdp.SnapshotProperties{
			Name: "test-snapshot",
			Items: []*sdp.Item{
				{
					Type:            "ec2-instance",
					UniqueAttribute: "instanceId",
					Attributes:      attrs1,
					Scope:           "us-east-1",
				},
				{
					Type:            "ec2-instance",
					UniqueAttribute: "instanceId",
					Attributes:      attrs2,
					Scope:           "us-west-2",
				},
				{
					Type:            "s3-bucket",
					UniqueAttribute: "bucketName",
					Attributes:      attrs3,
					Scope:           "global",
				},
			},
			Edges: []*sdp.Edge{
				{
					From: &sdp.Reference{
						Type:                 "ec2-instance",
						UniqueAttributeValue: "i-12345",
						Scope:                "us-east-1",
					},
					To: &sdp.Reference{
						Type:                 "s3-bucket",
						UniqueAttributeValue: "my-test-bucket",
						Scope:                "global",
					},
				},
			},
		},
	}
}

func TestNewSnapshotIndex(t *testing.T) {
	snapshot := createTestSnapshot()

	index, err := NewSnapshotIndex(snapshot)
	if err != nil {
		t.Fatalf("NewSnapshotIndex failed: %v", err)
	}

	if index == nil {
		t.Fatal("Expected index to be non-nil")
	}

	// Verify all items are indexed
	allItems := index.GetAllItems()
	if len(allItems) != 3 {
		t.Errorf("Expected 3 items, got %d", len(allItems))
	}

	// Verify edges are stored
	if len(index.edges) != 1 {
		t.Errorf("Expected 1 edge, got %d", len(index.edges))
	}
}

func TestLinkedItemsHydrated(t *testing.T) {
	snapshot := createTestSnapshot()
	index, err := NewSnapshotIndex(snapshot)
	if err != nil {
		t.Fatalf("NewSnapshotIndex failed: %v", err)
	}

	// The ec2-instance i-12345 is the From side of the edge to s3-bucket
	ec2 := index.GetByGUN("us-east-1.ec2-instance.i-12345")
	if ec2 == nil {
		t.Fatal("expected to find ec2 instance")
	}
	linked := ec2.GetLinkedItems()
	if len(linked) != 1 {
		t.Fatalf("Expected 1 linked item on ec2 instance, got %d", len(linked))
	}
	ref := linked[0].GetItem()
	if ref.GetType() != "s3-bucket" || ref.GetUniqueAttributeValue() != "my-test-bucket" || ref.GetScope() != "global" {
		t.Errorf("Unexpected linked item reference: %v", ref)
	}

	// The s3-bucket is only on the To side of the edge, so it should have no LinkedItems
	bucket := index.GetByGUN("global.s3-bucket.my-test-bucket")
	if bucket == nil {
		t.Fatal("expected to find s3 bucket")
	}
	if len(bucket.GetLinkedItems()) != 0 {
		t.Errorf("Expected 0 linked items on bucket (it is only a To target), got %d", len(bucket.GetLinkedItems()))
	}

	// The us-west-2 instance has no edges at all
	ec2West := index.GetByGUN("us-west-2.ec2-instance.i-67890")
	if ec2West == nil {
		t.Fatal("expected to find us-west-2 ec2 instance")
	}
	if len(ec2West.GetLinkedItems()) != 0 {
		t.Errorf("Expected 0 linked items on us-west-2 instance, got %d", len(ec2West.GetLinkedItems()))
	}
}

func TestGetByGUN(t *testing.T) {
	snapshot := createTestSnapshot()
	index, _ := NewSnapshotIndex(snapshot)

	// Test getting item by GUN
	gun := "us-east-1.ec2-instance.i-12345"
	item := index.GetByGUN(gun)
	if item == nil {
		t.Fatalf("Expected to find item with GUN %s", gun)
	}

	if item.UniqueAttributeValue() != "i-12345" {
		t.Errorf("Expected unique attribute 'i-12345', got '%s'", item.UniqueAttributeValue())
	}

	// Test non-existent GUN
	item = index.GetByGUN("nonexistent.type.query")
	if item != nil {
		t.Error("Expected nil for non-existent GUN")
	}
}

func TestGetByReference(t *testing.T) {
	snapshot := createTestSnapshot()
	index, _ := NewSnapshotIndex(snapshot)

	// Test getting item by reference
	ref := &sdp.Reference{
		Type:                 "ec2-instance",
		UniqueAttributeValue: "i-12345",
		Scope:                "us-east-1",
	}

	item := index.GetByReference(ref)
	if item == nil {
		t.Fatal("Expected to find item by reference")
	}

	if item.UniqueAttributeValue() != "i-12345" {
		t.Errorf("Expected unique attribute 'i-12345', got '%s'", item.UniqueAttributeValue())
	}
}

func TestGetAllTypes(t *testing.T) {
	snapshot := createTestSnapshot()
	index, _ := NewSnapshotIndex(snapshot)

	types := index.GetAllTypes()
	if len(types) != 2 {
		t.Errorf("Expected 2 unique types, got %d", len(types))
	}

	// Verify expected types exist
	typeMap := make(map[string]bool)
	for _, itemType := range types {
		typeMap[itemType] = true
	}

	expectedTypes := []string{"ec2-instance", "s3-bucket"}
	for _, expected := range expectedTypes {
		if !typeMap[expected] {
			t.Errorf("Expected type '%s' not found", expected)
		}
	}
}

func TestEdgesFromAndEdgesTo(t *testing.T) {
	snapshot := createTestSnapshot()
	index, _ := NewSnapshotIndex(snapshot)

	refFrom := &sdp.Reference{
		Type:                 "ec2-instance",
		UniqueAttributeValue: "i-12345",
		Scope:                "us-east-1",
	}
	refTo := &sdp.Reference{
		Type:                 "s3-bucket",
		UniqueAttributeValue: "my-test-bucket",
		Scope:                "global",
	}

	fromEdges := index.EdgesFrom(refFrom)
	if len(fromEdges) != 1 {
		t.Errorf("Expected 1 edge from ec2-instance i-12345, got %d", len(fromEdges))
	}
	if len(fromEdges) > 0 && !fromEdges[0].GetTo().IsEqual(refTo) {
		t.Error("EdgesFrom: expected To reference to be s3-bucket my-test-bucket")
	}

	toEdges := index.EdgesTo(refTo)
	if len(toEdges) != 1 {
		t.Errorf("Expected 1 edge to s3-bucket my-test-bucket, got %d", len(toEdges))
	}
	if len(toEdges) > 0 && !toEdges[0].GetFrom().IsEqual(refFrom) {
		t.Error("EdgesTo: expected From reference to be ec2-instance i-12345")
	}

	// No edges from the bucket (it only appears as To)
	fromBucket := index.EdgesFrom(refTo)
	if len(fromBucket) != 0 {
		t.Errorf("Expected 0 edges from bucket, got %d", len(fromBucket))
	}
	// No edges to the us-east-1 instance (it only appears as From in this snapshot)
	toInstance := index.EdgesTo(refFrom)
	if len(toInstance) != 0 {
		t.Errorf("Expected 0 edges to us-east-1 instance, got %d", len(toInstance))
	}
}

func TestNeighborItems(t *testing.T) {
	snapshot := createTestSnapshot()
	index, _ := NewSnapshotIndex(snapshot)

	ec2East := index.GetByGUN("us-east-1.ec2-instance.i-12345")
	if ec2East == nil {
		t.Fatal("expected to find us-east-1 ec2 instance")
	}
	neighbors := index.NeighborItems(ec2East)
	if len(neighbors) != 1 {
		t.Fatalf("Expected 1 neighbor of us-east-1 ec2 instance, got %d", len(neighbors))
	}
	if neighbors[0].GloballyUniqueName() != "global.s3-bucket.my-test-bucket" {
		t.Errorf("Expected neighbor to be s3-bucket, got %s", neighbors[0].GloballyUniqueName())
	}

	bucket := index.GetByGUN("global.s3-bucket.my-test-bucket")
	if bucket == nil {
		t.Fatal("expected to find s3 bucket")
	}
	neighbors = index.NeighborItems(bucket)
	if len(neighbors) != 1 {
		t.Fatalf("Expected 1 neighbor of s3 bucket, got %d", len(neighbors))
	}
	if neighbors[0].GloballyUniqueName() != "us-east-1.ec2-instance.i-12345" {
		t.Errorf("Expected neighbor to be ec2-instance i-12345, got %s", neighbors[0].GloballyUniqueName())
	}

	// us-west-2 instance has no edges
	ec2West := index.GetByGUN("us-west-2.ec2-instance.i-67890")
	if ec2West == nil {
		t.Fatal("expected to find us-west-2 ec2 instance")
	}
	neighbors = index.NeighborItems(ec2West)
	if len(neighbors) != 0 {
		t.Errorf("Expected 0 neighbors for us-west-2 instance, got %d", len(neighbors))
	}
}

func TestNewSnapshotIndexNilSnapshot(t *testing.T) {
	_, err := NewSnapshotIndex(nil)
	if err == nil {
		t.Error("Expected error for nil snapshot, got nil")
	}
}

func TestNewSnapshotIndexNilProperties(t *testing.T) {
	snapshot := &sdp.Snapshot{}
	_, err := NewSnapshotIndex(snapshot)
	if err == nil {
		t.Error("Expected error for nil properties, got nil")
	}
}
