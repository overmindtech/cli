// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: snapshots.proto

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
	// SnapshotsServiceName is the fully-qualified name of the SnapshotsService service.
	SnapshotsServiceName = "snapshots.SnapshotsService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// SnapshotsServiceListSnapshotsProcedure is the fully-qualified name of the SnapshotsService's
	// ListSnapshots RPC.
	SnapshotsServiceListSnapshotsProcedure = "/snapshots.SnapshotsService/ListSnapshots"
	// SnapshotsServiceCreateSnapshotProcedure is the fully-qualified name of the SnapshotsService's
	// CreateSnapshot RPC.
	SnapshotsServiceCreateSnapshotProcedure = "/snapshots.SnapshotsService/CreateSnapshot"
	// SnapshotsServiceGetSnapshotProcedure is the fully-qualified name of the SnapshotsService's
	// GetSnapshot RPC.
	SnapshotsServiceGetSnapshotProcedure = "/snapshots.SnapshotsService/GetSnapshot"
	// SnapshotsServiceUpdateSnapshotProcedure is the fully-qualified name of the SnapshotsService's
	// UpdateSnapshot RPC.
	SnapshotsServiceUpdateSnapshotProcedure = "/snapshots.SnapshotsService/UpdateSnapshot"
	// SnapshotsServiceDeleteSnapshotProcedure is the fully-qualified name of the SnapshotsService's
	// DeleteSnapshot RPC.
	SnapshotsServiceDeleteSnapshotProcedure = "/snapshots.SnapshotsService/DeleteSnapshot"
	// SnapshotsServiceListSnapshotByGUNProcedure is the fully-qualified name of the SnapshotsService's
	// ListSnapshotByGUN RPC.
	SnapshotsServiceListSnapshotByGUNProcedure = "/snapshots.SnapshotsService/ListSnapshotByGUN"
)

// SnapshotsServiceClient is a client for the snapshots.SnapshotsService service.
type SnapshotsServiceClient interface {
	ListSnapshots(context.Context, *connect.Request[sdp_go.ListSnapshotsRequest]) (*connect.Response[sdp_go.ListSnapshotResponse], error)
	CreateSnapshot(context.Context, *connect.Request[sdp_go.CreateSnapshotRequest]) (*connect.Response[sdp_go.CreateSnapshotResponse], error)
	GetSnapshot(context.Context, *connect.Request[sdp_go.GetSnapshotRequest]) (*connect.Response[sdp_go.GetSnapshotResponse], error)
	UpdateSnapshot(context.Context, *connect.Request[sdp_go.UpdateSnapshotRequest]) (*connect.Response[sdp_go.UpdateSnapshotResponse], error)
	DeleteSnapshot(context.Context, *connect.Request[sdp_go.DeleteSnapshotRequest]) (*connect.Response[sdp_go.DeleteSnapshotResponse], error)
	ListSnapshotByGUN(context.Context, *connect.Request[sdp_go.ListSnapshotsByGUNRequest]) (*connect.Response[sdp_go.ListSnapshotsByGUNResponse], error)
}

// NewSnapshotsServiceClient constructs a client for the snapshots.SnapshotsService service. By
// default, it uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses,
// and sends uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the
// connect.WithGRPC() or connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewSnapshotsServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) SnapshotsServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	snapshotsServiceMethods := sdp_go.File_snapshots_proto.Services().ByName("SnapshotsService").Methods()
	return &snapshotsServiceClient{
		listSnapshots: connect.NewClient[sdp_go.ListSnapshotsRequest, sdp_go.ListSnapshotResponse](
			httpClient,
			baseURL+SnapshotsServiceListSnapshotsProcedure,
			connect.WithSchema(snapshotsServiceMethods.ByName("ListSnapshots")),
			connect.WithClientOptions(opts...),
		),
		createSnapshot: connect.NewClient[sdp_go.CreateSnapshotRequest, sdp_go.CreateSnapshotResponse](
			httpClient,
			baseURL+SnapshotsServiceCreateSnapshotProcedure,
			connect.WithSchema(snapshotsServiceMethods.ByName("CreateSnapshot")),
			connect.WithClientOptions(opts...),
		),
		getSnapshot: connect.NewClient[sdp_go.GetSnapshotRequest, sdp_go.GetSnapshotResponse](
			httpClient,
			baseURL+SnapshotsServiceGetSnapshotProcedure,
			connect.WithSchema(snapshotsServiceMethods.ByName("GetSnapshot")),
			connect.WithClientOptions(opts...),
		),
		updateSnapshot: connect.NewClient[sdp_go.UpdateSnapshotRequest, sdp_go.UpdateSnapshotResponse](
			httpClient,
			baseURL+SnapshotsServiceUpdateSnapshotProcedure,
			connect.WithSchema(snapshotsServiceMethods.ByName("UpdateSnapshot")),
			connect.WithClientOptions(opts...),
		),
		deleteSnapshot: connect.NewClient[sdp_go.DeleteSnapshotRequest, sdp_go.DeleteSnapshotResponse](
			httpClient,
			baseURL+SnapshotsServiceDeleteSnapshotProcedure,
			connect.WithSchema(snapshotsServiceMethods.ByName("DeleteSnapshot")),
			connect.WithClientOptions(opts...),
		),
		listSnapshotByGUN: connect.NewClient[sdp_go.ListSnapshotsByGUNRequest, sdp_go.ListSnapshotsByGUNResponse](
			httpClient,
			baseURL+SnapshotsServiceListSnapshotByGUNProcedure,
			connect.WithSchema(snapshotsServiceMethods.ByName("ListSnapshotByGUN")),
			connect.WithClientOptions(opts...),
		),
	}
}

// snapshotsServiceClient implements SnapshotsServiceClient.
type snapshotsServiceClient struct {
	listSnapshots     *connect.Client[sdp_go.ListSnapshotsRequest, sdp_go.ListSnapshotResponse]
	createSnapshot    *connect.Client[sdp_go.CreateSnapshotRequest, sdp_go.CreateSnapshotResponse]
	getSnapshot       *connect.Client[sdp_go.GetSnapshotRequest, sdp_go.GetSnapshotResponse]
	updateSnapshot    *connect.Client[sdp_go.UpdateSnapshotRequest, sdp_go.UpdateSnapshotResponse]
	deleteSnapshot    *connect.Client[sdp_go.DeleteSnapshotRequest, sdp_go.DeleteSnapshotResponse]
	listSnapshotByGUN *connect.Client[sdp_go.ListSnapshotsByGUNRequest, sdp_go.ListSnapshotsByGUNResponse]
}

// ListSnapshots calls snapshots.SnapshotsService.ListSnapshots.
func (c *snapshotsServiceClient) ListSnapshots(ctx context.Context, req *connect.Request[sdp_go.ListSnapshotsRequest]) (*connect.Response[sdp_go.ListSnapshotResponse], error) {
	return c.listSnapshots.CallUnary(ctx, req)
}

// CreateSnapshot calls snapshots.SnapshotsService.CreateSnapshot.
func (c *snapshotsServiceClient) CreateSnapshot(ctx context.Context, req *connect.Request[sdp_go.CreateSnapshotRequest]) (*connect.Response[sdp_go.CreateSnapshotResponse], error) {
	return c.createSnapshot.CallUnary(ctx, req)
}

// GetSnapshot calls snapshots.SnapshotsService.GetSnapshot.
func (c *snapshotsServiceClient) GetSnapshot(ctx context.Context, req *connect.Request[sdp_go.GetSnapshotRequest]) (*connect.Response[sdp_go.GetSnapshotResponse], error) {
	return c.getSnapshot.CallUnary(ctx, req)
}

// UpdateSnapshot calls snapshots.SnapshotsService.UpdateSnapshot.
func (c *snapshotsServiceClient) UpdateSnapshot(ctx context.Context, req *connect.Request[sdp_go.UpdateSnapshotRequest]) (*connect.Response[sdp_go.UpdateSnapshotResponse], error) {
	return c.updateSnapshot.CallUnary(ctx, req)
}

// DeleteSnapshot calls snapshots.SnapshotsService.DeleteSnapshot.
func (c *snapshotsServiceClient) DeleteSnapshot(ctx context.Context, req *connect.Request[sdp_go.DeleteSnapshotRequest]) (*connect.Response[sdp_go.DeleteSnapshotResponse], error) {
	return c.deleteSnapshot.CallUnary(ctx, req)
}

// ListSnapshotByGUN calls snapshots.SnapshotsService.ListSnapshotByGUN.
func (c *snapshotsServiceClient) ListSnapshotByGUN(ctx context.Context, req *connect.Request[sdp_go.ListSnapshotsByGUNRequest]) (*connect.Response[sdp_go.ListSnapshotsByGUNResponse], error) {
	return c.listSnapshotByGUN.CallUnary(ctx, req)
}

// SnapshotsServiceHandler is an implementation of the snapshots.SnapshotsService service.
type SnapshotsServiceHandler interface {
	ListSnapshots(context.Context, *connect.Request[sdp_go.ListSnapshotsRequest]) (*connect.Response[sdp_go.ListSnapshotResponse], error)
	CreateSnapshot(context.Context, *connect.Request[sdp_go.CreateSnapshotRequest]) (*connect.Response[sdp_go.CreateSnapshotResponse], error)
	GetSnapshot(context.Context, *connect.Request[sdp_go.GetSnapshotRequest]) (*connect.Response[sdp_go.GetSnapshotResponse], error)
	UpdateSnapshot(context.Context, *connect.Request[sdp_go.UpdateSnapshotRequest]) (*connect.Response[sdp_go.UpdateSnapshotResponse], error)
	DeleteSnapshot(context.Context, *connect.Request[sdp_go.DeleteSnapshotRequest]) (*connect.Response[sdp_go.DeleteSnapshotResponse], error)
	ListSnapshotByGUN(context.Context, *connect.Request[sdp_go.ListSnapshotsByGUNRequest]) (*connect.Response[sdp_go.ListSnapshotsByGUNResponse], error)
}

// NewSnapshotsServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewSnapshotsServiceHandler(svc SnapshotsServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	snapshotsServiceMethods := sdp_go.File_snapshots_proto.Services().ByName("SnapshotsService").Methods()
	snapshotsServiceListSnapshotsHandler := connect.NewUnaryHandler(
		SnapshotsServiceListSnapshotsProcedure,
		svc.ListSnapshots,
		connect.WithSchema(snapshotsServiceMethods.ByName("ListSnapshots")),
		connect.WithHandlerOptions(opts...),
	)
	snapshotsServiceCreateSnapshotHandler := connect.NewUnaryHandler(
		SnapshotsServiceCreateSnapshotProcedure,
		svc.CreateSnapshot,
		connect.WithSchema(snapshotsServiceMethods.ByName("CreateSnapshot")),
		connect.WithHandlerOptions(opts...),
	)
	snapshotsServiceGetSnapshotHandler := connect.NewUnaryHandler(
		SnapshotsServiceGetSnapshotProcedure,
		svc.GetSnapshot,
		connect.WithSchema(snapshotsServiceMethods.ByName("GetSnapshot")),
		connect.WithHandlerOptions(opts...),
	)
	snapshotsServiceUpdateSnapshotHandler := connect.NewUnaryHandler(
		SnapshotsServiceUpdateSnapshotProcedure,
		svc.UpdateSnapshot,
		connect.WithSchema(snapshotsServiceMethods.ByName("UpdateSnapshot")),
		connect.WithHandlerOptions(opts...),
	)
	snapshotsServiceDeleteSnapshotHandler := connect.NewUnaryHandler(
		SnapshotsServiceDeleteSnapshotProcedure,
		svc.DeleteSnapshot,
		connect.WithSchema(snapshotsServiceMethods.ByName("DeleteSnapshot")),
		connect.WithHandlerOptions(opts...),
	)
	snapshotsServiceListSnapshotByGUNHandler := connect.NewUnaryHandler(
		SnapshotsServiceListSnapshotByGUNProcedure,
		svc.ListSnapshotByGUN,
		connect.WithSchema(snapshotsServiceMethods.ByName("ListSnapshotByGUN")),
		connect.WithHandlerOptions(opts...),
	)
	return "/snapshots.SnapshotsService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case SnapshotsServiceListSnapshotsProcedure:
			snapshotsServiceListSnapshotsHandler.ServeHTTP(w, r)
		case SnapshotsServiceCreateSnapshotProcedure:
			snapshotsServiceCreateSnapshotHandler.ServeHTTP(w, r)
		case SnapshotsServiceGetSnapshotProcedure:
			snapshotsServiceGetSnapshotHandler.ServeHTTP(w, r)
		case SnapshotsServiceUpdateSnapshotProcedure:
			snapshotsServiceUpdateSnapshotHandler.ServeHTTP(w, r)
		case SnapshotsServiceDeleteSnapshotProcedure:
			snapshotsServiceDeleteSnapshotHandler.ServeHTTP(w, r)
		case SnapshotsServiceListSnapshotByGUNProcedure:
			snapshotsServiceListSnapshotByGUNHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedSnapshotsServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedSnapshotsServiceHandler struct{}

func (UnimplementedSnapshotsServiceHandler) ListSnapshots(context.Context, *connect.Request[sdp_go.ListSnapshotsRequest]) (*connect.Response[sdp_go.ListSnapshotResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("snapshots.SnapshotsService.ListSnapshots is not implemented"))
}

func (UnimplementedSnapshotsServiceHandler) CreateSnapshot(context.Context, *connect.Request[sdp_go.CreateSnapshotRequest]) (*connect.Response[sdp_go.CreateSnapshotResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("snapshots.SnapshotsService.CreateSnapshot is not implemented"))
}

func (UnimplementedSnapshotsServiceHandler) GetSnapshot(context.Context, *connect.Request[sdp_go.GetSnapshotRequest]) (*connect.Response[sdp_go.GetSnapshotResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("snapshots.SnapshotsService.GetSnapshot is not implemented"))
}

func (UnimplementedSnapshotsServiceHandler) UpdateSnapshot(context.Context, *connect.Request[sdp_go.UpdateSnapshotRequest]) (*connect.Response[sdp_go.UpdateSnapshotResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("snapshots.SnapshotsService.UpdateSnapshot is not implemented"))
}

func (UnimplementedSnapshotsServiceHandler) DeleteSnapshot(context.Context, *connect.Request[sdp_go.DeleteSnapshotRequest]) (*connect.Response[sdp_go.DeleteSnapshotResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("snapshots.SnapshotsService.DeleteSnapshot is not implemented"))
}

func (UnimplementedSnapshotsServiceHandler) ListSnapshotByGUN(context.Context, *connect.Request[sdp_go.ListSnapshotsByGUNRequest]) (*connect.Response[sdp_go.ListSnapshotsByGUNResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("snapshots.SnapshotsService.ListSnapshotByGUN is not implemented"))
}
