package test

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/overmindtech/cli/sdp-go"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// This test data is designed to provide a full-featured graph to exercise all
// parts of the system. The graph is as follows:
//
//                      +--------+
//                      | admins |
//                      +--------+
//                        |
//                        |
//                        v
// +--------------+   b +--------+ b
// | motorcycling | <-- | dylan  | -+
// +--------------+     +--------+  |
//                        |b        |
//                        |         |
//                        vb        |
//                      +--------+  |
//                      | manny  |  |
//                      +--------+  |
//                        |b        |
//                        |         |
//                        v         |
//                      +--------+  |
//                      | london | <+
//                      +--------+
//                        |b
//                        |
//                        vb
//                      +----+
//                      | gb |
//                      +----+
//
// arrows indicate edge directions. b annotations indicate blast radius
// propagation.

// this global atomic variable keeps track of the generation count for test
// items. It is increased every time a new item is created, and is used to
// ensure that users of the test-adapter can determine that queries have hit the
// actual adapter and were not cached.
var generation atomic.Int32

// createTestItem Creates a simple item for testing
func createTestItem(typ, value string) *sdp.Item {
	thisGen := generation.Add(1)
	return &sdp.Item{
		Type:            typ,
		UniqueAttribute: "name",
		Attributes: &sdp.ItemAttributes{
			AttrStruct: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"name": {
						Kind: &structpb.Value_StringValue{
							StringValue: value,
						},
					},
					"generation": {
						Kind: &structpb.Value_NumberValue{
							// good enough for google, good enough for testing
							NumberValue: float64(thisGen),
						},
					},
				},
			},
		},
		Metadata: &sdp.Metadata{
			SourceName:            fmt.Sprintf("test-%v-adapter", typ),
			Timestamp:             timestamppb.Now(),
			SourceDuration:        durationpb.New(time.Second),
			SourceDurationPerItem: durationpb.New(time.Second),
			Hidden:                true,
		},
		Scope:             "test",
		LinkedItemQueries: []*sdp.LinkedItemQuery{},
		LinkedItems:       []*sdp.LinkedItem{},
	}
}

func admins() *sdp.Item {
	i := createTestItem("test-group", "test-admins")

	i.LinkedItemQueries = []*sdp.LinkedItemQuery{
		{
			Query: &sdp.Query{
				Type:   "test-person",
				Method: sdp.QueryMethod_GET,
				Query:  "test-dylan",
				Scope:  "test",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// the show must go on
				In:  false,
				Out: false,
			},
		},
	}

	return i
}

func dylan() *sdp.Item {
	i := createTestItem("test-person", "test-dylan")

	i.LinkedItemQueries = []*sdp.LinkedItemQuery{
		{
			Query: &sdp.Query{
				Type:   "test-dog",
				Method: sdp.QueryMethod_GET,
				Query:  "test-manny",
				Scope:  "test",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// best friends
				In:  true,
				Out: true,
			},
		},
		{
			Query: &sdp.Query{
				Type:   "test-hobby",
				Method: sdp.QueryMethod_GET,
				Query:  "test-motorcycling",
				Scope:  "test",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// accidents happen
				In: true,
				// motorcycles will endure
				Out: false,
			},
		},
		{
			Query: &sdp.Query{
				Type:   "test-location",
				Method: sdp.QueryMethod_GET,
				Query:  "test-london",
				Scope:  "test",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// we are what we eat
				In: true,
				// london don't care
				Out: false,
			},
		},
	}

	return i
}

func manny() *sdp.Item {
	i := createTestItem("test-dog", "test-manny")

	i.LinkedItemQueries = []*sdp.LinkedItemQuery{
		{
			Query: &sdp.Query{
				Type:   "test-location",
				Method: sdp.QueryMethod_GET,
				Query:  "test-london",
				Scope:  "test",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// we are what we eat
				In: true,
				// london don't care
				Out: false,
			},
		},
	}

	return i
}

func motorcycling() *sdp.Item {
	return createTestItem("test-hobby", "test-motorcycling")
}

func knitting() *sdp.Item {
	return createTestItem("test-hobby", "test-knitting")
}

func london() *sdp.Item {
	l := createTestItem("test-location", "test-london")
	l.LinkedItemQueries = []*sdp.LinkedItemQuery{
		{
			Query: &sdp.Query{
				Type:   "test-region",
				Method: sdp.QueryMethod_GET,
				Query:  "test-gb",
				Scope:  "test",
			},
			BlastPropagation: &sdp.BlastPropagation{
				// politics, enough said
				In:  true,
				Out: true,
			},
		},
	}

	return l
}

func gb() *sdp.Item {
	return createTestItem("test-region", "test-gb")
}
