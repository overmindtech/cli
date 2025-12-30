package proc

import (
	"context"
	"fmt"
	"sort"
	"testing"

	_ "github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

func Test_adapters(t *testing.T) {
	ctx := context.Background()
	discoveryAdapters, err := adapters(
		ctx,
		"project",
		[]string{"region"},
		[]string{"zone"},
		"",
		nil,
		false,
	)
	if err != nil {
		t.Fatalf("error creating adapters: %v", err)
	}

	numberOfAdapters := len(discoveryAdapters)

	if numberOfAdapters == 0 {
		t.Fatal("Expected at least one adapter, got none")
	}

	if len(Metadata.AllAdapterMetadata()) != numberOfAdapters {
		t.Fatalf("Expected %d adapters in metadata, got %d", numberOfAdapters, len(Metadata.AllAdapterMetadata()))
	}

	// Check if the Spanner adapter is present
	// Because it is created externally and it needs to be registered during the initialization of the source
	// we need to ensure that it is included in the discoveryAdapters list.
	spannerAdapterFound := false
	for _, adapter := range discoveryAdapters {
		if adapter.Type() == gcpshared.SpannerDatabase.String() {
			spannerAdapterFound = true
			break
		}
	}

	if !spannerAdapterFound {
		t.Fatal("Expected to find Spanner adapter in the list of adapters")
	}

	aiPlatformCustomJobFound := false
	for _, adapter := range discoveryAdapters {
		if adapter.Type() == gcpshared.AIPlatformCustomJob.String() {
			aiPlatformCustomJobFound = true
			break
		}
	}

	if !aiPlatformCustomJobFound {
		t.Fatal("Expected to find AIPlatform Custom Job adapter in the list of adapters")
	}

	t.Logf("GCP Adapters found: %v", len(discoveryAdapters))
}

func Test_ensureMandatoryFieldsInDynamicAdapters(t *testing.T) {
	predefinedRoles := make(map[string]bool, len(gcpshared.SDPAssetTypeToAdapterMeta))
	for sdpItemType, meta := range gcpshared.SDPAssetTypeToAdapterMeta {
		t.Run(sdpItemType.String(), func(t *testing.T) {
			if meta.InDevelopment == true {
				t.Skipf("InDevelopment is true for %s", sdpItemType.String())
			}

			if meta.GetEndpointBaseURLFunc == nil {
				t.Errorf("GetEndpointBaseURLFunc is nil for %s", sdpItemType)
			}

			if meta.Scope == "" {
				t.Errorf("Scope is empty for %s", sdpItemType)
			}

			if len(meta.UniqueAttributeKeys) == 0 {
				t.Errorf("UniqueAttributeKeys is empty for %s", sdpItemType)
			}

			if len(meta.IAMPermissions) == 0 {
				t.Errorf("IAMPermissions is empty for %s", sdpItemType)
			}

			if len(meta.PredefinedRole) == 0 {
				t.Errorf("PredefinedRoles is empty for %s", sdpItemType)
			}

			role, ok := gcpshared.PredefinedRoles[meta.PredefinedRole]
			if !ok {
				t.Errorf("PredefinedRole %s is not in the PredefinedRoles map", meta.PredefinedRole)
			}

			foundPerm := false
			for _, perm := range role.IAMPermissions {
				for _, iamPerm := range meta.IAMPermissions {
					if perm == iamPerm {
						foundPerm = true
						break
					}
				}
			}

			if !foundPerm {
				t.Errorf("IAMPermissions %s is not in the PredefinedRole %s", meta.IAMPermissions, meta.PredefinedRole)
			}

			predefinedRoles[meta.PredefinedRole] = true
		})
	}

	roles := make([]string, 0, len(predefinedRoles))
	for r := range gcpshared.PredefinedRoles {
		roles = append(roles, r)
	}
	sort.Strings(roles)

	for _, r := range roles {
		fmt.Println("\"" + r + "\"")
	}
}

func Test_detectParentType(t *testing.T) {
	tests := []struct {
		name          string
		parent        string
		expectedType  ParentType
		expectedError bool
	}{
		{
			name:          "empty parent",
			parent:        "",
			expectedType:  ParentTypeUnknown,
			expectedError: true,
		},
		{
			name:          "organization format",
			parent:        "organizations/123456789012",
			expectedType:  ParentTypeOrganization,
			expectedError: false,
		},
		{
			name:          "folder format",
			parent:        "folders/987654321098",
			expectedType:  ParentTypeFolder,
			expectedError: false,
		},
		{
			name:          "explicit project format",
			parent:        "projects/my-project-id",
			expectedType:  ParentTypeProject,
			expectedError: false,
		},
		{
			name:          "project id format - simple",
			parent:        "my-project-id",
			expectedType:  ParentTypeProject,
			expectedError: false,
		},
		{
			name:          "project id format - with numbers",
			parent:        "my-project-123",
			expectedType:  ParentTypeProject,
			expectedError: false,
		},
		{
			name:          "project id format - with dashes",
			parent:        "my-project-test-123",
			expectedType:  ParentTypeProject,
			expectedError: false,
		},
		{
			name:          "too short to be valid",
			parent:        "short",
			expectedType:  ParentTypeUnknown,
			expectedError: true,
		},
		{
			name:          "too long to be valid project",
			parent:        "this-is-a-very-long-project-id-that-exceeds-the-thirty-character-limit",
			expectedType:  ParentTypeUnknown,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parentType, err := detectParentType(tt.parent)

			if tt.expectedError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if parentType != tt.expectedType {
				t.Errorf("expected parent type %v, got %v", tt.expectedType, parentType)
			}
		})
	}
}

func Test_normalizeParent(t *testing.T) {
	tests := []struct {
		name           string
		parent         string
		parentType     ParentType
		expectedResult string
		expectedError  bool
	}{
		{
			name:           "organization - already normalized",
			parent:         "organizations/123456789012",
			parentType:     ParentTypeOrganization,
			expectedResult: "organizations/123456789012",
			expectedError:  false,
		},
		{
			name:           "organization - empty ID",
			parent:         "organizations/",
			parentType:     ParentTypeOrganization,
			expectedResult: "",
			expectedError:  true,
		},
		{
			name:           "folder - already normalized",
			parent:         "folders/987654321098",
			parentType:     ParentTypeFolder,
			expectedResult: "folders/987654321098",
			expectedError:  false,
		},
		{
			name:           "folder - empty ID",
			parent:         "folders/",
			parentType:     ParentTypeFolder,
			expectedResult: "",
			expectedError:  true,
		},
		{
			name:           "project - explicit format",
			parent:         "projects/my-project-id",
			parentType:     ParentTypeProject,
			expectedResult: "my-project-id",
			expectedError:  false,
		},
		{
			name:           "project - empty ID",
			parent:         "projects/",
			parentType:     ParentTypeProject,
			expectedResult: "",
			expectedError:  true,
		},
		{
			name:           "project - just id",
			parent:         "my-project-id",
			parentType:     ParentTypeProject,
			expectedResult: "my-project-id",
			expectedError:  false,
		},
		{
			name:           "unknown type",
			parent:         "something",
			parentType:     ParentTypeUnknown,
			expectedResult: "",
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeParent(tt.parent, tt.parentType)

			if tt.expectedError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if result != tt.expectedResult {
				t.Errorf("expected result %q, got %q", tt.expectedResult, result)
			}
		})
	}
}
