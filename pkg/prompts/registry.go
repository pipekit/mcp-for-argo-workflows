// Package prompts implements MCP prompt handlers for Argo Workflows operations.
package prompts

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// PromptRegistrar is a function that registers a prompt with the MCP server.
type PromptRegistrar func(s *mcp.Server, client argo.ClientInterface)

// AllPrompts returns all prompt registrars in the order they should be registered.
func AllPrompts() []PromptRegistrar {
	return []PromptRegistrar{
		RegisterWhyDidThisFail,
	}
}

// RegisterAll registers all prompts with the MCP server.
func RegisterAll(s *mcp.Server, client argo.ClientInterface) {
	for _, register := range AllPrompts() {
		register(s, client)
	}
}

// RegisterWhyDidThisFail registers the why_did_this_fail prompt.
func RegisterWhyDidThisFail(s *mcp.Server, client argo.ClientInterface) {
	s.AddPrompt(WhyDidThisFailPrompt(), WhyDidThisFailHandler(client))
}
