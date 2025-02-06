package sdp

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var _u = uuid.New()

var query = Query{
	Type:   "user",
	Method: QueryMethod_LIST,
	RecursionBehaviour: &Query_RecursionBehaviour{
		LinkDepth: 10,
	},
	Scope:    "test",
	UUID:     _u[:],
	Deadline: timestamppb.New(time.Now().Add(10 * time.Second)),
}

var itemAttributes = ItemAttributes{
	AttrStruct: &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"foo": {
				Kind: &structpb.Value_StringValue{
					StringValue: "bar",
				},
			},
		},
	},
}

var metadata = Metadata{
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
}

var item = Item{
	Type:            "user",
	UniqueAttribute: "name",
	Attributes:      &itemAttributes,
	Metadata:        &metadata,
}

var items = Items{
	Items: []*Item{
		&item,
	},
}

var reference = Reference{
	Type:                 "user",
	UniqueAttributeValue: "dylan",
	Scope:                "test",
}

var queryError = QueryError{
	ErrorType:   QueryError_OTHER,
	ErrorString: "uh oh",
	Scope:       "test",
}

var ru = uuid.New()

var response = Response{
	Responder:     "test",
	ResponderUUID: ru[:],
	State:         ResponderState_WORKING,
	NextUpdateIn: &durationpb.Duration{
		Seconds: 10,
		Nanos:   0,
	},
}

var messages = []proto.Message{
	&query,
	&itemAttributes,
	&metadata,
	&item,
	&items,
	&reference,
	&queryError,
	&response,
}

// TestEncode Make sure that we can encode all of the message types without
// raising any errors
func TestEncode(t *testing.T) {
	for _, message := range messages {
		_, err := proto.Marshal(message)
		if err != nil {
			t.Error(err)
		}
	}
}

var decodeTests = []struct {
	Message proto.Message
	Target  proto.Message
}{
	{
		Message: &query,
		Target:  &Query{},
	},
	{
		Message: &itemAttributes,
		Target:  &ItemAttributes{},
	},
	{
		Message: &metadata,
		Target:  &Metadata{},
	},
	{
		Message: &item,
		Target:  &Item{},
	},
	{
		Message: &items,
		Target:  &Items{},
	},
	{
		Message: &reference,
		Target:  &Reference{},
	},
	{
		Message: &queryError,
		Target:  &QueryError{},
	},
	{
		Message: &response,
		Target:  &Response{},
	},
}

// TestDecode Make sure that we can decode all of the message
func TestDecode(t *testing.T) {
	for _, decTest := range decodeTests {
		// Marshal to binary
		b, err := proto.Marshal(decTest.Message)

		if err != nil {
			t.Fatal(err)
		}

		err = Unmarshal(context.Background(), b, decTest.Target)

		if err != nil {
			t.Error(err)
		}
	}
}
