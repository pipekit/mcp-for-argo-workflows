// Package resources implements MCP resources for Argo Workflows documentation.
package resources

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Definition defines a single MCP resource with its metadata and documentation file.
type Definition struct {
	URI         string
	Name        string
	Title       string
	Description string
	DocFile     string
}

// Resource returns the MCP resource definition.
func (d Definition) Resource() *mcp.Resource {
	return &mcp.Resource{
		URI:         d.URI,
		Name:        d.Name,
		Title:       d.Title,
		Description: d.Description,
		MIMEType:    "text/markdown",
	}
}

// Handler returns the MCP resource handler that serves the documentation content.
func (d Definition) Handler() mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if req.Params.URI != d.URI {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}

		content, err := readDoc(d.DocFile)
		if err != nil {
			return nil, err
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      d.URI,
					MIMEType: "text/markdown",
					Text:     content,
				},
			},
		}, nil
	}
}

// AllDefinitions returns all resource definitions.
// Resources are grouped by category: schemas, template types, and examples.
func AllDefinitions() []Definition {
	return []Definition{
		// CRD Schema Resources
		{
			URI:         "argo://schemas/workflow",
			Name:        "workflow-schema",
			Title:       "Argo Workflow CRD Schema",
			Description: "Complete schema documentation for the Workflow custom resource definition",
			DocFile:     "workflow_schema.md",
		},
		{
			URI:         "argo://schemas/workflow-template",
			Name:        "workflow-template-schema",
			Title:       "Argo WorkflowTemplate CRD Schema",
			Description: "Complete schema documentation for the WorkflowTemplate custom resource definition",
			DocFile:     "workflow_template_schema.md",
		},
		{
			URI:         "argo://schemas/cluster-workflow-template",
			Name:        "cluster-workflow-template-schema",
			Title:       "Argo ClusterWorkflowTemplate CRD Schema",
			Description: "Complete schema documentation for the ClusterWorkflowTemplate custom resource definition",
			DocFile:     "cluster_workflow_template_schema.md",
		},
		{
			URI:         "argo://schemas/cron-workflow",
			Name:        "cron-workflow-schema",
			Title:       "Argo CronWorkflow CRD Schema",
			Description: "Complete schema documentation for the CronWorkflow custom resource definition",
			DocFile:     "cron_workflow_schema.md",
		},

		// Variables, Expressions, and Data Flow Resources
		{
			URI:         "argo://docs/variables",
			Name:        "docs-variables",
			Title:       "Argo Workflows Variables Reference",
			Description: "Complete reference of template variables for inputs, outputs, workflow, pod, and loop variables",
			DocFile:     "variables.md",
		},
		{
			URI:         "argo://docs/expressions",
			Name:        "docs-expressions",
			Title:       "Argo Workflows Expressions Reference",
			Description: "Expression syntax, operators, and functions for data manipulation in workflows",
			DocFile:     "expressions.md",
		},
		{
			URI:         "argo://docs/parameters",
			Name:        "docs-parameters",
			Title:       "Argo Workflows Parameters Reference",
			Description: "Parameter definition, passing patterns, and validation in workflows",
			DocFile:     "parameters.md",
		},
		{
			URI:         "argo://docs/artifacts",
			Name:        "docs-artifacts",
			Title:       "Argo Workflows Artifacts Reference",
			Description: "Artifact system for file passing, storage backends, and artifact repositories",
			DocFile:     "artifacts.md",
		},
		{
			URI:         "argo://docs/outputs",
			Name:        "docs-outputs",
			Title:       "Argo Workflows Outputs Reference",
			Description: "Capturing and using outputs from workflow templates including parameters and artifacts",
			DocFile:     "outputs.md",
		},

		// Template Type Resources
		{
			URI:         "argo://docs/template-types",
			Name:        "template-types-overview",
			Title:       "Argo Workflows Template Types Overview",
			Description: "Documentation for the Argo Workflows Template Types Overview",
			DocFile:     "template_types_overview.md",
		},
		{
			URI:         "argo://docs/template-types/container",
			Name:        "template-types-container",
			Title:       "Container Template Type",
			Description: "Documentation for the Container Template Type",
			DocFile:     "template_types_container.md",
		},
		{
			URI:         "argo://docs/template-types/script",
			Name:        "template-types-script",
			Title:       "Script Template Type",
			Description: "Documentation for the Script Template Type",
			DocFile:     "template_types_script.md",
		},
		{
			URI:         "argo://docs/template-types/dag",
			Name:        "template-types-dag",
			Title:       "DAG Template Type",
			Description: "Documentation for the DAG Template Type",
			DocFile:     "template_types_dag.md",
		},
		{
			URI:         "argo://docs/template-types/steps",
			Name:        "template-types-steps",
			Title:       "Steps Template Type",
			Description: "Documentation for the Steps Template Type",
			DocFile:     "template_types_steps.md",
		},
		{
			URI:         "argo://docs/template-types/suspend",
			Name:        "template-types-suspend",
			Title:       "Suspend Template Type",
			Description: "Documentation for the Suspend Template Type",
			DocFile:     "template_types_suspend.md",
		},
		{
			URI:         "argo://docs/template-types/resource",
			Name:        "template-types-resource",
			Title:       "Resource Template Type",
			Description: "Documentation for the Resource Template Type",
			DocFile:     "template_types_resource.md",
		},
		{
			URI:         "argo://docs/template-types/http",
			Name:        "template-types-http",
			Title:       "HTTP Template Type",
			Description: "Documentation for the HTTP Template Type",
			DocFile:     "template_types_http.md",
		},

		// Example Workflow Resources
		{
			URI:         "argo://examples/hello-world",
			Name:        "examples-hello-world",
			Title:       "Hello World Workflow Example",
			Description: "Simplest workflow example with a single container template",
			DocFile:     "examples_hello_world.md",
		},
		{
			URI:         "argo://examples/multi-step",
			Name:        "examples-multi-step",
			Title:       "Multi-Step Workflow Example",
			Description: "Sequential steps with data passing between steps",
			DocFile:     "examples_multi_step.md",
		},
		{
			URI:         "argo://examples/dag-diamond",
			Name:        "examples-dag-diamond",
			Title:       "DAG Diamond Pattern Example",
			Description: "Classic diamond DAG with fan-out and fan-in pattern",
			DocFile:     "examples_dag_diamond.md",
		},
		{
			URI:         "argo://examples/parameters",
			Name:        "examples-parameters",
			Title:       "Parameters Example",
			Description: "Input parameters, default values, and parameter passing patterns",
			DocFile:     "examples_parameters.md",
		},
		{
			URI:         "argo://examples/artifacts",
			Name:        "examples-artifacts",
			Title:       "Artifacts Example",
			Description: "Artifact passing between steps with S3/GCS configuration",
			DocFile:     "examples_artifacts.md",
		},
		{
			URI:         "argo://examples/loops",
			Name:        "examples-loops",
			Title:       "Loops Example",
			Description: "withItems, withParam, and withSequence for iteration patterns",
			DocFile:     "examples_loops.md",
		},
		{
			URI:         "argo://examples/conditionals",
			Name:        "examples-conditionals",
			Title:       "Conditionals Example",
			Description: "Conditional step execution using when expressions",
			DocFile:     "examples_conditionals.md",
		},
		{
			URI:         "argo://examples/retries",
			Name:        "examples-retries",
			Title:       "Retries Example",
			Description: "Retry strategies and retryPolicy configuration",
			DocFile:     "examples_retries.md",
		},
		{
			URI:         "argo://examples/timeout-limits",
			Name:        "examples-timeout-limits",
			Title:       "Timeout and Limits Example",
			Description: "activeDeadlineSeconds and template-level timeout configurations",
			DocFile:     "examples_timeout_limits.md",
		},
		{
			URI:         "argo://examples/resource-management",
			Name:        "examples-resource-management",
			Title:       "Resource Management Example",
			Description: "CPU/memory requests and limits, pod priority, and resource optimization",
			DocFile:     "examples_resource_management.md",
		},
		{
			URI:         "argo://examples/volumes",
			Name:        "examples-volumes",
			Title:       "Volumes Example",
			Description: "Volume mounts including PVC, ConfigMap, Secret, and shared volumes",
			DocFile:     "examples_volumes.md",
		},
		{
			URI:         "argo://examples/exit-handlers",
			Name:        "examples-exit-handlers",
			Title:       "Exit Handlers Example",
			Description: "OnExit handlers for cleanup and status-specific actions",
			DocFile:     "examples_exit_handlers.md",
		},
	}
}

// RegisterAll registers all resources with the MCP server.
func RegisterAll(s *mcp.Server) {
	for _, def := range AllDefinitions() {
		s.AddResource(def.Resource(), def.Handler())
	}
}
