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

type BigQueryTableClient interface {
	Get(ctx context.Context, projectID, datasetID, tableID string) (*bigquery.TableMetadata, error)
	List(ctx context.Context, projectID, datasetID string, toSDPItem func(table *bigquery.TableMetadata) (*sdp.Item, *sdp.QueryError)) ([]*sdp.Item, *sdp.QueryError)
}

type bigQueryTableClient struct {
	client *bigquery.Client
}

func (b bigQueryTableClient) Get(ctx context.Context, projectID, datasetID, tableID string) (*bigquery.TableMetadata, error) {
	ds := b.client.DatasetInProject(projectID, datasetID)
	if ds == nil {
		return nil, fmt.Errorf("dataset %s not found in project %s", datasetID, projectID)
	}

	table := ds.Table(tableID)
	if table == nil {
		return nil, fmt.Errorf("table %s not found in dataset %s in project %s", tableID, datasetID, projectID)
	}

	return table.Metadata(ctx)
}

func (b bigQueryTableClient) List(ctx context.Context, projectID, datasetID string, toSDPItem func(table *bigquery.TableMetadata) (*sdp.Item, *sdp.QueryError)) ([]*sdp.Item, *sdp.QueryError) {
	ds := b.client.DatasetInProject(projectID, datasetID)
	if ds == nil {
		return nil, QueryError(fmt.Errorf("dataset %s not found in project %s", datasetID, projectID))
	}

	tableIterator := ds.Tables(ctx)
	if tableIterator == nil {
		return nil, QueryError(fmt.Errorf("failed to create table iterator for dataset %s in project %s", datasetID, projectID))
	}

	var items []*sdp.Item
	for {
		table, err := tableIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, QueryError(fmt.Errorf("error iterating tables: %w", err))
		}

		meta, err := table.Metadata(ctx)
		if err != nil {
			return nil, QueryError(fmt.Errorf("error getting metadata for table %s: %w", table.TableID, err))
		}

		var sdpErr *sdp.QueryError
		item, sdpErr := toSDPItem(meta)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func NewBigQueryTableClient(client *bigquery.Client) BigQueryTableClient {
	return &bigQueryTableClient{
		client: client,
	}
}
