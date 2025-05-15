package shared

import (
	"strings"

	"github.com/overmindtech/cli/sdp-go"
)

// ToAttributesWithExclude converts an interface to SDP attributes using the `sdp.ToAttributesSorted`
// function, and also allows the user to exclude certain top-level fields from
// the resulting attributes
func ToAttributesWithExclude(i interface{}, exclusions ...string) (*sdp.ItemAttributes, error) {
	attrs, err := sdp.ToAttributesViaJson(i)
	if err != nil {
		return nil, err
	}

	for _, exclusion := range exclusions {
		if s := attrs.GetAttrStruct(); s != nil {
			delete(s.GetFields(), exclusion)
		}
	}

	return attrs, nil
}

// CompositeLookupKey creates a composite lookup key from multiple query parts.
func CompositeLookupKey(queryParts ...string) string {
	// Join the query parts with the default separator "|"
	return strings.Join(queryParts, QuerySeparator)
}
