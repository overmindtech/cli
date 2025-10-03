package cmd

import (
	"errors"
	"testing"

	"github.com/overmindtech/cli/sdp-go"
)

func TestValidateChangeStatus(t *testing.T) {
	tests := []struct {
		name        string
		statusStr   string
		expected    sdp.ChangeStatus
		expectError bool
	}{
		{
			name:        "valid defining status",
			statusStr:   "CHANGE_STATUS_DEFINING",
			expected:    sdp.ChangeStatus_CHANGE_STATUS_DEFINING,
			expectError: false,
		},
		{
			name:        "valid happening status",
			statusStr:   "CHANGE_STATUS_HAPPENING",
			expected:    sdp.ChangeStatus_CHANGE_STATUS_HAPPENING,
			expectError: false,
		},
		{
			name:        "valid done status",
			statusStr:   "CHANGE_STATUS_DONE",
			expected:    sdp.ChangeStatus_CHANGE_STATUS_DONE,
			expectError: false,
		},
		{
			name:        "invalid status - empty string",
			statusStr:   "",
			expected:    sdp.ChangeStatus_CHANGE_STATUS_UNSPECIFIED,
			expectError: true,
		},
		{
			name:        "invalid status - random string",
			statusStr:   "INVALID_STATUS",
			expected:    sdp.ChangeStatus_CHANGE_STATUS_UNSPECIFIED,
			expectError: true,
		},
		{
			name:        "invalid status - unspecified",
			statusStr:   "CHANGE_STATUS_UNSPECIFIED",
			expected:    sdp.ChangeStatus_CHANGE_STATUS_UNSPECIFIED,
			expectError: true,
		},
		{
			name:        "invalid status - processing",
			statusStr:   "CHANGE_STATUS_PROCESSING",
			expected:    sdp.ChangeStatus_CHANGE_STATUS_UNSPECIFIED,
			expectError: true,
		},
		{
			name:        "invalid status - lowercase",
			statusStr:   "change_status_defining",
			expected:    sdp.ChangeStatus_CHANGE_STATUS_UNSPECIFIED,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateChangeStatus(tt.statusStr)

			if tt.expectError {
				if err == nil {
					t.Errorf("validateChangeStatus() expected error but got none")
				}
				// Check that it returns a flagError
				var fError flagError
				if !errors.As(err, &fError) {
					t.Errorf("validateChangeStatus() expected flagError but got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("validateChangeStatus() unexpected error: %v", err)
				}
			}

			if result != tt.expected {
				t.Errorf("validateChangeStatus() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
