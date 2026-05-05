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
	"k8s.io/apimachinery/pkg/types"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestRetryWorkflowTool(t *testing.T) {
	tool := RetryWorkflowTool()

	assert.Equal(t, "retry_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Retry", "description should mention retry")
}

func TestRetryWorkflowHandler(t *testing.T) {
	tests := []struct {
		setupMock     func(*mocks.MockWorkflowServiceClient)
		validate      func(*testing.T, *RetryWorkflowOutput, *mcp.CallToolResult)
		name          string
		input         RetryWorkflowInput
		wantErr       bool
		expectAPICall bool
	}{
		{
			name: "success - basic retry",
			input: RetryWorkflowInput{
				Name:      "failed-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("RetryWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowRetryRequest) bool {
					return req.Name == "failed-workflow" && req.Namespace == "default"
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "failed-workflow",
							Namespace: "default",
							UID:       types.UID("test-uid-123"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowRunning,
						},
					},
					nil,
				)
			},
			wantErr:       false,
			expectAPICall: true,
			validate: func(t *testing.T, output *RetryWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "failed-workflow", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Equal(t, "test-uid-123", output.UID)
				assert.Equal(t, "Running", output.Phase)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "retried successfully")
			},
		},
		{
			name: "success - retry with restart successful",
			input: RetryWorkflowInput{
				Name:              "failed-workflow",
				Namespace:         "default",
				RestartSuccessful: true,
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("RetryWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowRetryRequest) bool {
					return req.Name == "failed-workflow" && req.RestartSuccessful == true
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "failed-workflow",
							Namespace: "default",
							UID:       types.UID("test-uid-123"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowRunning,
						},
					},
					nil,
				)
			},
			wantErr:       false,
			expectAPICall: true,
			validate: func(t *testing.T, output *RetryWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "failed-workflow", output.Name)
				assert.Equal(t, "Running", output.Phase)
			},
		},
		{
			name: "success - retry with node field selector",
			input: RetryWorkflowInput{
				Name:              "failed-workflow",
				Namespace:         "default",
				NodeFieldSelector: "phase=Failed",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("RetryWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowRetryRequest) bool {
					return req.Name == "failed-workflow" && req.NodeFieldSelector == "phase=Failed"
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "failed-workflow",
							Namespace: "default",
							UID:       types.UID("test-uid-123"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowRunning,
						},
					},
					nil,
				)
			},
			wantErr:       false,
			expectAPICall: true,
			validate: func(t *testing.T, output *RetryWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "failed-workflow", output.Name)
			},
		},
		{
			name: "success - retry with parameters",
			input: RetryWorkflowInput{
				Name:       "failed-workflow",
				Namespace:  "default",
				Parameters: []string{"param1=value1", "param2=value2"},
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("RetryWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowRetryRequest) bool {
					return req.Name == "failed-workflow" &&
						len(req.Parameters) == 2 &&
						req.Parameters[0] == "param1=value1" &&
						req.Parameters[1] == "param2=value2"
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "failed-workflow",
							Namespace: "default",
							UID:       types.UID("test-uid-123"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowRunning,
						},
					},
					nil,
				)
			},
			wantErr:       false,
			expectAPICall: true,
			validate: func(t *testing.T, output *RetryWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "failed-workflow", output.Name)
			},
		},
		{
			name: "success - uses default namespace",
			input: RetryWorkflowInput{
				Name: "failed-workflow",
				// Namespace not specified - should use default
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("RetryWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowRetryRequest) bool {
					return req.Namespace == "argo" // default namespace from mock
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "failed-workflow",
							Namespace: "argo",
							UID:       types.UID("test-uid-123"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowRunning,
						},
					},
					nil,
				)
			},
			wantErr:       false,
			expectAPICall: true,
			validate: func(t *testing.T, output *RetryWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "error - empty name",
			input: RetryWorkflowInput{
				Name: "",
			},
			setupMock: func(_ *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr:       true,
			expectAPICall: false,
		},
		{
			name: "error - whitespace-only name",
			input: RetryWorkflowInput{
				Name: "   ",
			},
			setupMock: func(_ *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr:       true,
			expectAPICall: false,
		},
		{
			name: "error - workflow not found",
			input: RetryWorkflowInput{
				Name:      "nonexistent-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("RetryWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("workflows.argoproj.io \"nonexistent-workflow\" not found"),
				)
			},
			wantErr:       true,
			expectAPICall: true,
		},
		{
			name: "error - workflow not in retryable state",
			input: RetryWorkflowInput{
				Name:      "running-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("RetryWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("workflow is not in a completed phase"),
				)
			},
			wantErr:       true,
			expectAPICall: true,
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
			handler := RetryWorkflowHandler(mockClient)
			ctx := t.Context()
			req := &mcp.CallToolRequest{}

			result, output, err := handler(ctx, req, tt.input)

			// Validate results
			if tt.wantErr {
				require.Error(t, err)
				// Verify no API call was made for validation errors
				if !tt.expectAPICall {
					mockService.AssertNotCalled(t, "RetryWorkflow", mock.Anything, mock.Anything)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, output)
			tt.validate(t, output, result)
		})
	}
}
