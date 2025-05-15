package adapters

import (
	"context"
	"fmt"

	container "cloud.google.com/go/container/apiv1"
	"cloud.google.com/go/container/apiv1/containerpb"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	ContainerCluster = shared.NewItemType(gcpshared.GCP, gcpshared.Container, gcpshared.Cluster)

	ContainerClusterLookupByName = shared.NewItemTypeLookup("name", ContainerCluster)
)

type containerClusterWrapper struct {
	client *container.ClusterManagerClient

	*gcpshared.ZoneBase
}

// NewContainerClusterWrapper creates a new wrapper for GCP Container Clusters
func NewContainerClusterWrapper(client *container.ClusterManagerClient, projectID, location string) sources.ListableWrapper {
	return &containerClusterWrapper{
		client: client,
		ZoneBase: gcpshared.NewZoneBase(
			projectID,
			location,
			sdp.AdapterCategory_ADAPTER_CATEGORY_COMPUTE_APPLICATION,
			ContainerCluster),
	}
}

// TerraformMappings returns the Terraform mappings for the Container Cluster
func (c containerClusterWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "google_container_cluster.name",
		},
	}
}

// GetLookups returns the ItemTypeLookups for the Container Cluster
func (c containerClusterWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		ContainerClusterLookupByName,
	}
}

// Get retrieves a Container Cluster by its name and converts it to an sdp.Item
func (c containerClusterWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	req := &containerpb.GetClusterRequest{
		// Specified in the format `projects/*/locations/*/clusters/*`.
		Name: fmt.Sprintf("projects/%s/locations/%s/clusters/%s", c.ProjectID(), c.Zone(), queryParts[0]),
	}

	cluster, err := c.client.GetCluster(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = c.gcpContainerClusterToSDPItem(cluster)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

// List retrieves all Container Clusters and converts them to sdp.Items
func (c containerClusterWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	req := &containerpb.ListClustersRequest{
		// 	// Specified in the format `projects/*/locations/*`.
		Parent: fmt.Sprintf("projects/%s/locations/%s", c.ProjectID(), c.Zone()),
	}

	resp, err := c.client.ListClusters(ctx, req)
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	var items []*sdp.Item
	for _, cluster := range resp.GetClusters() {
		item, err := c.gcpContainerClusterToSDPItem(cluster)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, nil
}

func (c containerClusterWrapper) gcpContainerClusterToSDPItem(cluster *containerpb.Cluster) (*sdp.Item, *sdp.QueryError) {
	// Implement the logic to map a GCP Container Cluster to an sdp.Resource

	// TODO: Set a new unique attribute: simpleName as the last item of the actual name
	// name will be in the format of projects/{project}/locations/{location}/clusters/{cluster_name}
	// So, in this source scope, the {{cluster_name}} will be unique
	return nil, nil
}
