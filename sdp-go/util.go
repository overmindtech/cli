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

// RenderItemDiff generates a diff between two items' attributes represented as
// maps. `changeData` is a slice of RoutineRollup objects that contain the
// rollups for this change. `rawData` is a slice of RoutineRollup objects that
// contain the the rollups for all changes in the last 30 days.
func RenderItemDiff(gun string, before, after map[string]any, changeData, rawData []RoutineRollup) string {
	flatB := flatten(before)
	flatA := flatten(after)

	allKeys := slices.Collect(maps.Keys(flatB))
	allKeys = slices.AppendSeq(allKeys, maps.Keys(flatA))

	slices.Sort(allKeys)
	allKeys = slices.Compact(allKeys)

	// allKeys now contains every attribute present in either the before or
	// after, so we can iterate over it to generate the diff and append stats.

	attrs := map[string][]string{}
	for _, ch := range changeData {
		var attrVals []string
		for _, raw := range rawData {
			if raw.Attr == ch.Attr && raw.Gun == ch.Gun {
				attrVals = append(attrVals, raw.Value)
			}
		}
		if len(attrVals) > 0 {
			attrs[fmt.Sprintf("%v.%v", ch.Gun, ch.Attr)] = attrVals
		}
	}

	para := []string{}
	for _, key := range allKeys {
		beforeValue, beforeExists := flatB[key]
		afterValue, afterExists := flatA[key]

		beforeValueStr := fmt.Sprintf("%v", beforeValue)
		afterValueStr := fmt.Sprintf("%v", afterValue)

		hasChanged := false
		if beforeExists && afterExists {
			if beforeValueStr != afterValueStr {
				// This is an update
				para = append(para, fmt.Sprintf("- %s: %s\n+ %s: %s", key, beforeValueStr, key, afterValueStr))
				hasChanged = true
			}
		} else if beforeExists && !afterExists {
			// This is a deletion
			para = append(para, fmt.Sprintf("- %s: %s", key, beforeValueStr))
			hasChanged = true
		} else if !beforeExists && afterExists {
			// This is a creation
			para = append(para, fmt.Sprintf("+ %s: %s", key, afterValueStr))
			hasChanged = true
		}

		if hasChanged {
			k := fmt.Sprintf("%v.%v", gun, key)
			v := attrs[k]
			slices.Sort(v)

			if len(v) > 0 {
				para = append(para, fmt.Sprintf("# → 🔁 This attribute has changed %d times in the last 30 days.\n#      The previous values were %v.", len(v), v))
			}
		}
	}
	return strings.Join(para, "\n")
}

type RoutineRollup struct {
	ChangeId uuid.UUID
	Gun      string
	Attr     string
	Value    string
}

func (rr RoutineRollup) String() string {
	val := fmt.Sprintf("%v", rr.Value)
	if len(val) > 100 {
		val = val[:100]
	}
	val = strings.ReplaceAll(val, "\n", " ")
	val = strings.ReplaceAll(val, "\t", " ")
	return fmt.Sprintf("change:%v\tgun:%v\tattr:%v\tval:%v", rr.ChangeId, rr.Gun, rr.Attr, val)
}

func RollupDiffs(data []*ItemDiff, rawData []RoutineRollup) []RoutineRollup {
	for _, diff := range data {
		afterItem := diff.GetAfter()
		afterMap := afterItem.GetAttributes().GetAttrStruct().AsMap()
		afterGun := afterItem.GloballyUniqueName()

		if len(afterMap) > 0 {
			rawData = append(rawData, walkMap(afterGun, "", afterMap)...)
		}
	}
	return rawData
}

func walkMap(gun string, key string, data map[string]any) []RoutineRollup {
	results := []RoutineRollup{}

	for k, v := range data {
		attr := k
		if key != "" {
			attr = fmt.Sprintf("%v.%v", key, k)
		}
		switch val := v.(type) {
		case map[string]any:
			results = append(results, walkMap(gun, attr, val)...)
		default:
			results = append(results, RoutineRollup{
				Gun:   gun,
				Attr:  attr,
				Value: fmt.Sprintf("%v", val),
			})
		}
	}

	return results
}
