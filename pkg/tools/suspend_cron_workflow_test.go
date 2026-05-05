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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestSuspendCronWorkflowTool(t *testing.T) {
	tool := SuspendCronWorkflowTool()

	assert.Equal(t, "suspend_cron_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Suspend")
	assert.Contains(t, tool.Description, "CronWorkflow")
}

func TestSuspendCronWorkflowInput(t *testing.T) {
	// Test default values
	input := SuspendCronWorkflowInput{}
	assert.Empty(t, input.Namespace)
	assert.Empty(t, input.Name)

	// Test with values
	input2 := SuspendCronWorkflowInput{
		Namespace: "test-namespace",
		Name:      "test-cron",
	}
	assert.Equal(t, "test-namespace", input2.Namespace)
	assert.Equal(t, "test-cron", input2.Name)
}

func TestSuspendCronWorkflowOutput(t *testing.T) {
	output := SuspendCronWorkflowOutput{
		Name:      "test-cron",
		Namespace: "default",
		Schedules: []string{"0 * * * *"},
		Suspended: true,
		Message:   "suspended successfully",
	}

	assert.Equal(t, "test-cron", output.Name)
	assert.Equal(t, "default", output.Namespace)
	assert.Equal(t, []string{"0 * * * *"}, output.Schedules)
	assert.True(t, output.Suspended)
	assert.NotEmpty(t, output.Message)
}

func TestSuspendCronWorkflowHandler(t *testing.T) {
	tests := []struct {
		setupMock func(*mocks.MockCronWorkflowServiceClient)
		validate  func(*testing.T, *SuspendCronWorkflowOutput, *mcp.CallToolResult)
		name      string
		input     SuspendCronWorkflowInput
		wantErr   bool
	}{
		{
			name: "success - suspend cron workflow",
			input: SuspendCronWorkflowInput{
				Namespace: "default",
				Name:      "my-cron",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("SuspendCronWorkflow", mock.Anything, mock.MatchedBy(func(req *cronworkflow.CronWorkflowSuspendRequest) bool {
					return req.Namespace == "default" && req.Name == "my-cron"
				})).Return(
					&wfv1.CronWorkflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-cron",
							Namespace: "default",
						},
						Spec: wfv1.CronWorkflowSpec{
							Schedules: []string{"0 * * * *"},
							Suspend:   true,
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *SuspendCronWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "my-cron", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Equal(t, []string{"0 * * * *"}, output.Schedules)
				assert.True(t, output.Suspended)
				assert.Contains(t, output.Message, "suspended successfully")
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "my-cron")
				assert.Contains(t, text.Text, "default")
				assert.Contains(t, text.Text, "suspended successfully")
				assert.Contains(t, text.Text, "paused")
			},
		},
		{
			name: "success - uses default namespace",
			input: SuspendCronWorkflowInput{
				Name: "my-cron",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("SuspendCronWorkflow", mock.Anything, mock.MatchedBy(func(req *cronworkflow.CronWorkflowSuspendRequest) bool {
					return req.Namespace == "argo" && req.Name == "my-cron" // default namespace from mock
				})).Return(
					&wfv1.CronWorkflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-cron",
							Namespace: "argo",
						},
						Spec: wfv1.CronWorkflowSpec{
							Schedules: []string{"*/5 * * * *"},
							Suspend:   true,
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *SuspendCronWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "my-cron", output.Name)
				assert.Equal(t, "argo", output.Namespace)
				assert.True(t, output.Suspended)
			},
		},
		{
			name: "success - name with whitespace trimmed",
			input: SuspendCronWorkflowInput{
				Namespace: "default",
				Name:      "  my-cron  ",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("SuspendCronWorkflow", mock.Anything, mock.MatchedBy(func(req *cronworkflow.CronWorkflowSuspendRequest) bool {
					return req.Name == "my-cron"
				})).Return(
					&wfv1.CronWorkflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-cron",
							Namespace: "default",
						},
						Spec: wfv1.CronWorkflowSpec{
							Schedules: []string{"0 0 * * *"},
							Suspend:   true,
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *SuspendCronWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "my-cron", output.Name)
				assert.True(t, output.Suspended)
			},
		},
		{
			name: "error - empty name",
			input: SuspendCronWorkflowInput{
				Namespace: "default",
				Name:      "",
			},
			setupMock: func(_ *mocks.MockCronWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - whitespace only name",
			input: SuspendCronWorkflowInput{
				Namespace: "default",
				Name:      "   ",
			},
			setupMock: func(_ *mocks.MockCronWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - API error (not found)",
			input: SuspendCronWorkflowInput{
				Namespace: "default",
				Name:      "nonexistent-cron",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("SuspendCronWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.NotFound, "cron workflow not found"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - API error (permission denied)",
			input: SuspendCronWorkflowInput{
				Namespace: "default",
				Name:      "protected-cron",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("SuspendCronWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.PermissionDenied, "user does not have permission"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - API error (connection refused)",
			input: SuspendCronWorkflowInput{
				Namespace: "default",
				Name:      "my-cron",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("SuspendCronWorkflow", mock.Anything, mock.Anything).Return(
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
			handler := SuspendCronWorkflowHandler(mockClient)
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
