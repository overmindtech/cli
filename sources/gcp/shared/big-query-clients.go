package shared

import (
	"context"
	"errors"
	"fmt"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
)

//go:generate mockgen -destination=./mocks/mock_big_query_dataset_client.go -package=mocks -source=big-query-clients.go -imports=sdp=github.com/overmindtech/cli/sdp-go
type BigQueryDatasetClient interface {
	Get(ctx context.Context, projectID, datasetID string) (*bigquery.DatasetMetadata, error)
	List(ctx context.Context, projectID string, toSDPItem func(ctx context.Context, dataset *bigquery.DatasetMetadata) (*sdp.Item, *sdp.QueryError)) ([]*sdp.Item, *sdp.QueryError)
	ListStream(ctx context.Context, projectID string, stream discovery.QueryResultStream, toSDPItem func(ctx context.Context, dataset *bigquery.DatasetMetadata) (*sdp.Item, *sdp.QueryError))
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
		return nil, QueryError(fmt.Errorf("failed to create dataset iterator for project %s", projectID), projectID, BigQueryDataset.String())
	}

	dsIterator.ProjectID = projectID

	var items []*sdp.Item
	for {
		ds, err := dsIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, QueryError(fmt.Errorf("error iterating datasets: %w", err), projectID, BigQueryDataset.String())
		}

		meta, err := ds.Metadata(ctx)
		if err != nil {
			return nil, QueryError(fmt.Errorf("error getting metadata for dataset %s: %w", ds.DatasetID, err), projectID, BigQueryDataset.String())
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

func (b bigQueryDatasetClient) ListStream(ctx context.Context, projectID string, stream discovery.QueryResultStream, toSDPItem func(ctx context.Context, dataset *bigquery.DatasetMetadata) (*sdp.Item, *sdp.QueryError)) {
	dsIterator := b.client.Datasets(ctx)
	if dsIterator == nil {
		stream.SendError(QueryError(fmt.Errorf("failed to create dataset iterator for project %s", projectID), projectID, BigQueryDataset.String()))
		return
	}

	dsIterator.ProjectID = projectID

	for {
		ds, err := dsIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(QueryError(fmt.Errorf("error iterating datasets: %w", err), projectID, BigQueryDataset.String()))
			return
		}

		meta, err := ds.Metadata(ctx)
		if err != nil {
			stream.SendError(QueryError(fmt.Errorf("error getting metadata for dataset %s: %w", ds.DatasetID, err), projectID, BigQueryDataset.String()))
			continue
		}

		item, sdpErr := toSDPItem(ctx, meta)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		stream.SendItem(item)
	}
}

func NewBigQueryDatasetClient(client *bigquery.Client) BigQueryDatasetClient {
	return &bigQueryDatasetClient{
		client: client,
	}
}

type BigQueryTableClient interface {
	Get(ctx context.Context, projectID, datasetID, tableID string) (*bigquery.TableMetadata, error)
	List(ctx context.Context, projectID, datasetID string, toSDPItem func(table *bigquery.TableMetadata) (*sdp.Item, *sdp.QueryError)) ([]*sdp.Item, *sdp.QueryError)
	ListStream(ctx context.Context, projectID, datasetID string, stream discovery.QueryResultStream, toSDPItem func(table *bigquery.TableMetadata) (*sdp.Item, *sdp.QueryError))
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
		return nil, QueryError(fmt.Errorf("dataset %s not found in project %s", datasetID, projectID), projectID, BigQueryTable.String())
	}

	tableIterator := ds.Tables(ctx)
	if tableIterator == nil {
		return nil, QueryError(fmt.Errorf("failed to create table iterator for dataset %s in project %s", datasetID, projectID), projectID, BigQueryTable.String())
	}

	var items []*sdp.Item
	for {
		table, err := tableIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, QueryError(fmt.Errorf("error iterating tables: %w", err), projectID, BigQueryTable.String())
		}

		meta, err := table.Metadata(ctx)
		if err != nil {
			return nil, QueryError(fmt.Errorf("error getting metadata for table %s: %w", table.TableID, err), projectID, BigQueryTable.String())
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

func (b bigQueryTableClient) ListStream(ctx context.Context, projectID, datasetID string, stream discovery.QueryResultStream, toSDPItem func(table *bigquery.TableMetadata) (*sdp.Item, *sdp.QueryError)) {
	ds := b.client.DatasetInProject(projectID, datasetID)
	if ds == nil {
		stream.SendError(QueryError(fmt.Errorf("dataset %s not found in project %s", datasetID, projectID), projectID, BigQueryTable.String()))
		return
	}

	tableIterator := ds.Tables(ctx)
	if tableIterator == nil {
		stream.SendError(QueryError(fmt.Errorf("failed to create table iterator for dataset %s in project %s", datasetID, projectID), projectID, BigQueryTable.String()))
		return
	}

	for {
		table, err := tableIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(QueryError(fmt.Errorf("error iterating tables: %w", err), projectID, BigQueryTable.String()))
			return
		}

		meta, err := table.Metadata(ctx)
		if err != nil {
			stream.SendError(QueryError(fmt.Errorf("error getting metadata for table %s: %w", table.TableID, err), projectID, BigQueryTable.String()))
			continue
		}

		item, sdpErr := toSDPItem(meta)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		stream.SendItem(item)
	}
}

func NewBigQueryTableClient(client *bigquery.Client) BigQueryTableClient {
	return &bigQueryTableClient{
		client: client,
	}
}

type BigQueryModelClient interface {
	Get(ctx context.Context, projectID, datasetID, modelID string) (*bigquery.ModelMetadata, error)
	List(ctx context.Context, projectID, datasetID string, toSDPItem func(datasetID string, dataset *bigquery.ModelMetadata) (*sdp.Item, *sdp.QueryError)) ([]*sdp.Item, *sdp.QueryError)
	ListStream(ctx context.Context, projectID, datasetID string, stream discovery.QueryResultStream, toSDPItem func(datasetID string, dataset *bigquery.ModelMetadata) (*sdp.Item, *sdp.QueryError))
}

type bigQueryModelClient struct {
	client *bigquery.Client
}

func NewBigQueryModelClient(client *bigquery.Client) BigQueryModelClient {
	return &bigQueryModelClient{
		client: client,
	}
}

func (b bigQueryModelClient) Get(ctx context.Context, projectID, datasetID, modelID string) (*bigquery.ModelMetadata, error) {
	ds := b.client.DatasetInProject(projectID, datasetID)
	if ds == nil {
		return nil, fmt.Errorf("dataset %s not found in project %s", datasetID, projectID)
	}

	model := ds.Model(modelID)
	if model == nil {
		return nil, fmt.Errorf("model %s not found in dataset %s in project %s", modelID, datasetID, projectID)
	}

	return model.Metadata(ctx)
}

func (b bigQueryModelClient) List(ctx context.Context, projectID, datasetID string, toSDPItem func(datasetID string, dataset *bigquery.ModelMetadata) (*sdp.Item, *sdp.QueryError)) ([]*sdp.Item, *sdp.QueryError) {
	ds := b.client.DatasetInProject(projectID, datasetID)
	if ds == nil {
		return nil, QueryError(fmt.Errorf("dataset %s not found in project %s", datasetID, projectID), projectID, BigQueryModel.String())
	}

	modelIterator := ds.Models(ctx)
	if modelIterator == nil {
		return nil, QueryError(fmt.Errorf("failed to create model iterator for dataset %s in project %s", datasetID, projectID), projectID, BigQueryModel.String())
	}

	var items []*sdp.Item
	for {
		model, err := modelIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, QueryError(fmt.Errorf("error iterating models: %w", err), projectID, BigQueryModel.String())
		}

		meta, err := model.Metadata(ctx)
		if err != nil {
			return nil, QueryError(fmt.Errorf("error getting metadata for model %s: %w", model.ModelID, err), projectID, BigQueryModel.String())
		}

		var sdpErr *sdp.QueryError
		item, sdpErr := toSDPItem(datasetID, meta)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (b bigQueryModelClient) ListStream(ctx context.Context, projectID, datasetID string, stream discovery.QueryResultStream, toSDPItem func(datasetID string, dataset *bigquery.ModelMetadata) (*sdp.Item, *sdp.QueryError)) {
	ds := b.client.DatasetInProject(projectID, datasetID)
	if ds == nil {
		stream.SendError(QueryError(fmt.Errorf("dataset %s not found in project %s", datasetID, projectID), projectID, BigQueryModel.String()))
		return
	}

	modelIterator := ds.Models(ctx)
	if modelIterator == nil {
		stream.SendError(QueryError(fmt.Errorf("failed to create model iterator for dataset %s in project %s", datasetID, projectID), projectID, BigQueryModel.String()))
		return
	}

	for {
		model, err := modelIterator.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			stream.SendError(QueryError(fmt.Errorf("error iterating models: %w", err), projectID, BigQueryModel.String()))
			return
		}

		meta, err := model.Metadata(ctx)
		if err != nil {
			stream.SendError(QueryError(fmt.Errorf("error getting metadata for model %s: %w", model.ModelID, err), projectID, BigQueryModel.String()))
			continue
		}

		item, sdpErr := toSDPItem(datasetID, meta)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		stream.SendItem(item)
	}
}
