package cmd

import (
	"fmt"
	"os"
	"strings"
	"testing"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/muesli/reflow/wordwrap"
)

func TestOSC8Hyperlink(t *testing.T) {
	t.Parallel()

	url := "https://app.overmind.tech/changes/abc/blast-radius?selectedRisk=xyz&utm_source=cli&cli_version=0.42.0"
	text := "View risk ↗"

	// In tests, stdout is not a TTY, so supportsOSCHyperlinks() returns false
	// and osc8Hyperlink falls back to the raw URL.
	result := osc8Hyperlink(url, text)
	if result != url {
		t.Errorf("osc8Hyperlink() = %q, want raw URL %q when stdout is not a TTY", result, url)
	}
}

func TestEnvSupportsOSC8(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want bool
	}{
		{"CI disables", map[string]string{"CI": "true"}, false},
		{"dumb terminal", map[string]string{"TERM": "dumb"}, false},
		{"screen without tmux", map[string]string{"TERM": "screen"}, false},
		{"screen with tmux and 256color", map[string]string{"TERM": "screen-256color", "TMUX": "/tmp/tmux-1000/default,12345,0"}, true},
		{"TERM_PROGRAM set", map[string]string{"TERM_PROGRAM": "iTerm.app"}, true},
		{"VTE_VERSION set", map[string]string{"VTE_VERSION": "6800"}, true},
		{"xterm-kitty", map[string]string{"TERM": "xterm-kitty"}, true},
		{"256color", map[string]string{"TERM": "xterm-256color"}, true},
		{"no signals", map[string]string{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CI", "")
			t.Setenv("TERM", "")
			t.Setenv("TMUX", "")
			t.Setenv("TERM_PROGRAM", "")
			t.Setenv("VTE_VERSION", "")
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if got := envSupportsOSCHyperlinks(); got != tt.want {
				t.Errorf("envSupportsOSCHyperlinks() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRenderRiskPreview prints an exact replica of the CLI risk output using
// the real lipgloss styles and theme. Run from an interactive terminal with:
//
//	go test ./cli/cmd/ -run TestRenderRiskPreview -v
//
// This is a visual inspection test, not an assertion-based test. It formats the
// OSC 8 escape directly because go test pipes stdout through the test runner,
// which fails the TTY check in supportsOSCHyperlinks. The real CLI runs in the user's
// terminal where the TTY check passes naturally.
func TestRenderRiskPreview(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("visual inspection test — skipped in CI")
	}
	InitPalette()

	changeURL := "https://app.overmind.tech/changes/d7f79e24-d123-40f2-9f5d-7296cff5fc7b"
	cliVersion := "0.42.0"

	type fakeRisk struct {
		title       string
		description string
		severity    string
		riskUUID    string
	}

	risks := []fakeRisk{
		{
			title:       "Security group opens port 22 to 0.0.0.0/0",
			description: "Opening SSH to all IPs exposes the instance to brute-force attacks and unauthorized access. The security group sg-0abc123 allows inbound TCP/22 from 0.0.0.0/0, making it reachable from any IP on the internet.",
			severity:    "high",
			riskUUID:    "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		},
		{
			title:       "Load balancer target group has no health check",
			description: "Without health checks, traffic may be routed to unhealthy instances causing user-facing errors. Target group arn:aws:elasticloadbalancing:us-east-1:123456:tg/my-tg has no health check configured.",
			severity:    "medium",
			riskUUID:    "b2c3d4e5-f6a7-8901-bcde-f12345678901",
		},
		{
			title:       "Route table change may affect private subnet connectivity",
			description: "Modifying route table rtb-0def456 could disrupt connectivity for instances in subnet-789ghi that rely on the NAT gateway for outbound traffic.",
			severity:    "low",
			riskUUID:    "c3d4e5f6-a7b8-9012-cdef-123456789012",
		},
	}

	osc8 := func(url, text string) string {
		return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
	}

	bits := []string{"", ""}
	bits = append(bits, styleH1().Render("Potential Risks"))
	bits = append(bits, "")

	for _, r := range risks {
		var severity string
		switch r.severity {
		case "high":
			severity = lipgloss.NewStyle().
				Background(ColorPalette.BgDanger).
				Foreground(ColorPalette.LabelTitle).
				Padding(0, 1).
				Bold(true).
				Render("High ‼")
		case "medium":
			severity = lipgloss.NewStyle().
				Background(ColorPalette.BgWarning).
				Foreground(ColorPalette.LabelTitle).
				Padding(0, 1).
				Render("Medium !")
		case "low":
			severity = lipgloss.NewStyle().
				Background(ColorPalette.LabelBase).
				Foreground(ColorPalette.LabelTitle).
				Padding(0, 1).
				Render("Low ⓘ ")
		}

		title := lipgloss.NewStyle().
			Foreground(ColorPalette.BgMain).
			PaddingRight(1).
			Bold(true).
			Render(r.title)

		bits = append(bits, fmt.Sprintf("%v%v\n\n%v",
			title,
			severity,
			wordwrap.String(r.description, 160)))

		riskURL := fmt.Sprintf("%v/blast-radius?selectedRisk=%v&utm_source=cli&cli_version=%v", changeURL, r.riskUUID, cliVersion)
		bits = append(bits, fmt.Sprintf("%v\n\n", osc8(riskURL, "View risk ↗")))
	}

	changeURLWithUTM := fmt.Sprintf("%v?utm_source=cli&cli_version=%v", changeURL, cliVersion)
	bits = append(bits, fmt.Sprintf("\nView the blast radius graph and risks:\n%v\n\n", osc8(changeURLWithUTM, "Open in Overmind ↗")))

	fmt.Println(strings.Join(bits, "\n"))
}
