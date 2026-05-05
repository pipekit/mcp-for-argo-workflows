package tools

import (
	"errors"
	"testing"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestDeleteWorkflowTemplateTool(t *testing.T) {
	tool := DeleteWorkflowTemplateTool()

	assert.Equal(t, "delete_workflow_template", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Delete")
	assert.Contains(t, tool.Description, "WorkflowTemplate")
}

func TestDeleteWorkflowTemplateInput(t *testing.T) {
	// Test default values
	input := DeleteWorkflowTemplateInput{}
	assert.Empty(t, input.Namespace)
	assert.Empty(t, input.Name)

	// Test with values
	input2 := DeleteWorkflowTemplateInput{
		Namespace: "test-namespace",
		Name:      "test-template",
	}
	assert.Equal(t, "test-namespace", input2.Namespace)
	assert.Equal(t, "test-template", input2.Name)
}

func TestDeleteWorkflowTemplateOutput(t *testing.T) {
	output := DeleteWorkflowTemplateOutput{
		Name:      "test-template",
		Namespace: "default",
		Message:   "deleted successfully",
	}

	assert.Equal(t, "test-template", output.Name)
	assert.Equal(t, "default", output.Namespace)
	assert.NotEmpty(t, output.Message)
}

func TestDeleteWorkflowTemplateHandler(t *testing.T) {
	tests := []struct {
		setupMock func(*mocks.MockWorkflowTemplateServiceClient)
		validate  func(*testing.T, *DeleteWorkflowTemplateOutput, *mcp.CallToolResult)
		name      string
		input     DeleteWorkflowTemplateInput
		wantErr   bool
	}{
		{
			name: "success - delete template in namespace",
			input: DeleteWorkflowTemplateInput{
				Namespace: "default",
				Name:      "my-template",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("DeleteWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *workflowtemplate.WorkflowTemplateDeleteRequest) bool {
					return req.Namespace == "default" && req.Name == "my-template"
				})).Return(
					&workflowtemplate.WorkflowTemplateDeleteResponse{},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *DeleteWorkflowTemplateOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "my-template", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Contains(t, output.Message, "deleted successfully")
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "my-template")
				assert.Contains(t, text.Text, "default")
				assert.Contains(t, text.Text, "deleted successfully")
			},
		},
		{
			name: "success - uses default namespace",
			input: DeleteWorkflowTemplateInput{
				Name: "my-template",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("DeleteWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *workflowtemplate.WorkflowTemplateDeleteRequest) bool {
					return req.Namespace == "argo" && req.Name == "my-template" // default namespace from mock
				})).Return(
					&workflowtemplate.WorkflowTemplateDeleteResponse{},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *DeleteWorkflowTemplateOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "my-template", output.Name)
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "success - name with whitespace trimmed",
			input: DeleteWorkflowTemplateInput{
				Namespace: "default",
				Name:      "  my-template  ",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("DeleteWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *workflowtemplate.WorkflowTemplateDeleteRequest) bool {
					return req.Name == "my-template"
				})).Return(
					&workflowtemplate.WorkflowTemplateDeleteResponse{},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *DeleteWorkflowTemplateOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "my-template", output.Name)
			},
		},
		{
			name: "error - empty name",
			input: DeleteWorkflowTemplateInput{
				Namespace: "default",
				Name:      "",
			},
			setupMock: func(_ *mocks.MockWorkflowTemplateServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - whitespace only name",
			input: DeleteWorkflowTemplateInput{
				Namespace: "default",
				Name:      "   ",
			},
			setupMock: func(_ *mocks.MockWorkflowTemplateServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - API error (not found)",
			input: DeleteWorkflowTemplateInput{
				Namespace: "default",
				Name:      "nonexistent-template",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("DeleteWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.NotFound, "workflow template not found"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - API error (permission denied)",
			input: DeleteWorkflowTemplateInput{
				Namespace: "default",
				Name:      "protected-template",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("DeleteWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.PermissionDenied, "user does not have permission"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - API error (connection refused)",
			input: DeleteWorkflowTemplateInput{
				Namespace: "default",
				Name:      "my-template",
			},
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("DeleteWorkflowTemplate", mock.Anything, mock.Anything).Return(
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
			mockService := newMockWorkflowTemplateService(t)
			mockClient.SetWorkflowTemplateService(mockService)

			// Setup mock expectations
			tt.setupMock(mockService)

			// Verify mock expectations even on error paths
			defer mockService.AssertExpectations(t)

			// Create handler and call it
			handler := DeleteWorkflowTemplateHandler(mockClient)
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
