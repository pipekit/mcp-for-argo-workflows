// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/pipekit/mcp-for-argo-workflows/pkg/argo"
)

// ListContextsToolName is the registered name of the list_contexts tool.
const ListContextsToolName = "list_contexts"

// ListContextsInput defines the input parameters for the list_contexts tool.
type ListContextsInput struct{}

// ListContextsOutput defines the output for the list_contexts tool.
type ListContextsOutput struct {
	// Default is the context used when a tool call does not specify one.
	Default string `json:"default,omitempty"`

	// Contexts are the kubeconfig context names available for the context
	// parameter on other tools.
	Contexts []string `json:"contexts"`
}

// ListContextsTool returns the MCP tool definition for list_contexts.
func ListContextsTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        ListContextsToolName,
		Description: "List the kubeconfig context names available for the optional 'context' parameter on other tools, and the default context used when none is specified.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// ListContextsHandler returns a handler function for the list_contexts tool.
func ListContextsHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, ListContextsInput) (*mcp.CallToolResult, *ListContextsOutput, error) {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ ListContextsInput) (*mcp.CallToolResult, *ListContextsOutput, error) {
		names, defaultContext, err := client.ListKubeContexts()
		if err != nil {
			return nil, nil, err
		}
		return nil, &ListContextsOutput{Contexts: names, Default: defaultContext}, nil
	}
}
