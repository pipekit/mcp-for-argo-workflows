// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/info"
	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
	"github.com/stretchr/testify/mock"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// MockClient is a mock implementation of the Argo client.
type MockClient struct {
	mock.Mock
	workflowService                workflow.WorkflowServiceClient
	workflowTemplateService        workflowtemplate.WorkflowTemplateServiceClient
	clusterWorkflowTemplateService clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient
	cronWorkflowService            cronworkflow.CronWorkflowServiceClient
	archivedWorkflowService        workflowarchive.ArchivedWorkflowServiceClient
	// ctx mirrors the real Client's context field for testing.
	ctx            context.Context //nolint:containedctx // Mirrors real Client's Argo SDK pattern
	namespace      string
	argoServerMode bool
}

// Ensure MockClient implements argo.ClientInterface.
var _ argo.ClientInterface = (*MockClient)(nil)

// NewMockClient creates a new mock client with default settings.
func NewMockClient(namespace string, argoServerMode bool) *MockClient {
	return &MockClient{
		namespace:      namespace,
		argoServerMode: argoServerMode,
	}
}

// SetWorkflowService sets the workflow service client for this mock.
func (m *MockClient) SetWorkflowService(service workflow.WorkflowServiceClient) {
	m.workflowService = service
}

// SetWorkflowTemplateService sets the workflow template service client for this mock.
func (m *MockClient) SetWorkflowTemplateService(service workflowtemplate.WorkflowTemplateServiceClient) {
	m.workflowTemplateService = service
}

// SetClusterWorkflowTemplateService sets the cluster workflow template service client for this mock.
func (m *MockClient) SetClusterWorkflowTemplateService(service clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient) {
	m.clusterWorkflowTemplateService = service
}

// SetCronWorkflowService sets the cron workflow service client for this mock.
func (m *MockClient) SetCronWorkflowService(service cronworkflow.CronWorkflowServiceClient) {
	m.cronWorkflowService = service
}

// SetArchivedWorkflowService sets the archived workflow service client for this mock.
func (m *MockClient) SetArchivedWorkflowService(service workflowarchive.ArchivedWorkflowServiceClient) {
	m.archivedWorkflowService = service
}

// SetContext sets the context for this mock client.
func (m *MockClient) SetContext(ctx context.Context) {
	m.ctx = ctx
}

// WorkflowService returns the workflow service client.
func (m *MockClient) WorkflowService() workflow.WorkflowServiceClient {
	return m.workflowService
}

// CronWorkflowService returns the cron workflow service client.
func (m *MockClient) CronWorkflowService() (cronworkflow.CronWorkflowServiceClient, error) {
	if m.cronWorkflowService != nil {
		return m.cronWorkflowService, nil
	}
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	svc, ok := args.Get(0).(cronworkflow.CronWorkflowServiceClient)
	if !ok {
		return nil, args.Error(1)
	}
	return svc, args.Error(1)
}

// WorkflowTemplateService returns the workflow template service client.
func (m *MockClient) WorkflowTemplateService() (workflowtemplate.WorkflowTemplateServiceClient, error) {
	if m.workflowTemplateService != nil {
		return m.workflowTemplateService, nil
	}
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	svc, ok := args.Get(0).(workflowtemplate.WorkflowTemplateServiceClient)
	if !ok {
		return nil, args.Error(1)
	}
	return svc, args.Error(1)
}

// ClusterWorkflowTemplateService returns the cluster workflow template service client.
func (m *MockClient) ClusterWorkflowTemplateService() (clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient, error) {
	if m.clusterWorkflowTemplateService != nil {
		return m.clusterWorkflowTemplateService, nil
	}
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	svc, ok := args.Get(0).(clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient)
	if !ok {
		return nil, args.Error(1)
	}
	return svc, args.Error(1)
}

// ArchivedWorkflowService returns the archived workflow service client.
func (m *MockClient) ArchivedWorkflowService() (workflowarchive.ArchivedWorkflowServiceClient, error) {
	if m.archivedWorkflowService != nil {
		return m.archivedWorkflowService, nil
	}
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	svc, ok := args.Get(0).(workflowarchive.ArchivedWorkflowServiceClient)
	if !ok {
		return nil, args.Error(1)
	}
	return svc, args.Error(1)
}

// InfoService returns the info service client (stub).
func (m *MockClient) InfoService() (info.InfoServiceClient, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	svc, ok := args.Get(0).(info.InfoServiceClient)
	if !ok {
		return nil, args.Error(1)
	}
	return svc, args.Error(1)
}

// IsArgoServerMode returns whether this client is in Argo Server mode.
func (m *MockClient) IsArgoServerMode() bool {
	return m.argoServerMode
}

// DefaultNamespace returns the default namespace configured for this client.
func (m *MockClient) DefaultNamespace() string {
	return m.namespace
}

// Context returns the context associated with this client.
func (m *MockClient) Context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}
