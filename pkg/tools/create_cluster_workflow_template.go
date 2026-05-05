// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// CreateClusterWorkflowTemplateInput defines the input parameters for the create_cluster_workflow_template tool.
type CreateClusterWorkflowTemplateInput struct {
	// Manifest is the ClusterWorkflowTemplate YAML manifest.
	Manifest string `json:"manifest" jsonschema:"ClusterWorkflowTemplate YAML manifest,required"`
}

// CreateClusterWorkflowTemplateOutput defines the output for the create_cluster_workflow_template tool.
type CreateClusterWorkflowTemplateOutput struct {
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt,omitempty"`
	Created   bool   `json:"created"`
}

// CreateClusterWorkflowTemplateTool returns the MCP tool definition for create_cluster_workflow_template.
func CreateClusterWorkflowTemplateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "create_cluster_workflow_template",
		Description: "Create or update a ClusterWorkflowTemplate from a YAML manifest. If the template already exists, it will be updated.",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(true),
		},
	}
}

// CreateClusterWorkflowTemplateHandler returns a handler function for the create_cluster_workflow_template tool.
func CreateClusterWorkflowTemplateHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, CreateClusterWorkflowTemplateInput) (*mcp.CallToolResult, *CreateClusterWorkflowTemplateOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input CreateClusterWorkflowTemplateInput) (*mcp.CallToolResult, *CreateClusterWorkflowTemplateOutput, error) {
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

		// Try to create the cluster workflow template first
		var resultCwft *wfv1.ClusterWorkflowTemplate
		var created bool

		resultCwft, err = cwftService.CreateClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateCreateRequest{
			Template: &cwft,
		})

		if err != nil {
			// Check if error is AlreadyExists - if so, try to update instead
			if grpcStatus, ok := status.FromError(err); ok && grpcStatus.Code() == codes.AlreadyExists {
				// Get the existing template to retrieve its resourceVersion
				existingCwft, getErr := cwftService.GetClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateGetRequest{
					Name: cwft.Name,
				})
				if getErr != nil {
					return nil, nil, fmt.Errorf("failed to get existing cluster workflow template for update: %w", getErr)
				}

				// Copy the resourceVersion to enable update
				cwft.ResourceVersion = existingCwft.ResourceVersion

				// Update the existing template
				resultCwft, err = cwftService.UpdateClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateUpdateRequest{
					Name:     cwft.Name,
					Template: &cwft,
				})
				if err != nil {
					return nil, nil, fmt.Errorf("failed to update cluster workflow template: %w", err)
				}
				created = false
			} else {
				return nil, nil, fmt.Errorf("failed to create cluster workflow template: %w", err)
			}
		} else {
			created = true
		}

		// Build the output
		output := &CreateClusterWorkflowTemplateOutput{
			Name:    resultCwft.Name,
			Created: created,
		}

		// Format timestamp
		if !resultCwft.CreationTimestamp.IsZero() {
			output.CreatedAt = resultCwft.CreationTimestamp.Format(time.RFC3339)
		}

		// Build human-readable result
		var resultText string
		if created {
			resultText = fmt.Sprintf("ClusterWorkflowTemplate %q created", output.Name)
		} else {
			resultText = fmt.Sprintf("ClusterWorkflowTemplate %q updated", output.Name)
		}

		return TextResult(resultText), output, nil
	}
}
