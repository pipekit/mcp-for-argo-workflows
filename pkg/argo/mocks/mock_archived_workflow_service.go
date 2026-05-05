// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowarchive"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockArchivedWorkflowServiceClient is a mock implementation of workflowarchive.ArchivedWorkflowServiceClient.
type MockArchivedWorkflowServiceClient struct {
	mock.Mock
}

// Ensure MockArchivedWorkflowServiceClient implements the interface.
var _ workflowarchive.ArchivedWorkflowServiceClient = (*MockArchivedWorkflowServiceClient)(nil)

// ListArchivedWorkflows mocks the ListArchivedWorkflows method.
func (m *MockArchivedWorkflowServiceClient) ListArchivedWorkflows(ctx context.Context, req *workflowarchive.ListArchivedWorkflowsRequest, opts ...grpc.CallOption) (*wfv1.WorkflowList, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	wfList, ok := args.Get(0).(*wfv1.WorkflowList)
	if !ok {
		return nil, args.Error(1)
	}
	return wfList, args.Error(1)
}

// GetArchivedWorkflow mocks the GetArchivedWorkflow method.
func (m *MockArchivedWorkflowServiceClient) GetArchivedWorkflow(ctx context.Context, req *workflowarchive.GetArchivedWorkflowRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	wf, ok := args.Get(0).(*wfv1.Workflow)
	if !ok {
		return nil, args.Error(1)
	}
	return wf, args.Error(1)
}

// DeleteArchivedWorkflow mocks the DeleteArchivedWorkflow method.
func (m *MockArchivedWorkflowServiceClient) DeleteArchivedWorkflow(ctx context.Context, req *workflowarchive.DeleteArchivedWorkflowRequest, opts ...grpc.CallOption) (*workflowarchive.ArchivedWorkflowDeletedResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	resp, ok := args.Get(0).(*workflowarchive.ArchivedWorkflowDeletedResponse)
	if !ok {
		return nil, args.Error(1)
	}
	return resp, args.Error(1)
}

// ListArchivedWorkflowLabelKeys mocks the ListArchivedWorkflowLabelKeys method.
func (m *MockArchivedWorkflowServiceClient) ListArchivedWorkflowLabelKeys(ctx context.Context, req *workflowarchive.ListArchivedWorkflowLabelKeysRequest, opts ...grpc.CallOption) (*wfv1.LabelKeys, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	keys, ok := args.Get(0).(*wfv1.LabelKeys)
	if !ok {
		return nil, args.Error(1)
	}
	return keys, args.Error(1)
}

// ListArchivedWorkflowLabelValues mocks the ListArchivedWorkflowLabelValues method.
func (m *MockArchivedWorkflowServiceClient) ListArchivedWorkflowLabelValues(ctx context.Context, req *workflowarchive.ListArchivedWorkflowLabelValuesRequest, opts ...grpc.CallOption) (*wfv1.LabelValues, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	values, ok := args.Get(0).(*wfv1.LabelValues)
	if !ok {
		return nil, args.Error(1)
	}
	return values, args.Error(1)
}

// RetryArchivedWorkflow mocks the RetryArchivedWorkflow method.
func (m *MockArchivedWorkflowServiceClient) RetryArchivedWorkflow(ctx context.Context, req *workflowarchive.RetryArchivedWorkflowRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	wf, ok := args.Get(0).(*wfv1.Workflow)
	if !ok {
		return nil, args.Error(1)
	}
	return wf, args.Error(1)
}

// ResubmitArchivedWorkflow mocks the ResubmitArchivedWorkflow method.
func (m *MockArchivedWorkflowServiceClient) ResubmitArchivedWorkflow(ctx context.Context, req *workflowarchive.ResubmitArchivedWorkflowRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	wf, ok := args.Get(0).(*wfv1.Workflow)
	if !ok {
		return nil, args.Error(1)
	}
	return wf, args.Error(1)
}
