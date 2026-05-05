// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/argoproj/argo-workflows/v4/workflow/convert"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"sigs.k8s.io/yaml"
)

// Output format constants.
const (
	// OutputFormatYAML is the YAML output format.
	OutputFormatYAML = "yaml"
	// OutputFormatJSON is the JSON output format.
	OutputFormatJSON = "json"
)

// ConvertWorkflowInput defines the input parameters for the convert_workflow tool.
type ConvertWorkflowInput struct {
	// Manifest is the workflow YAML manifest to convert.
	Manifest string `json:"manifest" jsonschema:"Workflow YAML manifest to convert,required"`

	// OutputFormat is the output format (yaml or json).
	OutputFormat string `json:"outputFormat,omitempty" jsonschema:"Output format: yaml (default) or json,enum=yaml,enum=json"`
}

// ConvertWorkflowOutput defines the output for the convert_workflow tool.
type ConvertWorkflowOutput struct {
	// Manifest is the converted manifest.
	Manifest string `json:"manifest"`

	// Format is the output format used.
	Format string `json:"format"`

	// Kind is the kind of manifest that was converted.
	Kind string `json:"kind"`

	// Name is the name of the workflow/template.
	Name string `json:"name,omitempty"`

	// Changes is a list of changes made during conversion.
	Changes []string `json:"changes,omitempty"`

	// Warnings is a list of warnings for manual review.
	Warnings []string `json:"warnings,omitempty"`
}

// ConvertWorkflowTool returns the MCP tool definition for convert_workflow.
func ConvertWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "convert_workflow",
		Description: "Convert an Argo Workflow manifest to a newer format, migrating deprecated fields. Supports Workflow, WorkflowTemplate, ClusterWorkflowTemplate, and CronWorkflow manifests.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// ConvertWorkflowHandler returns a handler function for the convert_workflow tool.
// Note: This tool doesn't require the Argo client since it works purely from YAML.
// It uses the official argo-workflows/workflow/convert package to perform conversions.
func ConvertWorkflowHandler() func(context.Context, *mcp.CallToolRequest, ConvertWorkflowInput) (*mcp.CallToolResult, *ConvertWorkflowOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, input ConvertWorkflowInput) (*mcp.CallToolResult, *ConvertWorkflowOutput, error) {
		// Validate manifest is provided
		if strings.TrimSpace(input.Manifest) == "" {
			return nil, nil, fmt.Errorf("manifest cannot be empty")
		}

		// Guard against oversized manifests (DoS hardening)
		const maxManifestBytes = 1 << 20 // 1 MiB
		if len(input.Manifest) > maxManifestBytes {
			return nil, nil, fmt.Errorf("manifest too large (%d bytes), max %d", len(input.Manifest), maxManifestBytes)
		}

		// Determine output format
		outputFormat := strings.ToLower(strings.TrimSpace(input.OutputFormat))
		if outputFormat == "" {
			outputFormat = OutputFormatYAML
		}
		if outputFormat != OutputFormatYAML && outputFormat != OutputFormatJSON {
			return nil, nil, fmt.Errorf("invalid output format: %s (must be %s or %s)", outputFormat, OutputFormatYAML, OutputFormatJSON)
		}

		// First, determine the kind of manifest
		var generic struct {
			Kind     string `json:"kind"`
			Metadata struct {
				Name         string `json:"name"`
				GenerateName string `json:"generateName"`
			} `json:"metadata"`
		}
		if err := yaml.Unmarshal([]byte(input.Manifest), &generic); err != nil {
			return nil, nil, fmt.Errorf("failed to parse manifest: %w", err)
		}

		kind := generic.Kind
		name := generic.Metadata.Name
		if name == "" {
			name = generic.Metadata.GenerateName
		}

		var changes []string
		var warnings []string
		var convertedManifest string

		// Convert based on kind using the official argo-workflows convert package
		switch kind {
		case KindWorkflow, "":
			var legacy convert.LegacyWorkflow
			if err := yaml.Unmarshal([]byte(input.Manifest), &legacy); err != nil {
				return nil, nil, fmt.Errorf("failed to parse %s manifest: %w", KindWorkflow, err)
			}
			if name == "" {
				name = legacy.Name
				if name == "" {
					name = legacy.GenerateName
				}
			}
			if kind == "" {
				kind = KindWorkflow
			}

			// Detect changes before conversion
			changes = detectWorkflowSpecChanges(&legacy.Spec)

			// Convert using official package
			converted := legacy.ToCurrent()

			// Serialize back
			var err error
			convertedManifest, err = serializeManifest(converted, outputFormat)
			if err != nil {
				return nil, nil, err
			}

		case KindWorkflowTemplate:
			var legacy convert.LegacyWorkflowTemplate
			if err := yaml.Unmarshal([]byte(input.Manifest), &legacy); err != nil {
				return nil, nil, fmt.Errorf("failed to parse %s manifest: %w", KindWorkflowTemplate, err)
			}
			if name == "" {
				name = legacy.Name
			}

			// Detect changes before conversion
			changes = detectWorkflowSpecChanges(&legacy.Spec)

			// Convert using official package
			converted := legacy.ToCurrent()

			// Serialize back
			var err error
			convertedManifest, err = serializeManifest(converted, outputFormat)
			if err != nil {
				return nil, nil, err
			}

		case KindClusterWorkflowTemplate:
			var legacy convert.LegacyClusterWorkflowTemplate
			if err := yaml.Unmarshal([]byte(input.Manifest), &legacy); err != nil {
				return nil, nil, fmt.Errorf("failed to parse %s manifest: %w", KindClusterWorkflowTemplate, err)
			}
			if name == "" {
				name = legacy.Name
			}

			// Detect changes before conversion
			changes = detectWorkflowSpecChanges(&legacy.Spec)

			// Convert using official package
			converted := legacy.ToCurrent()

			// Serialize back
			var err error
			convertedManifest, err = serializeManifest(converted, outputFormat)
			if err != nil {
				return nil, nil, err
			}

		case KindCronWorkflow:
			var legacy convert.LegacyCronWorkflow
			if err := yaml.Unmarshal([]byte(input.Manifest), &legacy); err != nil {
				return nil, nil, fmt.Errorf("failed to parse %s manifest: %w", KindCronWorkflow, err)
			}
			if name == "" {
				name = legacy.Name
			}

			// Detect CronWorkflow-specific changes
			cronChanges, cronWarnings := detectCronWorkflowSpecChanges(&legacy.Spec)
			changes = append(changes, cronChanges...)
			warnings = append(warnings, cronWarnings...)

			// Detect WorkflowSpec changes
			wfChanges := detectWorkflowSpecChanges(&legacy.Spec.WorkflowSpec)
			changes = append(changes, wfChanges...)

			// Convert using official package
			converted := legacy.ToCurrent()

			// Serialize back
			var err error
			convertedManifest, err = serializeManifest(converted, outputFormat)
			if err != nil {
				return nil, nil, err
			}

		default:
			return nil, nil, fmt.Errorf("unsupported manifest kind: %s (must be %s, %s, %s, or %s)", kind, KindWorkflow, KindWorkflowTemplate, KindClusterWorkflowTemplate, KindCronWorkflow)
		}

		// Build output
		output := &ConvertWorkflowOutput{
			Manifest: convertedManifest,
			Format:   outputFormat,
			Kind:     kind,
			Name:     name,
			Changes:  changes,
			Warnings: warnings,
		}

		// Build human-readable result
		var resultText strings.Builder
		fmt.Fprintf(&resultText, "Converted %s %q to %s format\n", kind, name, outputFormat)

		if len(changes) > 0 {
			resultText.WriteString("\nChanges made:\n")
			for _, change := range changes {
				fmt.Fprintf(&resultText, "  - %s\n", change)
			}
		} else {
			resultText.WriteString("\nNo changes needed - manifest is already up to date.\n")
		}

		if len(warnings) > 0 {
			resultText.WriteString("\nWarnings (manual review recommended):\n")
			for _, warning := range warnings {
				fmt.Fprintf(&resultText, "  - %s\n", warning)
			}
		}

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText.String()},
			},
		}

		return result, output, nil
	}
}

// detectWorkflowSpecChanges detects deprecated fields in a LegacyWorkflowSpec before conversion.
func detectWorkflowSpecChanges(spec *convert.LegacyWorkflowSpec) []string {
	var changes []string

	// Check for deprecated synchronization at spec level
	if spec.Synchronization != nil {
		syncChanges := detectSynchronizationChanges(spec.Synchronization)
		changes = append(changes, syncChanges...)
	}

	// Check for deprecated fields in templates
	for i := range spec.Templates {
		tmpl := &spec.Templates[i]
		tmplChanges := detectTemplateChanges(tmpl)
		changes = append(changes, tmplChanges...)
	}

	return changes
}

// detectTemplateChanges detects deprecated fields in a LegacyTemplate.
func detectTemplateChanges(tmpl *convert.LegacyTemplate) []string {
	var changes []string

	// Check for deprecated synchronization in template
	if tmpl.Synchronization != nil {
		syncChanges := detectSynchronizationChanges(tmpl.Synchronization)
		changes = append(changes, syncChanges...)
	}

	return changes
}

// detectSynchronizationChanges detects deprecated mutex/semaphore fields.
func detectSynchronizationChanges(sync *convert.LegacySynchronization) []string {
	var changes []string

	if sync.Mutex != nil {
		changes = append(changes, "Migrated spec.synchronization.mutex to spec.synchronization.mutexes array")
	}
	if sync.Semaphore != nil {
		changes = append(changes, "Migrated spec.synchronization.semaphore to spec.synchronization.semaphores array")
	}

	return changes
}

// detectCronWorkflowSpecChanges detects deprecated fields in a LegacyCronWorkflowSpec.
func detectCronWorkflowSpecChanges(spec *convert.LegacyCronWorkflowSpec) ([]string, []string) {
	var changes []string
	var warnings []string

	// Check for deprecated schedule field
	if spec.Schedule != "" {
		changes = append(changes, "Migrated spec.schedule to spec.schedules array")
	}

	// Check for missing concurrencyPolicy
	if spec.ConcurrencyPolicy == "" {
		warnings = append(warnings, "No concurrencyPolicy set - defaults to 'Allow' which may cause overlapping runs")
	}

	return changes, warnings
}

// serializeManifest serializes the manifest to the specified format.
func serializeManifest(obj interface{}, format string) (string, error) {
	switch format {
	case "json":
		data, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to serialize manifest to JSON: %w", err)
		}
		return string(data), nil
	case "yaml":
		data, err := yaml.Marshal(obj)
		if err != nil {
			return "", fmt.Errorf("failed to serialize manifest to YAML: %w", err)
		}
		return string(data), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}
