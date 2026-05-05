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

func TestTerminateWorkflowTool(t *testing.T) {
	tool := TerminateWorkflowTool()

	assert.Equal(t, "terminate_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Immediately terminate", "description should mention immediate termination")
	assert.Contains(t, tool.Description, "stop_workflow", "description should mention stop_workflow alternative")
}

func TestTerminateWorkflowHandler(t *testing.T) {
	tests := []struct {
		setupMock     func(*mocks.MockWorkflowServiceClient)
		validate      func(*testing.T, *TerminateWorkflowOutput, *mcp.CallToolResult)
		name          string
		input         TerminateWorkflowInput
		wantErr       bool
		expectAPICall bool
	}{
		{
			name: "success - terminate running workflow",
			input: TerminateWorkflowInput{
				Name:      "running-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("TerminateWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowTerminateRequest) bool {
					return req.Name == "running-workflow" && req.Namespace == "default"
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "running-workflow",
							Namespace: "default",
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowFailed,
						},
					},
					nil,
				)
			},
			wantErr:       false,
			expectAPICall: true,
			validate: func(t *testing.T, output *TerminateWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "running-workflow", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Equal(t, "Failed", output.Phase)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "terminated")
			},
		},
		{
			name: "success - uses default namespace",
			input: TerminateWorkflowInput{
				Name: "running-workflow",
				// Namespace not specified - should use default
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("TerminateWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowTerminateRequest) bool {
					return req.Namespace == "argo" // default namespace from mock
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "running-workflow",
							Namespace: "argo",
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowFailed,
						},
					},
					nil,
				)
			},
			wantErr:       false,
			expectAPICall: true,
			validate: func(t *testing.T, output *TerminateWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "error - empty name",
			input: TerminateWorkflowInput{
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
			input: TerminateWorkflowInput{
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
			input: TerminateWorkflowInput{
				Name:      "nonexistent-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("TerminateWorkflow", mock.Anything, mock.Anything).Return(
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
			handler := TerminateWorkflowHandler(mockClient)
			ctx := t.Context()
			req := &mcp.CallToolRequest{}

			result, output, err := handler(ctx, req, tt.input)

			// Validate results
			if tt.wantErr {
				require.Error(t, err)
				// Verify no API call was made for validation errors
				if !tt.expectAPICall {
					mockService.AssertNotCalled(t, "TerminateWorkflow", mock.Anything, mock.Anything)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, output)
			tt.validate(t, output, result)
		})
	}
}
