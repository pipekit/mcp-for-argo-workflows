//go:build e2e

package e2e

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/tools"
)

// TestGetWorkflowNode tests retrieving details of a specific node within a workflow.
func TestGetWorkflowNode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Use the client's context which contains the KubeClient
	clientCtx := cluster.ArgoClient.Context()

	// Submit a DAG workflow with multiple nodes
	t.Log("Submitting DAG workflow...")
	manifest := LoadTestDataFile(t, "dag-workflow.yaml")

	submitHandler := tools.SubmitWorkflowHandler(cluster.ArgoClient)
	submitInput := tools.SubmitWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Manifest:  manifest,
	}

	_, submitOutput, err := submitHandler(clientCtx, nil, submitInput)
	require.NoError(t, err, "Failed to submit workflow")
	require.NotNil(t, submitOutput)

	workflowName := submitOutput.Name
	t.Logf("Submitted workflow: %s", workflowName)

	// Cleanup at the end
	defer func() {
		deleteHandler := tools.DeleteWorkflowHandler(cluster.ArgoClient)
		deleteInput := tools.DeleteWorkflowInput{
			Namespace: cluster.ArgoNamespace,
			Name:      workflowName,
		}
		_, _, _ = deleteHandler(clientCtx, nil, deleteInput)
	}()

	// Wait for workflow to complete
	t.Log("Waiting for workflow to complete...")
	finalPhase := cluster.WaitForWorkflowPhase(t, cluster.ArgoNamespace, workflowName,
		2*time.Minute, "Succeeded", "Failed", "Error")

	// We need a completed workflow to inspect nodes, but continue even if failed
	assert.Contains(t, []string{"Succeeded", "Failed", "Error"}, finalPhase,
		"Workflow should reach terminal state")

	// Get the node details by display name (known from dag-workflow.yaml)
	// DAG has: task-a, task-b, task-c, task-d
	t.Log("Getting workflow node details for 'task-a'...")
	nodeHandler := tools.GetWorkflowNodeHandler(cluster.ArgoClient)
	nodeInput := tools.GetWorkflowNodeInput{
		Namespace:    cluster.ArgoNamespace,
		WorkflowName: workflowName,
		NodeName:     "task-a", // Display name from the DAG workflow
	}

	_, nodeOutput, err := nodeHandler(clientCtx, nil, nodeInput)
	require.NoError(t, err, "Failed to get workflow node")
	require.NotNil(t, nodeOutput)

	// Verify expected fields
	assert.Equal(t, "task-a", nodeOutput.DisplayName, "Should have correct display name")
	assert.NotEmpty(t, nodeOutput.ID, "Node should have ID")
	assert.NotEmpty(t, nodeOutput.Type, "Node should have type")
	assert.NotEmpty(t, nodeOutput.Phase, "Node should have phase")

	// For completed nodes, we should have timing info
	if nodeOutput.Phase == "Succeeded" || nodeOutput.Phase == "Failed" {
		assert.NotEmpty(t, nodeOutput.StartedAt, "Completed node should have startedAt")
		assert.NotEmpty(t, nodeOutput.FinishedAt, "Completed node should have finishedAt")
		assert.NotEmpty(t, nodeOutput.Duration, "Completed node should have duration")
	}

	t.Logf("Node details: ID=%s, Type=%s, Phase=%s, Duration=%s",
		nodeOutput.ID, nodeOutput.Type, nodeOutput.Phase, nodeOutput.Duration)
}

// TestGetWorkflowNode_ByDisplayName tests retrieving a node by its display name.
func TestGetWorkflowNode_ByDisplayName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Use the client's context which contains the KubeClient
	clientCtx := cluster.ArgoClient.Context()

	// Submit a DAG workflow with multiple nodes
	t.Log("Submitting DAG workflow...")
	manifest := LoadTestDataFile(t, "dag-workflow.yaml")

	submitHandler := tools.SubmitWorkflowHandler(cluster.ArgoClient)
	submitInput := tools.SubmitWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Manifest:  manifest,
	}

	_, submitOutput, err := submitHandler(clientCtx, nil, submitInput)
	require.NoError(t, err, "Failed to submit workflow")
	require.NotNil(t, submitOutput)

	workflowName := submitOutput.Name
	t.Logf("Submitted workflow: %s", workflowName)

	// Cleanup at the end
	defer func() {
		deleteHandler := tools.DeleteWorkflowHandler(cluster.ArgoClient)
		deleteInput := tools.DeleteWorkflowInput{
			Namespace: cluster.ArgoNamespace,
			Name:      workflowName,
		}
		_, _, _ = deleteHandler(clientCtx, nil, deleteInput)
	}()

	// Wait for workflow to complete
	t.Log("Waiting for workflow to complete...")
	finalPhase := cluster.WaitForWorkflowPhase(t, cluster.ArgoNamespace, workflowName,
		2*time.Minute, "Succeeded", "Failed", "Error")

	assert.Contains(t, []string{"Succeeded", "Failed", "Error"}, finalPhase,
		"Workflow should reach terminal state")

	// Get node by display name - use task-d (the final node that depends on task-b and task-c)
	// This tests a different node than TestGetWorkflowNode which uses task-a
	t.Log("Getting workflow node by display name 'task-d'...")
	nodeHandler := tools.GetWorkflowNodeHandler(cluster.ArgoClient)
	nodeInput := tools.GetWorkflowNodeInput{
		Namespace:    cluster.ArgoNamespace,
		WorkflowName: workflowName,
		NodeName:     "task-d", // Display name of final DAG node
	}

	_, nodeOutput, err := nodeHandler(clientCtx, nil, nodeInput)
	require.NoError(t, err, "Failed to get workflow node by display name")
	require.NotNil(t, nodeOutput)

	// Verify we got the right node
	assert.Equal(t, "task-d", nodeOutput.DisplayName, "Should find node by display name")
	assert.NotEmpty(t, nodeOutput.ID, "Node should have ID")
	t.Logf("Found node: ID=%s, DisplayName=%s, Phase=%s", nodeOutput.ID, nodeOutput.DisplayName, nodeOutput.Phase)
}

// TestGetWorkflowNode_NotFound tests error handling when a node doesn't exist.
func TestGetWorkflowNode_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Use the client's context which contains the KubeClient
	clientCtx := cluster.ArgoClient.Context()

	// Submit a simple workflow
	t.Log("Submitting hello-world workflow...")
	manifest := LoadTestDataFile(t, "hello-world.yaml")

	submitHandler := tools.SubmitWorkflowHandler(cluster.ArgoClient)
	submitInput := tools.SubmitWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Manifest:  manifest,
	}

	_, submitOutput, err := submitHandler(clientCtx, nil, submitInput)
	require.NoError(t, err, "Failed to submit workflow")
	require.NotNil(t, submitOutput)

	workflowName := submitOutput.Name
	t.Logf("Submitted workflow: %s", workflowName)

	// Cleanup at the end
	defer func() {
		deleteHandler := tools.DeleteWorkflowHandler(cluster.ArgoClient)
		deleteInput := tools.DeleteWorkflowInput{
			Namespace: cluster.ArgoNamespace,
			Name:      workflowName,
		}
		_, _, _ = deleteHandler(clientCtx, nil, deleteInput)
	}()

	// Wait for workflow to complete
	t.Log("Waiting for workflow to complete...")
	cluster.WaitForWorkflowPhase(t, cluster.ArgoNamespace, workflowName,
		2*time.Minute, "Succeeded", "Failed", "Error")

	// Try to get a non-existent node
	t.Log("Getting non-existent node...")
	nodeHandler := tools.GetWorkflowNodeHandler(cluster.ArgoClient)
	nodeInput := tools.GetWorkflowNodeInput{
		Namespace:    cluster.ArgoNamespace,
		WorkflowName: workflowName,
		NodeName:     "non-existent-node",
	}

	_, _, err = nodeHandler(clientCtx, nil, nodeInput)
	require.Error(t, err, "Should error when node doesn't exist")
	assert.Contains(t, err.Error(), "not found", "Error should mention node not found")
}

// TestRenderWorkflowGraph tests rendering a workflow as a graph in various formats.
func TestRenderWorkflowGraph(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Use the client's context which contains the KubeClient
	clientCtx := cluster.ArgoClient.Context()

	// Submit a DAG workflow with multiple nodes for interesting graph
	t.Log("Submitting DAG workflow...")
	manifest := LoadTestDataFile(t, "dag-workflow.yaml")

	submitHandler := tools.SubmitWorkflowHandler(cluster.ArgoClient)
	submitInput := tools.SubmitWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Manifest:  manifest,
	}

	_, submitOutput, err := submitHandler(clientCtx, nil, submitInput)
	require.NoError(t, err, "Failed to submit workflow")
	require.NotNil(t, submitOutput)

	workflowName := submitOutput.Name
	t.Logf("Submitted workflow: %s", workflowName)

	// Cleanup at the end
	defer func() {
		deleteHandler := tools.DeleteWorkflowHandler(cluster.ArgoClient)
		deleteInput := tools.DeleteWorkflowInput{
			Namespace: cluster.ArgoNamespace,
			Name:      workflowName,
		}
		_, _, _ = deleteHandler(clientCtx, nil, deleteInput)
	}()

	// Wait for workflow to complete
	t.Log("Waiting for workflow to complete...")
	finalPhase := cluster.WaitForWorkflowPhase(t, cluster.ArgoNamespace, workflowName,
		2*time.Minute, "Succeeded", "Failed", "Error")

	assert.Contains(t, []string{"Succeeded", "Failed", "Error"}, finalPhase,
		"Workflow should reach terminal state")

	renderHandler := tools.RenderWorkflowGraphHandler(cluster.ArgoClient)

	// Test Mermaid format
	t.Run("mermaid format", func(t *testing.T) {
		input := tools.RenderWorkflowGraphInput{
			Namespace: cluster.ArgoNamespace,
			Name:      workflowName,
			Format:    "mermaid",
		}

		_, output, err := renderHandler(clientCtx, nil, input)
		require.NoError(t, err, "Failed to render workflow graph as mermaid")
		require.NotNil(t, output)

		assert.Equal(t, "mermaid", output.Format)
		assert.Greater(t, output.NodeCount, 0, "Should have nodes")
		assert.Contains(t, output.Graph, "flowchart TD", "Should be a Mermaid flowchart")

		// Should contain class definitions for status colors
		assert.Contains(t, output.Graph, "classDef succeeded", "Should have status class definitions")

		t.Logf("Mermaid graph (%d nodes):\n%s", output.NodeCount, output.Graph)
	})

	// Test ASCII format
	t.Run("ascii format", func(t *testing.T) {
		input := tools.RenderWorkflowGraphInput{
			Namespace: cluster.ArgoNamespace,
			Name:      workflowName,
			Format:    "ascii",
		}

		_, output, err := renderHandler(clientCtx, nil, input)
		require.NoError(t, err, "Failed to render workflow graph as ASCII")
		require.NotNil(t, output)

		assert.Equal(t, "ascii", output.Format)
		assert.Greater(t, output.NodeCount, 0, "Should have nodes")
		// ASCII format should have tree characters
		assert.True(t,
			strings.Contains(output.Graph, "├") ||
				strings.Contains(output.Graph, "└") ||
				strings.Contains(output.Graph, "│"),
			"Should have ASCII tree characters")

		t.Logf("ASCII graph (%d nodes):\n%s", output.NodeCount, output.Graph)
	})

	// Test DOT format
	t.Run("dot format", func(t *testing.T) {
		input := tools.RenderWorkflowGraphInput{
			Namespace: cluster.ArgoNamespace,
			Name:      workflowName,
			Format:    "dot",
		}

		_, output, err := renderHandler(clientCtx, nil, input)
		require.NoError(t, err, "Failed to render workflow graph as DOT")
		require.NotNil(t, output)

		assert.Equal(t, "dot", output.Format)
		assert.Greater(t, output.NodeCount, 0, "Should have nodes")
		assert.Contains(t, output.Graph, "digraph workflow", "Should be a DOT digraph")
		assert.Contains(t, output.Graph, "rankdir=TB", "Should have top-to-bottom direction")

		t.Logf("DOT graph (%d nodes):\n%s", output.NodeCount, output.Graph)
	})

	// Test default format (should be mermaid)
	t.Run("default format", func(t *testing.T) {
		input := tools.RenderWorkflowGraphInput{
			Namespace: cluster.ArgoNamespace,
			Name:      workflowName,
			// No format specified
		}

		_, output, err := renderHandler(clientCtx, nil, input)
		require.NoError(t, err, "Failed to render workflow graph with default format")
		require.NotNil(t, output)

		assert.Equal(t, "mermaid", output.Format, "Default format should be mermaid")
		assert.Contains(t, output.Graph, "flowchart TD", "Default should be Mermaid")
	})
}

// TestRenderWorkflowGraph_WithoutStatus tests rendering without status colors.
func TestRenderWorkflowGraph_WithoutStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Use the client's context which contains the KubeClient
	clientCtx := cluster.ArgoClient.Context()

	// Submit a simple workflow
	t.Log("Submitting hello-world workflow...")
	manifest := LoadTestDataFile(t, "hello-world.yaml")

	submitHandler := tools.SubmitWorkflowHandler(cluster.ArgoClient)
	submitInput := tools.SubmitWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Manifest:  manifest,
	}

	_, submitOutput, err := submitHandler(clientCtx, nil, submitInput)
	require.NoError(t, err, "Failed to submit workflow")
	require.NotNil(t, submitOutput)

	workflowName := submitOutput.Name
	t.Logf("Submitted workflow: %s", workflowName)

	// Cleanup at the end
	defer func() {
		deleteHandler := tools.DeleteWorkflowHandler(cluster.ArgoClient)
		deleteInput := tools.DeleteWorkflowInput{
			Namespace: cluster.ArgoNamespace,
			Name:      workflowName,
		}
		_, _, _ = deleteHandler(clientCtx, nil, deleteInput)
	}()

	// Wait for workflow to complete
	t.Log("Waiting for workflow to complete...")
	cluster.WaitForWorkflowPhase(t, cluster.ArgoNamespace, workflowName,
		2*time.Minute, "Succeeded", "Failed", "Error")

	// Render without status
	t.Log("Rendering workflow graph without status...")
	renderHandler := tools.RenderWorkflowGraphHandler(cluster.ArgoClient)
	includeStatus := false
	input := tools.RenderWorkflowGraphInput{
		Namespace:     cluster.ArgoNamespace,
		Name:          workflowName,
		Format:        "mermaid",
		IncludeStatus: &includeStatus,
	}

	_, output, err := renderHandler(clientCtx, nil, input)
	require.NoError(t, err, "Failed to render workflow graph without status")
	require.NotNil(t, output)

	// Should NOT contain class definitions when status is disabled
	assert.NotContains(t, output.Graph, "classDef succeeded", "Should not have status class definitions")
	assert.NotContains(t, output.Graph, ":::succeeded", "Should not have node status classes")

	t.Logf("Graph without status:\n%s", output.Graph)
}

// TestRenderManifestGraph tests rendering a workflow manifest as a graph without submitting.
func TestRenderManifestGraph(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	renderHandler := tools.RenderManifestGraphHandler()

	// Test with DAG workflow
	t.Run("dag workflow", func(t *testing.T) {
		manifest := LoadTestDataFile(t, "dag-workflow.yaml")

		input := tools.RenderManifestGraphInput{
			Manifest: manifest,
			Format:   "mermaid",
		}

		_, output, err := renderHandler(context.Background(), nil, input)
		require.NoError(t, err, "Failed to render manifest graph")
		require.NotNil(t, output)

		assert.Equal(t, "mermaid", output.Format)
		assert.Equal(t, "Workflow", output.Kind)
		assert.Greater(t, output.NodeCount, 0, "Should have nodes")
		assert.Contains(t, output.Graph, "flowchart TD", "Should be a Mermaid flowchart")

		// DAG workflow should show dependencies
		// The dag-workflow.yaml has: task-a -> task-b, task-a -> task-c, task-b,task-c -> task-d
		assert.Contains(t, output.Graph, "task_a", "Should contain task-a")
		assert.Contains(t, output.Graph, "task_b", "Should contain task-b")
		assert.Contains(t, output.Graph, "task_c", "Should contain task-c")
		assert.Contains(t, output.Graph, "task_d", "Should contain task-d")

		t.Logf("DAG manifest graph (%d nodes):\n%s", output.NodeCount, output.Graph)
	})

	// Test with WorkflowTemplate
	t.Run("workflow template", func(t *testing.T) {
		manifest := LoadTestDataFile(t, "workflow-template.yaml")

		input := tools.RenderManifestGraphInput{
			Manifest: manifest,
			Format:   "mermaid",
		}

		_, output, err := renderHandler(context.Background(), nil, input)
		require.NoError(t, err, "Failed to render workflow template graph")
		require.NotNil(t, output)

		assert.Equal(t, "mermaid", output.Format)
		assert.Equal(t, "WorkflowTemplate", output.Kind)
		assert.Greater(t, output.NodeCount, 0, "Should have nodes")

		t.Logf("WorkflowTemplate manifest graph (%d nodes):\n%s", output.NodeCount, output.Graph)
	})

	// Test with CronWorkflow
	t.Run("cron workflow", func(t *testing.T) {
		manifest := LoadTestDataFile(t, "cron-workflow.yaml")

		input := tools.RenderManifestGraphInput{
			Manifest: manifest,
			Format:   "mermaid",
		}

		_, output, err := renderHandler(context.Background(), nil, input)
		require.NoError(t, err, "Failed to render cron workflow graph")
		require.NotNil(t, output)

		assert.Equal(t, "mermaid", output.Format)
		assert.Equal(t, "CronWorkflow", output.Kind)
		assert.Greater(t, output.NodeCount, 0, "Should have nodes")

		t.Logf("CronWorkflow manifest graph (%d nodes):\n%s", output.NodeCount, output.Graph)
	})

	// Test ASCII format
	t.Run("ascii format", func(t *testing.T) {
		manifest := LoadTestDataFile(t, "dag-workflow.yaml")

		input := tools.RenderManifestGraphInput{
			Manifest: manifest,
			Format:   "ascii",
		}

		_, output, err := renderHandler(context.Background(), nil, input)
		require.NoError(t, err, "Failed to render manifest graph as ASCII")
		require.NotNil(t, output)

		assert.Equal(t, "ascii", output.Format)
		assert.Greater(t, output.NodeCount, 0, "Should have nodes")

		t.Logf("ASCII manifest graph (%d nodes):\n%s", output.NodeCount, output.Graph)
	})

	// Test DOT format
	t.Run("dot format", func(t *testing.T) {
		manifest := LoadTestDataFile(t, "dag-workflow.yaml")

		input := tools.RenderManifestGraphInput{
			Manifest: manifest,
			Format:   "dot",
		}

		_, output, err := renderHandler(context.Background(), nil, input)
		require.NoError(t, err, "Failed to render manifest graph as DOT")
		require.NotNil(t, output)

		assert.Equal(t, "dot", output.Format)
		assert.Greater(t, output.NodeCount, 0, "Should have nodes")
		assert.Contains(t, output.Graph, "digraph workflow", "Should be a DOT digraph")

		t.Logf("DOT manifest graph (%d nodes):\n%s", output.NodeCount, output.Graph)
	})
}

// TestRenderManifestGraph_InvalidManifest tests error handling for invalid manifests.
func TestRenderManifestGraph_InvalidManifest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	renderHandler := tools.RenderManifestGraphHandler()

	t.Run("empty manifest", func(t *testing.T) {
		input := tools.RenderManifestGraphInput{
			Manifest: "",
			Format:   "mermaid",
		}

		_, _, err := renderHandler(context.Background(), nil, input)
		require.Error(t, err, "Should error on empty manifest")
		assert.Contains(t, err.Error(), "empty", "Error should mention empty")
	})

	t.Run("invalid yaml syntax", func(t *testing.T) {
		input := tools.RenderManifestGraphInput{
			Manifest: "{invalid yaml: [unclosed",
			Format:   "mermaid",
		}

		_, _, err := renderHandler(context.Background(), nil, input)
		require.Error(t, err, "Should error on invalid YAML syntax")
	})

	t.Run("unsupported kind", func(t *testing.T) {
		input := tools.RenderManifestGraphInput{
			Manifest: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
`,
			Format: "mermaid",
		}

		_, _, err := renderHandler(context.Background(), nil, input)
		require.Error(t, err, "Should error on unsupported kind")
		assert.Contains(t, err.Error(), "unsupported", "Error should mention unsupported")
	})
}

// TestRenderManifestGraph_ClusterWorkflowTemplate tests rendering ClusterWorkflowTemplate.
func TestRenderManifestGraph_ClusterWorkflowTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	renderHandler := tools.RenderManifestGraphHandler()

	manifest := LoadTestDataFile(t, "cluster-workflow-template.yaml")

	input := tools.RenderManifestGraphInput{
		Manifest: manifest,
		Format:   "mermaid",
	}

	_, output, err := renderHandler(context.Background(), nil, input)
	require.NoError(t, err, "Failed to render cluster workflow template graph")
	require.NotNil(t, output)

	assert.Equal(t, "mermaid", output.Format)
	assert.Equal(t, "ClusterWorkflowTemplate", output.Kind)
	assert.Greater(t, output.NodeCount, 0, "Should have nodes")

	t.Logf("ClusterWorkflowTemplate manifest graph (%d nodes):\n%s", output.NodeCount, output.Graph)
}
