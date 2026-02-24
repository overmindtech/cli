package shared

import (
	"strings"
	"unicode"
)

// azureProviderToAPI maps Azure resource provider namespaces to the short API names used in
// item types (see models.go). Enables generated linked queries to match existing adapter
// naming: azure-{api}-{resource} with kebab-case resource.
var azureProviderToAPI = map[string]string{
	"microsoft.compute":          "compute",
	"microsoft.network":         "network",
	"microsoft.storage":         "storage",
	"microsoft.sql":             "sql",
	"microsoft.documentdb":      "documentdb",
	"microsoft.keyvault":        "keyvault",
	"microsoft.managedidentity": "managedidentity",
	"microsoft.batch":           "batch",
	"microsoft.dbforpostgresql":  "dbforpostgresql",
	"microsoft.elasticsan":      "elasticsan",
	"microsoft.authorization":   "authorization",
	"microsoft.maintenance":     "maintenance",
	"microsoft.resources":       "resources",
}

// CamelCaseToKebab converts Azure camelCase resource type (e.g. virtualNetworks, publicIPAddresses)
// to kebab-case (e.g. virtual-networks, public-ip-addresses) to match project convention in models.go.
// Consecutive uppercase letters are treated as a single acronym (e.g. IP stays together).
func CamelCaseToKebab(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		if unicode.IsUpper(r) {
			prevLower := i > 0 && unicode.IsLower(runes[i-1])
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			// Insert hyphen before uppercase when: after a lowercase letter, or when this uppercase starts a word (next is lower)
			if i > 0 && (prevLower || (unicode.IsUpper(runes[i-1]) && nextLower)) {
				b.WriteByte('-')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// SingularizeResourceType converts Azure plural resource type to singular form to match
// models.go (e.g. virtual-networks -> virtual-network, galleries -> gallery, identities -> identity).
func SingularizeResourceType(kebab string) string {
	if kebab == "" {
		return kebab
	}
	// -ies -> -y (e.g. galleries -> gallery, user-assigned-identities -> user-assigned-identity)
	if strings.HasSuffix(kebab, "ies") {
		return strings.TrimSuffix(kebab, "ies") + "y"
	}
	// -addresses -> -address (e.g. public-ip-addresses -> public-ip-address)
	if strings.HasSuffix(kebab, "addresses") {
		return strings.TrimSuffix(kebab, "addresses") + "address"
	}
	if strings.HasSuffix(kebab, "s") {
		return strings.TrimSuffix(kebab, "s")
	}
	return kebab
}

// ItemTypeFromLinkedResourceID derives an item type string from an Azure resource ID for use in
// LinkedItemQueries (e.g. ResourceNavigationLink, ServiceAssociationLink). Uses short API names
// and kebab-case singular resource types so generated types match existing adapter naming
// (e.g. azure-network-virtual-network). For unknown providers, returns empty so callers can
// fall back to a generic type such as "azure-resource".
func ItemTypeFromLinkedResourceID(resourceID string) string {
	if resourceID == "" {
		return ""
	}
	parts := strings.Split(strings.Trim(resourceID, "/"), "/")
	for i, part := range parts {
		if strings.EqualFold(part, "providers") && i+2 < len(parts) {
			provider := strings.ToLower(parts[i+1])
			resourceTypeRaw := parts[i+2]
			api, ok := azureProviderToAPI[provider]
			if !ok {
				return ""
			}
			resourceType := SingularizeResourceType(CamelCaseToKebab(resourceTypeRaw))
			if resourceType == "" {
				return ""
			}
			return "azure-" + api + "-" + resourceType
		}
	}
	return ""
}
