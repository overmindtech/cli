package dynamic

import (
	"context"
	"fmt"
	"net/http"

	"buf.build/go/protovalidate"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

// AdapterConfig holds the configuration for a GCP dynamic adapter.
type AdapterConfig struct {
	ProjectID           string
	Scope               string
	GetURLFunc          gcpshared.EndpointFunc
	SDPAssetType        shared.ItemType
	SDPAdapterCategory  sdp.AdapterCategory
	TerraformMappings   []*sdp.TerraformMapping
	Linker              *gcpshared.Linker
	HTTPClient          *http.Client
	UniqueAttributeKeys []string
	IAMPermissions      []string // List of IAM permissions required by the adapter
}

// Adapter implements discovery.ListableAdapter for GCP dynamic adapters.
type Adapter struct {
	projectID           string
	httpCli             *http.Client
	getURLFunc          gcpshared.EndpointFunc
	scope               string
	sdpAssetType        shared.ItemType
	sdpAdapterCategory  sdp.AdapterCategory
	terraformMappings   []*sdp.TerraformMapping
	potentialLinks      []string
	linker              *gcpshared.Linker
	uniqueAttributeKeys []string
	iamPermissions      []string
}

// NewAdapter creates a new GCP dynamic adapter.
func NewAdapter(config *AdapterConfig) (discovery.Adapter, error) {
	a := Adapter{
		projectID:           config.ProjectID,
		scope:               config.Scope,
		httpCli:             config.HTTPClient,
		getURLFunc:          config.GetURLFunc,
		sdpAssetType:        config.SDPAssetType,
		sdpAdapterCategory:  config.SDPAdapterCategory,
		terraformMappings:   config.TerraformMappings,
		linker:              config.Linker,
		potentialLinks:      potentialLinksFromBlasts(config.SDPAssetType, gcpshared.BlastPropagations),
		uniqueAttributeKeys: config.UniqueAttributeKeys,
		iamPermissions:      config.IAMPermissions,
	}

	if a.httpCli == nil {
		gcpHTTPCliWithOtel, err := gcpshared.GCPHTTPClientWithOtel()
		if err != nil {
			return nil, err
		}

		a.httpCli = gcpHTTPCliWithOtel
	}

	return a, nil
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
				"failed to construct the URL for the query \"%s\". GET method description: %s",
				query,
				g.Metadata().GetSupportedQueryMethods().GetGetDescription(),
			),
		}
	}

	resp, err := externalCallSingle(ctx, g.httpCli, url)
	if err != nil {
		return nil, err
	}

	return externalToSDP(ctx, g.projectID, g.scope, g.uniqueAttributeKeys, resp, g.sdpAssetType, g.linker)
}

func (g Adapter) Validate() error {
	return protovalidate.Validate(g.Metadata())
}
