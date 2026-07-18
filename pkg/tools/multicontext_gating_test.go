package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pipekit/mcp-for-argo-workflows/pkg/argo"
	"github.com/pipekit/mcp-for-argo-workflows/pkg/argo/mocks"
)

// newTestSession registers all tools against the given client and returns a
// connected in-memory MCP client session.
func newTestSession(t *testing.T, client argo.ClientInterface, readOnly bool) *mcp.ClientSession {
	t.Helper()

	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.0"}, nil)
	RegisterAll(server, client, readOnly)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	_, err := server.Connect(t.Context(), serverTransport, nil)
	require.NoError(t, err)

	session, err := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0.0.0"}, nil).
		Connect(t.Context(), clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// listedSchemas returns each listed tool's input schema properties, keyed by
// tool name.
func listedSchemas(t *testing.T, session *mcp.ClientSession) map[string]map[string]any {
	t.Helper()

	result, err := session.ListTools(t.Context(), &mcp.ListToolsParams{})
	require.NoError(t, err)
	require.NotEmpty(t, result.Tools)

	schemas := make(map[string]map[string]any, len(result.Tools))
	for _, tool := range result.Tools {
		raw, marshalErr := json.Marshal(tool.InputSchema)
		require.NoError(t, marshalErr)
		var schema struct {
			Properties map[string]any `json:"properties"`
		}
		require.NoError(t, json.Unmarshal(raw, &schema))
		schemas[tool.Name] = schema.Properties
	}
	return schemas
}

// TestMultiContextDisabled_FailsClosed sweeps every registered tool when the
// client does not support per-call context selection: the context property
// must be absent from every schema, list_contexts must not exist, and a call
// passing a context must be rejected. Iterating the live registry means new
// tools are covered automatically.
func TestMultiContextDisabled_FailsClosed(t *testing.T) {
	mockClient := mocks.NewMockClient("argo", false)
	session := newTestSession(t, mockClient, false)

	schemas := listedSchemas(t, session)
	assert.NotContains(t, schemas, ListContextsToolName)
	for name, properties := range schemas {
		assert.NotContains(t, properties, "context", "tool %s must not advertise a context parameter", name)
	}

	// A forced context argument is rejected by schema validation before any
	// handler runs (the inferred schemas reject unknown properties).
	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "list_workflows",
		Arguments: map[string]any{"context": "anything"},
	})
	if err == nil {
		require.NotNil(t, result)
		assert.True(t, result.IsError, "call passing a context must fail when multi-context is disabled")
	}
}

// TestMultiContextEnabled_AdvertisesContext checks the flip side: cluster-facing
// tools advertise the context parameter and list_contexts is registered.
func TestMultiContextEnabled_AdvertisesContext(t *testing.T) {
	mockClient := mocks.NewMockClient("argo", false)
	mockClient.SetMultiContextEnabled(true)
	mockClient.On("ListKubeContexts").Return([]string{"alpha", "beta"}, "alpha", nil)
	session := newTestSession(t, mockClient, false)

	schemas := listedSchemas(t, session)
	require.Contains(t, schemas, ListContextsToolName)

	// Tools that never touch the cluster take no context parameter.
	localOnly := map[string]bool{
		"convert_workflow":      true,
		"render_manifest_graph": true,
		ListContextsToolName:    true,
	}
	for name, properties := range schemas {
		if localOnly[name] {
			assert.NotContains(t, properties, "context", "local tool %s must not take a context parameter", name)
			continue
		}
		assert.Contains(t, properties, "context", "cluster-facing tool %s must advertise the context parameter", name)
	}

	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: ListContextsToolName})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError)

	raw, err := json.Marshal(result.StructuredContent)
	require.NoError(t, err)
	var output ListContextsOutput
	require.NoError(t, json.Unmarshal(raw, &output))
	assert.Equal(t, []string{"alpha", "beta"}, output.Contexts)
	assert.Equal(t, "alpha", output.Default)
}

func TestResolveClient(t *testing.T) {
	t.Run("empty context passes through", func(t *testing.T) {
		mockClient := mocks.NewMockClient("argo", false)
		ctx := t.Context()
		resolvedCtx, resolvedClient, err := ResolveClient(ctx, mockClient, "  ")
		require.NoError(t, err)
		assert.Equal(t, ctx, resolvedCtx)
		assert.Same(t, mockClient, resolvedClient)
	})

	t.Run("error passes through", func(t *testing.T) {
		mockClient := mocks.NewMockClient("argo", false)
		mockClient.On("ForKubeContext", "beta").Return(nil, argo.ErrMultiContextUnavailable)
		_, _, err := ResolveClient(t.Context(), mockClient, "beta")
		require.ErrorIs(t, err, argo.ErrMultiContextUnavailable)
	})

	t.Run("resolved client and merged context are returned", func(t *testing.T) {
		type authKey struct{}
		resolvedClient := mocks.NewMockClient("argo", false)
		resolvedClient.SetContext(context.WithValue(context.Background(), authKey{}, "beta-cluster"))

		mockClient := mocks.NewMockClient("argo", false)
		mockClient.On("ForKubeContext", "beta").Return(resolvedClient, nil)

		ctx, client, err := ResolveClient(t.Context(), mockClient, "beta")
		require.NoError(t, err)
		assert.Same(t, resolvedClient, client)
		// The merged context must carry the resolved client's auth values.
		assert.Equal(t, "beta-cluster", ctx.Value(authKey{}))
	})

	t.Run("receiver returned for default context", func(t *testing.T) {
		mockClient := mocks.NewMockClient("argo", false)
		mockClient.On("ForKubeContext", "alpha").Return(mockClient, nil)
		ctx := t.Context()
		resolvedCtx, resolvedClient, err := ResolveClient(ctx, mockClient, "alpha")
		require.NoError(t, err)
		assert.Equal(t, ctx, resolvedCtx)
		assert.Same(t, mockClient, resolvedClient)
	})
}
