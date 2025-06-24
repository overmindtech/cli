package manual

import (
	"context"
	"errors"

	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
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

func (c computeHealthCheckWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetHealthCheckRequest{
		Project:     c.ProjectID(),
		HealthCheck: queryParts[0],
	}

	healthCheck, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = c.gcpComputeHealthCheckToSDPItem(healthCheck)
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
			return nil, gcpshared.QueryError(err)
		}

		var sdpErr *sdp.QueryError
		var item *sdp.Item
		item, sdpErr = c.gcpComputeHealthCheckToSDPItem(healthCheck)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

// gcpComputeHealthCheckToSDPItem converts a GCP HealthCheck to an SDP Item
func (c computeHealthCheckWrapper) gcpComputeHealthCheckToSDPItem(healthCheck *computepb.HealthCheck) (*sdp.Item, *sdp.QueryError) {
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

	//Healthcheck type has no relevant links

	return sdpItem, nil
}
