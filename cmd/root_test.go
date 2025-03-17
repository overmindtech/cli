package cmd

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/overmindtech/cli/auth"
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
	claims := &auth.CustomClaims{
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

func TestSaveTokenFile(t *testing.T) {
	// Setup temporary directory for testing
	tempDir := t.TempDir()
	app := "https://localhost.df.overmind-demo.com:3000"

	claims := auth.CustomClaims{
		Scope:       "scope1 scope2",
		AccountName: "test",
	}
	jsonClaims, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("unexpected fail marshalling claims: %v", err)
	}
	claimsSection := base64.RawURLEncoding.EncodeToString([]byte(jsonClaims))
	accessToken := fmt.Sprintf("%s.%s.%s", "header", claimsSection, "signature")
	token := &oauth2.Token{
		AccessToken: accessToken,
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	// Test saving the token file
	err = saveLocalTokenFile(tempDir, app, token)
	if err != nil {
		t.Fatalf("unexpected fail saving token file: %v", err)
	}
	// Test reading the token file
	readAppToken, readClaims, err := readLocalTokenFile(tempDir, app, nil)
	if err != nil {
		t.Fatalf("unexpected fail reading token file: %v", err)
	}
	if readAppToken.AccessToken != token.AccessToken {
		t.Fatalf("expected: %v, got: %v", token.AccessToken, readAppToken.AccessToken)
	}
	if readClaims[0] != "scope1" {
		t.Fatalf("expected: %v, got: %v", "scope1", readClaims[0])
	}
	if readClaims[1] != "scope2" {
		t.Fatalf("expected: %v, got: %v", "scope2", readClaims[1])
	}

	// lets read a token from a non existent app
	nonExistentToken, _, err := readLocalTokenFile(tempDir, "otherApp", nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if nonExistentToken == readAppToken {
		t.Fatalf("expected different tokens, got the same")
	}

	// lets write the token to a different app
	otherApp := "otherApp"
	err = saveLocalTokenFile(tempDir, otherApp, token)
	if err != nil {
		t.Fatalf("unexpected fail saving token file: %v", err)
	}
	readAppToken, _, err = readLocalTokenFile(tempDir, otherApp, nil)
	if err != nil {
		t.Fatalf("unexpected fail reading token file: %v", err)
	}
	if readAppToken.AccessToken != token.AccessToken {
		t.Fatalf("expected: %v, got: %v", token.AccessToken, readAppToken.AccessToken)
	}

	// lets update the first app token
	claims = auth.CustomClaims{
		Scope:       "scope3 scope4",
		AccountName: "test",
	}
	jsonClaims, err = json.Marshal(claims)
	if err != nil {
		t.Fatalf("unexpected fail marshalling claims: %v", err)
	}
	claimsSection = base64.RawURLEncoding.EncodeToString([]byte(jsonClaims))
	accessToken = fmt.Sprintf("%s.%s.%s", "header", claimsSection, "signature")
	newToken := &oauth2.Token{
		AccessToken: accessToken,
		Expiry:      time.Now().Add(1 * time.Hour),
	}
	err = saveLocalTokenFile(tempDir, app, newToken)
	if err != nil {
		t.Fatalf("unexpected fail saving token file: %v", err)
	}
	_, lastClaims, err := readLocalTokenFile(tempDir, app, nil)
	if err != nil {
		t.Fatalf("unexpected fail reading token file: %v", err)
	}
	if lastClaims[0] != "scope3" {
		t.Fatalf("expected: %v, got: %v", "scope3", lastClaims[0])
	}
	if lastClaims[1] != "scope4" {
		t.Fatalf("expected: %v, got: %v", "scope4", lastClaims[1])
	}
}
