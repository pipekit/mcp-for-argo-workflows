// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowarchive"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"k8s.io/utils/ptr"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// =============================================================================
// Delete Archived Workflow
// =============================================================================

// DeleteArchivedWorkflowInput defines the input parameters for the delete_archived_workflow tool.
type DeleteArchivedWorkflowInput struct {
	// UID is the unique identifier of the archived workflow.
	UID string `json:"uid" jsonschema:"Workflow UID,required"`
}

// DeleteArchivedWorkflowOutput defines the output for the delete_archived_workflow tool.
type DeleteArchivedWorkflowOutput struct {
	// UID is the UID of the deleted workflow.
	UID string `json:"uid"`

	// Message confirms the deletion.
	Message string `json:"message"`
}

// DeleteArchivedWorkflowTool returns the MCP tool definition for delete_archived_workflow.
func DeleteArchivedWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "delete_archived_workflow",
		Description: "Delete an archived workflow from the archive. Requires Argo Server connection (not available in direct K8s mode).",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(true),
		},
	}
}

// DeleteArchivedWorkflowHandler returns a handler function for the delete_archived_workflow tool.
func DeleteArchivedWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, DeleteArchivedWorkflowInput) (*mcp.CallToolResult, *DeleteArchivedWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input DeleteArchivedWorkflowInput) (*mcp.CallToolResult, *DeleteArchivedWorkflowOutput, error) {
		// Validate UID is provided
		if strings.TrimSpace(input.UID) == "" {
			return nil, nil, fmt.Errorf("workflow UID cannot be empty")
		}

		// Get the archived workflow service client
		archiveService, err := client.ArchivedWorkflowService()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get archived workflow service: %w", err)
		}

		// Delete the archived workflow
		_, err = archiveService.DeleteArchivedWorkflow(ctx, &workflowarchive.DeleteArchivedWorkflowRequest{
			Uid: input.UID,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to delete archived workflow: %w", err)
		}

		// Build output
		output := &DeleteArchivedWorkflowOutput{
			UID:     input.UID,
			Message: fmt.Sprintf("Archived workflow %s deleted successfully", input.UID),
		}

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: output.Message},
			},
		}

		return result, output, nil
	}
}

// =============================================================================
// Resubmit Archived Workflow
// =============================================================================

// ResubmitArchivedWorkflowInput defines the input parameters for the resubmit_archived_workflow tool.
type ResubmitArchivedWorkflowInput struct {
	// UID is the unique identifier of the archived workflow.
	UID string `json:"uid" jsonschema:"Workflow UID,required"`

	// Namespace to submit to (uses original if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Namespace to submit to (uses original if not specified)"`

	// Memoized enables re-use of successful memoized steps.
	Memoized bool `json:"memoized,omitempty" jsonschema:"Re-use successful memoized steps"`
}

// ResubmitArchivedWorkflowOutput defines the output for the resubmit_archived_workflow tool.
type ResubmitArchivedWorkflowOutput struct {
	// Name is the name of the new workflow.
	Name string `json:"name"`

	// Namespace is the namespace of the new workflow.
	Namespace string `json:"namespace"`

	// UID is the UID of the new workflow.
	UID string `json:"uid"`

	// Message describes the result.
	Message string `json:"message"`
}

// ResubmitArchivedWorkflowTool returns the MCP tool definition for resubmit_archived_workflow.
func ResubmitArchivedWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "resubmit_archived_workflow",
		Description: "Resubmit an archived workflow as a new workflow. Requires Argo Server connection (not available in direct K8s mode).",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(false),
		},
	}
}

// ResubmitArchivedWorkflowHandler returns a handler function for the resubmit_archived_workflow tool.
func ResubmitArchivedWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, ResubmitArchivedWorkflowInput) (*mcp.CallToolResult, *ResubmitArchivedWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input ResubmitArchivedWorkflowInput) (*mcp.CallToolResult, *ResubmitArchivedWorkflowOutput, error) {
		// Validate UID is provided
		if strings.TrimSpace(input.UID) == "" {
			return nil, nil, fmt.Errorf("workflow UID cannot be empty")
		}

		// Get the archived workflow service client
		archiveService, err := client.ArchivedWorkflowService()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get archived workflow service: %w", err)
		}

		// Resubmit the archived workflow
		wf, err := archiveService.ResubmitArchivedWorkflow(ctx, &workflowarchive.ResubmitArchivedWorkflowRequest{
			Uid:       input.UID,
			Namespace: input.Namespace,
			Memoized:  input.Memoized,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to resubmit archived workflow: %w", err)
		}

		// Build output
		output := &ResubmitArchivedWorkflowOutput{
			Name:      wf.Name,
			Namespace: wf.Namespace,
			UID:       string(wf.UID),
			Message:   fmt.Sprintf("Archived workflow resubmitted as %s in namespace %s", wf.Name, wf.Namespace),
		}

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: output.Message},
			},
		}

		return result, output, nil
	}
}

// =============================================================================
// Retry Archived Workflow
// =============================================================================

// RetryArchivedWorkflowInput defines the input parameters for the retry_archived_workflow tool.
type RetryArchivedWorkflowInput struct {
	// UID is the unique identifier of the archived workflow.
	UID string `json:"uid" jsonschema:"Workflow UID,required"`

	// Namespace to retry in (uses original if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Namespace to retry in (uses original if not specified)"`

	// NodeFieldSelector selects specific nodes to retry.
	NodeFieldSelector string `json:"nodeFieldSelector,omitempty" jsonschema:"Selector to filter nodes to retry"`

	// RestartSuccessful restarts successful nodes as well.
	RestartSuccessful bool `json:"restartSuccessful,omitempty" jsonschema:"Restart successful nodes as well"`
}

// RetryArchivedWorkflowOutput defines the output for the retry_archived_workflow tool.
type RetryArchivedWorkflowOutput struct {
	// Name is the name of the retried workflow.
	Name string `json:"name"`

	// Namespace is the namespace of the retried workflow.
	Namespace string `json:"namespace"`

	// UID is the UID of the retried workflow.
	UID string `json:"uid"`

	// Message describes the result.
	Message string `json:"message"`
}

// RetryArchivedWorkflowTool returns the MCP tool definition for retry_archived_workflow.
func RetryArchivedWorkflowTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "retry_archived_workflow",
		Description: "Retry a failed archived workflow. Requires Argo Server connection (not available in direct K8s mode).",
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr.To(true),
		},
	}
}

// RetryArchivedWorkflowHandler returns a handler function for the retry_archived_workflow tool.
func RetryArchivedWorkflowHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, RetryArchivedWorkflowInput) (*mcp.CallToolResult, *RetryArchivedWorkflowOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input RetryArchivedWorkflowInput) (*mcp.CallToolResult, *RetryArchivedWorkflowOutput, error) {
		// Validate UID is provided
		if strings.TrimSpace(input.UID) == "" {
			return nil, nil, fmt.Errorf("workflow UID cannot be empty")
		}

		// Get the archived workflow service client
		archiveService, err := client.ArchivedWorkflowService()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get archived workflow service: %w", err)
		}

		// Retry the archived workflow
		wf, err := archiveService.RetryArchivedWorkflow(ctx, &workflowarchive.RetryArchivedWorkflowRequest{
			Uid:               input.UID,
			Namespace:         input.Namespace,
			RestartSuccessful: input.RestartSuccessful,
			NodeFieldSelector: input.NodeFieldSelector,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to retry archived workflow: %w", err)
		}

		// Build output
		output := &RetryArchivedWorkflowOutput{
			Name:      wf.Name,
			Namespace: wf.Namespace,
			UID:       string(wf.UID),
			Message:   fmt.Sprintf("Archived workflow retried as %s in namespace %s", wf.Name, wf.Namespace),
		}

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: output.Message},
			},
		}

		return result, output, nil
	}
}
