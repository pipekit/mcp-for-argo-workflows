// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// GetWorkflowInput defines the input parameters for the get_workflow tool.
type GetWorkflowInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Name is the workflow name.
	Name string `json:"name" jsonschema:"Workflow name,required"`
}

// GetWorkflowOutput defines the output for the get_workflow tool.
type GetWorkflowOutput struct {
	// NodeSummary provides a summary of node statuses.
	NodeSummary *NodeSummary `json:"nodeSummary,omitempty"`
	// Name is the workflow name.
	Name string `json:"name"`
	// Namespace is the namespace where the workflow exists.
	Namespace string `json:"namespace"`
	// UID is the unique identifier of the workflow.
	UID string `json:"uid"`
	// Phase is the current workflow status phase.
	Phase string `json:"phase"`
	// Message provides additional status information.
	Message string `json:"message,omitempty"`
	// StartedAt is when the workflow started.
	StartedAt string `json:"startedAt,omitempty"`
	// FinishedAt is when the workflow finished.
	FinishedAt string `json:"finishedAt,omitempty"`
	// Duration is the workflow duration in a human-readable format.
	Duration string `json:"duration,omitempty"`
	// Progress shows completed/total nodes (e.g., "3/5").
	Progress string `json:"progress,omitempty"`
	// Parameters are the workflow input parameters.
	Parameters []ParameterInfo `json:"parameters,omitempty"`
}

// ParameterInfo represents a workflow parameter.
type ParameterInfo struct {
	// Name is the parameter name.
	Name string `json:"name"`

	// Value is the parameter value.
	Value string `json:"value,omitempty"`
}

// NodeSummary provides a summary of workflow node statuses.
type NodeSummary struct {
	// Total is the total number of nodes.
	Total int `json:"total"`

	// Succeeded is the count of succeeded nodes.
	Succeeded int `json:"succeeded"`

	// Failed is the count of failed nodes.
	Failed int `json:"failed"`

	// Running is the count of running nodes.
	Running int `json:"running"`

	// Pending is the count of pending nodes.
	Pending int `json:"pending"`

	// Skipped is the count of skipped nodes.
	Skipped int `json:"skipped"`

	// Error is the count of error nodes.
	Error int `json:"error"`

	// Omitted is the count of omitted nodes.
	Omitted int `json:"omitted"`
}

// GetWorkflowTool returns the MCP tool definition for get_workflow.
func GetWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_workflow",
		Description: "Get detailed information about an Argo Workflow. When connected via Argo Server, this can also retrieve archived workflows.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// GetWorkflowHandler returns a handler function for the get_workflow tool.
func GetWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, GetWorkflowInput) (*mcp.CallToolResult, *GetWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input GetWorkflowInput) (*mcp.CallToolResult, *GetWorkflowOutput, error) {
		// Validate name is provided
		if strings.TrimSpace(input.Name) == "" {
			return nil, nil, fmt.Errorf("workflow name cannot be empty")
		}

		// Determine namespace
		namespace := input.Namespace
		if namespace == "" {
			namespace = client.DefaultNamespace()
		}

		// Get the workflow service client
		wfService := client.WorkflowService()

		// Get the workflow
		wf, err := wfService.GetWorkflow(ctx, &workflow.WorkflowGetRequest{
			Namespace: namespace,
			Name:      input.Name,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get workflow: %w", err)
		}

		// Build the output
		output := buildGetWorkflowOutput(wf)

		return nil, output, nil
	}
}

// buildGetWorkflowOutput constructs the output from a workflow object.
func buildGetWorkflowOutput(wf *wfv1.Workflow) *GetWorkflowOutput {
	output := &GetWorkflowOutput{
		Name:      wf.Name,
		Namespace: wf.Namespace,
		UID:       string(wf.UID),
		Phase:     string(wf.Status.Phase),
		Message:   wf.Status.Message,
	}

	// Set a default phase if empty
	if output.Phase == "" {
		output.Phase = PhasePending
	}

	// Set timing information
	if !wf.Status.StartedAt.Time.IsZero() {
		output.StartedAt = wf.Status.StartedAt.Format(time.RFC3339)
	}
	if !wf.Status.FinishedAt.Time.IsZero() {
		output.FinishedAt = wf.Status.FinishedAt.Format(time.RFC3339)
	}

	// Calculate duration
	if !wf.Status.StartedAt.Time.IsZero() {
		endTime := wf.Status.FinishedAt.Time
		if endTime.IsZero() {
			endTime = time.Now()
		}
		duration := endTime.Sub(wf.Status.StartedAt.Time)
		output.Duration = formatDuration(duration)
	}

	// Set progress
	if wf.Status.Progress != "" {
		output.Progress = string(wf.Status.Progress)
	}

	// Extract parameters
	if len(wf.Spec.Arguments.Parameters) > 0 {
		output.Parameters = make([]ParameterInfo, 0, len(wf.Spec.Arguments.Parameters))
		for _, p := range wf.Spec.Arguments.Parameters {
			param := ParameterInfo{Name: p.Name}
			if p.Value != nil {
				param.Value = string(*p.Value)
			}
			output.Parameters = append(output.Parameters, param)
		}
	}

	// Build node summary
	if len(wf.Status.Nodes) > 0 {
		output.NodeSummary = buildNodeSummary(wf.Status.Nodes)
	}

	return output
}

// buildNodeSummary creates a summary of node statuses.
func buildNodeSummary(nodes wfv1.Nodes) *NodeSummary {
	summary := &NodeSummary{
		Total: len(nodes),
	}

	for _, node := range nodes {
		switch node.Phase {
		case wfv1.NodeSucceeded:
			summary.Succeeded++
		case wfv1.NodeFailed:
			summary.Failed++
		case wfv1.NodeRunning:
			summary.Running++
		case wfv1.NodePending:
			summary.Pending++
		case wfv1.NodeSkipped:
			summary.Skipped++
		case wfv1.NodeError:
			summary.Error++
		case wfv1.NodeOmitted:
			summary.Omitted++
		}
	}

	return summary
}

// formatDuration formats a duration in a human-readable format.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
}
