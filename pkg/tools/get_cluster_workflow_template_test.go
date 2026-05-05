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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestGetClusterWorkflowTemplateTool(t *testing.T) {
	tool := GetClusterWorkflowTemplateTool()

	assert.Equal(t, "get_cluster_workflow_template", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Get")
	assert.Contains(t, tool.Description, "ClusterWorkflowTemplate")
}

func TestGetClusterWorkflowTemplateInput(t *testing.T) {
	// Test required field
	input := GetClusterWorkflowTemplateInput{
		Name: "my-cluster-template",
	}
	assert.Equal(t, "my-cluster-template", input.Name)
}

func TestGetClusterWorkflowTemplateOutput(t *testing.T) {
	output := GetClusterWorkflowTemplateOutput{
		Name:       "test-cluster-template",
		CreatedAt:  "2025-01-01T00:00:00Z",
		Entrypoint: "main",
		Labels:     map[string]string{"app": "test"},
		Arguments: []TemplateParameterInfo{
			{Name: "message", Default: "hello"},
		},
		Templates: []TemplateInfo{
			{Name: "main", Type: "container"},
		},
	}

	assert.Equal(t, "test-cluster-template", output.Name)
	assert.Equal(t, "main", output.Entrypoint)
	assert.Equal(t, "2025-01-01T00:00:00Z", output.CreatedAt)
	assert.Equal(t, "test", output.Labels["app"])
	assert.Len(t, output.Arguments, 1)
	assert.Len(t, output.Templates, 1)
	assert.Equal(t, "container", output.Templates[0].Type)
}

func TestGetClusterWorkflowTemplateHandler(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		setupMock func(*mocks.MockClusterWorkflowTemplateServiceClient)
		validate  func(*testing.T, *GetClusterWorkflowTemplateOutput, *mcp.CallToolResult)
		name      string
		errMsg    string
		input     GetClusterWorkflowTemplateInput
		wantErr   bool
	}{
		{
			name: "success - get template with full details",
			input: GetClusterWorkflowTemplateInput{
				Name: "my-cluster-template",
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("GetClusterWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *clusterworkflowtemplate.ClusterWorkflowTemplateGetRequest) bool {
					return req.Name == "my-cluster-template"
				})).Return(
					&wfv1.ClusterWorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "my-cluster-template",
							CreationTimestamp: metav1.Time{Time: testTime},
							Labels:            map[string]string{"app": "test"},
							Annotations:       map[string]string{"description": "test template"},
						},
						Spec: wfv1.WorkflowSpec{
							Entrypoint: "main",
							Arguments: wfv1.Arguments{
								Parameters: []wfv1.Parameter{
									{
										Name:        "message",
										Default:     wfv1.AnyStringPtr("hello"),
										Description: wfv1.AnyStringPtr("The message to print"),
									},
								},
							},
							Templates: []wfv1.Template{
								{
									Name:      "main",
									Container: &corev1.Container{Name: "main", Image: "alpine"},
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetClusterWorkflowTemplateOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "my-cluster-template", output.Name)
				assert.Equal(t, "main", output.Entrypoint)
				assert.NotEmpty(t, output.CreatedAt)
				assert.Equal(t, "test", output.Labels["app"])
				assert.Equal(t, "test template", output.Annotations["description"])

				require.Len(t, output.Arguments, 1)
				assert.Equal(t, "message", output.Arguments[0].Name)
				assert.Equal(t, "hello", output.Arguments[0].Default)
				assert.Equal(t, "The message to print", output.Arguments[0].Description)

				require.Len(t, output.Templates, 1)
				assert.Equal(t, "main", output.Templates[0].Name)
				assert.Equal(t, "container", output.Templates[0].Type)

				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "my-cluster-template")
				assert.Contains(t, text.Text, "Entrypoint: main")
			},
		},
		{
			name: "success - template with DAG",
			input: GetClusterWorkflowTemplateInput{
				Name: "dag-cluster-template",
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("GetClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					&wfv1.ClusterWorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name: "dag-cluster-template",
						},
						Spec: wfv1.WorkflowSpec{
							Entrypoint: "diamond",
							Templates: []wfv1.Template{
								{
									Name: "diamond",
									DAG: &wfv1.DAGTemplate{
										Tasks: []wfv1.DAGTask{
											{Name: "A"},
											{Name: "B", Dependencies: []string{"A"}},
										},
									},
								},
								{
									Name:      "task",
									Container: &corev1.Container{Name: "task"},
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetClusterWorkflowTemplateOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "dag-cluster-template", output.Name)
				require.Len(t, output.Templates, 2)
				assert.Equal(t, "dag", output.Templates[0].Type)
				assert.Equal(t, "container", output.Templates[1].Type)
			},
		},
		{
			name: "success - template with enum parameters",
			input: GetClusterWorkflowTemplateInput{
				Name: "param-cluster-template",
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("GetClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					&wfv1.ClusterWorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name: "param-cluster-template",
						},
						Spec: wfv1.WorkflowSpec{
							Arguments: wfv1.Arguments{
								Parameters: []wfv1.Parameter{
									{
										Name: "env",
										Enum: []wfv1.AnyString{"dev", "staging", "prod"},
									},
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetClusterWorkflowTemplateOutput, _ *mcp.CallToolResult) {
				require.Len(t, output.Arguments, 1)
				assert.Equal(t, "env", output.Arguments[0].Name)
				assert.Equal(t, []string{"dev", "staging", "prod"}, output.Arguments[0].Enum)
			},
		},
		{
			name: "success - minimal template",
			input: GetClusterWorkflowTemplateInput{
				Name: "minimal",
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("GetClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					&wfv1.ClusterWorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name: "minimal",
						},
						Spec: wfv1.WorkflowSpec{},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetClusterWorkflowTemplateOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "minimal", output.Name)
				assert.Empty(t, output.Entrypoint)
				assert.Empty(t, output.Arguments)
				assert.Empty(t, output.Templates)
			},
		},
		{
			name: "error - empty name",
			input: GetClusterWorkflowTemplateInput{
				Name: "",
			},
			setupMock: func(_ *mocks.MockClusterWorkflowTemplateServiceClient) {},
			wantErr:   true,
			errMsg:    "name is required",
		},
		{
			name: "error - template not found",
			input: GetClusterWorkflowTemplateInput{
				Name: "nonexistent",
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("GetClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("clusterworkflowtemplates.argoproj.io \"nonexistent\" not found"),
				)
			},
			wantErr: true,
			errMsg:  "not found",
		},
		{
			name: "error - API failure",
			input: GetClusterWorkflowTemplateInput{
				Name: "my-cluster-template",
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("GetClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("connection refused"),
				)
			},
			wantErr: true,
			errMsg:  "connection refused",
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
			handler := GetClusterWorkflowTemplateHandler(mockClient)
			ctx := t.Context()
			req := &mcp.CallToolRequest{}

			result, output, err := handler(ctx, req, tt.input)

			// Validate results
			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, output)
			tt.validate(t, output, result)
		})
	}
}
