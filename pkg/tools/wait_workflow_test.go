package tools

import (
	"context"
	"errors"
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

func TestWaitWorkflowTool(t *testing.T) {
	tool := WaitWorkflowTool()

	assert.Equal(t, "wait_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
}

func TestWaitWorkflowHandler_NameValidation(t *testing.T) {
	// Test handler validation directly - name validation occurs before client is used,
	// so we can pass nil and test that validation returns the expected errors.
	handler := WaitWorkflowHandler(nil)

	tests := []struct {
		input       WaitWorkflowInput
		name        string
		errContains string
		wantErr     bool
	}{
		{
			name: "empty name returns error",
			input: WaitWorkflowInput{
				Name: "",
			},
			wantErr:     true,
			errContains: "workflow name cannot be empty",
		},
		{
			name: "whitespace-only name returns error",
			input: WaitWorkflowInput{
				Name: "   ",
			},
			wantErr:     true,
			errContains: "workflow name cannot be empty",
		},
		{
			name: "whitespace-padded name returns error",
			input: WaitWorkflowInput{
				Name: "  \t\n  ",
			},
			wantErr:     true,
			errContains: "workflow name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := handler(t.Context(), nil, tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			}
		})
	}
}

func TestWaitWorkflowHandler_TimeoutValidation(t *testing.T) {
	// Test handler timeout validation directly - timeout validation occurs before
	// client is used (after name validation), so we can pass nil client.
	handler := WaitWorkflowHandler(nil)

	tests := []struct {
		input       WaitWorkflowInput
		name        string
		errContains string
		wantErr     bool
	}{
		{
			name: "invalid timeout format returns error",
			input: WaitWorkflowInput{
				Name:    "my-workflow",
				Timeout: "invalid",
			},
			wantErr:     true,
			errContains: "invalid timeout format",
		},
		{
			name: "negative timeout returns error",
			input: WaitWorkflowInput{
				Name:    "my-workflow",
				Timeout: "-5m",
			},
			wantErr:     true,
			errContains: "invalid timeout: must be a positive duration",
		},
		{
			name: "zero timeout returns error",
			input: WaitWorkflowInput{
				Name:    "my-workflow",
				Timeout: "0s",
			},
			wantErr:     true,
			errContains: "invalid timeout: must be a positive duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := handler(t.Context(), nil, tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			}
		})
	}
}

func TestWaitWorkflowOutput(t *testing.T) {
	tests := []struct {
		check  func(t *testing.T, o *WaitWorkflowOutput)
		name   string
		output WaitWorkflowOutput
	}{
		{
			name: "output has expected fields",
			output: WaitWorkflowOutput{
				Name:       "test-workflow",
				Namespace:  "default",
				Phase:      "Succeeded",
				Message:    "test message",
				StartedAt:  "2024-01-01T00:00:00Z",
				FinishedAt: "2024-01-01T00:05:00Z",
				Duration:   "5m0s",
				Progress:   "3/3",
				TimedOut:   false,
			},
			check: func(t *testing.T, o *WaitWorkflowOutput) {
				assert.Equal(t, "test-workflow", o.Name)
				assert.Equal(t, "default", o.Namespace)
				assert.Equal(t, "Succeeded", o.Phase)
				assert.Equal(t, "test message", o.Message)
				assert.Equal(t, "2024-01-01T00:00:00Z", o.StartedAt)
				assert.Equal(t, "2024-01-01T00:05:00Z", o.FinishedAt)
				assert.Equal(t, "5m0s", o.Duration)
				assert.Equal(t, "3/3", o.Progress)
				assert.False(t, o.TimedOut)
			},
		},
		{
			name: "timed out output",
			output: WaitWorkflowOutput{
				Name:      "test-workflow",
				Namespace: "default",
				Phase:     "Running",
				TimedOut:  true,
			},
			check: func(t *testing.T, o *WaitWorkflowOutput) {
				assert.True(t, o.TimedOut)
				assert.Equal(t, "Running", o.Phase)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, &tt.output)
		})
	}
}

func TestWaitWorkflowHandler(t *testing.T) {
	startTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 1, 15, 10, 5, 30, 0, time.UTC)

	tests := []struct {
		setupMock func(*mocks.MockWorkflowServiceClient)
		validate  func(*testing.T, *WaitWorkflowOutput, *mcp.CallToolResult)
		input     WaitWorkflowInput
		name      string
		wantErr   bool
	}{
		{
			name: "success - wait for workflow to succeed",
			input: WaitWorkflowInput{
				Name:      "test-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				events := []*workflow.WorkflowWatchEvent{
					mocks.NewWatchEvent("ADDED", &wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{Name: "test-workflow", Namespace: "default"},
						Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowPending},
					}),
					mocks.NewWatchEvent("MODIFIED", &wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{Name: "test-workflow", Namespace: "default"},
						Status: wfv1.WorkflowStatus{
							Phase:    wfv1.WorkflowRunning,
							Progress: "1/3",
						},
					}),
					mocks.NewWatchEvent("MODIFIED", &wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{Name: "test-workflow", Namespace: "default"},
						Status: wfv1.WorkflowStatus{
							Phase:      wfv1.WorkflowSucceeded,
							Progress:   "3/3",
							StartedAt:  metav1.Time{Time: startTime},
							FinishedAt: metav1.Time{Time: endTime},
							Message:    "Workflow completed successfully",
						},
					}),
				}
				stream := mocks.NewMockWatchWorkflowsStream(events)
				m.On("WatchWorkflows", mock.Anything, mock.MatchedBy(func(req *workflow.WatchWorkflowsRequest) bool {
					return req.Namespace == "default" &&
						req.ListOptions != nil &&
						req.ListOptions.FieldSelector == "metadata.name=test-workflow"
				})).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *WaitWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "test-workflow", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Equal(t, "Succeeded", output.Phase)
				assert.Equal(t, "3/3", output.Progress)
				assert.Equal(t, "Workflow completed successfully", output.Message)
				assert.Equal(t, "2025-01-15T10:00:00Z", output.StartedAt)
				assert.Equal(t, "2025-01-15T10:05:30Z", output.FinishedAt)
				assert.Equal(t, "5m30s", output.Duration)
				assert.False(t, output.TimedOut)
				// Verify result text
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, textContent.Text, "Succeeded")
				assert.Contains(t, textContent.Text, "5m30s")
			},
		},
		{
			name: "success - wait for workflow to fail",
			input: WaitWorkflowInput{
				Name:      "failing-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				events := []*workflow.WorkflowWatchEvent{
					mocks.NewWatchEvent("ADDED", &wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{Name: "failing-workflow", Namespace: "default"},
						Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowPending},
					}),
					mocks.NewWatchEvent("MODIFIED", &wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{Name: "failing-workflow", Namespace: "default"},
						Status: wfv1.WorkflowStatus{
							Phase:     wfv1.WorkflowFailed,
							Message:   "Step failed: exit code 1",
							StartedAt: metav1.Time{Time: startTime},
						},
					}),
				}
				stream := mocks.NewMockWatchWorkflowsStream(events)
				m.On("WatchWorkflows", mock.Anything, mock.Anything).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *WaitWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "Failed", output.Phase)
				assert.Equal(t, "Step failed: exit code 1", output.Message)
				assert.False(t, output.TimedOut)
			},
		},
		{
			name: "success - wait for workflow to error",
			input: WaitWorkflowInput{
				Name:      "error-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				events := []*workflow.WorkflowWatchEvent{
					mocks.NewWatchEvent("MODIFIED", &wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{Name: "error-workflow", Namespace: "default"},
						Status: wfv1.WorkflowStatus{
							Phase:   wfv1.WorkflowError,
							Message: "Internal error occurred",
						},
					}),
				}
				stream := mocks.NewMockWatchWorkflowsStream(events)
				m.On("WatchWorkflows", mock.Anything, mock.Anything).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *WaitWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "Error", output.Phase)
			},
		},
		{
			name: "success - default namespace used",
			input: WaitWorkflowInput{
				Name: "test-workflow",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				events := []*workflow.WorkflowWatchEvent{
					mocks.NewWatchEvent("MODIFIED", &wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{Name: "test-workflow", Namespace: "argo"},
						Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowSucceeded},
					}),
				}
				stream := mocks.NewMockWatchWorkflowsStream(events)
				m.On("WatchWorkflows", mock.Anything, mock.MatchedBy(func(req *workflow.WatchWorkflowsRequest) bool {
					return req.Namespace == "argo" // Default namespace from mock
				})).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *WaitWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "success - no events received (EOF immediately)",
			input: WaitWorkflowInput{
				Name:      "test-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				stream := mocks.NewMockWatchWorkflowsStream([]*workflow.WorkflowWatchEvent{})
				m.On("WatchWorkflows", mock.Anything, mock.Anything).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *WaitWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "Unknown", output.Phase)
				assert.Contains(t, output.Message, "No workflow events received")
			},
		},
		{
			name: "success - event with nil object skipped",
			input: WaitWorkflowInput{
				Name:      "test-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				events := []*workflow.WorkflowWatchEvent{
					{Type: "MODIFIED", Object: nil}, // Nil object should be skipped
					mocks.NewWatchEvent("MODIFIED", &wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{Name: "test-workflow", Namespace: "default"},
						Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowSucceeded},
					}),
				}
				stream := mocks.NewMockWatchWorkflowsStream(events)
				m.On("WatchWorkflows", mock.Anything, mock.Anything).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *WaitWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "Succeeded", output.Phase)
			},
		},
		{
			name: "success - wait with running workflow returns on EOF",
			input: WaitWorkflowInput{
				Name:      "long-workflow",
				Namespace: "default",
				Timeout:   "100ms",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				// Create a stream that returns one event then ends with EOF
				stream := mocks.NewMockWatchWorkflowsStream([]*workflow.WorkflowWatchEvent{
					mocks.NewWatchEvent("ADDED", &wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{Name: "long-workflow", Namespace: "default"},
						Status:     wfv1.WorkflowStatus{Phase: wfv1.WorkflowRunning, Progress: "1/10"},
					}),
				})
				m.On("WatchWorkflows", mock.Anything, mock.Anything).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *WaitWorkflowOutput, result *mcp.CallToolResult) {
				// Since the stream ends with EOF after processing events, and no completed phase,
				// this will show the last known phase
				assert.Equal(t, "Running", output.Phase)
				// Verify result text is present
				require.NotNil(t, result)
			},
		},
		{
			name: "success - wait deadline exceeded returns timeout",
			input: WaitWorkflowInput{
				Name:      "timeout-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				// Create a stream that returns DeadlineExceeded error
				stream := mocks.NewMockWatchWorkflowsStreamWithError(context.DeadlineExceeded)
				m.On("WatchWorkflows", mock.Anything, mock.Anything).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *WaitWorkflowOutput, result *mcp.CallToolResult) {
				assert.True(t, output.TimedOut)
				assert.Equal(t, "Unknown", output.Phase)
				assert.Contains(t, output.Message, "Timed out")
				// Verify result text mentions timeout
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, textContent.Text, "timed out")
			},
		},
		{
			name: "success - workflow already completed immediately",
			input: WaitWorkflowInput{
				Name:      "completed-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				events := []*workflow.WorkflowWatchEvent{
					mocks.NewWatchEvent("ADDED", &wfv1.Workflow{
						ObjectMeta: metav1.ObjectMeta{Name: "completed-workflow", Namespace: "default"},
						Status: wfv1.WorkflowStatus{
							Phase:      wfv1.WorkflowSucceeded,
							Message:    "Already completed",
							StartedAt:  metav1.Time{Time: startTime},
							FinishedAt: metav1.Time{Time: endTime},
						},
					}),
				}
				stream := mocks.NewMockWatchWorkflowsStream(events)
				m.On("WatchWorkflows", mock.Anything, mock.Anything).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *WaitWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "Succeeded", output.Phase)
				assert.False(t, output.TimedOut)
			},
		},
		{
			name: "error - empty name",
			input: WaitWorkflowInput{
				Name:      "",
				Namespace: "default",
			},
			setupMock: func(_ *mocks.MockWorkflowServiceClient) {
				// No mock needed - fails validation
			},
			wantErr: true,
		},
		{
			name: "error - invalid timeout format",
			input: WaitWorkflowInput{
				Name:    "test-workflow",
				Timeout: "invalid",
			},
			setupMock: func(_ *mocks.MockWorkflowServiceClient) {
				// No mock needed - fails timeout validation
			},
			wantErr: true,
		},
		{
			name: "error - negative timeout",
			input: WaitWorkflowInput{
				Name:    "test-workflow",
				Timeout: "-5m",
			},
			setupMock: func(_ *mocks.MockWorkflowServiceClient) {
				// No mock needed - fails timeout validation
			},
			wantErr: true,
		},
		{
			name: "error - zero timeout",
			input: WaitWorkflowInput{
				Name:    "test-workflow",
				Timeout: "0s",
			},
			setupMock: func(_ *mocks.MockWorkflowServiceClient) {
				// No mock needed - fails timeout validation
			},
			wantErr: true,
		},
		{
			name: "error - workflow not found",
			input: WaitWorkflowInput{
				Name:      "missing-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("WatchWorkflows", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.NotFound, "workflow not found"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - permission denied",
			input: WaitWorkflowInput{
				Name:      "protected-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("WatchWorkflows", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.PermissionDenied, "permission denied"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - stream receive error",
			input: WaitWorkflowInput{
				Name:      "test-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				stream := mocks.NewMockWatchWorkflowsStreamWithError(errors.New("stream error"))
				m.On("WatchWorkflows", mock.Anything, mock.Anything).Return(stream, nil)
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

			// Create handler and call it
			handler := WaitWorkflowHandler(mockClient)
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
