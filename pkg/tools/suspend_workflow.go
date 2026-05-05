// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/utils/ptr"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// SuspendWorkflowInput defines the input parameters for the suspend_workflow tool.
type SuspendWorkflowInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Name is the workflow name.
	Name string `json:"name" jsonschema:"Workflow name,required"`
}

// SuspendWorkflowOutput defines the output for the suspend_workflow tool.
type SuspendWorkflowOutput struct {
	// Name is the workflow name.
	Name string `json:"name"`

	// Namespace is the namespace of the workflow.
	Namespace string `json:"namespace"`

	// Phase is the current workflow phase.
	Phase string `json:"phase"`

	// Message provides additional status information.
	Message string `json:"message,omitempty"`
}

// SuspendWorkflowTool returns the MCP tool definition for suspend_workflow.
func SuspendWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "suspend_workflow",
		Description: "Suspend a running Argo Workflow, pausing its execution",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(false),
		},
	}
}

// SuspendWorkflowHandler returns a handler function for the suspend_workflow tool.
func SuspendWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, SuspendWorkflowInput) (*mcp.CallToolResult, *SuspendWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input SuspendWorkflowInput) (*mcp.CallToolResult, *SuspendWorkflowOutput, error) {
		// Validate and normalize name
		workflowName, err := ValidateName(input.Name)
		if err != nil {
			return nil, nil, err
		}

		// Determine namespace
		namespace := ResolveNamespace(input.Namespace, client)

		// Get the workflow service client
		wfService := client.WorkflowService()

		// Suspend the workflow
		wf, err := wfService.SuspendWorkflow(ctx, &workflow.WorkflowSuspendRequest{
			Name:      workflowName,
			Namespace: namespace,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to suspend workflow: %w", err)
		}

		// Build the output
		output := &SuspendWorkflowOutput{
			Name:      wf.Name,
			Namespace: wf.Namespace,
			Phase:     string(wf.Status.Phase),
			Message:   wf.Status.Message,
		}

		// Build human-readable result
		resultText := fmt.Sprintf("Workflow %q in namespace %q suspended. Phase: %s",
			output.Name, output.Namespace, output.Phase)

		return TextResult(resultText), output, nil
	}
}
