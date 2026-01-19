package proc

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	_ "github.com/overmindtech/cli/sources/gcp/dynamic"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"google.golang.org/protobuf/types/known/structpb"
)

func Test_adapters(t *testing.T) {
	ctx := context.Background()
	discoveryAdapters, err := adapters(
		ctx,
		"project",
		[]string{"region"},
		[]string{"zone"},
		"",
		gcpshared.NewLinker(),
		false,
		sdpcache.NewNoOpCache(),
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

			if meta.GetEndpointFunc == nil {
				t.Errorf("GetEndpointFunc is nil for %s", sdpItemType)
			}

			if meta.LocationLevel == "" {
				t.Errorf("LocationLevel is empty for %s", sdpItemType)
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

// mockAdapter is a mock implementation of discovery.Adapter for testing
type mockAdapter struct {
	projectID    string
	shouldError  bool
	errorMessage string
	callCount    *atomic.Int32
}

func newMockAdapter(projectID string, shouldError bool, errorMessage string) *mockAdapter {
	return &mockAdapter{
		projectID:    projectID,
		shouldError:  shouldError,
		errorMessage: errorMessage,
		callCount:    &atomic.Int32{},
	}
}

func (m *mockAdapter) Type() string {
	return gcpshared.CloudResourceManagerProject.String()
}

func (m *mockAdapter) Name() string {
	return "mock-adapter"
}

func (m *mockAdapter) Scopes() []string {
	return []string{"*"}
}

func (m *mockAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type: m.Type(),
	}
}

func (m *mockAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	m.callCount.Add(1)

	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}

	// Return a mock item with the project ID
	item := &sdp.Item{
		Type:            m.Type(),
		UniqueAttribute: "name",
		Attributes: &sdp.ItemAttributes{
			AttrStruct: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"projectId": structpb.NewStringValue(m.projectID),
				},
			},
		},
	}

	return item, nil
}

func (m *mockAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockAdapter) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockAdapter) GetCallCount() int32 {
	return m.callCount.Load()
}

func TestNewProjectHealthChecker(t *testing.T) {
	tests := []struct {
		name          string
		projectIDs    []string
		adapters      map[string]discovery.Adapter
		cacheDuration time.Duration
		expectValid   bool
	}{
		{
			name:          "valid inputs",
			projectIDs:    []string{"project-1", "project-2"},
			adapters:      map[string]discovery.Adapter{"project-1": newMockAdapter("project-1", false, "")},
			cacheDuration: 1 * time.Minute,
			expectValid:   true,
		},
		{
			name:          "empty project IDs",
			projectIDs:    []string{},
			adapters:      map[string]discovery.Adapter{},
			cacheDuration: 1 * time.Minute,
			expectValid:   true,
		},
		{
			name:          "zero cache duration",
			projectIDs:    []string{"project-1"},
			adapters:      map[string]discovery.Adapter{"project-1": newMockAdapter("project-1", false, "")},
			cacheDuration: 0,
			expectValid:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewProjectHealthChecker(tt.projectIDs, tt.adapters, tt.cacheDuration)

			if checker == nil {
				t.Fatal("expected checker to be non-nil")
			}

			if len(checker.projectIDs) != len(tt.projectIDs) {
				t.Errorf("expected %d project IDs, got %d", len(tt.projectIDs), len(checker.projectIDs))
			}

			if checker.cacheDuration != tt.cacheDuration {
				t.Errorf("expected cache duration %v, got %v", tt.cacheDuration, checker.cacheDuration)
			}
		})
	}
}

func TestProjectHealthChecker_Check_Success(t *testing.T) {
	ctx := context.Background()
	projectIDs := []string{"project-1", "project-2"}
	adapters := map[string]discovery.Adapter{
		"project-1": newMockAdapter("project-1", false, ""),
		"project-2": newMockAdapter("project-2", false, ""),
	}

	checker := NewProjectHealthChecker(projectIDs, adapters, 1*time.Minute)
	result, err := checker.Check(ctx)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if result.SuccessCount != 2 {
		t.Errorf("expected 2 successes, got %d", result.SuccessCount)
	}

	if result.FailureCount != 0 {
		t.Errorf("expected 0 failures, got %d", result.FailureCount)
	}

	if len(result.ProjectErrors) != 0 {
		t.Errorf("expected 0 project errors, got %d", len(result.ProjectErrors))
	}
}

func TestProjectHealthChecker_Check_Failures(t *testing.T) {
	ctx := context.Background()
	projectIDs := []string{"project-1", "project-2", "project-3"}
	adapters := map[string]discovery.Adapter{
		"project-1": newMockAdapter("project-1", false, ""),
		"project-2": newMockAdapter("project-2", true, "permission denied"),
		"project-3": newMockAdapter("project-3", true, "not found"),
	}

	checker := NewProjectHealthChecker(projectIDs, adapters, 1*time.Minute)
	result, err := checker.Check(ctx)

	if err == nil {
		t.Error("expected error, got nil")
	}

	if result.SuccessCount != 1 {
		t.Errorf("expected 1 success, got %d", result.SuccessCount)
	}

	if result.FailureCount != 2 {
		t.Errorf("expected 2 failures, got %d", result.FailureCount)
	}

	if len(result.ProjectErrors) != 2 {
		t.Errorf("expected 2 project errors, got %d", len(result.ProjectErrors))
	}

	if _, exists := result.ProjectErrors["project-2"]; !exists {
		t.Error("expected error for project-2")
	}

	if _, exists := result.ProjectErrors["project-3"]; !exists {
		t.Error("expected error for project-3")
	}
}

func TestProjectHealthChecker_Check_MissingAdapter(t *testing.T) {
	ctx := context.Background()
	projectIDs := []string{"project-1", "project-2"}
	adapters := map[string]discovery.Adapter{
		"project-1": newMockAdapter("project-1", false, ""),
		// project-2 adapter is missing
	}

	checker := NewProjectHealthChecker(projectIDs, adapters, 1*time.Minute)
	result, err := checker.Check(ctx)

	if err == nil {
		t.Error("expected error, got nil")
	}

	if result.SuccessCount != 1 {
		t.Errorf("expected 1 success, got %d", result.SuccessCount)
	}

	if result.FailureCount != 1 {
		t.Errorf("expected 1 failure, got %d", result.FailureCount)
	}

	if _, exists := result.ProjectErrors["project-2"]; !exists {
		t.Error("expected error for project-2")
	}
}

func TestProjectHealthChecker_Check_Caching(t *testing.T) {
	ctx := context.Background()
	projectIDs := []string{"project-1"}

	tests := []struct {
		name          string
		cacheDuration time.Duration
		sleepBetween  time.Duration
		expectCached  bool
	}{
		{
			name:          "cache hit within duration",
			cacheDuration: 1 * time.Minute,
			sleepBetween:  100 * time.Millisecond,
			expectCached:  true,
		},
		{
			name:          "cache miss after expiry",
			cacheDuration: 100 * time.Millisecond,
			sleepBetween:  200 * time.Millisecond,
			expectCached:  false,
		},
		{
			name:          "zero cache duration always misses",
			cacheDuration: 0,
			sleepBetween:  0,
			expectCached:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh mock adapter for each test
			mockAdpt := newMockAdapter("project-1", false, "")
			adapters := map[string]discovery.Adapter{
				"project-1": mockAdpt,
			}

			checker := NewProjectHealthChecker(projectIDs, adapters, tt.cacheDuration)

			// First call
			_, err := checker.Check(ctx)
			if err != nil {
				t.Fatalf("unexpected error on first call: %v", err)
			}

			firstCallCount := mockAdpt.GetCallCount()
			if firstCallCount != 1 {
				t.Errorf("expected 1 call after first check, got %d", firstCallCount)
			}

			// Sleep if needed
			if tt.sleepBetween > 0 {
				time.Sleep(tt.sleepBetween)
			}

			// Second call
			_, err = checker.Check(ctx)
			if err != nil {
				t.Fatalf("unexpected error on second call: %v", err)
			}

			secondCallCount := mockAdpt.GetCallCount()

			if tt.expectCached {
				// Should still be 1 call (cached)
				if secondCallCount != 1 {
					t.Errorf("expected cached result (1 total call), got %d calls", secondCallCount)
				}
			} else {
				// Should be 2 calls (not cached)
				if secondCallCount != 2 {
					t.Errorf("expected non-cached result (2 total calls), got %d calls", secondCallCount)
				}
			}
		})
	}
}

func TestProjectHealthChecker_Check_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	projectIDs := []string{"project-1"}
	mockAdpt := newMockAdapter("project-1", false, "")
	adapters := map[string]discovery.Adapter{
		"project-1": mockAdpt,
	}

	checker := NewProjectHealthChecker(projectIDs, adapters, 1*time.Minute)

	// Run multiple checks concurrently
	const concurrency = 10
	var wg sync.WaitGroup
	errors := make(chan error, concurrency)

	for range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := checker.Check(ctx)
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check if any errors occurred
	for err := range errors {
		t.Errorf("unexpected error during concurrent access: %v", err)
	}

	// The first goroutine should run the check, others should use cache
	// So we expect exactly 1 call
	callCount := mockAdpt.GetCallCount()
	if callCount != 1 {
		t.Errorf("expected 1 call with caching, got %d", callCount)
	}
}

func TestProjectPermissionCheckResult_FormatError(t *testing.T) {
	tests := []struct {
		name          string
		result        *ProjectPermissionCheckResult
		expectError   bool
		expectContain []string
	}{
		{
			name: "no failures",
			result: &ProjectPermissionCheckResult{
				SuccessCount:  2,
				FailureCount:  0,
				ProjectErrors: map[string]error{},
			},
			expectError: false,
		},
		{
			name: "single failure",
			result: &ProjectPermissionCheckResult{
				SuccessCount: 1,
				FailureCount: 1,
				ProjectErrors: map[string]error{
					"project-1": fmt.Errorf("permission denied"),
				},
			},
			expectError:   true,
			expectContain: []string{"1 out of 2", "50.0%", "project-1", "permission denied"},
		},
		{
			name: "multiple failures",
			result: &ProjectPermissionCheckResult{
				SuccessCount: 1,
				FailureCount: 2,
				ProjectErrors: map[string]error{
					"project-1": fmt.Errorf("permission denied"),
					"project-2": fmt.Errorf("not found"),
				},
			},
			expectError:   true,
			expectContain: []string{"2 out of 3", "66.7%", "project-1", "project-2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.result.FormatError()

			if tt.expectError && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}

			if err != nil {
				errStr := err.Error()
				for _, expected := range tt.expectContain {
					if !contains(errStr, expected) {
						t.Errorf("expected error to contain %q, got: %s", expected, errStr)
					}
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr))))
}
