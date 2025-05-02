package sdp

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const WILDCARD = "*"

// Copy copies all information from one item pointer to another
func (bp *BlastPropagation) Copy(dest *BlastPropagation) {
	dest.In = bp.GetIn()
	dest.Out = bp.GetOut()
}

func (bp *BlastPropagation) IsEqual(other *BlastPropagation) bool {
	return bp.GetIn() == other.GetIn() && bp.GetOut() == other.GetOut()
}

// Copy copies all information from one item pointer to another
func (liq *LinkedItemQuery) Copy(dest *LinkedItemQuery) {
	dest.Query = &Query{}
	if liq.GetQuery() != nil {
		liq.GetQuery().Copy(dest.GetQuery())
	}
	dest.BlastPropagation = &BlastPropagation{}
	if liq.GetBlastPropagation() != nil {
		liq.GetBlastPropagation().Copy(dest.GetBlastPropagation())
	}
}

// Copy copies all information from one item pointer to another
func (li *LinkedItem) Copy(dest *LinkedItem) {
	dest.Item = &Reference{}
	if li.GetItem() != nil {
		li.GetItem().Copy(dest.GetItem())
	}
	dest.BlastPropagation = &BlastPropagation{}
	if li.GetBlastPropagation() != nil {
		li.GetBlastPropagation().Copy(dest.GetBlastPropagation())
	}
}

// UniqueAttributeValue returns the value of whatever the Unique Attribute is
// for this item. This will then be converted to a string and returned
func (i *Item) UniqueAttributeValue() string {
	var value interface{}
	var err error

	value, err = i.GetAttributes().Get(i.GetUniqueAttribute())

	if err == nil {
		return fmt.Sprint(value)
	}

	return ""
}

// Reference returns an SDP reference for the item
func (i *Item) Reference() *Reference {
	return &Reference{
		Scope:                i.GetScope(),
		Type:                 i.GetType(),
		UniqueAttributeValue: i.UniqueAttributeValue(),
	}
}

// GloballyUniqueName Returns a string that defines the Item globally. This a
// combination of the following values:
//
//   - scope
//   - type
//   - uniqueAttributeValue
//
// They are concatenated with dots (.)
func (i *Item) GloballyUniqueName() string {
	return strings.Join([]string{
		i.GetScope(),
		i.GetType(),
		i.UniqueAttributeValue(),
	},
		".",
	)
}

// Copy copies all information from one item pointer to another
func (i *Item) Copy(dest *Item) {
	// Values can be copied directly
	dest.Type = i.GetType()
	dest.UniqueAttribute = i.GetUniqueAttribute()
	dest.Scope = i.GetScope()

	// We need to check that any pointers are actually populated with pointers
	// to somewhere in memory. If they are nil then there is no data structure
	// to copy the data into, therefore we need to create it first
	if dest.GetMetadata() == nil {
		dest.Metadata = &Metadata{}
	}

	if dest.GetAttributes() == nil {
		dest.Attributes = &ItemAttributes{}
	}

	i.GetMetadata().Copy(dest.GetMetadata())
	i.GetAttributes().Copy(dest.GetAttributes())

	// TODO(LIQs): delete this; it's not part of `(*sdp.Item)` anymore
	dest.LinkedItemQueries = make([]*LinkedItemQuery, 0)
	dest.LinkedItems = make([]*LinkedItem, 0)

	for _, r := range i.GetLinkedItemQueries() {
		liq := &LinkedItemQuery{}

		r.Copy(liq)

		dest.LinkedItemQueries = append(dest.LinkedItemQueries, liq)
	}

	for _, r := range i.GetLinkedItems() {
		newLinkedItem := &LinkedItem{}

		r.Copy(newLinkedItem)

		dest.LinkedItems = append(dest.LinkedItems, newLinkedItem)
	}

	if i.Health == nil {
		dest.Health = nil
	} else {
		dest.Health = i.GetHealth().Enum()
	}

	if i.Tags != nil {
		dest.Tags = make(map[string]string)

		for k, v := range i.GetTags() {
			dest.Tags[k] = v
		}
	}
}

// Hash Returns a 12 character hash for the item. This is likely but not
// guaranteed to be unique. The hash is calculated using the GloballyUniqueName
func (i *Item) Hash() string {
	return HashSum(([]byte(fmt.Sprint(i.GloballyUniqueName()))))
}

func (e *Edge) IsEqual(other *Edge) bool {
	return e.GetFrom().IsEqual(other.GetFrom()) &&
		e.GetTo().IsEqual(other.GetTo()) &&
		e.GetBlastPropagation().IsEqual(other.GetBlastPropagation())
}

// Hash Returns a 12 character hash for the item. This is likely but not
// guaranteed to be unique. The hash is calculated using the GloballyUniqueName
func (r *Reference) Hash() string {
	return HashSum(([]byte(fmt.Sprint(r.GloballyUniqueName()))))
}

// GloballyUniqueName Returns a string that defines the Item globally. This a
// combination of the following values:
//
//   - scope
//   - type
//   - uniqueAttributeValue
//
// They are concatenated with dots (.)
func (r *Reference) GloballyUniqueName() string {
	if r == nil {
		// in the llm templates nil references are processed, and after spending
		// half an hour on trying to figure out what was happening in the
		// reflect code, I decided to just return an empty string here. DS,
		// 2025-02-26
		return ""
	}
	if r.GetIsQuery() {
		if r.GetMethod() == QueryMethod_GET {
			// GET queries are single items
			return fmt.Sprintf("%v.%v.%v", r.GetScope(), r.GetType(), r.GetQuery())
		}
		panic(fmt.Sprintf("cannot get globally unique name for query reference: %v", r))
	}
	return fmt.Sprintf("%v.%v.%v", r.GetScope(), r.GetType(), r.GetUniqueAttributeValue())
}

// Key returns a globally unique string for this reference, even if it is a GET query
func (r *Reference) Key() string {
	if r == nil {
		panic("cannot get key for nil reference")
	}
	if r.GetIsQuery() {
		if r.IsSingle() {
			// GET queries without wildcards are single items
			return fmt.Sprintf("%v.%v.%v", r.GetScope(), r.GetType(), r.GetQuery())
		}
		return fmt.Sprintf("%v: %v.%v.%v", r.GetMethod(), r.GetScope(), r.GetType(), r.GetQuery())
	}
	return r.GloballyUniqueName()
}

// IsSingle returns true if this references a single item, false if it is a LIST
// or SEARCH query, or a GET query with scope and/or type wildcards.
func (r *Reference) IsSingle() bool {
	// nil reference is never good
	if r == nil {
		return false
	}
	// if it is a query, then it is only a single item if it is a GET query with no wildcards
	if r.GetIsQuery() {
		return r.GetMethod() == QueryMethod_GET && r.GetScope() != "*" && r.GetType() != "*"
	}
	// if it is not a query, then it is always single item
	return true
}

func (r *Reference) IsEqual(other *Reference) bool {
	return r.GetScope() == other.GetScope() &&
		r.GetType() == other.GetType() &&
		r.GetUniqueAttributeValue() == other.GetUniqueAttributeValue() &&
		r.GetIsQuery() == other.GetIsQuery() &&
		r.GetMethod() == other.GetMethod() &&
		r.GetQuery() == other.GetQuery()
}

// Copy copies all information from one Reference pointer to another
func (r *Reference) Copy(dest *Reference) {
	dest.Type = r.GetType()
	dest.UniqueAttributeValue = r.GetUniqueAttributeValue()
	dest.Scope = r.GetScope()
}

func (r *Reference) ToQuery() *Query {
	if !r.GetIsQuery() {
		return &Query{
			Scope:  r.GetScope(),
			Type:   r.GetType(),
			Method: QueryMethod_GET,
			Query:  r.GetUniqueAttributeValue(),
		}
	}

	return &Query{
		Scope:  r.GetScope(),
		Type:   r.GetType(),
		Method: r.GetMethod(),
		Query:  r.GetQuery(),
	}
}

// Copy copies all information from one Metadata pointer to another
func (m *Metadata) Copy(dest *Metadata) {
	if m == nil {
		// Protect from copy being called on a nil pointer
		return
	}

	dest.SourceName = m.GetSourceName()
	dest.Hidden = m.GetHidden()

	if m.GetSourceQuery() != nil {
		dest.SourceQuery = &Query{}
		m.GetSourceQuery().Copy(dest.GetSourceQuery())
	}

	dest.Timestamp = &timestamppb.Timestamp{
		Seconds: m.GetTimestamp().GetSeconds(),
		Nanos:   m.GetTimestamp().GetNanos(),
	}

	dest.SourceDuration = &durationpb.Duration{
		Seconds: m.GetSourceDuration().GetSeconds(),
		Nanos:   m.GetSourceDuration().GetNanos(),
	}

	dest.SourceDurationPerItem = &durationpb.Duration{
		Seconds: m.GetSourceDurationPerItem().GetSeconds(),
		Nanos:   m.GetSourceDurationPerItem().GetNanos(),
	}
}

// Copy copies all information from one CancelQuery pointer to another
func (c *CancelQuery) Copy(dest *CancelQuery) {
	if c == nil {
		return
	}

	dest.UUID = c.GetUUID()
}

// Get Returns the value of a given attribute by name. If the attribute is
// a nested hash, nested values can be referenced using dot notation e.g.
// location.country
func (a *ItemAttributes) Get(name string) (interface{}, error) {
	var result interface{}

	// Start at the beginning of the map, we will then traverse down as required
	result = a.GetAttrStruct().AsMap()

	for _, section := range strings.Split(name, ".") {
		// Check that the data we're using is in the supported format
		var m map[string]interface{}

		m, isMap := result.(map[string]interface{})

		if !isMap {
			return nil, fmt.Errorf("attribute %v not found", name)
		}

		v, keyExists := m[section]

		if keyExists {
			result = v
		} else {
			return nil, fmt.Errorf("attribute %v not found", name)
		}
	}

	return result, nil
}

// Set sets an attribute. Values are converted to structpb versions and an error
// will be returned if this fails. Note that this does *not* yet support
// dot notation e.g. location.country
func (a *ItemAttributes) Set(name string, value interface{}) error {
	// Check to make sure that the pointer is not nil
	if a == nil {
		return errors.New("Set called on nil pointer")
	}

	// Ensure that this interface will be able to be converted to a struct value
	sanitizedValue := sanitizeInterface(value, false, DefaultTransforms)
	structValue, err := structpb.NewValue(sanitizedValue)
	if err != nil {
		return err
	}

	fields := a.GetAttrStruct().GetFields()

	fields[name] = structValue

	return nil
}

// Copy copies all information from one ItemAttributes pointer to another
func (a *ItemAttributes) Copy(dest *ItemAttributes) {
	m := a.GetAttrStruct().AsMap()

	dest.AttrStruct, _ = structpb.NewStruct(m)
}

// Copy copies all information from one Query pointer to another
func (qrb *Query_RecursionBehaviour) Copy(dest *Query_RecursionBehaviour) {
	dest.LinkDepth = qrb.GetLinkDepth()
	dest.FollowOnlyBlastPropagation = qrb.GetFollowOnlyBlastPropagation()
}

// IsSingle returns true if this query can only return a single item.
func (q *Query) IsSingle() bool {
	return q.GetMethod() == QueryMethod_GET && q.GetScope() != "*" && q.GetType() != "*"
}

// Reference returns an SDP reference equivalent to this Query
func (q *Query) Reference() *Reference {
	if q.IsSingle() {
		return &Reference{
			Scope:                q.GetScope(),
			Type:                 q.GetType(),
			UniqueAttributeValue: q.GetQuery(),
		}
	}
	return &Reference{
		Scope:   q.GetScope(),
		Type:    q.GetType(),
		IsQuery: true,
		Query:   q.GetQuery(),
		Method:  q.GetMethod(),
	}
}

// Subject returns a NATS subject for all traffic relating to this query
func (q *Query) Subject() string {
	return fmt.Sprintf("query.%v", q.GetUUIDParsed())
}

// Copy copies all information from one Query pointer to another
func (q *Query) Copy(dest *Query) {
	dest.Type = q.GetType()
	dest.Method = q.GetMethod()
	dest.Query = q.GetQuery()
	dest.RecursionBehaviour = &Query_RecursionBehaviour{}
	if q.GetRecursionBehaviour() != nil {
		q.GetRecursionBehaviour().Copy(dest.GetRecursionBehaviour())
	}
	dest.Scope = q.GetScope()
	dest.IgnoreCache = q.GetIgnoreCache()
	dest.UUID = q.GetUUID()

	if q.GetDeadline() != nil {
		// allocate a new value
		dest.Deadline = timestamppb.New(q.GetDeadline().AsTime())
	}
}

// TimeoutContext returns a context and cancel function representing the timeout
// for this request
func (q *Query) TimeoutContext(ctx context.Context) (context.Context, context.CancelFunc) {
	// If there is no deadline, treat that as infinite
	if q == nil || !q.GetDeadline().IsValid() {
		return context.WithCancel(ctx)
	}

	return context.WithDeadline(ctx, q.GetDeadline().AsTime())
}

// GetUUIDParsed returns this request's UUID. If there's an error parsing it,
// generates and stores a fresh one
func (r *Query) GetUUIDParsed() uuid.UUID {
	if r == nil {
		return uuid.UUID{}
	}
	// Extract and parse the UUID
	reqUUID, uuidErr := uuid.FromBytes(r.GetUUID())
	if uuidErr != nil {
		reqUUID = uuid.New()
		r.UUID = reqUUID[:]
	}
	return reqUUID
}

func NewQueryResponseFromItem(item *Item) *QueryResponse {
	return &QueryResponse{
		ResponseType: &QueryResponse_NewItem{
			NewItem: item,
		},
	}
}

func NewQueryResponseFromEdge(edge *Edge) *QueryResponse {
	return &QueryResponse{
		ResponseType: &QueryResponse_Edge{
			Edge: edge,
		},
	}
}

func NewQueryResponseFromError(qe *QueryError) *QueryResponse {
	return &QueryResponse{
		ResponseType: &QueryResponse_Error{
			Error: qe,
		},
	}
}

func NewQueryResponseFromResponse(r *Response) *QueryResponse {
	return &QueryResponse{
		ResponseType: &QueryResponse_Response{
			Response: r,
		},
	}
}

func (qr *QueryResponse) ToGatewayResponse() *GatewayResponse {
	switch qr.GetResponseType().(type) {
	case *QueryResponse_NewItem:
		return &GatewayResponse{
			ResponseType: &GatewayResponse_NewItem{
				NewItem: qr.GetNewItem(),
			},
		}
	case *QueryResponse_Edge:
		return &GatewayResponse{
			ResponseType: &GatewayResponse_NewEdge{
				NewEdge: qr.GetEdge(),
			},
		}
	case *QueryResponse_Error:
		return &GatewayResponse{
			ResponseType: &GatewayResponse_QueryError{
				QueryError: qr.GetError(),
			},
		}
	case *QueryResponse_Response:
		return &GatewayResponse{
			ResponseType: &GatewayResponse_QueryStatus{
				QueryStatus: qr.GetResponse().ToQueryStatus(),
			},
		}
	default:
		panic(fmt.Sprintf("encountered unknown QueryResponse type: %T", qr))
	}
}

func (x *CancelQuery) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

func (x *Expand) GetUUIDParsed() *uuid.UUID {
	u, err := uuid.FromBytes(x.GetUUID())
	if err != nil {
		return nil
	}
	return &u
}

// AddDefaultTransforms adds the default transforms to a TransformMap
func AddDefaultTransforms(customTransforms TransformMap) TransformMap {
	for k, v := range DefaultTransforms {
		if _, ok := customTransforms[k]; !ok {
			customTransforms[k] = v
		}
	}
	return customTransforms
}

// Converts to attributes using an additional set of custom transformers. These
// can be used to change the transform behaviour of known types to do things
// like redaction of sensitive data or simplification of complex types.
//
// For example this could be used to completely remove anything of type
// `Secret`:
//
// ```go
//
//	TransformMap{
//	    reflect.TypeOf(Secret{}): func(i interface{}) interface{} {
//	        // Remove it
//	        return "REDACTED"
//	    },
//	}
//
// ```
//
// Note that you need to use `AddDefaultTransforms(TransformMap) TransformMap`
// to get sensible default transformations.
func ToAttributesCustom(m map[string]interface{}, sort bool, customTransforms TransformMap) (*ItemAttributes, error) {
	return toAttributes(m, sort, customTransforms)
}

// Converts a map[string]interface{} to an ItemAttributes object, sorting all
// slices alphabetically.This should be used when the item doesn't contain array
// attributes that are explicitly sorted, especially if these are sometimes
// returned in a different order
func ToAttributesSorted(m map[string]interface{}) (*ItemAttributes, error) {
	return toAttributes(m, true, DefaultTransforms)
}

// ToAttributes Converts a map[string]interface{} to an ItemAttributes object
func ToAttributes(m map[string]interface{}) (*ItemAttributes, error) {
	return toAttributes(m, false, DefaultTransforms)
}

func toAttributes(m map[string]interface{}, sort bool, customTransforms TransformMap) (*ItemAttributes, error) {
	if m == nil {
		return nil, nil
	}

	var s map[string]*structpb.Value
	var err error

	s = make(map[string]*structpb.Value)

	// Loop over the map
	for k, v := range m {
		sanitizedValue := sanitizeInterface(v, sort, customTransforms)
		structValue, err := structpb.NewValue(sanitizedValue)
		if err != nil {
			return nil, err
		}

		s[k] = structValue
	}

	return &ItemAttributes{
		AttrStruct: &structpb.Struct{
			Fields: s,
		},
	}, err
}

// ToAttributesViaJson Converts any struct to a set of attributes by marshalling
// to JSON and then back again. This is less performant than ToAttributes() but
// does save work when copying large structs to attributes in their entirety
func ToAttributesViaJson(v interface{}) (*ItemAttributes, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}

	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}

	return ToAttributes(m)
}

// A function that transforms one data type into another that is compatible with
// protobuf. This is used to convert things like time.Time into a string
type TransformFunc func(interface{}) interface{}

// A map of types to transform functions
type TransformMap map[reflect.Type]TransformFunc

// The default transforms that are used when converting to attributes
var DefaultTransforms = TransformMap{
	// Time should be in RFC3339Nano format i.e. 2006-01-02T15:04:05.999999999Z07:00
	reflect.TypeOf(time.Time{}): func(i interface{}) interface{} {
		return i.(time.Time).Format(time.RFC3339Nano)
	},
	// Duration should be in string format
	reflect.TypeOf(time.Duration(0)): func(i interface{}) interface{} {
		return i.(time.Duration).String()
	},
}

// sanitizeInterface Ensures that en interface is in a format that can be
// converted to a protobuf value. The structpb.ToValue() function expects things
// to be in one of the following formats:
//
//	╔════════════════════════╤════════════════════════════════════════════╗
//	║ Go type                │ Conversion                                 ║
//	╠════════════════════════╪════════════════════════════════════════════╣
//	║ nil                    │ stored as NullValue                        ║
//	║ bool                   │ stored as BoolValue                        ║
//	║ int, int32, int64      │ stored as NumberValue                      ║
//	║ uint, uint32, uint64   │ stored as NumberValue                      ║
//	║ float32, float64       │ stored as NumberValue                      ║
//	║ string                 │ stored as StringValue; must be valid UTF-8 ║
//	║ []byte                 │ stored as StringValue; base64-encoded      ║
//	║ map[string]interface{} │ stored as StructValue                      ║
//	║ []interface{}          │ stored as ListValue                        ║
//	╚════════════════════════╧════════════════════════════════════════════╝
//
// However this means that a data type like []string won't work, despite the
// function being perfectly able to represent it in a protobuf struct. This
// function does its best to example the available data type to ensure that as
// long as the data can in theory be represented by a protobuf struct, the
// conversion will work.
func sanitizeInterface(i interface{}, sortArrays bool, customTransforms TransformMap) interface{} {
	if i == nil {
		return nil
	}

	v := reflect.ValueOf(i)
	t := v.Type()

	// Use the transform for this specific type if it exists
	if tFunc, ok := customTransforms[t]; ok {
		// Reset the value and type to the transformed value. This means that
		// even if the function returns something bad, we will then transform it
		i = tFunc(i)

		if i == nil {
			return nil
		}

		v = reflect.ValueOf(i)
		t = v.Type()
	}

	switch v.Kind() { //nolint:exhaustive // we fall through to the default case
	case reflect.Bool:
		return v.Bool()
	case reflect.Int:
		return v.Int()
	case reflect.Int8:
		return v.Int()
	case reflect.Int16:
		return v.Int()
	case reflect.Int32:
		return v.Int()
	case reflect.Int64:
		return v.Int()
	case reflect.Uint:
		return v.Uint()
	case reflect.Uint8:
		return v.Uint()
	case reflect.Uint16:
		return v.Uint()
	case reflect.Uint32:
		return v.Uint()
	case reflect.Uint64:
		return v.Uint()
	case reflect.Float32:
		return v.Float()
	case reflect.Float64:
		return v.Float()
	case reflect.String:
		return fmt.Sprint(v)
	case reflect.Array, reflect.Slice:
		// We need to check the type of each element in the array and do
		// conversion on that

		// returnSlice Returns the array in the format that protobuf can deal with
		var returnSlice []interface{}

		returnSlice = make([]interface{}, v.Len())

		for i := range v.Len() {
			returnSlice[i] = sanitizeInterface(v.Index(i).Interface(), sortArrays, customTransforms)
		}

		if sortArrays {
			sortInterfaceArray(returnSlice)
		}

		return returnSlice
	case reflect.Map:
		var returnMap map[string]interface{}

		returnMap = make(map[string]interface{})

		for _, mapKey := range v.MapKeys() {
			// Convert the key to a string
			stringKey := fmt.Sprint(mapKey.Interface())

			// Convert the value to a compatible interface
			value := sanitizeInterface(v.MapIndex(mapKey).Interface(), sortArrays, customTransforms)
			returnMap[stringKey] = value
		}

		return returnMap
	case reflect.Struct:
		// In the case of a struct we basically want to turn it into a
		// map[string]interface{}
		var returnMap map[string]interface{}

		returnMap = make(map[string]interface{})

		// Range over fields
		n := t.NumField()
		for i := range n {
			field := t.Field(i)

			if field.PkgPath != "" {
				// If this has a PkgPath then it is an un-exported fiend and
				// should be ignored
				continue
			}

			// Get the zero value for this field
			zeroValue := reflect.Zero(field.Type).Interface()
			fieldValue := v.Field(i).Interface()

			// Check if the field is it's nil value
			// Check if there actually was a field with that name
			if !reflect.DeepEqual(fieldValue, zeroValue) {
				returnMap[field.Name] = fieldValue
			}
		}

		return sanitizeInterface(returnMap, sortArrays, customTransforms)
	case reflect.Ptr:
		// Get the zero value for this field
		zero := reflect.Zero(t)

		// Check if the field is it's nil value
		if reflect.DeepEqual(v, zero) {
			return nil
		}

		return sanitizeInterface(v.Elem().Interface(), sortArrays, customTransforms)
	default:
		// If we don't recognize the type then we need to see what the
		// underlying type is and see if we can convert that
		return i
	}
}

// Sorts an interface slice by converting each item to a string and sorting
// these strings
func sortInterfaceArray(input []interface{}) {
	sort.Slice(input, func(i, j int) bool {
		return fmt.Sprint(input[i]) < fmt.Sprint(input[j])
	})
}

// HashSum is a function that takes a byte array and returns a 12 character hash for use in neo4j
func HashSum(b []byte) string {
	var paddedEncoding *base32.Encoding
	var unpaddedEncoding *base32.Encoding

	shaSum := sha256.Sum256(b)

	// We need to specify a custom encoding here since dGraph has fairly strict
	// requirements about what name a variable can have
	paddedEncoding = base32.NewEncoding("abcdefghijklmnopqrstuvwxyzABCDEF")

	// We also can't have padding since "=" is not allowed in variable names
	unpaddedEncoding = paddedEncoding.WithPadding(base32.NoPadding)

	return unpaddedEncoding.EncodeToString(shaSum[:11])
}
