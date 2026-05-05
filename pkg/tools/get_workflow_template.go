// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// GetWorkflowTemplateInput defines the input parameters for the get_workflow_template tool.
type GetWorkflowTemplateInput struct {
	// Name is the workflow template name (required).
	Name string `json:"name" jsonschema:"Workflow template name,required"`

	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`
}

// TemplateRef represents a reference to a template.
type TemplateRef struct {
	// Name is the template name.
	Name string `json:"name"`

	// Template is the template being referenced.
	Template string `json:"template,omitempty"`
}

// TemplateParameterInfo represents a parameter definition in a workflow template.
type TemplateParameterInfo struct {
	// Name is the parameter name.
	Name string `json:"name"`

	// Default is the default value.
	Default string `json:"default,omitempty"`

	// Description is the parameter description.
	Description string `json:"description,omitempty"`

	// Enum lists allowed values.
	Enum []string `json:"enum,omitempty"`
}

// TemplateInfo represents a template definition within the workflow template.
type TemplateInfo struct {
	// Name is the template name.
	Name string `json:"name"`

	// Type indicates what kind of template this is (container, script, dag, steps, etc.).
	Type string `json:"type"`
}

// GetWorkflowTemplateOutput defines the output for the get_workflow_template tool.
type GetWorkflowTemplateOutput struct {
	Labels              map[string]string       `json:"labels,omitempty"`
	Annotations         map[string]string       `json:"annotations,omitempty"`
	WorkflowTemplateRef *TemplateRef            `json:"workflowTemplateRef,omitempty"`
	Name                string                  `json:"name"`
	Namespace           string                  `json:"namespace"`
	CreatedAt           string                  `json:"createdAt,omitempty"`
	Entrypoint          string                  `json:"entrypoint,omitempty"`
	Arguments           []TemplateParameterInfo `json:"arguments,omitempty"`
	Templates           []TemplateInfo          `json:"templates,omitempty"`
}

// GetWorkflowTemplateTool returns the MCP tool definition for get_workflow_template.
func GetWorkflowTemplateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_workflow_template",
		Description: "Get details of a specific WorkflowTemplate",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// GetWorkflowTemplateHandler returns a handler function for the get_workflow_template tool.
func GetWorkflowTemplateHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, GetWorkflowTemplateInput) (*mcp.CallToolResult, *GetWorkflowTemplateOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input GetWorkflowTemplateInput) (*mcp.CallToolResult, *GetWorkflowTemplateOutput, error) {
		// Validate required input
		if input.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}

		// Determine namespace
		namespace := input.Namespace
		if namespace == "" {
			namespace = client.DefaultNamespace()
		}

		// Get the workflow template service client
		wftService, err := client.WorkflowTemplateService()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get workflow template service: %w", err)
		}

		// Get the workflow template
		wft, err := wftService.GetWorkflowTemplate(ctx, &workflowtemplate.WorkflowTemplateGetRequest{
			Name:      input.Name,
			Namespace: namespace,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get workflow template: %w", err)
		}

		// Build output
		output := &GetWorkflowTemplateOutput{
			Name:        wft.Name,
			Namespace:   wft.Namespace,
			Labels:      wft.Labels,
			Annotations: wft.Annotations,
			Entrypoint:  wft.Spec.Entrypoint,
		}

		// Format timestamp
		if !wft.CreationTimestamp.IsZero() {
			output.CreatedAt = wft.CreationTimestamp.Format(time.RFC3339)
		}

		// Extract arguments/parameters
		if wft.Spec.Arguments.Parameters != nil {
			output.Arguments = make([]TemplateParameterInfo, 0, len(wft.Spec.Arguments.Parameters))
			for _, param := range wft.Spec.Arguments.Parameters {
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
		if wft.Spec.Templates != nil {
			output.Templates = make([]TemplateInfo, 0, len(wft.Spec.Templates))
			for _, tmpl := range wft.Spec.Templates {
				info := TemplateInfo{
					Name: tmpl.Name,
					Type: determineTemplateType(&tmpl),
				}
				output.Templates = append(output.Templates, info)
			}
		}

		// Check for workflow template reference
		if wft.Spec.WorkflowTemplateRef != nil {
			output.WorkflowTemplateRef = &TemplateRef{
				Name: wft.Spec.WorkflowTemplateRef.Name,
			}
		}

		// Build human-readable result
		resultText := fmt.Sprintf("WorkflowTemplate %q in namespace %q", output.Name, output.Namespace)
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

// determineTemplateType determines the type of a template based on its definition.
func determineTemplateType(tmpl *wfv1.Template) string {
	switch {
	case tmpl.Container != nil:
		return "container"
	case tmpl.Script != nil:
		return "script"
	case tmpl.DAG != nil:
		return "dag"
	case tmpl.Steps != nil:
		return "steps"
	case tmpl.Resource != nil:
		return "resource"
	case tmpl.Suspend != nil:
		return "suspend"
	case tmpl.HTTP != nil:
		return "http"
	case tmpl.Plugin != nil:
		return "plugin"
	case tmpl.ContainerSet != nil:
		return "containerSet"
	case tmpl.Data != nil:
		return "data"
	default:
		return "unknown"
	}
}
