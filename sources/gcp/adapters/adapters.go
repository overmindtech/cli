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

	autoscalerCli, err := compute.NewAutoscalersRESTClient(ctx)
	if err != nil {
		return nil, err
	}

	computeImagesCli, err := compute.NewImagesRESTClient(ctx)
	if err != nil {
		return nil, err
	}

	computeForwardingCli, err := compute.NewForwardingRulesRESTClient(ctx)
	if err != nil {
		return nil, err
	}

	backendServiceCli, err := compute.NewBackendServicesRESTClient(ctx)
	if err != nil {
		return nil, err
	}

	var adapters []discovery.Adapter

	for _, region := range regions {
		adapters = append(adapters,
			sources.WrapperToAdapter(NewComputeAddress(shared.NewComputeAddressClient(addressCli), projectID, region)),
			sources.WrapperToAdapter(NewComputeForwardingRule(shared.NewComputeForwardingRuleClient(computeForwardingCli), projectID, region)),
		)
	}

	for _, zone := range zones {
		adapters = append(adapters,
			sources.WrapperToAdapter(NewComputeInstance(shared.NewComputeInstanceClient(instanceCli), projectID, zone)),
			sources.WrapperToAdapter(NewComputeAutoscaler(shared.NewComputeAutoscalerClient(autoscalerCli), projectID, zone)),
		)
	}

	// global - project level - adapters
	adapters = append(adapters,
		sources.WrapperToAdapter(NewComputeBackendService(shared.NewComputeBackendServiceClient(backendServiceCli), projectID)),
		sources.WrapperToAdapter(NewComputeImage(shared.NewComputeImagesClient(computeImagesCli), projectID)),
	)

	// Register the metadata for each adapter
	for _, adapter := range adapters {
		Metadata.Register(adapter.Metadata())
	}

	return adapters, nil
}
