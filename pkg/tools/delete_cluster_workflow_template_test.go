package tools

import (
	"errors"
	"testing"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestDeleteClusterWorkflowTemplateTool(t *testing.T) {
	tool := DeleteClusterWorkflowTemplateTool()

	assert.Equal(t, "delete_cluster_workflow_template", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Delete")
	assert.Contains(t, tool.Description, "ClusterWorkflowTemplate")
}

func TestDeleteClusterWorkflowTemplateInput(t *testing.T) {
	// Test default values
	input := DeleteClusterWorkflowTemplateInput{}
	assert.Empty(t, input.Name)

	// Test with values
	input2 := DeleteClusterWorkflowTemplateInput{
		Name: "test-cluster-template",
	}
	assert.Equal(t, "test-cluster-template", input2.Name)
}

func TestDeleteClusterWorkflowTemplateOutput(t *testing.T) {
	output := DeleteClusterWorkflowTemplateOutput{
		Name:    "test-cluster-template",
		Message: "deleted successfully",
	}

	assert.Equal(t, "test-cluster-template", output.Name)
	assert.NotEmpty(t, output.Message)
}

func TestDeleteClusterWorkflowTemplateHandler(t *testing.T) {
	tests := []struct {
		setupMock func(*mocks.MockClusterWorkflowTemplateServiceClient)
		validate  func(*testing.T, *DeleteClusterWorkflowTemplateOutput, *mcp.CallToolResult)
		name      string
		input     DeleteClusterWorkflowTemplateInput
		wantErr   bool
	}{
		{
			name: "success - delete cluster template",
			input: DeleteClusterWorkflowTemplateInput{
				Name: "my-cluster-template",
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("DeleteClusterWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *clusterworkflowtemplate.ClusterWorkflowTemplateDeleteRequest) bool {
					return req.Name == "my-cluster-template"
				})).Return(
					&clusterworkflowtemplate.ClusterWorkflowTemplateDeleteResponse{},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *DeleteClusterWorkflowTemplateOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "my-cluster-template", output.Name)
				assert.Contains(t, output.Message, "deleted successfully")
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "my-cluster-template")
				assert.Contains(t, text.Text, "deleted successfully")
			},
		},
		{
			name: "success - name with whitespace trimmed",
			input: DeleteClusterWorkflowTemplateInput{
				Name: "  my-cluster-template  ",
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("DeleteClusterWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *clusterworkflowtemplate.ClusterWorkflowTemplateDeleteRequest) bool {
					return req.Name == "my-cluster-template"
				})).Return(
					&clusterworkflowtemplate.ClusterWorkflowTemplateDeleteResponse{},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *DeleteClusterWorkflowTemplateOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "my-cluster-template", output.Name)
			},
		},
		{
			name: "error - empty name",
			input: DeleteClusterWorkflowTemplateInput{
				Name: "",
			},
			setupMock: func(_ *mocks.MockClusterWorkflowTemplateServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - whitespace only name",
			input: DeleteClusterWorkflowTemplateInput{
				Name: "   ",
			},
			setupMock: func(_ *mocks.MockClusterWorkflowTemplateServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - API error (not found)",
			input: DeleteClusterWorkflowTemplateInput{
				Name: "nonexistent-template",
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("DeleteClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.NotFound, "cluster workflow template not found"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - API error (permission denied)",
			input: DeleteClusterWorkflowTemplateInput{
				Name: "protected-template",
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("DeleteClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.PermissionDenied, "user does not have permission"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - API error (connection refused)",
			input: DeleteClusterWorkflowTemplateInput{
				Name: "my-cluster-template",
			},
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("DeleteClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
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
			mockService := newMockClusterWorkflowTemplateService(t)
			mockClient.SetClusterWorkflowTemplateService(mockService)

			// Setup mock expectations
			tt.setupMock(mockService)

			// Verify mock expectations even on error paths
			defer mockService.AssertExpectations(t)

			// Create handler and call it
			handler := DeleteClusterWorkflowTemplateHandler(mockClient)
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
