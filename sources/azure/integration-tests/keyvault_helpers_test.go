package integrationtests

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// ensureKeyVaultKey ensures a Key Vault key exists and returns its (versioned) key URL.
//
// Note: this uses the Azure CLI for data-plane operations to keep integration test setup simple.
func ensureKeyVaultKey(ctx context.Context, vaultName, keyName string) (string, error) {
	// If the key already exists, return its current (versioned) URL.
	showCmd := exec.CommandContext(ctx, "az", "keyvault", "key", "show",
		"--vault-name", vaultName,
		"--name", keyName,
		"--query", "key.kid",
		"-o", "tsv",
	)
	if out, err := showCmd.CombinedOutput(); err == nil {
		keyURL := strings.TrimSpace(string(out))
		if keyURL != "" {
			log.Printf("Key Vault key %s already exists in vault %s", keyName, vaultName)
			return keyURL, nil
		}
	}

	createCmd := exec.CommandContext(ctx, "az", "keyvault", "key", "create",
		"--vault-name", vaultName,
		"--name", keyName,
		"--kty", "RSA",
		"--size", "2048",
		"--query", "key.kid",
		"-o", "tsv",
	)
	out, err := createCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create key vault key: %w, output: %s", err, string(out))
	}
	keyURL := strings.TrimSpace(string(out))
	if keyURL == "" {
		return "", fmt.Errorf("created key but key URL was empty")
	}
	log.Printf("Key Vault key %s created in vault %s", keyName, vaultName)
	return keyURL, nil
}

// grantKeyVaultCryptoAccess grants an identity access to Key Vault key crypto operations.
//
// Different vaults may use access policies or RBAC for authorization, so we attempt both.
func grantKeyVaultCryptoAccess(ctx context.Context, vaultName, vaultResourceID, principalID string) error {
	// Try access-policy based authorization.
	// This is idempotent: if policy exists, it updates.
	setPolicyCmd := exec.CommandContext(ctx, "az", "keyvault", "set-policy",
		"--name", vaultName,
		"--object-id", principalID,
		"--key-permissions", "get", "wrapKey", "unwrapKey",
	)
	if out, err := setPolicyCmd.CombinedOutput(); err != nil {
		log.Printf("Key Vault set-policy failed (may be RBAC-enabled vault): %v, output: %s", err, string(out))
	}

	// Try RBAC based authorization.
	// This can fail if the vault isn't RBAC-enabled for data-plane, but it won't hurt to try.
	roleCmd := exec.CommandContext(ctx, "az", "role", "assignment", "create",
		"--assignee-object-id", principalID,
		"--assignee-principal-type", "ServicePrincipal",
		"--role", "Key Vault Crypto Service Encryption User",
		"--scope", vaultResourceID,
	)
	if out, err := roleCmd.CombinedOutput(); err != nil {
		// If the assignment already exists, treat it as success.
		if strings.Contains(string(out), "RoleAssignmentExists") || strings.Contains(string(out), "already exists") {
			return nil
		}
		log.Printf("Key Vault role assignment failed (may be access-policy vault): %v, output: %s", err, string(out))
	}

	return nil
}
