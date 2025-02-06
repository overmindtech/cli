package sdp

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestValidateItem(t *testing.T) {
	t.Run("item is fine", func(t *testing.T) {
		err := newItem().Validate()

		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Item is nil", func(t *testing.T) {
		var i *Item
		err := i.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("item has empty Type", func(t *testing.T) {
		i := newItem()

		i.Type = ""

		err := i.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("item has empty UniqueAttribute", func(t *testing.T) {
		i := newItem()

		i.UniqueAttribute = ""

		err := i.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("item has nil Attributes", func(t *testing.T) {
		i := newItem()

		i.Attributes = nil

		err := i.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("item has empty Scope", func(t *testing.T) {
		i := newItem()

		i.Scope = ""

		err := i.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("item has empty UniqueAttributeValue", func(t *testing.T) {
		i := newItem()

		err := i.GetAttributes().Set(i.GetUniqueAttribute(), "")
		if err != nil {
			t.Fatal(err)
		}

		err = i.Validate()
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestValidateReference(t *testing.T) {
	t.Run("Reference is fine", func(t *testing.T) {
		r := newReference()

		err := r.Validate()

		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Reference is nil", func(t *testing.T) {
		var r *Reference

		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("reference has empty Type", func(t *testing.T) {
		r := newReference()

		r.Type = ""

		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("reference has empty UniqueAttributeValue", func(t *testing.T) {
		r := newReference()

		r.UniqueAttributeValue = ""

		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("reference has empty Scope", func(t *testing.T) {
		r := newReference()

		r.Scope = ""

		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestValidateEdge(t *testing.T) {
	t.Run("Edge is fine", func(t *testing.T) {
		e := newEdge()

		err := e.Validate()

		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Edge has nil From", func(t *testing.T) {
		e := newEdge()

		e.From = nil

		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("Edge has nil To", func(t *testing.T) {
		e := newEdge()

		e.To = nil

		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("Edge has invalid From", func(t *testing.T) {
		e := newEdge()

		e.From.Type = ""

		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("Edge has invalid To", func(t *testing.T) {
		e := newEdge()

		e.To.Scope = ""

		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestValidateResponse(t *testing.T) {
	t.Run("Response is fine", func(t *testing.T) {
		r := newResponse()

		err := r.Validate()

		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Response is nil", func(t *testing.T) {
		var r *Response

		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("Response has empty Responder", func(t *testing.T) {
		r := newResponse()
		r.Responder = ""

		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("Response has empty UUID", func(t *testing.T) {
		r := newResponse()
		r.UUID = nil

		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestValidateQueryError(t *testing.T) {
	t.Run("QueryError is fine", func(t *testing.T) {
		e := newQueryError()

		err := e.Validate()

		if err != nil {
			t.Error(err)
		}
	})

	t.Run("QueryError is nil", func(t *testing.T) {

	})

	t.Run("QueryError has empty UUID", func(t *testing.T) {
		e := newQueryError()
		e.UUID = nil
		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("QueryError has empty ErrorString", func(t *testing.T) {
		e := newQueryError()
		e.ErrorString = ""
		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("QueryError has empty Scope", func(t *testing.T) {
		e := newQueryError()
		e.Scope = ""
		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("QueryError has empty SourceName", func(t *testing.T) {
		e := newQueryError()
		e.SourceName = ""
		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("QueryError has empty ItemType", func(t *testing.T) {
		e := newQueryError()
		e.ItemType = ""
		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("QueryError has empty ResponderName", func(t *testing.T) {
		e := newQueryError()
		e.ResponderName = ""
		err := e.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestValidateQuery(t *testing.T) {
	t.Run("Query is fine", func(t *testing.T) {
		r := newQuery()

		err := r.Validate()

		if err != nil {
			t.Error(err)
		}
	})

	t.Run("Query is nil", func(t *testing.T) {

	})

	t.Run("Query has empty Type", func(t *testing.T) {
		r := newQuery()
		r.Type = ""
		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}

	})

	t.Run("Query has empty Scope", func(t *testing.T) {
		r := newQuery()
		r.Scope = ""
		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}

	})

	t.Run("Response has empty UUID", func(t *testing.T) {
		r := newQuery()
		r.UUID = nil
		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}

	})

	t.Run("Query cannot have empty Query when method is Get", func(t *testing.T) {
		r := newQuery()
		r.Method = QueryMethod_GET
		r.Query = ""
		err := r.Validate()

		if err == nil {
			t.Error("expected error")
		}
	})

}

func newQuery() *Query {
	u := uuid.New()

	return &Query{
		Type:   "person",
		Method: QueryMethod_GET,
		Query:  "Dylan",
		RecursionBehaviour: &Query_RecursionBehaviour{
			LinkDepth: 1,
		},
		Scope:       "global",
		UUID:        u[:],
		Deadline:    timestamppb.New(time.Now().Add(1 * time.Second)),
		IgnoreCache: false,
	}
}

func newQueryError() *QueryError {
	u := uuid.New()

	return &QueryError{
		UUID:          u[:],
		ErrorType:     QueryError_OTHER,
		ErrorString:   "bad",
		Scope:         "global",
		SourceName:    "test-source",
		ItemType:      "test",
		ResponderName: "test-responder",
	}
}

func newResponse() *Response {
	u := uuid.New()

	ru := uuid.New()

	return &Response{
		Responder:     "foo",
		ResponderUUID: ru[:],
		State:         ResponderState_WORKING,
		NextUpdateIn:  durationpb.New(time.Second),
		UUID:          u[:],
	}
}

func newEdge() *Edge {
	return &Edge{
		From: newReference(),
		To:   newReference(),
	}
}

func newReference() *Reference {
	return &Reference{
		Type:                 "person",
		UniqueAttributeValue: "Dylan",
		Scope:                "global",
	}
}

func newItem() *Item {
	return &Item{
		Type:              "user",
		UniqueAttribute:   "name",
		Scope:             "test",
		LinkedItemQueries: []*LinkedItemQuery{},
		LinkedItems:       []*LinkedItem{},
		Attributes: &ItemAttributes{
			AttrStruct: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"name": {
						Kind: &structpb.Value_StringValue{
							StringValue: "bar",
						},
					},
				},
			},
		},
		Metadata: &Metadata{
			SourceName: "users",
			SourceQuery: &Query{
				Type:   "user",
				Method: QueryMethod_LIST,
				Query:  "*",
				RecursionBehaviour: &Query_RecursionBehaviour{
					LinkDepth: 12,
				},
				Scope: "testScope",
			},
			Timestamp: timestamppb.Now(),
			SourceDuration: &durationpb.Duration{
				Seconds: 1,
				Nanos:   1,
			},
			SourceDurationPerItem: &durationpb.Duration{
				Seconds: 0,
				Nanos:   500,
			},
		},
	}
}
