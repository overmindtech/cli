package manual

import (
	"context"
	"errors"
	"fmt"
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
	globalClient     gcpshared.ComputeHealthCheckClient
	regionalClient   gcpshared.ComputeRegionHealthCheckClient
	projectLocations []gcpshared.LocationInfo // For global health checks
	regionLocations  []gcpshared.LocationInfo // For regional health checks
	*shared.Base
}

// NewComputeHealthCheck creates a new computeHealthCheckWrapper instance that handles both global and regional health checks.
func NewComputeHealthCheck(globalClient gcpshared.ComputeHealthCheckClient, regionalClient gcpshared.ComputeRegionHealthCheckClient, projectLocations []gcpshared.LocationInfo, regionLocations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	// Combine all locations for scope generation
	allLocations := make([]gcpshared.LocationInfo, 0, len(projectLocations)+len(regionLocations))
	allLocations = append(allLocations, projectLocations...)
	allLocations = append(allLocations, regionLocations...)

	scopes := make([]string, 0, len(allLocations))
	for _, location := range allLocations {
		scopes = append(scopes, location.ToScope())
	}

	return &computeHealthCheckWrapper{
		globalClient:     globalClient,
		regionalClient:   regionalClient,
		projectLocations: projectLocations,
		regionLocations:  regionLocations,
		Base:             shared.NewBase(sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION, gcpshared.ComputeHealthCheck, scopes),
	}
}

// validateAndParseScope parses the scope and validates it against configured locations.
// Returns the LocationInfo if valid, or a QueryError if the scope is invalid or not configured.
func (c computeHealthCheckWrapper) validateAndParseScope(scope string) (gcpshared.LocationInfo, *sdp.QueryError) {
	location, err := gcpshared.LocationFromScope(scope)
	if err != nil {
		return gcpshared.LocationInfo{}, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	// Check if the location is in the adapter's configured locations
	allLocations := append([]gcpshared.LocationInfo{}, c.projectLocations...)
	allLocations = append(allLocations, c.regionLocations...)

	for _, configuredLoc := range allLocations {
		if location.Equals(configuredLoc) {
			return location, nil
		}
	}

	return gcpshared.LocationInfo{}, &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOSCOPE,
		ErrorString: fmt.Sprintf("scope %s not found in adapter's configured locations", scope),
	}
}

func (c computeHealthCheckWrapper) IAMPermissions() []string {
	return []string{
		"compute.healthChecks.get",
		"compute.healthChecks.list",
		"compute.regionHealthChecks.get",
		"compute.regionHealthChecks.list",
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
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_compute_region_health_check.name",
		},
	}
}

func (c computeHealthCheckWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeHealthCheckLookupByName,
	}
}

func (c computeHealthCheckWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	// Parse and validate the scope
	location, err := c.validateAndParseScope(scope)
	if err != nil {
		return nil, err
	}

	// Route to the appropriate API based on whether the scope includes a region
	if location.Regional() {
		// Regional health check
		req := &computepb.GetRegionHealthCheckRequest{
			Project:     location.ProjectID,
			Region:      location.Region,
			HealthCheck: queryParts[0],
		}

		healthCheck, getErr := c.regionalClient.Get(ctx, req)
		if getErr != nil {
			return nil, gcpshared.QueryError(getErr, scope, c.Type())
		}

		return GcpComputeHealthCheckToSDPItem(healthCheck, location, gcpshared.ComputeHealthCheck)
	}

	// Global health check
	req := &computepb.GetHealthCheckRequest{
		Project:     location.ProjectID,
		HealthCheck: queryParts[0],
	}

	healthCheck, getErr := c.globalClient.Get(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	return GcpComputeHealthCheckToSDPItem(healthCheck, location, gcpshared.ComputeHealthCheck)
}

func (c computeHealthCheckWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	// Parse and validate the scope
	location, err := c.validateAndParseScope(scope)
	if err != nil {
		return nil, err
	}

	var items []*sdp.Item

	// Route to the appropriate API based on whether the scope includes a region
	if location.Regional() {
		// Regional health checks
		it := c.regionalClient.List(ctx, &computepb.ListRegionHealthChecksRequest{
			Project: location.ProjectID,
			Region:  location.Region,
		})

		for {
			healthCheck, iterErr := it.Next()
			if errors.Is(iterErr, iterator.Done) {
				break
			}
			if iterErr != nil {
				return nil, gcpshared.QueryError(iterErr, scope, c.Type())
			}

			item, sdpErr := GcpComputeHealthCheckToSDPItem(healthCheck, location, gcpshared.ComputeHealthCheck)
			if sdpErr != nil {
				return nil, sdpErr
			}

			items = append(items, item)
		}
	} else {
		// Global health checks
		it := c.globalClient.List(ctx, &computepb.ListHealthChecksRequest{
			Project: location.ProjectID,
		})

		for {
			healthCheck, iterErr := it.Next()
			if errors.Is(iterErr, iterator.Done) {
				break
			}
			if iterErr != nil {
				return nil, gcpshared.QueryError(iterErr, scope, c.Type())
			}

			item, sdpErr := GcpComputeHealthCheckToSDPItem(healthCheck, location, gcpshared.ComputeHealthCheck)
			if sdpErr != nil {
				return nil, sdpErr
			}

			items = append(items, item)
		}
	}

	return items, nil
}

func (c computeHealthCheckWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	// Parse and validate the scope
	location, err := c.validateAndParseScope(scope)
	if err != nil {
		stream.SendError(err)
		return
	}

	// Route to the appropriate API based on whether the scope includes a region
	if location.Regional() {
		// Regional health checks
		it := c.regionalClient.List(ctx, &computepb.ListRegionHealthChecksRequest{
			Project: location.ProjectID,
			Region:  location.Region,
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

			item, sdpErr := GcpComputeHealthCheckToSDPItem(healthCheck, location, gcpshared.ComputeHealthCheck)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}

			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	} else {
		// Global health checks
		it := c.globalClient.List(ctx, &computepb.ListHealthChecksRequest{
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

			item, sdpErr := GcpComputeHealthCheckToSDPItem(healthCheck, location, gcpshared.ComputeHealthCheck)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}

			cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}

// GcpComputeHealthCheckToSDPItem converts a GCP health check to an SDP item.
// This function is shared by both global and regional health check adapters.
func GcpComputeHealthCheckToSDPItem(healthCheck *computepb.HealthCheck, location gcpshared.LocationInfo, itemType shared.ItemType) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(healthCheck)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            itemType.String(),
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
