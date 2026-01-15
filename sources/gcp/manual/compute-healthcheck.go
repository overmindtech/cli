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
func NewComputeHealthCheck(client gcpshared.ComputeHealthCheckClient, projectID string) sources.ListableWrapper {
	return &computeHealthCheckWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
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

// PotentialLinks returns the potential links for the compute health check wrapper
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
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_health_check#argument-reference
			TerraformQueryMap: "google_compute_health_check.name",
		},
	}
}

func (c computeHealthCheckWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeHealthCheckLookupByName,
	}
}

func (c computeHealthCheckWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetHealthCheckRequest{
		Project:     c.ProjectID(),
		HealthCheck: queryParts[0],
	}

	healthCheck, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = c.gcpComputeHealthCheckToSDPItem(ctx, healthCheck)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

func (c computeHealthCheckWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	results := c.client.List(ctx, &computepb.ListHealthChecksRequest{
		Project: c.ProjectID(),
	})

	var items []*sdp.Item
	for {
		healthCheck, err := results.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return nil, gcpshared.QueryError(err, c.DefaultScope(), c.Type())
		}

		var sdpErr *sdp.QueryError
		var item *sdp.Item
		item, sdpErr = c.gcpComputeHealthCheckToSDPItem(ctx, healthCheck)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

// ListStream implements the Streamer interface
func (c computeHealthCheckWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	it := c.client.List(ctx, &computepb.ListHealthChecksRequest{
		Project: c.ProjectID(),
	})

	for {
		healthCheck, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(gcpshared.QueryError(err, c.DefaultScope(), c.Type()))
			return
		}

		item, sdpErr := c.gcpComputeHealthCheckToSDPItem(ctx, healthCheck)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

// gcpComputeHealthCheckToSDPItem converts a GCP HealthCheck to an SDP Item
func (c computeHealthCheckWrapper) gcpComputeHealthCheckToSDPItem(ctx context.Context, healthCheck *computepb.HealthCheck) (*sdp.Item, *sdp.QueryError) {
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
		Scope:           c.DefaultScope(),
	}

	// Link to host field from HTTP health checks (can be IP address or DNS name)
	if httpHealthCheck := healthCheck.GetHttpHealthCheck(); httpHealthCheck != nil {
		if host := httpHealthCheck.GetHost(); host != "" {
			linkHostToNetworkResource(sdpItem, host)
		}
	}

	// Link to host field from HTTPS health checks (can be IP address or DNS name)
	if httpsHealthCheck := healthCheck.GetHttpsHealthCheck(); httpsHealthCheck != nil {
		if host := httpsHealthCheck.GetHost(); host != "" {
			linkHostToNetworkResource(sdpItem, host)
		}
	}

	// Link to host field from HTTP/2 health checks (can be IP address or DNS name)
	if http2HealthCheck := healthCheck.GetHttp2HealthCheck(); http2HealthCheck != nil {
		if host := http2HealthCheck.GetHost(); host != "" {
			linkHostToNetworkResource(sdpItem, host)
		}
	}

	// Link to source regions (array of region names for global health checks)
	for _, regionName := range healthCheck.GetSourceRegions() {
		if regionName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   gcpshared.ComputeRegion.String(),
					Method: sdp.QueryMethod_GET,
					Query:  regionName,
					Scope:  c.ProjectID(), // Regions are project-scoped resources
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,
					Out: false,
				},
			})
		}
	}

	// Link to region field (output only, for regional health checks)
	if region := healthCheck.GetRegion(); region != "" {
		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   gcpshared.ComputeRegion.String(),
				Method: sdp.QueryMethod_GET,
				Query:  region,
				Scope:  c.ProjectID(), // Regions are project-scoped resources
			},
			BlastPropagation: &sdp.BlastPropagation{
				In:  true,
				Out: false,
			},
		})
	}

	return sdpItem, nil
}

// linkHostToNetworkResource creates a linked item query for a host value that can be either an IP address or DNS name.
// The linker automatically detects the type, but for manual adapters we need to determine it ourselves.
func linkHostToNetworkResource(sdpItem *sdp.Item, host string) {
	if host == "" {
		return
	}

	// Check if the host is an IP address
	if net.ParseIP(host) != nil {
		// It's an IP address
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
		// It's a DNS name
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
