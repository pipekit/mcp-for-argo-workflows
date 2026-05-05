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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestCreateWorkflowTemplateTool(t *testing.T) {
	tool := CreateWorkflowTemplateTool()

	assert.Equal(t, "create_workflow_template", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Create")
	assert.Contains(t, tool.Description, "WorkflowTemplate")
}

func TestCreateWorkflowTemplateInput(t *testing.T) {
	// Test default values
	input := CreateWorkflowTemplateInput{}
	assert.Empty(t, input.Namespace)
	assert.Empty(t, input.Manifest)

	// Test with values
	input2 := CreateWorkflowTemplateInput{
		Namespace: "test-namespace",
		Manifest:  "test-manifest",
	}
	assert.Equal(t, "test-namespace", input2.Namespace)
	assert.Equal(t, "test-manifest", input2.Manifest)
}

func TestCreateWorkflowTemplateOutput(t *testing.T) {
	output := CreateWorkflowTemplateOutput{
		Name:      "test-template",
		Namespace: "default",
		CreatedAt: "2025-01-01T00:00:00Z",
		Created:   true,
	}

	assert.Equal(t, "test-template", output.Name)
	assert.Equal(t, "default", output.Namespace)
	assert.NotEmpty(t, output.CreatedAt)
	assert.True(t, output.Created)
}

// loadTestWorkflowTemplateYAML loads the raw YAML content of a workflow template fixture.
func loadTestWorkflowTemplateYAML(t *testing.T, filename string) string {
	t.Helper()
	return loadTestWorkflowYAML(t, filename)
}

func TestCreateWorkflowTemplateHandler(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		setupMock func(*mocks.MockWorkflowTemplateServiceClient)
		validate  func(*testing.T, *CreateWorkflowTemplateOutput, *mcp.CallToolResult)
		name      string
		input     CreateWorkflowTemplateInput
		wantErr   bool
	}{
		{
			name: "success - create simple template",
			input: CreateWorkflowTemplateInput{
				Manifest:  loadTestWorkflowTemplateYAML(t, "simple_workflow_template.yaml"),
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("CreateWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *workflowtemplate.WorkflowTemplateCreateRequest) bool {
					return req.Namespace == "default" && req.Template.Name == "hello-world-template"
				})).Return(
					&wfv1.WorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "hello-world-template",
							Namespace:         "default",
							CreationTimestamp: metav1.Time{Time: testTime},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *CreateWorkflowTemplateOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "hello-world-template", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Equal(t, testTime.Format(time.RFC3339), output.CreatedAt)
				assert.True(t, output.Created)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "hello-world-template")
				assert.Contains(t, text.Text, "default")
				assert.Contains(t, text.Text, "created")
			},
		},
		{
			name: "success - uses default namespace",
			input: CreateWorkflowTemplateInput{
				Manifest: loadTestWorkflowTemplateYAML(t, "simple_workflow_template.yaml"),
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("CreateWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *workflowtemplate.WorkflowTemplateCreateRequest) bool {
					return req.Namespace == "argo" // default namespace from mock
				})).Return(
					&wfv1.WorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "hello-world-template",
							Namespace:         "argo",
							CreationTimestamp: metav1.Time{Time: testTime},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *CreateWorkflowTemplateOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "hello-world-template", output.Name)
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "success - manifest without explicit kind",
			input: CreateWorkflowTemplateInput{
				Namespace: "default",
				Manifest: `
apiVersion: argoproj.io/v1alpha1
metadata:
  name: no-kind-template
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: alpine
      command: [echo, hello]
`,
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("CreateWorkflowTemplate", mock.Anything, mock.Anything).Return(
					&wfv1.WorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "no-kind-template",
							Namespace:         "default",
							CreationTimestamp: metav1.Time{Time: testTime},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *CreateWorkflowTemplateOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "no-kind-template", output.Name)
			},
		},
		{
			name: "error - empty manifest",
			input: CreateWorkflowTemplateInput{
				Manifest: "",
			},
			setupMock: func(_ *mocks.MockWorkflowTemplateServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - whitespace only manifest",
			input: CreateWorkflowTemplateInput{
				Manifest: "   \n\t  \n  ",
			},
			setupMock: func(_ *mocks.MockWorkflowTemplateServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - manifest too large",
			input: CreateWorkflowTemplateInput{
				Manifest: string(make([]byte, 2*1024*1024)), // 2MB > 1MB limit
			},
			setupMock: func(_ *mocks.MockWorkflowTemplateServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - invalid YAML",
			input: CreateWorkflowTemplateInput{
				Manifest: "invalid: yaml: content:\n  - {{{",
			},
			setupMock: func(_ *mocks.MockWorkflowTemplateServiceClient) {
				// No mock needed - should fail parsing
			},
			wantErr: true,
		},
		{
			name: "error - wrong kind (Workflow instead of WorkflowTemplate)",
			input: CreateWorkflowTemplateInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: wrong-kind
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
			input: CreateWorkflowTemplateInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: wrong-kind
spec:
  schedule: "* * * * *"
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
			name: "error - API error (permission denied)",
			input: CreateWorkflowTemplateInput{
				Manifest: loadTestWorkflowTemplateYAML(t, "simple_workflow_template.yaml"),
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("CreateWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.PermissionDenied, "user does not have permission"),
				)
			},
			wantErr: true,
		},
		{
			name: "success - updates existing template when already exists",
			input: CreateWorkflowTemplateInput{
				Manifest:  loadTestWorkflowTemplateYAML(t, "simple_workflow_template.yaml"),
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				// First call returns AlreadyExists
				m.On("CreateWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.AlreadyExists, "workflow template already exists"),
				)
				// Get existing template to fetch resourceVersion
				m.On("GetWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *workflowtemplate.WorkflowTemplateGetRequest) bool {
					return req.Namespace == "default" && req.Name == "hello-world-template"
				})).Return(
					&wfv1.WorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "hello-world-template",
							Namespace:       "default",
							ResourceVersion: "12345",
						},
					},
					nil,
				)
				// Then update is called and succeeds
				m.On("UpdateWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *workflowtemplate.WorkflowTemplateUpdateRequest) bool {
					return req.Namespace == "default" && req.Name == "hello-world-template"
				})).Return(
					&wfv1.WorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "hello-world-template",
							Namespace:         "default",
							CreationTimestamp: metav1.Time{Time: testTime},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *CreateWorkflowTemplateOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "hello-world-template", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.False(t, output.Created) // Was updated, not created
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "updated")
			},
		},
		{
			name: "error - update fails after already exists",
			input: CreateWorkflowTemplateInput{
				Manifest:  loadTestWorkflowTemplateYAML(t, "simple_workflow_template.yaml"),
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				// First call returns AlreadyExists
				m.On("CreateWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.AlreadyExists, "workflow template already exists"),
				)
				// Get existing template to fetch resourceVersion
				m.On("GetWorkflowTemplate", mock.Anything, mock.Anything).Return(
					&wfv1.WorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "hello-world-template",
							Namespace:       "default",
							ResourceVersion: "12345",
						},
					},
					nil,
				)
				// Update fails
				m.On("UpdateWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.PermissionDenied, "user does not have permission to update"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - get fails when fetching existing template for update",
			input: CreateWorkflowTemplateInput{
				Manifest:  loadTestWorkflowTemplateYAML(t, "simple_workflow_template.yaml"),
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				// First call returns AlreadyExists
				m.On("CreateWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.AlreadyExists, "workflow template already exists"),
				)
				// Get fails
				m.On("GetWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.NotFound, "workflow template not found"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - API error (connection refused)",
			input: CreateWorkflowTemplateInput{
				Manifest: loadTestWorkflowTemplateYAML(t, "simple_workflow_template.yaml"),
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("CreateWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("connection refused"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - unknown field in manifest",
			input: CreateWorkflowTemplateInput{
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
			handler := CreateWorkflowTemplateHandler(mockClient)
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
