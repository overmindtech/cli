// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: logs.proto

package sdpconnect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	sdp_go "github.com/overmindtech/cli/sdp-go"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// LogsServiceName is the fully-qualified name of the LogsService service.
	LogsServiceName = "logs.LogsService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// LogsServiceGetLogRecordsProcedure is the fully-qualified name of the LogsService's GetLogRecords
	// RPC.
	LogsServiceGetLogRecordsProcedure = "/logs.LogsService/GetLogRecords"
)

// LogsServiceClient is a client for the logs.LogsService service.
type LogsServiceClient interface {
	// GetLogRecords returns a stream of log records from the upstream API. The
	// source is expected to use sane defaults within the limits of the
	// underlying API and SDP capabilities (message size, etc). Each chunk is
	// roughly a page of the upstream APIs pagination.
	GetLogRecords(context.Context, *connect.Request[sdp_go.GetLogRecordsRequest]) (*connect.ServerStreamForClient[sdp_go.GetLogRecordsResponse], error)
}

// NewLogsServiceClient constructs a client for the logs.LogsService service. By default, it uses
// the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewLogsServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) LogsServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	logsServiceMethods := sdp_go.File_logs_proto.Services().ByName("LogsService").Methods()
	return &logsServiceClient{
		getLogRecords: connect.NewClient[sdp_go.GetLogRecordsRequest, sdp_go.GetLogRecordsResponse](
			httpClient,
			baseURL+LogsServiceGetLogRecordsProcedure,
			connect.WithSchema(logsServiceMethods.ByName("GetLogRecords")),
			connect.WithClientOptions(opts...),
		),
	}
}

// logsServiceClient implements LogsServiceClient.
type logsServiceClient struct {
	getLogRecords *connect.Client[sdp_go.GetLogRecordsRequest, sdp_go.GetLogRecordsResponse]
}

// GetLogRecords calls logs.LogsService.GetLogRecords.
func (c *logsServiceClient) GetLogRecords(ctx context.Context, req *connect.Request[sdp_go.GetLogRecordsRequest]) (*connect.ServerStreamForClient[sdp_go.GetLogRecordsResponse], error) {
	return c.getLogRecords.CallServerStream(ctx, req)
}

// LogsServiceHandler is an implementation of the logs.LogsService service.
type LogsServiceHandler interface {
	// GetLogRecords returns a stream of log records from the upstream API. The
	// source is expected to use sane defaults within the limits of the
	// underlying API and SDP capabilities (message size, etc). Each chunk is
	// roughly a page of the upstream APIs pagination.
	GetLogRecords(context.Context, *connect.Request[sdp_go.GetLogRecordsRequest], *connect.ServerStream[sdp_go.GetLogRecordsResponse]) error
}

// NewLogsServiceHandler builds an HTTP handler from the service implementation. It returns the path
// on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewLogsServiceHandler(svc LogsServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	logsServiceMethods := sdp_go.File_logs_proto.Services().ByName("LogsService").Methods()
	logsServiceGetLogRecordsHandler := connect.NewServerStreamHandler(
		LogsServiceGetLogRecordsProcedure,
		svc.GetLogRecords,
		connect.WithSchema(logsServiceMethods.ByName("GetLogRecords")),
		connect.WithHandlerOptions(opts...),
	)
	return "/logs.LogsService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case LogsServiceGetLogRecordsProcedure:
			logsServiceGetLogRecordsHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedLogsServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedLogsServiceHandler struct{}

func (UnimplementedLogsServiceHandler) GetLogRecords(context.Context, *connect.Request[sdp_go.GetLogRecordsRequest], *connect.ServerStream[sdp_go.GetLogRecordsResponse]) error {
	return connect.NewError(connect.CodeUnimplemented, errors.New("logs.LogsService.GetLogRecords is not implemented"))
}
