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

// TerminateWorkflowInput defines the input parameters for the terminate_workflow tool.
type TerminateWorkflowInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Name is the workflow name.
	Name string `json:"name" jsonschema:"Workflow name,required"`
}

// TerminateWorkflowOutput defines the output for the terminate_workflow tool.
type TerminateWorkflowOutput struct {
	// Name is the workflow name.
	Name string `json:"name"`

	// Namespace is the namespace of the workflow.
	Namespace string `json:"namespace"`

	// Phase is the current workflow phase.
	Phase string `json:"phase"`

	// Message provides additional status information.
	Message string `json:"message,omitempty"`
}

// TerminateWorkflowTool returns the MCP tool definition for terminate_workflow.
func TerminateWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "terminate_workflow",
		Description: "Immediately terminate an Argo Workflow, skipping exit handlers. Use stop_workflow for graceful termination that runs exit handlers.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(true),
		},
	}
}

// TerminateWorkflowHandler returns a handler function for the terminate_workflow tool.
func TerminateWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, TerminateWorkflowInput) (*mcp.CallToolResult, *TerminateWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input TerminateWorkflowInput) (*mcp.CallToolResult, *TerminateWorkflowOutput, error) {
		// Validate and normalize name
		workflowName, err := ValidateName(input.Name)
		if err != nil {
			return nil, nil, err
		}

		// Determine namespace
		namespace := ResolveNamespace(input.Namespace, client)

		// Get the workflow service client
		wfService := client.WorkflowService()

		// Terminate the workflow
		wf, err := wfService.TerminateWorkflow(ctx, &workflow.WorkflowTerminateRequest{
			Name:      workflowName,
			Namespace: namespace,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to terminate workflow: %w", err)
		}

		// Build the output
		output := &TerminateWorkflowOutput{
			Name:      wf.Name,
			Namespace: wf.Namespace,
			Phase:     string(wf.Status.Phase),
			Message:   wf.Status.Message,
		}

		// Build human-readable result
		resultText := fmt.Sprintf("Workflow %q in namespace %q terminated. Phase: %s",
			output.Name, output.Namespace, output.Phase)

		return TextResult(resultText), output, nil
	}
}
