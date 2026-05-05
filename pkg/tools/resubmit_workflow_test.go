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

func TestResubmitWorkflowTool(t *testing.T) {
	tool := ResubmitWorkflowTool()

	assert.Equal(t, "resubmit_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Resubmit", "description should mention resubmit")
}

func TestResubmitWorkflowHandler(t *testing.T) {
	tests := []struct {
		setupMock     func(*mocks.MockWorkflowServiceClient)
		validate      func(*testing.T, *ResubmitWorkflowOutput, *mcp.CallToolResult)
		name          string
		input         ResubmitWorkflowInput
		wantErr       bool
		expectAPICall bool
	}{
		{
			name: "success - basic resubmit",
			input: ResubmitWorkflowInput{
				Name:      "completed-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("ResubmitWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowResubmitRequest) bool {
					return req.Name == "completed-workflow" && req.Namespace == "default"
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "completed-workflow-resubmit-12345",
							Namespace: "default",
							UID:       types.UID("new-uid-123"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowPending,
						},
					},
					nil,
				)
			},
			wantErr:       false,
			expectAPICall: true,
			validate: func(t *testing.T, output *ResubmitWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "completed-workflow-resubmit-12345", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Equal(t, "new-uid-123", output.UID)
				assert.Equal(t, "Pending", output.Phase)
				assert.Equal(t, "completed-workflow", output.OriginalWorkflow)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "resubmitted")
				assert.Contains(t, text.Text, "completed-workflow-resubmit-12345")
			},
		},
		{
			name: "success - resubmit with memoized",
			input: ResubmitWorkflowInput{
				Name:      "completed-workflow",
				Namespace: "default",
				Memoized:  true,
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("ResubmitWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowResubmitRequest) bool {
					return req.Name == "completed-workflow" && req.Memoized == true
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "completed-workflow-resubmit-12345",
							Namespace: "default",
							UID:       types.UID("new-uid-123"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowPending,
						},
					},
					nil,
				)
			},
			wantErr:       false,
			expectAPICall: true,
			validate: func(t *testing.T, output *ResubmitWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "completed-workflow-resubmit-12345", output.Name)
			},
		},
		{
			name: "success - resubmit with parameters",
			input: ResubmitWorkflowInput{
				Name:       "completed-workflow",
				Namespace:  "default",
				Parameters: []string{"param1=newvalue1", "param2=newvalue2"},
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("ResubmitWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowResubmitRequest) bool {
					return req.Name == "completed-workflow" &&
						len(req.Parameters) == 2 &&
						req.Parameters[0] == "param1=newvalue1" &&
						req.Parameters[1] == "param2=newvalue2"
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "completed-workflow-resubmit-12345",
							Namespace: "default",
							UID:       types.UID("new-uid-123"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowPending,
						},
					},
					nil,
				)
			},
			wantErr:       false,
			expectAPICall: true,
			validate: func(t *testing.T, output *ResubmitWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "completed-workflow-resubmit-12345", output.Name)
			},
		},
		{
			name: "success - uses default namespace",
			input: ResubmitWorkflowInput{
				Name: "completed-workflow",
				// Namespace not specified - should use default
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("ResubmitWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowResubmitRequest) bool {
					return req.Namespace == "argo" // default namespace from mock
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "completed-workflow-resubmit-12345",
							Namespace: "argo",
							UID:       types.UID("new-uid-123"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowPending,
						},
					},
					nil,
				)
			},
			wantErr:       false,
			expectAPICall: true,
			validate: func(t *testing.T, output *ResubmitWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "error - empty name",
			input: ResubmitWorkflowInput{
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
			input: ResubmitWorkflowInput{
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
			input: ResubmitWorkflowInput{
				Name:      "nonexistent-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("ResubmitWorkflow", mock.Anything, mock.Anything).Return(
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
			handler := ResubmitWorkflowHandler(mockClient)
			ctx := t.Context()
			req := &mcp.CallToolRequest{}

			result, output, err := handler(ctx, req, tt.input)

			// Validate results
			if tt.wantErr {
				require.Error(t, err)
				// Verify no API call was made for validation errors
				if !tt.expectAPICall {
					mockService.AssertNotCalled(t, "ResubmitWorkflow", mock.Anything, mock.Anything)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, output)
			tt.validate(t, output, result)
		})
	}
}
