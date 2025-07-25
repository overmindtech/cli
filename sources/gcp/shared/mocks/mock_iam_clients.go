// Code generated by MockGen. DO NOT EDIT.
// Source: iam-clients.go
//
// Generated by this command:
//
//	mockgen -destination=./mocks/mock_iam_clients.go -package=mocks -source=iam-clients.go
//

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	adminpb "cloud.google.com/go/iam/admin/apiv1/adminpb"
	gax "github.com/googleapis/gax-go/v2"
	shared "github.com/overmindtech/cli/sources/gcp/shared"
	gomock "go.uber.org/mock/gomock"
)

// MockIAMServiceAccountClient is a mock of IAMServiceAccountClient interface.
type MockIAMServiceAccountClient struct {
	ctrl     *gomock.Controller
	recorder *MockIAMServiceAccountClientMockRecorder
	isgomock struct{}
}

// MockIAMServiceAccountClientMockRecorder is the mock recorder for MockIAMServiceAccountClient.
type MockIAMServiceAccountClientMockRecorder struct {
	mock *MockIAMServiceAccountClient
}

// NewMockIAMServiceAccountClient creates a new mock instance.
func NewMockIAMServiceAccountClient(ctrl *gomock.Controller) *MockIAMServiceAccountClient {
	mock := &MockIAMServiceAccountClient{ctrl: ctrl}
	mock.recorder = &MockIAMServiceAccountClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIAMServiceAccountClient) EXPECT() *MockIAMServiceAccountClientMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockIAMServiceAccountClient) Get(ctx context.Context, req *adminpb.GetServiceAccountRequest, opts ...gax.CallOption) (*adminpb.ServiceAccount, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, req}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Get", varargs...)
	ret0, _ := ret[0].(*adminpb.ServiceAccount)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockIAMServiceAccountClientMockRecorder) Get(ctx, req any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, req}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockIAMServiceAccountClient)(nil).Get), varargs...)
}

// List mocks base method.
func (m *MockIAMServiceAccountClient) List(ctx context.Context, req *adminpb.ListServiceAccountsRequest, opts ...gax.CallOption) shared.IAMServiceAccountIterator {
	m.ctrl.T.Helper()
	varargs := []any{ctx, req}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "List", varargs...)
	ret0, _ := ret[0].(shared.IAMServiceAccountIterator)
	return ret0
}

// List indicates an expected call of List.
func (mr *MockIAMServiceAccountClientMockRecorder) List(ctx, req any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, req}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockIAMServiceAccountClient)(nil).List), varargs...)
}

// MockIAMServiceAccountIterator is a mock of IAMServiceAccountIterator interface.
type MockIAMServiceAccountIterator struct {
	ctrl     *gomock.Controller
	recorder *MockIAMServiceAccountIteratorMockRecorder
	isgomock struct{}
}

// MockIAMServiceAccountIteratorMockRecorder is the mock recorder for MockIAMServiceAccountIterator.
type MockIAMServiceAccountIteratorMockRecorder struct {
	mock *MockIAMServiceAccountIterator
}

// NewMockIAMServiceAccountIterator creates a new mock instance.
func NewMockIAMServiceAccountIterator(ctrl *gomock.Controller) *MockIAMServiceAccountIterator {
	mock := &MockIAMServiceAccountIterator{ctrl: ctrl}
	mock.recorder = &MockIAMServiceAccountIteratorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIAMServiceAccountIterator) EXPECT() *MockIAMServiceAccountIteratorMockRecorder {
	return m.recorder
}

// Next mocks base method.
func (m *MockIAMServiceAccountIterator) Next() (*adminpb.ServiceAccount, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Next")
	ret0, _ := ret[0].(*adminpb.ServiceAccount)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Next indicates an expected call of Next.
func (mr *MockIAMServiceAccountIteratorMockRecorder) Next() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Next", reflect.TypeOf((*MockIAMServiceAccountIterator)(nil).Next))
}

// MockIAMServiceAccountKeyClient is a mock of IAMServiceAccountKeyClient interface.
type MockIAMServiceAccountKeyClient struct {
	ctrl     *gomock.Controller
	recorder *MockIAMServiceAccountKeyClientMockRecorder
	isgomock struct{}
}

// MockIAMServiceAccountKeyClientMockRecorder is the mock recorder for MockIAMServiceAccountKeyClient.
type MockIAMServiceAccountKeyClientMockRecorder struct {
	mock *MockIAMServiceAccountKeyClient
}

// NewMockIAMServiceAccountKeyClient creates a new mock instance.
func NewMockIAMServiceAccountKeyClient(ctrl *gomock.Controller) *MockIAMServiceAccountKeyClient {
	mock := &MockIAMServiceAccountKeyClient{ctrl: ctrl}
	mock.recorder = &MockIAMServiceAccountKeyClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockIAMServiceAccountKeyClient) EXPECT() *MockIAMServiceAccountKeyClientMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockIAMServiceAccountKeyClient) Get(ctx context.Context, req *adminpb.GetServiceAccountKeyRequest, opts ...gax.CallOption) (*adminpb.ServiceAccountKey, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, req}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Get", varargs...)
	ret0, _ := ret[0].(*adminpb.ServiceAccountKey)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockIAMServiceAccountKeyClientMockRecorder) Get(ctx, req any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, req}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockIAMServiceAccountKeyClient)(nil).Get), varargs...)
}

// Search mocks base method.
func (m *MockIAMServiceAccountKeyClient) Search(ctx context.Context, req *adminpb.ListServiceAccountKeysRequest, opts ...gax.CallOption) (*adminpb.ListServiceAccountKeysResponse, error) {
	m.ctrl.T.Helper()
	varargs := []any{ctx, req}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Search", varargs...)
	ret0, _ := ret[0].(*adminpb.ListServiceAccountKeysResponse)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Search indicates an expected call of Search.
func (mr *MockIAMServiceAccountKeyClientMockRecorder) Search(ctx, req any, opts ...any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]any{ctx, req}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Search", reflect.TypeOf((*MockIAMServiceAccountKeyClient)(nil).Search), varargs...)
}
