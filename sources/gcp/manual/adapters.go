package manual

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	compute "cloud.google.com/go/compute/apiv1"
	iamAdmin "cloud.google.com/go/iam/admin/apiv1"
	kms "cloud.google.com/go/kms/apiv1"
	logging "cloud.google.com/go/logging/apiv2"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/gcp/shared"
)

// Adapters returns a slice of discovery.Adapter instances for GCP Source.
// It initializes GCP clients if initGCPClients is true, and creates adapters for the specified locations.
// Otherwise, it uses nil clients, which is useful for enumerating adapters for documentation purposes.
func Adapters(ctx context.Context, projectLocations, regionLocations, zoneLocations []shared.LocationInfo, tokenSource *oauth2.TokenSource, initGCPClients bool, cache sdpcache.Cache) ([]discovery.Adapter, error) {
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
		iamServiceAccountKeyCli   *iamAdmin.IamClient
		iamServiceAccountCli      *iamAdmin.IamClient
		kmsKeyRingCli             *kms.KeyManagementClient
		kmsCryptoKeyCli           *kms.KeyManagementClient
		kmsCryptoKeyVersionCli    *kms.KeyManagementClient
		bigQueryDatasetCli        *bigquery.Client
		loggingConfigCli          *logging.ConfigClient
		nodeGroupCli              *compute.NodeGroupsClient
		nodeTemplateCli           *compute.NodeTemplatesClient
		regionBackendServiceCli   *compute.RegionBackendServicesClient
		regionHealthCheckCli      *compute.RegionHealthChecksClient
	)

	if initGCPClients {
		opts := []option.ClientOption{}
		if tokenSource != nil {
			opts = append(opts, option.WithTokenSource(*tokenSource))
		}

		instanceCli, err = compute.NewInstancesRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute instances client: %w", err)
		}

		addressCli, err = compute.NewAddressesRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute addresses client: %w", err)
		}

		autoscalerCli, err = compute.NewAutoscalersRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute autoscalers client: %w", err)
		}

		computeImagesCli, err = compute.NewImagesRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute images client: %w", err)
		}

		computeForwardingCli, err = compute.NewForwardingRulesRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute forwarding rules client: %w", err)
		}

		computeHealthCheckCli, err = compute.NewHealthChecksRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute health checks client: %w", err)
		}

		computeReservationCli, err = compute.NewReservationsRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute reservations client: %w", err)
		}

		computeSecurityPolicyCli, err = compute.NewSecurityPoliciesRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute security policies client: %w", err)
		}

		computeSnapshotCli, err = compute.NewSnapshotsRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute snapshots client: %w", err)
		}

		computeInstantSnapshotCli, err = compute.NewInstantSnapshotsRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute instant snapshots client: %w", err)
		}

		computeMachineImageCli, err = compute.NewMachineImagesRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute machine images client: %w", err)
		}

		backendServiceCli, err = compute.NewBackendServicesRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute backend services client: %w", err)
		}

		instanceGroupCli, err = compute.NewInstanceGroupsRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute instance groups client: %w", err)
		}

		instanceGroupManagerCli, err = compute.NewInstanceGroupManagersRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute instance group managers client: %w", err)
		}

		diskCli, err = compute.NewDisksRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute disks client: %w", err)
		}

		// IAM
		iamServiceAccountKeyCli, err = iamAdmin.NewIamClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create IAM service account key client: %w", err)
		}

		iamServiceAccountCli, err = iamAdmin.NewIamClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create IAM service account client: %w", err)
		}

		// KMS
		kmsKeyRingCli, err = kms.NewKeyManagementClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create KMS key ring client: %w", err)
		}

		kmsCryptoKeyCli, err = kms.NewKeyManagementClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create KMS crypto key client: %w", err)
		}

		kmsCryptoKeyVersionCli, err = kms.NewKeyManagementClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create KMS crypto key version client: %w", err)
		}

		// Extract project ID from projectLocations for BigQuery client initialization.
		//
		// IMPORTANT: The project ID passed to bigquery.NewClient() is used for:
		// 1. Billing - all BigQuery operations are billed to this project
		// 2. Client initialization - required parameter, cannot be omitted
		//
		// This does NOT restrict which projects we can query. All actual API operations
		// in our codebase explicitly specify the target project using:
		// - DatasetInProject(projectID, datasetID) for Get operations
		// - dsIterator.ProjectID = projectID for List operations
		//
		// Therefore, using the first project ID here allows the adapter to query
		// resources across ALL configured projects. The only consideration is billing:
		// if projects have separate billing accounts, operations will be billed to
		// the first project. If all projects share billing, this doesn't matter.
		//
		// We use the first project ID rather than bigquery.DetectProjectID because:
		// - Auto-detection fails in containerized/Kubernetes environments
		// - We have explicit project IDs available in projectLocations
		// - Explicit configuration is more reliable than environment detection
		var bigQueryProjectID string
		for _, loc := range projectLocations {
			if loc.ProjectID != "" {
				bigQueryProjectID = loc.ProjectID
				break
			}
		}
		if bigQueryProjectID == "" {
			return nil, fmt.Errorf("at least one project location with a valid project ID is required to create BigQuery client")
		}

		bigQueryDatasetCli, err = bigquery.NewClient(ctx, bigQueryProjectID, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create bigquery client: %w", err)
		}

		loggingConfigCli, err = logging.NewConfigClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create logging config client: %w", err)
		}

		nodeGroupCli, err = compute.NewNodeGroupsRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute node groups client: %w", err)
		}

		nodeTemplateCli, err = compute.NewNodeTemplatesRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute node templates client: %w", err)
		}

		regionBackendServiceCli, err = compute.NewRegionBackendServicesRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute region backend services client: %w", err)
		}

		regionHealthCheckCli, err = compute.NewRegionHealthChecksRESTClient(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create compute region health checks client: %w", err)
		}
	}

	var adapters []discovery.Adapter

	// Multi-scope regional adapters (one adapter per type handling all regions)
	if len(regionLocations) > 0 {
		adapters = append(adapters,
			sources.WrapperToAdapter(NewComputeAddress(shared.NewComputeAddressClient(addressCli), regionLocations), cache),
			sources.WrapperToAdapter(NewComputeForwardingRule(shared.NewComputeForwardingRuleClient(computeForwardingCli), regionLocations), cache),
			sources.WrapperToAdapter(NewComputeNodeTemplate(shared.NewComputeNodeTemplateClient(nodeTemplateCli), regionLocations), cache),
		)
	}

	// Multi-scope zonal adapters (one adapter per type handling all zones)
	if len(zoneLocations) > 0 {
		adapters = append(adapters,
			sources.WrapperToAdapter(NewComputeInstance(shared.NewComputeInstanceClient(instanceCli), zoneLocations), cache),
			sources.WrapperToAdapter(NewComputeAutoscaler(shared.NewComputeAutoscalerClient(autoscalerCli), zoneLocations), cache),
			sources.WrapperToAdapter(NewComputeInstanceGroup(shared.NewComputeInstanceGroupsClient(instanceGroupCli), zoneLocations), cache),
			sources.WrapperToAdapter(NewComputeInstanceGroupManager(shared.NewComputeInstanceGroupManagerClient(instanceGroupManagerCli), zoneLocations), cache),
			sources.WrapperToAdapter(NewComputeReservation(shared.NewComputeReservationClient(computeReservationCli), zoneLocations), cache),
			sources.WrapperToAdapter(NewComputeInstantSnapshot(shared.NewComputeInstantSnapshotsClient(computeInstantSnapshotCli), zoneLocations), cache),
			sources.WrapperToAdapter(NewComputeDisk(shared.NewComputeDiskClient(diskCli), zoneLocations), cache),
			sources.WrapperToAdapter(NewComputeNodeGroup(shared.NewComputeNodeGroupClient(nodeGroupCli), zoneLocations), cache),
		)
	}

	// Dual-scope adapters (handle both global and regional)
	if len(projectLocations) > 0 || len(regionLocations) > 0 {
		adapters = append(adapters,
			sources.WrapperToAdapter(
				NewComputeBackendService(
					shared.NewComputeBackendServiceClient(backendServiceCli),
					shared.NewComputeRegionBackendServiceClient(regionBackendServiceCli),
					projectLocations,
					regionLocations,
				),
				cache,
			),
			sources.WrapperToAdapter(
				NewComputeHealthCheck(
					shared.NewComputeHealthCheckClient(computeHealthCheckCli),
					shared.NewComputeRegionHealthCheckClient(regionHealthCheckCli),
					projectLocations,
					regionLocations,
				),
				cache,
			),
		)
	}

	// global - project level - adapters
	if len(projectLocations) > 0 {
		adapters = append(adapters,
			sources.WrapperToAdapter(NewComputeImage(shared.NewComputeImagesClient(computeImagesCli), projectLocations), cache),
			sources.WrapperToAdapter(NewComputeSecurityPolicy(shared.NewComputeSecurityPolicyClient(computeSecurityPolicyCli), projectLocations), cache),
			sources.WrapperToAdapter(NewComputeMachineImage(shared.NewComputeMachineImageClient(computeMachineImageCli), projectLocations), cache),
			sources.WrapperToAdapter(NewComputeSnapshot(shared.NewComputeSnapshotsClient(computeSnapshotCli), projectLocations), cache),
			sources.WrapperToAdapter(NewIAMServiceAccountKey(shared.NewIAMServiceAccountKeyClient(iamServiceAccountKeyCli), projectLocations), cache),
			sources.WrapperToAdapter(NewIAMServiceAccount(shared.NewIAMServiceAccountClient(iamServiceAccountCli), projectLocations), cache),
			sources.WrapperToAdapter(NewCloudKMSKeyRing(shared.NewCloudKMSKeyRingClient(kmsKeyRingCli), projectLocations), cache),
			sources.WrapperToAdapter(NewCloudKMSCryptoKey(shared.NewCloudKMSCryptoKeyClient(kmsCryptoKeyCli), projectLocations), cache),
			sources.WrapperToAdapter(NewCloudKMSCryptoKeyVersion(shared.NewCloudKMSCryptoKeyVersionClient(kmsCryptoKeyVersionCli), projectLocations), cache),
			sources.WrapperToAdapter(NewBigQueryDataset(shared.NewBigQueryDatasetClient(bigQueryDatasetCli), projectLocations), cache),
			sources.WrapperToAdapter(NewLoggingSink(shared.NewLoggingConfigClient(loggingConfigCli), projectLocations), cache),
			sources.WrapperToAdapter(NewBigQueryRoutine(shared.NewBigQueryRoutineClient(bigQueryDatasetCli), projectLocations), cache),
		)
	}

	return adapters, nil
}
