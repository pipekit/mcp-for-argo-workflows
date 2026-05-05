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

func TestGetWorkflowNodeTool(t *testing.T) {
	tool := GetWorkflowNodeTool()

	assert.Equal(t, "get_workflow_node", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "node")
}

func TestGetWorkflowNodeInput(t *testing.T) {
	// Test default values
	input := GetWorkflowNodeInput{}
	assert.Empty(t, input.Namespace)
	assert.Empty(t, input.WorkflowName)
	assert.Empty(t, input.NodeName)

	// Test with values
	input2 := GetWorkflowNodeInput{
		Namespace:    "test-namespace",
		WorkflowName: "test-workflow",
		NodeName:     "test-node",
	}
	assert.Equal(t, "test-namespace", input2.Namespace)
	assert.Equal(t, "test-workflow", input2.WorkflowName)
	assert.Equal(t, "test-node", input2.NodeName)
}

func TestGetWorkflowNodeOutput(t *testing.T) {
	output := GetWorkflowNodeOutput{
		ID:           "node-123",
		Name:         "test-workflow[0].step-a",
		DisplayName:  "step-a",
		Type:         "Pod",
		TemplateName: "step-template",
		Phase:        "Succeeded",
		Message:      "completed successfully",
	}

	assert.Equal(t, "node-123", output.ID)
	assert.Equal(t, "test-workflow[0].step-a", output.Name)
	assert.Equal(t, "step-a", output.DisplayName)
	assert.Equal(t, "Pod", output.Type)
	assert.Equal(t, "step-template", output.TemplateName)
	assert.Equal(t, "Succeeded", output.Phase)
	assert.NotEmpty(t, output.Message)
}

func TestGetWorkflowNodeHandler(t *testing.T) {
	startTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 1, 15, 10, 1, 30, 0, time.UTC)

	tests := []struct {
		setupMock func(*mocks.MockWorkflowServiceClient)
		validate  func(*testing.T, *GetWorkflowNodeOutput, *mcp.CallToolResult)
		name      string
		input     GetWorkflowNodeInput
		wantErr   bool
	}{
		{
			name: "success - get node by name",
			input: GetWorkflowNodeInput{
				Namespace:    "default",
				WorkflowName: "test-workflow",
				NodeName:     "test-workflow[0].step-a",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("GetWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowGetRequest) bool {
					return req.Name == "test-workflow" && req.Namespace == "default"
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-workflow",
							Namespace: "default",
							UID:       types.UID("test-uid"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowSucceeded,
							Nodes: wfv1.Nodes{
								"node-abc": wfv1.NodeStatus{
									ID:           "node-abc",
									Name:         "test-workflow[0].step-a",
									DisplayName:  "step-a",
									Type:         wfv1.NodeTypePod,
									TemplateName: "step-template",
									Phase:        wfv1.NodeSucceeded,
									StartedAt:    metav1.Time{Time: startTime},
									FinishedAt:   metav1.Time{Time: endTime},
									HostNodeName: "node-1",
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetWorkflowNodeOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "node-abc", output.ID)
				assert.Equal(t, "test-workflow[0].step-a", output.Name)
				assert.Equal(t, "step-a", output.DisplayName)
				assert.Equal(t, "Pod", output.Type)
				assert.Equal(t, "step-template", output.TemplateName)
				assert.Equal(t, "Succeeded", output.Phase)
				assert.Equal(t, "2025-01-15T10:00:00Z", output.StartedAt)
				assert.Equal(t, "2025-01-15T10:01:30Z", output.FinishedAt)
				assert.Equal(t, "1m30s", output.Duration)
				assert.Equal(t, "node-1", output.HostNodeName)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "step-a")
				assert.Contains(t, text.Text, "Pod")
				assert.Contains(t, text.Text, "Succeeded")
			},
		},
		{
			name: "success - get node by ID",
			input: GetWorkflowNodeInput{
				Namespace:    "default",
				WorkflowName: "test-workflow",
				NodeName:     "node-123",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("GetWorkflow", mock.Anything, mock.Anything).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-workflow",
							Namespace: "default",
							UID:       types.UID("test-uid"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowRunning,
							Nodes: wfv1.Nodes{
								"node-123": wfv1.NodeStatus{
									ID:          "node-123",
									Name:        "test-workflow[0].step-b",
									DisplayName: "step-b",
									Type:        wfv1.NodeTypePod,
									Phase:       wfv1.NodeRunning,
									StartedAt:   metav1.Time{Time: startTime},
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetWorkflowNodeOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "node-123", output.ID)
				assert.Equal(t, "step-b", output.DisplayName)
				assert.Equal(t, "Running", output.Phase)
				assert.Empty(t, output.FinishedAt)
				assert.NotEmpty(t, output.Duration) // Still running
			},
		},
		{
			name: "success - get node by display name",
			input: GetWorkflowNodeInput{
				Namespace:    "default",
				WorkflowName: "test-workflow",
				NodeName:     "step-c",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("GetWorkflow", mock.Anything, mock.Anything).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-workflow",
							Namespace: "default",
							UID:       types.UID("test-uid"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowSucceeded,
							Nodes: wfv1.Nodes{
								"node-xyz": wfv1.NodeStatus{
									ID:          "node-xyz",
									Name:        "test-workflow[0].step-c",
									DisplayName: "step-c",
									Type:        wfv1.NodeTypePod,
									Phase:       wfv1.NodeSucceeded,
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetWorkflowNodeOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "node-xyz", output.ID)
				assert.Equal(t, "step-c", output.DisplayName)
			},
		},
		{
			name: "success - node with inputs and outputs",
			input: GetWorkflowNodeInput{
				Namespace:    "default",
				WorkflowName: "test-workflow",
				NodeName:     "node-io",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				exitCode := "0"
				result := "success"
				m.On("GetWorkflow", mock.Anything, mock.Anything).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-workflow",
							Namespace: "default",
							UID:       types.UID("test-uid"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowSucceeded,
							Nodes: wfv1.Nodes{
								"node-io": wfv1.NodeStatus{
									ID:          "node-io",
									Name:        "test-workflow[0].process",
									DisplayName: "process",
									Type:        wfv1.NodeTypePod,
									Phase:       wfv1.NodeSucceeded,
									Inputs: &wfv1.Inputs{
										Parameters: []wfv1.Parameter{
											{Name: "input-param", Value: wfv1.AnyStringPtr("input-value")},
										},
										Artifacts: []wfv1.Artifact{
											{Name: "input-file", Path: "/tmp/input"},
										},
									},
									Outputs: &wfv1.Outputs{
										Parameters: []wfv1.Parameter{
											{Name: "output-param", Value: wfv1.AnyStringPtr("output-value")},
										},
										Artifacts: []wfv1.Artifact{
											{Name: "output-file", Path: "/tmp/output"},
										},
										ExitCode: &exitCode,
										Result:   &result,
									},
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetWorkflowNodeOutput, result *mcp.CallToolResult) {
				require.NotNil(t, output.Inputs)
				require.Len(t, output.Inputs.Parameters, 1)
				assert.Equal(t, "input-param", output.Inputs.Parameters[0].Name)
				assert.Equal(t, "input-value", output.Inputs.Parameters[0].Value)
				require.Len(t, output.Inputs.Artifacts, 1)
				assert.Equal(t, "input-file", output.Inputs.Artifacts[0].Name)
				assert.Equal(t, "/tmp/input", output.Inputs.Artifacts[0].Path)

				require.NotNil(t, output.Outputs)
				require.Len(t, output.Outputs.Parameters, 1)
				assert.Equal(t, "output-param", output.Outputs.Parameters[0].Name)
				assert.Equal(t, "output-value", output.Outputs.Parameters[0].Value)
				require.Len(t, output.Outputs.Artifacts, 1)
				assert.Equal(t, "output-file", output.Outputs.Artifacts[0].Name)
				assert.Equal(t, "0", output.Outputs.ExitCode)
				assert.Equal(t, "success", output.Outputs.Result)

				// Verify text result contains inputs/outputs
				require.NotNil(t, result)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "input-param")
				assert.Contains(t, text.Text, "output-param")
			},
		},
		{
			name: "success - node with children",
			input: GetWorkflowNodeInput{
				Namespace:    "default",
				WorkflowName: "test-workflow",
				NodeName:     "dag-node",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("GetWorkflow", mock.Anything, mock.Anything).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-workflow",
							Namespace: "default",
							UID:       types.UID("test-uid"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowRunning,
							Nodes: wfv1.Nodes{
								"dag-node": wfv1.NodeStatus{
									ID:          "dag-node",
									Name:        "test-workflow",
									DisplayName: "test-workflow",
									Type:        wfv1.NodeTypeDAG,
									Phase:       wfv1.NodeRunning,
									Children:    []string{"child-1", "child-2", "child-3"},
								},
								"child-1": wfv1.NodeStatus{ID: "child-1", Phase: wfv1.NodeSucceeded},
								"child-2": wfv1.NodeStatus{ID: "child-2", Phase: wfv1.NodeRunning},
								"child-3": wfv1.NodeStatus{ID: "child-3", Phase: wfv1.NodePending},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetWorkflowNodeOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "DAG", output.Type)
				assert.Equal(t, "Running", output.Phase)
				require.Len(t, output.Children, 3)
				assert.Contains(t, output.Children, "child-1")
				assert.Contains(t, output.Children, "child-2")
				assert.Contains(t, output.Children, "child-3")
			},
		},
		{
			name: "success - uses default namespace",
			input: GetWorkflowNodeInput{
				WorkflowName: "test-workflow",
				NodeName:     "node-1",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("GetWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowGetRequest) bool {
					return req.Namespace == "argo" // default namespace from mock
				})).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-workflow",
							Namespace: "argo",
							UID:       types.UID("test-uid"),
						},
						Status: wfv1.WorkflowStatus{
							Nodes: wfv1.Nodes{
								"node-1": wfv1.NodeStatus{
									ID:    "node-1",
									Name:  "test-workflow[0].step",
									Type:  wfv1.NodeTypePod,
									Phase: wfv1.NodeSucceeded,
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetWorkflowNodeOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "node-1", output.ID)
			},
		},
		{
			name: "success - failed node with message",
			input: GetWorkflowNodeInput{
				Namespace:    "default",
				WorkflowName: "test-workflow",
				NodeName:     "failed-node",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("GetWorkflow", mock.Anything, mock.Anything).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-workflow",
							Namespace: "default",
							UID:       types.UID("test-uid"),
						},
						Status: wfv1.WorkflowStatus{
							Phase: wfv1.WorkflowFailed,
							Nodes: wfv1.Nodes{
								"failed-node": wfv1.NodeStatus{
									ID:           "failed-node",
									Name:         "test-workflow[0].failing-step",
									DisplayName:  "failing-step",
									Type:         wfv1.NodeTypePod,
									Phase:        wfv1.NodeFailed,
									Message:      "Error: container exit code 1",
									StartedAt:    metav1.Time{Time: startTime},
									FinishedAt:   metav1.Time{Time: endTime},
									HostNodeName: "node-2",
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetWorkflowNodeOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "Failed", output.Phase)
				assert.Equal(t, "Error: container exit code 1", output.Message)
				assert.Equal(t, "node-2", output.HostNodeName)
				require.NotNil(t, result)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "Failed")
				assert.Contains(t, text.Text, "Error: container exit code 1")
			},
		},
		{
			name: "error - empty workflow name",
			input: GetWorkflowNodeInput{
				Namespace:    "default",
				WorkflowName: "",
				NodeName:     "node-1",
			},
			setupMock: func(_ *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail validation
			},
			wantErr: true,
		},
		{
			name: "error - whitespace workflow name",
			input: GetWorkflowNodeInput{
				Namespace:    "default",
				WorkflowName: "   ",
				NodeName:     "node-1",
			},
			setupMock: func(_ *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail validation
			},
			wantErr: true,
		},
		{
			name: "error - empty node name",
			input: GetWorkflowNodeInput{
				Namespace:    "default",
				WorkflowName: "test-workflow",
				NodeName:     "",
			},
			setupMock: func(_ *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail validation
			},
			wantErr: true,
		},
		{
			name: "error - whitespace node name",
			input: GetWorkflowNodeInput{
				Namespace:    "default",
				WorkflowName: "test-workflow",
				NodeName:     "   ",
			},
			setupMock: func(_ *mocks.MockWorkflowServiceClient) {
				// No mock needed - should fail validation
			},
			wantErr: true,
		},
		{
			name: "error - workflow not found",
			input: GetWorkflowNodeInput{
				Namespace:    "default",
				WorkflowName: "nonexistent-workflow",
				NodeName:     "node-1",
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
			name: "error - node not found in workflow",
			input: GetWorkflowNodeInput{
				Namespace:    "default",
				WorkflowName: "test-workflow",
				NodeName:     "nonexistent-node",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("GetWorkflow", mock.Anything, mock.Anything).Return(
					&wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-workflow",
							Namespace: "default",
							UID:       types.UID("test-uid"),
						},
						Status: wfv1.WorkflowStatus{
							Nodes: wfv1.Nodes{
								"node-123": wfv1.NodeStatus{
									ID:   "node-123",
									Name: "test-workflow[0].step",
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: true,
		},
		{
			name: "error - permission denied",
			input: GetWorkflowNodeInput{
				Namespace:    "default",
				WorkflowName: "protected-workflow",
				NodeName:     "node-1",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("GetWorkflow", mock.Anything, mock.Anything).Return(
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
			mockService := newMockWorkflowService(t)
			mockClient.SetWorkflowService(mockService)

			// Setup mock expectations
			tt.setupMock(mockService)

			// Verify mock expectations even on error paths
			defer mockService.AssertExpectations(t)

			// Create handler and call it
			handler := GetWorkflowNodeHandler(mockClient)
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

func TestFindNode(t *testing.T) {
	nodes := wfv1.Nodes{
		"node-id-123": wfv1.NodeStatus{
			ID:          "node-id-123",
			Name:        "workflow[0].step-a",
			DisplayName: "step-a",
		},
		"node-id-456": wfv1.NodeStatus{
			ID:          "node-id-456",
			Name:        "workflow[0].step-b",
			DisplayName: "step-b",
		},
	}

	tests := []struct {
		nameOrID   string
		name       string
		expectedID string
		wantErr    bool
	}{
		{
			name:       "find by ID",
			nameOrID:   "node-id-123",
			expectedID: "node-id-123",
			wantErr:    false,
		},
		{
			name:       "find by name",
			nameOrID:   "workflow[0].step-b",
			expectedID: "node-id-456",
			wantErr:    false,
		},
		{
			name:       "find by display name",
			nameOrID:   "step-a",
			expectedID: "node-id-123",
			wantErr:    false,
		},
		{
			name:     "not found",
			nameOrID: "nonexistent",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := findNode(nodes, tt.nameOrID)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
				return
			}

			require.NoError(t, err)
			require.NotNil(t, node)
			assert.Equal(t, tt.expectedID, node.ID)
		})
	}
}

func TestBuildNodeOutput(t *testing.T) {
	startTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 1, 15, 10, 5, 30, 0, time.UTC)

	tests := []struct {
		node     *wfv1.NodeStatus
		validate func(t *testing.T, output *GetWorkflowNodeOutput)
		name     string
	}{
		{
			name: "complete node",
			node: &wfv1.NodeStatus{
				ID:           "node-123",
				Name:         "workflow[0].step",
				DisplayName:  "step",
				Type:         wfv1.NodeTypePod,
				TemplateName: "my-template",
				Phase:        wfv1.NodeSucceeded,
				Message:      "completed",
				StartedAt:    metav1.Time{Time: startTime},
				FinishedAt:   metav1.Time{Time: endTime},
				BoundaryID:   "boundary-1",
				PodIP:        "10.0.0.5",
				HostNodeName: "node-1",
				Children:     []string{"child-1"},
				Progress:     "1/1",
			},
			validate: func(t *testing.T, output *GetWorkflowNodeOutput) {
				assert.Equal(t, "node-123", output.ID)
				assert.Equal(t, "workflow[0].step", output.Name)
				assert.Equal(t, "step", output.DisplayName)
				assert.Equal(t, "Pod", output.Type)
				assert.Equal(t, "my-template", output.TemplateName)
				assert.Equal(t, "Succeeded", output.Phase)
				assert.Equal(t, "completed", output.Message)
				assert.Equal(t, "2025-01-15T10:00:00Z", output.StartedAt)
				assert.Equal(t, "2025-01-15T10:05:30Z", output.FinishedAt)
				assert.Equal(t, "5m30s", output.Duration)
				assert.Equal(t, "boundary-1", output.BoundaryID)
				assert.Equal(t, "10.0.0.5", output.PodIP)
				assert.Equal(t, "node-1", output.HostNodeName)
				assert.Equal(t, "1/1", output.Progress)
				require.Len(t, output.Children, 1)
			},
		},
		{
			name: "pending node with no timestamps",
			node: &wfv1.NodeStatus{
				ID:          "node-pending",
				Name:        "workflow[0].pending-step",
				DisplayName: "pending-step",
				Type:        wfv1.NodeTypePod,
				Phase:       "",
			},
			validate: func(t *testing.T, output *GetWorkflowNodeOutput) {
				assert.Equal(t, "Pending", output.Phase)
				assert.Empty(t, output.StartedAt)
				assert.Empty(t, output.FinishedAt)
				assert.Empty(t, output.Duration)
			},
		},
		{
			name: "running node",
			node: &wfv1.NodeStatus{
				ID:        "node-running",
				Name:      "workflow[0].running-step",
				Type:      wfv1.NodeTypePod,
				Phase:     wfv1.NodeRunning,
				StartedAt: metav1.Time{Time: startTime},
			},
			validate: func(t *testing.T, output *GetWorkflowNodeOutput) {
				assert.Equal(t, "Running", output.Phase)
				assert.NotEmpty(t, output.StartedAt)
				assert.Empty(t, output.FinishedAt)
				assert.NotEmpty(t, output.Duration) // Calculated from now
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := buildNodeOutput(tt.node)
			require.NotNil(t, output)
			tt.validate(t, output)
		})
	}
}

func TestBuildNodeInputsOutput(t *testing.T) {
	tests := []struct {
		inputs   *wfv1.Inputs
		expected *NodeInputsOutput
		name     string
	}{
		{
			name:     "nil inputs",
			inputs:   nil,
			expected: nil,
		},
		{
			name:     "empty inputs",
			inputs:   &wfv1.Inputs{},
			expected: nil,
		},
		{
			name: "with parameters only",
			inputs: &wfv1.Inputs{
				Parameters: []wfv1.Parameter{
					{Name: "param1", Value: wfv1.AnyStringPtr("value1")},
				},
			},
			expected: &NodeInputsOutput{
				Parameters: []ParameterInfo{
					{Name: "param1", Value: "value1"},
				},
			},
		},
		{
			name: "with artifacts only",
			inputs: &wfv1.Inputs{
				Artifacts: []wfv1.Artifact{
					{Name: "artifact1", Path: "/tmp/file"},
				},
			},
			expected: &NodeInputsOutput{
				Artifacts: []ArtifactInfo{
					{Name: "artifact1", Path: "/tmp/file"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildNodeInputsOutput(tt.inputs)
			if tt.expected == nil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Len(t, result.Parameters, len(tt.expected.Parameters))
			assert.Len(t, result.Artifacts, len(tt.expected.Artifacts))
		})
	}
}

func TestBuildNodeOutputsOutput(t *testing.T) {
	exitCode := "0"
	resultVal := "success"

	tests := []struct {
		outputs  *wfv1.Outputs
		expected *NodeOutputsOutput
		name     string
	}{
		{
			name:     "nil outputs",
			outputs:  nil,
			expected: nil,
		},
		{
			name:     "empty outputs",
			outputs:  &wfv1.Outputs{},
			expected: nil,
		},
		{
			name: "with exit code and result",
			outputs: &wfv1.Outputs{
				ExitCode: &exitCode,
				Result:   &resultVal,
			},
			expected: &NodeOutputsOutput{
				ExitCode: "0",
				Result:   "success",
			},
		},
		{
			name: "with parameters and artifacts",
			outputs: &wfv1.Outputs{
				Parameters: []wfv1.Parameter{
					{Name: "out-param", Value: wfv1.AnyStringPtr("out-value")},
				},
				Artifacts: []wfv1.Artifact{
					{Name: "out-artifact", Path: "/tmp/out"},
				},
			},
			expected: &NodeOutputsOutput{
				Parameters: []ParameterInfo{
					{Name: "out-param", Value: "out-value"},
				},
				Artifacts: []ArtifactInfo{
					{Name: "out-artifact", Path: "/tmp/out"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildNodeOutputsOutput(tt.outputs)
			if tt.expected == nil {
				assert.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			assert.Equal(t, tt.expected.ExitCode, result.ExitCode)
			assert.Equal(t, tt.expected.Result, result.Result)
			assert.Len(t, result.Parameters, len(tt.expected.Parameters))
			assert.Len(t, result.Artifacts, len(tt.expected.Artifacts))
			if len(tt.expected.Parameters) > 0 {
				assert.Equal(t, tt.expected.Parameters[0].Name, result.Parameters[0].Name)
				assert.Equal(t, tt.expected.Parameters[0].Value, result.Parameters[0].Value)
			}
			if len(tt.expected.Artifacts) > 0 {
				assert.Equal(t, tt.expected.Artifacts[0].Name, result.Artifacts[0].Name)
				assert.Equal(t, tt.expected.Artifacts[0].Path, result.Artifacts[0].Path)
			}
		})
	}
}

func TestBuildNodeResultText(t *testing.T) {
	output := &GetWorkflowNodeOutput{
		ID:           "node-123",
		Name:         "workflow[0].step",
		DisplayName:  "step",
		Type:         "Pod",
		TemplateName: "my-template",
		Phase:        "Succeeded",
		Message:      "completed successfully",
		StartedAt:    "2025-01-15T10:00:00Z",
		FinishedAt:   "2025-01-15T10:01:00Z",
		Duration:     "1m0s",
		PodIP:        "10.0.0.5",
		HostNodeName: "node-1",
	}

	result := buildNodeResultText(output, "test-workflow", "default")

	assert.Contains(t, result, "workflow[0].step")
	assert.Contains(t, result, "test-workflow")
	assert.Contains(t, result, "default")
	assert.Contains(t, result, "Pod")
	assert.Contains(t, result, "Succeeded")
	assert.Contains(t, result, "my-template")
	assert.Contains(t, result, "completed successfully")
	assert.Contains(t, result, "1m0s")
	assert.Contains(t, result, "10.0.0.5")
	assert.Contains(t, result, "node-1")
}
