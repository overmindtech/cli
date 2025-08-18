//go:generate mockgen -destination=./mocks/mock_compute_instance_client.go -package=mocks -source=compute-clients.go
package shared

import (
	"context"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	"github.com/googleapis/gax-go/v2"
)

// ComputeInstanceIterator is an interface for iterating over compute instances
type ComputeInstanceIterator interface {
	Next() (*computepb.Instance, error)
}

// ComputeInstanceClient is an interface for the Compute Instance client
type ComputeInstanceClient interface {
	Get(ctx context.Context, req *computepb.GetInstanceRequest, opts ...gax.CallOption) (*computepb.Instance, error)
	List(ctx context.Context, req *computepb.ListInstancesRequest, opts ...gax.CallOption) ComputeInstanceIterator
}

type computeInstanceClient struct {
	instanceClient *compute.InstancesClient
}

// NewComputeInstanceClient creates a new ComputeInstanceClient
func NewComputeInstanceClient(instanceClient *compute.InstancesClient) ComputeInstanceClient {
	return &computeInstanceClient{
		instanceClient: instanceClient,
	}
}

// Get retrieves a compute instance
func (c computeInstanceClient) Get(ctx context.Context, req *computepb.GetInstanceRequest, opts ...gax.CallOption) (*computepb.Instance, error) {
	return c.instanceClient.Get(ctx, req, opts...)
}

// List lists compute instances and returns an iterator
func (c computeInstanceClient) List(ctx context.Context, req *computepb.ListInstancesRequest, opts ...gax.CallOption) ComputeInstanceIterator {
	return c.instanceClient.List(ctx, req, opts...)
}

// ComputeAddressIterator is an interface for iterating over compute address
type ComputeAddressIterator interface {
	Next() (*computepb.Address, error)
}

// ComputeAddressClient is an interface for the Compute Engine Address client
type ComputeAddressClient interface {
	Get(ctx context.Context, req *computepb.GetAddressRequest, opts ...gax.CallOption) (*computepb.Address, error)
	List(ctx context.Context, req *computepb.ListAddressesRequest, opts ...gax.CallOption) ComputeAddressIterator
}

type computeAddressClient struct {
	addressClient *compute.AddressesClient
}

// NewComputeAddressClient creates a new ComputeAddressClient
func NewComputeAddressClient(addressClient *compute.AddressesClient) ComputeAddressClient {
	return &computeAddressClient{
		addressClient: addressClient,
	}
}

// Get retrieves a compute address
func (c computeAddressClient) Get(ctx context.Context, req *computepb.GetAddressRequest, opts ...gax.CallOption) (*computepb.Address, error) {
	return c.addressClient.Get(ctx, req, opts...)
}

// List lists compute address and returns an iterator
func (c computeAddressClient) List(ctx context.Context, req *computepb.ListAddressesRequest, opts ...gax.CallOption) ComputeAddressIterator {
	return c.addressClient.List(ctx, req, opts...)
}

// ComputeImageIterator is an interface for iterating over compute images
type ComputeImageIterator interface {
	Next() (*computepb.Image, error)
}

// ComputeImagesClient is an interface for the Compute Images client
type ComputeImagesClient interface {
	Get(ctx context.Context, req *computepb.GetImageRequest, opts ...gax.CallOption) (*computepb.Image, error)
	List(ctx context.Context, req *computepb.ListImagesRequest, opts ...gax.CallOption) ComputeImageIterator
}

type computeImagesClient struct {
	imageClient *compute.ImagesClient
}

// NewComputeImagesClient creates a new ComputeImagesClient
func NewComputeImagesClient(imageClient *compute.ImagesClient) ComputeImagesClient {
	return &computeImagesClient{
		imageClient: imageClient,
	}
}

// Get retrieves a compute image
func (c computeImagesClient) Get(ctx context.Context, req *computepb.GetImageRequest, opts ...gax.CallOption) (*computepb.Image, error) {
	return c.imageClient.Get(ctx, req, opts...)
}

// List lists compute images and returns an iterator
func (c computeImagesClient) List(ctx context.Context, req *computepb.ListImagesRequest, opts ...gax.CallOption) ComputeImageIterator {
	return c.imageClient.List(ctx, req, opts...)
}

// ComputeInstanceGroupManagerIterator is an interface for iterating over instance group managers
type ComputeInstanceGroupManagerIterator interface {
	Next() (*computepb.InstanceGroupManager, error)
}

// ComputeInstanceGroupManagerClient is an interface for the Compute Instance Group Manager client
type ComputeInstanceGroupManagerClient interface {
	Get(ctx context.Context, req *computepb.GetInstanceGroupManagerRequest, opts ...gax.CallOption) (*computepb.InstanceGroupManager, error)
	List(ctx context.Context, req *computepb.ListInstanceGroupManagersRequest, opts ...gax.CallOption) ComputeInstanceGroupManagerIterator
}

type computeInstanceGroupManagerClient struct {
	instanceGroupManagersClient *compute.InstanceGroupManagersClient
}

// NewComputeInstanceGroupManagerClient creates a new ComputeInstanceGroupManagerClient
func NewComputeInstanceGroupManagerClient(instanceGroupManagersClient *compute.InstanceGroupManagersClient) ComputeInstanceGroupManagerClient {
	return &computeInstanceGroupManagerClient{
		instanceGroupManagersClient: instanceGroupManagersClient,
	}
}

// Get retrieves a compute instance group manager
func (c computeInstanceGroupManagerClient) Get(ctx context.Context, req *computepb.GetInstanceGroupManagerRequest, opts ...gax.CallOption) (*computepb.InstanceGroupManager, error) {
	return c.instanceGroupManagersClient.Get(ctx, req, opts...)
}

// List lists compute instance group managers and returns an iterator
func (c computeInstanceGroupManagerClient) List(ctx context.Context, req *computepb.ListInstanceGroupManagersRequest, opts ...gax.CallOption) ComputeInstanceGroupManagerIterator {
	return c.instanceGroupManagersClient.List(ctx, req, opts...)
}

type ForwardingRuleIterator interface {
	Next() (*computepb.ForwardingRule, error)
}

// ComputeForwardingRuleClient is an interface for the Compute Engine Forwarding Rule client
type ComputeForwardingRuleClient interface {
	Get(ctx context.Context, req *computepb.GetForwardingRuleRequest, opts ...gax.CallOption) (*computepb.ForwardingRule, error)
	List(ctx context.Context, req *computepb.ListForwardingRulesRequest, opts ...gax.CallOption) ForwardingRuleIterator
}

type computeForwardingRuleClient struct {
	client *compute.ForwardingRulesClient
}

func (c computeForwardingRuleClient) Get(ctx context.Context, req *computepb.GetForwardingRuleRequest, opts ...gax.CallOption) (*computepb.ForwardingRule, error) {
	return c.client.Get(ctx, req, opts...)
}

func (c computeForwardingRuleClient) List(ctx context.Context, req *computepb.ListForwardingRulesRequest, opts ...gax.CallOption) ForwardingRuleIterator {
	return c.client.List(ctx, req, opts...)
}

// NewComputeForwardingRuleClient creates a new ComputeForwardingRuleClient
func NewComputeForwardingRuleClient(forwardingRuleClient *compute.ForwardingRulesClient) ComputeForwardingRuleClient {
	return &computeForwardingRuleClient{
		client: forwardingRuleClient,
	}
}

// Interface for interating over compute autoscalers.
type ComputeAutoscalerIterator interface {
	Next() (*computepb.Autoscaler, error)
}

// Interface for accessing compute autoscaler resources.
type ComputeAutoscalerClient interface {
	Get(ctx context.Context, req *computepb.GetAutoscalerRequest, opts ...gax.CallOption) (*computepb.Autoscaler, error)
	List(ctx context.Context, req *computepb.ListAutoscalersRequest, opts ...gax.CallOption) ComputeAutoscalerIterator
}

// Wrapper for a ComputeAutoscalerClient implementation.
type computeAutoscalerClient struct {
	autoscalerClient *compute.AutoscalersClient
}

// Create a ComputeAutoscalerClient from a real GCP client.
func NewComputeAutoscalerClient(autoscalerClient *compute.AutoscalersClient) ComputeAutoscalerClient {
	return &computeAutoscalerClient{
		autoscalerClient: autoscalerClient,
	}
}

func (c computeAutoscalerClient) Get(ctx context.Context, req *computepb.GetAutoscalerRequest, opts ...gax.CallOption) (*computepb.Autoscaler, error) {
	return c.autoscalerClient.Get(ctx, req, opts...)
}

func (c computeAutoscalerClient) List(ctx context.Context, req *computepb.ListAutoscalersRequest, opts ...gax.CallOption) ComputeAutoscalerIterator {
	return c.autoscalerClient.List(ctx, req, opts...)
}

// ComputeBackendServiceClient is an interface for the Compute Engine Backend Service client
type ComputeBackendServiceClient interface {
	Get(ctx context.Context, req *computepb.GetBackendServiceRequest, opts ...gax.CallOption) (*computepb.BackendService, error)
	List(ctx context.Context, req *computepb.ListBackendServicesRequest, opts ...gax.CallOption) ComputeBackendServiceIterator
}

type ComputeBackendServiceIterator interface {
	Next() (*computepb.BackendService, error)
}

type computeBackendServiceClient struct {
	client *compute.BackendServicesClient
}

func (c computeBackendServiceClient) Get(ctx context.Context, req *computepb.GetBackendServiceRequest, opts ...gax.CallOption) (*computepb.BackendService, error) {
	return c.client.Get(ctx, req, opts...)
}

func (c computeBackendServiceClient) List(ctx context.Context, req *computepb.ListBackendServicesRequest, opts ...gax.CallOption) ComputeBackendServiceIterator {
	return c.client.List(ctx, req, opts...)
}

// NewComputeBackendServiceClient creates a new ComputeBackendServiceClient
func NewComputeBackendServiceClient(backendServiceClient *compute.BackendServicesClient) ComputeBackendServiceClient {
	return &computeBackendServiceClient{
		client: backendServiceClient,
	}
}

// ComputeInstanceGroupIterator is an interface for iterating over compute instance groups
type ComputeInstanceGroupIterator interface {
	Next() (*computepb.InstanceGroup, error)
}

// ComputeInstanceGroupsClient is an interface for the Compute Engine Instance Groups client
type ComputeInstanceGroupsClient interface {
	Get(ctx context.Context, req *computepb.GetInstanceGroupRequest, opts ...gax.CallOption) (*computepb.InstanceGroup, error)
	List(ctx context.Context, req *computepb.ListInstanceGroupsRequest, opts ...gax.CallOption) ComputeInstanceGroupIterator
}

type computeInstanceGroupsClient struct {
	client *compute.InstanceGroupsClient
}

// NewComputeInstanceGroupsClient creates a new ComputeInstanceGroupsClient
func NewComputeInstanceGroupsClient(instanceGroupsClient *compute.InstanceGroupsClient) ComputeInstanceGroupsClient {
	return &computeInstanceGroupsClient{
		client: instanceGroupsClient,
	}
}

// Get retrieves a compute instance group
func (c computeInstanceGroupsClient) Get(ctx context.Context, req *computepb.GetInstanceGroupRequest, opts ...gax.CallOption) (*computepb.InstanceGroup, error) {
	return c.client.Get(ctx, req, opts...)
}

// List lists compute instance groups and returns an iterator
func (c computeInstanceGroupsClient) List(ctx context.Context, req *computepb.ListInstanceGroupsRequest, opts ...gax.CallOption) ComputeInstanceGroupIterator {
	return c.client.List(ctx, req, opts...)
}

// Interface for interating over compute node groups.
type ComputeNodeGroupIterator interface {
	Next() (*computepb.NodeGroup, error)
}

// Interface for accessing compute NodeGroup resources.
type ComputeNodeGroupClient interface {
	Get(ctx context.Context, req *computepb.GetNodeGroupRequest, opts ...gax.CallOption) (*computepb.NodeGroup, error)
	List(ctx context.Context, req *computepb.ListNodeGroupsRequest, opts ...gax.CallOption) ComputeNodeGroupIterator
}

// Wrapper for a ComputeNodeGroupClient implementation.
type computeNodeGroupClient struct {
	nodeGroupClient *compute.NodeGroupsClient
}

// Create a ComputeNodeGroupClient from a real GCP client.
func NewComputeNodeGroupClient(NodeGroupClient *compute.NodeGroupsClient) ComputeNodeGroupClient {
	return &computeNodeGroupClient{
		nodeGroupClient: NodeGroupClient,
	}
}

func (c computeNodeGroupClient) Get(ctx context.Context, req *computepb.GetNodeGroupRequest, opts ...gax.CallOption) (*computepb.NodeGroup, error) {
	return c.nodeGroupClient.Get(ctx, req, opts...)
}

func (c computeNodeGroupClient) List(ctx context.Context, req *computepb.ListNodeGroupsRequest, opts ...gax.CallOption) ComputeNodeGroupIterator {
	return c.nodeGroupClient.List(ctx, req, opts...)
}

// ComputeHealthCheckIterator is an interface for iterating over compute health checks
type ComputeHealthCheckIterator interface {
	Next() (*computepb.HealthCheck, error)
}

// ComputeHealthCheckClient is an interface for the Compute Engine Health Checks client
type ComputeHealthCheckClient interface {
	Get(ctx context.Context, req *computepb.GetHealthCheckRequest, opts ...gax.CallOption) (*computepb.HealthCheck, error)
	List(ctx context.Context, req *computepb.ListHealthChecksRequest, opts ...gax.CallOption) ComputeHealthCheckIterator
}

type computeHealthCheckClient struct {
	client *compute.HealthChecksClient
}

// NewComputeHealthCheckClient creates a new ComputeHealthCheckClient
func NewComputeHealthCheckClient(healthChecksClient *compute.HealthChecksClient) ComputeHealthCheckClient {
	return &computeHealthCheckClient{
		client: healthChecksClient,
	}
}

// Get retrieves a compute health check
func (c computeHealthCheckClient) Get(ctx context.Context, req *computepb.GetHealthCheckRequest, opts ...gax.CallOption) (*computepb.HealthCheck, error) {
	return c.client.Get(ctx, req, opts...)
}

// List lists compute health checks and returns an iterator
func (c computeHealthCheckClient) List(ctx context.Context, req *computepb.ListHealthChecksRequest, opts ...gax.CallOption) ComputeHealthCheckIterator {
	return c.client.List(ctx, req, opts...)
}

// Interface for interating over compute node groups.
type ComputeNodeTemplateIterator interface {
	Next() (*computepb.NodeTemplate, error)
}

// Interface for accessing compute NodeTemplate resources.
type ComputeNodeTemplateClient interface {
	Get(ctx context.Context, req *computepb.GetNodeTemplateRequest, opts ...gax.CallOption) (*computepb.NodeTemplate, error)
	List(ctx context.Context, req *computepb.ListNodeTemplatesRequest, opts ...gax.CallOption) ComputeNodeTemplateIterator
}

// Wrapper for a ComputeNodeTemplateClient implementation.
type computeNodeTemplateClient struct {
	nodeTemplateClient *compute.NodeTemplatesClient
}

// Create a ComputeNodeTemplateClient from a real GCP client.
func NewComputeNodeTemplateClient(NodeTemplateClient *compute.NodeTemplatesClient) ComputeNodeTemplateClient {
	return &computeNodeTemplateClient{
		nodeTemplateClient: NodeTemplateClient,
	}
}

func (c computeNodeTemplateClient) Get(ctx context.Context, req *computepb.GetNodeTemplateRequest, opts ...gax.CallOption) (*computepb.NodeTemplate, error) {
	return c.nodeTemplateClient.Get(ctx, req, opts...)
}

func (c computeNodeTemplateClient) List(ctx context.Context, req *computepb.ListNodeTemplatesRequest, opts ...gax.CallOption) ComputeNodeTemplateIterator {
	return c.nodeTemplateClient.List(ctx, req, opts...)
}

// ComputeReservationIterator is an interface for iterating over compute reservations
type ComputeReservationIterator interface {
	Next() (*computepb.Reservation, error)
}

// ComputeReservationClient is an interface for the Compute Engine Reservations client
type ComputeReservationClient interface {
	Get(ctx context.Context, req *computepb.GetReservationRequest, opts ...gax.CallOption) (*computepb.Reservation, error)
	List(ctx context.Context, req *computepb.ListReservationsRequest, opts ...gax.CallOption) ComputeReservationIterator
}

type computeReservationClient struct {
	client *compute.ReservationsClient
}

// NewComputeReservationClient creates a new ComputeReservationClient
func NewComputeReservationClient(reservationsClient *compute.ReservationsClient) ComputeReservationClient {
	return &computeReservationClient{
		client: reservationsClient,
	}
}

// Get retrieves a compute reservation
func (c computeReservationClient) Get(ctx context.Context, req *computepb.GetReservationRequest, opts ...gax.CallOption) (*computepb.Reservation, error) {
	return c.client.Get(ctx, req, opts...)
}

// List lists compute reservations and returns an iterator
func (c computeReservationClient) List(ctx context.Context, req *computepb.ListReservationsRequest, opts ...gax.CallOption) ComputeReservationIterator {
	return c.client.List(ctx, req, opts...)
}

// ComputeSecurityPolicyIterator is an interface for iterating over compute security policies
type ComputeSecurityPolicyIterator interface {
	Next() (*computepb.SecurityPolicy, error)
}

// ComputeSecurityPolicyClient is an interface for the Compute Security Policies client
type ComputeSecurityPolicyClient interface {
	Get(ctx context.Context, req *computepb.GetSecurityPolicyRequest, opts ...gax.CallOption) (*computepb.SecurityPolicy, error)
	List(ctx context.Context, req *computepb.ListSecurityPoliciesRequest, opts ...gax.CallOption) ComputeSecurityPolicyIterator
}

type computeSecurityPolicyClient struct {
	client *compute.SecurityPoliciesClient
}

// NewComputeSecurityPolicyClient creates a new ComputeSecurityPolicyClient
func NewComputeSecurityPolicyClient(securityPolicyClient *compute.SecurityPoliciesClient) ComputeSecurityPolicyClient {
	return &computeSecurityPolicyClient{
		client: securityPolicyClient,
	}
}

// Get retrieves a compute security policy
func (c computeSecurityPolicyClient) Get(ctx context.Context, req *computepb.GetSecurityPolicyRequest, opts ...gax.CallOption) (*computepb.SecurityPolicy, error) {
	return c.client.Get(ctx, req, opts...)
}

// List lists compute security policies and returns an iterator
func (c computeSecurityPolicyClient) List(ctx context.Context, req *computepb.ListSecurityPoliciesRequest, opts ...gax.CallOption) ComputeSecurityPolicyIterator {
	return c.client.List(ctx, req, opts...)
}

// ComputeInstantSnapshotIterator is an interface for iterating over compute instant snapshots
type ComputeInstantSnapshotIterator interface {
	Next() (*computepb.InstantSnapshot, error)
}

// ComputeInstantSnapshotsClient is an interface for the Compute Instant Snapshots client
type ComputeInstantSnapshotsClient interface {
	Get(ctx context.Context, req *computepb.GetInstantSnapshotRequest, opts ...gax.CallOption) (*computepb.InstantSnapshot, error)
	List(ctx context.Context, req *computepb.ListInstantSnapshotsRequest, opts ...gax.CallOption) ComputeInstantSnapshotIterator
}

type computeInstantSnapshotsClient struct {
	client *compute.InstantSnapshotsClient
}

// NewComputeInstantSnapshotsClient creates a new ComputeInstantSnapshotsClient
func NewComputeInstantSnapshotsClient(instantSnapshotsClient *compute.InstantSnapshotsClient) ComputeInstantSnapshotsClient {
	return &computeInstantSnapshotsClient{
		client: instantSnapshotsClient,
	}
}

// Get retrieves a compute instant snapshot
func (c computeInstantSnapshotsClient) Get(ctx context.Context, req *computepb.GetInstantSnapshotRequest, opts ...gax.CallOption) (*computepb.InstantSnapshot, error) {
	return c.client.Get(ctx, req, opts...)
}

// List lists compute instant snapshots and returns an iterator
func (c computeInstantSnapshotsClient) List(ctx context.Context, req *computepb.ListInstantSnapshotsRequest, opts ...gax.CallOption) ComputeInstantSnapshotIterator {
	return c.client.List(ctx, req, opts...)
}

// ComputeDiskIterator is an interface for iterating over compute disks
type ComputeDiskIterator interface {
	Next() (*computepb.Disk, error)
}

// ComputeDiskClient is an interface for the Compute Engine Disk client
type ComputeDiskClient interface {
	Get(ctx context.Context, req *computepb.GetDiskRequest, opts ...gax.CallOption) (*computepb.Disk, error)
	List(ctx context.Context, req *computepb.ListDisksRequest, opts ...gax.CallOption) ComputeDiskIterator
}

type computeDiskClient struct {
	client *compute.DisksClient
}

// NewComputeDiskClient creates a new ComputeDiskClient
func NewComputeDiskClient(client *compute.DisksClient) ComputeDiskClient {
	return &computeDiskClient{
		client: client,
	}
}

// Get retrieves a compute disk
func (c computeDiskClient) Get(ctx context.Context, req *computepb.GetDiskRequest, opts ...gax.CallOption) (*computepb.Disk, error) {
	return c.client.Get(ctx, req, opts...)
}

// List lists compute disks and returns an iterator
func (c computeDiskClient) List(ctx context.Context, req *computepb.ListDisksRequest, opts ...gax.CallOption) ComputeDiskIterator {
	return c.client.List(ctx, req, opts...)
}

// ComputeMachineImageIterator is an interface for iterating over compute machine images
type ComputeMachineImageIterator interface {
	Next() (*computepb.MachineImage, error)
}

// ComputeMachineImageClient is an interface for the Compute Engine Machine Images client
type ComputeMachineImageClient interface {
	Get(ctx context.Context, req *computepb.GetMachineImageRequest, opts ...gax.CallOption) (*computepb.MachineImage, error)
	List(ctx context.Context, req *computepb.ListMachineImagesRequest, opts ...gax.CallOption) ComputeMachineImageIterator
}

type computeMachineImageClient struct {
	client *compute.MachineImagesClient
}

// NewComputeMachineImageClient creates a new ComputeMachineImageClient
func NewComputeMachineImageClient(machineImageClient *compute.MachineImagesClient) ComputeMachineImageClient {
	return &computeMachineImageClient{
		client: machineImageClient,
	}
}

// Get retrieves a compute machine image
func (c computeMachineImageClient) Get(ctx context.Context, req *computepb.GetMachineImageRequest, opts ...gax.CallOption) (*computepb.MachineImage, error) {
	return c.client.Get(ctx, req, opts...)
}

// List lists compute machine images and returns an iterator
func (c computeMachineImageClient) List(ctx context.Context, req *computepb.ListMachineImagesRequest, opts ...gax.CallOption) ComputeMachineImageIterator {
	return c.client.List(ctx, req, opts...)
}

// ComputeSnapshotIterator is an interface for iterating over compute snapshots
type ComputeSnapshotIterator interface {
	Next() (*computepb.Snapshot, error)
}

// ComputeSnapshotsClient is an interface for the Compute Snapshots client
type ComputeSnapshotsClient interface {
	Get(ctx context.Context, req *computepb.GetSnapshotRequest, opts ...gax.CallOption) (*computepb.Snapshot, error)
	List(ctx context.Context, req *computepb.ListSnapshotsRequest, opts ...gax.CallOption) ComputeSnapshotIterator
}

type computeSnapshotsClient struct {
	snapshotClient *compute.SnapshotsClient
}

// NewComputeSnapshotsClient creates a new ComputeSnapshotsClient
func NewComputeSnapshotsClient(snapshotClient *compute.SnapshotsClient) ComputeSnapshotsClient {
	return &computeSnapshotsClient{
		snapshotClient: snapshotClient,
	}
}

// Get retrieves a compute snapshot
func (c computeSnapshotsClient) Get(ctx context.Context, req *computepb.GetSnapshotRequest, opts ...gax.CallOption) (*computepb.Snapshot, error) {
	return c.snapshotClient.Get(ctx, req, opts...)
}

// List lists compute snapshots and returns an iterator
func (c computeSnapshotsClient) List(ctx context.Context, req *computepb.ListSnapshotsRequest, opts ...gax.CallOption) ComputeSnapshotIterator {
	return c.snapshotClient.List(ctx, req, opts...)
}

// ComputeRegionBackendServiceIterator is an interface for iterating over compute region backend services
type ComputeRegionBackendServiceIterator interface {
	Next() (*computepb.BackendService, error)
}

// ComputeRegionBackendServiceClient is an interface for the Compute Engine Region Backend Service client
type ComputeRegionBackendServiceClient interface {
	Get(ctx context.Context, req *computepb.GetRegionBackendServiceRequest, opts ...gax.CallOption) (*computepb.BackendService, error)
	List(ctx context.Context, req *computepb.ListRegionBackendServicesRequest, opts ...gax.CallOption) ComputeRegionBackendServiceIterator
}

type computeRegionBackendServiceClient struct {
	client *compute.RegionBackendServicesClient
}

func (c computeRegionBackendServiceClient) Get(ctx context.Context, req *computepb.GetRegionBackendServiceRequest, opts ...gax.CallOption) (*computepb.BackendService, error) {
	return c.client.Get(ctx, req, opts...)
}

func (c computeRegionBackendServiceClient) List(ctx context.Context, req *computepb.ListRegionBackendServicesRequest, opts ...gax.CallOption) ComputeRegionBackendServiceIterator {
	return c.client.List(ctx, req, opts...)
}

// NewComputeRegionBackendServiceClient creates a new ComputeRegionBackendServiceClient
func NewComputeRegionBackendServiceClient(regionBackendServiceClient *compute.RegionBackendServicesClient) ComputeRegionBackendServiceClient {
	return &computeRegionBackendServiceClient{
		client: regionBackendServiceClient,
	}
}
