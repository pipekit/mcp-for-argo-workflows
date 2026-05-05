package server

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

// TestNewServer tests that NewServer creates a valid server instance.
func TestNewServer(t *testing.T) {
	tests := []struct {
		name    string
		srvName string
		version string
	}{
		{
			name:    "basic server creation",
			srvName: "test-server",
			version: "1.0.0",
		},
		{
			name:    "server with different name",
			srvName: "mcp-argo-server",
			version: "2.5.0",
		},
		{
			name:    "server with empty version",
			srvName: "test",
			version: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(tt.srvName, tt.version)
			require.NotNil(t, srv)
			require.NotNil(t, srv.mcp)
		})
	}
}

// TestGetMCPServer tests that GetMCPServer returns the underlying MCP server.
func TestGetMCPServer(t *testing.T) {
	srv := NewServer("test-server", "1.0.0")
	require.NotNil(t, srv)

	mcpServer := srv.GetMCPServer()
	require.NotNil(t, mcpServer)

	// Verify it's the same instance
	assert.Same(t, srv.mcp, mcpServer)
}

// TestRegisterTools tests that RegisterTools doesn't panic with a mock client.
func TestRegisterTools(t *testing.T) {
	srv := NewServer("test-server", "1.0.0")
	require.NotNil(t, srv)

	mockClient := mocks.NewMockClient("default", false)

	// RegisterTools should not panic
	assert.NotPanics(t, func() {
		srv.RegisterTools(mockClient)
	})
}

// TestRegisterResources tests that RegisterResources doesn't panic.
func TestRegisterResources(t *testing.T) {
	srv := NewServer("test-server", "1.0.0")
	require.NotNil(t, srv)

	// RegisterResources should not panic
	assert.NotPanics(t, func() {
		srv.RegisterResources()
	})
}

// TestRegisterClusterResources tests that RegisterClusterResources doesn't panic.
func TestRegisterClusterResources(t *testing.T) {
	srv := NewServer("test-server", "1.0.0")
	require.NotNil(t, srv)

	mockClient := mocks.NewMockClient("default", true)

	// RegisterClusterResources should not panic
	assert.NotPanics(t, func() {
		srv.RegisterClusterResources(mockClient)
	})
}

// TestRegisterPrompts tests that RegisterPrompts doesn't panic.
func TestRegisterPrompts(t *testing.T) {
	srv := NewServer("test-server", "1.0.0")
	require.NotNil(t, srv)

	mockClient := mocks.NewMockClient("default", false)

	// RegisterPrompts should not panic
	assert.NotPanics(t, func() {
		srv.RegisterPrompts(mockClient)
	})
}

// TestServerRegistrationOrder tests that registration order doesn't matter.
func TestServerRegistrationOrder(t *testing.T) {
	mockClient := mocks.NewMockClient("default", true)

	// Order 1: Tools, Resources, ClusterResources, Prompts
	srv1 := NewServer("test-server-1", "1.0.0")
	assert.NotPanics(t, func() {
		srv1.RegisterTools(mockClient)
		srv1.RegisterResources()
		srv1.RegisterClusterResources(mockClient)
		srv1.RegisterPrompts(mockClient)
	})

	// Order 2: Prompts, ClusterResources, Resources, Tools
	srv2 := NewServer("test-server-2", "1.0.0")
	assert.NotPanics(t, func() {
		srv2.RegisterPrompts(mockClient)
		srv2.RegisterClusterResources(mockClient)
		srv2.RegisterResources()
		srv2.RegisterTools(mockClient)
	})
}

// TestMultipleRegistrations tests that calling registration functions multiple times works.
func TestMultipleRegistrations(t *testing.T) {
	srv := NewServer("test-server", "1.0.0")
	require.NotNil(t, srv)

	mockClient := mocks.NewMockClient("default", false)

	// Multiple registrations should not panic
	// Note: This may result in duplicate registrations, but should not cause errors
	assert.NotPanics(t, func() {
		srv.RegisterTools(mockClient)
		srv.RegisterTools(mockClient)
		srv.RegisterResources()
		srv.RegisterResources()
	})
}

// TestServerWithDifferentClientModes tests server with different client configurations.
func TestServerWithDifferentClientModes(t *testing.T) {
	tests := []struct {
		name           string
		namespace      string
		argoServerMode bool
	}{
		{
			name:           "kubernetes mode with default namespace",
			namespace:      "default",
			argoServerMode: false,
		},
		{
			name:           "argo server mode with custom namespace",
			namespace:      "argo",
			argoServerMode: true,
		},
		{
			name:           "kubernetes mode with empty namespace",
			namespace:      "",
			argoServerMode: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer("test-server", "1.0.0")
			require.NotNil(t, srv)

			mockClient := mocks.NewMockClient(tt.namespace, tt.argoServerMode)

			// All registration functions should work regardless of client mode
			assert.NotPanics(t, func() {
				srv.RegisterTools(mockClient)
				srv.RegisterResources()
				srv.RegisterClusterResources(mockClient)
				srv.RegisterPrompts(mockClient)
			})
		})
	}
}

// TestRunStdio_ContextCancellation tests that RunStdio exits when context is cancelled.
func TestRunStdio_ContextCancellation(t *testing.T) {
	srv := NewServer("test-server", "1.0.0")
	require.NotNil(t, srv)

	// Save original stdin/stdout
	origStdin := os.Stdin
	origStdout := os.Stdout
	defer func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
	}()

	// Create pipes for stdin/stdout
	stdinReader, stdinWriter, err := os.Pipe()
	require.NoError(t, err)
	defer stdinReader.Close()
	// Note: stdinWriter is closed explicitly below to trigger shutdown

	stdoutReader, stdoutWriter, err := os.Pipe()
	require.NoError(t, err)
	defer stdoutReader.Close()
	defer stdoutWriter.Close()

	os.Stdin = stdinReader
	os.Stdout = stdoutWriter

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Run server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.RunStdio(ctx)
	}()

	// Give the server a moment to start
	time.Sleep(50 * time.Millisecond)

	// Cancel the context
	cancel()

	// Close stdin to help trigger shutdown
	stdinWriter.Close()

	// Server should exit
	select {
	case err := <-errChan:
		// The error could be nil (clean shutdown), context.Canceled, or io.EOF
		// All are acceptable
		if err != nil {
			assert.True(t, errors.Is(err, context.Canceled) || errors.Is(err, io.EOF),
				"expected context.Canceled or io.EOF, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down within timeout")
	}
}

// TestRunStdio_ImmediatelyCancelledContext tests RunStdio with an already cancelled context.
func TestRunStdio_ImmediatelyCancelledContext(t *testing.T) {
	srv := NewServer("test-server", "1.0.0")
	require.NotNil(t, srv)

	// Save original stdin/stdout
	origStdin := os.Stdin
	origStdout := os.Stdout
	defer func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
	}()

	// Create pipes for stdin/stdout
	stdinReader, stdinWriter, err := os.Pipe()
	require.NoError(t, err)
	defer stdinReader.Close()
	// Note: stdinWriter is closed explicitly below to trigger shutdown

	stdoutReader, stdoutWriter, err := os.Pipe()
	require.NoError(t, err)
	defer stdoutReader.Close()
	defer stdoutWriter.Close()

	os.Stdin = stdinReader
	os.Stdout = stdoutWriter

	// Create an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Run in goroutine with timeout
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.RunStdio(ctx)
	}()

	// Close stdin to help with shutdown
	stdinWriter.Close()

	// Should exit relatively quickly
	select {
	case err := <-errChan:
		// Either no error (clean) or context.Canceled or io.EOF is acceptable
		if err != nil {
			assert.True(t, errors.Is(err, context.Canceled) || errors.Is(err, io.EOF),
				"expected context.Canceled or io.EOF, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down within timeout")
	}
}
