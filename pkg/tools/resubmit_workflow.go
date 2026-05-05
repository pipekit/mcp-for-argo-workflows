// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/utils/ptr"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// ResubmitWorkflowInput defines the input parameters for the resubmit_workflow tool.
type ResubmitWorkflowInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Name is the workflow name to resubmit.
	Name string `json:"name" jsonschema:"Workflow name to resubmit,required"`

	// Parameters are parameter overrides in key=value format.
	Parameters []string `json:"parameters,omitempty" jsonschema:"Parameter overrides in key=value format"`

	// Memoized indicates whether to re-use successful memoized steps.
	Memoized bool `json:"memoized,omitempty" jsonschema:"Re-use successful memoized steps"`
}

// ResubmitWorkflowOutput defines the output for the resubmit_workflow tool.
type ResubmitWorkflowOutput struct {
	// Name is the new workflow name.
	Name string `json:"name"`

	// Namespace is the namespace of the new workflow.
	Namespace string `json:"namespace"`

	// UID is the unique identifier of the new workflow.
	UID string `json:"uid"`

	// Phase is the initial workflow phase.
	Phase string `json:"phase"`

	// Message provides additional status information.
	Message string `json:"message,omitempty"`

	// OriginalWorkflow is the name of the original workflow that was resubmitted.
	OriginalWorkflow string `json:"originalWorkflow"`
}

// ResubmitWorkflowTool returns the MCP tool definition for resubmit_workflow.
func ResubmitWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "resubmit_workflow",
		Description: "Resubmit a completed Argo Workflow, creating a new workflow execution",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(false),
		},
	}
}

// ResubmitWorkflowHandler returns a handler function for the resubmit_workflow tool.
func ResubmitWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, ResubmitWorkflowInput) (*mcp.CallToolResult, *ResubmitWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ResubmitWorkflowInput) (*mcp.CallToolResult, *ResubmitWorkflowOutput, error) {
		// Validate and normalize name
		workflowName := strings.TrimSpace(input.Name)
		if workflowName == "" {
			return nil, nil, fmt.Errorf("workflow name cannot be empty")
		}

		// Determine namespace
		namespace := strings.TrimSpace(input.Namespace)
		if namespace == "" {
			namespace = client.DefaultNamespace()
		}

		// Get the workflow service client
		wfService := client.WorkflowService()

		// Build the resubmit request
		req := &workflow.WorkflowResubmitRequest{
			Name:       workflowName,
			Namespace:  namespace,
			Memoized:   input.Memoized,
			Parameters: input.Parameters,
		}

		// Resubmit the workflow
		newWf, err := wfService.ResubmitWorkflow(ctx, req)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resubmit workflow: %w", err)
		}

		// Build the output
		output := &ResubmitWorkflowOutput{
			Name:             newWf.Name,
			Namespace:        newWf.Namespace,
			UID:              string(newWf.UID),
			Phase:            string(newWf.Status.Phase),
			Message:          newWf.Status.Message,
			OriginalWorkflow: workflowName,
		}

		// Set a default phase if empty
		if output.Phase == "" {
			output.Phase = PhasePending
		}

		// Build human-readable result
		resultText := fmt.Sprintf("Workflow %q resubmitted as %q in namespace %q. Phase: %s",
			workflowName, output.Name, output.Namespace, output.Phase)

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}

		return result, output, nil
	}
}
