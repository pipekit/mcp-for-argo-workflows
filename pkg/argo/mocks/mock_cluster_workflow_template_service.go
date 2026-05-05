// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockClusterWorkflowTemplateServiceClient is a mock implementation of clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient.
type MockClusterWorkflowTemplateServiceClient struct {
	mock.Mock
}

// Ensure MockClusterWorkflowTemplateServiceClient implements the interface.
var _ clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient = (*MockClusterWorkflowTemplateServiceClient)(nil)

// CreateClusterWorkflowTemplate mocks the CreateClusterWorkflowTemplate method.
func (m *MockClusterWorkflowTemplateServiceClient) CreateClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtemplate.ClusterWorkflowTemplateCreateRequest, opts ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	cwft, ok := args.Get(0).(*wfv1.ClusterWorkflowTemplate)
	if !ok {
		return nil, args.Error(1)
	}
	return cwft, args.Error(1)
}

// GetClusterWorkflowTemplate mocks the GetClusterWorkflowTemplate method.
func (m *MockClusterWorkflowTemplateServiceClient) GetClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtemplate.ClusterWorkflowTemplateGetRequest, opts ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	cwft, ok := args.Get(0).(*wfv1.ClusterWorkflowTemplate)
	if !ok {
		return nil, args.Error(1)
	}
	return cwft, args.Error(1)
}

// ListClusterWorkflowTemplates mocks the ListClusterWorkflowTemplates method.
func (m *MockClusterWorkflowTemplateServiceClient) ListClusterWorkflowTemplates(ctx context.Context, req *clusterworkflowtemplate.ClusterWorkflowTemplateListRequest, opts ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplateList, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	cwftList, ok := args.Get(0).(*wfv1.ClusterWorkflowTemplateList)
	if !ok {
		return nil, args.Error(1)
	}
	return cwftList, args.Error(1)
}

// UpdateClusterWorkflowTemplate mocks the UpdateClusterWorkflowTemplate method.
func (m *MockClusterWorkflowTemplateServiceClient) UpdateClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtemplate.ClusterWorkflowTemplateUpdateRequest, opts ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	cwft, ok := args.Get(0).(*wfv1.ClusterWorkflowTemplate)
	if !ok {
		return nil, args.Error(1)
	}
	return cwft, args.Error(1)
}

// DeleteClusterWorkflowTemplate mocks the DeleteClusterWorkflowTemplate method.
func (m *MockClusterWorkflowTemplateServiceClient) DeleteClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtemplate.ClusterWorkflowTemplateDeleteRequest, opts ...grpc.CallOption) (*clusterworkflowtemplate.ClusterWorkflowTemplateDeleteResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	resp, ok := args.Get(0).(*clusterworkflowtemplate.ClusterWorkflowTemplateDeleteResponse)
	if !ok {
		return nil, args.Error(1)
	}
	return resp, args.Error(1)
}

// LintClusterWorkflowTemplate mocks the LintClusterWorkflowTemplate method.
func (m *MockClusterWorkflowTemplateServiceClient) LintClusterWorkflowTemplate(ctx context.Context, req *clusterworkflowtemplate.ClusterWorkflowTemplateLintRequest, opts ...grpc.CallOption) (*wfv1.ClusterWorkflowTemplate, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	cwft, ok := args.Get(0).(*wfv1.ClusterWorkflowTemplate)
	if !ok {
		return nil, args.Error(1)
	}
	return cwft, args.Error(1)
}
