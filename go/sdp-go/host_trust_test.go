package sdp

import "testing"

func TestIsTrustedHost(t *testing.T) {
	tests := []struct {
		host    string
		trusted bool
	}{
		// Trusted Overmind domains (callers must pass hostname without port)
		{"app.overmind.tech", true},
		{"api.overmind.tech", true},
		{"overmind.tech", true},
		{"df.overmind-demo.com", true},
		{"staging.overmind-demo.com", true},
		{"overmind-demo.com", true},

		// Case insensitive
		{"APP.OVERMIND.TECH", true},
		{"DF.Overmind-Demo.Com", true},

		// Localhost variants
		{"localhost", true},
		{"127.0.0.1", true},
		{"127.0.0.2", true},
		{"127.255.255.254", true},
		{"::1", true},

		// Untrusted domains
		{"evil.com", false},
		{"attacker.io", false},
		{"overmind.tech.evil.com", false},
		{"notovermind.tech", false},
		{"fakeovermind-demo.com", false},
		{"overmind-demo.com.evil.com", false},

		// Sneaky substrings that should not match
		{"xovermind.tech", false},
		{"xovermind-demo.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := IsTrustedHost(tt.host)
			if got != tt.trusted {
				t.Errorf("IsTrustedHost(%q) = %v, want %v", tt.host, got, tt.trusted)
			}
		})
	}
}

func TestIsLocalHost(t *testing.T) {
	tests := []struct {
		host  string
		local bool
	}{
		{"localhost", true},
		{"127.0.0.1", true},
		{"127.0.0.2", true},
		{"127.255.255.254", true},
		{"::1", true},
		{"app.overmind.tech", false},
		{"evil.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := IsLocalHost(tt.host)
			if got != tt.local {
				t.Errorf("IsLocalHost(%q) = %v, want %v", tt.host, got, tt.local)
			}
		})
	}
}

func TestValidateAppURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"https production", "https://app.overmind.tech", false},
		{"https dogfood", "https://df.overmind-demo.com", false},
		{"http localhost", "http://localhost:3000", false},
		{"http 127.0.0.1", "http://127.0.0.1:8080", false},
		{"http ipv6 loopback", "http://[::1]", false},
		{"http ipv6 loopback with port", "http://[::1]:8080", false},

		// HTTP to non-local is rejected
		{"http remote", "http://app.overmind.tech", true},
		{"http evil", "http://evil.com", true},

		// Invalid URL
		{"invalid", "://bad", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ValidateAppURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAppURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}
