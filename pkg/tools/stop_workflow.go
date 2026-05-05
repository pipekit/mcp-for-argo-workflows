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

// StopWorkflowInput defines the input parameters for the stop_workflow tool.
type StopWorkflowInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Name is the workflow name.
	Name string `json:"name" jsonschema:"Workflow name,required"`

	// NodeFieldSelector is a selector for specific nodes to stop.
	NodeFieldSelector string `json:"nodeFieldSelector,omitempty" jsonschema:"Selector for specific nodes to stop"`

	// Message is an optional message to record on the workflow.
	Message string `json:"message,omitempty" jsonschema:"Message to record on the workflow"`
}

// StopWorkflowOutput defines the output for the stop_workflow tool.
type StopWorkflowOutput struct {
	// Name is the workflow name.
	Name string `json:"name"`

	// Namespace is the namespace of the workflow.
	Namespace string `json:"namespace"`

	// Phase is the current workflow phase.
	Phase string `json:"phase"`

	// Message provides additional status information.
	Message string `json:"message,omitempty"`
}

// StopWorkflowTool returns the MCP tool definition for stop_workflow.
func StopWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "stop_workflow",
		Description: "Gracefully stop a running Argo Workflow. Exit handlers will still run. Use terminate_workflow for immediate termination.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(true),
		},
	}
}

// StopWorkflowHandler returns a handler function for the stop_workflow tool.
func StopWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, StopWorkflowInput) (*mcp.CallToolResult, *StopWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input StopWorkflowInput) (*mcp.CallToolResult, *StopWorkflowOutput, error) {
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

		// Stop the workflow
		wf, err := wfService.StopWorkflow(ctx, &workflow.WorkflowStopRequest{
			Name:              workflowName,
			Namespace:         namespace,
			NodeFieldSelector: input.NodeFieldSelector,
			Message:           input.Message,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to stop workflow: %w", err)
		}

		// Build the output
		output := &StopWorkflowOutput{
			Name:      wf.Name,
			Namespace: wf.Namespace,
			Phase:     string(wf.Status.Phase),
			Message:   wf.Status.Message,
		}

		// Build human-readable result
		resultText := fmt.Sprintf("Workflow %q in namespace %q stopped. Phase: %s",
			output.Name, output.Namespace, output.Phase)

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}

		return result, output, nil
	}
}
