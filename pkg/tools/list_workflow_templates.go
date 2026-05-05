// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// ListWorkflowTemplatesInput defines the input parameters for the list_workflow_templates tool.
type ListWorkflowTemplatesInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Labels is the label selector to filter templates.
	Labels string `json:"labels,omitempty" jsonschema:"Label selector (e.g. 'app=myapp,env=prod')"`
}

// WorkflowTemplateSummary represents a concise summary of a workflow template.
type WorkflowTemplateSummary struct {
	// Name is the workflow template name.
	Name string `json:"name"`

	// Namespace is the namespace where the workflow template exists.
	Namespace string `json:"namespace"`

	// CreatedAt is when the workflow template was created.
	CreatedAt string `json:"createdAt"`
}

// ListWorkflowTemplatesOutput defines the output for the list_workflow_templates tool.
type ListWorkflowTemplatesOutput struct {
	// Templates is the list of workflow template summaries.
	Templates []WorkflowTemplateSummary `json:"templates"`

	// Total is the total number of templates matching the criteria.
	Total int `json:"total"`
}

// ListWorkflowTemplatesTool returns the MCP tool definition for list_workflow_templates.
func ListWorkflowTemplatesTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "list_workflow_templates",
		Description: "List WorkflowTemplates in a namespace",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// ListWorkflowTemplatesHandler returns a handler function for the list_workflow_templates tool.
func ListWorkflowTemplatesHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, ListWorkflowTemplatesInput) (*mcp.CallToolResult, *ListWorkflowTemplatesOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ListWorkflowTemplatesInput) (*mcp.CallToolResult, *ListWorkflowTemplatesOutput, error) {
		// Determine namespace
		namespace := input.Namespace
		if namespace == "" {
			namespace = client.DefaultNamespace()
		}

		// Get the workflow template service client
		wftService, err := client.WorkflowTemplateService()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get workflow template service: %w", err)
		}

		// Build list options
		listOpts := &metav1.ListOptions{}
		if input.Labels != "" {
			listOpts.LabelSelector = input.Labels
		}

		// List workflow templates
		listResp, err := wftService.ListWorkflowTemplates(ctx, &workflowtemplate.WorkflowTemplateListRequest{
			Namespace:   namespace,
			ListOptions: listOpts,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to list workflow templates: %w", err)
		}

		// Convert to summaries
		summaries := make([]WorkflowTemplateSummary, 0, len(listResp.Items))
		for _, wft := range listResp.Items {
			summary := WorkflowTemplateSummary{
				Name:      wft.Name,
				Namespace: wft.Namespace,
			}

			// Format timestamps
			if !wft.CreationTimestamp.IsZero() {
				summary.CreatedAt = wft.CreationTimestamp.Format(time.RFC3339)
			}

			summaries = append(summaries, summary)
		}

		// Build output
		output := &ListWorkflowTemplatesOutput{
			Templates: summaries,
			Total:     len(summaries),
		}

		// Build human-readable result
		resultText := fmt.Sprintf("Found %d workflow template(s) in namespace %q", output.Total, namespace)

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}

		return result, output, nil
	}
}
