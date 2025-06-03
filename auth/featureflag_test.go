package auth

import (
	"context"
	"testing"

	"github.com/posthog/posthog-go"
)

func TestFeatureFlagEnabled(t *testing.T) {
	type args struct {
		ctx           context.Context
		posthogClient posthog.Client
		accountName   string
		subject       string
		key           string
	}
	tests := []struct {
		name        string
		args        args
		wantEnabled bool
	}{
		{
			name: "nil client returns false",
			args: args{
				ctx:           context.Background(),
				posthogClient: nil,
				accountName:   "test-account",
				subject:       "user-1",
				key:           "flag-1",
			},
			wantEnabled: false,
		},
		{
			name: "enqueue error returns false",
			args: args{
				ctx:           context.Background(),
				posthogClient: &mockPosthogClient{enqueueErr: true},
				accountName:   "test-account",
				subject:       "user-2",
				key:           "flag-2",
			},
			wantEnabled: false,
		},
		{
			name: "feature flag check error returns false",
			args: args{
				ctx:           context.Background(),
				posthogClient: &mockPosthogClient{featureFlagErr: true},
				accountName:   "test-account",
				subject:       "user-3",
				key:           "flag-3",
			},
			wantEnabled: false,
		},
		{
			name: "feature flag disabled returns false",
			args: args{
				ctx:           context.Background(),
				posthogClient: &mockPosthogClient{featureFlagValue: false},
				accountName:   "test-account",
				subject:       "user-4",
				key:           "flag-4",
			},
			wantEnabled: false,
		},
		{
			name: "feature flag enabled returns true",
			args: args{
				ctx:           context.Background(),
				posthogClient: &mockPosthogClient{featureFlagValue: true},
				accountName:   "test-account",
				subject:       "user-5",
				key:           "flag-5",
			},
			wantEnabled: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEnabled := FeatureFlagEnabledCustom(tt.args.ctx, tt.args.posthogClient, tt.args.accountName, tt.args.subject, tt.args.key)
			if gotEnabled != tt.wantEnabled {
				t.Errorf("FeatureFlagEnabled() = %v, want %v (test: %s)", gotEnabled, tt.wantEnabled, tt.name)
			}
		})
	}
}

// mockPosthogClient implements posthog.Client for testing
type mockPosthogClient struct {
	enqueueErr       bool
	featureFlagErr   bool
	featureFlagValue bool
}

func (m *mockPosthogClient) Enqueue(event posthog.Message) error {
	if m.enqueueErr {
		return errMock("enqueue error")
	}
	return nil
}

func (m *mockPosthogClient) IsFeatureEnabled(payload posthog.FeatureFlagPayload) (interface{}, error) {
	if m.featureFlagErr {
		return false, errMock("feature flag error")
	}
	return m.featureFlagValue, nil
}

// The following methods are required to satisfy the posthog.Client interface but are not used in these tests.

func (m *mockPosthogClient) Close() error                          { return nil }
func (m *mockPosthogClient) Capture(event posthog.Capture) error   { return nil }
func (m *mockPosthogClient) Flush() error                          { return nil }
func (m *mockPosthogClient) Identify(event posthog.Identify) error { return nil }
func (m *mockPosthogClient) Alias(event posthog.Alias) error       { return nil }
func (m *mockPosthogClient) GetFeatureFlag(payload posthog.FeatureFlagPayload) (interface{}, error) {
	return nil, nil
}
func (m *mockPosthogClient) Shutdown() {}
func (m *mockPosthogClient) GetFeatureFlagPayload(payload posthog.FeatureFlagPayload) (string, error) {
	return "", nil
}
func (m *mockPosthogClient) GetAllFlags(payload posthog.FeatureFlagPayloadNoKey) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
func (m *mockPosthogClient) GetFeatureFlags() ([]posthog.FeatureFlag, error) {
	return []posthog.FeatureFlag{}, nil
}
func (m *mockPosthogClient) GetLastCapturedEvent() *posthog.Capture { return nil }
func (m *mockPosthogClient) GetRemoteConfigPayload(flagKey string) (string, error) {
	return "", nil
}
func (m *mockPosthogClient) ReloadFeatureFlags() error { return nil }

type errMock string

func (e errMock) Error() string { return string(e) }
