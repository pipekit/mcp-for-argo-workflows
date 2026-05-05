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

func TestSuspendWorkflowTool(t *testing.T) {
	tool := SuspendWorkflowTool()

	assert.Equal(t, "suspend_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Suspend", "description should mention suspend")
}

func TestSuspendWorkflowHandler(t *testing.T) {
	tests := []struct {
		setupMock     func(*mocks.MockWorkflowServiceClient)
		validate      func(*testing.T, *SuspendWorkflowOutput, *mcp.CallToolResult)
		name          string
		input         SuspendWorkflowInput
		wantErr       bool
		expectAPICall bool
	}{
		{
			name: "success - suspend running workflow",
			input: SuspendWorkflowInput{
				Name:      "running-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("SuspendWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowSuspendRequest) bool {
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
			validate: func(t *testing.T, output *SuspendWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "running-workflow", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Equal(t, "Running", output.Phase)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "suspended")
			},
		},
		{
			name: "success - uses default namespace",
			input: SuspendWorkflowInput{
				Name: "running-workflow",
				// Namespace not specified - should use default
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("SuspendWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowSuspendRequest) bool {
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
			validate: func(t *testing.T, output *SuspendWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "error - empty name",
			input: SuspendWorkflowInput{
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
			input: SuspendWorkflowInput{
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
			input: SuspendWorkflowInput{
				Name:      "nonexistent-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("SuspendWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("workflows.argoproj.io \"nonexistent-workflow\" not found"),
				)
			},
			wantErr:       true,
			expectAPICall: true,
		},
		{
			name: "error - workflow not running",
			input: SuspendWorkflowInput{
				Name:      "completed-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("SuspendWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("workflow is not running"),
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
			handler := SuspendWorkflowHandler(mockClient)
			ctx := t.Context()
			req := &mcp.CallToolRequest{}

			result, output, err := handler(ctx, req, tt.input)

			// Validate results
			if tt.wantErr {
				require.Error(t, err)
				// Verify no API call was made for validation errors
				if !tt.expectAPICall {
					mockService.AssertNotCalled(t, "SuspendWorkflow", mock.Anything, mock.Anything)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, output)
			tt.validate(t, output, result)
		})
	}
}
