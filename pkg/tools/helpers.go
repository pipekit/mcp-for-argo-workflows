// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// ResolveNamespace returns the trimmed namespace if provided, otherwise falls back
// to the client's default namespace.
func ResolveNamespace(namespace string, client argo.ClientInterface) string {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return client.DefaultNamespace()
	}
	return namespace
}

// ValidateName trims and validates a workflow name, returning an error if empty.
func ValidateName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("workflow name cannot be empty")
	}
	return name, nil
}

// TextResult creates a CallToolResult with a single text content.
func TextResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}
