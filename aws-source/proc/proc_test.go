package proc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/smithy-go"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testAdapter is a minimal adapter for testing
type testAdapter struct {
	adapterType string
	scopes      []string
}

func (t *testAdapter) Type() string {
	return t.adapterType
}

func (t *testAdapter) Name() string {
	return "test-adapter"
}

func (t *testAdapter) Scopes() []string {
	return t.scopes
}

func (t *testAdapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type:            t.adapterType,
		DescriptiveName: "Test Adapter",
	}
}

func (t *testAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	return nil, &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOTFOUND,
		ErrorString: "not implemented for test",
		Scope:       scope,
	}
}

// TestInitializeAwsSourceEngine_RetryClearsAdapters tests that when a retry
// occurs, adapters from the previous attempt are cleared to avoid duplicate
// registration errors. This test verifies the fix for the issue where
// adapters from a previous retry attempt would remain in the engine, causing
// "adapter with type X and overlapping scopes already exists" errors.
func TestInitializeAwsSourceEngine_RetryClearsAdapters(t *testing.T) {
	// Create a minimal engine config without NATS to avoid needing a real connection
	ec := &discovery.EngineConfig{
		MaxParallelExecutions: 10,
		SourceName:            "test-aws-source",
		EngineType:            "aws",
		Version:               "test",
	}

	// Create an engine manually to test the clearing behavior
	engine, err := discovery.NewEngine(ec)
	require.NoError(t, err)

	// Create a test adapter to simulate a partial success scenario
	// where some adapters were added before a failure
	testAdapter := &testAdapter{
		adapterType: "ec2-address",
		scopes:      []string{"123456789012.us-east-1"},
	}

	err = engine.AddAdapters(testAdapter)
	require.NoError(t, err)

	// Verify adapter was added by checking available scopes
	scopes, _ := engine.GetAvailableScopesAndMetadata()
	assert.Contains(t, scopes, "123456789012.us-east-1", "Scope should be present before clear")

	// Verify we can't add the same adapter again (this would cause the error we're fixing)
	err = engine.AddAdapters(testAdapter)
	require.Error(t, err, "Should get error when adding duplicate adapter")
	require.Contains(t, err.Error(), "overlapping scopes already exists", "Error should mention overlapping scopes")

	// Clear adapters (simulating what happens before retry in InitializeAwsSourceEngine)
	engine.ClearAdapters()

	// Verify adapter was cleared by checking scopes
	scopes, _ = engine.GetAvailableScopesAndMetadata()
	assert.NotContains(t, scopes, "123456789012.us-east-1", "Scope should not be present after clear")

	// Now we should be able to add the adapter again without error
	// This simulates what happens on retry - adapters are cleared, so we can add them again
	err = engine.AddAdapters(testAdapter)
	require.NoError(t, err, "Should be able to add adapter again after clearing")

	// Verify adapter was added again
	scopes, _ = engine.GetAvailableScopesAndMetadata()
	assert.Contains(t, scopes, "123456789012.us-east-1", "Scope should be present after re-adding")
}

// mockAPIError implements smithy.APIError for testing
type mockAPIError struct {
	code    string
	message string
}

func (m *mockAPIError) Error() string {
	return m.message
}

func (m *mockAPIError) ErrorCode() string {
	return m.code
}

func (m *mockAPIError) ErrorMessage() string {
	return m.message
}

func (m *mockAPIError) ErrorFault() smithy.ErrorFault {
	return smithy.FaultUnknown
}

func TestIsOptInRegionError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedResult bool
	}{
		{
			name:           "nil error returns false",
			err:            nil,
			expectedResult: false,
		},
		{
			name: "InvalidIdentityToken with OIDC message returns true",
			err: &mockAPIError{
				code:    "InvalidIdentityToken",
				message: "InvalidIdentityToken: No OpenIDConnect provider found in your account for https://oidc.eks.eu-west-2.amazonaws.com/id/ABC123",
			},
			expectedResult: true,
		},
		{
			name: "wrapped InvalidIdentityToken with OIDC message returns true",
			err: fmt.Errorf("operation error STS: AssumeRoleWithWebIdentity: %w", &mockAPIError{
				code:    "InvalidIdentityToken",
				message: "No OpenIDConnect provider found in your account",
			}),
			expectedResult: true,
		},
		{
			name: "InvalidIdentityToken without OIDC message returns false",
			err: &mockAPIError{
				code:    "InvalidIdentityToken",
				message: "Invalid identity token for some other reason",
			},
			expectedResult: false,
		},
		{
			name: "different error code returns false",
			err: &mockAPIError{
				code:    "AccessDenied",
				message: "Access denied",
			},
			expectedResult: false,
		},
		{
			name:           "non-AWS error returns false",
			err:            errors.New("some random error"),
			expectedResult: false,
		},
		{
			name:           "error with OIDC text but not API error returns false",
			err:            errors.New("No OpenIDConnect provider found"),
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isOptInRegionError(tt.err)
			if result != tt.expectedResult {
				t.Errorf("isOptInRegionError() = %v, want %v for error: %v", result, tt.expectedResult, tt.err)
			}
		})
	}
}

func TestWrapRegionError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		region       string
		shouldWrap   bool
		expectedText string
	}{
		{
			name:         "nil error returns nil",
			err:          nil,
			region:       "us-east-1",
			shouldWrap:   false,
			expectedText: "",
		},
		{
			name: "opt-in region error gets wrapped",
			err: &mockAPIError{
				code:    "InvalidIdentityToken",
				message: "No OpenIDConnect provider found in your account",
			},
			region:       "eu-central-2",
			shouldWrap:   true,
			expectedText: "region 'eu-central-2' is not enabled",
		},
		{
			name: "wrapped opt-in region error gets additional context",
			err: fmt.Errorf("operation error STS: AssumeRoleWithWebIdentity: %w", &mockAPIError{
				code:    "InvalidIdentityToken",
				message: "No OpenIDConnect provider found in your account",
			}),
			region:       "ap-south-2",
			shouldWrap:   true,
			expectedText: "region 'ap-south-2' is not enabled",
		},
		{
			name: "InvalidIdentityToken without OIDC text not wrapped",
			err: &mockAPIError{
				code:    "InvalidIdentityToken",
				message: "some other message",
			},
			region:       "me-central-1",
			shouldWrap:   false,
			expectedText: "",
		},
		{
			name:         "unrelated error not wrapped",
			err:          errors.New("some other AWS error"),
			region:       "us-west-2",
			shouldWrap:   false,
			expectedText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := wrapRegionError(tt.err, tt.region)

			if tt.err == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("expected error, got nil")
				return
			}

			resultMsg := result.Error()

			if tt.shouldWrap {
				if !strings.Contains(resultMsg, tt.expectedText) {
					t.Errorf("expected wrapped error to contain '%s', got: %v", tt.expectedText, resultMsg)
				}
				// Verify the original error is preserved (wrapped with %w)
				if !errors.Is(result, tt.err) {
					t.Errorf("expected wrapped error to contain original error")
				}
			} else {
				if strings.Contains(resultMsg, "region") && strings.Contains(resultMsg, "not enabled") {
					t.Errorf("expected error not to be wrapped, but it was: %v", resultMsg)
				}
			}
		})
	}
}
