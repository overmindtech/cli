package clients

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7"
)

//go:generate mockgen -destination=../shared/mocks/mock_compute_disk_access_private_endpoint_connection_client.go -package=mocks -source=compute-disk-access-private-endpoint-connection-client.go

// ComputeDiskAccessPrivateEndpointConnectionsPager is a type alias for the generic Pager interface with disk access private endpoint connection list response type.
type ComputeDiskAccessPrivateEndpointConnectionsPager = Pager[armcompute.DiskAccessesClientListPrivateEndpointConnectionsResponse]

// ComputeDiskAccessPrivateEndpointConnectionsClient is an interface for interacting with Azure disk access private endpoint connections.
type ComputeDiskAccessPrivateEndpointConnectionsClient interface {
	Get(ctx context.Context, resourceGroupName string, diskAccessName string, privateEndpointConnectionName string) (armcompute.DiskAccessesClientGetAPrivateEndpointConnectionResponse, error)
	NewListPrivateEndpointConnectionsPager(resourceGroupName string, diskAccessName string, options *armcompute.DiskAccessesClientListPrivateEndpointConnectionsOptions) ComputeDiskAccessPrivateEndpointConnectionsPager
}

type computeDiskAccessPrivateEndpointConnectionsClient struct {
	client *armcompute.DiskAccessesClient
}

func (c *computeDiskAccessPrivateEndpointConnectionsClient) Get(ctx context.Context, resourceGroupName string, diskAccessName string, privateEndpointConnectionName string) (armcompute.DiskAccessesClientGetAPrivateEndpointConnectionResponse, error) {
	return c.client.GetAPrivateEndpointConnection(ctx, resourceGroupName, diskAccessName, privateEndpointConnectionName, nil)
}

func (c *computeDiskAccessPrivateEndpointConnectionsClient) NewListPrivateEndpointConnectionsPager(resourceGroupName string, diskAccessName string, options *armcompute.DiskAccessesClientListPrivateEndpointConnectionsOptions) ComputeDiskAccessPrivateEndpointConnectionsPager {
	return c.client.NewListPrivateEndpointConnectionsPager(resourceGroupName, diskAccessName, options)
}

// NewComputeDiskAccessPrivateEndpointConnectionsClient creates a new ComputeDiskAccessPrivateEndpointConnectionsClient from the Azure SDK DiskAccesses client.
func NewComputeDiskAccessPrivateEndpointConnectionsClient(client *armcompute.DiskAccessesClient) ComputeDiskAccessPrivateEndpointConnectionsClient {
	return &computeDiskAccessPrivateEndpointConnectionsClient{client: client}
}
