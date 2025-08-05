package manual

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	compute "cloud.google.com/go/compute/apiv1"
	iam "cloud.google.com/go/iam/admin/apiv1"
	kms "cloud.google.com/go/kms/apiv1"
	logging "cloud.google.com/go/logging/apiv2"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/shared"
)

// Adapters returns a slice of discovery.Adapter instances for GCP Source.
// It initializes GCP clients if initGCPClients is true, and creates adapters for the specified project ID, regions, and zones.
// Otherwise, it uses nil clients, which is useful for enumerating adapters for documentation purposes.
func Adapters(ctx context.Context, projectID string, regions []string, zones []string, initGCPClients bool) ([]discovery.Adapter, error) {
	var err error
	var (
		instanceCli               *compute.InstancesClient
		addressCli                *compute.AddressesClient
		autoscalerCli             *compute.AutoscalersClient
		computeImagesCli          *compute.ImagesClient
		computeForwardingCli      *compute.ForwardingRulesClient
		computeHealthCheckCli     *compute.HealthChecksClient
		computeReservationCli     *compute.ReservationsClient
		computeSecurityPolicyCli  *compute.SecurityPoliciesClient
		computeSnapshotCli        *compute.SnapshotsClient
		computeInstantSnapshotCli *compute.InstantSnapshotsClient
		computeMachineImageCli    *compute.MachineImagesClient
		backendServiceCli         *compute.BackendServicesClient
		instanceGroupCli          *compute.InstanceGroupsClient
		instanceGroupManagerCli   *compute.InstanceGroupManagersClient
		diskCli                   *compute.DisksClient
		iamServiceAccountKeyCli   *iam.IamClient
		iamServiceAccountCli      *iam.IamClient
		kmsKeyRingCli             *kms.KeyManagementClient
		kmsCryptoKeyCli           *kms.KeyManagementClient
		bigQueryDatasetCli        *bigquery.Client
		loggingConfigCli          *logging.ConfigClient
		nodeGroupCli              *compute.NodeGroupsClient
	)

	if initGCPClients {
		instanceCli, err = compute.NewInstancesRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute instances client: %w", err)
		}

		addressCli, err = compute.NewAddressesRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute addresses client: %w", err)
		}

		autoscalerCli, err = compute.NewAutoscalersRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute autoscalers client: %w", err)
		}

		computeImagesCli, err = compute.NewImagesRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute images client: %w", err)
		}

		computeForwardingCli, err = compute.NewForwardingRulesRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute forwarding rules client: %w", err)
		}

		computeHealthCheckCli, err = compute.NewHealthChecksRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute health checks client: %w", err)
		}

		computeReservationCli, err = compute.NewReservationsRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute reservations client: %w", err)
		}

		computeSecurityPolicyCli, err = compute.NewSecurityPoliciesRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute security policies client: %w", err)
		}

		computeSnapshotCli, err = compute.NewSnapshotsRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute snapshots client: %w", err)
		}

		computeInstantSnapshotCli, err = compute.NewInstantSnapshotsRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute instant snapshots client: %w", err)
		}

		computeMachineImageCli, err = compute.NewMachineImagesRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute machine images client: %w", err)
		}

		backendServiceCli, err = compute.NewBackendServicesRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute backend services client: %w", err)
		}

		instanceGroupCli, err = compute.NewInstanceGroupsRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute instance groups client: %w", err)
		}

		instanceGroupManagerCli, err = compute.NewInstanceGroupManagersRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute instance group managers client: %w", err)
		}

		diskCli, err = compute.NewDisksRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute disks client: %w", err)
		}

		//IAM
		iamServiceAccountKeyCli, err = iam.NewIamClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create IAM service account key client: %w", err)
		}

		iamServiceAccountCli, err = iam.NewIamClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create IAM service account client: %w", err)
		}

		//KMS
		kmsKeyRingCli, err = kms.NewKeyManagementClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create KMS key ring client: %w", err)
		}

		kmsCryptoKeyCli, err = kms.NewKeyManagementClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create KMS crypto key client: %w", err)
		}

		bigQueryDatasetCli, err = bigquery.NewClient(ctx, projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to create bigquery client: %w", err)
		}

		loggingConfigCli, err = logging.NewConfigClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create logging config client: %w", err)
		}

		nodeGroupCli, err = compute.NewNodeGroupsRESTClient(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute node groups client: %w", err)
		}
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
			sources.WrapperToAdapter(NewComputeNodeGroup(shared.NewComputeNodeGroupClient(nodeGroupCli), projectID, zone)),
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
		sources.WrapperToAdapter(NewIAMServiceAccountKey(shared.NewIAMServiceAccountKeyClient(iamServiceAccountKeyCli), projectID)),
		sources.WrapperToAdapter(NewIAMServiceAccount(shared.NewIAMServiceAccountClient(iamServiceAccountCli), projectID)),
		sources.WrapperToAdapter(NewCloudKMSKeyRing(shared.NewCloudKMSKeyRingClient(kmsKeyRingCli), projectID)),
		sources.WrapperToAdapter(NewCloudKMSCryptoKey(shared.NewCloudKMSCryptoKeyClient(kmsCryptoKeyCli), projectID)),
		sources.WrapperToAdapter(NewBigQueryDataset(shared.NewBigQueryDatasetClient(bigQueryDatasetCli), projectID)),
		sources.WrapperToAdapter(NewBigQueryTable(shared.NewBigQueryTableClient(bigQueryDatasetCli), projectID)),
		sources.WrapperToAdapter(NewLoggingSink(shared.NewLoggingConfigClient(loggingConfigCli), projectID)),
	)

	return adapters, nil
}
