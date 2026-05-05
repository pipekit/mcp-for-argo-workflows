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

func TestResumeWorkflowTool(t *testing.T) {
	tool := ResumeWorkflowTool()

	assert.Equal(t, "resume_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Resume", "description should mention resume")
}

func TestResumeWorkflowHandler(t *testing.T) {
	tests := []struct {
		setupMock     func(*mocks.MockWorkflowServiceClient)
		validate      func(*testing.T, *ResumeWorkflowOutput, *mcp.CallToolResult)
		name          string
		input         ResumeWorkflowInput
		wantErr       bool
		expectAPICall bool
	}{
		{
			name: "success - resume suspended workflow",
			input: ResumeWorkflowInput{
				Name:      "suspended-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("ResumeWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowResumeRequest) bool {
					return req.Name == "suspended-workflow" && req.Namespace == "default"
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "suspended-workflow",
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
			validate: func(t *testing.T, output *ResumeWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "suspended-workflow", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Equal(t, "Running", output.Phase)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "resumed")
			},
		},
		{
			name: "success - resume with node field selector",
			input: ResumeWorkflowInput{
				Name:              "suspended-workflow",
				Namespace:         "default",
				NodeFieldSelector: "name=step1",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("ResumeWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowResumeRequest) bool {
					return req.Name == "suspended-workflow" && req.NodeFieldSelector == "name=step1"
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "suspended-workflow",
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
			validate: func(t *testing.T, output *ResumeWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "suspended-workflow", output.Name)
			},
		},
		{
			name: "success - uses default namespace",
			input: ResumeWorkflowInput{
				Name: "suspended-workflow",
				// Namespace not specified - should use default
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("ResumeWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowResumeRequest) bool {
					return req.Namespace == "argo" // default namespace from mock
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "suspended-workflow",
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
			validate: func(t *testing.T, output *ResumeWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "error - empty name",
			input: ResumeWorkflowInput{
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
			input: ResumeWorkflowInput{
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
			input: ResumeWorkflowInput{
				Name:      "nonexistent-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("ResumeWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("workflows.argoproj.io \"nonexistent-workflow\" not found"),
				)
			},
			wantErr:       true,
			expectAPICall: true,
		},
		{
			name: "error - workflow not suspended",
			input: ResumeWorkflowInput{
				Name:      "running-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("ResumeWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("workflow is not suspended"),
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
			handler := ResumeWorkflowHandler(mockClient)
			ctx := t.Context()
			req := &mcp.CallToolRequest{}

			result, output, err := handler(ctx, req, tt.input)

			// Validate results
			if tt.wantErr {
				require.Error(t, err)
				// Verify no API call was made for validation errors
				if !tt.expectAPICall {
					mockService.AssertNotCalled(t, "ResumeWorkflow", mock.Anything, mock.Anything)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, output)
			tt.validate(t, output, result)
		})
	}
}
