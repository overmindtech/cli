package sdp

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/google/uuid"
)

// CalculatePaginationOffsetLimit Calculates the offset and limit for pagination
// in SQL queries, along with the current page and total pages that should be
// included in the response
//
// This also sets sane defaults for the page size if pagination is not provided.
// These defaults are page 1 with a page size of 10
//
// NOTE: If there are no items, then this will return 0 for all values
func CalculatePaginationOffsetLimit(pagination *PaginationRequest, totalItems int32) (offset, limit, page, totalPages int32) {
	if totalItems == 0 {
		// If there are no items, there are no pages
		return 0, 0, 0, 0
	}

	var requestedPageSize int32
	var requestedPage int32

	if pagination == nil {
		// Set sane defaults
		requestedPageSize = 10
		requestedPage = 1
	} else {
		requestedPageSize = pagination.GetPageSize()
		requestedPage = pagination.GetPage()
	}

	// pagesize is at least 10, at most 100
	limit = min(100, max(10, requestedPageSize))
	// calculate the total number of pages
	totalPages = int32(math.Ceil(float64(totalItems) / float64(limit)))

	// page has to be at least 1, and at most totalPages
	page = min(totalPages, requestedPage)
	page = max(1, page)

	// calculate the offset
	if totalPages == 0 {
		offset = 0
	} else {
		offset = (page * limit) - limit
	}
	return offset, limit, page, totalPages
}

// An object that returns all of the adapter metadata for a given source
type AdapterMetadataProvider interface {
	AllAdapterMetadata() []*AdapterMetadata
}

// A list of adapter metadata, this is used to store all the adapter metadata
// for a given source so that it can be retrieved later for the purposes of
// generating documentation and Terraform mappings
type AdapterMetadataList struct {
	// The list of adapter metadata
	list []*AdapterMetadata
}

// AllAdapterMetadata returns all the adapter metadata
func (a *AdapterMetadataList) AllAdapterMetadata() []*AdapterMetadata {
	return a.list
}

// RegisterAdapterMetadata registers a new adapter metadata with the list and
// returns a pointer to that same metadata to be used elsewhere
func (a *AdapterMetadataList) Register(metadata *AdapterMetadata) *AdapterMetadata {
	if a == nil {
		return metadata
	}

	a.list = append(a.list, metadata)

	return metadata
}

// flatten the data map, so that we can use it to compare the attributes of two
// items. It will recursively flatten the map and return a new map with the
// flattened data.
func flatten(data map[string]any) map[string]any {
	flattened := make(map[string]any)
	for k, v := range data {
		switch v := v.(type) {
		case map[string]any:
			for subK, subV := range flatten(v) {
				flattened[k+"."+subK] = subV
			}
		default:
			flattened[k] = v
		}
	}
	return flattened
}

// RenderItemDiff generates a diff between two items
func RenderItemDiff(before, after map[string]any) string {
	flatB := flatten(before)
	flatA := flatten(after)

	allKeys := slices.Collect(maps.Keys(flatB))
	allKeys = slices.AppendSeq(allKeys, maps.Keys(flatA))

	slices.Sort(allKeys)
	allKeys = slices.Compact(allKeys)

	// allKeys now contains every attribute present in either the before or
	// after, so we can iterate over it to generate the diff and append stats.
	para := []string{}
	for _, key := range allKeys {
		beforeValue, beforeExists := flatB[key]
		afterValue, afterExists := flatA[key]

		beforeValueStr := fmt.Sprintf("%v", beforeValue)
		afterValueStr := fmt.Sprintf("%v", afterValue)

		if beforeExists && afterExists {
			if beforeValueStr != afterValueStr {
				// This is an update
				para = append(para, fmt.Sprintf("- %s: %s\n+ %s: %s", key, beforeValueStr, key, afterValueStr))
			}
		} else if beforeExists && !afterExists {
			// This is a deletion
			para = append(para, fmt.Sprintf("- %s: %s", key, beforeValueStr))
		} else if !beforeExists && afterExists {
			// This is a creation
			para = append(para, fmt.Sprintf("+ %s: %s", key, afterValueStr))
		}
	}
	return strings.Join(para, "\n")
}

type RoutineRollUp struct {
	ChangeId uuid.UUID
	Gun      string
	Attr     string
	Value    string
}

func (rr RoutineRollUp) String() string {
	val := fmt.Sprintf("%v", rr.Value)
	if len(val) > 100 {
		val = val[:100]
	}
	val = strings.ReplaceAll(val, "\n", " ")
	val = strings.ReplaceAll(val, "\t", " ")
	return fmt.Sprintf("change:%v\tgun:%v\tattr:%v\tval:%v", rr.ChangeId, rr.Gun, rr.Attr, val)
}

func WalkMapToRoutineRollUp(gun string, key string, data map[string]any) []RoutineRollUp {
	results := []RoutineRollUp{}

	for k, v := range data {
		attr := k
		if key != "" {
			attr = fmt.Sprintf("%v.%v", key, k)
		}
		switch val := v.(type) {
		case map[string]any:
			results = append(results, WalkMapToRoutineRollUp(gun, attr, val)...)
		default:
			results = append(results, RoutineRollUp{
				Gun:   gun,
				Attr:  attr,
				Value: fmt.Sprintf("%v", val),
			})
		}
	}

	return results
}

// GcpSANameFromAccountName generates a GCP service account name from the given
// Service account must be 6-30 characters long, and must comply with the
// `^[a-zA-Z][a-zA-Z\d\-]*[a-zA-Z\d]$` regex.
//
// This regex returned from an error message when trying to create a service account.
// Unfortunately, we could not find any documentation on this.
// The account name is expected to be in the format of a UUID, which is 36 characters long,
// and contains dashes.
// The service account name must be 30 characters or less,
// and must start with a letter, end with a letter or digit, and can only contain
// letters, digits, and dashes.
// So we keep the SA name simple: Start with "C-" and take the first 28 characters of the account name.
func GcpSANameFromAccountName(accountName string) string {
	if accountName == "" {
		return ""
	}

	accountName = strings.ReplaceAll(accountName, "-", "")

	if len(accountName) >= 6 {
		// Ensure the account name is at most 30 characters long
		// We will prefix it with "C-" to ensure it starts with a letter
		// and truncate it to 28 characters after the prefix
		if len(accountName) > 28 {
			accountName = accountName[:28]
		}

		return "C-" + accountName
	}

	return ""
}
