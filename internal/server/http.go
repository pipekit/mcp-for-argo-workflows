package server

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// MCPPath is the endpoint the streamable HTTP transport serves. Mounting on
	// one exact path (rather than on the root) lets a reverse proxy publish the
	// MCP endpoint without also publishing everything else the process serves.
	MCPPath = "/mcp"
	// HealthPath is a liveness endpoint for the streamable HTTP transport. The
	// MCP endpoint itself is not usable as a probe target: an unauthenticated
	// GET without a session is a client error, not a sign the server is down.
	HealthPath = "/healthz"
)

// streamableSessionTimeout is how long a streamable HTTP session may sit idle
// before the server closes it.
//
// The SDK never expires sessions by default, and a streamable session outlives
// the request that created it: it ends on an explicit DELETE or on this timer,
// nothing else. A client that crashes, sleeps or loses its connection therefore
// leaks a session, its JSON-RPC connection and a goroutine for the lifetime of
// the process. HTTP+SSE has no such problem, because there the session ends
// with the GET that opened it.
//
// Half an hour is well past any gap between calls in an interactive session,
// while still bounding what an abandoned client costs.
const streamableSessionTimeout = 30 * time.Minute

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
//
// The MCP specification superseded HTTP+SSE with streamable HTTP; see
// RunStreamableHTTP.
func (s *Server) RunHTTP(ctx context.Context, addr string) error {
	// Create an SSE handler that returns our MCP server for each new session
	handler := mcp.NewSSEHandler(func(_ *http.Request) *mcp.Server {
		return s.mcp
	}, nil)

	return serve(ctx, addr, handler, "http")
}

// RunStreamableHTTP runs the MCP server with the streamable HTTP transport.
// It handles graceful shutdown on SIGINT and SIGTERM signals.
//
// Unlike HTTP+SSE, streamable HTTP needs no standing server-to-client channel:
// each POST carries its own response stream. That matters behind a reverse
// proxy which closes idle streams - dropping the optional GET stream leaves the
// session usable, whereas under HTTP+SSE it strands every later response.
func (s *Server) RunStreamableHTTP(ctx context.Context, addr string) error {
	handler := mcp.NewStreamableHTTPHandler(func(_ *http.Request) *mcp.Server {
		return s.mcp
	}, &mcp.StreamableHTTPOptions{SessionTimeout: streamableSessionTimeout})

	mux := http.NewServeMux()
	mux.Handle(MCPPath, handler)
	mux.HandleFunc(HealthPath, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(w, "ok"); err != nil {
			slog.Debug("failed to write health response", "error", err)
		}
	})

	return serve(ctx, addr, mux, "streamable-http")
}

// serve runs handler on addr until ctx is cancelled or the listener fails,
// then shuts the server down gracefully. transport names the MCP transport for
// log messages.
func serve(ctx context.Context, addr string, handler http.Handler, transport string) error {
	httpServer := newHTTPServer(ctx, addr, handler)

	// Create a context that cancels on SIGINT or SIGTERM
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("starting MCP server", "transport", transport, "addr", addr)

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
		slog.Info("shutting down HTTP server", "transport", transport)
		//nolint:contextcheck // Use fresh context for graceful shutdown after cancellation
		if err := httpServer.Shutdown(context.Background()); err != nil {
			return err
		}
	case err := <-errChan:
		return err
	}

	slog.Info("MCP server shutdown gracefully", "transport", transport)
	return nil
}
