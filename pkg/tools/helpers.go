// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/pipekit/mcp-for-argo-workflows/pkg/argo"
)

// KubeContextInput is embedded in the input of tools that operate against the
// cluster, adding the optional per-call kubeconfig context parameter. The
// property is stripped from tool schemas when multi-context is unavailable.
type KubeContextInput struct {
	// KubeContext selects the kubeconfig context to run the call against.
	KubeContext string `json:"context,omitempty" jsonschema:"Kubeconfig context to run this call against (defaults to the server's configured context)"`
}

// ResolveClient resolves the client and call context for the requested
// kubeconfig context. Handlers must call this before using the captured
// client or request context: the Argo SDK embeds each cluster's
// authentication in its client's context, so both the returned client AND the
// returned context must be used for the call to reach the selected cluster.
func ResolveClient(ctx context.Context, client argo.ClientInterface, kubeContext string) (context.Context, argo.ClientInterface, error) {
	kubeContext = strings.TrimSpace(kubeContext)
	if kubeContext == "" {
		return ctx, client, nil
	}
	resolved, err := client.ForKubeContext(kubeContext)
	if err != nil {
		return nil, nil, err
	}
	if resolved == client {
		return ctx, client, nil
	}
	slog.Info("tool call using kubeconfig context", "context", kubeContext)
	return argo.MergeContext(ctx, resolved.Context()), resolved, nil
}

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
