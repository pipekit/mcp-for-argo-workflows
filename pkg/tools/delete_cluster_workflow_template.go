// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/utils/ptr"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// DeleteClusterWorkflowTemplateInput defines the input parameters for the delete_cluster_workflow_template tool.
type DeleteClusterWorkflowTemplateInput struct {
	// Name is the ClusterWorkflowTemplate name.
	Name string `json:"name" jsonschema:"ClusterWorkflowTemplate name,required"`
}

// DeleteClusterWorkflowTemplateOutput defines the output for the delete_cluster_workflow_template tool.
type DeleteClusterWorkflowTemplateOutput struct {
	// Name is the deleted cluster workflow template name.
	Name string `json:"name"`

	// Message provides confirmation of the deletion.
	Message string `json:"message"`
}

// DeleteClusterWorkflowTemplateTool returns the MCP tool definition for delete_cluster_workflow_template.
func DeleteClusterWorkflowTemplateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "delete_cluster_workflow_template",
		Description: "Delete a ClusterWorkflowTemplate",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(true),
		},
	}
}

// DeleteClusterWorkflowTemplateHandler returns a handler function for the delete_cluster_workflow_template tool.
func DeleteClusterWorkflowTemplateHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, DeleteClusterWorkflowTemplateInput) (*mcp.CallToolResult, *DeleteClusterWorkflowTemplateOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input DeleteClusterWorkflowTemplateInput) (*mcp.CallToolResult, *DeleteClusterWorkflowTemplateOutput, error) {
		// Validate and normalize name
		name, err := ValidateName(input.Name)
		if err != nil {
			return nil, nil, err
		}

		// Get the cluster workflow template service client
		cwftService, err := client.ClusterWorkflowTemplateService()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get cluster workflow template service: %w", err)
		}

		// Delete the cluster workflow template
		_, err = cwftService.DeleteClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateDeleteRequest{
			Name: name,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to delete cluster workflow template: %w", err)
		}

		// Build the output
		output := &DeleteClusterWorkflowTemplateOutput{
			Name:    name,
			Message: fmt.Sprintf("ClusterWorkflowTemplate %q deleted successfully", name),
		}

		// Build human-readable result
		resultText := fmt.Sprintf("ClusterWorkflowTemplate %q deleted successfully", name)

		return TextResult(resultText), output, nil
	}
}
