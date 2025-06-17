//go:generate mockgen -destination=./mocks/mock_big_query_dataset_client.go -package=mocks -source=big-query-clients.go -imports=sdp=github.com/overmindtech/cli/sdp-go
package shared

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/sdp-go"
)

type BigQueryDatasetClient interface {
	Get(ctx context.Context, projectID, datasetID string) (*bigquery.DatasetMetadata, error)
	List(ctx context.Context, projectID string, toSDPItem func(ctx context.Context, dataset *bigquery.DatasetMetadata) (*sdp.Item, *sdp.QueryError)) ([]*sdp.Item, *sdp.QueryError)
}

type bigQueryDatasetClient struct {
	client *bigquery.Client
}

func (b bigQueryDatasetClient) Get(ctx context.Context, projectID, datasetID string) (*bigquery.DatasetMetadata, error) {
	ds := b.client.DatasetInProject(projectID, datasetID)

	if ds == nil {
		return nil, fmt.Errorf("dataset %s not found in project %s", datasetID, projectID)
	}

	return ds.Metadata(ctx)
}

func (b bigQueryDatasetClient) List(ctx context.Context, projectID string, toSDPItem func(ctx context.Context, dataset *bigquery.DatasetMetadata) (*sdp.Item, *sdp.QueryError)) ([]*sdp.Item, *sdp.QueryError) {
	dsIterator := b.client.Datasets(ctx)
	if dsIterator == nil {
		return nil, QueryError(fmt.Errorf("failed to create dataset iterator for project %s", projectID))
	}

	dsIterator.ProjectID = projectID

	var items []*sdp.Item
	for {
		ds, err := dsIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, QueryError(fmt.Errorf("error iterating datasets: %w", err))
		}

		meta, err := ds.Metadata(ctx)
		if err != nil {
			return nil, QueryError(fmt.Errorf("error getting metadata for dataset %s: %w", ds.DatasetID, err))
		}

		var sdpErr *sdp.QueryError
		item, sdpErr := toSDPItem(ctx, meta)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func NewBigQueryDatasetClient(client *bigquery.Client) BigQueryDatasetClient {
	return &bigQueryDatasetClient{
		client: client,
	}
}
