package cmd

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/overmindtech/sdp-go"
	"golang.org/x/oauth2"
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
	claimBytes, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("unexpected fail marshalling claims: %v", err)
	}

	fakeAccessToken := fmt.Sprintf(".%v.", base64.RawURLEncoding.EncodeToString(claimBytes))
	token := &oauth2.Token{
		AccessToken:  fakeAccessToken,
		TokenType:    "",
		RefreshToken: "",
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
			if pass, _, _ := HasScopesFlexible(token, tc.RequiredScopes); pass != tc.ShouldPass {
				t.Fatalf("expected: %v, got: %v", tc.ShouldPass, !tc.ShouldPass)
			}
		})
	}
}

func Test_getAppUrl(t *testing.T) {
	type args struct {
		frontend string
		app      string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "empty", args: args{frontend: "", app: ""}, want: "https://app.overmind.tech"},
		{name: "empty app", args: args{frontend: "https://app.overmind.tech", app: ""}, want: "https://app.overmind.tech"},
		{name: "empty frontend", args: args{frontend: "", app: "https://app.overmind.tech"}, want: "https://app.overmind.tech"},
		{name: "same", args: args{frontend: "https://app.overmind.tech", app: "https://app.overmind.tech"}, want: "https://app.overmind.tech"},
		{name: "different", args: args{frontend: "https://app.overmind.tech", app: "https://app.overmind.tech/changes/123"}, want: "https://app.overmind.tech/changes/123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAppUrl(tt.args.frontend, tt.args.app)
			if got != tt.want {
				t.Errorf("getAppUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}
