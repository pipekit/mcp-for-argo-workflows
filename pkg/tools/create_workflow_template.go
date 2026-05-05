// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// CreateWorkflowTemplateInput defines the input parameters for the create_workflow_template tool.
type CreateWorkflowTemplateInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Manifest is the WorkflowTemplate YAML manifest.
	Manifest string `json:"manifest" jsonschema:"WorkflowTemplate YAML manifest,required"`
}

// CreateWorkflowTemplateOutput defines the output for the create_workflow_template tool.
type CreateWorkflowTemplateOutput struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	CreatedAt string `json:"createdAt,omitempty"`
	Created   bool   `json:"created"`
}

// CreateWorkflowTemplateTool returns the MCP tool definition for create_workflow_template.
func CreateWorkflowTemplateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "create_workflow_template",
		Description: "Create or update a WorkflowTemplate from a YAML manifest. If the template already exists, it will be updated.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(true),
		},
	}
}

// CreateWorkflowTemplateHandler returns a handler function for the create_workflow_template tool.
func CreateWorkflowTemplateHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, CreateWorkflowTemplateInput) (*mcp.CallToolResult, *CreateWorkflowTemplateOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CreateWorkflowTemplateInput) (*mcp.CallToolResult, *CreateWorkflowTemplateOutput, error) {
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
		namespace := ResolveNamespace(input.Namespace, client)
		wft.Namespace = namespace

		// Get the workflow template service client
		wftService, err := client.WorkflowTemplateService()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get workflow template service: %w", err)
		}

		// Try to create the workflow template first
		var resultWft *wfv1.WorkflowTemplate
		var created bool

		resultWft, err = wftService.CreateWorkflowTemplate(ctx, &workflowtemplate.WorkflowTemplateCreateRequest{
			Namespace: namespace,
			Template:  &wft,
		})

		if err != nil {
			// Check if error is AlreadyExists - if so, try to update instead
			if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.AlreadyExists {
				// Get the existing template to retrieve its resourceVersion
				existingWft, getErr := wftService.GetWorkflowTemplate(ctx, &workflowtemplate.WorkflowTemplateGetRequest{
					Namespace: namespace,
					Name:      wft.Name,
				})
				if getErr != nil {
					return nil, nil, fmt.Errorf("failed to get existing workflow template for update: %w", getErr)
				}

				// Copy the resourceVersion to enable update
				wft.ResourceVersion = existingWft.ResourceVersion

				// Update the existing template
				resultWft, err = wftService.UpdateWorkflowTemplate(ctx, &workflowtemplate.WorkflowTemplateUpdateRequest{
					Namespace: namespace,
					Name:      wft.Name,
					Template:  &wft,
				})
				if err != nil {
					return nil, nil, fmt.Errorf("failed to update workflow template: %w", err)
				}
				created = false
			} else {
				return nil, nil, fmt.Errorf("failed to create workflow template: %w", err)
			}
		} else {
			created = true
		}

		// Build the output
		output := &CreateWorkflowTemplateOutput{
			Name:      resultWft.Name,
			Namespace: resultWft.Namespace,
			Created:   created,
		}

		// Format timestamp
		if !resultWft.CreationTimestamp.IsZero() {
			output.CreatedAt = resultWft.CreationTimestamp.Format(time.RFC3339)
		}

		// Build human-readable result
		var resultText string
		if created {
			resultText = fmt.Sprintf("WorkflowTemplate %q created in namespace %q", output.Name, output.Namespace)
		} else {
			resultText = fmt.Sprintf("WorkflowTemplate %q updated in namespace %q", output.Name, output.Namespace)
		}

		return TextResult(resultText), output, nil
	}
}
