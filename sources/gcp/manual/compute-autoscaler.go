package manual

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/compute/apiv1/computepb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var ComputeAutoscalerLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeAutoscaler)

type computeAutoscalerWrapper struct {
	client gcpshared.ComputeAutoscalerClient

	*gcpshared.ZoneBase
}

// NewComputeAutoscaler creates a new computeAutoscalerWrapper instance.
func NewComputeAutoscaler(client gcpshared.ComputeAutoscalerClient, projectID, zone string) sources.ListableWrapper {
	return &computeAutoscalerWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			projectID,
			zone,
			sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
			gcpshared.ComputeAutoscaler,
		),
	}
}

func (c computeAutoscalerWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.ComputeInstanceGroupManager,
	)
}

func (c computeAutoscalerWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_address#argument-reference
			TerraformQueryMap: "google_compute_autoscaler.name",
		},
	}
}

func (c computeAutoscalerWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeAutoscalerLookupByName,
	}
}

func (c computeAutoscalerWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetAutoscalerRequest{
		Project:    c.ProjectID(),
		Zone:       c.Zone(),
		Autoscaler: queryParts[0],
	}

	autoscaler, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = c.gcpComputeAutoscalerToSDPItem(autoscaler)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

func (c computeAutoscalerWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	results := c.client.List(ctx, &computepb.ListAutoscalersRequest{
		Project: c.ProjectID(),
		Zone:    c.Zone(),
	})

	var items []*sdp.Item
	for {
		autoscaler, err := results.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return nil, gcpshared.QueryError(err)
		}

		var sdpErr *sdp.QueryError
		var item *sdp.Item
		item, sdpErr = c.gcpComputeAutoscalerToSDPItem(autoscaler)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeAutoscalerWrapper) gcpComputeAutoscalerToSDPItem(autoscaler *computepb.Autoscaler) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(autoscaler)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeAutoscaler.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),
		// Autoscalers don't have labels.
	}

	instanceGroupManagerName := autoscaler.GetTarget()
	if instanceGroupManagerName != "" {
		zone := gcpshared.ExtractPathParam("zones", instanceGroupManagerName)
		igmNameParts := strings.Split(instanceGroupManagerName, "/")
		igmName := igmNameParts[len(igmNameParts)-1]

		sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
			Query: &sdp.Query{
				Type:   gcpshared.ComputeInstanceGroupManager.String(),
				Method: sdp.QueryMethod_GET,
				Query:  igmName,
				Scope:  gcpshared.ZonalScope(c.ProjectID(), zone),
			},
			BlastPropagation: &sdp.BlastPropagation{
				// Updating the IGM will affect this autoscaler's operation, but it's a weak connection.
				In: true,

				// Updating the autoscaler will directly affect the IGM.
				Out: true,
			},
		})
	}

	return sdpItem, nil
}
