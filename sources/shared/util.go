package shared

import "github.com/overmindtech/cli/sdp-go"

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
