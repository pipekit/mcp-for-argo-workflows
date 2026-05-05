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

func TestStopWorkflowTool(t *testing.T) {
	tool := StopWorkflowTool()

	assert.Equal(t, "stop_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Gracefully stop", "description should mention graceful stop")
	assert.Contains(t, tool.Description, "terminate_workflow", "description should mention terminate_workflow alternative")
}

func TestStopWorkflowHandler(t *testing.T) {
	tests := []struct {
		setupMock     func(*mocks.MockWorkflowServiceClient)
		validate      func(*testing.T, *StopWorkflowOutput, *mcp.CallToolResult)
		name          string
		input         StopWorkflowInput
		wantErr       bool
		expectAPICall bool
	}{
		{
			name: "success - stop running workflow",
			input: StopWorkflowInput{
				Name:      "running-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("StopWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowStopRequest) bool {
					return req.Name == "running-workflow" && req.Namespace == "default"
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "running-workflow",
							Namespace: "default",
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
			validate: func(t *testing.T, output *StopWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "running-workflow", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Equal(t, "Running", output.Phase)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "stopped")
			},
		},
		{
			name: "success - stop with message",
			input: StopWorkflowInput{
				Name:      "running-workflow",
				Namespace: "default",
				Message:   "Stopping for maintenance",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("StopWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowStopRequest) bool {
					return req.Name == "running-workflow" && req.Message == "Stopping for maintenance"
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "running-workflow",
							Namespace: "default",
						},
						Status: wfv1.WorkflowStatus{
							Phase:   wfv1.WorkflowRunning,
							Message: "Stopping for maintenance",
						},
					},
					nil,
				)
			},
			wantErr:       false,
			expectAPICall: true,
			validate: func(t *testing.T, output *StopWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "Stopping for maintenance", output.Message)
			},
		},
		{
			name: "success - stop with node field selector",
			input: StopWorkflowInput{
				Name:              "running-workflow",
				Namespace:         "default",
				NodeFieldSelector: "name=step1",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("StopWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowStopRequest) bool {
					return req.Name == "running-workflow" && req.NodeFieldSelector == "name=step1"
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "running-workflow",
							Namespace: "default",
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
			validate: func(t *testing.T, output *StopWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "running-workflow", output.Name)
			},
		},
		{
			name: "success - uses default namespace",
			input: StopWorkflowInput{
				Name: "running-workflow",
				// Namespace not specified - should use default
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("StopWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowStopRequest) bool {
					return req.Namespace == "argo" // default namespace from mock
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "running-workflow",
							Namespace: "argo",
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
			validate: func(t *testing.T, output *StopWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "error - empty name",
			input: StopWorkflowInput{
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
			input: StopWorkflowInput{
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
			input: StopWorkflowInput{
				Name:      "nonexistent-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("StopWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("workflows.argoproj.io \"nonexistent-workflow\" not found"),
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
			handler := StopWorkflowHandler(mockClient)
			ctx := t.Context()
			req := &mcp.CallToolRequest{}

			result, output, err := handler(ctx, req, tt.input)

			// Validate results
			if tt.wantErr {
				require.Error(t, err)
				// Verify no API call was made for validation errors
				if !tt.expectAPICall {
					mockService.AssertNotCalled(t, "StopWorkflow", mock.Anything, mock.Anything)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, output)
			tt.validate(t, output, result)
		})
	}
}
