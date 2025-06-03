package manual

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

	computeHealthCheckCli, err := compute.NewHealthChecksRESTClient(ctx)
	if err != nil {
		return nil, err
	}

	computeReservationCli, err := compute.NewReservationsRESTClient(ctx)
	if err != nil {
		return nil, err
	}

	computeSecurityPolicyCli, err := compute.NewSecurityPoliciesRESTClient(ctx)
	if err != nil {
		return nil, err
	}

	computeSnapshotCli, err := compute.NewSnapshotsRESTClient(ctx)
	if err != nil {
		return nil, err
	}

	computeInstantSnapshotCli, err := compute.NewInstantSnapshotsRESTClient(ctx)
	if err != nil {
		return nil, err
	}

	computeMachineImageCli, err := compute.NewMachineImagesRESTClient(ctx)
	if err != nil {
		return nil, err
	}

	backendServiceCli, err := compute.NewBackendServicesRESTClient(ctx)
	if err != nil {
		return nil, err
	}

	instanceGroupCli, err := compute.NewInstanceGroupsRESTClient(ctx)
	if err != nil {
		return nil, err
	}

	instanceGroupManagerCli, err := compute.NewInstanceGroupManagersRESTClient(ctx)
	if err != nil {
		return nil, err
	}

	diskCli, err := compute.NewDisksRESTClient(ctx)
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
			sources.WrapperToAdapter(NewComputeInstanceGroup(shared.NewComputeInstanceGroupsClient(instanceGroupCli), projectID, zone)),
			sources.WrapperToAdapter(NewComputeInstanceGroupManager(shared.NewComputeInstanceGroupManagerClient(instanceGroupManagerCli), projectID, zone)),
			sources.WrapperToAdapter(NewComputeReservation(shared.NewComputeReservationClient(computeReservationCli), projectID, zone)),
			sources.WrapperToAdapter(NewComputeInstantSnapshot(shared.NewComputeInstantSnapshotsClient(computeInstantSnapshotCli), projectID, zone)),
			sources.WrapperToAdapter(NewComputeDisk(shared.NewComputeDiskClient(diskCli), projectID, zone)),
		)
	}

	// global - project level - adapters
	adapters = append(adapters,
		sources.WrapperToAdapter(NewComputeBackendService(shared.NewComputeBackendServiceClient(backendServiceCli), projectID)),
		sources.WrapperToAdapter(NewComputeImage(shared.NewComputeImagesClient(computeImagesCli), projectID)),
		sources.WrapperToAdapter(NewComputeHealthCheck(shared.NewComputeHealthCheckClient(computeHealthCheckCli), projectID)),
		sources.WrapperToAdapter(NewComputeSecurityPolicy(shared.NewComputeSecurityPolicyClient(computeSecurityPolicyCli), projectID)),
		sources.WrapperToAdapter(NewComputeMachineImage(shared.NewComputeMachineImageClient(computeMachineImageCli), projectID)),
		sources.WrapperToAdapter(NewComputeSnapshot(shared.NewComputeSnapshotsClient(computeSnapshotCli), projectID)),
	)

	// Register the metadata for each adapter
	for _, adapter := range adapters {
		Metadata.Register(adapter.Metadata())
	}

	return adapters, nil
}
