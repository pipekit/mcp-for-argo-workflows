package tools

import (
	"errors"
	"testing"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestLintCronWorkflowTool(t *testing.T) {
	tool := LintCronWorkflowTool()

	assert.Equal(t, "lint_cron_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Validate", "description should mention validation")
}

func TestLintCronWorkflowHandler(t *testing.T) {
	tests := []struct {
		setupMock func(*mocks.MockCronWorkflowServiceClient)
		validate  func(*testing.T, *LintCronWorkflowOutput, *mcp.CallToolResult)
		name      string
		input     LintCronWorkflowInput
		wantErr   bool
	}{
		{
			name: "success - valid cron workflow",
			input: LintCronWorkflowInput{
				Manifest:  loadTestWorkflowYAML(t, "simple_cron_workflow.yaml"),
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("LintCronWorkflow", mock.Anything, mock.Anything).Return(
					&wfv1.CronWorkflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "hello-world-cron",
							Namespace: "default",
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *LintCronWorkflowOutput, result *mcp.CallToolResult) {
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
			input: LintCronWorkflowInput{
				Manifest: loadTestWorkflowYAML(t, "simple_cron_workflow.yaml"),
				// Namespace not specified - should use default
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("LintCronWorkflow", mock.Anything, mock.MatchedBy(func(req *cronworkflow.LintCronWorkflowRequest) bool {
					return req.Namespace == "argo" // default namespace from mock
				})).Return(
					&wfv1.CronWorkflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "hello-world-cron",
							Namespace: "argo",
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *LintCronWorkflowOutput, _ *mcp.CallToolResult) {
				assert.True(t, output.Valid)
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "validation error - invalid cron workflow",
			input: LintCronWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: invalid-cron
spec:
  schedules:
    - "0 * * * *"
  workflowSpec:
    entrypoint: nonexistent-template
    templates:
    - name: whalesay
      container:
        image: docker/whalesay:latest
`,
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("LintCronWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("spec.workflowSpec.entrypoint \"nonexistent-template\" is not a template"),
				)
			},
			wantErr: false, // Not a handler error, just invalid cron workflow
			validate: func(t *testing.T, output *LintCronWorkflowOutput, result *mcp.CallToolResult) {
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
			input: LintCronWorkflowInput{
				Manifest: "",
			},
			setupMock: func(_ *mocks.MockCronWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - whitespace-only manifest",
			input: LintCronWorkflowInput{
				Manifest: "   \n\t  ",
			},
			setupMock: func(_ *mocks.MockCronWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - manifest too large",
			input: LintCronWorkflowInput{
				Manifest: string(make([]byte, 2*1024*1024)), // 2MB
			},
			setupMock: func(_ *mocks.MockCronWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - invalid YAML",
			input: LintCronWorkflowInput{
				Manifest: "invalid: yaml: content:\n  - {{{",
			},
			setupMock: func(_ *mocks.MockCronWorkflowServiceClient) {
				// No mock needed - should fail parsing
			},
			wantErr: true,
		},
		{
			name: "error - wrong kind (Workflow)",
			input: LintCronWorkflowInput{
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
			setupMock: func(_ *mocks.MockCronWorkflowServiceClient) {
				// No mock needed - should fail kind validation
			},
			wantErr: true,
		},
		{
			name: "error - wrong kind (WorkflowTemplate)",
			input: LintCronWorkflowInput{
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
			setupMock: func(_ *mocks.MockCronWorkflowServiceClient) {
				// No mock needed - should fail kind validation
			},
			wantErr: true,
		},
		{
			name: "error - unknown field in manifest",
			input: LintCronWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: test-cron
spec:
  unknownField: invalid
  schedules:
    - "0 * * * *"
  workflowSpec:
    entrypoint: main
    templates:
    - name: main
      container:
        image: alpine:latest
`,
			},
			setupMock: func(_ *mocks.MockCronWorkflowServiceClient) {
				// No mock needed - should fail strict parsing
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client and service
			mockClient := newMockClient(t, "argo", true)
			mockService := newMockCronWorkflowService(t)
			mockClient.SetCronWorkflowService(mockService)

			// Setup mock expectations
			tt.setupMock(mockService)

			// Verify mock expectations even on error paths
			defer mockService.AssertExpectations(t)

			// Create handler and call it
			handler := LintCronWorkflowHandler(mockClient)
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
