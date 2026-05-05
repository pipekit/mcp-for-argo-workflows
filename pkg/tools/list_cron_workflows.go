// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// ListCronWorkflowsInput defines the input parameters for the list_cron_workflows tool.
type ListCronWorkflowsInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Labels is the label selector to filter cron workflows.
	Labels string `json:"labels,omitempty" jsonschema:"Label selector (e.g. 'app=myapp,env=prod')"`
}

// CronWorkflowSummary represents a concise summary of a cron workflow.
type CronWorkflowSummary struct {
	Name              string   `json:"name"`
	Namespace         string   `json:"namespace"`
	LastScheduledTime string   `json:"lastScheduledTime,omitempty"`
	CreatedAt         string   `json:"createdAt"`
	Schedules         []string `json:"schedules"`
	Suspended         bool     `json:"suspended"`
}

// ListCronWorkflowsOutput defines the output for the list_cron_workflows tool.
type ListCronWorkflowsOutput struct {
	// CronWorkflows is the list of cron workflow summaries.
	CronWorkflows []CronWorkflowSummary `json:"cronWorkflows"`

	// Total is the total number of cron workflows matching the criteria.
	Total int `json:"total"`
}

// ListCronWorkflowsTool returns the MCP tool definition for list_cron_workflows.
func ListCronWorkflowsTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "list_cron_workflows",
		Description: "List CronWorkflows (scheduled workflows) in a namespace",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// ListCronWorkflowsHandler returns a handler function for the list_cron_workflows tool.
func ListCronWorkflowsHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, ListCronWorkflowsInput) (*mcp.CallToolResult, *ListCronWorkflowsOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ListCronWorkflowsInput) (*mcp.CallToolResult, *ListCronWorkflowsOutput, error) {
		// Determine namespace
		namespace := input.Namespace
		if namespace == "" {
			namespace = client.DefaultNamespace()
		}

		// Get the cron workflow service client
		cronService, err := client.CronWorkflowService()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get cron workflow service: %w", err)
		}

		// Build list options
		listOpts := &metav1.ListOptions{}
		if input.Labels != "" {
			listOpts.LabelSelector = input.Labels
		}

		// List cron workflows
		listResp, err := cronService.ListCronWorkflows(ctx, &cronworkflow.ListCronWorkflowsRequest{
			Namespace:   namespace,
			ListOptions: listOpts,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to list cron workflows: %w", err)
		}

		// Convert to summaries
		summaries := make([]CronWorkflowSummary, 0, len(listResp.Items))
		for _, cw := range listResp.Items {
			summary := CronWorkflowSummary{
				Name:      cw.Name,
				Namespace: cw.Namespace,
				Schedules: getSchedules(&cw.Spec),
				Suspended: cw.Spec.Suspend,
			}

			// Format timestamps
			if !cw.CreationTimestamp.IsZero() {
				summary.CreatedAt = cw.CreationTimestamp.Format(time.RFC3339)
			}

			// Get last scheduled time from status
			if cw.Status.LastScheduledTime != nil && !cw.Status.LastScheduledTime.IsZero() {
				summary.LastScheduledTime = cw.Status.LastScheduledTime.Format(time.RFC3339)
			}

			summaries = append(summaries, summary)
		}

		// Build output
		output := &ListCronWorkflowsOutput{
			CronWorkflows: summaries,
			Total:         len(summaries),
		}

		// Build human-readable result
		resultText := fmt.Sprintf("Found %d cron workflow(s) in namespace %q", output.Total, namespace)

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}

		return result, output, nil
	}
}
