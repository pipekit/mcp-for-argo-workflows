// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// CreateCronWorkflowInput defines the input parameters for the create_cron_workflow tool.
type CreateCronWorkflowInput struct {
	// Manifest is the YAML manifest of the CronWorkflow to create (required).
	Manifest string `json:"manifest" jsonschema:"CronWorkflow YAML manifest,required"`

	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`
}

// CreateCronWorkflowOutput defines the output for the create_cron_workflow tool.
type CreateCronWorkflowOutput struct {
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
	CreatedAt         string            `json:"createdAt"`
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace"`
	Timezone          string            `json:"timezone,omitempty"`
	Entrypoint        string            `json:"entrypoint,omitempty"`
	ConcurrencyPolicy string            `json:"concurrencyPolicy,omitempty"`
	Schedules         []string          `json:"schedules"`
	Suspended         bool              `json:"suspended"`
	Created           bool              `json:"created"`
}

// CreateCronWorkflowTool returns the MCP tool definition for create_cron_workflow.
func CreateCronWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "create_cron_workflow",
		Description: "Create or update a CronWorkflow from a YAML manifest. If the cron workflow already exists, it will be updated.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(true),
		},
	}
}

// CreateCronWorkflowHandler returns a handler function for the create_cron_workflow tool.
func CreateCronWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, CreateCronWorkflowInput) (*mcp.CallToolResult, *CreateCronWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CreateCronWorkflowInput) (*mcp.CallToolResult, *CreateCronWorkflowOutput, error) {
		// Validate manifest is provided
		if strings.TrimSpace(input.Manifest) == "" {
			return nil, nil, fmt.Errorf("manifest cannot be empty")
		}

		// Guard against oversized manifests (DoS hardening)
		const maxManifestBytes = 1 << 20 // 1 MiB
		if len(input.Manifest) > maxManifestBytes {
			return nil, nil, fmt.Errorf("manifest exceeds maximum size of %d bytes", maxManifestBytes)
		}

		// Parse the YAML manifest
		var cronWf wfv1.CronWorkflow
		if err := yaml.UnmarshalStrict([]byte(input.Manifest), &cronWf); err != nil {
			return nil, nil, fmt.Errorf("failed to parse CronWorkflow manifest: %w", err)
		}

		// Validate that the manifest is a CronWorkflow
		if cronWf.Kind != "" && cronWf.Kind != "CronWorkflow" {
			return nil, nil, fmt.Errorf("manifest kind must be CronWorkflow, got %q", cronWf.Kind)
		}

		// Validate name
		if cronWf.Name == "" {
			return nil, nil, fmt.Errorf("cron workflow name is required in manifest")
		}

		// Validate schedule
		if len(cronWf.Spec.Schedules) == 0 {
			return nil, nil, fmt.Errorf("cron workflow schedule is required in manifest (use 'schedules' array)")
		}

		// Resolve namespace - prefer input namespace, then manifest namespace, then default
		namespace := input.Namespace
		if namespace == "" {
			namespace = cronWf.Namespace
		}
		namespace = ResolveNamespace(namespace, client)
		cronWf.Namespace = namespace

		// Get the cron workflow service client
		cronService, err := client.CronWorkflowService()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get cron workflow service: %w", err)
		}

		// Try to create the cron workflow first
		var resultCronWf *wfv1.CronWorkflow
		var wasCreated bool

		resultCronWf, err = cronService.CreateCronWorkflow(ctx, &cronworkflow.CreateCronWorkflowRequest{
			Namespace:    namespace,
			CronWorkflow: &cronWf,
		})

		if err != nil {
			// Check if error is AlreadyExists - if so, try to update instead
			if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.AlreadyExists {
				// Get the existing cron workflow to retrieve its resourceVersion
				existingCronWf, getErr := cronService.GetCronWorkflow(ctx, &cronworkflow.GetCronWorkflowRequest{
					Namespace: namespace,
					Name:      cronWf.Name,
				})
				if getErr != nil {
					return nil, nil, fmt.Errorf("failed to get existing cron workflow for update: %w", getErr)
				}

				// Copy the resourceVersion to enable update
				cronWf.ResourceVersion = existingCronWf.ResourceVersion

				// Update the existing cron workflow
				resultCronWf, err = cronService.UpdateCronWorkflow(ctx, &cronworkflow.UpdateCronWorkflowRequest{
					Namespace:    namespace,
					Name:         cronWf.Name,
					CronWorkflow: &cronWf,
				})
				if err != nil {
					return nil, nil, fmt.Errorf("failed to update cron workflow: %w", err)
				}
				wasCreated = false
			} else {
				return nil, nil, fmt.Errorf("failed to create cron workflow: %w", err)
			}
		} else {
			wasCreated = true
		}

		// Build output - normalize to schedules array for consistent output
		schedules := getSchedules(&resultCronWf.Spec)
		output := &CreateCronWorkflowOutput{
			Name:              resultCronWf.Name,
			Namespace:         resultCronWf.Namespace,
			Schedules:         schedules,
			Timezone:          resultCronWf.Spec.Timezone,
			ConcurrencyPolicy: string(resultCronWf.Spec.ConcurrencyPolicy),
			Suspended:         resultCronWf.Spec.Suspend,
			Labels:            resultCronWf.Labels,
			Annotations:       resultCronWf.Annotations,
			Created:           wasCreated,
		}

		// Format creation timestamp
		if !resultCronWf.CreationTimestamp.IsZero() {
			output.CreatedAt = resultCronWf.CreationTimestamp.Format(time.RFC3339)
		}

		// Get entrypoint if available
		if resultCronWf.Spec.WorkflowSpec.Entrypoint != "" {
			output.Entrypoint = resultCronWf.Spec.WorkflowSpec.Entrypoint
		}

		// Build human-readable result
		var resultText string
		if wasCreated {
			resultText = fmt.Sprintf("CronWorkflow %q created in namespace %q", output.Name, output.Namespace)
		} else {
			resultText = fmt.Sprintf("CronWorkflow %q updated in namespace %q", output.Name, output.Namespace)
		}
		resultText += fmt.Sprintf("\nSchedule(s): %s", strings.Join(output.Schedules, ", "))
		if output.Timezone != "" {
			resultText += fmt.Sprintf(" (%s)", output.Timezone)
		}
		if output.Suspended {
			resultText += "\nStatus: Suspended"
		} else {
			resultText += "\nStatus: Active"
		}

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}

		return result, output, nil
	}
}

// getSchedules returns the schedules from a CronWorkflowSpec, normalizing
// legacy single schedule to an array for consistent output.
func getSchedules(spec *wfv1.CronWorkflowSpec) []string {
	if spec == nil {
		return []string{}
	}
	return spec.GetSchedules()
}
