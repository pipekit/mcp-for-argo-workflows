// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"sigs.k8s.io/yaml"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// LintWorkflowInput defines the input parameters for the lint_workflow tool.
type LintWorkflowInput struct {
	// Namespace is the Kubernetes namespace for template resolution (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace for template resolution (uses default if not specified)"`

	// Manifest is the workflow YAML manifest to validate.
	Manifest string `json:"manifest" jsonschema:"Workflow YAML manifest to validate,required"`
}

// LintWorkflowOutput defines the output for the lint_workflow tool.
type LintWorkflowOutput struct {
	Name         string   `json:"name,omitempty"`
	GenerateName string   `json:"generateName,omitempty"`
	Namespace    string   `json:"namespace"`
	Errors       []string `json:"errors,omitempty"`
	Valid        bool     `json:"valid"`
}

// LintWorkflowTool returns the MCP tool definition for lint_workflow.
func LintWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "lint_workflow",
		Description: "Validate an Argo Workflow YAML manifest before submission. Always run this before submit_workflow.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// LintWorkflowHandler returns a handler function for the lint_workflow tool.
func LintWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, LintWorkflowInput) (*mcp.CallToolResult, *LintWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input LintWorkflowInput) (*mcp.CallToolResult, *LintWorkflowOutput, error) {
		// Validate manifest is provided
		if strings.TrimSpace(input.Manifest) == "" {
			return nil, nil, fmt.Errorf("manifest cannot be empty")
		}

		// Guard against oversized manifests (DoS hardening)
		const maxManifestBytes = 1 << 20 // 1 MiB
		if len(input.Manifest) > maxManifestBytes {
			return nil, nil, fmt.Errorf("manifest too large (%d bytes), max %d", len(input.Manifest), maxManifestBytes)
		}

		// Parse the YAML manifest into a Workflow object
		var wf wfv1.Workflow
		if err := yaml.UnmarshalStrict([]byte(input.Manifest), &wf); err != nil {
			return nil, nil, fmt.Errorf("failed to parse workflow manifest: %w", err)
		}

		// Validate that the manifest is a Workflow
		if wf.Kind != "" && wf.Kind != "Workflow" {
			return nil, nil, fmt.Errorf("manifest must be a Workflow, got %q", wf.Kind)
		}

		// Determine namespace
		namespace := strings.TrimSpace(input.Namespace)
		if namespace == "" {
			namespace = client.DefaultNamespace()
		}
		wf.Namespace = namespace

		// Get the workflow service client
		wfService := client.WorkflowService()

		// Lint the workflow
		_, err := wfService.LintWorkflow(ctx, &workflow.WorkflowLintRequest{
			Namespace: namespace,
			Workflow:  &wf,
		})

		// Build the output
		output := &LintWorkflowOutput{
			Name:         wf.Name,
			GenerateName: wf.GenerateName,
			Namespace:    namespace,
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
			nameInfo := output.Name
			if nameInfo == "" && output.GenerateName != "" {
				nameInfo = fmt.Sprintf("(generateName: %s)", output.GenerateName)
			}
			resultText = fmt.Sprintf("Workflow %s is valid", nameInfo)
		} else {
			resultText = fmt.Sprintf("Workflow validation failed:\n%s", strings.Join(output.Errors, "\n"))
		}

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}

		return result, output, nil
	}
}
