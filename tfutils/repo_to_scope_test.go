package tfutils

import "testing"

func TestRepoToScope(t *testing.T) {
	tests := []struct {
		name     string
		repo     string
		expected string
	}{
		{
			name:     "https URL",
			repo:     "https://github.com/overmindtech/workspace",
			expected: "github.com/overmindtech/workspace",
		},
		{
			name:     "http URL",
			repo:     "http://github.com/overmindtech/workspace",
			expected: "github.com/overmindtech/workspace",
		},
		{
			name:     "URL without protocol",
			repo:     "github.com/overmindtech/workspace",
			expected: "github.com/overmindtech/workspace",
		},
		{
			name:     "GitLab https URL",
			repo:     "https://gitlab.com/company/project",
			expected: "gitlab.com/company/project",
		},
		{
			name:     "GitLab http URL",
			repo:     "http://gitlab.com/company/project",
			expected: "gitlab.com/company/project",
		},
		{
			name:     "Bitbucket URL",
			repo:     "https://bitbucket.org/team/repo",
			expected: "bitbucket.org/team/repo",
		},
		{
			name:     "Self-hosted Git with https",
			repo:     "https://git.company.com/team/project",
			expected: "git.company.com/team/project",
		},
		{
			name:     "Self-hosted Git with http",
			repo:     "http://git.internal.local/repo",
			expected: "git.internal.local/repo",
		},
		{
			name:     "URL with port",
			repo:     "https://git.company.com:8080/team/project",
			expected: "git.company.com:8080/team/project",
		},
		{
			name:     "URL with path and query params",
			repo:     "https://github.com/overmindtech/workspace.git?ref=main",
			expected: "github.com/overmindtech/workspace.git",
		},
		{
			name:     "URL with trailing slash",
			repo:     "https://github.com/overmindtech/workspace/",
			expected: "github.com/overmindtech/workspace/",
		},
		{
			name:     "Supports custom protocols",
			repo:     "custom://github.com/overmindtech/workspace",
			expected: "github.com/overmindtech/workspace",
		},
		{
			name:     "Empty string",
			repo:     "",
			expected: "",
		},
		{
			name:     "Case sensitivity test",
			repo:     "HTTPS://GitHub.com/User/Repo",
			expected: "GitHub.com/User/Repo",
		},
		{
			name:     "SSH URL (should remain unchanged)",
			repo:     "git@github.com:overmindtech/workspace.git",
			expected: "git@github.com:overmindtech/workspace.git",
		},
		{
			name:     "File path (should remain unchanged)",
			repo:     "/local/path/to/repo",
			expected: "/local/path/to/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RepoToScope(tt.repo)
			if result != tt.expected {
				t.Errorf("RepoToScope(%q) = %q, expected %q", tt.repo, result, tt.expected)
			}
		})
	}
}
