// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	corev1 "k8s.io/api/core/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

const (
	// defaultTailLines is the default number of lines to return from the end of logs.
	defaultTailLines = 100

	// maxLogBytes is the maximum size of logs to return (1 MiB).
	maxLogBytes = 1 << 20
)

// LogsWorkflowInput defines the input parameters for the logs_workflow tool.
type LogsWorkflowInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Name is the workflow name.
	Name string `json:"name" jsonschema:"Workflow name,required"`

	// PodName is the specific pod name (omit for all pods).
	PodName string `json:"podName,omitempty" jsonschema:"Specific pod name (omit for all pods)"`

	// Container is the container name (default: main).
	Container string `json:"container,omitempty" jsonschema:"Container name (default: main)"`

	// TailLines is the number of lines from the end (default: 100).
	TailLines *int64 `json:"tailLines,omitempty" jsonschema:"Number of lines from the end (default: 100)"`

	// Grep filters log lines containing this string.
	Grep string `json:"grep,omitempty" jsonschema:"Filter log lines containing this string"`
}

// LogsWorkflowOutput defines the output for the logs_workflow tool.
type LogsWorkflowOutput struct {
	Name      string           `json:"name"`
	Namespace string           `json:"namespace"`
	Message   string           `json:"message,omitempty"`
	Logs      []LogEntryOutput `json:"logs"`
	Truncated bool             `json:"truncated,omitempty"`
}

// LogEntryOutput represents a single log entry.
type LogEntryOutput struct {
	// PodName is the name of the pod that produced this log entry.
	PodName string `json:"podName,omitempty"`

	// Content is the log content.
	Content string `json:"content"`
}

// LogsWorkflowTool returns the MCP tool definition for logs_workflow.
func LogsWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "logs_workflow",
		Description: "Retrieve logs from an Argo Workflow's pods",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// LogsWorkflowHandler returns a handler function for the logs_workflow tool.
func LogsWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, LogsWorkflowInput) (*mcp.CallToolResult, *LogsWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input LogsWorkflowInput) (*mcp.CallToolResult, *LogsWorkflowOutput, error) { //nolint:gocognit // Handler logic is sequential and readable
		// Validate and normalize name
		workflowName, err := ValidateName(input.Name)
		if err != nil {
			return nil, nil, err
		}

		// Determine namespace
		namespace := ResolveNamespace(input.Namespace, client)

		// Create a cancelable context for the stream to ensure proper cleanup
		streamCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		// Set default tail lines
		tailLines := int64(defaultTailLines)
		if input.TailLines != nil && *input.TailLines > 0 {
			tailLines = *input.TailLines
		}

		// Build log options with default container "main"
		// Argo Workflows pods use emissary executor which creates multiple containers:
		// init (setup), wait (wait for completion), main (actual workload)
		container := input.Container
		if container == "" {
			container = "main"
		}
		logOptions := &corev1.PodLogOptions{
			TailLines: &tailLines,
			Container: container,
		}

		// Build the request
		req := &workflow.WorkflowLogRequest{
			Namespace:  namespace,
			Name:       workflowName,
			PodName:    input.PodName,
			LogOptions: logOptions,
			Grep:       input.Grep,
		}

		// Get the workflow service client
		wfService := client.WorkflowService()

		// Get the log stream (use cancelable context for proper cleanup on truncation)
		stream, err := wfService.WorkflowLogs(streamCtx, req)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get workflow logs: %w", err)
		}

		// Collect logs from the stream
		logs := []LogEntryOutput{}
		var totalBytes int
		truncated := false

		for {
			entry, recvErr := stream.Recv()
			if errors.Is(recvErr, io.EOF) {
				break
			}
			if recvErr != nil {
				return nil, nil, fmt.Errorf("failed to receive log entry: %w", recvErr)
			}

			entryBytes := len(entry.Content)

			// Check if adding this entry would exceed the max log size
			if totalBytes+entryBytes > maxLogBytes {
				truncated = true
				break
			}

			totalBytes += entryBytes
			logs = append(logs, LogEntryOutput{
				PodName: entry.PodName,
				Content: entry.Content,
			})
		}

		// Build the output
		output := &LogsWorkflowOutput{
			Name:      workflowName,
			Namespace: namespace,
			Logs:      logs,
			Truncated: truncated,
		}

		switch {
		case truncated:
			output.Message = fmt.Sprintf("Logs truncated after %d bytes (max: %d bytes)", totalBytes, maxLogBytes)
		case len(logs) == 0:
			output.Message = "No logs available"
		default:
			output.Message = fmt.Sprintf("Retrieved %d log entries", len(logs))
		}

		return nil, output, nil
	}
}
