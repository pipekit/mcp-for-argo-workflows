// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/utils/ptr"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// RetryWorkflowInput defines the input parameters for the retry_workflow tool.
type RetryWorkflowInput struct {
	Namespace         string   `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`
	Name              string   `json:"name" jsonschema:"Workflow name,required"`
	NodeFieldSelector string   `json:"nodeFieldSelector,omitempty" jsonschema:"Selector for nodes to restart (e.g. phase=Failed)"`
	Parameters        []string `json:"parameters,omitempty" jsonschema:"Parameter overrides in key=value format"`
	RestartSuccessful bool     `json:"restartSuccessful,omitempty" jsonschema:"Also restart successful nodes"`
}

// RetryWorkflowOutput defines the output for the retry_workflow tool.
type RetryWorkflowOutput struct {
	// Name is the workflow name.
	Name string `json:"name"`

	// Namespace is the namespace of the workflow.
	Namespace string `json:"namespace"`

	// UID is the unique identifier of the workflow.
	UID string `json:"uid"`

	// Phase is the current workflow phase after retry.
	Phase string `json:"phase"`

	// Message provides additional status information.
	Message string `json:"message,omitempty"`
}

// RetryWorkflowTool returns the MCP tool definition for retry_workflow.
func RetryWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "retry_workflow",
		Description: "Retry a failed Argo Workflow from the point of failure",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(true),
		},
	}
}

// RetryWorkflowHandler returns a handler function for the retry_workflow tool.
func RetryWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, RetryWorkflowInput) (*mcp.CallToolResult, *RetryWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input RetryWorkflowInput) (*mcp.CallToolResult, *RetryWorkflowOutput, error) {
		// Validate and normalize name
		workflowName := strings.TrimSpace(input.Name)
		if workflowName == "" {
			return nil, nil, fmt.Errorf("workflow name cannot be empty")
		}

		// Determine namespace
		namespace := strings.TrimSpace(input.Namespace)
		if namespace == "" {
			namespace = client.DefaultNamespace()
		}

		// Get the workflow service client
		wfService := client.WorkflowService()

		// Build the retry request
		req := &workflow.WorkflowRetryRequest{
			Name:              workflowName,
			Namespace:         namespace,
			RestartSuccessful: input.RestartSuccessful,
			NodeFieldSelector: input.NodeFieldSelector,
			Parameters:        input.Parameters,
		}

		// Retry the workflow
		retriedWf, err := wfService.RetryWorkflow(ctx, req)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to retry workflow: %w", err)
		}

		// Build the output
		output := &RetryWorkflowOutput{
			Name:      retriedWf.Name,
			Namespace: retriedWf.Namespace,
			UID:       string(retriedWf.UID),
			Phase:     string(retriedWf.Status.Phase),
			Message:   retriedWf.Status.Message,
		}

		// Set a default phase if empty
		if output.Phase == "" {
			output.Phase = "Running"
		}

		// Build human-readable result
		resultText := fmt.Sprintf("Workflow %q in namespace %q retried successfully. Phase: %s",
			output.Name, output.Namespace, output.Phase)

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}

		return result, output, nil
	}
}
