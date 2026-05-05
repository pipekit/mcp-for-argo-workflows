package tools

import (
	"testing"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestApplyParameterOverrides(t *testing.T) {
	tests := []struct {
		wantParams []wfv1.Parameter
		workflow   *wfv1.Workflow
		name       string
		params     []string
		wantErr    bool
	}{
		{
			name: "update existing parameter",
			workflow: &wfv1.Workflow{
				Spec: wfv1.WorkflowSpec{
					Arguments: wfv1.Arguments{
						Parameters: []wfv1.Parameter{
							{Name: "message", Value: wfv1.AnyStringPtr("hello")},
						},
					},
				},
			},
			params:  []string{"message=world"},
			wantErr: false,
			wantParams: []wfv1.Parameter{
				{Name: "message", Value: wfv1.AnyStringPtr("world")},
			},
		},
		{
			name: "add new parameter",
			workflow: &wfv1.Workflow{
				Spec: wfv1.WorkflowSpec{
					Arguments: wfv1.Arguments{
						Parameters: []wfv1.Parameter{},
					},
				},
			},
			params:  []string{"newparam=newvalue"},
			wantErr: false,
			wantParams: []wfv1.Parameter{
				{Name: "newparam", Value: wfv1.AnyStringPtr("newvalue")},
			},
		},
		{
			name: "multiple parameters",
			workflow: &wfv1.Workflow{
				Spec: wfv1.WorkflowSpec{
					Arguments: wfv1.Arguments{
						Parameters: []wfv1.Parameter{
							{Name: "existing", Value: wfv1.AnyStringPtr("old")},
						},
					},
				},
			},
			params:  []string{"existing=new", "another=value"},
			wantErr: false,
			wantParams: []wfv1.Parameter{
				{Name: "existing", Value: wfv1.AnyStringPtr("new")},
				{Name: "another", Value: wfv1.AnyStringPtr("value")},
			},
		},
		{
			name: "parameter with equals in value",
			workflow: &wfv1.Workflow{
				Spec: wfv1.WorkflowSpec{
					Arguments: wfv1.Arguments{
						Parameters: []wfv1.Parameter{},
					},
				},
			},
			params:  []string{"equation=a=b+c"},
			wantErr: false,
			wantParams: []wfv1.Parameter{
				{Name: "equation", Value: wfv1.AnyStringPtr("a=b+c")},
			},
		},
		{
			name: "invalid parameter format - no equals",
			workflow: &wfv1.Workflow{
				Spec: wfv1.WorkflowSpec{
					Arguments: wfv1.Arguments{
						Parameters: []wfv1.Parameter{},
					},
				},
			},
			params:  []string{"invalid"},
			wantErr: true,
		},
		{
			name: "empty workflow arguments",
			workflow: &wfv1.Workflow{
				Spec: wfv1.WorkflowSpec{},
			},
			params:  []string{"param=value"},
			wantErr: false,
			wantParams: []wfv1.Parameter{
				{Name: "param", Value: wfv1.AnyStringPtr("value")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := applyParameterOverrides(tt.workflow, tt.params)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			gotParams := tt.workflow.Spec.Arguments.Parameters
			require.Len(t, gotParams, len(tt.wantParams))

			for i, want := range tt.wantParams {
				got := gotParams[i]
				assert.Equal(t, want.Name, got.Name, "param[%d].Name mismatch", i)
				require.NotNil(t, got.Value, "param[%d].Value is nil", i)
				require.NotNil(t, want.Value, "want param[%d].Value is nil", i)
				assert.Equal(t, want.Value.String(), got.Value.String(), "param[%d].Value mismatch", i)
			}
		})
	}
}

func TestSubmitWorkflowTool(t *testing.T) {
	tool := SubmitWorkflowTool()

	assert.Equal(t, "submit_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
}

func TestSubmitWorkflowHandler(t *testing.T) {
	tests := []struct {
		setupMock func(*mocks.MockWorkflowServiceClient)
		validate  func(*testing.T, *SubmitWorkflowOutput)
		name      string
		input     SubmitWorkflowInput
		wantErr   bool
	}{
		{
			name: "success - simple workflow",
			input: SubmitWorkflowInput{
				Manifest:  loadTestWorkflowYAML(t, "simple_workflow.yaml"),
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("CreateWorkflow", mock.Anything, mock.Anything).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "hello-world-abc123",
							Namespace: "default",
							UID:       types.UID("test-uid-123"),
						},
						Status: wfv1.WorkflowStatus{
							Phase:   wfv1.WorkflowPending,
							Message: "Workflow created",
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *SubmitWorkflowOutput) {
				assert.Equal(t, "hello-world-abc123", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Equal(t, "test-uid-123", output.UID)
				assert.Equal(t, "Pending", output.Phase)
			},
		},
		{
			name: "success - workflow with parameters",
			input: SubmitWorkflowInput{
				Manifest:   loadTestWorkflowYAML(t, "workflow_with_params.yaml"),
				Parameters: []string{"message=goodbye world", "count=5"},
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("CreateWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowCreateRequest) bool {
					// Verify parameters were applied
					if len(req.Workflow.Spec.Arguments.Parameters) != 2 {
						return false
					}
					for _, p := range req.Workflow.Spec.Arguments.Parameters {
						if p.Name == "message" && p.Value != nil && p.Value.String() == "goodbye world" {
							continue
						}
						if p.Name == "count" && p.Value != nil && p.Value.String() == "5" {
							continue
						}
						return false
					}
					return true
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "param-workflow-xyz789",
							Namespace: "argo",
							UID:       types.UID("param-uid-789"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowPending,
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *SubmitWorkflowOutput) {
				assert.Equal(t, "param-workflow-xyz789", output.Name)
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "success - override generateName",
			input: SubmitWorkflowInput{
				Manifest:     loadTestWorkflowYAML(t, "simple_workflow.yaml"),
				GenerateName: "custom-prefix-",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("CreateWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowCreateRequest) bool {
					return req.Workflow.GenerateName == "custom-prefix-"
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "custom-prefix-abc",
							Namespace: "argo",
							UID:       types.UID("custom-uid"),
						},
						Status: wfv1.WorkflowStatus{},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *SubmitWorkflowOutput) {
				assert.Equal(t, "custom-prefix-abc", output.Name)
				assert.Equal(t, "Pending", output.Phase) // Default when empty
			},
		},
		{
			name: "success - add labels",
			input: SubmitWorkflowInput{
				Manifest: loadTestWorkflowYAML(t, "simple_workflow.yaml"),
				Labels: map[string]string{
					"env":  "test",
					"team": "platform",
				},
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("CreateWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowCreateRequest) bool {
					return req.Workflow.Labels["env"] == "test" &&
						req.Workflow.Labels["team"] == "platform"
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "labeled-workflow",
							Namespace: "argo",
							UID:       types.UID("labeled-uid"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowRunning,
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *SubmitWorkflowOutput) {
				assert.Equal(t, "labeled-workflow", output.Name)
				assert.Equal(t, "Running", output.Phase)
			},
		},
		{
			name: "error - empty manifest",
			input: SubmitWorkflowInput{
				Manifest: "",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - manifest too large",
			input: SubmitWorkflowInput{
				Manifest: string(make([]byte, 2*1024*1024)), // 2MB
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - invalid YAML",
			input: SubmitWorkflowInput{
				Manifest: "invalid: yaml: content:\n  - {{{",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail parsing
			},
			wantErr: true,
		},
		{
			name: "error - wrong kind",
			input: SubmitWorkflowInput{
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
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail kind validation
			},
			wantErr: true,
		},
		{
			name: "error - invalid parameter format",
			input: SubmitWorkflowInput{
				Manifest:   loadTestWorkflowYAML(t, "simple_workflow.yaml"),
				Parameters: []string{"invalid-no-equals"},
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail parameter parsing
			},
			wantErr: true,
		},
		{
			name: "error - API error (permission denied)",
			input: SubmitWorkflowInput{
				Manifest: loadTestWorkflowYAML(t, "simple_workflow.yaml"),
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("CreateWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.PermissionDenied, "user does not have permission to create workflows"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - API error (already exists)",
			input: SubmitWorkflowInput{
				Manifest: `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: existing-workflow
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello"]
`,
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("CreateWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.AlreadyExists, "workflow already exists"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - unknown field in manifest",
			input: SubmitWorkflowInput{
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
			handler := SubmitWorkflowHandler(mockClient)
			ctx := t.Context()
			req := &mcp.CallToolRequest{}

			result, output, err := handler(ctx, req, tt.input)

			// Validate results
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Nil(t, result) // Handler returns nil for result
			require.NotNil(t, output)
			tt.validate(t, output)
		})
	}
}
