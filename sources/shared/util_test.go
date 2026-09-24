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

func TestToAttributesRedacting_deletesLeafThroughArrays(t *testing.T) {
	t.Parallel()

	payload := map[string]any{
		"template": map[string]any{
			"containers": []any{
				map[string]any{
					"image": "us-central1-docker.pkg.dev/p/repo/image:latest",
					"env": []any{
						map[string]any{
							"name":  "DATABASE_URL",
							"value": "postgres://array-env-sentinel-9f3a",
						},
						map[string]any{
							"name": "API_KEY",
							"valueSource": map[string]any{
								"secretKeyRef": map[string]any{
									"secret": "projects/p/secrets/api-key",
								},
							},
						},
					},
				},
				map[string]any{
					"image": "us-central1-docker.pkg.dev/p/repo/sidecar:latest",
					"env": []any{
						map[string]any{
							"name":  "SIDECAR_TOKEN",
							"value": "sidecar-env-sentinel-7c2b",
						},
					},
				},
			},
		},
	}

	attrs, err := ToAttributesRedacting(payload, []string{"template.containers.env.value"}, nil)
	if err != nil {
		t.Fatalf("ToAttributesRedacting: %v", err)
	}

	attrMap := attrs.GetAttrStruct().AsMap()
	attrsJSON := mustMarshalAttrs(t, attrMap)
	if containsJSONStringValue(attrsJSON, "postgres://array-env-sentinel-9f3a") {
		t.Fatalf("plaintext env value leaked in attributes: %s", attrsJSON)
	}
	if containsJSONStringValue(attrsJSON, "sidecar-env-sentinel-7c2b") {
		t.Fatalf("plaintext sidecar env value leaked in attributes: %s", attrsJSON)
	}

	template, ok := attrMap["template"].(map[string]any)
	if !ok {
		t.Fatalf("expected template object, got %T", attrMap["template"])
	}
	containers, ok := template["containers"].([]any)
	if !ok {
		t.Fatalf("expected containers array, got %T", template["containers"])
	}
	if len(containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(containers))
	}

	first, ok := containers[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first container object, got %T", containers[0])
	}
	if first["image"] != "us-central1-docker.pkg.dev/p/repo/image:latest" {
		t.Errorf("expected first container image to remain, got %v", first["image"])
	}
	firstEnv, ok := first["env"].([]any)
	if !ok {
		t.Fatalf("expected first env array, got %T", first["env"])
	}
	if len(firstEnv) != 2 {
		t.Fatalf("expected 2 env entries on first container, got %d", len(firstEnv))
	}
	assertEnvEntry(t, firstEnv[0], "DATABASE_URL", false, false)
	assertEnvEntry(t, firstEnv[1], "API_KEY", true, false)

	second, ok := containers[1].(map[string]any)
	if !ok {
		t.Fatalf("expected second container object, got %T", containers[1])
	}
	if second["image"] != "us-central1-docker.pkg.dev/p/repo/sidecar:latest" {
		t.Errorf("expected second container image to remain, got %v", second["image"])
	}
	secondEnv, ok := second["env"].([]any)
	if !ok {
		t.Fatalf("expected second env array, got %T", second["env"])
	}
	if len(secondEnv) != 1 {
		t.Fatalf("expected 1 env entry on second container, got %d", len(secondEnv))
	}
	assertEnvEntry(t, secondEnv[0], "SIDECAR_TOKEN", false, false)
}

func TestToAttributesRedacting_blanksMapValues(t *testing.T) {
	t.Parallel()

	const sentinel = "map-env-sentinel-4e1d"
	payload := map[string]any{
		"environmentVariables": map[string]any{
			"NPM_TOKEN":     sentinel,
			"DATABASE_URL":  "postgres://example",
			"NUMERIC_VALUE": 42,
		},
		"runtime": "python39",
	}

	attrs, err := ToAttributesRedacting(payload, nil, []string{"environmentVariables"})
	if err != nil {
		t.Fatalf("ToAttributesRedacting: %v", err)
	}

	attrMap := attrs.GetAttrStruct().AsMap()
	attrsJSON := mustMarshalAttrs(t, attrMap)
	if containsJSONStringValue(attrsJSON, sentinel) {
		t.Fatalf("sentinel secret string leaked in attributes: %s", attrsJSON)
	}

	envVars, ok := attrMap["environmentVariables"].(map[string]any)
	if !ok {
		t.Fatalf("expected environmentVariables map, got %T", attrMap["environmentVariables"])
	}
	for _, key := range []string{"NPM_TOKEN", "DATABASE_URL", "NUMERIC_VALUE"} {
		if _, exists := envVars[key]; !exists {
			t.Errorf("expected key %q to remain", key)
		}
		if envVars[key] != "" {
			t.Errorf("expected %q to be empty string, got %#v", key, envVars[key])
		}
	}
	if attrMap["runtime"] != "python39" {
		t.Errorf("expected sibling runtime to remain, got %v", attrMap["runtime"])
	}
}

func TestToAttributesRedacting_nonArrayExcludeAndMissingBlankPath(t *testing.T) {
	t.Parallel()

	payload := map[string]any{
		"labels": map[string]any{"env": "prod"},
		"properties": map[string]any{
			"value": "nested-exclude-sentinel-2a8c",
			"uri":   "https://example.invalid/secret",
		},
		"keep": "visible",
	}

	attrs, err := ToAttributesRedacting(payload, []string{"properties.value"}, []string{"missing.environmentVariables"})
	if err != nil {
		t.Fatalf("ToAttributesRedacting: %v", err)
	}

	attrMap := attrs.GetAttrStruct().AsMap()
	attrsJSON := mustMarshalAttrs(t, attrMap)
	if containsJSONStringValue(attrsJSON, "nested-exclude-sentinel-2a8c") {
		t.Fatalf("excluded nested value leaked in attributes: %s", attrsJSON)
	}
	if _, ok := attrMap["labels"]; !ok {
		t.Fatalf("expected labels to remain when not excluded, got %v", attrMap)
	}
	properties, ok := attrMap["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties object, got %T", attrMap["properties"])
	}
	if _, hasValue := properties["value"]; hasValue {
		t.Errorf("expected properties.value to be deleted, got %v", properties["value"])
	}
	if properties["uri"] != "https://example.invalid/secret" {
		t.Errorf("expected properties.uri to remain, got %v", properties["uri"])
	}
	if attrMap["keep"] != "visible" {
		t.Errorf("expected keep to remain, got %v", attrMap["keep"])
	}
}

func assertEnvEntry(t *testing.T, entry any, name string, wantValueSource bool, wantValue bool) {
	t.Helper()

	env, ok := entry.(map[string]any)
	if !ok {
		t.Fatalf("expected env object, got %T", entry)
	}
	if env["name"] != name {
		t.Errorf("expected env name %q, got %v", name, env["name"])
	}
	if _, hasValue := env["value"]; hasValue != wantValue {
		t.Errorf("env %q: expected value present=%v, got %v", name, wantValue, env["value"])
	}
	if _, hasSource := env["valueSource"]; hasSource != wantValueSource {
		t.Errorf("env %q: expected valueSource present=%v, got %v", name, wantValueSource, env["valueSource"])
	}
}

func mustMarshalAttrs(t *testing.T, attrMap map[string]any) string {
	t.Helper()

	b, err := json.Marshal(attrMap)
	if err != nil {
		t.Fatalf("marshal attributes: %v", err)
	}
	return string(b)
}

func containsJSONStringValue(attrsJSON, value string) bool {
	var m map[string]any
	if err := json.Unmarshal([]byte(attrsJSON), &m); err != nil {
		return false
	}
	return anyContainsStringValue(m, value)
}

func anyContainsStringValue(v any, value string) bool {
	switch x := v.(type) {
	case string:
		return x == value
	case map[string]any:
		for _, child := range x {
			if anyContainsStringValue(child, value) {
				return true
			}
		}
	case []any:
		for _, child := range x {
			if anyContainsStringValue(child, value) {
				return true
			}
		}
	}
	return false
}
