package tools

import (
	"errors"
	"strings"
	"testing"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestLogsWorkflowTool(t *testing.T) {
	tool := LogsWorkflowTool()

	assert.Equal(t, "logs_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
}

func TestLogsWorkflowHandler_Validation(t *testing.T) {
	// Test handler validation directly - name validation occurs before client is used,
	// so we can pass nil and test that validation returns the expected errors.
	handler := LogsWorkflowHandler(nil)

	tests := []struct {
		input       LogsWorkflowInput
		name        string
		errContains string
		wantErr     bool
	}{
		{
			name: "empty name returns error",
			input: LogsWorkflowInput{
				Name: "",
			},
			wantErr:     true,
			errContains: "workflow name cannot be empty",
		},
		{
			name: "whitespace-only name returns error",
			input: LogsWorkflowInput{
				Name: "   ",
			},
			wantErr:     true,
			errContains: "workflow name cannot be empty",
		},
		{
			name: "whitespace-padded name returns error",
			input: LogsWorkflowInput{
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

func TestLogsWorkflowOutput(t *testing.T) {
	tests := []struct {
		check  func(t *testing.T, o *LogsWorkflowOutput)
		name   string
		output LogsWorkflowOutput
	}{
		{
			name: "output with logs",
			output: LogsWorkflowOutput{
				Name:      "test-workflow",
				Namespace: "default",
				Logs: []LogEntryOutput{
					{PodName: "pod-1", Content: "log line 1"},
					{PodName: "pod-1", Content: "log line 2"},
				},
				Message: "Retrieved 2 log entries",
			},
			check: func(t *testing.T, o *LogsWorkflowOutput) {
				assert.Equal(t, "test-workflow", o.Name)
				assert.Equal(t, "default", o.Namespace)
				assert.Len(t, o.Logs, 2)
				assert.False(t, o.Truncated)
			},
		},
		{
			name: "output with truncation",
			output: LogsWorkflowOutput{
				Name:      "test-workflow",
				Namespace: "default",
				Logs: []LogEntryOutput{
					{PodName: "pod-1", Content: "partial logs..."},
				},
				Truncated: true,
				Message:   "Logs truncated after 1048576 bytes",
			},
			check: func(t *testing.T, o *LogsWorkflowOutput) {
				assert.True(t, o.Truncated)
				assert.Contains(t, o.Message, "truncated")
			},
		},
		{
			name: "output with no logs",
			output: LogsWorkflowOutput{
				Name:      "test-workflow",
				Namespace: "default",
				Logs:      []LogEntryOutput{},
				Message:   "No logs available",
			},
			check: func(t *testing.T, o *LogsWorkflowOutput) {
				assert.Empty(t, o.Logs)
				assert.Contains(t, o.Message, "No logs")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, &tt.output)
		})
	}
}

func TestLogEntryOutput(t *testing.T) {
	entry := LogEntryOutput{
		PodName: "my-pod",
		Content: "2025-01-15T10:00:00Z INFO Starting process...",
	}

	assert.Equal(t, "my-pod", entry.PodName)
	assert.Contains(t, entry.Content, "Starting process")
}

func TestLogsWorkflowHandler(t *testing.T) {
	tests := []struct {
		setupMock func(*mocks.MockWorkflowServiceClient)
		validate  func(*testing.T, *LogsWorkflowOutput)
		input     LogsWorkflowInput
		name      string
		wantErr   bool
	}{
		{
			name: "success - get logs with multiple entries",
			input: LogsWorkflowInput{
				Name:      "test-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				stream := mocks.NewMockWorkflowLogsStream([]*workflow.LogEntry{
					mocks.NewLogEntry("pod-1", "Starting job..."),
					mocks.NewLogEntry("pod-1", "Processing data..."),
					mocks.NewLogEntry("pod-1", "Job completed successfully"),
				})
				m.On("WorkflowLogs", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowLogRequest) bool {
					return req.Name == "test-workflow" && req.Namespace == "default"
				})).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *LogsWorkflowOutput) {
				assert.Equal(t, "test-workflow", output.Name)
				assert.Equal(t, "default", output.Namespace)
				require.Len(t, output.Logs, 3)
				assert.Equal(t, "pod-1", output.Logs[0].PodName)
				assert.Equal(t, "Starting job...", output.Logs[0].Content)
				assert.Equal(t, "Job completed successfully", output.Logs[2].Content)
				assert.False(t, output.Truncated)
				assert.Contains(t, output.Message, "Retrieved 3 log entries")
			},
		},
		{
			name: "success - get logs from specific pod",
			input: LogsWorkflowInput{
				Name:      "test-workflow",
				Namespace: "default",
				PodName:   "specific-pod",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				stream := mocks.NewMockWorkflowLogsStream([]*workflow.LogEntry{
					mocks.NewLogEntry("specific-pod", "Log from specific pod"),
				})
				m.On("WorkflowLogs", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowLogRequest) bool {
					return req.Name == "test-workflow" && req.PodName == "specific-pod"
				})).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *LogsWorkflowOutput) {
				require.Len(t, output.Logs, 1)
				assert.Equal(t, "specific-pod", output.Logs[0].PodName)
			},
		},
		{
			name: "success - get logs with custom container",
			input: LogsWorkflowInput{
				Name:      "test-workflow",
				Namespace: "default",
				Container: "sidecar",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				stream := mocks.NewMockWorkflowLogsStream([]*workflow.LogEntry{
					mocks.NewLogEntry("pod-1", "Sidecar log"),
				})
				m.On("WorkflowLogs", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowLogRequest) bool {
					return req.LogOptions != nil && req.LogOptions.Container == "sidecar"
				})).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *LogsWorkflowOutput) {
				require.Len(t, output.Logs, 1)
			},
		},
		{
			name: "success - get logs with grep filter",
			input: LogsWorkflowInput{
				Name:      "test-workflow",
				Namespace: "default",
				Grep:      "ERROR",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				stream := mocks.NewMockWorkflowLogsStream([]*workflow.LogEntry{
					mocks.NewLogEntry("pod-1", "ERROR: Something went wrong"),
				})
				m.On("WorkflowLogs", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowLogRequest) bool {
					return req.Grep == "ERROR"
				})).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *LogsWorkflowOutput) {
				require.Len(t, output.Logs, 1)
				assert.Contains(t, output.Logs[0].Content, "ERROR")
			},
		},
		{
			name: "success - get logs with custom tail lines",
			input: LogsWorkflowInput{
				Name:      "test-workflow",
				Namespace: "default",
				TailLines: func() *int64 { v := int64(50); return &v }(),
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				stream := mocks.NewMockWorkflowLogsStream([]*workflow.LogEntry{
					mocks.NewLogEntry("pod-1", "Log line"),
				})
				m.On("WorkflowLogs", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowLogRequest) bool {
					return req.LogOptions != nil && req.LogOptions.TailLines != nil && *req.LogOptions.TailLines == 50
				})).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *LogsWorkflowOutput) {
				require.Len(t, output.Logs, 1)
			},
		},
		{
			name: "success - default namespace used",
			input: LogsWorkflowInput{
				Name: "test-workflow",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				stream := mocks.NewMockWorkflowLogsStream([]*workflow.LogEntry{
					mocks.NewLogEntry("pod-1", "Log from default namespace"),
				})
				m.On("WorkflowLogs", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowLogRequest) bool {
					return req.Namespace == "argo" // Default namespace from mock client
				})).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *LogsWorkflowOutput) {
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "success - empty logs",
			input: LogsWorkflowInput{
				Name:      "test-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				stream := mocks.NewMockWorkflowLogsStream([]*workflow.LogEntry{})
				m.On("WorkflowLogs", mock.Anything, mock.Anything).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *LogsWorkflowOutput) {
				assert.Empty(t, output.Logs)
				assert.Contains(t, output.Message, "No logs available")
			},
		},
		{
			name: "success - logs truncated due to size limit",
			input: LogsWorkflowInput{
				Name:      "test-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				// Create a large log entry that will trigger truncation
				largeContent := strings.Repeat("X", maxLogBytes+100)
				stream := mocks.NewMockWorkflowLogsStream([]*workflow.LogEntry{
					mocks.NewLogEntry("pod-1", largeContent),
				})
				m.On("WorkflowLogs", mock.Anything, mock.Anything).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *LogsWorkflowOutput) {
				// Should be truncated since the entry exceeds maxLogBytes
				assert.True(t, output.Truncated)
				assert.Contains(t, output.Message, "truncated")
			},
		},
		{
			name: "success - multiple pods",
			input: LogsWorkflowInput{
				Name:      "multi-step-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				stream := mocks.NewMockWorkflowLogsStream([]*workflow.LogEntry{
					mocks.NewLogEntry("step-1-pod", "Starting step 1"),
					mocks.NewLogEntry("step-1-pod", "Step 1 completed"),
					mocks.NewLogEntry("step-2-pod", "Starting step 2"),
					mocks.NewLogEntry("step-2-pod", "Step 2 completed"),
				})
				m.On("WorkflowLogs", mock.Anything, mock.Anything).Return(stream, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, output *LogsWorkflowOutput) {
				require.Len(t, output.Logs, 4)
				assert.Equal(t, "step-1-pod", output.Logs[0].PodName)
				assert.Equal(t, "step-2-pod", output.Logs[2].PodName)
			},
		},
		{
			name: "error - empty name",
			input: LogsWorkflowInput{
				Name:      "",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				// No mock needed - fails validation
			},
			wantErr: true,
		},
		{
			name: "error - workflow not found",
			input: LogsWorkflowInput{
				Name:      "missing-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("WorkflowLogs", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.NotFound, "workflow not found"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - permission denied",
			input: LogsWorkflowInput{
				Name:      "protected-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				m.On("WorkflowLogs", mock.Anything, mock.Anything).Return(
					nil,
					status.Error(codes.PermissionDenied, "permission denied"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - stream receive error",
			input: LogsWorkflowInput{
				Name:      "test-workflow",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockWorkflowServiceClient) {
				stream := mocks.NewMockWorkflowLogsStreamWithError(errors.New("stream error"))
				m.On("WorkflowLogs", mock.Anything, mock.Anything).Return(stream, nil)
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
			handler := LogsWorkflowHandler(mockClient)
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
