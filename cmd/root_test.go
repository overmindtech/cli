package cmd

import (
	"testing"

	"github.com/overmindtech/sdp-go"
)

func TestParseChangeUrl(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "https://app.overmind.tech/changes/3e717be8-2478-4938-aa9e-70496d496904", want: "3e717be8-2478-4938-aa9e-70496d496904"},
		{input: "https://app.overmind.tech/changes/b4454604-b92a-41a7-9f0d-fa66063a7c74/", want: "b4454604-b92a-41a7-9f0d-fa66063a7c74"},
		{input: "https://app.overmind.tech/changes/c36f1af4-d55c-4f63-937b-ac5ede7a0cc9/blast-radius", want: "c36f1af4-d55c-4f63-937b-ac5ede7a0cc9"},
	}

	for _, tc := range tests {
		u, err := parseChangeUrl(tc.input)
		if err != nil {
			t.Fatalf("unexpected fail: %v", err)
		}
		if u.String() != tc.want {
			t.Fatalf("expected: %v, got: %v", tc.want, u)
		}
	}
}

func TestHasScopesFlexible(t *testing.T) {
	claims := &sdp.CustomClaims{
		Scope:       "changes:read users:write",
		AccountName: "test",
	}

	tests := []struct {
		Name           string
		RequiredScopes []string
		ShouldPass     bool
	}{
		{
			Name:           "Same scope",
			RequiredScopes: []string{"changes:read"},
			ShouldPass:     true,
		},
		{
			Name:           "Multiple scopes",
			RequiredScopes: []string{"changes:read", "users:write"},
			ShouldPass:     true,
		},
		{
			Name:           "Missing scope",
			RequiredScopes: []string{"changes:read", "users:write", "colours:create"},
			ShouldPass:     false,
		},
		{
			Name:           "Write instead of read",
			RequiredScopes: []string{"users:read"},
			ShouldPass:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			if pass, _ := HasScopesFlexible(claims, tc.RequiredScopes); pass != tc.ShouldPass {
				t.Fatalf("expected: %v, got: %v", tc.ShouldPass, !tc.ShouldPass)
			}
		})
	}
}
