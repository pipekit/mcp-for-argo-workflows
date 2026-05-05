package tools

import (
	"errors"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestListWorkflowTemplatesTool(t *testing.T) {
	tool := ListWorkflowTemplatesTool()

	assert.Equal(t, "list_workflow_templates", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "List")
	assert.Contains(t, tool.Description, "WorkflowTemplates")
}

func TestListWorkflowTemplatesInput(t *testing.T) {
	// Test default values
	input := ListWorkflowTemplatesInput{}
	assert.Empty(t, input.Namespace)
	assert.Empty(t, input.Labels)

	// Test with values
	input2 := ListWorkflowTemplatesInput{
		Namespace: "test-namespace",
		Labels:    "app=test",
	}
	assert.Equal(t, "test-namespace", input2.Namespace)
	assert.Equal(t, "app=test", input2.Labels)
}

func TestWorkflowTemplateSummary(t *testing.T) {
	summary := WorkflowTemplateSummary{
		Name:      "test-template",
		Namespace: "default",
		CreatedAt: "2025-01-01T00:00:00Z",
	}

	assert.Equal(t, "test-template", summary.Name)
	assert.Equal(t, "default", summary.Namespace)
	assert.NotEmpty(t, summary.CreatedAt)
}

func TestListWorkflowTemplatesOutput(t *testing.T) {
	output := ListWorkflowTemplatesOutput{
		Templates: []WorkflowTemplateSummary{
			{Name: "tmpl-1", Namespace: "default"},
			{Name: "tmpl-2", Namespace: "default"},
		},
		Total: 2,
	}

	assert.Len(t, output.Templates, 2)
	assert.Equal(t, 2, output.Total)
	assert.Equal(t, "tmpl-1", output.Templates[0].Name)
	assert.Equal(t, "tmpl-2", output.Templates[1].Name)
}

// newMockWorkflowTemplateService creates a new mock workflow template service client.
func newMockWorkflowTemplateService(t *testing.T) *mocks.MockWorkflowTemplateServiceClient {
	t.Helper()
	m := &mocks.MockWorkflowTemplateServiceClient{}
	m.Test(t)
	return m
}

func TestListWorkflowTemplatesHandler(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		setupMock func(*mocks.MockWorkflowTemplateServiceClient)
		validate  func(*testing.T, *ListWorkflowTemplatesOutput, *mcp.CallToolResult)
		name      string
		input     ListWorkflowTemplatesInput
		wantErr   bool
	}{
		{
			name: "success - list templates in namespace",
			input: ListWorkflowTemplatesInput{
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("ListWorkflowTemplates", mock.Anything, mock.MatchedBy(func(req *workflowtemplate.WorkflowTemplateListRequest) bool {
					return req.Namespace == "default"
				})).Return(
					&wfv1.WorkflowTemplateList{
						Items: wfv1.WorkflowTemplates{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "template-1",
									Namespace:         "default",
									CreationTimestamp: metav1.Time{Time: testTime},
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "template-2",
									Namespace:         "default",
									CreationTimestamp: metav1.Time{Time: testTime.Add(time.Hour)},
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *ListWorkflowTemplatesOutput, result *mcp.CallToolResult) {
				assert.Equal(t, 2, output.Total)
				assert.Len(t, output.Templates, 2)
				assert.Equal(t, "template-1", output.Templates[0].Name)
				assert.Equal(t, "default", output.Templates[0].Namespace)
				assert.Equal(t, "template-2", output.Templates[1].Name)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "Found 2 workflow template(s)")
				assert.Contains(t, text.Text, "default")
			},
		},
		{
			name: "success - list templates with labels",
			input: ListWorkflowTemplatesInput{
				Namespace: "default",
				Labels:    "app=myapp",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("ListWorkflowTemplates", mock.Anything, mock.MatchedBy(func(req *workflowtemplate.WorkflowTemplateListRequest) bool {
					return req.Namespace == "default" && req.ListOptions != nil && req.ListOptions.LabelSelector == "app=myapp"
				})).Return(
					&wfv1.WorkflowTemplateList{
						Items: wfv1.WorkflowTemplates{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "template-myapp",
									Namespace:         "default",
									CreationTimestamp: metav1.Time{Time: testTime},
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *ListWorkflowTemplatesOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, 1, output.Total)
				assert.Equal(t, "template-myapp", output.Templates[0].Name)
			},
		},
		{
			name:  "success - uses default namespace",
			input: ListWorkflowTemplatesInput{},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("ListWorkflowTemplates", mock.Anything, mock.MatchedBy(func(req *workflowtemplate.WorkflowTemplateListRequest) bool {
					return req.Namespace == "argo" // default namespace from mock
				})).Return(
					&wfv1.WorkflowTemplateList{
						Items: wfv1.WorkflowTemplates{},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *ListWorkflowTemplatesOutput, result *mcp.CallToolResult) {
				assert.Equal(t, 0, output.Total)
				assert.Empty(t, output.Templates)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "Found 0 workflow template(s)")
			},
		},
		{
			name: "success - empty result",
			input: ListWorkflowTemplatesInput{
				Namespace: "empty-namespace",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("ListWorkflowTemplates", mock.Anything, mock.Anything).Return(
					&wfv1.WorkflowTemplateList{
						Items: wfv1.WorkflowTemplates{},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *ListWorkflowTemplatesOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, 0, output.Total)
				assert.Empty(t, output.Templates)
			},
		},
		{
			name: "error - API failure",
			input: ListWorkflowTemplatesInput{
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("ListWorkflowTemplates", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("connection refused"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - namespace not found",
			input: ListWorkflowTemplatesInput{
				Namespace: "nonexistent",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("ListWorkflowTemplates", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("namespaces \"nonexistent\" not found"),
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client and service
			mockClient := newMockClient(t, "argo", true)
			mockService := newMockWorkflowTemplateService(t)
			mockClient.SetWorkflowTemplateService(mockService)

			// Setup mock expectations
			tt.setupMock(mockService)

			// Verify mock expectations even on error paths
			defer mockService.AssertExpectations(t)

			// Create handler and call it
			handler := ListWorkflowTemplatesHandler(mockClient)
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
