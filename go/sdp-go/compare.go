package sdp

import "fmt"

// Comparer is an object that can be compared for the purposes of sorting.
// Basically anything that implements this interface is sortable
type Comparer interface {
	Compare(b *Item) int
}

// Compare compares two Items for the purposes of sorting. This sorts based on
// the string conversion of the Type, followed by the UniqueAttribute
func (i *Item) Compare(r *Item) (int, error) {
	// Convert to strings
	right := fmt.Sprintf("%v: %v", r.GetType(), r.UniqueAttributeValue())
	left := fmt.Sprintf("%v: %v", i.GetType(), i.UniqueAttributeValue())

	// Compare the strings and return the value
	switch {
	case left > right:
		return 1, nil
	case left < right:
		return -1, nil
	default:
		return 0, nil
	}
}

// CompareError is returned when two Items cannot be compared because their
// UniqueAttributeValue() is not sortable
type CompareError Item

// Error returns the string when the error is handled
func (c *CompareError) Error() string {
	return (fmt.Sprintf(
		"Item %v unique attribute: %v of type %v does not implement interface fmt.Stringer. Cannot sort.",
		c.Type,
		c.UniqueAttribute,
		c.Type,
	))
}
