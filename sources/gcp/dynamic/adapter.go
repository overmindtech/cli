package dynamic

import (
	"context"
	"fmt"
	"net/http"

	"buf.build/go/protovalidate"
	log "github.com/sirupsen/logrus"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

// AdapterConfig holds the configuration for a GCP dynamic adapter.
type AdapterConfig struct {
	Locations            []gcpshared.LocationInfo
	GetURLFunc           gcpshared.EndpointFunc
	SDPAssetType         shared.ItemType
	SDPAdapterCategory   sdp.AdapterCategory
	TerraformMappings    []*sdp.TerraformMapping
	Linker               *gcpshared.Linker
	HTTPClient           *http.Client
	UniqueAttributeKeys  []string
	IAMPermissions       []string // List of IAM permissions required by the adapter
	NameSelector         string   // By default, it is `name`, but can be overridden for outlier cases
	ListResponseSelector string
}

// Adapter implements discovery.ListableAdapter for GCP dynamic adapters.
type Adapter struct {
	locations            []gcpshared.LocationInfo
	httpCli              *http.Client
	cache                sdpcache.Cache
	getURLFunc           gcpshared.EndpointFunc
	sdpAssetType         shared.ItemType
	sdpAdapterCategory   sdp.AdapterCategory
	terraformMappings    []*sdp.TerraformMapping
	potentialLinks       []string
	linker               *gcpshared.Linker
	uniqueAttributeKeys  []string
	iamPermissions       []string
	nameSelector         string // By default, it is `name`, but can be overridden for outlier cases
	listResponseSelector string
}

// NewAdapter creates a new GCP dynamic adapter.
func NewAdapter(config *AdapterConfig, cache sdpcache.Cache) discovery.Adapter {
	return Adapter{
		locations:            config.Locations,
		httpCli:              config.HTTPClient,
		cache:                cache,
		getURLFunc:           config.GetURLFunc,
		sdpAssetType:         config.SDPAssetType,
		sdpAdapterCategory:   config.SDPAdapterCategory,
		terraformMappings:    config.TerraformMappings,
		linker:               config.Linker,
		potentialLinks:       potentialLinksFromBlasts(config.SDPAssetType, gcpshared.BlastPropagations),
		uniqueAttributeKeys:  config.UniqueAttributeKeys,
		iamPermissions:       config.IAMPermissions,
		nameSelector:         config.NameSelector,
		listResponseSelector: config.ListResponseSelector,
	}
}

func (g Adapter) Type() string {
	return g.sdpAssetType.String()
}

func (g Adapter) Name() string {
	return fmt.Sprintf("%s-adapter", g.sdpAssetType.String())
}

func (g Adapter) Metadata() *sdp.AdapterMetadata {
	return &sdp.AdapterMetadata{
		Type:            g.sdpAssetType.String(),
		Category:        g.sdpAdapterCategory,
		DescriptiveName: g.sdpAssetType.Readable(),
		SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
			Get:            true,
			GetDescription: getDescription(g.sdpAssetType, g.uniqueAttributeKeys),
		},
		TerraformMappings: g.terraformMappings,
		PotentialLinks:    g.potentialLinks,
	}
}

func (g Adapter) Scopes() []string {
	return gcpshared.LocationsToScopes(g.locations)
}

// validateScope checks if the requested scope matches one of the adapter's locations.
func (g Adapter) validateScope(scope string) (gcpshared.LocationInfo, error) {
	requestedLoc, err := gcpshared.LocationFromScope(scope)
	if err != nil {
		return gcpshared.LocationInfo{}, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("invalid scope format: %v", err),
		}
	}

	for _, validLoc := range g.locations {
		if requestedLoc.Equals(validLoc) {
			return requestedLoc, nil
		}
	}
	return gcpshared.LocationInfo{}, &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOSCOPE,
		ErrorString: fmt.Sprintf("requested scope %v does not match any adapter scope %v", scope, g.Scopes()),
	}
}

func (g Adapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	location, err := g.validateScope(scope)
	if err != nil {
		return nil, err
	}

	cacheHit, ck, cachedItem, qErr, done := g.cache.Lookup(
		ctx,
		g.Name(),
		sdp.QueryMethod_GET,
		scope,
		g.Type(),
		query,
		ignoreCache,
	)
	defer done()
	if qErr != nil {
		log.WithContext(ctx).WithFields(log.Fields{
			"ovm.source.type":      "gcp",
			"ovm.source.adapter":   g.Name(),
			"ovm.source.scope":     scope,
			"ovm.source.method":    sdp.QueryMethod_GET.String(),
			"ovm.source.cache-key": ck,
		}).WithError(qErr).Error("failed to lookup item in cache")
	}

	if cacheHit && len(cachedItem) > 0 {
		return cachedItem[0], nil
	}

	url := g.getURLFunc(query, location)
	if url == "" {
		err := &sdp.QueryError{
			ErrorType: sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf(
				"failed to construct the URL for the query \"%s\". GET method description: %s",
				query,
				g.Metadata().GetSupportedQueryMethods().GetGetDescription(),
			),
		}
		g.cache.StoreError(ctx, err, shared.DefaultCacheDuration, ck)
		return nil, err
	}

	resp, err := externalCallSingle(ctx, g.httpCli, url)
	if err != nil {
		g.cache.StoreError(ctx, err, shared.DefaultCacheDuration, ck)
		return nil, err
	}

	item, err := externalToSDP(ctx, location, g.uniqueAttributeKeys, resp, g.sdpAssetType, g.linker, g.nameSelector)
	if err != nil {
		g.cache.StoreError(ctx, err, shared.DefaultCacheDuration, ck)
		return nil, err
	}

	g.cache.StoreItem(ctx, item, shared.DefaultCacheDuration, ck)

	return item, nil
}

func (g Adapter) Validate() error {
	return protovalidate.Validate(g.Metadata())
}
