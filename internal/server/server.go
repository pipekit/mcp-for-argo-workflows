// Package server provides the MCP server implementation for Argo Workflows.
package server

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
	"github.com/Joibel/mcp-for-argo-workflows/pkg/prompts"
	"github.com/Joibel/mcp-for-argo-workflows/pkg/resources"
	"github.com/Joibel/mcp-for-argo-workflows/pkg/tools"
)

// Server wraps the MCP server and provides methods for managing tools and resources.
type Server struct {
	mcp *mcp.Server
}

// NewServer creates and initializes a new MCP server instance.
// It configures the server with the given name and version.
func NewServer(name, version string) *Server {
	implementation := &mcp.Implementation{
		Name:    name,
		Version: version,
	}

	// Create the MCP server with basic options
	// Tools capability is enabled by default when tools are added
	mcpServer := mcp.NewServer(implementation, nil)

	return &Server{
		mcp: mcpServer,
	}
}

// RegisterTools registers all Argo Workflows MCP tools with the server.
func (s *Server) RegisterTools(client argo.ClientInterface) {
	tools.RegisterAll(s.mcp, client)
}

// RegisterResources registers all Argo Workflows MCP resources with the server.
func (s *Server) RegisterResources() {
	resources.RegisterAll(s.mcp)
}

// RegisterClusterResources registers all dynamic cluster resources with the server.
// These resources query the Argo cluster at runtime.
func (s *Server) RegisterClusterResources(client argo.ClientInterface) {
	resources.RegisterClusterResources(s.mcp, client)
}

// RegisterPrompts registers all Argo Workflows MCP prompts with the server.
func (s *Server) RegisterPrompts(client argo.ClientInterface) {
	prompts.RegisterAll(s.mcp, client)
}

// GetMCPServer returns the underlying MCP server instance.
// This is useful for transport setup and starting the server.
func (s *Server) GetMCPServer() *mcp.Server {
	return s.mcp
}
