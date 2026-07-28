package sdp

import (
	"strings"
	"testing"

	"buf.build/go/protovalidate"
)

// Brent-only IntegrationStatus validation lives here (brent* basename) so
// Copybara's go/sdp-go/brent* exclusion keeps it out of public repos that
// receive go/sdp-go/** without brent.pb.go.

func TestValidateIntegrationStatus(t *testing.T) {
	t.Parallel()

	validConnected := &IntegrationStatus{
		Provider:     IntegrationProvider_INTEGRATION_PROVIDER_GITHUB,
		OrgInstalled: true,
		Status:       IntegrationConnectionStatus_INTEGRATION_CONNECTION_STATUS_CONNECTED,
		Summary:      "Connected",
		Detail:       "This integration is connected and ready to use.",
	}
	validNotConnected := &IntegrationStatus{
		Provider:     IntegrationProvider_INTEGRATION_PROVIDER_GITLAB,
		OrgInstalled: false,
		Status:       IntegrationConnectionStatus_INTEGRATION_CONNECTION_STATUS_NOT_CONNECTED,
		Summary:      "Not connected",
		Detail:       "This integration is not connected for this workspace.",
	}

	t.Run("connected defaults pass", func(t *testing.T) {
		t.Parallel()
		if err := protovalidate.Validate(validConnected); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
	t.Run("not connected defaults pass", func(t *testing.T) {
		t.Parallel()
		if err := protovalidate.Validate(validNotConnected); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
	t.Run("summary length 33 fails", func(t *testing.T) {
		t.Parallel()
		status := cloneIntegrationStatus(validConnected)
		status.Summary = strings.Repeat("a", 33)
		if err := protovalidate.Validate(status); err == nil {
			t.Error("expected error for summary length 33, got nil")
		}
	})
	t.Run("detail length 161 fails", func(t *testing.T) {
		t.Parallel()
		status := cloneIntegrationStatus(validConnected)
		status.Detail = strings.Repeat("b", 161)
		if err := protovalidate.Validate(status); err == nil {
			t.Error("expected error for detail length 161, got nil")
		}
	})
	t.Run("empty summary fails", func(t *testing.T) {
		t.Parallel()
		status := cloneIntegrationStatus(validConnected)
		status.Summary = ""
		if err := protovalidate.Validate(status); err == nil {
			t.Error("expected error for empty summary, got nil")
		}
	})
	t.Run("empty detail fails", func(t *testing.T) {
		t.Parallel()
		status := cloneIntegrationStatus(validConnected)
		status.Detail = ""
		if err := protovalidate.Validate(status); err == nil {
			t.Error("expected error for empty detail, got nil")
		}
	})
	t.Run("unspecified status fails", func(t *testing.T) {
		t.Parallel()
		status := cloneIntegrationStatus(validConnected)
		status.Status = IntegrationConnectionStatus_INTEGRATION_CONNECTION_STATUS_UNSPECIFIED
		if err := protovalidate.Validate(status); err == nil {
			t.Error("expected error for UNSPECIFIED status, got nil")
		}
	})
}

func cloneIntegrationStatus(in *IntegrationStatus) *IntegrationStatus {
	return &IntegrationStatus{
		Provider:     in.GetProvider(),
		OrgInstalled: in.GetOrgInstalled(),
		UserBound:    in.GetUserBound(),
		Status:       in.GetStatus(),
		Summary:      in.GetSummary(),
		Detail:       in.GetDetail(),
	}
}
