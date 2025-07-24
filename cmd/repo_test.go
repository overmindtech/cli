package cmd

import (
	"errors"
	"os"
	"testing"
)

type testDetector struct {
	requiredEnvVarsCallback func() []string
	repoURLCallback         func(map[string]string) (string, error)
}

func (d *testDetector) RequiredEnvVars() []string {
	return d.requiredEnvVarsCallback()
}

func (d *testDetector) DetectRepoURL(envVars map[string]string) (string, error) {
	return d.repoURLCallback(envVars)
}

func TestDetectRepoURL(t *testing.T) {
	t.Parallel()

	t.Run("no detectors", func(t *testing.T) {
		t.Parallel()
		detectors := []RepoDetector{}

		repoURL, err := DetectRepoURL(detectors)
		if err == nil {
			t.Fatal("expected error")
		}
		if repoURL != "" {
			t.Fatalf("expected empty repoURL, got %q", repoURL)
		}
	})

	t.Run("with a failing detector", func(t *testing.T) {
		t.Parallel()
		detectors := []RepoDetector{
			&testDetector{
				requiredEnvVarsCallback: func() []string {
					return []string{"FOO"}
				},
				repoURLCallback: func(map[string]string) (string, error) {
					return "", errors.New("failed to detect repo URL")
				},
			},
		}

		repoURL, err := DetectRepoURL(detectors)
		if err == nil {
			t.Fatal("expected error")
		}
		if repoURL != "" {
			t.Fatalf("expected empty repoURL, got %q", repoURL)
		}
	})

	t.Run("with multiple failing detectors", func(t *testing.T) {
		t.Parallel()
		detectors := []RepoDetector{
			&testDetector{
				requiredEnvVarsCallback: func() []string {
					return []string{"FOO"}
				},
				repoURLCallback: func(map[string]string) (string, error) {
					return "", errors.New("mint")
				},
			},
			&testDetector{
				requiredEnvVarsCallback: func() []string {
					return []string{"BAR"}
				},
				repoURLCallback: func(map[string]string) (string, error) {
					return "", errors.New("choc")
				},
			},
		}

		repoURL, err := DetectRepoURL(detectors)
		if err == nil {
			t.Fatal("expected error")
		}
		if repoURL != "" {
			t.Fatalf("expected empty repoURL, got %q", repoURL)
		}
		if err.Error() != "mint\nchoc" {
			t.Fatalf("expected error to contain both messages, got %q", err.Error())
		}
	})

	t.Run("with a successful detector", func(t *testing.T) {
		t.Parallel()
		detectors := []RepoDetector{
			&testDetector{
				requiredEnvVarsCallback: func() []string {
					return []string{"FOO"}
				},
				repoURLCallback: func(map[string]string) (string, error) {
					return "https://example.com/foo", nil
				},
			},
		}

		repoURL, err := DetectRepoURL(detectors)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repoURL != "https://example.com/foo" {
			t.Fatalf("expected repoURL to be %q, got %q", "https://example.com/foo", repoURL)
		}
	})

	t.Run("with multiple detectors, one successful", func(t *testing.T) {
		t.Parallel()
		detectors := []RepoDetector{
			&testDetector{
				requiredEnvVarsCallback: func() []string {
					return []string{"FOO"}
				},
				repoURLCallback: func(map[string]string) (string, error) {
					return "", nil
				},
			},
			&testDetector{
				requiredEnvVarsCallback: func() []string {
					return []string{"BAR"}
				},
				repoURLCallback: func(map[string]string) (string, error) {
					return "https://example.com/bar", nil
				},
			},
		}

		repoURL, err := DetectRepoURL(detectors)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repoURL != "https://example.com/bar" {
			t.Fatalf("expected repoURL to be %q, got %q", "https://example.com/bar", repoURL)
		}
	})
}

func TestRepoDetectorGithubActions(t *testing.T) {
	t.Parallel()

	t.Run("with valid values", func(t *testing.T) {
		t.Parallel()

		envVars := map[string]string{
			"GITHUB_REPOSITORY": "owner/repo",
			"GITHUB_SERVER_URL": "https://github.com",
		}

		detector := &RepoDetectorGithubActions{}

		repoURL, err := detector.DetectRepoURL(envVars)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedRepoUrl := "https://github.com/owner/repo"
		if repoURL != expectedRepoUrl {
			t.Fatalf("expected repoURL to be %q, got %q", expectedRepoUrl, repoURL)
		}
	})

	t.Run("with missing GITHUB_REPOSITORY", func(t *testing.T) {
		t.Parallel()

		envVars := map[string]string{
			"GITHUB_SERVER_URL": "https://github.com",
		}

		detector := &RepoDetectorGithubActions{}

		repoURL, err := detector.DetectRepoURL(envVars)
		if err == nil {
			t.Fatal("expected error")
		}
		if repoURL != "" {
			t.Fatalf("expected empty repoURL, got %q", repoURL)
		}
	})
}

func TestRepoDetectorJenkins(t *testing.T) {
	t.Parallel()
	t.Run("with valid GIT_URL", func(t *testing.T) {
		t.Parallel()
		envVars := map[string]string{
			"GIT_URL": "https://example.com/repo.git",
		}
		detector := &RepoDetectorJenkins{}
		repoURL, err := detector.DetectRepoURL(envVars)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedRepoUrl := "https://example.com/repo.git"
		if repoURL != expectedRepoUrl {
			t.Fatalf("expected repoURL to be %q, got %q", expectedRepoUrl, repoURL)
		}
	})

	t.Run("missing GIT_URL", func(t *testing.T) {
		t.Parallel()
		envVars := map[string]string{}
		detector := &RepoDetectorJenkins{}
		_, err := detector.DetectRepoURL(envVars)
		if err == nil {
			t.Fatal("expected error")
		}
		expectedError := "GIT_URL not set"
		if err.Error() != expectedError {
			t.Fatalf("expected error to be %q, got %q", expectedError, err.Error())
		}
	})
}

func TestRepoDetectorGitlab(t *testing.T) {
	t.Parallel()
	t.Run("with valid CI_SERVER_URL and CI_PROJECT_PATH", func(t *testing.T) {
		t.Parallel()
		envVars := map[string]string{
			"CI_SERVER_URL":   "https://gitlab.com",
			"CI_PROJECT_PATH": "owner/repo",
		}
		detector := &RepoDetectorGitlab{}
		repoURL, err := detector.DetectRepoURL(envVars)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedRepoUrl := "https://gitlab.com/owner/repo"
		if repoURL != expectedRepoUrl {
			t.Fatalf("expected repoURL to be %q, got %q", expectedRepoUrl, repoURL)
		}
	})

	t.Run("missing CI_SERVER_URL", func(t *testing.T) {
		t.Parallel()
		envVars := map[string]string{
			"CI_PROJECT_PATH": "owner/repo",
		}
		detector := &RepoDetectorGitlab{}
		_, err := detector.DetectRepoURL(envVars)
		if err == nil {
			t.Fatal("expected error")
		}
		expectedError := "CI_SERVER_URL not set"
		if err.Error() != expectedError {
			t.Fatalf("expected error to be %q, got %q", expectedError, err.Error())
		}
	})

	t.Run("missing CI_PROJECT_PATH", func(t *testing.T) {
		t.Parallel()
		envVars := map[string]string{
			"CI_SERVER_URL": "https://gitlab.com",
		}
		detector := &RepoDetectorGitlab{}
		_, err := detector.DetectRepoURL(envVars)
		if err == nil {
			t.Fatal("expected error")
		}
		expectedError := "CI_PROJECT_PATH not set"
		if err.Error() != expectedError {
			t.Fatalf("expected error to be %q, got %q", expectedError, err.Error())
		}
	})
}

func TestRepoDetectorCircleCI(t *testing.T) {
	t.Parallel()
	t.Run("with valid CIRCLE_REPOSITORY_URL", func(t *testing.T) {
		t.Parallel()
		envVars := map[string]string{
			"CIRCLE_REPOSITORY_URL": "https://example.com/repo.git",
		}
		detector := &RepoDetectorCircleCI{}
		repoURL, err := detector.DetectRepoURL(envVars)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedRepoUrl := "https://example.com/repo.git"
		if repoURL != expectedRepoUrl {
			t.Fatalf("expected repoURL to be %q, got %q", expectedRepoUrl, repoURL)
		}
	})

	t.Run("missing CIRCLE_REPOSITORY_URL", func(t *testing.T) {
		t.Parallel()
		envVars := map[string]string{}
		detector := &RepoDetectorCircleCI{}
		_, err := detector.DetectRepoURL(envVars)
		if err == nil {
			t.Fatal("expected error")
		}
		expectedError := "CIRCLE_REPOSITORY_URL not set"
		if err.Error() != expectedError {
			t.Fatalf("expected error to be %q, got %q", expectedError, err.Error())
		}
	})
}

func TestRepoDetectorAzureDevOps(t *testing.T) {
	t.Parallel()
	t.Run("with valid BUILD_REPOSITORY_URI", func(t *testing.T) {
		t.Parallel()
		envVars := map[string]string{
			"BUILD_REPOSITORY_URI": "https://dev.azure.com/organization/project/_git/repo",
		}
		detector := &RepoDetectorAzureDevOps{}
		repoURL, err := detector.DetectRepoURL(envVars)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedRepoUrl := "https://dev.azure.com/organization/project/_git/repo"
		if repoURL != expectedRepoUrl {
			t.Fatalf("expected repoURL to be %q, got %q", expectedRepoUrl, repoURL)
		}
	})

	t.Run("missing BUILD_REPOSITORY_URI", func(t *testing.T) {
		t.Parallel()
		envVars := map[string]string{}
		detector := &RepoDetectorAzureDevOps{}
		_, err := detector.DetectRepoURL(envVars)
		if err == nil {
			t.Fatal("expected error")
		}
		expectedError := "BUILD_REPOSITORY_URI not set"
		if err.Error() != expectedError {
			t.Fatalf("expected error to be %q, got %q", expectedError, err.Error())
		}
	})
}

func TestRepoDetectorGitConfig(t *testing.T) {
	t.Parallel()

	t.Run("With a simple gitconfig", func(t *testing.T) {
		t.Parallel()

		gitconfig := `[core]
        repositoryformatversion = 0
        filemode = true
        bare = false
        logallrefupdates = true
        ignorecase = true
        precomposeunicode = true
[remote "origin"]
        url = git@github.com:overmindtech/cli.git`

		// Write gitconfig to a temporary file
		gitConfigFile, err := os.CreateTemp("", "gitconfig")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		t.Cleanup(func() {
			os.Remove(gitConfigFile.Name())
		})

		_, err = gitConfigFile.WriteString(gitconfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		detector := RepoDetectorGitConfig{
			gitconfigPath: gitConfigFile.Name(),
		}

		url, err := detector.DetectRepoURL(map[string]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedUrl := "git@github.com:overmindtech/cli.git"

		if url != expectedUrl {
			t.Fatalf("expected url to be %q, got %q", expectedUrl, url)
		}
	})

	t.Run("with no gitconfig", func(t *testing.T) {
		t.Parallel()

		detector := RepoDetectorGitConfig{
			gitconfigPath: "nonexistent-path",
		}

		_, err := detector.DetectRepoURL(map[string]string{})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("with a gitconfig with no remote", func(t *testing.T) {
		t.Parallel()

		gitconfig := `[core]
		repositoryformatversion = 0
		filemode = true
		bare = false
		logallrefupdates = true
		ignorecase = true
		precomposeunicode = true`

		// Write gitconfig to a temporary file
		gitConfigFile, err := os.CreateTemp("", "gitconfig")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		t.Cleanup(func() {
			os.Remove(gitConfigFile.Name())
		})

		_, err = gitConfigFile.WriteString(gitconfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		detector := RepoDetectorGitConfig{
			gitconfigPath: gitConfigFile.Name(),
		}

		_, err = detector.DetectRepoURL(map[string]string{})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("with an empty gitconfig", func(t *testing.T) {
		t.Parallel()

		gitconfig := ``

		// Write gitconfig to a temporary file
		gitConfigFile, err := os.CreateTemp("", "gitconfig")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		t.Cleanup(func() {
			os.Remove(gitConfigFile.Name())
		})

		_, err = gitConfigFile.WriteString(gitconfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		detector := RepoDetectorGitConfig{
			gitconfigPath: gitConfigFile.Name(),
		}

		_, err = detector.DetectRepoURL(map[string]string{})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("with a gitconfig that isn't a valid ini file", func(t *testing.T) {
		t.Parallel()

		gitconfig := `not a valid ini file! =======`

		// Write gitconfig to a temporary file
		gitConfigFile, err := os.CreateTemp("", "gitconfig")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		t.Cleanup(func() {
			os.Remove(gitConfigFile.Name())
		})

		_, err = gitConfigFile.WriteString(gitconfig)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		detector := RepoDetectorGitConfig{
			gitconfigPath: gitConfigFile.Name(),
		}

		_, err = detector.DetectRepoURL(map[string]string{})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestRepoDetectorScalr(t *testing.T) {
	t.Parallel()

	t.Run("with valid SCALR_WORKSPACE_NAME and SCALR_ENVIRONMENT_NAME", func(t *testing.T) {
		t.Parallel()

		envVars := map[string]string{
			"SCALR_WORKSPACE_NAME":   "my-workspace",
			"SCALR_ENVIRONMENT_NAME": "production",
		}

		detector := &RepoDetectorScalr{}

		repoURL, err := detector.DetectRepoURL(envVars)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedRepoURL := "scalr://production/my-workspace"
		if repoURL != expectedRepoURL {
			t.Fatalf("expected repoURL to be %q, got %q", expectedRepoURL, repoURL)
		}
	})

	t.Run("with missing SCALR_WORKSPACE_NAME", func(t *testing.T) {
		t.Parallel()

		envVars := map[string]string{
			"SCALR_ENVIRONMENT_NAME": "production",
		}

		detector := &RepoDetectorScalr{}

		repoURL, err := detector.DetectRepoURL(envVars)
		if err == nil {
			t.Fatal("expected error")
		}
		if repoURL != "" {
			t.Fatalf("expected empty repoURL, got %q", repoURL)
		}

		expectedError := "SCALR_WORKSPACE_NAME not set"
		if err.Error() != expectedError {
			t.Fatalf("expected error to be %q, got %q", expectedError, err.Error())
		}
	})

	t.Run("with missing SCALR_ENVIRONMENT_NAME", func(t *testing.T) {
		t.Parallel()

		envVars := map[string]string{
			"SCALR_WORKSPACE_NAME": "my-workspace",
		}

		detector := &RepoDetectorScalr{}

		repoURL, err := detector.DetectRepoURL(envVars)
		if err == nil {
			t.Fatal("expected error")
		}
		if repoURL != "" {
			t.Fatalf("expected empty repoURL, got %q", repoURL)
		}

		expectedError := "SCALR_ENVIRONMENT_NAME not set"
		if err.Error() != expectedError {
			t.Fatalf("expected error to be %q, got %q", expectedError, err.Error())
		}
	})

	t.Run("with both variables missing", func(t *testing.T) {
		t.Parallel()

		envVars := map[string]string{}

		detector := &RepoDetectorScalr{}

		repoURL, err := detector.DetectRepoURL(envVars)
		if err == nil {
			t.Fatal("expected error")
		}
		if repoURL != "" {
			t.Fatalf("expected empty repoURL, got %q", repoURL)
		}

		expectedError := "SCALR_WORKSPACE_NAME not set"
		if err.Error() != expectedError {
			t.Fatalf("expected error to be %q, got %q", expectedError, err.Error())
		}
	})

	t.Run("with empty values", func(t *testing.T) {
		t.Parallel()

		envVars := map[string]string{
			"SCALR_WORKSPACE_NAME":   "",
			"SCALR_ENVIRONMENT_NAME": "",
		}

		detector := &RepoDetectorScalr{}

		repoURL, err := detector.DetectRepoURL(envVars)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedRepoURL := "scalr:///"
		if repoURL != expectedRepoURL {
			t.Fatalf("expected repoURL to be %q, got %q", expectedRepoURL, repoURL)
		}
	})

	t.Run("with special characters", func(t *testing.T) {
		t.Parallel()

		envVars := map[string]string{
			"SCALR_WORKSPACE_NAME":   "my-workspace-with-dashes_and_underscores",
			"SCALR_ENVIRONMENT_NAME": "prod-env_123",
		}

		detector := &RepoDetectorScalr{}

		repoURL, err := detector.DetectRepoURL(envVars)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedRepoURL := "scalr://prod-env_123/my-workspace-with-dashes_and_underscores"
		if repoURL != expectedRepoURL {
			t.Fatalf("expected repoURL to be %q, got %q", expectedRepoURL, repoURL)
		}
	})
}
