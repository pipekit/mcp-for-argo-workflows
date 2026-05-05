// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// GetCronWorkflowInput defines the input parameters for the get_cron_workflow tool.
type GetCronWorkflowInput struct {
	// Name is the cron workflow name (required).
	Name string `json:"name" jsonschema:"CronWorkflow name,required"`

	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`
}

// ActiveWorkflowRef represents a reference to an active workflow spawned by this CronWorkflow.
type ActiveWorkflowRef struct {
	// Name is the name of the active workflow.
	Name string `json:"name"`

	// Namespace is the namespace of the active workflow.
	Namespace string `json:"namespace"`

	// UID is the unique identifier of the active workflow.
	UID string `json:"uid,omitempty"`
}

// GetCronWorkflowOutput defines the output for the get_cron_workflow tool.
type GetCronWorkflowOutput struct {
	Labels                     map[string]string   `json:"labels,omitempty"`
	Annotations                map[string]string   `json:"annotations,omitempty"`
	FailedJobsHistoryLimit     *int32              `json:"failedJobsHistoryLimit,omitempty"`
	SuccessfulJobsHistoryLimit *int32              `json:"successfulJobsHistoryLimit,omitempty"`
	StartingDeadlineSeconds    *int64              `json:"startingDeadlineSeconds,omitempty"`
	CreatedAt                  string              `json:"createdAt,omitempty"`
	Timezone                   string              `json:"timezone,omitempty"`
	ConcurrencyPolicy          string              `json:"concurrencyPolicy,omitempty"`
	Schedules                  []string            `json:"schedules"`
	LastScheduledTime          string              `json:"lastScheduledTime,omitempty"`
	Entrypoint                 string              `json:"entrypoint,omitempty"`
	Namespace                  string              `json:"namespace"`
	Name                       string              `json:"name"`
	ActiveWorkflows            []ActiveWorkflowRef `json:"activeWorkflows,omitempty"`
	SucceededCount             int64               `json:"succeededCount"`
	FailedCount                int64               `json:"failedCount"`
	Suspended                  bool                `json:"suspended"`
}

// GetCronWorkflowTool returns the MCP tool definition for get_cron_workflow.
func GetCronWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_cron_workflow",
		Description: "Get details of a CronWorkflow",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// GetCronWorkflowHandler returns a handler function for the get_cron_workflow tool.
func GetCronWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, GetCronWorkflowInput) (*mcp.CallToolResult, *GetCronWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input GetCronWorkflowInput) (*mcp.CallToolResult, *GetCronWorkflowOutput, error) {
		// Validate required input
		if input.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}

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

		// Get the cron workflow
		cw, err := cronService.GetCronWorkflow(ctx, &cronworkflow.GetCronWorkflowRequest{
			Name:      input.Name,
			Namespace: namespace,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get cron workflow: %w", err)
		}

		// Build output - normalize to schedules array for consistent output
		output := &GetCronWorkflowOutput{
			Name:                       cw.Name,
			Namespace:                  cw.Namespace,
			Schedules:                  getSchedules(&cw.Spec),
			Timezone:                   cw.Spec.Timezone,
			ConcurrencyPolicy:          string(cw.Spec.ConcurrencyPolicy),
			Suspended:                  cw.Spec.Suspend,
			Labels:                     cw.Labels,
			Annotations:                cw.Annotations,
			Entrypoint:                 cw.Spec.WorkflowSpec.Entrypoint,
			SuccessfulJobsHistoryLimit: cw.Spec.SuccessfulJobsHistoryLimit,
			FailedJobsHistoryLimit:     cw.Spec.FailedJobsHistoryLimit,
			StartingDeadlineSeconds:    cw.Spec.StartingDeadlineSeconds,
			SucceededCount:             cw.Status.Succeeded,
			FailedCount:                cw.Status.Failed,
		}

		// Format creation timestamp
		if !cw.CreationTimestamp.IsZero() {
			output.CreatedAt = cw.CreationTimestamp.Format(time.RFC3339)
		}

		// Format last scheduled time
		if cw.Status.LastScheduledTime != nil && !cw.Status.LastScheduledTime.IsZero() {
			output.LastScheduledTime = cw.Status.LastScheduledTime.Format(time.RFC3339)
		}

		// Extract active workflows
		if len(cw.Status.Active) > 0 {
			output.ActiveWorkflows = make([]ActiveWorkflowRef, 0, len(cw.Status.Active))
			for _, ref := range cw.Status.Active {
				output.ActiveWorkflows = append(output.ActiveWorkflows, ActiveWorkflowRef{
					Name:      ref.Name,
					Namespace: ref.Namespace,
					UID:       string(ref.UID),
				})
			}
		}

		// Build human-readable result
		resultText := fmt.Sprintf("CronWorkflow %q in namespace %q", output.Name, output.Namespace)
		resultText += fmt.Sprintf("\nSchedule(s): %s", strings.Join(output.Schedules, ", "))
		if output.Timezone != "" {
			resultText += fmt.Sprintf(" (%s)", output.Timezone)
		}
		if output.Suspended {
			resultText += "\nStatus: Suspended"
		} else {
			resultText += "\nStatus: Active"
		}
		if output.LastScheduledTime != "" {
			resultText += fmt.Sprintf("\nLast scheduled: %s", output.LastScheduledTime)
		}
		if len(output.ActiveWorkflows) > 0 {
			resultText += fmt.Sprintf("\nActive workflows: %d", len(output.ActiveWorkflows))
		}
		resultText += fmt.Sprintf("\nHistory: %d succeeded, %d failed", output.SucceededCount, output.FailedCount)

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}

		return result, output, nil
	}
}
