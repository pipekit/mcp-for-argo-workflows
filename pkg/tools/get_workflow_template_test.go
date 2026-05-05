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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestGetWorkflowTemplateTool(t *testing.T) {
	tool := GetWorkflowTemplateTool()

	assert.Equal(t, "get_workflow_template", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Get")
	assert.Contains(t, tool.Description, "WorkflowTemplate")
}

func TestGetWorkflowTemplateInput(t *testing.T) {
	// Test required field
	input := GetWorkflowTemplateInput{
		Name: "my-template",
	}
	assert.Equal(t, "my-template", input.Name)
	assert.Empty(t, input.Namespace)

	// Test with namespace
	input2 := GetWorkflowTemplateInput{
		Name:      "my-template",
		Namespace: "custom-ns",
	}
	assert.Equal(t, "my-template", input2.Name)
	assert.Equal(t, "custom-ns", input2.Namespace)
}

func TestGetWorkflowTemplateOutput(t *testing.T) {
	output := GetWorkflowTemplateOutput{
		Name:       "test-template",
		Namespace:  "default",
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

	assert.Equal(t, "test-template", output.Name)
	assert.Equal(t, "default", output.Namespace)
	assert.Equal(t, "main", output.Entrypoint)
	assert.Equal(t, "2025-01-01T00:00:00Z", output.CreatedAt)
	assert.Equal(t, "test", output.Labels["app"])
	assert.Len(t, output.Arguments, 1)
	assert.Len(t, output.Templates, 1)
	assert.Equal(t, "container", output.Templates[0].Type)
}

func TestDetermineTemplateType(t *testing.T) {
	tests := []struct {
		name     string
		template *wfv1.Template
		expected string
	}{
		{
			name: "container template",
			template: &wfv1.Template{
				Container: &corev1.Container{Name: "main"},
			},
			expected: "container",
		},
		{
			name: "script template",
			template: &wfv1.Template{
				Script: &wfv1.ScriptTemplate{},
			},
			expected: "script",
		},
		{
			name: "dag template",
			template: &wfv1.Template{
				DAG: &wfv1.DAGTemplate{},
			},
			expected: "dag",
		},
		{
			name: "steps template",
			template: &wfv1.Template{
				Steps: []wfv1.ParallelSteps{},
			},
			expected: "steps",
		},
		{
			name: "resource template",
			template: &wfv1.Template{
				Resource: &wfv1.ResourceTemplate{},
			},
			expected: "resource",
		},
		{
			name: "suspend template",
			template: &wfv1.Template{
				Suspend: &wfv1.SuspendTemplate{},
			},
			expected: "suspend",
		},
		{
			name: "http template",
			template: &wfv1.Template{
				HTTP: &wfv1.HTTP{},
			},
			expected: "http",
		},
		{
			name: "plugin template",
			template: &wfv1.Template{
				Plugin: &wfv1.Plugin{},
			},
			expected: "plugin",
		},
		{
			name: "containerSet template",
			template: &wfv1.Template{
				ContainerSet: &wfv1.ContainerSetTemplate{},
			},
			expected: "containerSet",
		},
		{
			name: "data template",
			template: &wfv1.Template{
				Data: &wfv1.Data{},
			},
			expected: "data",
		},
		{
			name:     "unknown template",
			template: &wfv1.Template{},
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineTemplateType(tt.template)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetWorkflowTemplateHandler(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		setupMock func(*mocks.MockWorkflowTemplateServiceClient)
		validate  func(*testing.T, *GetWorkflowTemplateOutput, *mcp.CallToolResult)
		name      string
		errMsg    string
		input     GetWorkflowTemplateInput
		wantErr   bool
	}{
		{
			name: "success - get template with full details",
			input: GetWorkflowTemplateInput{
				Name:      "my-template",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("GetWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *workflowtemplate.WorkflowTemplateGetRequest) bool {
					return req.Name == "my-template" && req.Namespace == "default"
				})).Return(
					&wfv1.WorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "my-template",
							Namespace:         "default",
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
			validate: func(t *testing.T, output *GetWorkflowTemplateOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "my-template", output.Name)
				assert.Equal(t, "default", output.Namespace)
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
				assert.Contains(t, text.Text, "my-template")
				assert.Contains(t, text.Text, "default")
				assert.Contains(t, text.Text, "Entrypoint: main")
			},
		},
		{
			name: "success - uses default namespace",
			input: GetWorkflowTemplateInput{
				Name: "my-template",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("GetWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *workflowtemplate.WorkflowTemplateGetRequest) bool {
					return req.Name == "my-template" && req.Namespace == "argo"
				})).Return(
					&wfv1.WorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-template",
							Namespace: "argo",
						},
						Spec: wfv1.WorkflowSpec{
							Entrypoint: "main",
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetWorkflowTemplateOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "my-template", output.Name)
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "success - template with DAG",
			input: GetWorkflowTemplateInput{
				Name:      "dag-template",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("GetWorkflowTemplate", mock.Anything, mock.Anything).Return(
					&wfv1.WorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "dag-template",
							Namespace: "default",
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
			validate: func(t *testing.T, output *GetWorkflowTemplateOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "dag-template", output.Name)
				require.Len(t, output.Templates, 2)
				assert.Equal(t, "dag", output.Templates[0].Type)
				assert.Equal(t, "container", output.Templates[1].Type)
			},
		},
		{
			name: "success - template with enum parameters",
			input: GetWorkflowTemplateInput{
				Name:      "param-template",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("GetWorkflowTemplate", mock.Anything, mock.Anything).Return(
					&wfv1.WorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "param-template",
							Namespace: "default",
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
			validate: func(t *testing.T, output *GetWorkflowTemplateOutput, _ *mcp.CallToolResult) {
				require.Len(t, output.Arguments, 1)
				assert.Equal(t, "env", output.Arguments[0].Name)
				assert.Equal(t, []string{"dev", "staging", "prod"}, output.Arguments[0].Enum)
			},
		},
		{
			name: "success - minimal template",
			input: GetWorkflowTemplateInput{
				Name:      "minimal",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("GetWorkflowTemplate", mock.Anything, mock.Anything).Return(
					&wfv1.WorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "minimal",
							Namespace: "default",
						},
						Spec: wfv1.WorkflowSpec{},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetWorkflowTemplateOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "minimal", output.Name)
				assert.Empty(t, output.Entrypoint)
				assert.Empty(t, output.Arguments)
				assert.Empty(t, output.Templates)
			},
		},
		{
			name: "error - empty name",
			input: GetWorkflowTemplateInput{
				Namespace: "default",
			},
			setupMock: func(_ *mocks.MockWorkflowTemplateServiceClient) {},
			wantErr:   true,
			errMsg:    "name is required",
		},
		{
			name: "error - template not found",
			input: GetWorkflowTemplateInput{
				Name:      "nonexistent",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("GetWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("workflowtemplates.argoproj.io \"nonexistent\" not found"),
				)
			},
			wantErr: true,
			errMsg:  "not found",
		},
		{
			name: "error - API failure",
			input: GetWorkflowTemplateInput{
				Name:      "my-template",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("GetWorkflowTemplate", mock.Anything, mock.Anything).Return(
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
			mockService := newMockWorkflowTemplateService(t)
			mockClient.SetWorkflowTemplateService(mockService)

			// Setup mock expectations
			tt.setupMock(mockService)

			// Verify mock expectations even on error paths
			defer mockService.AssertExpectations(t)

			// Create handler and call it
			handler := GetWorkflowTemplateHandler(mockClient)
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
