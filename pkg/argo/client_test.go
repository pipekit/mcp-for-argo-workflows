package argo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_NilConfig(t *testing.T) {
	client, err := NewClient(t.Context(), nil)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "config cannot be nil")
}

func TestNewClient_InvalidKubeconfig(t *testing.T) {
	// Test that an invalid kubeconfig path returns an error instead of panicking
	config := &Config{
		Kubeconfig: "/nonexistent/path/to/kubeconfig",
		Namespace:  "default",
	}

	client, err := NewClient(t.Context(), config)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to create Argo API client")
}

func TestClient_DefaultNamespace(t *testing.T) {
	// Test the DefaultNamespace method by creating a client struct directly
	// (bypassing NewClient which requires a real connection)
	client := &Client{
		config: &Config{
			Namespace: "test-namespace",
		},
	}

	assert.Equal(t, "test-namespace", client.DefaultNamespace())
}

func TestClient_IsArgoServerMode(t *testing.T) {
	tests := []struct {
		name       string
		argoServer string
		expected   bool
	}{
		{
			name:       "with argo server",
			argoServer: "localhost:2746",
			expected:   true,
		},
		{
			name:       "without argo server",
			argoServer: "",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{
				config: &Config{
					ArgoServer: tt.argoServer,
				},
			}
			assert.Equal(t, tt.expected, client.IsArgoServerMode())
		})
	}
}

func TestClient_Context(t *testing.T) {
	// Test the Context method by creating a client struct directly
	// (bypassing NewClient which requires a real connection)
	testCtx := context.WithValue(context.Background(), testCtxKey{}, "test-value")

	client := &Client{
		config: &Config{
			Namespace: "test-namespace",
		},
		ctx: testCtx,
	}

	// Verify we get back the same context
	assert.Equal(t, testCtx, client.Context())

	// Verify we can retrieve the value from the context
	val := client.Context().Value(testCtxKey{})
	assert.Equal(t, "test-value", val)
}

// testCtxKey is a custom type for context keys to avoid collisions.
type testCtxKey struct{}

func TestClient_Context_Nil(t *testing.T) {
	// Test that Context returns nil if ctx field is nil
	client := &Client{
		config: &Config{
			Namespace: "test-namespace",
		},
		ctx: nil,
	}

	assert.Nil(t, client.Context())
}

func TestClient_ArchivedWorkflowService_NotArgoServerMode(t *testing.T) {
	// Test that ArchivedWorkflowService returns error when not in Argo Server mode
	client := &Client{
		config: &Config{
			ArgoServer: "", // Not in Argo Server mode
			Namespace:  "default",
		},
	}

	svc, err := client.ArchivedWorkflowService()
	require.Error(t, err)
	assert.Nil(t, svc)
	assert.ErrorIs(t, err, ErrArchivedWorkflowsNotSupported)
}

func TestErrArchivedWorkflowsNotSupported(t *testing.T) {
	// Test that the error message is as expected
	assert.Contains(t, ErrArchivedWorkflowsNotSupported.Error(), "archived workflows are only supported")
}
