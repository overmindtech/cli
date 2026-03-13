package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	githubReleasesURL   = "https://api.github.com/repos/overmindtech/cli/releases/latest"
	versionCheckTimeout = 3 * time.Second
)

// githubReleaseResponse represents the response from GitHub API for a release
type githubReleaseResponse struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
}

// checkVersion checks if the current CLI version is out of date by comparing
// it with the latest release from GitHub. Returns the latest version and whether
// an update is available. Errors are logged but not returned to avoid blocking
// command execution.
func checkVersion(ctx context.Context, currentVersion string) (latestVersion string, updateAvailable bool) {
	// Skip check for dev builds
	if currentVersion == "dev" || currentVersion == "" {
		return "", false
	}

	// Create a context with timeout to avoid blocking too long
	checkCtx, cancel := context.WithTimeout(ctx, versionCheckTimeout)
	defer cancel()

	// Timeout is handled by the context timeout above
	client := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, githubReleasesURL, nil)
	if err != nil {
		log.WithError(err).Debug("Failed to create version check request")
		return "", false
	}

	// Set User-Agent to identify the CLI
	req.Header.Set("User-Agent", fmt.Sprintf("overmind-cli/%s", currentVersion))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Debug("Failed to check for CLI updates")
		return "", false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{
			"status_code": resp.StatusCode,
		}).Debug("Failed to check for CLI updates: non-200 response")
		return "", false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Debug("Failed to read version check response")
		return "", false
	}

	var release githubReleaseResponse
	if err := json.Unmarshal(body, &release); err != nil {
		log.WithError(err).Debug("Failed to parse version check response")
		return "", false
	}

	latestVersion = strings.TrimPrefix(release.TagName, "v")
	currentVersionTrimmed := strings.TrimPrefix(currentVersion, "v")

	// Use proper semantic version comparison
	currentSemver, err := semver.NewVersion(currentVersionTrimmed)
	if err != nil {
		log.WithError(err).WithField("version", currentVersionTrimmed).Debug("Failed to parse current version as semver, skipping comparison")
		return latestVersion, false
	}

	latestSemver, err := semver.NewVersion(latestVersion)
	if err != nil {
		log.WithError(err).WithField("version", latestVersion).Debug("Failed to parse latest version as semver, skipping comparison")
		return latestVersion, false
	}

	// Check if latest version is greater than current version
	if latestSemver.GreaterThan(currentSemver) {
		updateAvailable = true
	}

	return latestVersion, updateAvailable
}

// displayVersionWarning displays a warning message if the CLI version is out of date
func displayVersionWarning(ctx context.Context, currentVersion string) {
	latestVersion, updateAvailable := checkVersion(ctx, currentVersion)
	if !updateAvailable {
		return
	}

	// Ensure both versions are displayed with "v" prefix for consistency
	currentDisplay := currentVersion
	if !strings.HasPrefix(currentVersion, "v") && currentVersion != "" {
		currentDisplay = "v" + currentVersion
	}
	latestDisplay := latestVersion
	if !strings.HasPrefix(latestVersion, "v") && latestVersion != "" {
		latestDisplay = "v" + latestVersion
	}

	// Display warning on stderr so it doesn't interfere with command output
	fmt.Fprintf(os.Stderr, "⚠️  Warning: You are using CLI version %s, but version %s is available. Please update to the latest version.\n", currentDisplay, latestDisplay)
}
