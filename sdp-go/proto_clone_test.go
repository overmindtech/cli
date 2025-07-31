package sdp

import (
	"testing"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestProtoCloneReplacesCustomCopy validates that proto.Clone works correctly
// for all SDP types and can replace the custom Copy methods
func TestProtoCloneReplacesCustomCopy(t *testing.T) {
	t.Run("Reference with all fields", func(t *testing.T) {
		original := &Reference{
			Type:                 "test",
			UniqueAttributeValue: "value", 
			Scope:                "scope",
			IsQuery:              true,
			Method:               QueryMethod_SEARCH,
			Query:                "search-term",
		}

		cloned := proto.Clone(original).(*Reference)
		
		if !proto.Equal(original, cloned) {
			t.Errorf("proto.Clone failed for Reference: %+v != %+v", original, cloned)
		}
		
		// Specifically check the fields that Copy() was missing
		if cloned.GetIsQuery() != original.GetIsQuery() {
			t.Errorf("IsQuery field not cloned correctly: got %v, want %v", cloned.GetIsQuery(), original.GetIsQuery())
		}
		if cloned.GetMethod() != original.GetMethod() {
			t.Errorf("Method field not cloned correctly: got %v, want %v", cloned.GetMethod(), original.GetMethod())
		}
		if cloned.GetQuery() != original.GetQuery() {
			t.Errorf("Query field not cloned correctly: got %v, want %v", cloned.GetQuery(), original.GetQuery())
		}
	})

	t.Run("Query with all fields", func(t *testing.T) {
		u := uuid.New()
		original := &Query{
			Type:   "test",
			Method: QueryMethod_GET,
			Query:  "value",
			Scope:  "scope", 
			UUID:   u[:],
			RecursionBehaviour: &Query_RecursionBehaviour{
				LinkDepth:                  5,
				FollowOnlyBlastPropagation: true,
			},
			IgnoreCache: true,
			Deadline:    timestamppb.Now(),
		}

		cloned := proto.Clone(original).(*Query)
		
		if !proto.Equal(original, cloned) {
			t.Errorf("proto.Clone failed for Query: %+v != %+v", original, cloned)
		}
	})

	t.Run("Item with all fields", func(t *testing.T) {
		original := &Item{
			Type:            "test",
			UniqueAttribute: "id",
			Scope:           "scope",
			Metadata: &Metadata{
				SourceName: "test-source",
				Hidden:     true,
				Timestamp:  timestamppb.Now(),
			},
			Health: Health_HEALTH_OK.Enum(),
			Tags: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
		}

		// Add attributes
		attrs, err := ToAttributes(map[string]interface{}{
			"name": "test-item",
			"port": 8080,
		})
		if err != nil {
			t.Fatal(err)
		}
		original.Attributes = attrs

		cloned := proto.Clone(original).(*Item)
		
		if !proto.Equal(original, cloned) {
			t.Errorf("proto.Clone failed for Item: %+v != %+v", original, cloned)
		}
	})

	t.Run("All other SDP types", func(t *testing.T) {
		// BlastPropagation
		bp := &BlastPropagation{In: true, Out: false}
		bpClone := proto.Clone(bp).(*BlastPropagation)
		if !proto.Equal(bp, bpClone) {
			t.Errorf("proto.Clone failed for BlastPropagation")
		}

		// LinkedItemQuery
		liq := &LinkedItemQuery{
			Query:            &Query{Type: "test", Method: QueryMethod_LIST},
			BlastPropagation: bp,
		}
		liqClone := proto.Clone(liq).(*LinkedItemQuery)
		if !proto.Equal(liq, liqClone) {
			t.Errorf("proto.Clone failed for LinkedItemQuery")
		}

		// LinkedItem
		li := &LinkedItem{
			Item:             &Reference{Type: "test", Scope: "scope"},
			BlastPropagation: bp,
		}
		liClone := proto.Clone(li).(*LinkedItem)
		if !proto.Equal(li, liClone) {
			t.Errorf("proto.Clone failed for LinkedItem")
		}

		// Metadata
		metadata := &Metadata{
			SourceName: "test-source",
			Hidden:     true,
			Timestamp:  timestamppb.Now(),
		}
		metadataClone := proto.Clone(metadata).(*Metadata)
		if !proto.Equal(metadata, metadataClone) {
			t.Errorf("proto.Clone failed for Metadata")
		}

		// CancelQuery
		u := uuid.New()
		cancelQuery := &CancelQuery{UUID: u[:]}
		cancelQueryClone := proto.Clone(cancelQuery).(*CancelQuery)
		if !proto.Equal(cancelQuery, cancelQueryClone) {
			t.Errorf("proto.Clone failed for CancelQuery")
		}
	})
}