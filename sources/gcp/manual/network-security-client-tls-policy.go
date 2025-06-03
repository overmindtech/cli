package manual

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/networksecurity/apiv1beta1/networksecuritypb"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	NetworkSecurityClientTlsPolicy = shared.NewItemType(gcpshared.GCP, gcpshared.NetworkSecurity, gcpshared.ClientTlsPolicy)

	NetworkSecurityClientTlsPolicyLookupByName     = shared.NewItemTypeLookup("name", NetworkSecurityClientTlsPolicy)
	NetworkSecurityClientTlsPolicyLookupByLocation = shared.NewItemTypeLookup("location", NetworkSecurityClientTlsPolicy)
)

type networkSecurityClientTlsPolicyWrapper struct {
	client gcpshared.NetworkSecurityClientTlsPolicyClient

	*gcpshared.ProjectBase
}

// NewNetworkSecurityClientTlsPolicy creates a new networkSecurityClientTlsPolicyWrapper instance
func NewNetworkSecurityClientTlsPolicy(client gcpshared.NetworkSecurityClientTlsPolicyClient, projectID string) sources.SearchableWrapper {
	return &networkSecurityClientTlsPolicyWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			NetworkSecurityClientTlsPolicy,
		),
	}
}

func (n networkSecurityClientTlsPolicyWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		// The order of these lookups matters
		// Location must be first
		NetworkSecurityClientTlsPolicyLookupByLocation,
		NetworkSecurityClientTlsPolicyLookupByName,
	}
}

func (n networkSecurityClientTlsPolicyWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	// A name of the ClientTlsPolicy to get. Must be in the format projects/*/locations/{location}/clientTlsPolicies/*.
	// https://cloud.google.com/service-mesh/docs/reference/network-security/rest/v1/projects.locations.clientTlsPolicies/get
	req := &networksecuritypb.GetClientTlsPolicyRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/clientTlsPolicies/%s", n.ProjectID(), queryParts[0], queryParts[1]),
	}

	p, err := n.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	// Convert the ClientTlsPolicy to a sdp.Item
	item, sdpErr := n.convertClientTlsPolicyToItem(p)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

func (n networkSecurityClientTlsPolicyWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			NetworkSecurityClientTlsPolicyLookupByLocation,
		},
	}
}

func (n networkSecurityClientTlsPolicyWrapper) Search(ctx context.Context, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	req := &networksecuritypb.ListClientTlsPoliciesRequest{
		// Required. The project and location from which the ClientTlsPolicies should
		// be listed, specified in the format `projects/*/locations/{location}`.
		Parent: fmt.Sprintf("projects/%s/locations/%s", n.ProjectID(), queryParts[0]),
	}

	it := n.client.List(ctx, req)
	var items []*sdp.Item
	for {
		p, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err)
		}

		item, sdpErr := n.convertClientTlsPolicyToItem(p)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (n networkSecurityClientTlsPolicyWrapper) convertClientTlsPolicyToItem(p *networksecuritypb.ClientTlsPolicy) (*sdp.Item, *sdp.QueryError) {
	log.Warnf("Not implemented yet: convertClientTlsPolicyToItem")

	return nil, nil
}
