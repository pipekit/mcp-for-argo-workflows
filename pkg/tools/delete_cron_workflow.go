// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/utils/ptr"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// DeleteCronWorkflowInput defines the input parameters for the delete_cron_workflow tool.
type DeleteCronWorkflowInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Name is the CronWorkflow name.
	Name string `json:"name" jsonschema:"CronWorkflow name,required"`
}

// DeleteCronWorkflowOutput defines the output for the delete_cron_workflow tool.
type DeleteCronWorkflowOutput struct {
	// Name is the deleted CronWorkflow name.
	Name string `json:"name"`

	// Namespace is the namespace where the CronWorkflow was deleted.
	Namespace string `json:"namespace"`

	// Message provides confirmation of the deletion.
	Message string `json:"message"`
}

// DeleteCronWorkflowTool returns the MCP tool definition for delete_cron_workflow.
func DeleteCronWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "delete_cron_workflow",
		Description: "Delete a CronWorkflow",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(true),
		},
	}
}

// DeleteCronWorkflowHandler returns a handler function for the delete_cron_workflow tool.
func DeleteCronWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, DeleteCronWorkflowInput) (*mcp.CallToolResult, *DeleteCronWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input DeleteCronWorkflowInput) (*mcp.CallToolResult, *DeleteCronWorkflowOutput, error) {
		// Validate and normalize name
		name, err := ValidateName(input.Name)
		if err != nil {
			return nil, nil, err
		}

		// Determine namespace
		namespace := ResolveNamespace(input.Namespace, client)

		// Get the cron workflow service client
		cronService, err := client.CronWorkflowService()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get cron workflow service: %w", err)
		}

		// Delete the cron workflow
		_, err = cronService.DeleteCronWorkflow(ctx, &cronworkflow.DeleteCronWorkflowRequest{
			Namespace: namespace,
			Name:      name,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to delete cron workflow: %w", err)
		}

		// Build the output
		output := &DeleteCronWorkflowOutput{
			Name:      name,
			Namespace: namespace,
			Message:   fmt.Sprintf("CronWorkflow %q deleted successfully", name),
		}

		// Build human-readable result
		resultText := fmt.Sprintf("CronWorkflow %q in namespace %q deleted successfully", name, namespace)

		return TextResult(resultText), output, nil
	}
}
