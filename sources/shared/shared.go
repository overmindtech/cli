package shared

import (
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Source represents the source of the item. It is usually the name of the
// source, e.g. "aws", "gcp", "azure", etc.
type Source string

// API represents the supported API from the source. It is usually the name of the
// API, e.g. "ec2", "s3", "compute-engine", etc.
type API string

// Resource represents the supported resource from the source. It is usually the name of the
// resource, e.g. "instance", "bucket", "disk", etc.
type Resource string

// ItemType represents the type of item. It is a combination of the Source, API and Resource.
type ItemType struct {
	Source   Source
	API      API
	Resource Resource
}

// String returns the string representation of the ItemType.
func (i ItemType) String() string {
	return fmt.Sprintf("%s-%s-%s", i.Source, i.API, i.Resource)
}

// Readable returns a human-readable string representation of the ItemType.
// For example, "AWS Ec2-Instance" or "GCP Compute Disk".
func (i ItemType) Readable() string {
	// Split the name by hyphens
	parts := strings.Split(i.String(), "-")

	// Capitalize the first part entirely
	if len(parts) > 0 {
		parts[0] = strings.ToUpper(parts[0])
	}

	// Capitalize the first letter of the remaining parts
	c := cases.Title(language.English)
	for i := 1; i < len(parts); i++ {
		parts[i] = c.String(parts[i])
	}

	// Join the parts with spaces
	return strings.Join(parts, " ")
}

// NewItemType creates a new ItemType from the given Source, API and Resource.
func NewItemType(source Source, api API, resource Resource) ItemType {
	return ItemType{
		Source:   source,
		API:      api,
		Resource: resource,
	}
}

// ItemTypeLookup is a struct that contains the ItemType and the string used to
// look it up.
// If it defines looking up an aws instance by "name" it will be
// ItemTypeLookup{By: "name", ItemType: ItemType{Source: aws.Source, API: aws.EC2, Resource: aws.Instance}}
type ItemTypeLookup struct {
	By       string
	ItemType ItemType
}

func (i ItemTypeLookup) Readable() string {
	return fmt.Sprintf(
		"%s-%s-%s-%s",
		i.ItemType.Source,
		i.ItemType.API,
		i.ItemType.Resource,
		i.By,
	)
}

// NewItemTypeLookup creates a new ItemTypeLookup from the given string and ItemType.
func NewItemTypeLookup(by string, itemType ItemType) ItemTypeLookup {
	return ItemTypeLookup{
		By:       by,
		ItemType: itemType,
	}
}

// NewItemTypesSet is convenience function that  creates a set of item types.
func NewItemTypesSet(items ...ItemType) map[ItemType]bool {
	m := make(map[ItemType]bool, len(items))
	for _, item := range items {
		m[item] = true
	}

	return m
}
