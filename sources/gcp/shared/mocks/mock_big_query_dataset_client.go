// Code generated by MockGen. DO NOT EDIT.
// Source: big-query-clients.go
//
// Generated by this command:
//
//	mockgen -destination=./mocks/mock_big_query_dataset_client.go -package=mocks -source=big-query-clients.go -imports=sdp=github.com/overmindtech/cli/sdp-go
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	bigquery "cloud.google.com/go/bigquery"
	sdp "github.com/overmindtech/cli/sdp-go"
	gomock "go.uber.org/mock/gomock"
)

// MockBigQueryDatasetClient is a mock of BigQueryDatasetClient interface.
type MockBigQueryDatasetClient struct {
	ctrl     *gomock.Controller
	recorder *MockBigQueryDatasetClientMockRecorder
	isgomock struct{}
}

// MockBigQueryDatasetClientMockRecorder is the mock recorder for MockBigQueryDatasetClient.
type MockBigQueryDatasetClientMockRecorder struct {
	mock *MockBigQueryDatasetClient
}

// NewMockBigQueryDatasetClient creates a new mock instance.
func NewMockBigQueryDatasetClient(ctrl *gomock.Controller) *MockBigQueryDatasetClient {
	mock := &MockBigQueryDatasetClient{ctrl: ctrl}
	mock.recorder = &MockBigQueryDatasetClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBigQueryDatasetClient) EXPECT() *MockBigQueryDatasetClientMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockBigQueryDatasetClient) Get(ctx context.Context, projectID, datasetID string) (*bigquery.DatasetMetadata, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, projectID, datasetID)
	ret0, _ := ret[0].(*bigquery.DatasetMetadata)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockBigQueryDatasetClientMockRecorder) Get(ctx, projectID, datasetID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockBigQueryDatasetClient)(nil).Get), ctx, projectID, datasetID)
}

// List mocks base method.
func (m *MockBigQueryDatasetClient) List(ctx context.Context, projectID string, toSDPItem func(context.Context, *bigquery.DatasetMetadata) (*sdp.Item, *sdp.QueryError)) ([]*sdp.Item, *sdp.QueryError) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, projectID, toSDPItem)
	ret0, _ := ret[0].([]*sdp.Item)
	ret1, _ := ret[1].(*sdp.QueryError)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockBigQueryDatasetClientMockRecorder) List(ctx, projectID, toSDPItem any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockBigQueryDatasetClient)(nil).List), ctx, projectID, toSDPItem)
}

// MockBigQueryTableClient is a mock of BigQueryTableClient interface.
type MockBigQueryTableClient struct {
	ctrl     *gomock.Controller
	recorder *MockBigQueryTableClientMockRecorder
	isgomock struct{}
}

// MockBigQueryTableClientMockRecorder is the mock recorder for MockBigQueryTableClient.
type MockBigQueryTableClientMockRecorder struct {
	mock *MockBigQueryTableClient
}

// NewMockBigQueryTableClient creates a new mock instance.
func NewMockBigQueryTableClient(ctrl *gomock.Controller) *MockBigQueryTableClient {
	mock := &MockBigQueryTableClient{ctrl: ctrl}
	mock.recorder = &MockBigQueryTableClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBigQueryTableClient) EXPECT() *MockBigQueryTableClientMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockBigQueryTableClient) Get(ctx context.Context, projectID, datasetID, tableID string) (*bigquery.TableMetadata, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, projectID, datasetID, tableID)
	ret0, _ := ret[0].(*bigquery.TableMetadata)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockBigQueryTableClientMockRecorder) Get(ctx, projectID, datasetID, tableID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockBigQueryTableClient)(nil).Get), ctx, projectID, datasetID, tableID)
}

// List mocks base method.
func (m *MockBigQueryTableClient) List(ctx context.Context, projectID, datasetID string, toSDPItem func(*bigquery.TableMetadata) (*sdp.Item, *sdp.QueryError)) ([]*sdp.Item, *sdp.QueryError) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, projectID, datasetID, toSDPItem)
	ret0, _ := ret[0].([]*sdp.Item)
	ret1, _ := ret[1].(*sdp.QueryError)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockBigQueryTableClientMockRecorder) List(ctx, projectID, datasetID, toSDPItem any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockBigQueryTableClient)(nil).List), ctx, projectID, datasetID, toSDPItem)
}



// MockBigQueryModelClient is a mock of BigQueryModelClient interface.
type MockBigQueryModelClient struct {
	ctrl     *gomock.Controller
	recorder *MockBigQueryModelClientRecorder
	isgomock struct{}
}

// MockBigQueryModelClientRecorder is the mock recorder for MockBigQueryModelClient.
type MockBigQueryModelClientRecorder struct {
	mock *MockBigQueryModelClient
}

// NewMockBigModelClient creates a new mock instance.
func NewMockBigModelClient(ctrl *gomock.Controller) *MockBigQueryModelClient {
	mock := &MockBigQueryModelClient{ctrl: ctrl}
	mock.recorder = &MockBigQueryModelClientRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBigQueryModelClient) EXPECT() *MockBigQueryModelClientRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockBigQueryModelClient) Get(ctx context.Context, projectID, datasetID, modelID string) (*bigquery.ModelMetadata, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, projectID, datasetID, modelID)
	ret0, _ := ret[0].(*bigquery.ModelMetadata)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockBigQueryModelClientRecorder) Get(ctx, projectID, datasetID, modelID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockBigQueryModelClient)(nil).Get), ctx, projectID, datasetID, modelID)
}

// List mocks base method.
func (m *MockBigQueryModelClient) List(ctx context.Context, projectID, datasetID string, toSDPItem func(context.Context,*bigquery.ModelMetadata) (*sdp.Item, *sdp.QueryError)) ([]*sdp.Item, *sdp.QueryError) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx, projectID, datasetID, toSDPItem)
	ret0, _ := ret[0].([]*sdp.Item)
	ret1, _ := ret[1].(*sdp.QueryError)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockBigQueryModelClientRecorder) List(ctx, projectID, datasetID, toSDPItem any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockBigQueryModelClient)(nil).List), ctx, projectID, datasetID, toSDPItem)
}
