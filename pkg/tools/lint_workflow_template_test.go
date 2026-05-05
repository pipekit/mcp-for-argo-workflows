package tools

import (
	"errors"
	"testing"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestLintWorkflowTemplateTool(t *testing.T) {
	tool := LintWorkflowTemplateTool()

	assert.Equal(t, "lint_workflow_template", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Validate", "description should mention validation")
}

func TestLintWorkflowTemplateHandler(t *testing.T) {
	tests := []struct {
		setupMock func(*mocks.MockWorkflowTemplateServiceClient)
		validate  func(*testing.T, *LintWorkflowTemplateOutput, *mcp.CallToolResult)
		name      string
		input     LintWorkflowTemplateInput
		wantErr   bool
	}{
		{
			name: "success - valid workflow template",
			input: LintWorkflowTemplateInput{
				Manifest:  loadTestWorkflowYAML(t, "simple_workflow_template.yaml"),
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("LintWorkflowTemplate", mock.Anything, mock.Anything).Return(
					&wfv1.WorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "hello-world-template",
							Namespace: "default",
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *LintWorkflowTemplateOutput, result *mcp.CallToolResult) {
				assert.True(t, output.Valid)
				assert.Equal(t, "default", output.Namespace)
				assert.Empty(t, output.Errors)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "valid")
			},
		},
		{
			name: "success - uses default namespace",
			input: LintWorkflowTemplateInput{
				Manifest: loadTestWorkflowYAML(t, "simple_workflow_template.yaml"),
				// Namespace not specified - should use default
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("LintWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *workflowtemplate.WorkflowTemplateLintRequest) bool {
					return req.Namespace == "argo" // default namespace from mock
				})).Return(
					&wfv1.WorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "hello-world-template",
							Namespace: "argo",
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *LintWorkflowTemplateOutput, _ *mcp.CallToolResult) {
				assert.True(t, output.Valid)
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "validation error - invalid workflow template",
			input: LintWorkflowTemplateInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: invalid-template
spec:
  entrypoint: nonexistent-template
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
`,
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("LintWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("spec.entrypoint \"nonexistent-template\" is not a template"),
				)
			},
			wantErr: false, // Not a handler error, just invalid template
			validate: func(t *testing.T, output *LintWorkflowTemplateOutput, result *mcp.CallToolResult) {
				assert.False(t, output.Valid)
				require.Len(t, output.Errors, 1)
				assert.Contains(t, output.Errors[0], "nonexistent-template")
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "validation failed")
			},
		},
		{
			name: "error - empty manifest",
			input: LintWorkflowTemplateInput{
				Manifest: "",
			},
			setupMock: func(_ *mocks.MockWorkflowTemplateServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - whitespace-only manifest",
			input: LintWorkflowTemplateInput{
				Manifest: "   \n\t  ",
			},
			setupMock: func(_ *mocks.MockWorkflowTemplateServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - manifest too large",
			input: LintWorkflowTemplateInput{
				Manifest: string(make([]byte, 2*1024*1024)), // 2MB
			},
			setupMock: func(_ *mocks.MockWorkflowTemplateServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - invalid YAML",
			input: LintWorkflowTemplateInput{
				Manifest: "invalid: yaml: content:\n  - {{{",
			},
			setupMock: func(_ *mocks.MockWorkflowTemplateServiceClient) {
				// No mock needed - should fail parsing
			},
			wantErr: true,
		},
		{
			name: "error - wrong kind (Workflow)",
			input: LintWorkflowTemplateInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: test-workflow
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: alpine
      command: [echo, hello]
`,
			},
			setupMock: func(_ *mocks.MockWorkflowTemplateServiceClient) {
				// No mock needed - should fail kind validation
			},
			wantErr: true,
		},
		{
			name: "error - wrong kind (CronWorkflow)",
			input: LintWorkflowTemplateInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron
spec:
  schedules:
    - "* * * * *"
  workflowSpec:
    entrypoint: main
    templates:
    - name: main
      container:
        image: alpine
        command: [echo, hello]
`,
			},
			setupMock: func(_ *mocks.MockWorkflowTemplateServiceClient) {
				// No mock needed - should fail kind validation
			},
			wantErr: true,
		},
		{
			name: "error - unknown field in manifest",
			input: LintWorkflowTemplateInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: test-template
spec:
  unknownField: invalid
  entrypoint: main
  templates:
  - name: main
    container:
      image: alpine:latest
`,
			},
			setupMock: func(_ *mocks.MockWorkflowTemplateServiceClient) {
				// No mock needed - should fail strict parsing
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
			handler := LintWorkflowTemplateHandler(mockClient)
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
