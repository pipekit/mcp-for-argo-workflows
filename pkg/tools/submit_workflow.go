// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// PhasePending is the default phase for newly created workflows.
const PhasePending = "Pending"

// SubmitWorkflowInput defines the input parameters for the submit_workflow tool.
type SubmitWorkflowInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Manifest is the workflow YAML manifest.
	Manifest string `json:"manifest" jsonschema:"Workflow YAML manifest,required"`

	// GenerateName overrides metadata.generateName.
	GenerateName string `json:"generateName,omitempty" jsonschema:"Override metadata.generateName"`

	// Labels are additional labels to add to the workflow.
	Labels map[string]string `json:"labels,omitempty" jsonschema:"Additional labels to add"`

	// Parameters are parameter overrides in key=value format.
	Parameters []string `json:"parameters,omitempty" jsonschema:"Parameter overrides in key=value format"`
}

// SubmitWorkflowOutput defines the output for the submit_workflow tool.
type SubmitWorkflowOutput struct {
	// Name is the workflow name.
	Name string `json:"name"`

	// Namespace is the namespace where the workflow was created.
	Namespace string `json:"namespace"`

	// UID is the unique identifier of the workflow.
	UID string `json:"uid"`

	// Phase is the initial workflow status phase.
	Phase string `json:"phase"`

	// Message provides additional status information.
	Message string `json:"message,omitempty"`
}

// SubmitWorkflowTool returns the MCP tool definition for submit_workflow.
func SubmitWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "submit_workflow",
		Description: "Submit an Argo Workflow from a YAML manifest. Run lint_workflow first to validate the manifest.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(false),
		},
	}
}

// SubmitWorkflowHandler returns a handler function for the submit_workflow tool.
func SubmitWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, SubmitWorkflowInput) (*mcp.CallToolResult, *SubmitWorkflowOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input SubmitWorkflowInput) (*mcp.CallToolResult, *SubmitWorkflowOutput, error) {
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
		namespace := input.Namespace
		if namespace == "" {
			namespace = client.DefaultNamespace()
		}
		wf.Namespace = namespace

		// Override generateName if provided
		if input.GenerateName != "" {
			wf.Name = ""
			wf.GenerateName = input.GenerateName
		}

		// Add labels if provided
		if len(input.Labels) > 0 {
			if wf.Labels == nil {
				wf.Labels = make(map[string]string)
			}
			for k, v := range input.Labels {
				wf.Labels[k] = v
			}
		}

		// Apply parameter overrides
		if len(input.Parameters) > 0 {
			if err := applyParameterOverrides(&wf, input.Parameters); err != nil {
				return nil, nil, fmt.Errorf("failed to apply parameter overrides: %w", err)
			}
		}

		// Get the workflow service client
		wfService := client.WorkflowService()

		// Create the workflow
		createdWf, err := wfService.CreateWorkflow(ctx, &workflow.WorkflowCreateRequest{
			Namespace: namespace,
			Workflow:  &wf,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create workflow: %w", err)
		}

		// Build the output
		output := &SubmitWorkflowOutput{
			Name:      createdWf.Name,
			Namespace: createdWf.Namespace,
			UID:       string(createdWf.UID),
			Phase:     string(createdWf.Status.Phase),
			Message:   createdWf.Status.Message,
		}

		// Set a default phase if empty (newly created workflows may not have a phase yet)
		if output.Phase == "" {
			output.Phase = PhasePending
		}

		return nil, output, nil
	}
}

// applyParameterOverrides applies parameter overrides to the workflow.
// Parameters should be in "key=value" format.
func applyParameterOverrides(wf *wfv1.Workflow, params []string) error {
	for _, param := range params {
		parts := strings.SplitN(param, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid parameter format %q, expected key=value", param)
		}
		key := strings.TrimSpace(parts[0])
		value := parts[1]
		if key == "" {
			return fmt.Errorf("invalid parameter format %q, key cannot be empty", param)
		}

		// Find and update the parameter in the workflow spec
		found := false
		for i, p := range wf.Spec.Arguments.Parameters {
			if p.Name == key {
				wf.Spec.Arguments.Parameters[i].Value = wfv1.AnyStringPtr(value)
				found = true
				break
			}
		}

		// If not found, add as a new parameter
		if !found {
			wf.Spec.Arguments.Parameters = append(wf.Spec.Arguments.Parameters, wfv1.Parameter{
				Name:  key,
				Value: wfv1.AnyStringPtr(value),
			})
		}
	}
	return nil
}
