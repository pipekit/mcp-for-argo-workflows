// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// GetClusterWorkflowTemplateInput defines the input parameters for the get_cluster_workflow_template tool.
type GetClusterWorkflowTemplateInput struct {
	// Name is the cluster workflow template name (required).
	Name string `json:"name" jsonschema:"ClusterWorkflowTemplate name,required"`
}

// GetClusterWorkflowTemplateOutput defines the output for the get_cluster_workflow_template tool.
type GetClusterWorkflowTemplateOutput struct {
	Labels              map[string]string       `json:"labels,omitempty"`
	Annotations         map[string]string       `json:"annotations,omitempty"`
	WorkflowTemplateRef *TemplateRef            `json:"workflowTemplateRef,omitempty"`
	Name                string                  `json:"name"`
	CreatedAt           string                  `json:"createdAt,omitempty"`
	Entrypoint          string                  `json:"entrypoint,omitempty"`
	Arguments           []TemplateParameterInfo `json:"arguments,omitempty"`
	Templates           []TemplateInfo          `json:"templates,omitempty"`
}

// GetClusterWorkflowTemplateTool returns the MCP tool definition for get_cluster_workflow_template.
func GetClusterWorkflowTemplateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_cluster_workflow_template",
		Description: "Get details of a specific ClusterWorkflowTemplate",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// GetClusterWorkflowTemplateHandler returns a handler function for the get_cluster_workflow_template tool.
func GetClusterWorkflowTemplateHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, GetClusterWorkflowTemplateInput) (*mcp.CallToolResult, *GetClusterWorkflowTemplateOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input GetClusterWorkflowTemplateInput) (*mcp.CallToolResult, *GetClusterWorkflowTemplateOutput, error) {
		// Validate required input
		if input.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}

		// Get the cluster workflow template service client
		cwftService, err := client.ClusterWorkflowTemplateService()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get cluster workflow template service: %w", err)
		}

		// Get the cluster workflow template
		cwft, err := cwftService.GetClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateGetRequest{
			Name: input.Name,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get cluster workflow template: %w", err)
		}

		// Build output
		output := &GetClusterWorkflowTemplateOutput{
			Name:        cwft.Name,
			Labels:      cwft.Labels,
			Annotations: cwft.Annotations,
			Entrypoint:  cwft.Spec.Entrypoint,
		}

		// Format timestamp
		if !cwft.CreationTimestamp.IsZero() {
			output.CreatedAt = cwft.CreationTimestamp.Format(time.RFC3339)
		}

		// Extract arguments/parameters
		if cwft.Spec.Arguments.Parameters != nil {
			output.Arguments = make([]TemplateParameterInfo, 0, len(cwft.Spec.Arguments.Parameters))
			for _, param := range cwft.Spec.Arguments.Parameters {
				info := TemplateParameterInfo{
					Name: param.Name,
				}
				if param.Description != nil {
					info.Description = param.Description.String()
				}
				if param.Default != nil {
					info.Default = param.Default.String()
				}
				if param.Enum != nil {
					info.Enum = make([]string, len(param.Enum))
					for i, e := range param.Enum {
						info.Enum[i] = string(e)
					}
				}
				output.Arguments = append(output.Arguments, info)
			}
		}

		// Extract template definitions
		if cwft.Spec.Templates != nil {
			output.Templates = make([]TemplateInfo, 0, len(cwft.Spec.Templates))
			for _, tmpl := range cwft.Spec.Templates {
				info := TemplateInfo{
					Name: tmpl.Name,
					Type: determineTemplateType(&tmpl),
				}
				output.Templates = append(output.Templates, info)
			}
		}

		// Check for workflow template reference
		if cwft.Spec.WorkflowTemplateRef != nil {
			output.WorkflowTemplateRef = &TemplateRef{
				Name: cwft.Spec.WorkflowTemplateRef.Name,
			}
		}

		// Build human-readable result
		resultText := fmt.Sprintf("ClusterWorkflowTemplate %q", output.Name)
		if output.Entrypoint != "" {
			resultText += fmt.Sprintf("\nEntrypoint: %s", output.Entrypoint)
		}
		if len(output.Templates) > 0 {
			resultText += fmt.Sprintf("\nTemplates: %d defined", len(output.Templates))
		}
		if len(output.Arguments) > 0 {
			resultText += fmt.Sprintf("\nParameters: %d defined", len(output.Arguments))
		}

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}

		return result, output, nil
	}
}
