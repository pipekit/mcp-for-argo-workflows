package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCtxKey struct{}

// startServer runs a server built by newHTTPServer on a free address and
// returns that address. The server is shut down when the test ends.
func startServer(base context.Context, t *testing.T, handler http.Handler) string {
	t.Helper()

	addr := getAvailableAddr(t)
	srv := newHTTPServer(base, addr, handler)

	go func() { _ = srv.ListenAndServe() }()
	//nolint:contextcheck // Fresh context: shutdown must not depend on the base being live.
	t.Cleanup(func() { _ = srv.Shutdown(context.Background()) })

	waitForServer(t, addr, 2*time.Second)
	return addr
}

// TestNewHTTPServer_RequestInheritsBaseValues tests that a request context
// carries the values of the context the server was started with. Tool calls
// need the Argo client metadata that lives there; without it the Argo REST
// client panics instead of returning an error.
func TestNewHTTPServer_RequestInheritsBaseValues(t *testing.T) {
	base := context.WithValue(t.Context(), testCtxKey{}, "from-base")

	var (
		mu  sync.Mutex
		got any
	)
	addr := startServer(base, t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		got = r.Context().Value(testCtxKey{})
	}))

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s/", addr))
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "from-base", got)
}

// TestNewHTTPServer_RequestSurvivesBaseCancellation tests that cancelling the
// context the server was started with does not cancel an in-flight request.
// Shutdown is what ends in-flight requests; the signal that triggers it must
// not cut their contexts first.
func TestNewHTTPServer_RequestSurvivesBaseCancellation(t *testing.T) {
	base, cancelBase := context.WithCancel(t.Context())

	entered := make(chan context.Context, 1)
	release := make(chan struct{})

	// Only /block holds a request open; every other path 404s, so the
	// readiness probe in startServer does not get stuck in the handler.
	mux := http.NewServeMux()
	mux.HandleFunc("/block", func(_ http.ResponseWriter, r *http.Request) {
		entered <- r.Context()
		<-release
	})

	addr := startServer(base, t, mux)

	go func() {
		client := &http.Client{Timeout: 5 * time.Second}
		if resp, err := client.Get(fmt.Sprintf("http://%s/block", addr)); err == nil {
			_ = resp.Body.Close()
		}
	}()

	var reqCtx context.Context
	select {
	case reqCtx = <-entered:
	case <-time.After(2 * time.Second):
		close(release)
		t.Fatal("handler was not reached")
	}

	cancelBase()

	select {
	case <-reqCtx.Done():
		close(release)
		t.Fatal("request context was cancelled by the base context")
	case <-time.After(200 * time.Millisecond):
	}

	close(release)
}
