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

// ResumeCronWorkflowInput defines the input parameters for the resume_cron_workflow tool.
type ResumeCronWorkflowInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Name is the CronWorkflow name.
	Name string `json:"name" jsonschema:"CronWorkflow name,required"`
}

// ResumeCronWorkflowOutput defines the output for the resume_cron_workflow tool.
type ResumeCronWorkflowOutput struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Message   string   `json:"message"`
	Schedules []string `json:"schedules"`
	Suspended bool     `json:"suspended"`
}

// ResumeCronWorkflowTool returns the MCP tool definition for resume_cron_workflow.
func ResumeCronWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "resume_cron_workflow",
		Description: "Resume a suspended CronWorkflow",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(false),
		},
	}
}

// ResumeCronWorkflowHandler returns a handler function for the resume_cron_workflow tool.
func ResumeCronWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, ResumeCronWorkflowInput) (*mcp.CallToolResult, *ResumeCronWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ResumeCronWorkflowInput) (*mcp.CallToolResult, *ResumeCronWorkflowOutput, error) {
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

		// Resume the cron workflow
		resumed, err := cronService.ResumeCronWorkflow(ctx, &cronworkflow.CronWorkflowResumeRequest{
			Namespace: namespace,
			Name:      name,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resume cron workflow: %w", err)
		}

		// Build the output
		output := &ResumeCronWorkflowOutput{
			Name:      resumed.Name,
			Namespace: resumed.Namespace,
			Schedules: resumed.Spec.GetSchedules(),
			Suspended: resumed.Spec.Suspend,
			Message:   fmt.Sprintf("CronWorkflow %q resumed successfully", name),
		}

		// Build human-readable result
		resultText := fmt.Sprintf("CronWorkflow %q in namespace %q resumed successfully", output.Name, output.Namespace)
		resultText += fmt.Sprintf("\nSchedule(s): %s (active)", strings.Join(output.Schedules, ", "))

		return TextResult(resultText), output, nil
	}
}
