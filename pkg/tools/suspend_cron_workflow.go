// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/utils/ptr"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// SuspendCronWorkflowInput defines the input parameters for the suspend_cron_workflow tool.
type SuspendCronWorkflowInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Name is the CronWorkflow name.
	Name string `json:"name" jsonschema:"CronWorkflow name,required"`
}

// SuspendCronWorkflowOutput defines the output for the suspend_cron_workflow tool.
type SuspendCronWorkflowOutput struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Message   string   `json:"message"`
	Schedules []string `json:"schedules"`
	Suspended bool     `json:"suspended"`
}

// SuspendCronWorkflowTool returns the MCP tool definition for suspend_cron_workflow.
func SuspendCronWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "suspend_cron_workflow",
		Description: "Suspend a CronWorkflow, preventing future scheduled runs",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(false),
		},
	}
}

// SuspendCronWorkflowHandler returns a handler function for the suspend_cron_workflow tool.
func SuspendCronWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, SuspendCronWorkflowInput) (*mcp.CallToolResult, *SuspendCronWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input SuspendCronWorkflowInput) (*mcp.CallToolResult, *SuspendCronWorkflowOutput, error) {
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

		// Suspend the cron workflow
		suspended, err := cronService.SuspendCronWorkflow(ctx, &cronworkflow.CronWorkflowSuspendRequest{
			Namespace: namespace,
			Name:      name,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to suspend cron workflow: %w", err)
		}

		// Build the output
		output := &SuspendCronWorkflowOutput{
			Name:      suspended.Name,
			Namespace: suspended.Namespace,
			Schedules: suspended.Spec.GetSchedules(),
			Suspended: suspended.Spec.Suspend,
			Message:   fmt.Sprintf("CronWorkflow %q suspended successfully", name),
		}

		// Build human-readable result
		resultText := fmt.Sprintf("CronWorkflow %q in namespace %q suspended successfully", output.Name, output.Namespace)
		resultText += fmt.Sprintf("\nSchedule(s): %s (paused)", strings.Join(output.Schedules, ", "))

		return TextResult(resultText), output, nil
	}
}
