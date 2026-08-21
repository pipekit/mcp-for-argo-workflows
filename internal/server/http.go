package server

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// newHTTPServer returns the HTTP server an MCP transport listens with.
//
// Tool calls run on a context derived from the incoming HTTP request, whereas
// the Argo client's metadata (its logger, the Kubernetes client) lives on the
// context built at startup. Under stdio those are the same context, so the gap
// never showed; under an HTTP transport the request context has none of that
// metadata, and the Argo REST client panics on the missing logger rather than
// returning an error. BaseContext makes every request inherit it.
//
// The base keeps the startup context's values but drops its cancellation, so
// in-flight requests are still ended by Shutdown rather than by the signal that
// triggers it.
func newHTTPServer(base context.Context, addr string, handler http.Handler) *http.Server {
	baseCtx := context.WithoutCancel(base)

	return &http.Server{
		Addr:    addr,
		Handler: handler,
		// Timeouts prevent Slowloris attacks
		ReadHeaderTimeout: 10 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return baseCtx },
	}
}

// RunHTTP runs the MCP server with HTTP/SSE transport.
// It handles graceful shutdown on SIGINT and SIGTERM signals.
func (s *Server) RunHTTP(ctx context.Context, addr string) error {
	// Create an SSE handler that returns our MCP server for each new session
	handler := mcp.NewSSEHandler(func(_ *http.Request) *mcp.Server {
		return s.mcp
	}, nil)

	httpServer := newHTTPServer(ctx, addr, handler)

	// Create a context that cancels on SIGINT or SIGTERM
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("starting MCP server", "transport", "http", "addr", addr)

	// Start HTTP server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
		close(errChan)
	}()

	// Wait for shutdown signal or error
	select {
	case <-ctx.Done():
		slog.Info("shutting down HTTP server")
		//nolint:contextcheck // Use fresh context for graceful shutdown after cancellation
		if err := httpServer.Shutdown(context.Background()); err != nil {
			return err
		}
	case err := <-errChan:
		return err
	}

	slog.Info("MCP server shutdown gracefully", "transport", "http")
	return nil
}
