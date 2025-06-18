//go:generate mockgen -destination=./mocks/mock_logging_config_client.go -package=mocks -source=logging-clients.go
package shared

import (
	"context"

	logging "cloud.google.com/go/logging/apiv2"
	"cloud.google.com/go/logging/apiv2/loggingpb"
)

type LoggingSinkIterator interface {
	Next() (*loggingpb.LogSink, error)
}

type LoggingConfigClient interface {
	ListSinks(ctx context.Context, request *loggingpb.ListSinksRequest) LoggingSinkIterator
	GetSink(ctx context.Context, req *loggingpb.GetSinkRequest) (*loggingpb.LogSink, error)
}

type loggingConfigClient struct {
	configCli *logging.ConfigClient
}

func (l loggingConfigClient) ListSinks(ctx context.Context, req *loggingpb.ListSinksRequest) LoggingSinkIterator {
	return l.configCli.ListSinks(ctx, req)
}

func (l loggingConfigClient) GetSink(ctx context.Context, req *loggingpb.GetSinkRequest) (*loggingpb.LogSink, error) {
	return l.configCli.GetSink(ctx, req)
}

// NewLoggingConfigClient creates a new logging config client
func NewLoggingConfigClient(cli *logging.ConfigClient) LoggingConfigClient {
	return &loggingConfigClient{
		configCli: cli,
	}
}
