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
// Intermediate arrays are walked in full so a path such as template.containers.env.value
// deletes value on every env entry of every container.
func ToAttributesWithExclude(i any, exclusions ...string) (*sdp.ItemAttributes, error) {
	return ToAttributesRedacting(i, exclusions, nil)
}

// ToAttributesWithPresence converts an interface to SDP attributes, excluding
// the given keys, then replaces presencePaths with true when the field exists.
// Use this for secret payloads where even a digest would let someone confirm a
// guessed value. The key stays at the same JSON path so add/remove is visible.
// A missing path is a no-op.
func ToAttributesWithPresence(i any, exclude []string, presencePaths []string) (*sdp.ItemAttributes, error) {
	return toAttributesTransformed(i, exclude, nil, presencePaths)
}

// ToAttributesRedacting converts an interface to SDP attributes, then redacts
// selected fields on that attribute copy. exclude drops keys (case-insensitive),
// and dotted paths descend maps and every element of arrays. blankMaps names
// maps whose values are each replaced with an empty string; keys are kept.
// A missing path is a no-op. A non-string map value is also replaced with an
// empty string so an unexpected JSON type cannot keep a secret.
func ToAttributesRedacting(i any, exclude []string, blankMaps []string) (*sdp.ItemAttributes, error) {
	return toAttributesTransformed(i, exclude, blankMaps, nil)
}

func toAttributesTransformed(i any, exclude, blankMaps, presencePaths []string) (*sdp.ItemAttributes, error) {
	b, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	for _, exclusion := range exclude {
		if exclusion == "" {
			continue
		}
		walkAndDelete(m, strings.Split(exclusion, "."))
	}

	for _, path := range blankMaps {
		if path == "" {
			continue
		}
		walkAndBlankMap(m, strings.Split(path, "."))
	}

	for _, path := range presencePaths {
		if path == "" {
			continue
		}
		walkAndMarkPresent(m, strings.Split(path, "."))
	}

	return sdp.ToAttributes(m)
}

func walkAndDelete(node any, path []string) {
	if len(path) == 0 || node == nil {
		return
	}

	switch current := node.(type) {
	case map[string]any:
		key := path[0]
		for k, v := range current {
			if !strings.EqualFold(k, key) {
				continue
			}
			if len(path) == 1 {
				delete(current, k)
				return
			}
			walkAndDelete(v, path[1:])
			return
		}
	case []any:
		for _, elem := range current {
			walkAndDelete(elem, path)
		}
	}
}

func walkAndBlankMap(node any, path []string) {
	if len(path) == 0 || node == nil {
		return
	}

	switch current := node.(type) {
	case map[string]any:
		key := path[0]
		for k, v := range current {
			if !strings.EqualFold(k, key) {
				continue
			}
			if len(path) == 1 {
				nested, ok := v.(map[string]any)
				if !ok {
					return
				}
				for mk := range nested {
					nested[mk] = ""
				}
				return
			}
			walkAndBlankMap(v, path[1:])
			return
		}
	case []any:
		for _, elem := range current {
			walkAndBlankMap(elem, path)
		}
	}
}

func walkAndMarkPresent(node any, path []string) {
	if len(path) == 0 || node == nil {
		return
	}

	switch current := node.(type) {
	case map[string]any:
		key := path[0]
		for k, v := range current {
			if !strings.EqualFold(k, key) {
				continue
			}
			if len(path) == 1 {
				current[k] = true
				return
			}
			walkAndMarkPresent(v, path[1:])
			return
		}
	case []any:
		for _, elem := range current {
			walkAndMarkPresent(elem, path)
		}
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
