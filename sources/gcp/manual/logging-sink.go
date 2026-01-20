package manual

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/logging/apiv2/loggingpb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var LoggingSinkLookupByName = shared.NewItemTypeLookup("name", gcpshared.LoggingSink)

// NewLoggingSink creates a new logging sink instance.
func NewLoggingSink(client gcpshared.LoggingConfigClient, locations []gcpshared.LocationInfo) sources.ListStreamableWrapper {
	return &loggingSinkWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_CONFIGURATION,
			gcpshared.LoggingSink,
		),
	}
}

// IAMPermissions returns the required IAM permissions for the logging sink wrapper
func (l loggingSinkWrapper) IAMPermissions() []string {
	return []string{
		"logging.sinks.get",
		"logging.sinks.list",
	}
}

func (l loggingSinkWrapper) PredefinedRole() string {
	return "roles/logging.viewer"
}

type loggingSinkWrapper struct {
	client gcpshared.LoggingConfigClient

	*gcpshared.ProjectBase
}

// assert interface
var _ sources.ListStreamableWrapper = (*loggingSinkWrapper)(nil)

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
		gcpshared.IAMServiceAccount,
	)
}

func (l loggingSinkWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := l.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	sink, getErr := l.client.GetSink(ctx, &loggingpb.GetSinkRequest{
		SinkName: fmt.Sprintf("projects/%s/sinks/%s", location.ProjectID, queryParts[0]),
	})
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, l.Type())
	}

	return l.gcpLoggingSinkToItem(sink, location)
}

func (l loggingSinkWrapper) List(ctx context.Context, scope string) ([]*sdp.Item, *sdp.QueryError) {
	location, err := l.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	it := l.client.ListSinks(ctx, &loggingpb.ListSinksRequest{
		Parent: fmt.Sprintf("projects/%s", location.ProjectID),
	})

	var items []*sdp.Item
	for {
		sink, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			return nil, gcpshared.QueryError(iterErr, scope, l.Type())
		}

		item, sdpErr := l.gcpLoggingSinkToItem(sink, location)
		if sdpErr != nil {
			return nil, sdpErr
		}

		items = append(items, item)
	}

	return items, nil
}

func (l loggingSinkWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string) {
	location, err := l.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	it := l.client.ListSinks(ctx, &loggingpb.ListSinksRequest{
		Parent: fmt.Sprintf("projects/%s", location.ProjectID),
	})

	for {
		sink, iterErr := it.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, l.Type()))
			return
		}

		item, sdpErr := l.gcpLoggingSinkToItem(sink, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

func (l loggingSinkWrapper) gcpLoggingSinkToItem(sink *loggingpb.LogSink, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(sink)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	item := &sdp.Item{
		Type:            gcpshared.LoggingSink.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           location.ToScope(),
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
						Scope:  location.ProjectID,
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

	// Link to IAM Service Account from writerIdentity
	// The writerIdentity field contains the IAM identity (service account email or group) under which
	// Cloud Logging writes the exported log entries. We only link if it's a service account email.
	// Format: service-account@project-id.iam.gserviceaccount.com
	if writerIdentity := sink.GetWriterIdentity(); writerIdentity != "" {
		if strings.Contains(writerIdentity, ".iam.gserviceaccount.com") {
			// Extract project ID from service account email
			// Format: {account-id}@{project-id}.iam.gserviceaccount.com
			parts := strings.Split(writerIdentity, "@")
			if len(parts) == 2 {
				domain := parts[1]
				// Remove .iam.gserviceaccount.com to get project ID
				projectID := strings.TrimSuffix(domain, ".iam.gserviceaccount.com")
				if projectID != "" {
					item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   gcpshared.IAMServiceAccount.String(),
							Method: sdp.QueryMethod_GET,
							Query:  writerIdentity, // Service account email
							Scope:  projectID,      // Project ID extracted from email
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,  // If the service account is deleted or its permissions are changed: The sink may fail to export logs
							Out: false, // Changes to the sink don't affect the service account
						},
					})
				}
			}
		}
	}

	return item, nil
}
