package tfutils

import (
	"net/url"
)

// This converts a repo value to a scope that can be used for Terraform changes
// that aren't mapped to a specific resource. Even if we can't map these
// changes, we want the GloballyUniqueName to sill be unique, so we need to
// include the repo as it's common for customers to have many repos or
// workspaces that could have a clashing names in Terraform. Think of a resource
// like "aws_instance.app_server". This is a common name and absolutely could
// clash with another resource in another repo or workspace.
func RepoToScope(repo string) string {
	// If repo is empty, use a fallback scope to ensure items have a scope
	if repo == "" {
		return "terraform_plan"
	}

	parsed, err := url.Parse(repo)
	if err != nil {
		return repo
	}

	// Remove the scheme (http, https, etc.) if it exists
	return parsed.Host + parsed.Path
}
