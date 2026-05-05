package tools

import (
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestListWorkflowsHandler(t *testing.T) {
	tests := []struct {
		setupMock func(*mocks.MockWorkflowServiceClient)
		validate  func(*testing.T, *mcp.CallToolResult, *ListWorkflowsOutput)
		name      string
		input     ListWorkflowsInput
		wantErr   bool
	}{
		{
			name: "success - list all workflows",
			input: ListWorkflowsInput{
				Namespace: stringPtr("default"),
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				now := time.Now()
				m.On("ListWorkflows", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowListRequest) bool {
					return req.Namespace == "default"
				})).Return(
					&wfv1.WorkflowList{
						Items: wfv1.Workflows{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "workflow-1",
									Namespace:         "default",
									CreationTimestamp: metav1.Time{Time: now.Add(-1 * time.Hour)},
								},
								Status: wfv1.WorkflowStatus{
									Phase: wfv1.WorkflowSucceeded,
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "workflow-2",
									Namespace:         "default",
									CreationTimestamp: metav1.Time{Time: now.Add(-30 * time.Minute)},
								},
								Status: wfv1.WorkflowStatus{
									Phase:   wfv1.WorkflowRunning,
									Message: "Step 2 of 5",
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, result *mcp.CallToolResult, output *ListWorkflowsOutput) {
				assert.False(t, result.IsError)
				assert.Len(t, result.Content, 1)
				require.IsType(t, &mcp.TextContent{}, result.Content[0])
				textContent, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok, "content should be TextContent")
				assert.Contains(t, textContent.Text, "Found 2 workflow(s)")
				assert.Equal(t, 2, output.Total)
				require.Len(t, output.Workflows, 2)
				assert.Equal(t, "workflow-1", output.Workflows[0].Name)
				assert.Equal(t, "Succeeded", output.Workflows[0].Phase)
				assert.Equal(t, "workflow-2", output.Workflows[1].Name)
				assert.Equal(t, "Running", output.Workflows[1].Phase)
				assert.Equal(t, "Step 2 of 5", output.Workflows[1].Message)
			},
		},
		{
			name: "success - list with status filter",
			input: ListWorkflowsInput{
				Namespace: stringPtr("default"),
				Status:    []string{"Running", "Pending"},
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				now := time.Now()
				m.On("ListWorkflows", mock.Anything, mock.Anything).Return(
					&wfv1.WorkflowList{
						Items: wfv1.Workflows{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "workflow-1",
									Namespace:         "default",
									CreationTimestamp: metav1.Time{Time: now},
								},
								Status: wfv1.WorkflowStatus{
									Phase: wfv1.WorkflowRunning,
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "workflow-2",
									Namespace:         "default",
									CreationTimestamp: metav1.Time{Time: now},
								},
								Status: wfv1.WorkflowStatus{
									Phase: wfv1.WorkflowSucceeded, // Should be filtered out
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "workflow-3",
									Namespace:         "default",
									CreationTimestamp: metav1.Time{Time: now},
								},
								Status: wfv1.WorkflowStatus{
									Phase: wfv1.WorkflowPending,
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, result *mcp.CallToolResult, output *ListWorkflowsOutput) {
				assert.Equal(t, 2, output.Total)
				require.Len(t, output.Workflows, 2)
				assert.Equal(t, "workflow-1", output.Workflows[0].Name)
				assert.Equal(t, "Running", output.Workflows[0].Phase)
				assert.Equal(t, "workflow-3", output.Workflows[1].Name)
				assert.Equal(t, "Pending", output.Workflows[1].Phase)
			},
		},
		{
			name: "success - list with limit",
			input: ListWorkflowsInput{
				Namespace: stringPtr("default"),
				Limit:     2,
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				now := time.Now()
				m.On("ListWorkflows", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowListRequest) bool {
					return req.ListOptions.Limit == 2
				})).Return(
					&wfv1.WorkflowList{
						Items: wfv1.Workflows{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "workflow-1",
									Namespace:         "default",
									CreationTimestamp: metav1.Time{Time: now},
								},
								Status: wfv1.WorkflowStatus{
									Phase: wfv1.WorkflowRunning,
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "workflow-2",
									Namespace:         "default",
									CreationTimestamp: metav1.Time{Time: now},
								},
								Status: wfv1.WorkflowStatus{
									Phase: wfv1.WorkflowSucceeded,
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, result *mcp.CallToolResult, output *ListWorkflowsOutput) {
				assert.Equal(t, 2, output.Total)
				require.Len(t, output.Workflows, 2)
			},
		},
		{
			name: "success - list with labels",
			input: ListWorkflowsInput{
				Namespace: stringPtr("default"),
				Labels:    "app=myapp,env=prod",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				now := time.Now()
				m.On("ListWorkflows", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowListRequest) bool {
					return req.ListOptions.LabelSelector == "app=myapp,env=prod"
				})).Return(
					&wfv1.WorkflowList{
						Items: wfv1.Workflows{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "filtered-workflow",
									Namespace:         "default",
									CreationTimestamp: metav1.Time{Time: now},
								},
								Status: wfv1.WorkflowStatus{
									Phase: wfv1.WorkflowSucceeded,
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, result *mcp.CallToolResult, output *ListWorkflowsOutput) {
				assert.Equal(t, 1, output.Total)
				require.Len(t, output.Workflows, 1)
				assert.Equal(t, "filtered-workflow", output.Workflows[0].Name)
			},
		},
		{
			name:  "success - list with default namespace",
			input: ListWorkflowsInput{
				// No namespace specified, should use default
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("ListWorkflows", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowListRequest) bool {
					return req.Namespace == "argo" // Default namespace from mock
				})).Return(
					&wfv1.WorkflowList{
						Items: wfv1.Workflows{},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, result *mcp.CallToolResult, output *ListWorkflowsOutput) {
				assert.Equal(t, 0, output.Total)
				assert.NotNil(t, output.Workflows)
				assert.Empty(t, output.Workflows)
			},
		},
		{
			name: "success - list all namespaces",
			input: ListWorkflowsInput{
				Namespace: stringPtr(""), // Empty string means all namespaces
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				now := time.Now()
				m.On("ListWorkflows", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowListRequest) bool {
					return req.Namespace == ""
				})).Return(
					&wfv1.WorkflowList{
						Items: wfv1.Workflows{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "workflow-ns1",
									Namespace:         "namespace1",
									CreationTimestamp: metav1.Time{Time: now},
								},
								Status: wfv1.WorkflowStatus{
									Phase: wfv1.WorkflowSucceeded,
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "workflow-ns2",
									Namespace:         "namespace2",
									CreationTimestamp: metav1.Time{Time: now},
								},
								Status: wfv1.WorkflowStatus{
									Phase: wfv1.WorkflowRunning,
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, result *mcp.CallToolResult, output *ListWorkflowsOutput) {
				require.IsType(t, &mcp.TextContent{}, result.Content[0])
				textContent, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok, "content should be TextContent")
				assert.Contains(t, textContent.Text, "across all namespaces")
				assert.Equal(t, 2, output.Total)
				require.Len(t, output.Workflows, 2)
				assert.Equal(t, "namespace1", output.Workflows[0].Namespace)
				assert.Equal(t, "namespace2", output.Workflows[1].Namespace)
			},
		},
		{
			name: "success - empty result",
			input: ListWorkflowsInput{
				Namespace: stringPtr("default"),
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("ListWorkflows", mock.Anything, mock.Anything).Return(
					&wfv1.WorkflowList{
						Items: wfv1.Workflows{},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, result *mcp.CallToolResult, output *ListWorkflowsOutput) {
				assert.Equal(t, 0, output.Total)
				assert.NotNil(t, output.Workflows)
				assert.Empty(t, output.Workflows)
			},
		},
		{
			name: "error - invalid status filter",
			input: ListWorkflowsInput{
				Namespace: stringPtr("default"),
				Status:    []string{"InvalidStatus"},
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail validation
			},
			wantErr: true,
		},
		{
			name: "error - API error (permission denied)",
			input: ListWorkflowsInput{
				Namespace: stringPtr("default"),
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("ListWorkflows", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.PermissionDenied, "user does not have permission to list workflows"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - API error (namespace not found)",
			input: ListWorkflowsInput{
				Namespace: stringPtr("invalid-namespace"),
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("ListWorkflows", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.NotFound, "namespace not found"),
				)
			},
			wantErr: true,
		},
		{
			name: "success - client-side limit with status filter",
			input: ListWorkflowsInput{
				Namespace: stringPtr("default"),
				Status:    []string{"Running"},
				Limit:     1,
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				now := time.Now()
				// When status filter is active, limit is not passed to server
				m.On("ListWorkflows", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowListRequest) bool {
					return req.ListOptions.Limit == 0 // Server-side limit should be 0
				})).Return(
					&wfv1.WorkflowList{
						Items: wfv1.Workflows{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "workflow-1",
									Namespace:         "default",
									CreationTimestamp: metav1.Time{Time: now},
								},
								Status: wfv1.WorkflowStatus{
									Phase: wfv1.WorkflowRunning,
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "workflow-2",
									Namespace:         "default",
									CreationTimestamp: metav1.Time{Time: now},
								},
								Status: wfv1.WorkflowStatus{
									Phase: wfv1.WorkflowRunning,
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, result *mcp.CallToolResult, output *ListWorkflowsOutput) {
				// Client-side limit should apply after filtering
				assert.Equal(t, 1, output.Total)
				require.Len(t, output.Workflows, 1)
				assert.Equal(t, "workflow-1", output.Workflows[0].Name)
			},
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
			handler := ListWorkflowsHandler(mockClient)
			ctx := t.Context()
			req := &mcp.CallToolRequest{}

			result, output, err := handler(ctx, req, tt.input)

			// Validate results
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.False(t, result.IsError)
			require.NotNil(t, output)
			tt.validate(t, result, output)
		})
	}
}

// stringPtr returns a pointer to the given string.
func stringPtr(s string) *string {
	return &s
}
