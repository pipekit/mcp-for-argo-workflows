// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// ListClusterWorkflowTemplatesInput defines the input parameters for the list_cluster_workflow_templates tool.
type ListClusterWorkflowTemplatesInput struct {
	// Labels is an optional label selector to filter templates.
	Labels string `json:"labels,omitempty" jsonschema:"Label selector to filter templates"`
}

// ClusterWorkflowTemplateSummary provides a summary of a ClusterWorkflowTemplate.
type ClusterWorkflowTemplateSummary struct {
	// Name is the template name.
	Name string `json:"name"`

	// CreatedAt is when the template was created.
	CreatedAt string `json:"createdAt,omitempty"`
}

// ListClusterWorkflowTemplatesOutput defines the output for the list_cluster_workflow_templates tool.
type ListClusterWorkflowTemplatesOutput struct {
	// Templates is the list of cluster workflow template summaries.
	Templates []ClusterWorkflowTemplateSummary `json:"templates"`

	// Total is the count of templates returned.
	Total int `json:"total"`
}

// ListClusterWorkflowTemplatesTool returns the MCP tool definition for list_cluster_workflow_templates.
func ListClusterWorkflowTemplatesTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "list_cluster_workflow_templates",
		Description: "List ClusterWorkflowTemplates (cluster-scoped templates available to all namespaces)",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// ListClusterWorkflowTemplatesHandler returns a handler function for the list_cluster_workflow_templates tool.
func ListClusterWorkflowTemplatesHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, ListClusterWorkflowTemplatesInput) (*mcp.CallToolResult, *ListClusterWorkflowTemplatesOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ListClusterWorkflowTemplatesInput) (*mcp.CallToolResult, *ListClusterWorkflowTemplatesOutput, error) {
		// Build the list request
		req := &clusterworkflowtemplate.ClusterWorkflowTemplateListRequest{}

		// Add label selector if provided
		if input.Labels != "" {
			req.ListOptions = &metav1.ListOptions{
				LabelSelector: input.Labels,
			}
		}

		// Get the cluster workflow template service client
		cwftService, err := client.ClusterWorkflowTemplateService()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get cluster workflow template service: %w", err)
		}

		// List the cluster workflow templates
		cwftList, err := cwftService.ListClusterWorkflowTemplates(ctx, req)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to list cluster workflow templates: %w", err)
		}

		// Build the output
		output := &ListClusterWorkflowTemplatesOutput{
			Templates: make([]ClusterWorkflowTemplateSummary, 0, len(cwftList.Items)),
			Total:     len(cwftList.Items),
		}

		for _, cwft := range cwftList.Items {
			summary := ClusterWorkflowTemplateSummary{
				Name: cwft.Name,
			}

			// Format timestamp
			if !cwft.CreationTimestamp.IsZero() {
				summary.CreatedAt = cwft.CreationTimestamp.Format(time.RFC3339)
			}

			output.Templates = append(output.Templates, summary)
		}

		// Build human-readable result
		resultText := fmt.Sprintf("Found %d cluster workflow template(s)", output.Total)

		return TextResult(resultText), output, nil
	}
}
