package tools

import (
	"errors"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestListClusterWorkflowTemplatesTool(t *testing.T) {
	tool := ListClusterWorkflowTemplatesTool()

	assert.Equal(t, "list_cluster_workflow_templates", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "List")
	assert.Contains(t, tool.Description, "ClusterWorkflowTemplates")
}

func TestListClusterWorkflowTemplatesInput(t *testing.T) {
	// Test default values
	input := ListClusterWorkflowTemplatesInput{}
	assert.Empty(t, input.Labels)

	// Test with values
	input2 := ListClusterWorkflowTemplatesInput{
		Labels: "app=test",
	}
	assert.Equal(t, "app=test", input2.Labels)
}

func TestClusterWorkflowTemplateSummary(t *testing.T) {
	summary := ClusterWorkflowTemplateSummary{
		Name:      "test-template",
		CreatedAt: "2025-01-01T00:00:00Z",
	}

	assert.Equal(t, "test-template", summary.Name)
	assert.NotEmpty(t, summary.CreatedAt)
}

func TestListClusterWorkflowTemplatesOutput(t *testing.T) {
	output := ListClusterWorkflowTemplatesOutput{
		Templates: []ClusterWorkflowTemplateSummary{
			{Name: "tmpl-1"},
			{Name: "tmpl-2"},
		},
		Total: 2,
	}

	assert.Len(t, output.Templates, 2)
	assert.Equal(t, 2, output.Total)
	assert.Equal(t, "tmpl-1", output.Templates[0].Name)
	assert.Equal(t, "tmpl-2", output.Templates[1].Name)
}

// newMockClusterWorkflowTemplateService creates a new mock cluster workflow template service client.
func newMockClusterWorkflowTemplateService(t *testing.T) *mocks.MockClusterWorkflowTemplateServiceClient {
	t.Helper()
	m := &mocks.MockClusterWorkflowTemplateServiceClient{}
	m.Test(t)
	return m
}

func TestListClusterWorkflowTemplatesHandler(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		setupMock func(*mocks.MockClusterWorkflowTemplateServiceClient)
		validate  func(*testing.T, *ListClusterWorkflowTemplatesOutput, *mcp.CallToolResult)
		name      string
		input     ListClusterWorkflowTemplatesInput
		wantErr   bool
	}{
		{
			name:  "success - list all templates",
			input: ListClusterWorkflowTemplatesInput{},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("ListClusterWorkflowTemplates", mock.Anything, mock.Anything).Return(
					&wfv1.ClusterWorkflowTemplateList{
						Items: []wfv1.ClusterWorkflowTemplate{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "cluster-template-1",
									CreationTimestamp: metav1.Time{Time: testTime},
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "cluster-template-2",
									CreationTimestamp: metav1.Time{Time: testTime.Add(time.Hour)},
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *ListClusterWorkflowTemplatesOutput, result *mcp.CallToolResult) {
				assert.Equal(t, 2, output.Total)
				assert.Len(t, output.Templates, 2)
				assert.Equal(t, "cluster-template-1", output.Templates[0].Name)
				assert.Equal(t, "cluster-template-2", output.Templates[1].Name)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "Found 2 cluster workflow template(s)")
			},
		},
		{
			name: "success - list templates with labels",
			input: ListClusterWorkflowTemplatesInput{
				Labels: "app=myapp",
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("ListClusterWorkflowTemplates", mock.Anything, mock.MatchedBy(func(req *clusterworkflowtemplate.ClusterWorkflowTemplateListRequest) bool {
					return req.ListOptions != nil && req.ListOptions.LabelSelector == "app=myapp"
				})).Return(
					&wfv1.ClusterWorkflowTemplateList{
						Items: []wfv1.ClusterWorkflowTemplate{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "template-myapp",
									CreationTimestamp: metav1.Time{Time: testTime},
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *ListClusterWorkflowTemplatesOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, 1, output.Total)
				assert.Equal(t, "template-myapp", output.Templates[0].Name)
			},
		},
		{
			name:  "success - empty result",
			input: ListClusterWorkflowTemplatesInput{},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("ListClusterWorkflowTemplates", mock.Anything, mock.Anything).Return(
					&wfv1.ClusterWorkflowTemplateList{
						Items: []wfv1.ClusterWorkflowTemplate{},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *ListClusterWorkflowTemplatesOutput, result *mcp.CallToolResult) {
				assert.Equal(t, 0, output.Total)
				assert.Empty(t, output.Templates)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "Found 0 cluster workflow template(s)")
			},
		},
		{
			name:  "error - API failure",
			input: ListClusterWorkflowTemplatesInput{},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("ListClusterWorkflowTemplates", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("connection refused"),
				)
			},
			wantErr: true,
		},
		{
			name:  "error - permission denied",
			input: ListClusterWorkflowTemplatesInput{},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("ListClusterWorkflowTemplates", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("user does not have permission to list cluster workflow templates"),
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client and service
			mockClient := newMockClient(t, "argo", true)
			mockService := newMockClusterWorkflowTemplateService(t)
			mockClient.SetClusterWorkflowTemplateService(mockService)

			// Setup mock expectations
			tt.setupMock(mockService)

			// Verify mock expectations even on error paths
			defer mockService.AssertExpectations(t)

			// Create handler and call it
			handler := ListClusterWorkflowTemplatesHandler(mockClient)
			ctx := t.Context()
			req := &mcp.CallToolRequest{}

			result, output, err := handler(ctx, req, tt.input)

			// Validate results
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, output)
			tt.validate(t, output, result)
		})
	}
}
