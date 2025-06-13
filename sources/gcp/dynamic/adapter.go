package dynamic

import (
	"context"
	"fmt"
	"net/http"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

// AdapterConfig holds the configuration for a GCP dynamic adapter.
type AdapterConfig struct {
	ProjectID           string
	Token               string
	Scope               string
	GetURLFunc          gcpshared.EndpointFunc
	SDPAssetType        shared.ItemType
	SDPAdapterCategory  sdp.AdapterCategory
	TerraformMappings   []*sdp.TerraformMapping
	Linker              *gcpshared.Linker
	HTTPClient          *http.Client
	UniqueAttributeKeys []string
}

// Adapter implements discovery.ListableAdapter for GCP dynamic adapters.
type Adapter struct {
	projectID           string
	httpCli             *http.Client
	httpHeaders         http.Header
	getURLFunc          gcpshared.EndpointFunc
	scope               string
	sdpAssetType        shared.ItemType
	sdpAdapterCategory  sdp.AdapterCategory
	terraformMappings   []*sdp.TerraformMapping
	potentialLinks      []string
	linker              *gcpshared.Linker
	uniqueAttributeKeys []string
}

// NewAdapter creates a new GCP dynamic adapter.
func NewAdapter(config *AdapterConfig) discovery.Adapter {

	return Adapter{
		projectID:  config.ProjectID,
		scope:      config.Scope,
		httpCli:    config.HTTPClient,
		getURLFunc: config.GetURLFunc,
		httpHeaders: http.Header{
			"Authorization": []string{"Bearer " + config.Token},
		},
		sdpAssetType:        config.SDPAssetType,
		sdpAdapterCategory:  config.SDPAdapterCategory,
		terraformMappings:   config.TerraformMappings,
		linker:              config.Linker,
		potentialLinks:      potentialLinksFromBlasts(config.SDPAssetType, gcpshared.BlastPropagations),
		uniqueAttributeKeys: config.UniqueAttributeKeys,
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
			GetDescription: getDescription(g.sdpAssetType, g.scope, g.uniqueAttributeKeys),
		},
		TerraformMappings: g.terraformMappings,
		PotentialLinks:    g.potentialLinks,
	}
}

func (g Adapter) Scopes() []string {
	return []string{g.scope}
}

func (g Adapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	if scope != g.scope {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: fmt.Sprintf("requested scope %v does not match any adapter scope %v", scope, g.Scopes()),
		}
	}

	url := g.getURLFunc(query)
	if url == "" {
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf(
				"no get endpoint found for query \"%s\". %s",
				query,
				g.Metadata().GetSupportedQueryMethods().GetGetDescription(),
			),
		}
	}

	resp, err := externalCallSingle(ctx, g.httpCli, g.httpHeaders, url)
	if err != nil {
		return nil, err
	}

	return externalToSDP(ctx, g.projectID, g.scope, g.uniqueAttributeKeys, resp, g.sdpAssetType, g.linker)
}
