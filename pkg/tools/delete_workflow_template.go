// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/utils/ptr"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// DeleteWorkflowTemplateInput defines the input parameters for the delete_workflow_template tool.
type DeleteWorkflowTemplateInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Name is the WorkflowTemplate name.
	Name string `json:"name" jsonschema:"WorkflowTemplate name,required"`
}

// DeleteWorkflowTemplateOutput defines the output for the delete_workflow_template tool.
type DeleteWorkflowTemplateOutput struct {
	// Name is the deleted workflow template name.
	Name string `json:"name"`

	// Namespace is the namespace where the workflow template was deleted.
	Namespace string `json:"namespace"`

	// Message provides confirmation of the deletion.
	Message string `json:"message"`
}

// DeleteWorkflowTemplateTool returns the MCP tool definition for delete_workflow_template.
func DeleteWorkflowTemplateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "delete_workflow_template",
		Description: "Delete a WorkflowTemplate",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(true),
		},
	}
}

// DeleteWorkflowTemplateHandler returns a handler function for the delete_workflow_template tool.
func DeleteWorkflowTemplateHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, DeleteWorkflowTemplateInput) (*mcp.CallToolResult, *DeleteWorkflowTemplateOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input DeleteWorkflowTemplateInput) (*mcp.CallToolResult, *DeleteWorkflowTemplateOutput, error) {
		// Validate and normalize name
		name, err := ValidateName(input.Name)
		if err != nil {
			return nil, nil, err
		}

		// Determine namespace
		namespace := ResolveNamespace(input.Namespace, client)

		// Get the workflow template service client
		wftService, err := client.WorkflowTemplateService()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get workflow template service: %w", err)
		}

		// Delete the workflow template
		_, err = wftService.DeleteWorkflowTemplate(ctx, &workflowtemplate.WorkflowTemplateDeleteRequest{
			Namespace: namespace,
			Name:      name,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to delete workflow template: %w", err)
		}

		// Build the output
		output := &DeleteWorkflowTemplateOutput{
			Name:      name,
			Namespace: namespace,
			Message:   fmt.Sprintf("WorkflowTemplate %q deleted successfully", name),
		}

		// Build human-readable result
		resultText := fmt.Sprintf("WorkflowTemplate %q in namespace %q deleted successfully", name, namespace)

		return TextResult(resultText), output, nil
	}
}
