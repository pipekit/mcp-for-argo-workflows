package tools

import (
	"errors"
	"testing"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestDeleteCronWorkflowTool(t *testing.T) {
	tool := DeleteCronWorkflowTool()

	assert.Equal(t, "delete_cron_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Delete")
	assert.Contains(t, tool.Description, "CronWorkflow")
}

func TestDeleteCronWorkflowInput(t *testing.T) {
	// Test default values
	input := DeleteCronWorkflowInput{}
	assert.Empty(t, input.Namespace)
	assert.Empty(t, input.Name)

	// Test with values
	input2 := DeleteCronWorkflowInput{
		Namespace: "test-namespace",
		Name:      "test-cron",
	}
	assert.Equal(t, "test-namespace", input2.Namespace)
	assert.Equal(t, "test-cron", input2.Name)
}

func TestDeleteCronWorkflowOutput(t *testing.T) {
	output := DeleteCronWorkflowOutput{
		Name:      "test-cron",
		Namespace: "default",
		Message:   "deleted successfully",
	}

	assert.Equal(t, "test-cron", output.Name)
	assert.Equal(t, "default", output.Namespace)
	assert.NotEmpty(t, output.Message)
}

func TestDeleteCronWorkflowHandler(t *testing.T) {
	tests := []struct {
		setupMock func(*mocks.MockCronWorkflowServiceClient)
		validate  func(*testing.T, *DeleteCronWorkflowOutput, *mcp.CallToolResult)
		name      string
		input     DeleteCronWorkflowInput
		wantErr   bool
	}{
		{
			name: "success - delete cron workflow in namespace",
			input: DeleteCronWorkflowInput{
				Namespace: "default",
				Name:      "my-cron",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("DeleteCronWorkflow", mock.Anything, mock.MatchedBy(func(req *cronworkflow.DeleteCronWorkflowRequest) bool {
					return req.Namespace == "default" && req.Name == "my-cron"
				})).Return(
					&cronworkflow.CronWorkflowDeletedResponse{},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *DeleteCronWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "my-cron", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Contains(t, output.Message, "deleted successfully")
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "my-cron")
				assert.Contains(t, text.Text, "default")
				assert.Contains(t, text.Text, "deleted successfully")
			},
		},
		{
			name: "success - uses default namespace",
			input: DeleteCronWorkflowInput{
				Name: "my-cron",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("DeleteCronWorkflow", mock.Anything, mock.MatchedBy(func(req *cronworkflow.DeleteCronWorkflowRequest) bool {
					return req.Namespace == "argo" && req.Name == "my-cron" // default namespace from mock
				})).Return(
					&cronworkflow.CronWorkflowDeletedResponse{},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *DeleteCronWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "my-cron", output.Name)
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "success - name with whitespace trimmed",
			input: DeleteCronWorkflowInput{
				Namespace: "default",
				Name:      "  my-cron  ",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("DeleteCronWorkflow", mock.Anything, mock.MatchedBy(func(req *cronworkflow.DeleteCronWorkflowRequest) bool {
					return req.Name == "my-cron"
				})).Return(
					&cronworkflow.CronWorkflowDeletedResponse{},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *DeleteCronWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "my-cron", output.Name)
			},
		},
		{
			name: "error - empty name",
			input: DeleteCronWorkflowInput{
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
			input: DeleteCronWorkflowInput{
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
			input: DeleteCronWorkflowInput{
				Namespace: "default",
				Name:      "nonexistent-cron",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("DeleteCronWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.NotFound, "cron workflow not found"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - API error (permission denied)",
			input: DeleteCronWorkflowInput{
				Namespace: "default",
				Name:      "protected-cron",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("DeleteCronWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.PermissionDenied, "user does not have permission"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - API error (connection refused)",
			input: DeleteCronWorkflowInput{
				Namespace: "default",
				Name:      "my-cron",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("DeleteCronWorkflow", mock.Anything, mock.Anything).Return(
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
			handler := DeleteCronWorkflowHandler(mockClient)
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
