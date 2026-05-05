// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockCronWorkflowServiceClient is a mock implementation of cronworkflow.CronWorkflowServiceClient.
type MockCronWorkflowServiceClient struct {
	mock.Mock
}

// Ensure MockCronWorkflowServiceClient implements the interface.
var _ cronworkflow.CronWorkflowServiceClient = (*MockCronWorkflowServiceClient)(nil)

// LintCronWorkflow mocks the LintCronWorkflow method.
func (m *MockCronWorkflowServiceClient) LintCronWorkflow(ctx context.Context, req *cronworkflow.LintCronWorkflowRequest, opts ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	cw, ok := args.Get(0).(*wfv1.CronWorkflow)
	if !ok {
		return nil, args.Error(1)
	}
	return cw, args.Error(1)
}

// CreateCronWorkflow mocks the CreateCronWorkflow method.
func (m *MockCronWorkflowServiceClient) CreateCronWorkflow(ctx context.Context, req *cronworkflow.CreateCronWorkflowRequest, opts ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	cw, ok := args.Get(0).(*wfv1.CronWorkflow)
	if !ok {
		return nil, args.Error(1)
	}
	return cw, args.Error(1)
}

// ListCronWorkflows mocks the ListCronWorkflows method.
func (m *MockCronWorkflowServiceClient) ListCronWorkflows(ctx context.Context, req *cronworkflow.ListCronWorkflowsRequest, opts ...grpc.CallOption) (*wfv1.CronWorkflowList, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	cwList, ok := args.Get(0).(*wfv1.CronWorkflowList)
	if !ok {
		return nil, args.Error(1)
	}
	return cwList, args.Error(1)
}

// GetCronWorkflow mocks the GetCronWorkflow method.
func (m *MockCronWorkflowServiceClient) GetCronWorkflow(ctx context.Context, req *cronworkflow.GetCronWorkflowRequest, opts ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	cw, ok := args.Get(0).(*wfv1.CronWorkflow)
	if !ok {
		return nil, args.Error(1)
	}
	return cw, args.Error(1)
}

// UpdateCronWorkflow mocks the UpdateCronWorkflow method.
func (m *MockCronWorkflowServiceClient) UpdateCronWorkflow(ctx context.Context, req *cronworkflow.UpdateCronWorkflowRequest, opts ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	cw, ok := args.Get(0).(*wfv1.CronWorkflow)
	if !ok {
		return nil, args.Error(1)
	}
	return cw, args.Error(1)
}

// DeleteCronWorkflow mocks the DeleteCronWorkflow method.
func (m *MockCronWorkflowServiceClient) DeleteCronWorkflow(ctx context.Context, req *cronworkflow.DeleteCronWorkflowRequest, opts ...grpc.CallOption) (*cronworkflow.CronWorkflowDeletedResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	resp, ok := args.Get(0).(*cronworkflow.CronWorkflowDeletedResponse)
	if !ok {
		return nil, args.Error(1)
	}
	return resp, args.Error(1)
}

// ResumeCronWorkflow mocks the ResumeCronWorkflow method.
func (m *MockCronWorkflowServiceClient) ResumeCronWorkflow(ctx context.Context, req *cronworkflow.CronWorkflowResumeRequest, opts ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	cw, ok := args.Get(0).(*wfv1.CronWorkflow)
	if !ok {
		return nil, args.Error(1)
	}
	return cw, args.Error(1)
}

// SuspendCronWorkflow mocks the SuspendCronWorkflow method.
func (m *MockCronWorkflowServiceClient) SuspendCronWorkflow(ctx context.Context, req *cronworkflow.CronWorkflowSuspendRequest, opts ...grpc.CallOption) (*wfv1.CronWorkflow, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	cw, ok := args.Get(0).(*wfv1.CronWorkflow)
	if !ok {
		return nil, args.Error(1)
	}
	return cw, args.Error(1)
}
