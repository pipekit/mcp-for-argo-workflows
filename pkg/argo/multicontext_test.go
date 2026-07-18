package argo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type kubeContextKey struct{}

const testKubeconfig = `apiVersion: v1
kind: Config
current-context: alpha
clusters:
- name: alpha-cluster
  cluster:
    server: https://alpha.example.com
- name: beta-cluster
  cluster:
    server: https://beta.example.com
- name: gamma-cluster
  cluster:
    server: https://gamma.example.com
contexts:
- name: alpha
  context:
    cluster: alpha-cluster
    user: alpha-user
- name: beta
  context:
    cluster: beta-cluster
    user: beta-user
- name: gamma
  context:
    cluster: gamma-cluster
    user: gamma-user
users:
- name: alpha-user
  user: {}
- name: beta-user
  user: {}
- name: gamma-user
  user: {}
`

func writeTestKubeconfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kubeconfig")
	require.NoError(t, os.WriteFile(path, []byte(testKubeconfig), 0o600))
	return path
}

// newTestMultiContextClient builds a MultiContextClient over the fixture
// kubeconfig with a fake client factory, so no cluster is contacted.
func newTestMultiContextClient(t *testing.T, config *Config, allowedContexts []string, factoryCalls *atomic.Int64, factoryErr error) (*MultiContextClient, error) {
	t.Helper()
	base := &Client{
		config: config,
		ctx:    context.WithValue(context.Background(), kubeContextKey{}, config.Context),
	}
	factory := func(_ context.Context, cfg *Config) (*Client, error) {
		factoryCalls.Add(1)
		if factoryErr != nil {
			return nil, factoryErr
		}
		return &Client{
			config: cfg,
			ctx:    context.WithValue(context.Background(), kubeContextKey{}, cfg.Context),
		}, nil
	}
	return newMultiContextClient(context.Background(), base, config, allowedContexts, factory)
}

func TestNewMultiContextClient_DefaultOutsideAllowlist(t *testing.T) {
	config := &Config{Kubeconfig: writeTestKubeconfig(t), Context: "alpha"}
	var calls atomic.Int64

	client, err := newTestMultiContextClient(t, config, []string{"beta"}, &calls, nil)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), `default context "alpha"`)
}

func TestNewMultiContextClient_ImplicitCurrentContextOutsideAllowlist(t *testing.T) {
	// No explicit --context: the kubeconfig's current-context (alpha) is the default.
	config := &Config{Kubeconfig: writeTestKubeconfig(t)}
	var calls atomic.Int64

	_, err := newTestMultiContextClient(t, config, []string{"beta"}, &calls, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `default context "alpha"`)
}

func TestMultiContextClient_ForKubeContext(t *testing.T) {
	config := &Config{Kubeconfig: writeTestKubeconfig(t), Context: "alpha", Namespace: "argo"}
	var calls atomic.Int64

	client, err := newTestMultiContextClient(t, config, nil, &calls, nil)
	require.NoError(t, err)

	// Empty and default names return the receiver without building anything.
	for _, name := range []string{"", "  ", "alpha", " alpha "} {
		resolved, resolveErr := client.ForKubeContext(name)
		require.NoError(t, resolveErr)
		assert.Same(t, client, resolved, "name %q should return the receiver", name)
	}
	assert.Zero(t, calls.Load())

	// A different context builds once and is cached afterwards.
	beta1, err := client.ForKubeContext("beta")
	require.NoError(t, err)
	beta2, err := client.ForKubeContext(" beta ")
	require.NoError(t, err)
	assert.Same(t, beta1, beta2)
	assert.Equal(t, int64(1), calls.Load())

	// The resolved client carries its own context values and namespace config.
	assert.Equal(t, "beta", beta1.Context().Value(kubeContextKey{}))
	assert.Equal(t, "argo", beta1.DefaultNamespace())

	// Unknown contexts never reach the factory.
	_, err = client.ForKubeContext("nope")
	require.Error(t, err)
	assert.Equal(t, int64(1), calls.Load())
}

func TestMultiContextClient_UnknownAndDisallowedErrorsAreIdentical(t *testing.T) {
	config := &Config{Kubeconfig: writeTestKubeconfig(t), Context: "alpha"}
	var calls atomic.Int64

	client, err := newTestMultiContextClient(t, config, []string{"alpha", "beta"}, &calls, nil)
	require.NoError(t, err)

	_, unknownErr := client.ForKubeContext("does-not-exist")
	require.Error(t, unknownErr)
	_, disallowedErr := client.ForKubeContext("gamma")
	require.Error(t, disallowedErr)

	// Identical wording apart from the name, so the allowlist does not reveal
	// which hidden contexts exist.
	assert.Equal(t,
		fmt.Sprintf("context %q is not available (use list_contexts to see available contexts)", "does-not-exist"),
		unknownErr.Error(),
	)
	assert.Equal(t,
		fmt.Sprintf("context %q is not available (use list_contexts to see available contexts)", "gamma"),
		disallowedErr.Error(),
	)
	assert.Zero(t, calls.Load())
}

func TestMultiContextClient_FailedBuildsAreNotCached(t *testing.T) {
	config := &Config{Kubeconfig: writeTestKubeconfig(t), Context: "alpha"}
	var calls atomic.Int64

	client, err := newTestMultiContextClient(t, config, nil, &calls, errors.New("boom"))
	require.NoError(t, err)

	_, err = client.ForKubeContext("beta")
	require.Error(t, err)
	_, err = client.ForKubeContext("beta")
	require.Error(t, err)
	assert.Equal(t, int64(2), calls.Load(), "failed builds must be retried, not cached")
}

func TestMultiContextClient_ConcurrentForKubeContext(t *testing.T) {
	config := &Config{Kubeconfig: writeTestKubeconfig(t), Context: "alpha"}
	var calls atomic.Int64

	client, err := newTestMultiContextClient(t, config, nil, &calls, nil)
	require.NoError(t, err)

	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			_, resolveErr := client.ForKubeContext("beta")
			assert.NoError(t, resolveErr)
		})
	}
	wg.Wait()
	assert.Equal(t, int64(1), calls.Load(), "concurrent first calls must build exactly one client")
}

func TestMultiContextClient_ListKubeContexts(t *testing.T) {
	config := &Config{Kubeconfig: writeTestKubeconfig(t), Context: "alpha"}
	var calls atomic.Int64

	t.Run("all contexts sorted", func(t *testing.T) {
		client, err := newTestMultiContextClient(t, config, nil, &calls, nil)
		require.NoError(t, err)

		names, defaultContext, err := client.ListKubeContexts()
		require.NoError(t, err)
		assert.Equal(t, []string{"alpha", "beta", "gamma"}, names)
		assert.Equal(t, "alpha", defaultContext)
	})

	t.Run("filtered by allowlist", func(t *testing.T) {
		client, err := newTestMultiContextClient(t, config, []string{" beta ", "alpha", ""}, &calls, nil)
		require.NoError(t, err)

		names, defaultContext, err := client.ListKubeContexts()
		require.NoError(t, err)
		assert.Equal(t, []string{"alpha", "beta"}, names)
		assert.Equal(t, "alpha", defaultContext)
	})
}

func TestClient_ForKubeContext_Unavailable(t *testing.T) {
	client := &Client{config: &Config{}, ctx: context.Background()}

	resolved, err := client.ForKubeContext("")
	require.NoError(t, err)
	assert.Same(t, client, resolved)

	_, err = client.ForKubeContext("anything")
	require.ErrorIs(t, err, ErrMultiContextUnavailable)

	_, _, err = client.ListKubeContexts()
	require.ErrorIs(t, err, ErrMultiContextUnavailable)

	assert.False(t, client.MultiContextEnabled())
}

func TestMergeContext(t *testing.T) {
	type requestKey struct{}
	type valuesKey struct{}
	type sharedKey struct{}

	requestCtx, cancel := context.WithCancel(context.Background())
	requestCtx = context.WithValue(requestCtx, requestKey{}, "request")
	requestCtx = context.WithValue(requestCtx, sharedKey{}, "from-request")
	valuesCtx := context.WithValue(context.Background(), valuesKey{}, "values")
	valuesCtx = context.WithValue(valuesCtx, sharedKey{}, "from-values")

	merged := MergeContext(requestCtx, valuesCtx)

	// Values context wins, request context is the fallback.
	assert.Equal(t, "from-values", merged.Value(sharedKey{}))
	assert.Equal(t, "values", merged.Value(valuesKey{}))
	assert.Equal(t, "request", merged.Value(requestKey{}))

	// Cancellation comes from the request context.
	require.NoError(t, merged.Err())
	cancel()
	require.Error(t, merged.Err())
	select {
	case <-merged.Done():
	default:
		t.Fatal("merged context must observe request cancellation")
	}
}
