package adapters

import (
	"context"

	compute "cloud.google.com/go/compute/apiv1"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/shared"
)

var Metadata = sdp.AdapterMetadataList{}

// Adapters returns a slice of discovery.Adapter instances for GCP Source.
func Adapters(ctx context.Context, projectID string, regions []string, zones []string) ([]discovery.Adapter, error) {
	instanceCli, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return nil, err
	}

	addressCli, err := compute.NewAddressesRESTClient(ctx)
	if err != nil {
		return nil, err
	}

	var adapters []discovery.Adapter

	for _, region := range regions {
		adapter := sources.WrapperToAdapter(
			NewComputeAddress(shared.NewComputeAddressClient(addressCli), projectID, region),
		)

		Metadata.Register(adapter.Metadata())

		adapters = append(adapters, adapter)
	}

	for _, zone := range zones {
		adapter := sources.WrapperToAdapter(
			NewComputeInstance(shared.NewComputeInstanceClient(instanceCli), projectID, zone),
		)

		Metadata.Register(adapter.Metadata())

		adapters = append(adapters, adapter)
	}

	return adapters, nil
}
