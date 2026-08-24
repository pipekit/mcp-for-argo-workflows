package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startStreamableHTTP starts the streamable HTTP transport on a free address
// and returns that address. The server is shut down when the test ends.
func startStreamableHTTP(t *testing.T) string {
	t.Helper()

	srv := NewServer("test-server", "1.0.0")
	addr := getAvailableAddr(t)

	ctx, cancel := context.WithCancel(t.Context())
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.RunStreamableHTTP(ctx, addr)
	}()

	t.Cleanup(func() {
		cancel()
		select {
		case err := <-errChan:
			assert.NoError(t, err, "server should shut down without error")
		case <-time.After(2 * time.Second):
			t.Error("server did not shut down within timeout")
		}
	})

	waitForServer(t, addr, 2*time.Second)
	return addr
}

// TestRunStreamableHTTP_Health tests that the liveness endpoint answers 200.
// Probes need a target that is 200 when the process is healthy, which the MCP
// endpoint itself is not.
func TestRunStreamableHTTP_Health(t *testing.T) {
	addr := startStreamableHTTP(t)

	client := &http.Client{Timeout: time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s%s", addr, HealthPath))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestRunStreamableHTTP_UnknownPath tests that only the MCP and health
// endpoints are served, so a proxy publishing one path publishes nothing else.
//
// "/mcp/" is in the list deliberately: MCPPath is registered as an exact
// pattern, and ServeMux only ever redirects towards a registered subtree, never
// away from one. A client or proxy that appends a trailing slash gets a 404, so
// clients must be configured with the exact path.
func TestRunStreamableHTTP_UnknownPath(t *testing.T) {
	addr := startStreamableHTTP(t)

	client := &http.Client{Timeout: time.Second}
	for _, path := range []string{"/", "/sse", "/metrics", MCPPath + "/"} {
		t.Run(path, func(t *testing.T) {
			resp, err := client.Get(fmt.Sprintf("http://%s%s", addr, path))
			require.NoError(t, err)
			defer func() { _ = resp.Body.Close() }()

			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		})
	}
}

// TestRunStreamableHTTP_Initialize tests that a client can complete the MCP
// handshake against the MCP endpoint and is issued a session.
func TestRunStreamableHTTP_Initialize(t *testing.T) {
	addr := startStreamableHTTP(t)

	const initialize = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":` +
		`{"protocolVersion":"2025-06-18","capabilities":{},` +
		`"clientInfo":{"name":"test","version":"1.0.0"}}}`

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost,
		fmt.Sprintf("http://%s%s", addr, MCPPath), strings.NewReader(initialize))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("Mcp-Session-Id"),
		"streamable HTTP should issue a session id on initialize")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "protocolVersion")
}

// TestRunStreamableHTTP_GracefulShutdown tests that cancelling the context
// shuts the server down without error.
func TestRunStreamableHTTP_GracefulShutdown(t *testing.T) {
	srv := NewServer("test-server", "1.0.0")
	addr := getAvailableAddr(t)

	ctx, cancel := context.WithCancel(t.Context())

	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.RunStreamableHTTP(ctx, addr)
	}()

	waitForServer(t, addr, 2*time.Second)
	cancel()

	select {
	case err := <-errChan:
		require.NoError(t, err, "server should shut down gracefully")
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down within timeout")
	}
}

// TestRunStreamableHTTP_InvalidAddress tests that a bad listen address is
// reported rather than silently ignored.
//
// The port is what makes this address invalid, so the failure comes out of
// parsing and never reaches a resolver: an unparseable host would send
// net.Listen to DNS, and a slow lookup would outlive the context, leaving the
// server to shut down cleanly and return nil.
func TestRunStreamableHTTP_InvalidAddress(t *testing.T) {
	srv := NewServer("test-server", "1.0.0")

	ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
	defer cancel()

	err := srv.RunStreamableHTTP(ctx, "127.0.0.1:99999")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid port")
}
