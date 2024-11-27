package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/ini.v1"
)

var AllDetectors = []RepoDetector{
	&RepoDetectorGithubActions{},
	&RepoDetectorJenkins{},
	&RepoDetectorGitlab{},
	&RepoDetectorCircleCI{},
	&RepoDetectorAzureDevOps{},
	&RepoDetectorSpacelift{},
	&RepoDetectorGitConfig{},
}

// Detects the URL of the repository that the user is working in based on the
// environment variables that are set in the user's shell. You should usually
// pass in `AllDetectors` to this function, though you can pass in a subset of
// detectors if you want to.
//
// Returns the URL of the repository that the user is working in, or an error if
// the URL could not be detected.
func DetectRepoURL(detectors []RepoDetector) (string, error) {
	var errs []error

	for _, detector := range detectors {
		if detector == nil {
			continue
		}

		envVars := make(map[string]string)
		for _, requiredVar := range detector.RequiredEnvVars() {
			if val, ok := os.LookupEnv(requiredVar); !ok {
				// If any of the required environment variables are not set, move on to the next detector
				break
			} else {
				envVars[requiredVar] = val
			}
		}

		repoURL, err := detector.DetectRepoURL(envVars)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if repoURL == "" {
			continue
		}

		return repoURL, nil
	}

	if len(errs) > 0 {
		return "", errors.Join(errs...)
	}

	return "", errors.New("no repository URL detected")
}

// RepoDetector is an interface for detecting the URL of the repository that the
// user is working in. Implementations should be able to detect the URL of the
// repository based on the environment variables that are set in the user's
// shell.
type RepoDetector interface {
	// Returns a list of environment variables that are required for the
	// implementation to detect the repository URL.
	//
	// This detector will only be run if all variables are present. If this is
	// an empty slice the detector will always run.
	RequiredEnvVars() []string

	// DetectRepoURL detects the URL of the repository that the user is working
	// in based on the environment variables that are set. The set of
	// environment variables that were returned by RequiredEnvVars() will be
	// passed in as a map, along with their values.
	//
	// This means that if RequiredEnvVars() returns ["GIT_DIR"], then
	// DetectRepoURL will be called with a map containing the value of the
	// GIT_DIR environment variable. i.e. envVars["GIT_DIR"] will contain the
	// value of the GIT_DIR environment variable.
	DetectRepoURL(envVars map[string]string) (string, error)
}

// Detects the repository URL based on the environment variables that are set in
// Github Actions by default.
//
// https://docs.github.com/en/actions/writing-workflows/choosing-what-your-workflow-does/store-information-in-variables
type RepoDetectorGithubActions struct{}

func (d *RepoDetectorGithubActions) RequiredEnvVars() []string {
	return []string{"GITHUB_SERVER_URL", "GITHUB_REPOSITORY"}
}

func (d *RepoDetectorGithubActions) DetectRepoURL(envVars map[string]string) (string, error) {
	serverURL, ok := envVars["GITHUB_SERVER_URL"]
	if !ok {
		return "", errors.New("GITHUB_SERVER_URL not set")
	}

	repo, ok := envVars["GITHUB_REPOSITORY"]
	if !ok {
		return "", errors.New("GITHUB_REPOSITORY not set")
	}

	return serverURL + "/" + repo, nil
}

// Detects the repository URL based on the environment variables that are set in
// Jenkins Git plugin by default.
//
// https://wiki.jenkins.io/JENKINS/Git-Plugin.html
type RepoDetectorJenkins struct{}

func (d *RepoDetectorJenkins) RequiredEnvVars() []string {
	return []string{"GIT_URL"}
}

func (d *RepoDetectorJenkins) DetectRepoURL(envVars map[string]string) (string, error) {
	gitURL, ok := envVars["GIT_URL"]
	if !ok {
		return "", errors.New("GIT_URL not set")
	}

	return gitURL, nil
}

// Detects the repository URL based on teh default env vars from Gitlab
//
// https://docs.gitlab.com/ee/ci/variables/predefined_variables.html
type RepoDetectorGitlab struct{}

func (d *RepoDetectorGitlab) RequiredEnvVars() []string {
	return []string{"CI_SERVER_URL", "CI_PROJECT_PATH"}
}

func (d *RepoDetectorGitlab) DetectRepoURL(envVars map[string]string) (string, error) {
	serverURL, ok := envVars["CI_SERVER_URL"]
	if !ok {
		return "", errors.New("CI_SERVER_URL not set")
	}

	projectPath, ok := envVars["CI_PROJECT_PATH"]
	if !ok {
		return "", errors.New("CI_PROJECT_PATH not set")
	}

	return serverURL + "/" + projectPath, nil
}

// Detects the repository URL based on the environment variables that are set in
// CircleCI by default.
//
// https://circleci.com/docs/variables/
type RepoDetectorCircleCI struct{}

func (d *RepoDetectorCircleCI) RequiredEnvVars() []string {
	return []string{"CIRCLE_REPOSITORY_URL"}
}

func (d *RepoDetectorCircleCI) DetectRepoURL(envVars map[string]string) (string, error) {
	repoURL, ok := envVars["CIRCLE_REPOSITORY_URL"]
	if !ok {
		return "", errors.New("CIRCLE_REPOSITORY_URL not set")
	}

	return repoURL, nil
}

// Detects the repository URL based on the environment variables that are set in
// Azure DevOps by default.
//
// https://learn.microsoft.com/en-us/azure/devops/pipelines/build/variables?view=azure-devops
type RepoDetectorAzureDevOps struct{}

func (d *RepoDetectorAzureDevOps) RequiredEnvVars() []string {
	return []string{"BUILD_REPOSITORY_URI"}
}

func (d *RepoDetectorAzureDevOps) DetectRepoURL(envVars map[string]string) (string, error) {
	repoURL, ok := envVars["BUILD_REPOSITORY_URI"]
	if !ok {
		return "", errors.New("BUILD_REPOSITORY_URI not set")
	}

	return repoURL, nil
}

// Detects the repository URL based on the environment variables that are set in
// Spacelift by default.
//
// https://docs.spacelift.io/concepts/configuration/environment.html#environment-variables
//
// Note that since Spacelift doesn't expose the full URL, you just get the last
// bit i.e. username/repo
type RepoDetectorSpacelift struct{}

func (d *RepoDetectorSpacelift) RequiredEnvVars() []string {
	return []string{"TF_VAR_spacelift_repository"}
}

func (d *RepoDetectorSpacelift) DetectRepoURL(envVars map[string]string) (string, error) {
	repoURL, ok := envVars["TF_VAR_spacelift_repository"]
	if !ok {
		return "", errors.New("TF_VAR_spacelift_repository not set")
	}

	return repoURL, nil
}

type RepoDetectorGitConfig struct {
	// Optional override path to the gitconfig file, only used for testing
	gitconfigPath string
}

func (d *RepoDetectorGitConfig) RequiredEnvVars() []string {
	return []string{""}
}

// Load the .git/config file and extract the remote URL from it
func (d *RepoDetectorGitConfig) DetectRepoURL(envVars map[string]string) (string, error) {
	var gitConfigPath string
	if d.gitconfigPath != "" {
		gitConfigPath = d.gitconfigPath
	} else {
		gitConfigPath = ".git/config"
	}

	// Try to read the .git/config file
	gitConfig, err := ini.Load(gitConfigPath)
	if err != nil {
		return "", fmt.Errorf("could not open .git/config to determine repo: %w", err)
	}

	for _, section := range gitConfig.Sections() {
		if strings.HasPrefix(section.Name(), "remote") {
			urlKey, err := section.GetKey("url")
			if err != nil {
				continue
			}

			return urlKey.String(), nil
		}
	}

	return "", fmt.Errorf("could not find remote URL in %v", gitConfigPath)
}
