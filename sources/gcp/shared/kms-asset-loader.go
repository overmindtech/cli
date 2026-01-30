package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"cloud.google.com/go/kms/apiv1/kmspb"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/singleflight"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources/shared"
)

// CloudKMSAssetLoader handles bulk loading of KMS resources via Cloud Asset API.
// It fetches all KMS resources (KeyRings, CryptoKeys, CryptoKeyVersions) in a single
// API call and stores them in sdpcache for efficient retrieval by adapters.
type CloudKMSAssetLoader struct {
	httpClient *http.Client
	projectID  string
	cache      sdpcache.Cache
	sourceName string

	// TTL-aware reloading
	mu           sync.Mutex
	lastLoadTime time.Time
	group        singleflight.Group
}

// NewCloudKMSAssetLoader creates a new CloudKMSAssetLoader.
func NewCloudKMSAssetLoader(
	httpClient *http.Client,
	projectID string,
	cache sdpcache.Cache,
	sourceName string,
	locations []LocationInfo,
) *CloudKMSAssetLoader {
	return &CloudKMSAssetLoader{
		httpClient: httpClient,
		projectID:  projectID,
		cache:      cache,
		sourceName: sourceName,
	}
}

// EnsureLoaded triggers bulk load if cache TTL has expired.
// Called by adapters on cache miss.
func (l *CloudKMSAssetLoader) EnsureLoaded(ctx context.Context) error {
	l.mu.Lock()
	timeSinceLastLoad := time.Since(l.lastLoadTime)
	l.mu.Unlock()

	// If data was loaded recently, skip reload
	if timeSinceLastLoad < shared.DefaultCacheDuration {
		return nil
	}

	// Use singleflight to ensure only one load runs at a time
	// Concurrent callers wait for the same result
	_, err, _ := l.group.Do("load", func() (interface{}, error) {
		// Double-check TTL after acquiring the flight
		l.mu.Lock()
		if time.Since(l.lastLoadTime) < shared.DefaultCacheDuration {
			l.mu.Unlock()
			return nil, nil
		}
		l.mu.Unlock()

		// Perform the bulk load
		if err := l.loadAll(ctx); err != nil {
			return nil, err
		}

		// Update last load time on success
		l.mu.Lock()
		l.lastLoadTime = time.Now()
		l.mu.Unlock()

		return nil, nil
	})
	return err
}

// cloudAssetResponse represents the response from Cloud Asset API
type cloudAssetResponse struct {
	Assets        []cloudAsset `json:"assets"`
	NextPageToken string       `json:"nextPageToken"`
}

// cloudAsset represents a single asset from Cloud Asset API
type cloudAsset struct {
	Name       string        `json:"name"`
	AssetType  string        `json:"assetType"`
	Resource   cloudResource `json:"resource"`
	Ancestors  []string      `json:"ancestors"`
	UpdateTime string        `json:"updateTime"`
}

// cloudResource contains the actual resource data
type cloudResource struct {
	Version              string          `json:"version"`
	DiscoveryDocumentURI string          `json:"discoveryDocumentUri"`
	DiscoveryName        string          `json:"discoveryName"`
	Parent               string          `json:"parent"`
	Data                 json.RawMessage `json:"data"`
}

// loadAll fetches all KMS resources from Cloud Asset API and stores in sdpcache
func (l *CloudKMSAssetLoader) loadAll(ctx context.Context) error {
	// Fetch all KMS assets
	assets, err := l.fetchAllAssets(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch KMS assets: %w", err)
	}

	// Track which resource types had items
	hasKeyRings := false
	hasCryptoKeys := false
	hasKeyVersions := false

	// Process and cache each asset
	for _, asset := range assets {
		switch asset.AssetType {
		case "cloudkms.googleapis.com/KeyRing":
			hasKeyRings = true
			if err := l.cacheKeyRing(ctx, asset); err != nil {
				// Log error but continue processing other assets
				log.WithContext(ctx).WithError(err).WithFields(log.Fields{
					"ovm.kms.assetType": asset.AssetType,
					"ovm.kms.assetName": asset.Name,
				}).Warn("failed to cache KMS KeyRing")
				continue
			}
		case "cloudkms.googleapis.com/CryptoKey":
			hasCryptoKeys = true
			if err := l.cacheCryptoKey(ctx, asset); err != nil {
				log.WithContext(ctx).WithError(err).WithFields(log.Fields{
					"ovm.kms.assetType": asset.AssetType,
					"ovm.kms.assetName": asset.Name,
				}).Warn("failed to cache KMS CryptoKey")
				continue
			}
		case "cloudkms.googleapis.com/CryptoKeyVersion":
			hasKeyVersions = true
			if err := l.cacheCryptoKeyVersion(ctx, asset); err != nil {
				log.WithContext(ctx).WithError(err).WithFields(log.Fields{
					"ovm.kms.assetType": asset.AssetType,
					"ovm.kms.assetName": asset.Name,
				}).Warn("failed to cache KMS CryptoKeyVersion")
				continue
			}
		}
	}

	// For types with no items, store NOTFOUND error so cache.Lookup() returns cacheHit=true
	notFoundErr := &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOTFOUND,
		ErrorString: "No resources found in Cloud Asset API",
	}

	scope := l.projectID

	if !hasKeyRings {
		listCacheKey := sdpcache.CacheKeyFromParts(l.sourceName, sdp.QueryMethod_LIST, scope, CloudKMSKeyRing.String(), "")
		l.cache.StoreError(ctx, notFoundErr, shared.DefaultCacheDuration, listCacheKey)
	}
	if !hasCryptoKeys {
		listCacheKey := sdpcache.CacheKeyFromParts(l.sourceName, sdp.QueryMethod_LIST, scope, CloudKMSCryptoKey.String(), "")
		l.cache.StoreError(ctx, notFoundErr, shared.DefaultCacheDuration, listCacheKey)
	}
	if !hasKeyVersions {
		listCacheKey := sdpcache.CacheKeyFromParts(l.sourceName, sdp.QueryMethod_LIST, scope, CloudKMSCryptoKeyVersion.String(), "")
		l.cache.StoreError(ctx, notFoundErr, shared.DefaultCacheDuration, listCacheKey)
	}

	return nil
}

// fetchAllAssets fetches all KMS assets from Cloud Asset API with pagination
func (l *CloudKMSAssetLoader) fetchAllAssets(ctx context.Context) ([]cloudAsset, error) {
	var allAssets []cloudAsset
	pageToken := ""

	for {
		assets, nextToken, err := l.fetchAssetsPage(ctx, pageToken)
		if err != nil {
			return nil, err
		}

		allAssets = append(allAssets, assets...)

		if nextToken == "" {
			break
		}
		pageToken = nextToken
	}

	return allAssets, nil
}

// fetchAssetsPage fetches a single page of KMS assets
func (l *CloudKMSAssetLoader) fetchAssetsPage(ctx context.Context, pageToken string) ([]cloudAsset, string, error) {
	// Build the Cloud Asset API URL
	baseURL := fmt.Sprintf("https://cloudasset.googleapis.com/v1/projects/%s/assets", l.projectID)

	params := url.Values{}
	params.Add("assetTypes", "cloudkms.googleapis.com/KeyRing")
	params.Add("assetTypes", "cloudkms.googleapis.com/CryptoKey")
	params.Add("assetTypes", "cloudkms.googleapis.com/CryptoKeyVersion")
	params.Set("contentType", "RESOURCE")
	if pageToken != "" {
		params.Set("pageToken", pageToken)
	}

	apiURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Cloud Asset API requires quota project header
	req.Header.Set("X-Goog-User-Project", l.projectID)

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("Cloud Asset API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	var response cloudAssetResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response.Assets, response.NextPageToken, nil
}

// cacheKeyRing converts a Cloud Asset to SDP Item and stores in cache
func (l *CloudKMSAssetLoader) cacheKeyRing(ctx context.Context, asset cloudAsset) error {
	// Parse the resource data into KeyRing protobuf
	var keyRing kmspb.KeyRing
	if err := protojson.Unmarshal(asset.Resource.Data, &keyRing); err != nil {
		return fmt.Errorf("failed to unmarshal KeyRing: %w", err)
	}

	// Extract path parameters from the asset name
	// Format: //cloudkms.googleapis.com/projects/{project}/locations/{location}/keyRings/{keyRing}
	resourceName := extractResourceName(asset.Name)
	keyRingVals := ExtractPathParams(resourceName, "locations", "keyRings")
	if len(keyRingVals) != 2 || keyRingVals[0] == "" || keyRingVals[1] == "" {
		return fmt.Errorf("invalid KeyRing name: %s", asset.Name)
	}

	// Create unique attribute key (location|keyRingName)
	uniqueAttr := shared.CompositeLookupKey(keyRingVals...)

	// Convert to SDP Item
	attributes, err := shared.ToAttributesWithExclude(&keyRing)
	if err != nil {
		return fmt.Errorf("failed to convert KeyRing to attributes: %w", err)
	}

	if err := attributes.Set("uniqueAttr", uniqueAttr); err != nil {
		return fmt.Errorf("failed to set unique attribute: %w", err)
	}

	scope := l.projectID
	item := &sdp.Item{
		Type:            CloudKMSKeyRing.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Add linked item queries
	item.LinkedItemQueries = l.keyRingLinkedQueries(keyRingVals, scope)

	// Store in cache with GET cache key pattern (for individual lookups)
	getCacheKey := sdpcache.CacheKeyFromParts(l.sourceName, sdp.QueryMethod_GET, scope, CloudKMSKeyRing.String(), uniqueAttr)
	l.cache.StoreItem(ctx, item, shared.DefaultCacheDuration, getCacheKey)

	// Also store with LIST cache key (for listing all KeyRings)
	listCacheKey := sdpcache.CacheKeyFromParts(l.sourceName, sdp.QueryMethod_LIST, scope, CloudKMSKeyRing.String(), "")
	l.cache.StoreItem(ctx, item, shared.DefaultCacheDuration, listCacheKey)

	// Also store with SEARCH cache key (for searching by location)
	// KeyRing search is by location only
	location := keyRingVals[0]
	searchCacheKey := sdpcache.CacheKeyFromParts(l.sourceName, sdp.QueryMethod_SEARCH, scope, CloudKMSKeyRing.String(), location)
	l.cache.StoreItem(ctx, item, shared.DefaultCacheDuration, searchCacheKey)

	return nil
}

// cacheCryptoKey converts a Cloud Asset to SDP Item and stores in cache
func (l *CloudKMSAssetLoader) cacheCryptoKey(ctx context.Context, asset cloudAsset) error {
	// Parse the resource data into CryptoKey protobuf
	var cryptoKey kmspb.CryptoKey
	if err := protojson.Unmarshal(asset.Resource.Data, &cryptoKey); err != nil {
		return fmt.Errorf("failed to unmarshal CryptoKey: %w", err)
	}

	// Extract path parameters
	// Format: //cloudkms.googleapis.com/projects/{project}/locations/{location}/keyRings/{keyRing}/cryptoKeys/{cryptoKey}
	resourceName := extractResourceName(asset.Name)
	values := ExtractPathParams(resourceName, "locations", "keyRings", "cryptoKeys")
	if len(values) != 3 || values[0] == "" || values[1] == "" || values[2] == "" {
		return fmt.Errorf("invalid CryptoKey name: %s", asset.Name)
	}

	// Create unique attribute key (location|keyRing|cryptoKey)
	uniqueAttr := shared.CompositeLookupKey(values...)

	// Convert to SDP Item
	attributes, err := shared.ToAttributesWithExclude(&cryptoKey, "labels")
	if err != nil {
		return fmt.Errorf("failed to convert CryptoKey to attributes: %w", err)
	}

	if err := attributes.Set("uniqueAttr", uniqueAttr); err != nil {
		return fmt.Errorf("failed to set unique attribute: %w", err)
	}

	scope := l.projectID
	item := &sdp.Item{
		Type:            CloudKMSCryptoKey.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            cryptoKey.GetLabels(),
	}

	// Add linked item queries
	item.LinkedItemQueries = l.cryptoKeyLinkedQueries(values, &cryptoKey, scope)

	// Store in cache with GET cache key (for individual lookups)
	getCacheKey := sdpcache.CacheKeyFromParts(l.sourceName, sdp.QueryMethod_GET, scope, CloudKMSCryptoKey.String(), uniqueAttr)
	l.cache.StoreItem(ctx, item, shared.DefaultCacheDuration, getCacheKey)

	// Also store with LIST cache key (for listing all CryptoKeys)
	listCacheKey := sdpcache.CacheKeyFromParts(l.sourceName, sdp.QueryMethod_LIST, scope, CloudKMSCryptoKey.String(), "")
	l.cache.StoreItem(ctx, item, shared.DefaultCacheDuration, listCacheKey)

	// Also store with SEARCH cache key (for searching by keyRing)
	// CryptoKey search is by location|keyRing
	location := values[0]
	keyRing := values[1]
	searchQuery := shared.CompositeLookupKey(location, keyRing)
	searchCacheKey := sdpcache.CacheKeyFromParts(l.sourceName, sdp.QueryMethod_SEARCH, scope, CloudKMSCryptoKey.String(), searchQuery)
	l.cache.StoreItem(ctx, item, shared.DefaultCacheDuration, searchCacheKey)

	return nil
}

// cacheCryptoKeyVersion converts a Cloud Asset to SDP Item and stores in cache
func (l *CloudKMSAssetLoader) cacheCryptoKeyVersion(ctx context.Context, asset cloudAsset) error {
	// Parse the resource data into CryptoKeyVersion protobuf
	var keyVersion kmspb.CryptoKeyVersion
	if err := protojson.Unmarshal(asset.Resource.Data, &keyVersion); err != nil {
		return fmt.Errorf("failed to unmarshal CryptoKeyVersion: %w", err)
	}

	// Extract path parameters
	// Format: //cloudkms.googleapis.com/projects/{project}/locations/{location}/keyRings/{keyRing}/cryptoKeys/{cryptoKey}/cryptoKeyVersions/{version}
	resourceName := extractResourceName(asset.Name)
	values := ExtractPathParams(resourceName, "locations", "keyRings", "cryptoKeys", "cryptoKeyVersions")
	if len(values) != 4 || values[0] == "" || values[1] == "" || values[2] == "" || values[3] == "" {
		return fmt.Errorf("invalid CryptoKeyVersion name: %s", asset.Name)
	}

	// Create unique attribute key (location|keyRing|cryptoKey|version)
	uniqueAttr := shared.CompositeLookupKey(values...)

	// Convert to SDP Item
	attributes, err := shared.ToAttributesWithExclude(&keyVersion)
	if err != nil {
		return fmt.Errorf("failed to convert CryptoKeyVersion to attributes: %w", err)
	}

	if err := attributes.Set("uniqueAttr", uniqueAttr); err != nil {
		return fmt.Errorf("failed to set unique attribute: %w", err)
	}

	scope := l.projectID
	item := &sdp.Item{
		Type:            CloudKMSCryptoKeyVersion.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
	}

	// Add linked item queries
	item.LinkedItemQueries = l.cryptoKeyVersionLinkedQueries(values, &keyVersion, scope)

	// Set health based on state
	item.Health = l.cryptoKeyVersionHealth(&keyVersion)

	// Store in cache with GET cache key (for individual lookups)
	getCacheKey := sdpcache.CacheKeyFromParts(l.sourceName, sdp.QueryMethod_GET, scope, CloudKMSCryptoKeyVersion.String(), uniqueAttr)
	l.cache.StoreItem(ctx, item, shared.DefaultCacheDuration, getCacheKey)

	// Also store with LIST cache key (for listing all CryptoKeyVersions)
	listCacheKey := sdpcache.CacheKeyFromParts(l.sourceName, sdp.QueryMethod_LIST, scope, CloudKMSCryptoKeyVersion.String(), "")
	l.cache.StoreItem(ctx, item, shared.DefaultCacheDuration, listCacheKey)

	// Also store with SEARCH cache key (for searching by cryptoKey)
	// CryptoKeyVersion search is by location|keyRing|cryptoKey
	location := values[0]
	keyRing := values[1]
	cryptoKeyName := values[2]
	searchQuery := shared.CompositeLookupKey(location, keyRing, cryptoKeyName)
	searchCacheKey := sdpcache.CacheKeyFromParts(l.sourceName, sdp.QueryMethod_SEARCH, scope, CloudKMSCryptoKeyVersion.String(), searchQuery)
	l.cache.StoreItem(ctx, item, shared.DefaultCacheDuration, searchCacheKey)

	return nil
}

// extractResourceName extracts the resource name from Cloud Asset name
// Example: //cloudkms.googleapis.com/projects/my-project/locations/global/keyRings/my-keyring
// Returns: projects/my-project/locations/global/keyRings/my-keyring
func extractResourceName(assetName string) string {
	// Remove the //cloudkms.googleapis.com/ prefix
	if len(assetName) > 2 && assetName[:2] == "//" {
		// Find the first / after the domain
		for i := 2; i < len(assetName); i++ {
			if assetName[i] == '/' {
				return assetName[i+1:]
			}
		}
	}
	return assetName
}

// keyRingLinkedQueries returns linked item queries for a KeyRing
func (l *CloudKMSAssetLoader) keyRingLinkedQueries(keyRingVals []string, scope string) []*sdp.LinkedItemQuery {
	var queries []*sdp.LinkedItemQuery

	// Link to IAM Policy
	queries = append(queries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   IAMPolicy.String(),
			Method: sdp.QueryMethod_GET,
			Query:  shared.CompositeLookupKey(keyRingVals...),
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,
			Out: true,
		},
	})

	// Link to CryptoKeys in this KeyRing
	queries = append(queries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   CloudKMSCryptoKey.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  shared.CompositeLookupKey(keyRingVals[0], keyRingVals[1]),
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  false,
			Out: true,
		},
	})

	return queries
}

// cryptoKeyLinkedQueries returns linked item queries for a CryptoKey
func (l *CloudKMSAssetLoader) cryptoKeyLinkedQueries(values []string, cryptoKey *kmspb.CryptoKey, scope string) []*sdp.LinkedItemQuery {
	var queries []*sdp.LinkedItemQuery
	kmsLocation := values[0]
	keyRing := values[1]
	cryptoKeyName := values[2]

	// Link to IAM Policy
	queries = append(queries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   IAMPolicy.String(),
			Method: sdp.QueryMethod_GET,
			Query:  shared.CompositeLookupKey(kmsLocation, keyRing, cryptoKeyName),
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,
			Out: true,
		},
	})

	// Link to parent KeyRing
	queries = append(queries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   CloudKMSKeyRing.String(),
			Method: sdp.QueryMethod_GET,
			Query:  shared.CompositeLookupKey(kmsLocation, keyRing),
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,
			Out: false,
		},
	})

	// Link to all CryptoKeyVersions
	queries = append(queries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   CloudKMSCryptoKeyVersion.String(),
			Method: sdp.QueryMethod_SEARCH,
			Query:  shared.CompositeLookupKey(kmsLocation, keyRing, cryptoKeyName),
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,
			Out: true,
		},
	})

	// Link to primary CryptoKeyVersion if present
	if primary := cryptoKey.GetPrimary(); primary != nil {
		if name := primary.GetName(); name != "" {
			keyVersionVals := ExtractPathParams(name, "locations", "keyRings", "cryptoKeys", "cryptoKeyVersions")
			if len(keyVersionVals) == 4 && keyVersionVals[0] != "" && keyVersionVals[1] != "" && keyVersionVals[2] != "" && keyVersionVals[3] != "" {
				queries = append(queries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   CloudKMSCryptoKeyVersion.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(keyVersionVals...),
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: true,
					},
				})
			}
		}

		// Link to ImportJob if present
		if importJob := primary.GetImportJob(); importJob != "" {
			importJobVals := ExtractPathParams(importJob, "locations", "keyRings", "importJobs")
			if len(importJobVals) == 3 && importJobVals[0] != "" && importJobVals[1] != "" && importJobVals[2] != "" {
				queries = append(queries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   CloudKMSImportJob.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(importJobVals...),
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,
						Out: false,
					},
				})
			}
		}

		// Link to EKM Connection if applicable
		if protectionLevel := primary.GetProtectionLevel(); protectionLevel == kmspb.ProtectionLevel_EXTERNAL_VPC {
			if cryptoKeyBackend := cryptoKey.GetCryptoKeyBackend(); cryptoKeyBackend != "" {
				backendVals := ExtractPathParams(cryptoKeyBackend, "locations", "ekmConnections")
				if len(backendVals) == 2 && backendVals[0] != "" && backendVals[1] != "" {
					queries = append(queries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   CloudKMSEKMConnection.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(backendVals...),
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					})
				}
			}
		}
	}

	return queries
}

// cryptoKeyVersionLinkedQueries returns linked item queries for a CryptoKeyVersion
func (l *CloudKMSAssetLoader) cryptoKeyVersionLinkedQueries(values []string, keyVersion *kmspb.CryptoKeyVersion, scope string) []*sdp.LinkedItemQuery {
	var queries []*sdp.LinkedItemQuery

	// Link to parent CryptoKey
	queries = append(queries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   CloudKMSCryptoKey.String(),
			Method: sdp.QueryMethod_GET,
			Query:  shared.CompositeLookupKey(values[0], values[1], values[2]),
			Scope:  scope,
		},
		BlastPropagation: &sdp.BlastPropagation{
			In:  true,
			Out: false,
		},
	})

	// Link to ImportJob if present
	if importJob := keyVersion.GetImportJob(); importJob != "" {
		importJobVals := ExtractPathParams(importJob, "locations", "keyRings", "importJobs")
		if len(importJobVals) == 3 && importJobVals[0] != "" && importJobVals[1] != "" && importJobVals[2] != "" {
			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   CloudKMSImportJob.String(),
					Method: sdp.QueryMethod_GET,
					Query:  shared.CompositeLookupKey(importJobVals...),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to EKM Connection if applicable
	if protectionLevel := keyVersion.GetProtectionLevel(); protectionLevel == kmspb.ProtectionLevel_EXTERNAL_VPC {
		if externalProtection := keyVersion.GetExternalProtectionLevelOptions(); externalProtection != nil {
			if ekmPath := externalProtection.GetEkmConnectionKeyPath(); ekmPath != "" {
				ekmVals := ExtractPathParams(ekmPath, "locations", "ekmConnections")
				if len(ekmVals) == 2 && ekmVals[0] != "" && ekmVals[1] != "" {
					queries = append(queries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   CloudKMSEKMConnection.String(),
							Method: sdp.QueryMethod_GET,
							Query:  shared.CompositeLookupKey(ekmVals...),
							Scope:  scope,
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					})
				}
			}
		}
	}

	return queries
}

// cryptoKeyVersionHealth returns the health status based on CryptoKeyVersion state
func (l *CloudKMSAssetLoader) cryptoKeyVersionHealth(keyVersion *kmspb.CryptoKeyVersion) *sdp.Health {
	switch keyVersion.GetState() {
	case kmspb.CryptoKeyVersion_CRYPTO_KEY_VERSION_STATE_UNSPECIFIED:
		return sdp.Health_HEALTH_UNKNOWN.Enum()
	case kmspb.CryptoKeyVersion_PENDING_GENERATION, kmspb.CryptoKeyVersion_PENDING_IMPORT:
		return sdp.Health_HEALTH_PENDING.Enum()
	case kmspb.CryptoKeyVersion_ENABLED:
		return sdp.Health_HEALTH_OK.Enum()
	case kmspb.CryptoKeyVersion_DISABLED:
		return sdp.Health_HEALTH_WARNING.Enum()
	case kmspb.CryptoKeyVersion_DESTROYED, kmspb.CryptoKeyVersion_DESTROY_SCHEDULED:
		return sdp.Health_HEALTH_ERROR.Enum()
	case kmspb.CryptoKeyVersion_IMPORT_FAILED, kmspb.CryptoKeyVersion_GENERATION_FAILED, kmspb.CryptoKeyVersion_EXTERNAL_DESTRUCTION_FAILED:
		return sdp.Health_HEALTH_ERROR.Enum()
	case kmspb.CryptoKeyVersion_PENDING_EXTERNAL_DESTRUCTION:
		return sdp.Health_HEALTH_PENDING.Enum()
	default:
		return sdp.Health_HEALTH_UNKNOWN.Enum()
	}
}

// GetItem performs the cache-lookup-load-recheck pattern for GET queries.
// Returns the item from cache, loading data if needed.
func (l *CloudKMSAssetLoader) GetItem(ctx context.Context, scope, itemType, uniqueAttr string) (*sdp.Item, *sdp.QueryError) {
	// Check cache first
	cacheHit, _, cachedItems, cachedErr, done := l.cache.Lookup(ctx, l.sourceName, sdp.QueryMethod_GET, scope, itemType, uniqueAttr, false)

	if cacheHit {
		done()
		if cachedErr != nil {
			return nil, cachedErr
		}
		if len(cachedItems) > 0 {
			return cachedItems[0], nil
		}
	}

	// Cache miss - trigger lazy bulk load
	if err := l.EnsureLoaded(ctx); err != nil {
		done()
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("failed to load KMS data from Cloud Asset API: %v", err),
		}
	}

	// Complete first lookup's pending work before second lookup to avoid self-deadlock
	done()

	// Re-check cache after bulk load
	cacheHit, _, cachedItems, cachedErr, done = l.cache.Lookup(ctx, l.sourceName, sdp.QueryMethod_GET, scope, itemType, uniqueAttr, false)
	defer done()

	if cacheHit {
		if cachedErr != nil {
			return nil, cachedErr
		}
		if len(cachedItems) > 0 {
			return cachedItems[0], nil
		}
	}

	// Item not found (may be newly created, Cloud Asset API has indexing delay)
	return nil, &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOTFOUND,
		ErrorString: fmt.Sprintf("%s %s not found (Cloud Asset API may have indexing delay for new resources)", itemType, uniqueAttr),
	}
}

// SearchItems performs the cache-lookup-load-recheck pattern for SEARCH queries.
// Streams matching items from cache, loading data if needed.
func (l *CloudKMSAssetLoader) SearchItems(ctx context.Context, stream discovery.QueryResultStream, scope, itemType, searchQuery string) {
	// Check cache first
	cacheHit, _, cachedItems, cachedErr, done := l.cache.Lookup(ctx, l.sourceName, sdp.QueryMethod_SEARCH, scope, itemType, searchQuery, false)

	if cacheHit {
		done()
		if cachedErr != nil {
			// For SEARCH, convert NOTFOUND to empty result
			if cachedErr.GetErrorType() == sdp.QueryError_NOTFOUND {
				return // Empty result is valid for SEARCH
			}
			stream.SendError(cachedErr)
			return
		}
		for _, item := range cachedItems {
			stream.SendItem(item)
		}
		return
	}

	// Cache miss - trigger lazy bulk load
	if err := l.EnsureLoaded(ctx); err != nil {
		done()
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("failed to load KMS data from Cloud Asset API: %v", err),
		})
		return
	}

	// Complete first lookup's pending work before second lookup to avoid self-deadlock
	done()

	// Re-check cache after bulk load
	cacheHit, _, cachedItems, cachedErr, done = l.cache.Lookup(ctx, l.sourceName, sdp.QueryMethod_SEARCH, scope, itemType, searchQuery, false)
	defer done()

	if cacheHit {
		if cachedErr != nil {
			// For SEARCH, convert NOTFOUND to empty result
			if cachedErr.GetErrorType() == sdp.QueryError_NOTFOUND {
				return // Empty result is valid for SEARCH
			}
			stream.SendError(cachedErr)
			return
		}
		for _, item := range cachedItems {
			stream.SendItem(item)
		}
		return
	}

	// No items found for this search - return empty result
}

// ListItems performs the cache-lookup-load-recheck pattern for LIST queries.
// Streams all items of the given type from cache, loading data if needed.
func (l *CloudKMSAssetLoader) ListItems(ctx context.Context, stream discovery.QueryResultStream, scope, itemType string) {
	// Check cache first (LIST cache key has empty query)
	cacheHit, _, cachedItems, cachedErr, done := l.cache.Lookup(ctx, l.sourceName, sdp.QueryMethod_LIST, scope, itemType, "", false)

	if cacheHit {
		done()
		if cachedErr != nil {
			// For LIST, convert NOTFOUND to empty result
			if cachedErr.GetErrorType() == sdp.QueryError_NOTFOUND {
				return // Empty result is valid for LIST
			}
			stream.SendError(cachedErr)
			return
		}
		for _, item := range cachedItems {
			stream.SendItem(item)
		}
		return
	}

	// Cache miss - trigger lazy bulk load
	if err := l.EnsureLoaded(ctx); err != nil {
		done()
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("failed to load KMS data from Cloud Asset API: %v", err),
		})
		return
	}

	// Complete first lookup's pending work before second lookup to avoid self-deadlock
	done()

	// Re-check cache after bulk load
	cacheHit, _, cachedItems, cachedErr, done = l.cache.Lookup(ctx, l.sourceName, sdp.QueryMethod_LIST, scope, itemType, "", false)
	defer done()

	if cacheHit {
		if cachedErr != nil {
			// For LIST, convert NOTFOUND to empty result
			if cachedErr.GetErrorType() == sdp.QueryError_NOTFOUND {
				return // Empty result is valid for LIST
			}
			stream.SendError(cachedErr)
			return
		}
		for _, item := range cachedItems {
			stream.SendItem(item)
		}
		return
	}

	// No items found - return empty result
}
