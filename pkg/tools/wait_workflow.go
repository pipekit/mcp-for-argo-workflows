// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// WaitWorkflowInput defines the input parameters for the wait_workflow tool.
type WaitWorkflowInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Name is the workflow name.
	Name string `json:"name" jsonschema:"Workflow name,required"`

	// Timeout is the maximum time to wait (e.g., '5m', '1h'). Default: no timeout.
	Timeout string `json:"timeout,omitempty" jsonschema:"Maximum time to wait (e.g. 5m or 1h). Default: no timeout"`
}

// WaitWorkflowOutput defines the output for the wait_workflow tool.
type WaitWorkflowOutput struct {
	// Name is the workflow name.
	Name string `json:"name"`

	// Namespace is the namespace of the workflow.
	Namespace string `json:"namespace"`

	// Phase is the final workflow phase.
	Phase string `json:"phase"`

	// Message provides additional status information.
	Message string `json:"message,omitempty"`

	// StartedAt is when the workflow started.
	StartedAt string `json:"startedAt,omitempty"`

	// FinishedAt is when the workflow finished.
	FinishedAt string `json:"finishedAt,omitempty"`

	// Duration is the workflow duration in a human-readable format.
	Duration string `json:"duration,omitempty"`

	// Progress shows completed/total nodes.
	Progress string `json:"progress,omitempty"`

	// TimedOut indicates if the wait operation timed out.
	TimedOut bool `json:"timedOut,omitempty"`
}

// WaitWorkflowTool returns the MCP tool definition for wait_workflow.
func WaitWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "wait_workflow",
		Description: "Wait for an Argo Workflow to complete and return its final status",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// WaitWorkflowHandler returns a handler function for the wait_workflow tool.
func WaitWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, WaitWorkflowInput) (*mcp.CallToolResult, *WaitWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input WaitWorkflowInput) (*mcp.CallToolResult, *WaitWorkflowOutput, error) {
		// Validate and normalize name
		workflowName, err := ValidateName(input.Name)
		if err != nil {
			return nil, nil, err
		}

		// Parse and validate timeout if provided (before client access)
		var timeout time.Duration
		if input.Timeout != "" {
			timeout, err = time.ParseDuration(input.Timeout)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid timeout format: %w", err)
			}
			if timeout <= 0 {
				return nil, nil, fmt.Errorf("invalid timeout: must be a positive duration")
			}
		}

		// Determine namespace
		namespace := ResolveNamespace(input.Namespace, client)

		// Create a context with timeout or cancellation for cleanup
		var waitCtx context.Context
		var cancel context.CancelFunc
		if timeout > 0 {
			waitCtx, cancel = context.WithTimeout(ctx, timeout)
		} else {
			waitCtx, cancel = context.WithCancel(ctx)
		}
		defer cancel()

		// Build the request with field selector to watch specific workflow
		req := &workflow.WatchWorkflowsRequest{
			Namespace: namespace,
			ListOptions: &metav1.ListOptions{
				FieldSelector: fmt.Sprintf("metadata.name=%s", workflowName),
			},
		}

		// Get the workflow service client
		wfService := client.WorkflowService()

		// Start watching (we use the watch API but only care about the final state)
		stream, err := wfService.WatchWorkflows(waitCtx, req)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to wait for workflow: %w", err)
		}

		// Wait until completion, tracking only the last workflow state
		var lastWorkflow *wfv1.Workflow
		timedOut := false

		for {
			event, recvErr := stream.Recv()
			if errors.Is(recvErr, io.EOF) {
				break
			}
			if recvErr != nil {
				// Check if it was a timeout (handle both context and gRPC status)
				if errors.Is(recvErr, context.DeadlineExceeded) ||
					errors.Is(waitCtx.Err(), context.DeadlineExceeded) ||
					status.Code(recvErr) == codes.DeadlineExceeded {
					timedOut = true
					break
				}
				return nil, nil, fmt.Errorf("failed to receive workflow event: %w", recvErr)
			}

			if event.Object == nil {
				continue
			}

			lastWorkflow = event.Object

			// Check if workflow has completed
			if isWorkflowCompleted(event.Object.Status.Phase) {
				break
			}
		}

		// Build the output
		output := &WaitWorkflowOutput{
			Name:      workflowName,
			Namespace: namespace,
			TimedOut:  timedOut,
		}

		if lastWorkflow != nil {
			output.Phase = string(lastWorkflow.Status.Phase)
			output.Message = lastWorkflow.Status.Message
			output.Progress = string(lastWorkflow.Status.Progress)

			if !lastWorkflow.Status.StartedAt.Time.IsZero() {
				output.StartedAt = lastWorkflow.Status.StartedAt.Format(time.RFC3339)
			}
			if !lastWorkflow.Status.FinishedAt.Time.IsZero() {
				output.FinishedAt = lastWorkflow.Status.FinishedAt.Format(time.RFC3339)
			}

			// Calculate duration
			if !lastWorkflow.Status.StartedAt.Time.IsZero() {
				endTime := lastWorkflow.Status.FinishedAt.Time
				if endTime.IsZero() {
					endTime = time.Now()
				}
				duration := endTime.Sub(lastWorkflow.Status.StartedAt.Time)
				output.Duration = formatDuration(duration)
			}
		} else {
			output.Phase = "Unknown"
			output.Message = "No workflow events received"
		}

		if timedOut {
			var timeoutMsg string
			if input.Timeout != "" {
				timeoutMsg = fmt.Sprintf("Timed out after %s. Last phase: %s", input.Timeout, output.Phase)
			} else {
				timeoutMsg = fmt.Sprintf("Timed out. Last phase: %s", output.Phase)
			}
			// Preserve existing workflow message if present
			if output.Message != "" {
				output.Message = fmt.Sprintf("%s | %s", output.Message, timeoutMsg)
			} else {
				output.Message = timeoutMsg
			}
		}

		// Build human-readable result
		resultText := fmt.Sprintf("Workflow %q in namespace %q: %s", workflowName, namespace, output.Phase)
		if output.Duration != "" {
			resultText += fmt.Sprintf(" (duration: %s)", output.Duration)
		}
		if output.TimedOut {
			resultText += " [timed out]"
		}

		return TextResult(resultText), output, nil
	}
}
