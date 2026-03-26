package cliauth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/overmindtech/cli/go/auth"
	"golang.org/x/oauth2"
)

type mockLogger struct {
	infoMsgs  []string
	errorMsgs []string
}

func (m *mockLogger) Info(msg string, keysAndValues ...any) {
	m.infoMsgs = append(m.infoMsgs, msg)
}

func (m *mockLogger) Error(msg string, keysAndValues ...any) {
	m.errorMsgs = append(m.errorMsgs, msg)
}

func TestExtractClaims(t *testing.T) {
	testToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzY29wZSI6ImFkbWluOnJlYWQgYWRtaW46d3JpdGUiLCJzdWIiOiJ0ZXN0LXVzZXIiLCJpYXQiOjEyMzQ1Njc4OTAsImV4cCI6OTk5OTk5OTk5OX0.placeholder"

	claims, err := ExtractClaims(testToken)
	if err != nil {
		t.Fatalf("ExtractClaims failed: %v", err)
	}

	if claims.Scope != "admin:read admin:write" {
		t.Errorf("Expected scope 'admin:read admin:write', got '%s'", claims.Scope)
	}
}

func TestExtractClaimsInvalidJWT(t *testing.T) {
	_, err := ExtractClaims("not-a-jwt")
	if err == nil {
		t.Fatal("Expected error for non-JWT token, got nil")
	}
}

func TestExtractClaimsInvalidBase64(t *testing.T) {
	_, err := ExtractClaims("header.!!!invalid-base64!!!.sig")
	if err == nil {
		t.Fatal("Expected error for invalid base64, got nil")
	}
}

func TestHasScopesFlexible(t *testing.T) {
	tests := []struct {
		name           string
		tokenScopes    string
		requiredScopes []string
		expectOK       bool
		expectMissing  string
	}{
		{
			name:           "exact match",
			tokenScopes:    "admin:read",
			requiredScopes: []string{"admin:read"},
			expectOK:       true,
		},
		{
			name:           "write satisfies read",
			tokenScopes:    "admin:write",
			requiredScopes: []string{"admin:read"},
			expectOK:       true,
		},
		{
			name:           "missing scope",
			tokenScopes:    "changes:read",
			requiredScopes: []string{"admin:read"},
			expectOK:       false,
			expectMissing:  "admin:read",
		},
		{
			name:           "multiple scopes all present",
			tokenScopes:    "admin:read changes:write",
			requiredScopes: []string{"admin:read", "changes:read"},
			expectOK:       true,
		},
		{
			name:           "read does not satisfy write",
			tokenScopes:    "admin:read",
			requiredScopes: []string{"admin:write"},
			expectOK:       false,
			expectMissing:  "admin:write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testToken := &oauth2.Token{
				AccessToken: createTestJWT(tt.tokenScopes),
				TokenType:   "Bearer",
			}

			ok, missing, err := HasScopesFlexible(testToken, tt.requiredScopes)
			if err != nil {
				t.Fatalf("HasScopesFlexible failed: %v", err)
			}

			if ok != tt.expectOK {
				t.Errorf("Expected ok=%v, got %v", tt.expectOK, ok)
			}

			if !tt.expectOK && missing != tt.expectMissing {
				t.Errorf("Expected missing='%s', got '%s'", tt.expectMissing, missing)
			}
		})
	}
}

func TestHasScopesFlexibleNilToken(t *testing.T) {
	_, _, err := HasScopesFlexible(nil, []string{"admin:read"})
	if err == nil {
		t.Fatal("Expected error for nil token, got nil")
	}
}

func TestReadWriteLocalToken(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cliauth-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log := &mockLogger{}
	app := "https://test.overmind.tech"
	token := &oauth2.Token{
		AccessToken: createTestJWT("admin:read admin:write"),
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	err = SaveLocalToken(tmpDir, app, token, log)
	if err != nil {
		t.Fatalf("SaveLocalToken failed: %v", err)
	}

	tokenPath := filepath.Join(tmpDir, ".overmind", "token.json")
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		t.Fatalf("Token file was not created")
	}

	readToken, scopes, err := ReadLocalToken(tmpDir, app, []string{"admin:read"}, log)
	if err != nil {
		t.Fatalf("ReadLocalToken failed: %v", err)
	}

	if readToken.AccessToken != token.AccessToken {
		t.Errorf("Token mismatch")
	}

	if len(scopes) != 2 {
		t.Errorf("Expected 2 scopes, got %d", len(scopes))
	}
}

func TestReadLocalTokenWrongApp(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cliauth-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log := &mockLogger{}
	app := "https://test.overmind.tech"
	token := &oauth2.Token{
		AccessToken: createTestJWT("admin:read"),
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	if err := SaveLocalToken(tmpDir, app, token, log); err != nil {
		t.Fatalf("SaveLocalToken failed: %v", err)
	}

	_, _, err = ReadLocalToken(tmpDir, "https://wrong.overmind.tech", []string{"admin:read"}, log)
	if err == nil {
		t.Errorf("Expected error for wrong app, got nil")
	}
}

func TestReadLocalTokenInsufficientScopes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cliauth-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log := &mockLogger{}
	app := "https://test.overmind.tech"
	token := &oauth2.Token{
		AccessToken: createTestJWT("changes:read"),
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	if err := SaveLocalToken(tmpDir, app, token, log); err != nil {
		t.Fatalf("SaveLocalToken failed: %v", err)
	}

	_, _, err = ReadLocalToken(tmpDir, app, []string{"admin:read"}, log)
	if err == nil {
		t.Errorf("Expected error for insufficient scopes, got nil")
	}
}

func TestReadLocalTokenFileNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cliauth-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log := &mockLogger{}
	_, _, err = ReadLocalToken(tmpDir, "https://test.overmind.tech", []string{"admin:read"}, log)
	if err == nil {
		t.Fatal("Expected error for missing file, got nil")
	}
}

func TestSaveLocalTokenSecurePermissions(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cliauth-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log := &mockLogger{}
	token := &oauth2.Token{
		AccessToken: createTestJWT("admin:read"),
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	if err := SaveLocalToken(tmpDir, "https://test.overmind.tech", token, log); err != nil {
		t.Fatalf("SaveLocalToken failed: %v", err)
	}

	dirInfo, err := os.Stat(filepath.Join(tmpDir, ".overmind"))
	if err != nil {
		t.Fatalf("Failed to stat directory: %v", err)
	}
	if dirInfo.Mode().Perm() != 0700 {
		t.Errorf("Expected directory permissions 0700, got %o", dirInfo.Mode().Perm())
	}

	fileInfo, err := os.Stat(filepath.Join(tmpDir, ".overmind", "token.json"))
	if err != nil {
		t.Fatalf("Failed to stat token file: %v", err)
	}
	if fileInfo.Mode().Perm() != 0600 {
		t.Errorf("Expected file permissions 0600, got %o", fileInfo.Mode().Perm())
	}
}

func TestSaveLocalTokenNilMap(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cliauth-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tokenPath := filepath.Join(tmpDir, ".overmind", "token.json")
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0700); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// Simulate a corrupt token file with null auth_entries
	if err := os.WriteFile(tokenPath, []byte(`{"auth_entries": null}`), 0600); err != nil {
		t.Fatalf("Failed to write token file: %v", err)
	}

	log := &mockLogger{}
	token := &oauth2.Token{
		AccessToken: createTestJWT("admin:read"),
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	err = SaveLocalToken(tmpDir, "https://test.overmind.tech", token, log)
	if err != nil {
		t.Fatalf("SaveLocalToken failed with nil map: %v", err)
	}

	readToken, _, err := ReadLocalToken(tmpDir, "https://test.overmind.tech", []string{"admin:read"}, log)
	if err != nil {
		t.Fatalf("ReadLocalToken failed: %v", err)
	}
	if readToken.AccessToken != token.AccessToken {
		t.Errorf("Token mismatch after nil map save")
	}
}

func TestReadLocalTokenNilEntry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cliauth-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tokenPath := filepath.Join(tmpDir, ".overmind", "token.json")
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0700); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := os.WriteFile(tokenPath, []byte(`{"auth_entries": {"https://test.overmind.tech": null}}`), 0600); err != nil {
		t.Fatalf("Failed to write token file: %v", err)
	}

	log := &mockLogger{}
	_, _, err = ReadLocalToken(tmpDir, "https://test.overmind.tech", []string{"admin:read"}, log)
	if err == nil {
		t.Fatal("Expected error for null token entry, got nil")
	}
	if !strings.Contains(err.Error(), "null") {
		t.Errorf("Expected error to mention 'null', got: %v", err)
	}
}

func TestReadLocalTokenNilToken(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cliauth-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tokenPath := filepath.Join(tmpDir, ".overmind", "token.json")
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0700); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	if err := os.WriteFile(tokenPath, []byte(`{"auth_entries": {"https://test.overmind.tech": {"token": null, "added_date": "2024-01-01T00:00:00Z"}}}`), 0600); err != nil {
		t.Fatalf("Failed to write token file: %v", err)
	}

	log := &mockLogger{}
	_, _, err = ReadLocalToken(tmpDir, "https://test.overmind.tech", []string{"admin:read"}, log)
	if err == nil {
		t.Fatal("Expected error for null token, got nil")
	}
	if !strings.Contains(err.Error(), "null") {
		t.Errorf("Expected error to mention 'null', got: %v", err)
	}
}

func TestSaveLocalTokenOverwriteExisting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cliauth-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log := &mockLogger{}
	app := "https://test.overmind.tech"

	token1 := &oauth2.Token{
		AccessToken: createTestJWT("admin:read"),
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(1 * time.Hour),
	}
	token2 := &oauth2.Token{
		AccessToken: createTestJWT("admin:write"),
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	if err := SaveLocalToken(tmpDir, app, token1, log); err != nil {
		t.Fatalf("SaveLocalToken (first) failed: %v", err)
	}
	if err := SaveLocalToken(tmpDir, app, token2, log); err != nil {
		t.Fatalf("SaveLocalToken (second) failed: %v", err)
	}

	readToken, _, err := ReadLocalToken(tmpDir, app, []string{"admin:write"}, log)
	if err != nil {
		t.Fatalf("ReadLocalToken failed: %v", err)
	}
	if readToken.AccessToken != token2.AccessToken {
		t.Errorf("Expected second token, got first")
	}
}

func TestSaveLocalTokenMultipleApps(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cliauth-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	log := &mockLogger{}
	app1 := "https://app.overmind.tech"
	app2 := "https://app.staging.overmind.tech"

	token1 := &oauth2.Token{
		AccessToken: createTestJWT("admin:read"),
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(1 * time.Hour),
	}
	token2 := &oauth2.Token{
		AccessToken: createTestJWT("admin:write"),
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	if err := SaveLocalToken(tmpDir, app1, token1, log); err != nil {
		t.Fatalf("SaveLocalToken (app1) failed: %v", err)
	}
	if err := SaveLocalToken(tmpDir, app2, token2, log); err != nil {
		t.Fatalf("SaveLocalToken (app2) failed: %v", err)
	}

	read1, _, err := ReadLocalToken(tmpDir, app1, []string{"admin:read"}, log)
	if err != nil {
		t.Fatalf("ReadLocalToken (app1) failed: %v", err)
	}
	if read1.AccessToken != token1.AccessToken {
		t.Errorf("App1 token mismatch")
	}

	read2, _, err := ReadLocalToken(tmpDir, app2, []string{"admin:write"}, log)
	if err != nil {
		t.Fatalf("ReadLocalToken (app2) failed: %v", err)
	}
	if read2.AccessToken != token2.AccessToken {
		t.Errorf("App2 token mismatch")
	}
}

func TestNoSliceMutationInScopeMerge(t *testing.T) {
	// Verify the pattern used in GetOauthToken doesn't mutate caller slices
	requiredScopes := make([]string, 1, 10) // extra capacity — the mutation scenario
	requiredScopes[0] = "admin:read"

	originalLen := len(requiredScopes)
	localScopes := []string{"changes:read", "config:read"}

	// This is the safe pattern used in GetOauthToken
	requestScopes := make([]string, 0, len(requiredScopes)+len(localScopes))
	requestScopes = append(requestScopes, requiredScopes...)
	requestScopes = append(requestScopes, localScopes...)

	if len(requiredScopes) != originalLen {
		t.Errorf("Original slice length changed from %d to %d", originalLen, len(requiredScopes))
	}
	if len(requestScopes) != 3 {
		t.Errorf("Expected 3 scopes in combined slice, got %d", len(requestScopes))
	}
}

// createTestJWT creates a minimal JWT token for testing (no signature verification)
func createTestJWT(scopes string) string {
	header := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"

	payload := auth.CustomClaims{
		Scope: scopes,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal test payload: %v", err))
	}

	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	return header + "." + payloadB64 + ".test-signature"
}
