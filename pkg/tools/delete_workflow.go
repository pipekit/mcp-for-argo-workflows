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

// DeleteWorkflowInput defines the input parameters for the delete_workflow tool.
type DeleteWorkflowInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Name is the workflow name.
	Name string `json:"name" jsonschema:"Workflow name,required"`

	// Force indicates whether to force deletion without waiting for graceful termination.
	Force bool `json:"force,omitempty" jsonschema:"Force deletion without waiting for graceful termination"`
}

// DeleteWorkflowOutput defines the output for the delete_workflow tool.
type DeleteWorkflowOutput struct {
	// Name is the deleted workflow name.
	Name string `json:"name"`

	// Namespace is the namespace where the workflow was deleted.
	Namespace string `json:"namespace"`

	// Message provides confirmation of the deletion.
	Message string `json:"message"`
}

// DeleteWorkflowTool returns the MCP tool definition for delete_workflow.
func DeleteWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "delete_workflow",
		Description: "Delete an Argo Workflow",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(true),
		},
	}
}

// DeleteWorkflowHandler returns a handler function for the delete_workflow tool.
func DeleteWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, DeleteWorkflowInput) (*mcp.CallToolResult, *DeleteWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input DeleteWorkflowInput) (*mcp.CallToolResult, *DeleteWorkflowOutput, error) {
		// Validate and normalize name
		name, err := ValidateName(input.Name)
		if err != nil {
			return nil, nil, err
		}

		// Determine namespace
		namespace := ResolveNamespace(input.Namespace, client)

		// Get the workflow service client
		wfService := client.WorkflowService()

		// Delete the workflow
		_, err = wfService.DeleteWorkflow(ctx, &workflow.WorkflowDeleteRequest{
			Namespace: namespace,
			Name:      name,
			Force:     input.Force,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to delete workflow: %w", err)
		}

		// Build the output
		output := &DeleteWorkflowOutput{
			Name:      name,
			Namespace: namespace,
			Message:   fmt.Sprintf("Workflow %q deleted successfully", name),
		}

		// Build human-readable result
		resultText := fmt.Sprintf("Workflow %q in namespace %q deleted successfully", name, namespace)

		return TextResult(resultText), output, nil
	}
}
