// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockWorkflowServiceClient is a mock implementation of workflow.WorkflowServiceClient.
type MockWorkflowServiceClient struct {
	mock.Mock
}

// CreateWorkflow mocks the CreateWorkflow method.
func (m *MockWorkflowServiceClient) CreateWorkflow(ctx context.Context, req *workflow.WorkflowCreateRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
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

// GetWorkflow mocks the GetWorkflow method.
func (m *MockWorkflowServiceClient) GetWorkflow(ctx context.Context, req *workflow.WorkflowGetRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
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

// ListWorkflows mocks the ListWorkflows method.
func (m *MockWorkflowServiceClient) ListWorkflows(ctx context.Context, req *workflow.WorkflowListRequest, opts ...grpc.CallOption) (*wfv1.WorkflowList, error) {
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

// WatchWorkflows mocks the WatchWorkflows method (stub for completeness).
func (m *MockWorkflowServiceClient) WatchWorkflows(ctx context.Context, req *workflow.WatchWorkflowsRequest, opts ...grpc.CallOption) (workflow.WorkflowService_WatchWorkflowsClient, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	client, ok := args.Get(0).(workflow.WorkflowService_WatchWorkflowsClient)
	if !ok {
		return nil, args.Error(1)
	}
	return client, args.Error(1)
}

// WatchEvents mocks the WatchEvents method (stub for completeness).
func (m *MockWorkflowServiceClient) WatchEvents(ctx context.Context, req *workflow.WatchEventsRequest, opts ...grpc.CallOption) (workflow.WorkflowService_WatchEventsClient, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	client, ok := args.Get(0).(workflow.WorkflowService_WatchEventsClient)
	if !ok {
		return nil, args.Error(1)
	}
	return client, args.Error(1)
}

// DeleteWorkflow mocks the DeleteWorkflow method (stub for future use).
func (m *MockWorkflowServiceClient) DeleteWorkflow(ctx context.Context, req *workflow.WorkflowDeleteRequest, opts ...grpc.CallOption) (*workflow.WorkflowDeleteResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	resp, ok := args.Get(0).(*workflow.WorkflowDeleteResponse)
	if !ok {
		return nil, args.Error(1)
	}
	return resp, args.Error(1)
}

// LintWorkflow mocks the LintWorkflow method (stub for future use).
func (m *MockWorkflowServiceClient) LintWorkflow(ctx context.Context, req *workflow.WorkflowLintRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
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

// SubmitWorkflow mocks the SubmitWorkflow method (stub for future use).
func (m *MockWorkflowServiceClient) SubmitWorkflow(ctx context.Context, req *workflow.WorkflowSubmitRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
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

// SuspendWorkflow mocks the SuspendWorkflow method (stub for future use).
func (m *MockWorkflowServiceClient) SuspendWorkflow(ctx context.Context, req *workflow.WorkflowSuspendRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
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

// ResumeWorkflow mocks the ResumeWorkflow method (stub for future use).
func (m *MockWorkflowServiceClient) ResumeWorkflow(ctx context.Context, req *workflow.WorkflowResumeRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
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

// TerminateWorkflow mocks the TerminateWorkflow method (stub for future use).
func (m *MockWorkflowServiceClient) TerminateWorkflow(ctx context.Context, req *workflow.WorkflowTerminateRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
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

// StopWorkflow mocks the StopWorkflow method (stub for future use).
func (m *MockWorkflowServiceClient) StopWorkflow(ctx context.Context, req *workflow.WorkflowStopRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
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

// SetWorkflow mocks the SetWorkflow method (stub for future use).
func (m *MockWorkflowServiceClient) SetWorkflow(ctx context.Context, req *workflow.WorkflowSetRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
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

// RetryWorkflow mocks the RetryWorkflow method (stub for future use).
func (m *MockWorkflowServiceClient) RetryWorkflow(ctx context.Context, req *workflow.WorkflowRetryRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
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

// ResubmitWorkflow mocks the ResubmitWorkflow method (stub for future use).
func (m *MockWorkflowServiceClient) ResubmitWorkflow(ctx context.Context, req *workflow.WorkflowResubmitRequest, opts ...grpc.CallOption) (*wfv1.Workflow, error) {
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

// WorkflowLogs mocks the WorkflowLogs method (stub for future use).
func (m *MockWorkflowServiceClient) WorkflowLogs(ctx context.Context, req *workflow.WorkflowLogRequest, opts ...grpc.CallOption) (workflow.WorkflowService_WorkflowLogsClient, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	client, ok := args.Get(0).(workflow.WorkflowService_WorkflowLogsClient)
	if !ok {
		return nil, args.Error(1)
	}
	return client, args.Error(1)
}

// PodLogs mocks the PodLogs method (stub for future use).
func (m *MockWorkflowServiceClient) PodLogs(ctx context.Context, req *workflow.WorkflowLogRequest, opts ...grpc.CallOption) (workflow.WorkflowService_PodLogsClient, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	client, ok := args.Get(0).(workflow.WorkflowService_PodLogsClient)
	if !ok {
		return nil, args.Error(1)
	}
	return client, args.Error(1)
}
