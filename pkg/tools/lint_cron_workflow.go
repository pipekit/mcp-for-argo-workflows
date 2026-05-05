// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"sigs.k8s.io/yaml"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// LintCronWorkflowInput defines the input parameters for the lint_cron_workflow tool.
type LintCronWorkflowInput struct {
	// Namespace is the Kubernetes namespace for template resolution (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace for template resolution (uses default if not specified)"`

	// Manifest is the CronWorkflow YAML manifest to validate.
	Manifest string `json:"manifest" jsonschema:"CronWorkflow YAML manifest to validate,required"`
}

// LintCronWorkflowOutput defines the output for the lint_cron_workflow tool.
type LintCronWorkflowOutput struct {
	Name      string   `json:"name,omitempty"`
	Namespace string   `json:"namespace"`
	Errors    []string `json:"errors,omitempty"`
	Valid     bool     `json:"valid"`
}

// LintCronWorkflowTool returns the MCP tool definition for lint_cron_workflow.
func LintCronWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "lint_cron_workflow",
		Description: "Validate a CronWorkflow YAML manifest before creation. Always run this before create_cron_workflow.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// LintCronWorkflowHandler returns a handler function for the lint_cron_workflow tool.
func LintCronWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, LintCronWorkflowInput) (*mcp.CallToolResult, *LintCronWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input LintCronWorkflowInput) (*mcp.CallToolResult, *LintCronWorkflowOutput, error) {
		// Validate manifest is provided
		if strings.TrimSpace(input.Manifest) == "" {
			return nil, nil, fmt.Errorf("manifest cannot be empty")
		}

		// Guard against oversized manifests (DoS hardening)
		const maxManifestBytes = 1 << 20 // 1 MiB
		if len(input.Manifest) > maxManifestBytes {
			return nil, nil, fmt.Errorf("manifest too large (%d bytes), max %d", len(input.Manifest), maxManifestBytes)
		}

		// Parse the YAML manifest into a CronWorkflow object
		var cw wfv1.CronWorkflow
		if err := yaml.UnmarshalStrict([]byte(input.Manifest), &cw); err != nil {
			return nil, nil, fmt.Errorf("failed to parse cron workflow manifest: %w", err)
		}

		// Validate that the manifest is a CronWorkflow
		if cw.Kind != "" && cw.Kind != "CronWorkflow" {
			return nil, nil, fmt.Errorf("manifest must be a CronWorkflow, got %q", cw.Kind)
		}

		// Determine namespace
		namespace := strings.TrimSpace(input.Namespace)
		if namespace == "" {
			namespace = client.DefaultNamespace()
		}
		cw.Namespace = namespace

		// Get the cron workflow service client
		cwService, err := client.CronWorkflowService()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get cron workflow service: %w", err)
		}

		// Lint the cron workflow
		_, err = cwService.LintCronWorkflow(ctx, &cronworkflow.LintCronWorkflowRequest{
			Namespace:    namespace,
			CronWorkflow: &cw,
		})

		// Build the output
		output := &LintCronWorkflowOutput{
			Name:      cw.Name,
			Namespace: namespace,
		}

		if err != nil {
			output.Valid = false
			output.Errors = []string{err.Error()}
		} else {
			output.Valid = true
		}

		// Build human-readable result
		var resultText string
		if output.Valid {
			resultText = fmt.Sprintf("CronWorkflow %s is valid", output.Name)
		} else {
			resultText = fmt.Sprintf("CronWorkflow validation failed:\n%s", strings.Join(output.Errors, "\n"))
		}

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}

		return result, output, nil
	}
}
