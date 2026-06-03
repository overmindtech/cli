package shared

import (
	"encoding/json"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2"
)

func TestCompositeLookupKey(t *testing.T) {
	tests := []struct {
		name       string
		queryParts []string
		expected   string
	}{
		{
			name:       "Single query part",
			queryParts: []string{"part1"},
			expected:   "part1",
		},
		{
			name:       "Multiple query parts",
			queryParts: []string{"part1", "part2", "part3"},
			expected:   "part1|part2|part3",
		},
		{
			name:       "Empty query parts",
			queryParts: []string{},
			expected:   "",
		},
		{
			name:       "Query parts with empty strings",
			queryParts: []string{"part1", "", "part3"},
			expected:   "part1||part3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompositeLookupKey(tt.queryParts...)
			if result != tt.expected {
				t.Errorf("CompositeLookupKey(%v) = %q; want %q", tt.queryParts, result, tt.expected)
			}
		})
	}
}

func TestToAttributesWithExclude_nestedPath(t *testing.T) {
	t.Parallel()

	secret := &armkeyvault.Secret{
		Name: new("test-secret"),
		Tags: map[string]*string{
			"env": new("prod"),
		},
		Properties: &armkeyvault.SecretProperties{
			Value:     new("secret-value"),
			SecretURI: new("https://vault.vault.azure.net/secrets/test-secret"),
		},
	}

	attrs, err := ToAttributesWithExclude(secret, "tags", "Properties.Value")
	if err != nil {
		t.Fatalf("ToAttributesWithExclude: %v", err)
	}

	attrMap := attrs.GetAttrStruct().AsMap()
	if _, ok := attrMap["tags"]; ok {
		t.Fatalf("expected tags to be excluded, got %v", attrMap["tags"])
	}

	b, err := json.Marshal(attrMap)
	if err != nil {
		t.Fatalf("marshal attributes: %v", err)
	}
	attrsJSON := string(b)
	if containsJSONStringValue(attrsJSON, "secret-value") {
		t.Fatalf("secret value leaked in attributes: %s", attrsJSON)
	}
	if !containsJSONStringValue(attrsJSON, "https://vault.vault.azure.net/secrets/test-secret") {
		t.Fatalf("expected secretUri to remain in attributes: %s", attrsJSON)
	}
}

func containsJSONStringValue(attrsJSON, value string) bool {
	var m map[string]any
	if err := json.Unmarshal([]byte(attrsJSON), &m); err != nil {
		return false
	}
	return mapContainsStringValue(m, value)
}

func mapContainsStringValue(m map[string]any, value string) bool {
	for _, v := range m {
		switch x := v.(type) {
		case string:
			if x == value {
				return true
			}
		case map[string]any:
			if mapContainsStringValue(x, value) {
				return true
			}
		}
	}
	return false
}
