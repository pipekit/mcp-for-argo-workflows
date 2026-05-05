// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// ValidWorkflowPhases contains the allowed workflow phases for status filtering.
//
//nolint:gochecknoglobals // Constant lookup map for valid workflow phases
var ValidWorkflowPhases = map[string]bool{
	"Pending":   true,
	"Running":   true,
	"Succeeded": true,
	"Failed":    true,
	"Error":     true,
}

// ListWorkflowsInput defines the input parameters for the list_workflows tool.
type ListWorkflowsInput struct {
	Namespace *string  `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified. use empty string for all namespaces)"`
	Labels    string   `json:"labels,omitempty" jsonschema:"Label selector (e.g. 'app=myapp,env=prod')"`
	Status    []string `json:"status,omitempty" jsonschema:"Filter by phase: Pending Running Succeeded Failed Error"`
	Limit     int64    `json:"limit,omitempty" jsonschema:"Maximum number of results"`
}

// WorkflowSummary represents a concise summary of a workflow.
type WorkflowSummary struct {
	// Name is the workflow name.
	Name string `json:"name"`

	// Namespace is the namespace where the workflow exists.
	Namespace string `json:"namespace"`

	// Phase is the current workflow phase.
	Phase string `json:"phase"`

	// CreatedAt is when the workflow was created.
	CreatedAt string `json:"createdAt"`

	// FinishedAt is when the workflow finished (if applicable).
	FinishedAt string `json:"finishedAt,omitempty"`

	// Message provides additional status information.
	Message string `json:"message,omitempty"`
}

// ListWorkflowsOutput defines the output for the list_workflows tool.
type ListWorkflowsOutput struct {
	// Workflows is the list of workflow summaries.
	Workflows []WorkflowSummary `json:"workflows"`

	// Total is the total number of workflows matching the criteria.
	Total int `json:"total"`
}

// ListWorkflowsTool returns the MCP tool definition for list_workflows.
func ListWorkflowsTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "list_workflows",
		Description: "List Argo Workflows in a namespace with optional filtering by status and labels. When connected via Argo Server, this includes both live and archived workflows.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// ListWorkflowsHandler returns a handler function for the list_workflows tool.
func ListWorkflowsHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, ListWorkflowsInput) (*mcp.CallToolResult, *ListWorkflowsOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ListWorkflowsInput) (*mcp.CallToolResult, *ListWorkflowsOutput, error) {
		// Determine namespace
		namespace := client.DefaultNamespace()
		if input.Namespace != nil {
			namespace = *input.Namespace
		}

		// Build phase filter map for efficient lookup
		phaseFilterMap := make(map[string]bool, len(input.Status))
		for _, status := range input.Status {
			if !ValidWorkflowPhases[status] {
				return nil, nil, fmt.Errorf("invalid status filter %q, must be one of: Pending, Running, Succeeded, Failed, Error", status)
			}
			phaseFilterMap[status] = true
		}

		// Build list options
		listOpts := &metav1.ListOptions{}

		// Apply label selector
		if input.Labels != "" {
			listOpts.LabelSelector = input.Labels
		}

		// Only apply server-side limit if no phase filtering is needed
		// When phase filtering is active, we need all results to filter client-side
		applyClientSideLimit := len(phaseFilterMap) > 0 && input.Limit > 0
		if input.Limit > 0 && !applyClientSideLimit {
			listOpts.Limit = input.Limit
		}

		// Get the workflow service client
		wfService := client.WorkflowService()

		// List workflows
		listResp, err := wfService.ListWorkflows(ctx, &workflow.WorkflowListRequest{
			Namespace:   namespace,
			ListOptions: listOpts,
			Fields:      "items.metadata,items.status.phase,items.status.message,items.status.finishedAt",
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to list workflows: %w", err)
		}

		// Convert to summaries and apply phase filter client-side if needed
		var summaries []WorkflowSummary
		for _, wf := range listResp.Items {
			// Apply phase filter using map lookup
			phase := string(wf.Status.Phase)
			if len(phaseFilterMap) > 0 && !phaseFilterMap[phase] {
				continue
			}

			summary := WorkflowSummary{
				Name:      wf.Name,
				Namespace: wf.Namespace,
				Phase:     phase,
				Message:   wf.Status.Message,
			}

			// Format timestamps
			if !wf.CreationTimestamp.IsZero() {
				summary.CreatedAt = wf.CreationTimestamp.Format(time.RFC3339)
			}
			if !wf.Status.FinishedAt.IsZero() {
				summary.FinishedAt = wf.Status.FinishedAt.Format(time.RFC3339)
			}

			summaries = append(summaries, summary)

			// Apply client-side limit after filtering
			if applyClientSideLimit && int64(len(summaries)) >= input.Limit {
				break
			}
		}

		// Build output
		output := &ListWorkflowsOutput{
			Workflows: summaries,
			Total:     len(summaries),
		}

		// Ensure Workflows is not nil for JSON marshaling
		if output.Workflows == nil {
			output.Workflows = []WorkflowSummary{}
		}

		// Build human-readable result
		resultText := fmt.Sprintf("Found %d workflow(s)", output.Total)
		if namespace != "" {
			resultText += fmt.Sprintf(" in namespace %q", namespace)
		} else {
			resultText += " across all namespaces"
		}

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}

		return result, output, nil
	}
}
