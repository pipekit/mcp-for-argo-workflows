package tools

import (
	"errors"
	"testing"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowarchive"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

// =============================================================================
// Delete Archived Workflow Tests
// =============================================================================

func TestDeleteArchivedWorkflowTool(t *testing.T) {
	tool := DeleteArchivedWorkflowTool()

	assert.Equal(t, "delete_archived_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Delete")
	assert.Contains(t, tool.Description, "archived workflow")
	assert.Contains(t, tool.Description, "Argo Server")
}

func TestDeleteArchivedWorkflowInput(t *testing.T) {
	// Test default values
	input := DeleteArchivedWorkflowInput{}
	assert.Empty(t, input.UID)

	// Test with values
	input2 := DeleteArchivedWorkflowInput{
		UID: "test-uid-123",
	}
	assert.Equal(t, "test-uid-123", input2.UID)
}

func TestDeleteArchivedWorkflowHandler(t *testing.T) {
	tests := []struct {
		setupMock func(*mocks.MockArchivedWorkflowServiceClient)
		validate  func(*testing.T, *DeleteArchivedWorkflowOutput, *mcp.CallToolResult)
		name      string
		input     DeleteArchivedWorkflowInput
		wantErr   bool
	}{
		{
			name: "success - delete archived workflow",
			input: DeleteArchivedWorkflowInput{
				UID: "test-uid-123",
			},
			setupMock: func(m *mocks.MockArchivedWorkflowServiceClient) {
				m.On("DeleteArchivedWorkflow", mock.Anything, mock.MatchedBy(func(req *workflowarchive.DeleteArchivedWorkflowRequest) bool {
					return req.Uid == "test-uid-123"
				})).Return(&workflowarchive.ArchivedWorkflowDeletedResponse{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *DeleteArchivedWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "test-uid-123", output.UID)
				assert.Contains(t, output.Message, "deleted successfully")
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "deleted successfully")
			},
		},
		{
			name: "error - empty UID",
			input: DeleteArchivedWorkflowInput{
				UID: "",
			},
			setupMock: func(_ *mocks.MockArchivedWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - whitespace only UID",
			input: DeleteArchivedWorkflowInput{
				UID: "   ",
			},
			setupMock: func(_ *mocks.MockArchivedWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - API error (not found)",
			input: DeleteArchivedWorkflowInput{
				UID: "nonexistent-uid",
			},
			setupMock: func(m *mocks.MockArchivedWorkflowServiceClient) {
				m.On("DeleteArchivedWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.NotFound, "archived workflow not found"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - API error (permission denied)",
			input: DeleteArchivedWorkflowInput{
				UID: "protected-uid",
			},
			setupMock: func(m *mocks.MockArchivedWorkflowServiceClient) {
				m.On("DeleteArchivedWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.PermissionDenied, "user does not have permission"),
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client and service
			mockClient := newMockClient(t, "argo", true)
			mockService := newMockArchivedWorkflowService(t)
			mockClient.SetArchivedWorkflowService(mockService)

			// Setup mock expectations
			tt.setupMock(mockService)

			// Verify mock expectations even on error paths
			defer mockService.AssertExpectations(t)

			// Create handler and call it
			handler := DeleteArchivedWorkflowHandler(mockClient)
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

func TestDeleteArchivedWorkflowHandler_ServiceError(t *testing.T) {
	// Create mock client that returns error for ArchivedWorkflowService
	mockClient := mocks.NewMockClient("argo", true)
	mockClient.On("ArchivedWorkflowService").Return(nil, errors.New("not in argo server mode"))

	handler := DeleteArchivedWorkflowHandler(mockClient)
	ctx := t.Context()
	req := &mcp.CallToolRequest{}
	input := DeleteArchivedWorkflowInput{UID: "test-uid"}

	_, _, err := handler(ctx, req, input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get archived workflow service")
}

// =============================================================================
// Resubmit Archived Workflow Tests
// =============================================================================

func TestResubmitArchivedWorkflowTool(t *testing.T) {
	tool := ResubmitArchivedWorkflowTool()

	assert.Equal(t, "resubmit_archived_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Resubmit")
	assert.Contains(t, tool.Description, "archived workflow")
	assert.Contains(t, tool.Description, "Argo Server")
}

func TestResubmitArchivedWorkflowInput(t *testing.T) {
	// Test default values
	input := ResubmitArchivedWorkflowInput{}
	assert.Empty(t, input.UID)
	assert.Empty(t, input.Namespace)
	assert.False(t, input.Memoized)

	// Test with values
	input2 := ResubmitArchivedWorkflowInput{
		UID:       "test-uid-123",
		Namespace: "custom-ns",
		Memoized:  true,
	}
	assert.Equal(t, "test-uid-123", input2.UID)
	assert.Equal(t, "custom-ns", input2.Namespace)
	assert.True(t, input2.Memoized)
}

func TestResubmitArchivedWorkflowHandler(t *testing.T) {
	tests := []struct {
		setupMock func(*mocks.MockArchivedWorkflowServiceClient)
		validate  func(*testing.T, *ResubmitArchivedWorkflowOutput, *mcp.CallToolResult)
		name      string
		input     ResubmitArchivedWorkflowInput
		wantErr   bool
	}{
		{
			name: "success - resubmit archived workflow",
			input: ResubmitArchivedWorkflowInput{
				UID: "test-uid-123",
			},
			setupMock: func(m *mocks.MockArchivedWorkflowServiceClient) {
				m.On("ResubmitArchivedWorkflow", mock.Anything, mock.MatchedBy(func(req *workflowarchive.ResubmitArchivedWorkflowRequest) bool {
					return req.Uid == "test-uid-123"
				})).Return(&wfv1.Workflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-workflow-resubmit",
						Namespace: "default",
						UID:       types.UID("new-uid-456"),
					},
				}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *ResubmitArchivedWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "my-workflow-resubmit", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Equal(t, "new-uid-456", output.UID)
				assert.Contains(t, output.Message, "resubmitted")
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "my-workflow-resubmit")
			},
		},
		{
			name: "success - resubmit with custom namespace",
			input: ResubmitArchivedWorkflowInput{
				UID:       "test-uid-123",
				Namespace: "custom-ns",
			},
			setupMock: func(m *mocks.MockArchivedWorkflowServiceClient) {
				m.On("ResubmitArchivedWorkflow", mock.Anything, mock.MatchedBy(func(req *workflowarchive.ResubmitArchivedWorkflowRequest) bool {
					return req.Uid == "test-uid-123" && req.Namespace == "custom-ns"
				})).Return(&wfv1.Workflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-workflow-resubmit",
						Namespace: "custom-ns",
						UID:       types.UID("new-uid-789"),
					},
				}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *ResubmitArchivedWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "my-workflow-resubmit", output.Name)
				assert.Equal(t, "custom-ns", output.Namespace)
				assert.Equal(t, "new-uid-789", output.UID)
			},
		},
		{
			name: "success - resubmit with memoization",
			input: ResubmitArchivedWorkflowInput{
				UID:      "test-uid-123",
				Memoized: true,
			},
			setupMock: func(m *mocks.MockArchivedWorkflowServiceClient) {
				m.On("ResubmitArchivedWorkflow", mock.Anything, mock.MatchedBy(func(req *workflowarchive.ResubmitArchivedWorkflowRequest) bool {
					return req.Uid == "test-uid-123" && req.Memoized == true
				})).Return(&wfv1.Workflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-workflow-memoized",
						Namespace: "default",
						UID:       types.UID("memoized-uid"),
					},
				}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *ResubmitArchivedWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "my-workflow-memoized", output.Name)
			},
		},
		{
			name: "error - empty UID",
			input: ResubmitArchivedWorkflowInput{
				UID: "",
			},
			setupMock: func(_ *mocks.MockArchivedWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - whitespace only UID",
			input: ResubmitArchivedWorkflowInput{
				UID: "   ",
			},
			setupMock: func(_ *mocks.MockArchivedWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - API error (not found)",
			input: ResubmitArchivedWorkflowInput{
				UID: "nonexistent-uid",
			},
			setupMock: func(m *mocks.MockArchivedWorkflowServiceClient) {
				m.On("ResubmitArchivedWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.NotFound, "archived workflow not found"),
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client and service
			mockClient := newMockClient(t, "argo", true)
			mockService := newMockArchivedWorkflowService(t)
			mockClient.SetArchivedWorkflowService(mockService)

			// Setup mock expectations
			tt.setupMock(mockService)

			// Verify mock expectations even on error paths
			defer mockService.AssertExpectations(t)

			// Create handler and call it
			handler := ResubmitArchivedWorkflowHandler(mockClient)
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

func TestResubmitArchivedWorkflowHandler_ServiceError(t *testing.T) {
	// Create mock client that returns error for ArchivedWorkflowService
	mockClient := mocks.NewMockClient("argo", true)
	mockClient.On("ArchivedWorkflowService").Return(nil, errors.New("not in argo server mode"))

	handler := ResubmitArchivedWorkflowHandler(mockClient)
	ctx := t.Context()
	req := &mcp.CallToolRequest{}
	input := ResubmitArchivedWorkflowInput{UID: "test-uid"}

	_, _, err := handler(ctx, req, input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get archived workflow service")
}

// =============================================================================
// Retry Archived Workflow Tests
// =============================================================================

func TestRetryArchivedWorkflowTool(t *testing.T) {
	tool := RetryArchivedWorkflowTool()

	assert.Equal(t, "retry_archived_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "Retry")
	assert.Contains(t, tool.Description, "archived workflow")
	assert.Contains(t, tool.Description, "Argo Server")
}

func TestRetryArchivedWorkflowInput(t *testing.T) {
	// Test default values
	input := RetryArchivedWorkflowInput{}
	assert.Empty(t, input.UID)
	assert.Empty(t, input.Namespace)
	assert.False(t, input.RestartSuccessful)
	assert.Empty(t, input.NodeFieldSelector)

	// Test with values
	input2 := RetryArchivedWorkflowInput{
		UID:               "test-uid-123",
		Namespace:         "custom-ns",
		RestartSuccessful: true,
		NodeFieldSelector: "phase=Failed",
	}
	assert.Equal(t, "test-uid-123", input2.UID)
	assert.Equal(t, "custom-ns", input2.Namespace)
	assert.True(t, input2.RestartSuccessful)
	assert.Equal(t, "phase=Failed", input2.NodeFieldSelector)
}

func TestRetryArchivedWorkflowHandler(t *testing.T) {
	tests := []struct {
		setupMock func(*mocks.MockArchivedWorkflowServiceClient)
		validate  func(*testing.T, *RetryArchivedWorkflowOutput, *mcp.CallToolResult)
		name      string
		input     RetryArchivedWorkflowInput
		wantErr   bool
	}{
		{
			name: "success - retry archived workflow",
			input: RetryArchivedWorkflowInput{
				UID: "test-uid-123",
			},
			setupMock: func(m *mocks.MockArchivedWorkflowServiceClient) {
				m.On("RetryArchivedWorkflow", mock.Anything, mock.MatchedBy(func(req *workflowarchive.RetryArchivedWorkflowRequest) bool {
					return req.Uid == "test-uid-123"
				})).Return(&wfv1.Workflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-workflow-retry",
						Namespace: "default",
						UID:       types.UID("retry-uid-456"),
					},
				}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *RetryArchivedWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "my-workflow-retry", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Equal(t, "retry-uid-456", output.UID)
				assert.Contains(t, output.Message, "retried")
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "my-workflow-retry")
			},
		},
		{
			name: "success - retry with custom namespace",
			input: RetryArchivedWorkflowInput{
				UID:       "test-uid-123",
				Namespace: "custom-ns",
			},
			setupMock: func(m *mocks.MockArchivedWorkflowServiceClient) {
				m.On("RetryArchivedWorkflow", mock.Anything, mock.MatchedBy(func(req *workflowarchive.RetryArchivedWorkflowRequest) bool {
					return req.Uid == "test-uid-123" && req.Namespace == "custom-ns"
				})).Return(&wfv1.Workflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-workflow-retry",
						Namespace: "custom-ns",
						UID:       types.UID("retry-uid-789"),
					},
				}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *RetryArchivedWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "my-workflow-retry", output.Name)
				assert.Equal(t, "custom-ns", output.Namespace)
				assert.Equal(t, "retry-uid-789", output.UID)
			},
		},
		{
			name: "success - retry with restart successful",
			input: RetryArchivedWorkflowInput{
				UID:               "test-uid-123",
				RestartSuccessful: true,
			},
			setupMock: func(m *mocks.MockArchivedWorkflowServiceClient) {
				m.On("RetryArchivedWorkflow", mock.Anything, mock.MatchedBy(func(req *workflowarchive.RetryArchivedWorkflowRequest) bool {
					return req.Uid == "test-uid-123" && req.RestartSuccessful == true
				})).Return(&wfv1.Workflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-workflow-restart",
						Namespace: "default",
						UID:       types.UID("restart-uid"),
					},
				}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *RetryArchivedWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "my-workflow-restart", output.Name)
			},
		},
		{
			name: "success - retry with node field selector",
			input: RetryArchivedWorkflowInput{
				UID:               "test-uid-123",
				NodeFieldSelector: "phase=Failed",
			},
			setupMock: func(m *mocks.MockArchivedWorkflowServiceClient) {
				m.On("RetryArchivedWorkflow", mock.Anything, mock.MatchedBy(func(req *workflowarchive.RetryArchivedWorkflowRequest) bool {
					return req.Uid == "test-uid-123" && req.NodeFieldSelector == "phase=Failed"
				})).Return(&wfv1.Workflow{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "my-workflow-selector",
						Namespace: "default",
						UID:       types.UID("selector-uid"),
					},
				}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *RetryArchivedWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "my-workflow-selector", output.Name)
			},
		},
		{
			name: "error - empty UID",
			input: RetryArchivedWorkflowInput{
				UID: "",
			},
			setupMock: func(_ *mocks.MockArchivedWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - whitespace only UID",
			input: RetryArchivedWorkflowInput{
				UID: "   ",
			},
			setupMock: func(_ *mocks.MockArchivedWorkflowServiceClient) {
				// No mock needed - should fail validation before API call
			},
			wantErr: true,
		},
		{
			name: "error - API error (not found)",
			input: RetryArchivedWorkflowInput{
				UID: "nonexistent-uid",
			},
			setupMock: func(m *mocks.MockArchivedWorkflowServiceClient) {
				m.On("RetryArchivedWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.NotFound, "archived workflow not found"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - API error (workflow not failed)",
			input: RetryArchivedWorkflowInput{
				UID: "succeeded-uid",
			},
			setupMock: func(m *mocks.MockArchivedWorkflowServiceClient) {
				m.On("RetryArchivedWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.InvalidArgument, "workflow is not in a failed state"),
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client and service
			mockClient := newMockClient(t, "argo", true)
			mockService := newMockArchivedWorkflowService(t)
			mockClient.SetArchivedWorkflowService(mockService)

			// Setup mock expectations
			tt.setupMock(mockService)

			// Verify mock expectations even on error paths
			defer mockService.AssertExpectations(t)

			// Create handler and call it
			handler := RetryArchivedWorkflowHandler(mockClient)
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

func TestRetryArchivedWorkflowHandler_ServiceError(t *testing.T) {
	// Create mock client that returns error for ArchivedWorkflowService
	mockClient := mocks.NewMockClient("argo", true)
	mockClient.On("ArchivedWorkflowService").Return(nil, errors.New("not in argo server mode"))

	handler := RetryArchivedWorkflowHandler(mockClient)
	ctx := t.Context()
	req := &mcp.CallToolRequest{}
	input := RetryArchivedWorkflowInput{UID: "test-uid"}

	_, _, err := handler(ctx, req, input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get archived workflow service")
}

// =============================================================================
// Helper Functions
// =============================================================================

// newMockArchivedWorkflowService creates a new mock archived workflow service client for testing.
func newMockArchivedWorkflowService(t *testing.T) *mocks.MockArchivedWorkflowServiceClient {
	t.Helper()
	return &mocks.MockArchivedWorkflowServiceClient{}
}
