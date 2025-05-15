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

// Create a new NewComputeAutoscalerClient from a real GCP client.
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
