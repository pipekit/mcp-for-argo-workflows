// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"sigs.k8s.io/yaml"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// LintWorkflowTemplateInput defines the input parameters for the lint_workflow_template tool.
type LintWorkflowTemplateInput struct {
	// Namespace is the Kubernetes namespace for template resolution (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace for template resolution (uses default if not specified)"`

	// Manifest is the WorkflowTemplate YAML manifest to validate.
	Manifest string `json:"manifest" jsonschema:"WorkflowTemplate YAML manifest to validate,required"`
}

// LintWorkflowTemplateOutput defines the output for the lint_workflow_template tool.
type LintWorkflowTemplateOutput struct {
	Name      string   `json:"name,omitempty"`
	Namespace string   `json:"namespace"`
	Errors    []string `json:"errors,omitempty"`
	Valid     bool     `json:"valid"`
}

// LintWorkflowTemplateTool returns the MCP tool definition for lint_workflow_template.
func LintWorkflowTemplateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "lint_workflow_template",
		Description: "Validate a WorkflowTemplate YAML manifest before creation. Always run this before create_workflow_template.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// LintWorkflowTemplateHandler returns a handler function for the lint_workflow_template tool.
func LintWorkflowTemplateHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, LintWorkflowTemplateInput) (*mcp.CallToolResult, *LintWorkflowTemplateOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input LintWorkflowTemplateInput) (*mcp.CallToolResult, *LintWorkflowTemplateOutput, error) {
		// Validate manifest is provided
		if strings.TrimSpace(input.Manifest) == "" {
			return nil, nil, fmt.Errorf("manifest cannot be empty")
		}

		// Guard against oversized manifests (DoS hardening)
		const maxManifestBytes = 1 << 20 // 1 MiB
		if len(input.Manifest) > maxManifestBytes {
			return nil, nil, fmt.Errorf("manifest too large (%d bytes), max %d", len(input.Manifest), maxManifestBytes)
		}

		// Parse the YAML manifest into a WorkflowTemplate object
		var wft wfv1.WorkflowTemplate
		if err := yaml.UnmarshalStrict([]byte(input.Manifest), &wft); err != nil {
			return nil, nil, fmt.Errorf("failed to parse workflow template manifest: %w", err)
		}

		// Validate that the manifest is a WorkflowTemplate
		if wft.Kind != "" && wft.Kind != "WorkflowTemplate" {
			return nil, nil, fmt.Errorf("manifest must be a WorkflowTemplate, got %q", wft.Kind)
		}

		// Determine namespace
		namespace := strings.TrimSpace(input.Namespace)
		if namespace == "" {
			namespace = client.DefaultNamespace()
		}
		wft.Namespace = namespace

		// Get the workflow template service client
		wftService, err := client.WorkflowTemplateService()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get workflow template service: %w", err)
		}

		// Lint the workflow template
		_, err = wftService.LintWorkflowTemplate(ctx, &workflowtemplate.WorkflowTemplateLintRequest{
			Namespace: namespace,
			Template:  &wft,
		})

		// Build the output
		output := &LintWorkflowTemplateOutput{
			Name:      wft.Name,
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
			resultText = fmt.Sprintf("WorkflowTemplate %s is valid", output.Name)
		} else {
			resultText = fmt.Sprintf("WorkflowTemplate validation failed:\n%s", strings.Join(output.Errors, "\n"))
		}

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}

		return result, output, nil
	}
}
