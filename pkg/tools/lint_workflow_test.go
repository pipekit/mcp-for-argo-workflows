package tools

import (
	"errors"
	"testing"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestLintWorkflowTool(t *testing.T) {
	tool := LintWorkflowTool()

	assert.Equal(t, "lint_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Validate", "description should mention validation")
}

func TestLintWorkflowHandler(t *testing.T) {
	tests := []struct {
		setupMock func(*mocks.MockWorkflowServiceClient)
		validate  func(*testing.T, *LintWorkflowOutput, *mcp.CallToolResult)
		name      string
		input     LintWorkflowInput
		wantErr   bool
	}{
		{
			name: "success - valid workflow",
			input: LintWorkflowInput{
				Manifest:  loadTestWorkflowYAML(t, "simple_workflow.yaml"),
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("LintWorkflow", mock.Anything, mock.Anything).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							GenerateName: "hello-world-",
							Namespace:    "default",
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *LintWorkflowOutput, result *mcp.CallToolResult) {
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
			input: LintWorkflowInput{
				Manifest: loadTestWorkflowYAML(t, "simple_workflow.yaml"),
				// Namespace not specified - should use default
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("LintWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowLintRequest) bool {
					return req.Namespace == "argo" // default namespace from mock
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							GenerateName: "hello-world-",
							Namespace:    "argo",
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *LintWorkflowOutput, _ *mcp.CallToolResult) {
				assert.True(t, output.Valid)
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "validation error - invalid workflow",
			input: LintWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: invalid-workflow-
spec:
  entrypoint: nonexistent-template
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
`,
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("LintWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("spec.entrypoint \"nonexistent-template\" is not a template"),
				)
			},
			wantErr: false, // Not a handler error, just invalid workflow
			validate: func(t *testing.T, output *LintWorkflowOutput, result *mcp.CallToolResult) {
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
			input: LintWorkflowInput{
				Manifest: "",
			},
			setupMock: func(_ *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - whitespace-only manifest",
			input: LintWorkflowInput{
				Manifest: "   \n\t  ",
			},
			setupMock: func(_ *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - manifest too large",
			input: LintWorkflowInput{
				Manifest: string(make([]byte, 2*1024*1024)), // 2MB
			},
			setupMock: func(_ *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - invalid YAML",
			input: LintWorkflowInput{
				Manifest: "invalid: yaml: content:\n  - {{{",
			},
			setupMock: func(_ *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail parsing
			},
			wantErr: true,
		},
		{
			name: "error - wrong kind",
			input: LintWorkflowInput{
				Manifest: `
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
spec:
  containers:
  - name: nginx
    image: nginx:latest
`,
			},
			setupMock: func(_ *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail kind validation
			},
			wantErr: true,
		},
		{
			name: "error - WorkflowTemplate kind (not supported yet)",
			input: LintWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: my-template
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
`,
			},
			setupMock: func(_ *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail kind validation
			},
			wantErr: true,
		},
		{
			name: "error - unknown field in manifest",
			input: LintWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: test-
spec:
  unknownField: invalid
  entrypoint: main
  templates:
  - name: main
    container:
      image: alpine:latest
`,
			},
			setupMock: func(_ *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail strict parsing
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client and service
			mockClient := newMockClient(t, "argo", true)
			mockService := newMockWorkflowService(t)
			mockClient.SetWorkflowService(mockService)

			// Setup mock expectations
			tt.setupMock(mockService)

			// Verify mock expectations even on error paths
			defer mockService.AssertExpectations(t)

			// Create handler and call it
			handler := LintWorkflowHandler(mockClient)
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
