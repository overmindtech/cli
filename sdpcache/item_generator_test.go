package sdpcache

import (
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/overmindtech/cli/sdp-go"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var Types = []string{
	"person",
	"dog",
	"kite",
	"flag",
	"cat",
	"leopard",
	"fish",
	"bird",
	"kangaroo",
	"ostrich",
	"emu",
	"hawk",
	"mole",
	"badger",
	"lemur",
}

const MaxAttributes = 30
const MaxTags = 10
const MaxTagKeyLength = 10
const MaxTagValueLength = 10
const MaxAttributeKeyLength = 20
const MaxAttributeValueLength = 50

// TODO(LIQs): rewrite this to `MaxEdges`
const MaxLinkedItems = 10

// TODO(LIQs): delete
const MaxLinkedItemQueries = 10

// GenerateRandomItem Generates a random item and the tags for this item. The
// tags include the name, type and a tag called "all" with a value of "all"
func GenerateRandomItem() *sdp.Item {
	attrs := make(map[string]interface{})

	name := randSeq(rand.Intn(MaxAttributeValueLength))
	typ := Types[rand.Intn(len(Types))]
	scope := randSeq(rand.Intn(MaxAttributeKeyLength))
	attrs["name"] = name

	for range rand.Intn(MaxAttributes) {
		attrs[randSeq(rand.Intn(MaxAttributeKeyLength))] = randSeq(rand.Intn(MaxAttributeValueLength))
	}

	attributes, _ := sdp.ToAttributes(attrs)

	tags := make(map[string]string)

	for range rand.Intn(MaxTags) {
		tags[randSeq(rand.Intn(MaxTagKeyLength))] = randSeq(rand.Intn(MaxTagValueLength))
	}

	// TODO(LIQs): rewrite this to `MaxEdges` and return and additional []*sdp.Edge
	linkedItems := make([]*sdp.LinkedItem, rand.Intn(MaxLinkedItems))

	for i := range linkedItems {
		linkedItems[i] = &sdp.LinkedItem{Item: &sdp.Reference{
			Type:                 randSeq(rand.Intn(MaxAttributeKeyLength)),
			UniqueAttributeValue: randSeq(rand.Intn(MaxAttributeValueLength)),
			Scope:                randSeq(rand.Intn(MaxAttributeKeyLength)),
		}}
	}

	linkedItemQueries := make([]*sdp.LinkedItemQuery, rand.Intn(MaxLinkedItemQueries))

	for i := range linkedItemQueries {
		linkedItemQueries[i] = &sdp.LinkedItemQuery{Query: &sdp.Query{
			Type:   randSeq(rand.Intn(MaxAttributeKeyLength)),
			Method: sdp.QueryMethod(rand.Intn(3)),
			Query:  randSeq(rand.Intn(MaxAttributeValueLength)),
			RecursionBehaviour: &sdp.Query_RecursionBehaviour{
				LinkDepth:                  rand.Uint32(),
				FollowOnlyBlastPropagation: rand.Intn(2) == 0,
			},
			Scope: randSeq(rand.Intn(MaxAttributeKeyLength)),
		}}
	}

	// Generate health (which is an int32 between 0 and 4)
	health := sdp.Health(rand.Intn(int(sdp.Health_HEALTH_PENDING) + 1))

	queryUuid := uuid.New()

	item := sdp.Item{
		Type:              typ,
		UniqueAttribute:   "name",
		Attributes:        attributes,
		Scope:             scope,
		LinkedItemQueries: linkedItemQueries,
		LinkedItems:       linkedItems,
		Metadata: &sdp.Metadata{
			SourceName: randSeq(rand.Intn(MaxAttributeKeyLength)),
			SourceQuery: &sdp.Query{
				Type:   typ,
				Method: sdp.QueryMethod_GET,
				Query:  name,
				RecursionBehaviour: &sdp.Query_RecursionBehaviour{
					LinkDepth: 1,
				},
				Scope: scope,
				UUID:  queryUuid[:],
			},
			Timestamp:             timestamppb.New(time.Now()),
			SourceDuration:        durationpb.New(time.Millisecond * time.Duration(rand.Int63())),
			SourceDurationPerItem: durationpb.New(time.Millisecond * time.Duration(rand.Int63())),
		},
		Tags:   tags,
		Health: &health,
	}

	return &item
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
