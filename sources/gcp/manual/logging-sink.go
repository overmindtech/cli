package manual

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/logging/apiv2/loggingpb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var LoggingSinkLookupByName = shared.NewItemTypeLookup("name", gcpshared.LoggingSink)

// NewLoggingSink creates a new logging sink instance
func NewLoggingSink(client gcpshared.LoggingConfigClient, projectID string) sources.ListableWrapper {
	return &loggingSinkWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			projectID,
			sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
			gcpshared.LoggingSink,
		),
	}
}

type loggingSinkWrapper struct {
	client gcpshared.LoggingConfigClient

	*gcpshared.ProjectBase
}

func (l loggingSinkWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		LoggingSinkLookupByName,
	}
}

func (l loggingSinkWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.StorageBucket,
		gcpshared.BigQueryDataset,
		gcpshared.PubSubTopic,
		gcpshared.LoggingBucket,
	)
}

func (l loggingSinkWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	// O: sink name
	sink, err := l.client.GetSink(ctx, &loggingpb.GetSinkRequest{
		SinkName: fmt.Sprintf("projects/%s/sinks/%s", l.ProjectID(), queryParts[0]),
	})
	if err != nil {
		return nil, gcpshared.QueryError(err)
	}

	var sdpErr *sdp.QueryError
	var item *sdp.Item
	item, sdpErr = l.gcpLoggingSinkToItem(sink)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

func (l loggingSinkWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	it := l.client.ListSinks(ctx, &loggingpb.ListSinksRequest{
		Parent: fmt.Sprintf("projects/%s", l.ProjectID()),
	})

	var items []*sdp.Item
	for {
		sink, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, gcpshared.QueryError(err)
		}

		var sdpErr *sdp.QueryError
		var item *sdp.Item
		item, sdpErr = l.gcpLoggingSinkToItem(sink)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (l loggingSinkWrapper) gcpLoggingSinkToItem(sink *loggingpb.LogSink) (*sdp.Item, *sdp.QueryError) {
	// Convert the GCP logging sink to an SDP item
	attributes, err := shared.ToAttributesWithExclude(sink)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	simpleName := gcpshared.ExtractPathParam("sinks", sink.GetName())
	if simpleName == "" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "unable to extract sink name from the full name",
		}
	}

	err = attributes.Set("uniqueAttr", simpleName)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	item := &sdp.Item{
		Type:            gcpshared.LoggingSink.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           l.DefaultScope(),
	}

	if sink.GetDestination() != "" {
		switch {
		case strings.HasPrefix(sink.GetDestination(), "storage.googleapis.com"):
			// "storage.googleapis.com/[GCS_BUCKET]"
			parts := strings.Split(sink.GetDestination(), "/")
			if len(parts) == 2 {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.StorageBucket.String(),
						Method: sdp.QueryMethod_GET,
						Query:  parts[1], // Bucket name
						Scope:  l.ProjectID(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // Changes to bucket affect sink
						Out: false, // Changes to sink don't affect bucket
					},
				})
			}
		case strings.HasPrefix(sink.GetDestination(), "bigquery.googleapis.com"):
			// "bigquery.googleapis.com/projects/[PROJECT_ID]/datasets/[DATASET]"
			values := gcpshared.ExtractPathParams(sink.GetDestination(), "projects", "datasets")
			if len(values) == 2 && values[0] != "" && values[1] != "" {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.BigQueryDataset.String(),
						Method: sdp.QueryMethod_GET,
						Query:  values[1], // Dataset ID
						Scope:  values[0], // Project ID
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // Changes to dataset affect sink
						Out: false, // Changes to sink don't affect dataset
					},
				})
			}
		case strings.HasPrefix(sink.GetDestination(), "pubsub.googleapis.com"):
			// "pubsub.googleapis.com/projects/[PROJECT_ID]/topics/[TOPIC_ID]"
			values := gcpshared.ExtractPathParams(sink.GetDestination(), "projects", "topics")
			if len(values) == 2 && values[0] != "" && values[1] != "" {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.PubSubTopic.String(),
						Method: sdp.QueryMethod_GET,
						Query:  values[1], // Topic ID
						Scope:  values[0], // Project ID
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // Changes to topic affect sink
						Out: false, // Changes to sink don't affect topic
					},
				})
			}
		case strings.HasPrefix(sink.GetDestination(), "logging.googleapis.com"):
			// "logging.googleapis.com/projects/[PROJECT_ID]/locations/[LOCATION_ID]/buckets/[BUCKET_ID]"
			values := gcpshared.ExtractPathParams(sink.GetDestination(), "projects", "locations", "buckets")
			if len(values) == 3 && values[0] != "" && values[1] != "" && values[2] != "" {
				item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.LoggingBucket.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(values[1], values[2]), // location|bucket_ID
						Scope:  values[0],                                       // Project ID
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true,  // Changes to bucket affect sink
						Out: false, // Changes to sink don't affect bucket
					},
				})
			}
		}
	}

	return item, nil
}
