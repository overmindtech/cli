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

var ComputeImageLookupByName = shared.NewItemTypeLookup("name", gcpshared.ComputeImage)

type computeImageWrapper struct {
	client gcpshared.ComputeImagesClient

	*gcpshared.ProjectBase
}

// NewComputeImage creates a new computeImageWrapper instance
func NewComputeImage(client gcpshared.ComputeImagesClient, projectID string) sources.ListableWrapper {
	return &computeImageWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			gcpshared.ComputeImage,
		),
	}
}

// TerraformMappings returns the Terraform mappings for the compute image wrapper
func (c computeImageWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_GET,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/compute_health_check#argument-reference
			TerraformQueryMap: "google_compute_image.name",
		},
	}
}

// GetLookups returns the lookups for the compute image wrapper
func (c computeImageWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ComputeImageLookupByName,
	}
}

// Get retrieves a compute image by its name
func (c computeImageWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &computepb.GetImageRequest{
		Project: c.ProjectID(),
		Image:   queryParts[0],
	}

	image, err := c.client.Get(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = c.gcpComputeImageToSDPItem(image)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil

}

// List lists compute images and converts them to sdp.Items.
func (c computeImageWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	it := c.client.List(ctx, &computepb.ListImagesRequest{
		Project: c.ProjectID(),
	})

	var items []*sdp.Item
	for {
		image, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err)
		}

		var sdpErr *sdp.QueryError
		var item *sdp.Item
		item, sdpErr = c.gcpComputeImageToSDPItem(image)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (c computeImageWrapper) gcpComputeImageToSDPItem(image *computepb.Image) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(image, "labels")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.ComputeImage.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           c.DefaultScope(),
		Tags:            image.GetLabels(),
	}

	switch image.GetStatus() {
	case computepb.Image_UNDEFINED_STATUS.String():
		sdpItem.Health = sdp.Health_HEALTH_UNKNOWN.Enum()
	case computepb.Image_FAILED.String():
		sdpItem.Health = sdp.Health_HEALTH_ERROR.Enum()
	case computepb.Image_PENDING.String(),
		computepb.Image_DELETING.String():
		sdpItem.Health = sdp.Health_HEALTH_PENDING.Enum()
	case computepb.Image_READY.String():
		sdpItem.Health = sdp.Health_HEALTH_OK.Enum()
	}

	return sdpItem, nil
}
