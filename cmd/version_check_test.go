package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckVersion(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion string
		latestTag      string
		wantUpdate     bool
		skipCheck      bool
	}{
		{
			name:           "outdated version",
			currentVersion: "1.0.0",
			latestTag:      "v1.0.1",
			wantUpdate:     true,
		},
		{
			name:           "current version",
			currentVersion: "1.0.1",
			latestTag:      "v1.0.1",
			wantUpdate:     false,
		},
		{
			name:           "dev version skipped",
			currentVersion: "dev",
			latestTag:      "v1.0.1",
			wantUpdate:     false,
			skipCheck:      true,
		},
		{
			name:           "empty version skipped",
			currentVersion: "",
			latestTag:      "v1.0.1",
			wantUpdate:     false,
			skipCheck:      true,
		},
		{
			name:           "version with v prefix",
			currentVersion: "v1.0.0",
			latestTag:      "v1.0.1",
			wantUpdate:     true,
		},
		{
			name:           "multi-digit minor version - user newer",
			currentVersion: "1.10.0",
			latestTag:      "v1.9.0",
			wantUpdate:     false,
		},
		{
			name:           "multi-digit minor version - update available",
			currentVersion: "1.9.0",
			latestTag:      "v1.10.0",
			wantUpdate:     true,
		},
		{
			name:           "multi-digit patch version - user newer",
			currentVersion: "1.0.10",
			latestTag:      "v1.0.9",
			wantUpdate:     false,
		},
		{
			name:           "multi-digit patch version - update available",
			currentVersion: "1.0.9",
			latestTag:      "v1.0.10",
			wantUpdate:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/repos/overmindtech/cli/releases/latest" {
					t.Errorf("Expected path /repos/overmindtech/cli/releases/latest, got %s", r.URL.Path)
				}

				release := githubReleaseResponse{
					TagName: tt.latestTag,
					Name:    tt.latestTag,
				}
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(release); err != nil {
					t.Errorf("Failed to encode release: %v", err)
				}
			}))
			defer server.Close()

			// Temporarily override the GitHub URL for testing
			originalURL := githubReleasesURL
			githubReleasesURL = server.URL + "/repos/overmindtech/cli/releases/latest"
			defer func() { githubReleasesURL = originalURL }()

			ctx := context.Background()
			latestVersion, updateAvailable := checkVersion(ctx, tt.currentVersion)

			if tt.skipCheck {
				if latestVersion != "" || updateAvailable {
					t.Errorf("Expected check to be skipped, but got latestVersion=%s, updateAvailable=%v", latestVersion, updateAvailable)
				}
				return
			}

			if updateAvailable != tt.wantUpdate {
				t.Errorf("checkVersion() updateAvailable = %v, want %v", updateAvailable, tt.wantUpdate)
			}

			if tt.wantUpdate && latestVersion == "" {
				t.Errorf("checkVersion() expected latestVersion to be set when update is available")
			}
		})
	}
}

func TestCheckVersionErrorScenarios(t *testing.T) {
	tests := []struct {
		name           string
		currentVersion string
		setupServer    func() *httptest.Server
		wantUpdate     bool
		wantVersion    string
	}{
		{
			name:           "network error - server unreachable",
			currentVersion: "1.0.0",
			setupServer: func() *httptest.Server {
				// Return nil to simulate unreachable server
				return nil
			},
			wantUpdate:  false,
			wantVersion: "",
		},
		{
			name:           "HTTP 404 not found",
			currentVersion: "1.0.0",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte("Not Found"))
				}))
			},
			wantUpdate:  false,
			wantVersion: "",
		},
		{
			name:           "HTTP 500 internal server error",
			currentVersion: "1.0.0",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte("Internal Server Error"))
				}))
			},
			wantUpdate:  false,
			wantVersion: "",
		},
		{
			name:           "malformed JSON response",
			currentVersion: "1.0.0",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{"invalid": json}`))
				}))
			},
			wantUpdate:  false,
			wantVersion: "",
		},
		{
			name:           "empty JSON response",
			currentVersion: "1.0.0",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(`{}`))
				}))
			},
			wantUpdate:  false,
			wantVersion: "",
		},
		{
			name:           "invalid semver in response",
			currentVersion: "1.0.0",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					release := githubReleaseResponse{
						TagName: "not-a-version",
						Name:    "not-a-version",
					}
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					_ = json.NewEncoder(w).Encode(release)
				}))
			},
			wantUpdate:  false,
			wantVersion: "not-a-version",
		},
		{
			name:           "timeout - server delays response",
			currentVersion: "1.0.0",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Delay longer than the timeout (3 seconds)
					time.Sleep(4 * time.Second)
					release := githubReleaseResponse{
						TagName: "v1.0.1",
						Name:    "v1.0.1",
					}
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(release)
				}))
			},
			wantUpdate:  false,
			wantVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Temporarily override the GitHub URL for testing
			originalURL := githubReleasesURL
			defer func() { githubReleasesURL = originalURL }()

			var server *httptest.Server
			if tt.setupServer != nil {
				server = tt.setupServer()
				if server != nil {
					defer server.Close()
					githubReleasesURL = server.URL + "/repos/overmindtech/cli/releases/latest"
				} else {
					// For network error test, use an invalid URL
					githubReleasesURL = "http://localhost:0/repos/overmindtech/cli/releases/latest"
				}
			}

			ctx := context.Background()
			latestVersion, updateAvailable := checkVersion(ctx, tt.currentVersion)

			if updateAvailable != tt.wantUpdate {
				t.Errorf("checkVersion() updateAvailable = %v, want %v", updateAvailable, tt.wantUpdate)
			}

			if latestVersion != tt.wantVersion {
				t.Errorf("checkVersion() latestVersion = %q, want %q", latestVersion, tt.wantVersion)
			}
		})
	}
}
