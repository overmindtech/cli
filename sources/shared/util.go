package shared

import (
	"encoding/json"
	"strings"

	"github.com/overmindtech/cli/go/sdp-go"
)

// ToAttributesWithExclude converts an interface to SDP attributes using the `sdp.ToAttributesSorted`
// function, and also allows the user to exclude certain fields from the resulting attributes.
// Top-level exclusions use the field name as it appears in JSON (e.g. "tags").
// Dot-separated paths exclude nested fields (e.g. "Properties.Value" matches properties.value).
func ToAttributesWithExclude(i any, exclusions ...string) (*sdp.ItemAttributes, error) {
	b, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	for _, exclusion := range exclusions {
		if exclusion == "" {
			continue
		}
		if strings.Contains(exclusion, ".") {
			deleteNestedMapKey(m, strings.Split(exclusion, "."))
		} else {
			deleteMapKey(m, exclusion)
		}
	}

	return sdp.ToAttributes(m)
}

func deleteMapKey(m map[string]any, key string) {
	for k := range m {
		if strings.EqualFold(k, key) {
			delete(m, k)
			return
		}
	}
}

func deleteNestedMapKey(m map[string]any, path []string) {
	if len(path) == 0 || m == nil {
		return
	}

	key := path[0]
	for k, v := range m {
		if !strings.EqualFold(k, key) {
			continue
		}
		if len(path) == 1 {
			delete(m, k)
			return
		}
		nested, ok := v.(map[string]any)
		if !ok {
			return
		}
		deleteNestedMapKey(nested, path[1:])
		return
	}
}

// CompositeLookupKey creates a composite lookup key from multiple query parts.
// It joins the parts using the default separator "|"
//
// Example usage:
//
//	key := CompositeLookupKey("part1", "part2", "part3")
//	Output: "part1|part2|part3"
func CompositeLookupKey(queryParts ...string) string {
	// Join the query parts with the default separator "|"
	return strings.Join(queryParts, QuerySeparator)
}
