package integrationtests

// GCP Cloud KMS Limitations
//
// This test compares the Cloud KMS direct API with the Cloud Asset Inventory API.
// Understanding the following GCP limitations is essential for working with KMS resources:
//
// 1. CryptoKey Deletion:
//    - CryptoKeys CANNOT be immediately deleted from GCP
//    - Must destroy all CryptoKeyVersions first (schedules for deletion after 24h by default)
//    - Even after version destruction, the CryptoKey resource remains (in DESTROYED state)
//    - The key name cannot be reused after destruction
//    - See: https://cloud.google.com/kms/docs/destroy-restore
//
// 2. KeyRing Deletion:
//    - KeyRings CANNOT be deleted at all in GCP
//    - Once created, they persist forever in the project
//    - This is by design for audit/compliance purposes
//    - See: https://cloud.google.com/kms/docs/resource-hierarchy
//
// 3. Resource Naming:
//    - KeyRing and CryptoKey names must be unique within their parent
//    - Names cannot be reused even after destruction
//    - This test uses a shared KeyRing to avoid proliferation
//
// 4. Asset Inventory Indexing:
//    - Cloud Asset Inventory indexes resources asynchronously
//    - New resources may take 1-5 minutes to appear in queries
//    - The test includes retry logic to handle this delay
//
// API Rate Limits (for reference):
//
// Cloud KMS API:
//   - Read requests: 300 queries per minute (QPM)
//   - Enforced per-second (QPS), not per-minute
//   - Exceeding limit returns RESOURCE_EXHAUSTED error
//   - See: https://cloud.google.com/kms/quotas
//
// Cloud Asset Inventory API:
//   - ListAssets: 100 QPM per project, 800 QPM per organization
//   - SearchAllResources: 400 QPM per project
//   - See: https://cloud.google.com/asset-inventory/docs/quota

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	"cloud.google.com/go/kms/apiv1/kmspb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

const (
	// Shared KeyRing name - reused across test runs since KeyRings cannot be deleted
	testKeyRingName = "integration-test-keyring"
	// Location for KMS resources
	testKMSLocation = "global"
	// CryptoKey name prefix - timestamp will be appended for uniqueness
	testCryptoKeyPrefix = "api-comparison-test-key"
)

// TestKMSvsAssetInventoryComparison compares the Cloud KMS direct API with the
// Cloud Asset Inventory API for retrieving CryptoKey information.
//
// This test demonstrates the differences in:
// - Calling conventions (URL structure, query parameters)
// - Response structure (direct resource vs wrapped asset)
// - Available metadata (ancestors, update times, etc.)
// - Rate limits and quotas
func TestKMSvsAssetInventoryComparison(t *testing.T) {
	projectID := os.Getenv("GCP_PROJECT_ID")
	if projectID == "" {
		t.Skip("GCP_PROJECT_ID environment variable not set")
	}

	ctx := context.Background()

	// Create KMS client for resource management
	kmsClient, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create KMS client: %v", err)
	}
	defer kmsClient.Close()

	// Create HTTP client for direct API calls
	httpClient, err := gcpshared.GCPHTTPClientWithOtel(ctx, "")
	if err != nil {
		t.Fatalf("Failed to create HTTP client: %v", err)
	}

	// Generate unique CryptoKey name for this test run
	cryptoKeyName := fmt.Sprintf("%s-%d", testCryptoKeyPrefix, time.Now().Unix())

	// Full resource names
	keyRingParent := fmt.Sprintf("projects/%s/locations/%s", projectID, testKMSLocation)
	keyRingFullName := fmt.Sprintf("%s/keyRings/%s", keyRingParent, testKeyRingName)
	cryptoKeyFullName := fmt.Sprintf("%s/cryptoKeys/%s", keyRingFullName, cryptoKeyName)

	t.Run("Setup", func(t *testing.T) {
		// Create KeyRing (idempotent - will succeed if already exists)
		err := createKeyRing(ctx, kmsClient, keyRingParent, testKeyRingName)
		if err != nil {
			t.Fatalf("Failed to create KeyRing: %v", err)
		}
		log.Printf("KeyRing ready: %s", keyRingFullName)

		// Create CryptoKey for this test
		err = createCryptoKey(ctx, kmsClient, keyRingFullName, cryptoKeyName)
		if err != nil {
			t.Fatalf("Failed to create CryptoKey: %v", err)
		}
		log.Printf("CryptoKey created: %s", cryptoKeyFullName)
	})

	t.Run("CompareAPIs", func(t *testing.T) {
		t.Log("=== GCP API Comparison: Cloud KMS vs Cloud Asset Inventory ===")
		t.Log("")

		// --- Cloud KMS Direct API ---
		t.Log("--- Cloud KMS Direct API ---")
		kmsURL := fmt.Sprintf("https://cloudkms.googleapis.com/v1/%s", cryptoKeyFullName)
		t.Logf("URL: %s", kmsURL)
		t.Logf("Method: GET")
		t.Logf("Required Permission: cloudkms.cryptoKeys.get")
		t.Logf("Rate Limit: 300 QPM (enforced per-second)")
		t.Log("")

		kmsStart := time.Now()
		kmsResponse, err := callKMSDirectAPI(ctx, httpClient, cryptoKeyFullName)
		kmsLatency := time.Since(kmsStart)
		if err != nil {
			t.Fatalf("Failed to call KMS API: %v", err)
		}
		t.Logf("Latency: %v", kmsLatency)
		t.Log("")

		// Pretty print KMS response
		kmsJSON, _ := json.MarshalIndent(kmsResponse, "", "  ")
		t.Logf("Response Structure (Cloud KMS):\n%s", string(kmsJSON))
		t.Log("")

		// --- Cloud Asset Inventory API ---
		t.Log("--- Cloud Asset Inventory API ---")
		assetURL := fmt.Sprintf(
			"https://cloudasset.googleapis.com/v1/projects/%s/assets?assetTypes=cloudkms.googleapis.com/CryptoKey&contentType=RESOURCE",
			projectID,
		)
		t.Logf("URL: %s", assetURL)
		t.Logf("Method: GET")
		t.Logf("Required Permission: cloudasset.assets.listResource")
		t.Logf("Rate Limit: 100 QPM per project (ListAssets)")
		t.Log("")

		// Asset Inventory may have indexing delay - retry with backoff
		var assetResponse map[string]interface{}
		var assetLatency time.Duration
		var foundAsset bool

		t.Log("Note: Cloud Asset Inventory indexes resources asynchronously.")
		t.Log("Retrying with backoff if the newly created key is not yet indexed...")
		t.Log("")

		for attempt := 1; attempt <= 10; attempt++ {
			assetStart := time.Now()
			assetResponse, err = callAssetInventoryAPI(ctx, httpClient, projectID, cryptoKeyFullName)
			assetLatency = time.Since(assetStart)

			if err != nil {
				t.Logf("Attempt %d: Error calling Asset Inventory API: %v", attempt, err)
			} else if assetResponse != nil {
				foundAsset = true
				t.Logf("Attempt %d: Found asset after %v", attempt, assetLatency)
				break
			} else {
				t.Logf("Attempt %d: Asset not yet indexed, waiting...", attempt)
			}

			// Exponential backoff: 5s, 10s, 20s, 40s... up to 60s
			waitTime := time.Duration(5*(1<<(attempt-1))) * time.Second
			if waitTime > 60*time.Second {
				waitTime = 60 * time.Second
			}
			time.Sleep(waitTime)
		}

		if !foundAsset {
			t.Log("WARNING: Asset not found in Cloud Asset Inventory after retries.")
			t.Log("This may indicate the indexing delay exceeds our retry window.")
			t.Log("The test will continue with partial comparison.")
		} else {
			// Pretty print Asset Inventory response
			assetJSON, _ := json.MarshalIndent(assetResponse, "", "  ")
			t.Logf("Response Structure (Cloud Asset Inventory):\n%s", string(assetJSON))
		}
		t.Log("")

		// --- Comparison Summary ---
		t.Log("=== Comparison Summary ===")
		t.Log("")
		t.Log("| Aspect                  | Cloud KMS API              | Cloud Asset Inventory API       |")
		t.Log("|-------------------------|----------------------------|---------------------------------|")
		t.Log("| Endpoint                | cloudkms.googleapis.com    | cloudasset.googleapis.com       |")
		t.Log("| Response Type           | Direct resource            | Wrapped in Asset object         |")
		t.Log("| Resource Data Location  | Root of response           | resource.data field             |")
		t.Log("| Rate Limit              | 300 QPM                    | 100 QPM (ListAssets)            |")
		t.Log("| Ancestry Info           | Not included               | Included (ancestors field)      |")
		t.Log("| IAM Policy              | Separate API call          | Optional (contentType param)    |")
		t.Log("| Update Timestamp        | createTime only            | updateTime + createTime         |")
		t.Logf("| Observed Latency        | %v                      | %v                           |", kmsLatency.Round(time.Millisecond), assetLatency.Round(time.Millisecond))
		t.Log("")

		t.Log("Key Differences:")
		t.Log("1. Cloud KMS returns the CryptoKey resource directly")
		t.Log("2. Cloud Asset Inventory wraps the resource with metadata (ancestors, assetType, updateTime)")
		t.Log("3. Asset Inventory can batch multiple asset types in a single request")
		t.Log("4. Asset Inventory provides resource hierarchy information (ancestors)")
		t.Log("5. Cloud KMS API has higher rate limits for targeted resource access")
		t.Log("6. Asset Inventory has indexing delay (resources not immediately available)")
	})

	t.Run("Teardown", func(t *testing.T) {
		// Note: We cannot delete CryptoKeys or KeyRings in GCP.
		// The best we can do is destroy the CryptoKeyVersion to make the key unusable.
		//
		// From GCP documentation:
		// "You cannot delete a CryptoKey or KeyRing resource. These resources are retained
		// indefinitely for audit and compliance purposes."
		//
		// To minimize resource accumulation, we:
		// 1. Destroy the primary CryptoKeyVersion (schedules it for deletion after 24h)
		// 2. Leave the CryptoKey in DESTROYED state
		// 3. Reuse the same KeyRing for all test runs

		err := destroyCryptoKeyVersion(ctx, kmsClient, cryptoKeyFullName)
		if err != nil {
			// Log but don't fail - the key will remain but be unusable
			log.Printf("Warning: Failed to destroy CryptoKeyVersion: %v", err)
			log.Printf("The CryptoKey %s will remain active but can be manually destroyed later", cryptoKeyFullName)
		} else {
			log.Printf("CryptoKeyVersion scheduled for destruction: %s", cryptoKeyFullName)
			log.Printf("Note: The CryptoKey resource itself cannot be deleted (GCP limitation)")
		}
	})
}

// createKeyRing creates a KeyRing if it doesn't already exist.
// KeyRings cannot be deleted, so this is idempotent.
func createKeyRing(ctx context.Context, client *kms.KeyManagementClient, parent, keyRingID string) error {
	req := &kmspb.CreateKeyRingRequest{
		Parent:    parent,
		KeyRingId: keyRingID,
		KeyRing:   &kmspb.KeyRing{},
	}

	_, err := client.CreateKeyRing(ctx, req)
	if err != nil {
		// Check for gRPC AlreadyExists error - KeyRing already exists is fine
		if st, ok := status.FromError(err); ok && st.Code() == codes.AlreadyExists {
			log.Printf("KeyRing already exists (expected): %s/%s", parent, keyRingID)
			return nil
		}
		return fmt.Errorf("failed to create KeyRing: %w", err)
	}

	return nil
}

// createCryptoKey creates a new CryptoKey for encryption/decryption.
func createCryptoKey(ctx context.Context, client *kms.KeyManagementClient, keyRingName, cryptoKeyID string) error {
	req := &kmspb.CreateCryptoKeyRequest{
		Parent:      keyRingName,
		CryptoKeyId: cryptoKeyID,
		CryptoKey: &kmspb.CryptoKey{
			Purpose: kmspb.CryptoKey_ENCRYPT_DECRYPT,
			Labels: map[string]string{
				"test":    "integration",
				"purpose": "api-comparison",
			},
		},
	}

	_, err := client.CreateCryptoKey(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create CryptoKey: %w", err)
	}

	return nil
}

// destroyCryptoKeyVersion destroys the primary version of a CryptoKey.
// This is the closest we can get to "deleting" a key in GCP.
// The version is scheduled for destruction after 24 hours by default.
func destroyCryptoKeyVersion(ctx context.Context, client *kms.KeyManagementClient, cryptoKeyName string) error {
	// First, get the CryptoKey to find its primary version
	getReq := &kmspb.GetCryptoKeyRequest{
		Name: cryptoKeyName,
	}

	cryptoKey, err := client.GetCryptoKey(ctx, getReq)
	if err != nil {
		return fmt.Errorf("failed to get CryptoKey: %w", err)
	}

	if cryptoKey.GetPrimary() == nil {
		log.Printf("CryptoKey has no primary version (may already be destroyed)")
		return nil
	}

	// Destroy the primary version
	destroyReq := &kmspb.DestroyCryptoKeyVersionRequest{
		Name: cryptoKey.GetPrimary().GetName(),
	}

	_, err = client.DestroyCryptoKeyVersion(ctx, destroyReq)
	if err != nil {
		return fmt.Errorf("failed to destroy CryptoKeyVersion: %w", err)
	}

	return nil
}

// callKMSDirectAPI calls the Cloud KMS REST API directly to get a CryptoKey.
func callKMSDirectAPI(ctx context.Context, httpClient *http.Client, cryptoKeyName string) (map[string]interface{}, error) {
	apiURL := fmt.Sprintf("https://cloudkms.googleapis.com/v1/%s", cryptoKeyName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KMS API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// callAssetInventoryAPI calls the Cloud Asset Inventory API to find a specific CryptoKey.
// Returns the asset if found, nil if not found (may indicate indexing delay).
func callAssetInventoryAPI(ctx context.Context, httpClient *http.Client, projectID, cryptoKeyName string) (map[string]interface{}, error) {
	// Build the Asset Inventory ListAssets URL
	baseURL := fmt.Sprintf("https://cloudasset.googleapis.com/v1/projects/%s/assets", projectID)

	params := url.Values{}
	params.Set("assetTypes", "cloudkms.googleapis.com/CryptoKey")
	params.Set("contentType", "RESOURCE")

	apiURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Cloud Asset Inventory API requires a quota project header when using user credentials
	// This tells GCP which project to bill for the API usage
	req.Header.Set("X-Goog-User-Project", projectID)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Asset Inventory API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Find the specific CryptoKey in the assets list
	assets, ok := result["assets"].([]interface{})
	if !ok || len(assets) == 0 {
		return nil, nil // No assets found - may indicate indexing delay
	}

	// The Asset Inventory uses full resource names with // prefix
	// e.g., //cloudkms.googleapis.com/projects/PROJECT/locations/global/keyRings/RING/cryptoKeys/KEY
	expectedAssetName := fmt.Sprintf("//cloudkms.googleapis.com/%s", cryptoKeyName)

	for _, asset := range assets {
		assetMap, ok := asset.(map[string]interface{})
		if !ok {
			continue
		}

		name, ok := assetMap["name"].(string)
		if !ok {
			continue
		}

		if strings.HasSuffix(name, cryptoKeyName) || name == expectedAssetName {
			return assetMap, nil
		}
	}

	return nil, nil // Specific key not found in results
}
