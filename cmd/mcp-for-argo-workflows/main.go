// Package main is the entry point for the MCP server for Argo Workflows.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/pipekit/mcp-for-argo-workflows/internal/config"
	"github.com/pipekit/mcp-for-argo-workflows/internal/server"
	"github.com/pipekit/mcp-for-argo-workflows/internal/version"
	"github.com/pipekit/mcp-for-argo-workflows/pkg/argo"
)

const serverName = "mcp-for-argo-workflows"

func main() {
	// Configure structured logging to stderr
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	// Create root context with signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	err := run(ctx)
	cancel() // Ensure signal handler is stopped before exit

	if err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	// Parse configuration from CLI flags and environment variables
	cfg, err := config.NewFromFlags()
	if err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Validate configuration
	if validateErr := cfg.Validate(); validateErr != nil {
		return fmt.Errorf("invalid configuration: %w", validateErr)
	}

	// Create the Argo Workflows client with the root context. In multi-context
	// mode the client can additionally serve other kubeconfig contexts per call.
	var argoClient argo.ClientInterface
	if cfg.MultiContextEnabled() {
		multiClient, clientErr := argo.NewMultiContextClient(ctx, cfg.ToArgoConfig(), cfg.AllowedContexts)
		if clientErr != nil {
			return fmt.Errorf("failed to create Argo client: %w", clientErr)
		}
		argoClient = multiClient
	} else {
		client, clientErr := argo.NewClient(ctx, cfg.ToArgoConfig())
		if clientErr != nil {
			return fmt.Errorf("failed to create Argo client: %w", clientErr)
		}
		argoClient = client
	}

	// Use the client's context which contains K8s auth metadata for all subsequent operations.
	// The Argo SDK embeds the K8s client in this context, which is required for authorization checks.
	//nolint:contextcheck // Intentionally replacing context with Argo SDK's context containing K8s client
	ctx = argoClient.Context()

	// Create the MCP server with name and version
	srv := server.NewServer(serverName, version.Version)

	// Register Argo Workflows tools
	srv.RegisterTools(argoClient, cfg.ReadOnly)

	// Register Argo CRD schema resources
	srv.RegisterResources()

	// Register dynamic cluster resources (requires Argo client)
	srv.RegisterClusterResources(argoClient)

	// Register MCP prompts
	srv.RegisterPrompts(argoClient)

	slog.Info("MCP server created",
		"name", serverName,
		"version", version.Version,
		"transport", cfg.Transport,
		"namespace", cfg.Namespace,
		"read_only", cfg.ReadOnly,
		"multi_context", cfg.MultiContextEnabled(),
	)

	if cfg.ReadOnly {
		slog.Info("read-only mode enabled: mutating tools are disabled")
	}

	// Log the reachable context set so operators can see exactly which
	// clusters tool calls may act on.
	if cfg.MultiContextEnabled() {
		if names, defaultContext, listErr := argoClient.ListKubeContexts(); listErr == nil {
			slog.Info("multi-context enabled: kubeconfig contexts are selectable per tool call",
				"contexts", names,
				"default", defaultContext,
			)
		}
	}

	// Start the server with the configured transport.
	// The ctx here is the Argo SDK context (see nolint:contextcheck above).
	if cfg.IsHTTPTransport() {
		slog.Info("starting HTTP transport", "addr", cfg.HTTPAddr)
		return srv.RunHTTP(ctx, cfg.HTTPAddr) //nolint:contextcheck // ctx is Argo SDK context with K8s client
	}

	// Default to stdio transport
	return srv.RunStdio(ctx) //nolint:contextcheck // ctx is Argo SDK context with K8s client
}
