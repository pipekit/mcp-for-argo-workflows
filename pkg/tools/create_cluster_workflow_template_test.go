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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestCreateClusterWorkflowTemplateTool(t *testing.T) {
	tool := CreateClusterWorkflowTemplateTool()

	assert.Equal(t, "create_cluster_workflow_template", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Create")
	assert.Contains(t, tool.Description, "ClusterWorkflowTemplate")
}

func TestCreateClusterWorkflowTemplateInput(t *testing.T) {
	// Test default values
	input := CreateClusterWorkflowTemplateInput{}
	assert.Empty(t, input.Manifest)

	// Test with values
	input2 := CreateClusterWorkflowTemplateInput{
		Manifest: "test-manifest",
	}
	assert.Equal(t, "test-manifest", input2.Manifest)
}

func TestCreateClusterWorkflowTemplateOutput(t *testing.T) {
	output := CreateClusterWorkflowTemplateOutput{
		Name:      "test-cluster-template",
		CreatedAt: "2025-01-01T00:00:00Z",
		Created:   true,
	}

	assert.Equal(t, "test-cluster-template", output.Name)
	assert.NotEmpty(t, output.CreatedAt)
	assert.True(t, output.Created)
}

// loadTestClusterWorkflowTemplateYAML loads the raw YAML content of a cluster workflow template fixture.
func loadTestClusterWorkflowTemplateYAML(t *testing.T, filename string) string {
	t.Helper()
	return loadTestWorkflowYAML(t, filename)
}

func TestCreateClusterWorkflowTemplateHandler(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		setupMock func(*mocks.MockClusterWorkflowTemplateServiceClient)
		validate  func(*testing.T, *CreateClusterWorkflowTemplateOutput, *mcp.CallToolResult)
		name      string
		input     CreateClusterWorkflowTemplateInput
		wantErr   bool
	}{
		{
			name: "success - create simple cluster template",
			input: CreateClusterWorkflowTemplateInput{
				Manifest: loadTestClusterWorkflowTemplateYAML(t, "simple_cluster_workflow_template.yaml"),
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("CreateClusterWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *clusterworkflowtemplate.ClusterWorkflowTemplateCreateRequest) bool {
					return req.Template.Name == "hello-world-cluster-template"
				})).Return(
					&wfv1.ClusterWorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "hello-world-cluster-template",
							CreationTimestamp: metav1.Time{Time: testTime},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *CreateClusterWorkflowTemplateOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "hello-world-cluster-template", output.Name)
				assert.Equal(t, testTime.Format(time.RFC3339), output.CreatedAt)
				assert.True(t, output.Created)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "hello-world-cluster-template")
				assert.Contains(t, text.Text, "created")
			},
		},
		{
			name: "success - manifest without explicit kind",
			input: CreateClusterWorkflowTemplateInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
metadata:
  name: no-kind-cluster-template
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: alpine
      command: [echo, hello]
`,
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("CreateClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					&wfv1.ClusterWorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "no-kind-cluster-template",
							CreationTimestamp: metav1.Time{Time: testTime},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *CreateClusterWorkflowTemplateOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "no-kind-cluster-template", output.Name)
			},
		},
		{
			name: "error - empty manifest",
			input: CreateClusterWorkflowTemplateInput{
				Manifest: "",
			},
			setupMock: func(_ *mocks.MockClusterWorkflowTemplateServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - whitespace only manifest",
			input: CreateClusterWorkflowTemplateInput{
				Manifest: "   \n\t  \n  ",
			},
			setupMock: func(_ *mocks.MockClusterWorkflowTemplateServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - manifest too large",
			input: CreateClusterWorkflowTemplateInput{
				Manifest: string(make([]byte, 2*1024*1024)), // 2MB > 1MB limit
			},
			setupMock: func(_ *mocks.MockClusterWorkflowTemplateServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - invalid YAML",
			input: CreateClusterWorkflowTemplateInput{
				Manifest: "invalid: yaml: content:\n  - {{{",
			},
			setupMock: func(_ *mocks.MockClusterWorkflowTemplateServiceClient) {
				// No mock needed - should fail parsing
			},
			wantErr: true,
		},
		{
			name: "error - wrong kind (Workflow instead of ClusterWorkflowTemplate)",
			input: CreateClusterWorkflowTemplateInput{
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
			setupMock: func(_ *mocks.MockClusterWorkflowTemplateServiceClient) {
				// No mock needed - should fail kind validation
			},
			wantErr: true,
		},
		{
			name: "error - wrong kind (WorkflowTemplate)",
			input: CreateClusterWorkflowTemplateInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
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
			setupMock: func(_ *mocks.MockClusterWorkflowTemplateServiceClient) {
				// No mock needed - should fail kind validation
			},
			wantErr: true,
		},
		{
			name: "error - API error (permission denied)",
			input: CreateClusterWorkflowTemplateInput{
				Manifest: loadTestClusterWorkflowTemplateYAML(t, "simple_cluster_workflow_template.yaml"),
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("CreateClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.PermissionDenied, "user does not have permission"),
				)
			},
			wantErr: true,
		},
		{
			name: "success - updates existing template when already exists",
			input: CreateClusterWorkflowTemplateInput{
				Manifest: loadTestClusterWorkflowTemplateYAML(t, "simple_cluster_workflow_template.yaml"),
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				// First call returns AlreadyExists
				m.On("CreateClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.AlreadyExists, "cluster workflow template already exists"),
				)
				// Get existing template to fetch resourceVersion
				m.On("GetClusterWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *clusterworkflowtemplate.ClusterWorkflowTemplateGetRequest) bool {
					return req.Name == "hello-world-cluster-template"
				})).Return(
					&wfv1.ClusterWorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "hello-world-cluster-template",
							ResourceVersion: "12345",
						},
					},
					nil,
				)
				// Then update is called and succeeds
				m.On("UpdateClusterWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *clusterworkflowtemplate.ClusterWorkflowTemplateUpdateRequest) bool {
					return req.Name == "hello-world-cluster-template"
				})).Return(
					&wfv1.ClusterWorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "hello-world-cluster-template",
							CreationTimestamp: metav1.Time{Time: testTime},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *CreateClusterWorkflowTemplateOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "hello-world-cluster-template", output.Name)
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
			input: CreateClusterWorkflowTemplateInput{
				Manifest: loadTestClusterWorkflowTemplateYAML(t, "simple_cluster_workflow_template.yaml"),
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				// First call returns AlreadyExists
				m.On("CreateClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.AlreadyExists, "cluster workflow template already exists"),
				)
				// Get existing template to fetch resourceVersion
				m.On("GetClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					&wfv1.ClusterWorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "hello-world-cluster-template",
							ResourceVersion: "12345",
						},
					},
					nil,
				)
				// Update fails
				m.On("UpdateClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.PermissionDenied, "user does not have permission to update"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - get fails when fetching existing template for update",
			input: CreateClusterWorkflowTemplateInput{
				Manifest: loadTestClusterWorkflowTemplateYAML(t, "simple_cluster_workflow_template.yaml"),
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				// First call returns AlreadyExists
				m.On("CreateClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.AlreadyExists, "cluster workflow template already exists"),
				)
				// Get fails
				m.On("GetClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.NotFound, "cluster workflow template not found"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - API error (connection refused)",
			input: CreateClusterWorkflowTemplateInput{
				Manifest: loadTestClusterWorkflowTemplateYAML(t, "simple_cluster_workflow_template.yaml"),
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("CreateClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("connection refused"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - unknown field in manifest",
			input: CreateClusterWorkflowTemplateInput{
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
			handler := CreateClusterWorkflowTemplateHandler(mockClient)
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
