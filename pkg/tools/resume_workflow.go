// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/utils/ptr"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// ResumeWorkflowInput defines the input parameters for the resume_workflow tool.
type ResumeWorkflowInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Name is the workflow name.
	Name string `json:"name" jsonschema:"Workflow name,required"`

	// NodeFieldSelector is a selector for specific nodes to resume.
	NodeFieldSelector string `json:"nodeFieldSelector,omitempty" jsonschema:"Selector for specific nodes to resume"`
}

// ResumeWorkflowOutput defines the output for the resume_workflow tool.
type ResumeWorkflowOutput struct {
	// Name is the workflow name.
	Name string `json:"name"`

	// Namespace is the namespace of the workflow.
	Namespace string `json:"namespace"`

	// Phase is the current workflow phase.
	Phase string `json:"phase"`

	// Message provides additional status information.
	Message string `json:"message,omitempty"`
}

// ResumeWorkflowTool returns the MCP tool definition for resume_workflow.
func ResumeWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "resume_workflow",
		Description: "Resume a suspended Argo Workflow",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(false),
		},
	}
}

// ResumeWorkflowHandler returns a handler function for the resume_workflow tool.
func ResumeWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, ResumeWorkflowInput) (*mcp.CallToolResult, *ResumeWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ResumeWorkflowInput) (*mcp.CallToolResult, *ResumeWorkflowOutput, error) {
		// Validate and normalize name
		workflowName, err := ValidateName(input.Name)
		if err != nil {
			return nil, nil, err
		}

		// Determine namespace
		namespace := ResolveNamespace(input.Namespace, client)

		// Get the workflow service client
		wfService := client.WorkflowService()

		// Resume the workflow
		wf, err := wfService.ResumeWorkflow(ctx, &workflow.WorkflowResumeRequest{
			Name:              workflowName,
			Namespace:         namespace,
			NodeFieldSelector: input.NodeFieldSelector,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resume workflow: %w", err)
		}

		// Build the output
		output := &ResumeWorkflowOutput{
			Name:      wf.Name,
			Namespace: wf.Namespace,
			Phase:     string(wf.Status.Phase),
			Message:   wf.Status.Message,
		}

		// Build human-readable result
		resultText := fmt.Sprintf("Workflow %q in namespace %q resumed. Phase: %s",
			output.Name, output.Namespace, output.Phase)

		return TextResult(resultText), output, nil
	}
}
