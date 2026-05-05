// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"sigs.k8s.io/yaml"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// LintClusterWorkflowTemplateInput defines the input parameters for the lint_cluster_workflow_template tool.
type LintClusterWorkflowTemplateInput struct {
	// Manifest is the ClusterWorkflowTemplate YAML manifest to validate.
	Manifest string `json:"manifest" jsonschema:"ClusterWorkflowTemplate YAML manifest to validate,required"`
}

// LintClusterWorkflowTemplateOutput defines the output for the lint_cluster_workflow_template tool.
type LintClusterWorkflowTemplateOutput struct {
	Name   string   `json:"name,omitempty"`
	Errors []string `json:"errors,omitempty"`
	Valid  bool     `json:"valid"`
}

// LintClusterWorkflowTemplateTool returns the MCP tool definition for lint_cluster_workflow_template.
func LintClusterWorkflowTemplateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "lint_cluster_workflow_template",
		Description: "Validate a ClusterWorkflowTemplate YAML manifest before creation. Always run this before create_cluster_workflow_template.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// LintClusterWorkflowTemplateHandler returns a handler function for the lint_cluster_workflow_template tool.
func LintClusterWorkflowTemplateHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, LintClusterWorkflowTemplateInput) (*mcp.CallToolResult, *LintClusterWorkflowTemplateOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input LintClusterWorkflowTemplateInput) (*mcp.CallToolResult, *LintClusterWorkflowTemplateOutput, error) {
		// Validate manifest is provided
		if strings.TrimSpace(input.Manifest) == "" {
			return nil, nil, fmt.Errorf("manifest cannot be empty")
		}

		// Guard against oversized manifests (DoS hardening)
		const maxManifestBytes = 1 << 20 // 1 MiB
		if len(input.Manifest) > maxManifestBytes {
			return nil, nil, fmt.Errorf("manifest too large (%d bytes), max %d", len(input.Manifest), maxManifestBytes)
		}

		// Parse the YAML manifest into a ClusterWorkflowTemplate object
		var cwft wfv1.ClusterWorkflowTemplate
		if err := yaml.UnmarshalStrict([]byte(input.Manifest), &cwft); err != nil {
			return nil, nil, fmt.Errorf("failed to parse cluster workflow template manifest: %w", err)
		}

		// Validate that the manifest is a ClusterWorkflowTemplate
		if cwft.Kind != "" && cwft.Kind != "ClusterWorkflowTemplate" {
			return nil, nil, fmt.Errorf("manifest must be a ClusterWorkflowTemplate, got %q", cwft.Kind)
		}

		// Get the cluster workflow template service client
		cwftService, err := client.ClusterWorkflowTemplateService()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get cluster workflow template service: %w", err)
		}

		// Lint the cluster workflow template
		_, err = cwftService.LintClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateLintRequest{
			Template: &cwft,
		})

		// Build the output
		output := &LintClusterWorkflowTemplateOutput{
			Name: cwft.Name,
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
			resultText = fmt.Sprintf("ClusterWorkflowTemplate %s is valid", output.Name)
		} else {
			resultText = fmt.Sprintf("ClusterWorkflowTemplate validation failed:\n%s", strings.Join(output.Errors, "\n"))
		}

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}

		return result, output, nil
	}
}
