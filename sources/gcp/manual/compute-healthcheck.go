package manual

import (
	"context"
	"errors"
	"net"

	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var ComputeHealthCheckLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeHealthCheck)

type computeHealthCheckWrapper struct {
	client gcpshared.ComputeHealthCheckClient
	*gcpshared.ProjectBase
}

// NewComputeHealthCheck creates a new computeHealthCheckWrapper instance.
func NewComputeHealthCheck(client gcpshared.ComputeHealthCheckClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &computeHealthCheckWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
			gcpshared.ComputeHealthCheck,
		),
	}
}

func (c computeHealthCheckWrapper) IAMPermissions() []string {
	return []string{
		"compute.healthChecks.get",
		"compute.healthChecks.list",
	}
}

func (c computeHealthCheckWrapper) PredefinedRole() string {
	return "roles/compute.viewer"
}

func (c computeHealthCheckWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		stdlib.NetworkIP,
		stdlib.NetworkDNS,
		gcpshared.ComputeRegion,
	)
}

func (c computeHealthCheckWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_health_check.name",
		},
	}
}

func (c computeHealthCheckWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeHealthCheckLookupByName,
	}
}

func (c computeHealthCheckWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	req := &computepb.GetHealthCheckRequest{
		Project:     location.ProjectID,
		HealthCheck: queryParts[0],
	}

	healthCheck, getErr := c.client.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return c.gcpComputeHealthCheckToSDPItem(healthCheck, location)
}

func (c computeHealthCheckWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	it := c.client.List(ctx, &computepb.ListHealthChecksRequest{
		Project: location.ProjectID,
	})

	var items []*sdp.Item
	for {
		healthCheck, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, gcpshared.QueryError(iterErr, scope, c.Type())
		}

		item, sdpErr := c.gcpComputeHealthCheckToSDPItem(healthCheck, location)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeHealthCheckWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	it := c.client.List(ctx, &computepb.ListHealthChecksRequest{
		Project: location.ProjectID,
	})

	for {
		healthCheck, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeHealthCheckToSDPItem(healthCheck, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

func (c computeHealthCheckWrapper) gcpComputeHealthCheckToSDPItem(healthCheck *computepb.HealthCheck, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(healthCheck)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeHealthCheck.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           location.ToScope(),
	}

	// Link to host field from HTTP health checks
	if httpHealthCheck := healthCheck.GetHttpHealthCheck(); httpHealthCheck != nil {
		if host := httpHealthCheck.GetHost(); host != "" {
			linkHostToNetworkResource(sdpItem, host)
		}
	}

	// Link to host field from HTTPS health checks
	if httpsHealthCheck := healthCheck.GetHttpsHealthCheck(); httpsHealthCheck != nil {
		if host := httpsHealthCheck.GetHost(); host != "" {
			linkHostToNetworkResource(sdpItem, host)
		}
	}

	// Link to host field from HTTP/2 health checks
	if http2HealthCheck := healthCheck.GetHttp2HealthCheck(); http2HealthCheck != nil {
		if host := http2HealthCheck.GetHost(); host != "" {
			linkHostToNetworkResource(sdpItem, host)
		}
	}

	// Link to source regions
	for _, regionName := range healthCheck.GetSourceRegions() {
		if regionName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeRegion.String(),
					Method: sdp.QueryMethod_GET,
					Query:  regionName,
					Scope:  location.ProjectID,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to region field
	if region := healthCheck.GetRegion(); region != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   gcpshared.ComputeRegion.String(),
				Method: sdp.QueryMethod_GET,
				Query:  region,
				Scope:  location.ProjectID,
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		})
	}

	return sdpItem, nil
}

func linkHostToNetworkResource(sdpItem *sdp.Item, host string) {
	if host == "" {
		return
	}

	if net.ParseIP(host) != nil {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkIP.String(),
				Method: sdp.QueryMethod_GET,
				Query:  host,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		})
	} else {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   stdlib.NetworkDNS.String(),
				Method: sdp.QueryMethod_SEARCH,
				Query:  host,
				Scope:  "global",
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: true,
			},
		})
	}
}
