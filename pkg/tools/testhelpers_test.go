package tools

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

// newMockClient creates a mock Argo client with the specified namespace and mode.
func newMockClient(t *testing.T, namespace string, argoServerMode bool) *mocks.MockClient {
	t.Helper()
	return mocks.NewMockClient(namespace, argoServerMode)
}

// newMockWorkflowService creates a new mock workflow service client.
func newMockWorkflowService(t *testing.T) *mocks.MockWorkflowServiceClient {
	t.Helper()
	m := &mocks.MockWorkflowServiceClient{}
	m.Test(t)
	return m
}

// loadTestWorkflowYAML loads the raw YAML content of a workflow fixture.
func loadTestWorkflowYAML(t *testing.T, filename string) string {
	t.Helper()

	// Get the current file's directory
	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get current file")
	testdataDir := filepath.Join(filepath.Dir(currentFile), "testdata")

	// Read the file
	path := filepath.Join(testdataDir, filename)
	data, err := os.ReadFile(path) //nolint:gosec // G304: reading test fixtures is safe
	require.NoError(t, err, "failed to read test workflow file: %s", filename)

	return string(data)
}
