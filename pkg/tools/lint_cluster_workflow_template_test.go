package tools

import (
	"errors"
	"testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestLintClusterWorkflowTemplateTool(t *testing.T) {
	tool := LintClusterWorkflowTemplateTool()

	assert.Equal(t, "lint_cluster_workflow_template", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Validate", "description should mention validation")
}

func TestLintClusterWorkflowTemplateHandler(t *testing.T) {
	tests := []struct {
		setupMock func(*mocks.MockClusterWorkflowTemplateServiceClient)
		validate  func(*testing.T, *LintClusterWorkflowTemplateOutput, *mcp.CallToolResult)
		name      string
		input     LintClusterWorkflowTemplateInput
		wantErr   bool
	}{
		{
			name: "success - valid cluster workflow template",
			input: LintClusterWorkflowTemplateInput{
				Manifest: loadTestWorkflowYAML(t, "simple_cluster_workflow_template.yaml"),
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("LintClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					&wfv1.ClusterWorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name: "hello-world-cluster-template",
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *LintClusterWorkflowTemplateOutput, result *mcp.CallToolResult) {
				assert.True(t, output.Valid)
				assert.Equal(t, "hello-world-cluster-template", output.Name)
				assert.Empty(t, output.Errors)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "valid")
			},
		},
		{
			name: "validation error - invalid cluster workflow template",
			input: LintClusterWorkflowTemplateInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
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
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("LintClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("spec.entrypoint \"nonexistent-template\" is not a template"),
				)
			},
			wantErr: false, // Not a handler error, just invalid template
			validate: func(t *testing.T, output *LintClusterWorkflowTemplateOutput, result *mcp.CallToolResult) {
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
			input: LintClusterWorkflowTemplateInput{
				Manifest: "",
			},
			setupMock: func(_ *mocks.MockClusterWorkflowTemplateServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - whitespace-only manifest",
			input: LintClusterWorkflowTemplateInput{
				Manifest: "   \n\t  ",
			},
			setupMock: func(_ *mocks.MockClusterWorkflowTemplateServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - manifest too large",
			input: LintClusterWorkflowTemplateInput{
				Manifest: string(make([]byte, 2*1024*1024)), // 2MB
			},
			setupMock: func(_ *mocks.MockClusterWorkflowTemplateServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - invalid YAML",
			input: LintClusterWorkflowTemplateInput{
				Manifest: "invalid: yaml: content:\n  - {{{",
			},
			setupMock: func(_ *mocks.MockClusterWorkflowTemplateServiceClient) {
				// No mock needed - should fail parsing
			},
			wantErr: true,
		},
		{
			name: "error - wrong kind (Workflow)",
			input: LintClusterWorkflowTemplateInput{
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
			setupMock: func(_ *mocks.MockClusterWorkflowTemplateServiceClient) {
				// No mock needed - should fail kind validation
			},
			wantErr: true,
		},
		{
			name: "error - wrong kind (WorkflowTemplate)",
			input: LintClusterWorkflowTemplateInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: test-template
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: alpine
      command: [echo, hello]
`,
			},
			setupMock: func(_ *mocks.MockClusterWorkflowTemplateServiceClient) {
				// No mock needed - should fail kind validation
			},
			wantErr: true,
		},
		{
			name: "error - unknown field in manifest",
			input: LintClusterWorkflowTemplateInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: test-cluster-template
spec:
  unknownField: invalid
  entrypoint: main
  templates:
  - name: main
    container:
      image: alpine:latest
`,
			},
			setupMock: func(_ *mocks.MockClusterWorkflowTemplateServiceClient) {
				// No mock needed - should fail strict parsing
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
			handler := LintClusterWorkflowTemplateHandler(mockClient)
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
