// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockWorkflowTemplateServiceClient is a mock implementation of workflowtemplate.WorkflowTemplateServiceClient.
type MockWorkflowTemplateServiceClient struct {
	mock.Mock
}

// Ensure MockWorkflowTemplateServiceClient implements the interface.
var _ workflowtemplate.WorkflowTemplateServiceClient = (*MockWorkflowTemplateServiceClient)(nil)

// CreateWorkflowTemplate mocks the CreateWorkflowTemplate method.
func (m *MockWorkflowTemplateServiceClient) CreateWorkflowTemplate(ctx context.Context, req *workflowtemplate.WorkflowTemplateCreateRequest, opts ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	wft, ok := args.Get(0).(*wfv1.WorkflowTemplate)
	if !ok {
		return nil, args.Error(1)
	}
	return wft, args.Error(1)
}

// GetWorkflowTemplate mocks the GetWorkflowTemplate method.
func (m *MockWorkflowTemplateServiceClient) GetWorkflowTemplate(ctx context.Context, req *workflowtemplate.WorkflowTemplateGetRequest, opts ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	wft, ok := args.Get(0).(*wfv1.WorkflowTemplate)
	if !ok {
		return nil, args.Error(1)
	}
	return wft, args.Error(1)
}

// ListWorkflowTemplates mocks the ListWorkflowTemplates method.
func (m *MockWorkflowTemplateServiceClient) ListWorkflowTemplates(ctx context.Context, req *workflowtemplate.WorkflowTemplateListRequest, opts ...grpc.CallOption) (*wfv1.WorkflowTemplateList, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	wftList, ok := args.Get(0).(*wfv1.WorkflowTemplateList)
	if !ok {
		return nil, args.Error(1)
	}
	return wftList, args.Error(1)
}

// UpdateWorkflowTemplate mocks the UpdateWorkflowTemplate method.
func (m *MockWorkflowTemplateServiceClient) UpdateWorkflowTemplate(ctx context.Context, req *workflowtemplate.WorkflowTemplateUpdateRequest, opts ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	wft, ok := args.Get(0).(*wfv1.WorkflowTemplate)
	if !ok {
		return nil, args.Error(1)
	}
	return wft, args.Error(1)
}

// DeleteWorkflowTemplate mocks the DeleteWorkflowTemplate method.
func (m *MockWorkflowTemplateServiceClient) DeleteWorkflowTemplate(ctx context.Context, req *workflowtemplate.WorkflowTemplateDeleteRequest, opts ...grpc.CallOption) (*workflowtemplate.WorkflowTemplateDeleteResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	resp, ok := args.Get(0).(*workflowtemplate.WorkflowTemplateDeleteResponse)
	if !ok {
		return nil, args.Error(1)
	}
	return resp, args.Error(1)
}

// LintWorkflowTemplate mocks the LintWorkflowTemplate method.
func (m *MockWorkflowTemplateServiceClient) LintWorkflowTemplate(ctx context.Context, req *workflowtemplate.WorkflowTemplateLintRequest, opts ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	wft, ok := args.Get(0).(*wfv1.WorkflowTemplate)
	if !ok {
		return nil, args.Error(1)
	}
	return wft, args.Error(1)
}
