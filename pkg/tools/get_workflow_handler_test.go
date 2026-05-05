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
	"k8s.io/apimachinery/pkg/types"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestGetWorkflowHandler(t *testing.T) {
	tests := []struct {
		setupMock func(*mocks.MockWorkflowServiceClient)
		validate  func(*testing.T, *GetWorkflowOutput)
		input     GetWorkflowInput
		name      string
		wantErr   bool
	}{
		{
			name: "success - get workflow",
			input: GetWorkflowInput{
				Name:      "test-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				startTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
				endTime := time.Date(2025, 1, 15, 10, 5, 30, 0, time.UTC)

				m.On("GetWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowGetRequest) bool {
					return req.Name == "test-workflow" && req.Namespace == "default"
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-workflow",
							Namespace: "default",
							UID:       types.UID("test-uid-123"),
						},
						Spec: wfv1.WorkflowSpec{
							Arguments: wfv1.Arguments{
								Parameters: []wfv1.Parameter{
									{Name: "message", Value: wfv1.AnyStringPtr("hello")},
								},
							},
						},
						Status: wfv1.WorkflowStatus{
							Phase:      wfv1.WorkflowSucceeded,
							Message:    "Workflow completed successfully",
							StartedAt:  metav1.Time{Time: startTime},
							FinishedAt: metav1.Time{Time: endTime},
							Progress:   "3/3",
							Nodes: wfv1.Nodes{
								"node1": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded},
								"node2": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded},
								"node3": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetWorkflowOutput) {
				assert.Equal(t, "test-workflow", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Equal(t, "test-uid-123", output.UID)
				assert.Equal(t, "Succeeded", output.Phase)
				assert.Equal(t, "Workflow completed successfully", output.Message)
				assert.Equal(t, "2025-01-15T10:00:00Z", output.StartedAt)
				assert.Equal(t, "2025-01-15T10:05:30Z", output.FinishedAt)
				assert.Equal(t, "5m30s", output.Duration)
				assert.Equal(t, "3/3", output.Progress)
				require.Len(t, output.Parameters, 1)
				assert.Equal(t, "message", output.Parameters[0].Name)
				assert.Equal(t, "hello", output.Parameters[0].Value)
				require.NotNil(t, output.NodeSummary)
				assert.Equal(t, 3, output.NodeSummary.Total)
				assert.Equal(t, 3, output.NodeSummary.Succeeded)
			},
		},
		{
			name: "success - get workflow with default namespace",
			input: GetWorkflowInput{
				Name: "test-workflow",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("GetWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowGetRequest) bool {
					return req.Name == "test-workflow" && req.Namespace == "argo"
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-workflow",
							Namespace: "argo",
							UID:       types.UID("test-uid"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowRunning,
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetWorkflowOutput) {
				assert.Equal(t, "test-workflow", output.Name)
				assert.Equal(t, "argo", output.Namespace)
				assert.Equal(t, "Running", output.Phase)
			},
		},
		{
			name: "success - pending workflow with no status",
			input: GetWorkflowInput{
				Name:      "pending-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("GetWorkflow", mock.Anything, mock.Anything).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "pending-workflow",
							Namespace: "default",
							UID:       types.UID("pending-uid"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: "",
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetWorkflowOutput) {
				assert.Equal(t, "pending-workflow", output.Name)
				assert.Equal(t, "Pending", output.Phase) // Default when empty
			},
		},
		{
			name: "success - failed workflow",
			input: GetWorkflowInput{
				Name:      "failed-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("GetWorkflow", mock.Anything, mock.Anything).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "failed-workflow",
							Namespace: "default",
							UID:       types.UID("failed-uid"),
						},
						Status: wfv1.WorkflowStatus{
							Phase:   wfv1.WorkflowFailed,
							Message: "Step failed",
							Nodes: wfv1.Nodes{
								"node1": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded},
								"node2": wfv1.NodeStatus{Phase: wfv1.NodeFailed},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetWorkflowOutput) {
				assert.Equal(t, "Failed", output.Phase)
				assert.Equal(t, "Step failed", output.Message)
				require.NotNil(t, output.NodeSummary)
				assert.Equal(t, 2, output.NodeSummary.Total)
				assert.Equal(t, 1, output.NodeSummary.Succeeded)
				assert.Equal(t, 1, output.NodeSummary.Failed)
			},
		},
		{
			name: "error - empty name",
			input: GetWorkflowInput{
				Name:      "",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail validation
			},
			wantErr: true,
		},
		{
			name: "error - workflow not found",
			input: GetWorkflowInput{
				Name:      "missing-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("GetWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.NotFound, "workflow not found"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - permission denied",
			input: GetWorkflowInput{
				Name:      "protected-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("GetWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.PermissionDenied, "user does not have permission to get workflows"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - invalid namespace",
			input: GetWorkflowInput{
				Name:      "test-workflow",
				Namespace: "invalid-namespace",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("GetWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.NotFound, "namespace not found"),
				)
			},
			wantErr: true,
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
			handler := GetWorkflowHandler(mockClient)
			ctx := t.Context()
			req := &mcp.CallToolRequest{}

			result, output, err := handler(ctx, req, tt.input)

			// Validate results
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Nil(t, result) // Handler returns nil for result
			require.NotNil(t, output)
			tt.validate(t, output)
		})
	}
}
