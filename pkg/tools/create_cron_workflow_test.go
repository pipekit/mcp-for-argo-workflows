package tools

import (
	"errors"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
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

func TestCreateCronWorkflowTool(t *testing.T) {
	tool := CreateCronWorkflowTool()

	assert.Equal(t, "create_cron_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Create")
	assert.Contains(t, tool.Description, "CronWorkflow")
}

func TestCreateCronWorkflowInput(t *testing.T) {
	// Test default values
	input := CreateCronWorkflowInput{}
	assert.Empty(t, input.Namespace)
	assert.Empty(t, input.Manifest)

	// Test with values
	input2 := CreateCronWorkflowInput{
		Namespace: "test-namespace",
		Manifest:  "test-manifest",
	}
	assert.Equal(t, "test-namespace", input2.Namespace)
	assert.Equal(t, "test-manifest", input2.Manifest)
}

func TestCreateCronWorkflowOutput(t *testing.T) {
	output := CreateCronWorkflowOutput{
		Name:              "test-cron",
		Namespace:         "default",
		Schedules:         []string{"0 * * * *"},
		Timezone:          "UTC",
		ConcurrencyPolicy: "Replace",
		Entrypoint:        "main",
		CreatedAt:         "2025-01-01T00:00:00Z",
		Suspended:         false,
		Created:           true,
	}

	assert.Equal(t, "test-cron", output.Name)
	assert.Equal(t, "default", output.Namespace)
	assert.Equal(t, []string{"0 * * * *"}, output.Schedules)
	assert.Equal(t, "UTC", output.Timezone)
	assert.Equal(t, "Replace", output.ConcurrencyPolicy)
	assert.Equal(t, "main", output.Entrypoint)
	assert.NotEmpty(t, output.CreatedAt)
	assert.False(t, output.Suspended)
	assert.True(t, output.Created)
}

// loadTestCronWorkflowYAML loads the raw YAML content of a cron workflow fixture.
func loadTestCronWorkflowYAML(t *testing.T, filename string) string {
	t.Helper()
	return loadTestWorkflowYAML(t, filename)
}

func TestCreateCronWorkflowHandler(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		setupMock func(*mocks.MockCronWorkflowServiceClient)
		validate  func(*testing.T, *CreateCronWorkflowOutput, *mcp.CallToolResult)
		name      string
		input     CreateCronWorkflowInput
		wantErr   bool
	}{
		{
			name: "success - create simple cron workflow",
			input: CreateCronWorkflowInput{
				Manifest:  loadTestCronWorkflowYAML(t, "simple_cron_workflow.yaml"),
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("CreateCronWorkflow", mock.Anything, mock.MatchedBy(func(req *cronworkflow.CreateCronWorkflowRequest) bool {
					return req.Namespace == "default" && req.CronWorkflow.Name == "hello-world-cron"
				})).Return(
					&wfv1.CronWorkflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "hello-world-cron",
							Namespace:         "default",
							CreationTimestamp: metav1.Time{Time: testTime},
						},
						Spec: wfv1.CronWorkflowSpec{
							Schedules:         []string{"0 * * * *"},
							ConcurrencyPolicy: wfv1.ReplaceConcurrent,
							WorkflowSpec: wfv1.WorkflowSpec{
								Entrypoint: "whalesay",
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *CreateCronWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "hello-world-cron", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Equal(t, []string{"0 * * * *"}, output.Schedules)
				assert.Equal(t, "Replace", output.ConcurrencyPolicy)
				assert.Equal(t, "whalesay", output.Entrypoint)
				assert.Equal(t, testTime.Format(time.RFC3339), output.CreatedAt)
				assert.False(t, output.Suspended)
				assert.True(t, output.Created)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "hello-world-cron")
				assert.Contains(t, text.Text, "default")
				assert.Contains(t, text.Text, "0 * * * *")
			},
		},
		{
			name: "success - uses default namespace",
			input: CreateCronWorkflowInput{
				Manifest: loadTestCronWorkflowYAML(t, "simple_cron_workflow.yaml"),
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("CreateCronWorkflow", mock.Anything, mock.MatchedBy(func(req *cronworkflow.CreateCronWorkflowRequest) bool {
					return req.Namespace == "argo" // default namespace from mock
				})).Return(
					&wfv1.CronWorkflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "hello-world-cron",
							Namespace:         "argo",
							CreationTimestamp: metav1.Time{Time: testTime},
						},
						Spec: wfv1.CronWorkflowSpec{
							Schedules: []string{"0 * * * *"},
							WorkflowSpec: wfv1.WorkflowSpec{
								Entrypoint: "whalesay",
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *CreateCronWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "hello-world-cron", output.Name)
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "success - cron workflow with timezone",
			input: CreateCronWorkflowInput{
				Namespace: "default",
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: timezone-cron
spec:
  schedules:
  - "0 9 * * *"
  timezone: "America/New_York"
  workflowSpec:
    entrypoint: main
    templates:
    - name: main
      container:
        image: alpine
        command: [echo, hello]
`,
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("CreateCronWorkflow", mock.Anything, mock.Anything).Return(
					&wfv1.CronWorkflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "timezone-cron",
							Namespace:         "default",
							CreationTimestamp: metav1.Time{Time: testTime},
						},
						Spec: wfv1.CronWorkflowSpec{
							Schedules: []string{"0 9 * * *"},
							Timezone:  "America/New_York",
							WorkflowSpec: wfv1.WorkflowSpec{
								Entrypoint: "main",
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *CreateCronWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "timezone-cron", output.Name)
				assert.Equal(t, "America/New_York", output.Timezone)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "America/New_York")
			},
		},
		{
			name: "success - suspended cron workflow",
			input: CreateCronWorkflowInput{
				Namespace: "default",
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: suspended-cron
spec:
  schedules:
  - "0 * * * *"
  suspend: true
  workflowSpec:
    entrypoint: main
    templates:
    - name: main
      container:
        image: alpine
        command: [echo, hello]
`,
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("CreateCronWorkflow", mock.Anything, mock.Anything).Return(
					&wfv1.CronWorkflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "suspended-cron",
							Namespace:         "default",
							CreationTimestamp: metav1.Time{Time: testTime},
						},
						Spec: wfv1.CronWorkflowSpec{
							Schedules: []string{"0 * * * *"},
							Suspend:   true,
							WorkflowSpec: wfv1.WorkflowSpec{
								Entrypoint: "main",
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *CreateCronWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "suspended-cron", output.Name)
				assert.True(t, output.Suspended)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "Suspended")
			},
		},
		{
			name: "success - manifest without explicit kind",
			input: CreateCronWorkflowInput{
				Namespace: "default",
				Manifest: `
apiVersion: argoproj.io/v1alpha1
metadata:
  name: no-kind-cron
spec:
  schedules:
  - "*/5 * * * *"
  workflowSpec:
    entrypoint: main
    templates:
    - name: main
      container:
        image: alpine
        command: [echo, hello]
`,
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("CreateCronWorkflow", mock.Anything, mock.Anything).Return(
					&wfv1.CronWorkflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "no-kind-cron",
							Namespace:         "default",
							CreationTimestamp: metav1.Time{Time: testTime},
						},
						Spec: wfv1.CronWorkflowSpec{
							Schedules: []string{"*/5 * * * *"},
							WorkflowSpec: wfv1.WorkflowSpec{
								Entrypoint: "main",
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *CreateCronWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "no-kind-cron", output.Name)
			},
		},
		{
			name: "error - empty manifest",
			input: CreateCronWorkflowInput{
				Manifest: "",
			},
			setupMock: func(_ *mocks.MockCronWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - whitespace only manifest",
			input: CreateCronWorkflowInput{
				Manifest: "   \n\t  \n  ",
			},
			setupMock: func(_ *mocks.MockCronWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - manifest too large",
			input: CreateCronWorkflowInput{
				Manifest: string(make([]byte, 2*1024*1024)), // 2MB > 1MB limit
			},
			setupMock: func(_ *mocks.MockCronWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - invalid YAML",
			input: CreateCronWorkflowInput{
				Manifest: "invalid: yaml: content:\n  - {{{",
			},
			setupMock: func(_ *mocks.MockCronWorkflowServiceClient) {
				// No mock needed - should fail parsing
			},
			wantErr: true,
		},
		{
			name: "error - wrong kind (Workflow instead of CronWorkflow)",
			input: CreateCronWorkflowInput{
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
			setupMock: func(_ *mocks.MockCronWorkflowServiceClient) {
				// No mock needed - should fail kind validation
			},
			wantErr: true,
		},
		{
			name: "error - wrong kind (WorkflowTemplate)",
			input: CreateCronWorkflowInput{
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
			setupMock: func(_ *mocks.MockCronWorkflowServiceClient) {
				// No mock needed - should fail kind validation
			},
			wantErr: true,
		},
		{
			name: "error - missing name in manifest",
			input: CreateCronWorkflowInput{
				Namespace: "default",
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  labels:
    app: test
spec:
  schedules:
  - "0 * * * *"
  workflowSpec:
    entrypoint: main
    templates:
    - name: main
      container:
        image: alpine
        command: [echo, hello]
`,
			},
			setupMock: func(_ *mocks.MockCronWorkflowServiceClient) {
				// No mock needed - should fail name validation
			},
			wantErr: true,
		},
		{
			name: "error - missing schedule in manifest",
			input: CreateCronWorkflowInput{
				Namespace: "default",
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: no-schedule-cron
spec:
  workflowSpec:
    entrypoint: main
    templates:
    - name: main
      container:
        image: alpine
        command: [echo, hello]
`,
			},
			setupMock: func(_ *mocks.MockCronWorkflowServiceClient) {
				// No mock needed - should fail schedule validation
			},
			wantErr: true,
		},
		{
			name: "error - API error (permission denied)",
			input: CreateCronWorkflowInput{
				Manifest: loadTestCronWorkflowYAML(t, "simple_cron_workflow.yaml"),
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("CreateCronWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.PermissionDenied, "user does not have permission"),
				)
			},
			wantErr: true,
		},
		{
			name: "success - updates existing cron workflow when already exists",
			input: CreateCronWorkflowInput{
				Manifest:  loadTestCronWorkflowYAML(t, "simple_cron_workflow.yaml"),
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				// First call returns AlreadyExists
				m.On("CreateCronWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.AlreadyExists, "cron workflow already exists"),
				)
				// Get existing cron workflow to fetch resourceVersion
				m.On("GetCronWorkflow", mock.Anything, mock.MatchedBy(func(req *cronworkflow.GetCronWorkflowRequest) bool {
					return req.Namespace == "default" && req.Name == "hello-world-cron"
				})).Return(
					&wfv1.CronWorkflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "hello-world-cron",
							Namespace:       "default",
							ResourceVersion: "12345",
						},
					},
					nil,
				)
				// Then update is called and succeeds
				m.On("UpdateCronWorkflow", mock.Anything, mock.MatchedBy(func(req *cronworkflow.UpdateCronWorkflowRequest) bool {
					return req.Namespace == "default" && req.Name == "hello-world-cron"
				})).Return(
					&wfv1.CronWorkflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "hello-world-cron",
							Namespace:         "default",
							CreationTimestamp: metav1.Time{Time: testTime},
						},
						Spec: wfv1.CronWorkflowSpec{
							Schedules: []string{"0 * * * *"},
							WorkflowSpec: wfv1.WorkflowSpec{
								Entrypoint: "whalesay",
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *CreateCronWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "hello-world-cron", output.Name)
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
			input: CreateCronWorkflowInput{
				Manifest:  loadTestCronWorkflowYAML(t, "simple_cron_workflow.yaml"),
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				// First call returns AlreadyExists
				m.On("CreateCronWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.AlreadyExists, "cron workflow already exists"),
				)
				// Get existing cron workflow to fetch resourceVersion
				m.On("GetCronWorkflow", mock.Anything, mock.Anything).Return(
					&wfv1.CronWorkflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:            "hello-world-cron",
							Namespace:       "default",
							ResourceVersion: "12345",
						},
					},
					nil,
				)
				// Update fails
				m.On("UpdateCronWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.PermissionDenied, "user does not have permission to update"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - get fails when fetching existing cron workflow for update",
			input: CreateCronWorkflowInput{
				Manifest:  loadTestCronWorkflowYAML(t, "simple_cron_workflow.yaml"),
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				// First call returns AlreadyExists
				m.On("CreateCronWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.AlreadyExists, "cron workflow already exists"),
				)
				// Get fails
				m.On("GetCronWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.NotFound, "cron workflow not found"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - API error (connection refused)",
			input: CreateCronWorkflowInput{
				Manifest: loadTestCronWorkflowYAML(t, "simple_cron_workflow.yaml"),
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("CreateCronWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("connection refused"),
				)
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
			handler := CreateCronWorkflowHandler(mockClient)
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
