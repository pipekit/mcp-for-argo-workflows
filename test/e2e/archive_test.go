//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowarchive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/tools"
)

// =============================================================================
// Phase 8: Archived Workflow Tools E2E Tests
//
// These tests require Argo Server mode (not direct Kubernetes API mode).
// Archive persistence is provided by quick-start-postgres.yaml which includes
// PostgreSQL for workflow archiving.
// =============================================================================

// submitAndArchiveWorkflow submits a workflow, waits for completion, and waits for archival.
// Returns the workflow name and UID.
func submitAndArchiveWorkflow(t *testing.T, cluster *E2ECluster, manifestFile string) (workflowName, workflowUID string) {
	t.Helper()

	clientCtx := cluster.ArgoClient.Context()

	// Load and submit workflow
	manifest := LoadTestDataFile(t, manifestFile)
	submitHandler := tools.SubmitWorkflowHandler(cluster.ArgoClient)
	submitInput := tools.SubmitWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Manifest:  manifest,
	}

	_, submitOutput, err := submitHandler(clientCtx, nil, submitInput)
	require.NoError(t, err, "Failed to submit workflow")

	workflowName = submitOutput.Name
	workflowUID = submitOutput.UID
	t.Logf("Submitted workflow: %s (UID: %s)", workflowName, workflowUID)

	// Wait for workflow to complete
	t.Log("Waiting for workflow to complete...")
	finalPhase := cluster.WaitForWorkflowPhase(t, cluster.ArgoNamespace, workflowName,
		2*time.Minute, "Succeeded", "Failed", "Error")
	t.Logf("Workflow completed with phase: %s", finalPhase)

	// Wait a bit for archival to happen (archive is async)
	t.Log("Waiting for workflow to be archived...")
	time.Sleep(5 * time.Second)

	// Verify workflow is in archive
	archiveService, err := cluster.ArgoClient.ArchivedWorkflowService()
	require.NoError(t, err, "Failed to get archive service")

	// Try to get the archived workflow
	_, err = archiveService.GetArchivedWorkflow(clientCtx, &workflowarchive.GetArchivedWorkflowRequest{
		Uid: workflowUID,
	})
	require.NoError(t, err, "Workflow should be in archive")

	t.Log("Workflow is in archive")
	return workflowName, workflowUID
}

// TestArchive_RequiresArgoServerMode verifies that archive tools require Argo Server mode.
func TestArchive_RequiresArgoServerMode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// This test documents the requirement - archive tools only work in Argo Server mode
	if cluster.ConnectionMode == ModeArgoServer {
		t.Log("In Argo Server mode - archive tools should be available")

		// Verify we can get the archive service
		archiveService, err := cluster.ArgoClient.ArchivedWorkflowService()
		require.NoError(t, err, "Should be able to get archive service in Argo Server mode")
		require.NotNil(t, archiveService, "Archive service should not be nil")

		// Verify we can list archived workflows (archive is configured)
		clientCtx := cluster.ArgoClient.Context()
		_, err = archiveService.ListArchivedWorkflows(clientCtx, &workflowarchive.ListArchivedWorkflowsRequest{})
		require.NoError(t, err, "Should be able to list archived workflows")
	} else {
		t.Log("In Kubernetes API mode - archive tools are not available")

		// In Kubernetes mode, archive service should return an error
		_, err := cluster.ArgoClient.ArchivedWorkflowService()
		assert.Error(t, err, "Archive service should not be available in Kubernetes mode")
	}
}

// TestArchive_DeleteArchivedWorkflow tests the delete_archived_workflow tool.
func TestArchive_DeleteArchivedWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Archive requires Argo Server mode
	if cluster.ConnectionMode != ModeArgoServer {
		t.Skip("Skipping test: archive requires Argo Server mode")
	}

	clientCtx := cluster.ArgoClient.Context()

	// Submit a workflow and wait for it to be archived
	workflowName, workflowUID := submitAndArchiveWorkflow(t, cluster, "hello-world.yaml")

	// Cleanup the live workflow
	defer func() {
		deleteHandler := tools.DeleteWorkflowHandler(cluster.ArgoClient)
		deleteInput := tools.DeleteWorkflowInput{
			Namespace: cluster.ArgoNamespace,
			Name:      workflowName,
		}
		_, _, _ = deleteHandler(clientCtx, nil, deleteInput)
	}()

	// Test delete_archived_workflow tool
	t.Log("Testing delete_archived_workflow tool...")
	deleteArchiveHandler := tools.DeleteArchivedWorkflowHandler(cluster.ArgoClient)
	deleteInput := tools.DeleteArchivedWorkflowInput{
		UID: workflowUID,
	}

	_, deleteOutput, err := deleteArchiveHandler(clientCtx, nil, deleteInput)
	require.NoError(t, err, "delete_archived_workflow should not return error")
	require.NotNil(t, deleteOutput)

	// Verify output
	assert.Equal(t, workflowUID, deleteOutput.UID, "Deleted UID should match")
	assert.NotEmpty(t, deleteOutput.Message, "Message should be set")

	t.Logf("Deleted archived workflow: %s", deleteOutput.Message)

	// Verify workflow is no longer in archive
	archiveService, err := cluster.ArgoClient.ArchivedWorkflowService()
	require.NoError(t, err, "Failed to get archive service")

	_, err = archiveService.GetArchivedWorkflow(clientCtx, &workflowarchive.GetArchivedWorkflowRequest{
		Uid: workflowUID,
	})
	assert.Error(t, err, "Archived workflow should be deleted")
}

// TestArchive_ResubmitArchivedWorkflow tests the resubmit_archived_workflow tool.
func TestArchive_ResubmitArchivedWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Archive requires Argo Server mode
	if cluster.ConnectionMode != ModeArgoServer {
		t.Skip("Skipping test: archive requires Argo Server mode")
	}

	clientCtx := cluster.ArgoClient.Context()

	// Submit a workflow and wait for it to be archived
	workflowName, workflowUID := submitAndArchiveWorkflow(t, cluster, "hello-world.yaml")

	var newWorkflowName string

	// Cleanup both workflows at the end
	defer func() {
		deleteHandler := tools.DeleteWorkflowHandler(cluster.ArgoClient)

		// Delete original
		deleteInput := tools.DeleteWorkflowInput{
			Namespace: cluster.ArgoNamespace,
			Name:      workflowName,
		}
		_, _, _ = deleteHandler(clientCtx, nil, deleteInput)

		// Delete resubmitted if it exists
		if newWorkflowName != "" {
			deleteInput.Name = newWorkflowName
			_, _, _ = deleteHandler(clientCtx, nil, deleteInput)
		}
	}()

	// Test resubmit_archived_workflow tool
	t.Log("Testing resubmit_archived_workflow tool...")
	resubmitHandler := tools.ResubmitArchivedWorkflowHandler(cluster.ArgoClient)
	resubmitInput := tools.ResubmitArchivedWorkflowInput{
		UID:       workflowUID,
		Namespace: cluster.ArgoNamespace,
	}

	_, resubmitOutput, err := resubmitHandler(clientCtx, nil, resubmitInput)
	require.NoError(t, err, "resubmit_archived_workflow should not return error")
	require.NotNil(t, resubmitOutput)

	newWorkflowName = resubmitOutput.Name

	// Verify output
	assert.NotEmpty(t, resubmitOutput.Name, "New workflow name should be set")
	assert.NotEmpty(t, resubmitOutput.Namespace, "Namespace should be set")
	assert.NotEmpty(t, resubmitOutput.UID, "UID should be set")
	assert.NotEmpty(t, resubmitOutput.Message, "Message should be set")

	t.Logf("Resubmitted archived workflow as: %s", newWorkflowName)

	// Verify new workflow exists
	assert.True(t, cluster.WorkflowExists(t, cluster.ArgoNamespace, newWorkflowName),
		"Resubmitted workflow should exist")

	// Wait for new workflow to complete
	t.Log("Waiting for resubmitted workflow to complete...")
	finalPhase := cluster.WaitForWorkflowPhase(t, cluster.ArgoNamespace, newWorkflowName,
		2*time.Minute, "Succeeded", "Failed", "Error")

	// The key thing is that the workflow was resubmitted and ran
	assert.Contains(t, []string{"Succeeded", "Failed", "Error"}, finalPhase,
		"Resubmitted workflow should complete")
}

// TestArchive_RetryArchivedWorkflow tests the retry_archived_workflow tool.
func TestArchive_RetryArchivedWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Archive requires Argo Server mode
	if cluster.ConnectionMode != ModeArgoServer {
		t.Skip("Skipping test: archive requires Argo Server mode")
	}

	clientCtx := cluster.ArgoClient.Context()

	// Submit a FAILING workflow and wait for it to be archived
	// Retry only makes sense for failed workflows
	workflowName, workflowUID := submitAndArchiveWorkflow(t, cluster, "failing-workflow.yaml")

	// Delete the live workflow - retry_archived_workflow only works when the workflow
	// is not present on the cluster (if it exists, Argo says to use regular retry instead)
	t.Log("Deleting live workflow to test retry from archive...")
	deleteHandler := tools.DeleteWorkflowHandler(cluster.ArgoClient)
	deleteInput := tools.DeleteWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Name:      workflowName,
	}
	_, _, err := deleteHandler(clientCtx, nil, deleteInput)
	require.NoError(t, err, "Failed to delete workflow before retry from archive")

	// Test retry_archived_workflow tool
	t.Log("Testing retry_archived_workflow tool...")
	retryHandler := tools.RetryArchivedWorkflowHandler(cluster.ArgoClient)
	retryInput := tools.RetryArchivedWorkflowInput{
		UID:               workflowUID,
		Namespace:         cluster.ArgoNamespace,
		RestartSuccessful: true, // Restart all nodes
	}

	_, retryOutput, err := retryHandler(clientCtx, nil, retryInput)
	require.NoError(t, err, "retry_archived_workflow should not return error")
	require.NotNil(t, retryOutput)

	// Cleanup the retried workflow at the end
	defer func() {
		if retryOutput != nil && retryOutput.Name != "" {
			cleanupInput := tools.DeleteWorkflowInput{
				Namespace: cluster.ArgoNamespace,
				Name:      retryOutput.Name,
			}
			_, _, _ = deleteHandler(clientCtx, nil, cleanupInput)
		}
	}()

	// Verify output
	assert.NotEmpty(t, retryOutput.Name, "Retried workflow name should be set")
	assert.NotEmpty(t, retryOutput.Namespace, "Namespace should be set")
	assert.NotEmpty(t, retryOutput.UID, "UID should be set")
	assert.NotEmpty(t, retryOutput.Message, "Message should be set")

	t.Logf("Retried archived workflow: %s", retryOutput.Name)

	// The retried workflow should exist
	assert.True(t, cluster.WorkflowExists(t, cluster.ArgoNamespace, retryOutput.Name),
		"Retried workflow should exist")

	// Wait for retried workflow to complete
	t.Log("Waiting for retried workflow to complete...")
	finalPhase := cluster.WaitForWorkflowPhase(t, cluster.ArgoNamespace, retryOutput.Name,
		2*time.Minute, "Succeeded", "Failed", "Error")

	// The key thing is that the workflow was retried and ran
	// It will likely fail again since we're not changing the parameters
	assert.Contains(t, []string{"Succeeded", "Failed", "Error"}, finalPhase,
		"Retried workflow should complete")
}

// TestArchive_DeleteArchivedWorkflow_NotFound tests delete with non-existent UID.
func TestArchive_DeleteArchivedWorkflow_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Archive requires Argo Server mode
	if cluster.ConnectionMode != ModeArgoServer {
		t.Skip("Skipping test: archive requires Argo Server mode")
	}

	clientCtx := cluster.ArgoClient.Context()

	// Test delete_archived_workflow with non-existent UID
	t.Log("Testing delete_archived_workflow with non-existent UID...")
	deleteHandler := tools.DeleteArchivedWorkflowHandler(cluster.ArgoClient)
	deleteInput := tools.DeleteArchivedWorkflowInput{
		UID: "non-existent-uid-12345",
	}

	_, _, err := deleteHandler(clientCtx, nil, deleteInput)
	assert.Error(t, err, "delete_archived_workflow should return error for non-existent UID")
	t.Logf("Expected error for non-existent UID: %v", err)
}

// TestArchive_ResubmitArchivedWorkflow_NotFound tests resubmit with non-existent UID.
func TestArchive_ResubmitArchivedWorkflow_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Archive requires Argo Server mode
	if cluster.ConnectionMode != ModeArgoServer {
		t.Skip("Skipping test: archive requires Argo Server mode")
	}

	clientCtx := cluster.ArgoClient.Context()

	// Test resubmit_archived_workflow with non-existent UID
	t.Log("Testing resubmit_archived_workflow with non-existent UID...")
	resubmitHandler := tools.ResubmitArchivedWorkflowHandler(cluster.ArgoClient)
	resubmitInput := tools.ResubmitArchivedWorkflowInput{
		UID: "non-existent-uid-12345",
	}

	_, _, err := resubmitHandler(clientCtx, nil, resubmitInput)
	assert.Error(t, err, "resubmit_archived_workflow should return error for non-existent UID")
	t.Logf("Expected error for non-existent UID: %v", err)
}

// TestArchive_RetryArchivedWorkflow_NotFound tests retry with non-existent UID.
func TestArchive_RetryArchivedWorkflow_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Archive requires Argo Server mode
	if cluster.ConnectionMode != ModeArgoServer {
		t.Skip("Skipping test: archive requires Argo Server mode")
	}

	clientCtx := cluster.ArgoClient.Context()

	// Test retry_archived_workflow with non-existent UID
	t.Log("Testing retry_archived_workflow with non-existent UID...")
	retryHandler := tools.RetryArchivedWorkflowHandler(cluster.ArgoClient)
	retryInput := tools.RetryArchivedWorkflowInput{
		UID: "non-existent-uid-12345",
	}

	_, _, err := retryHandler(clientCtx, nil, retryInput)
	assert.Error(t, err, "retry_archived_workflow should return error for non-existent UID")
	t.Logf("Expected error for non-existent UID: %v", err)
}

// TestArchive_DeleteArchivedWorkflow_EmptyUID tests delete with empty UID.
func TestArchive_DeleteArchivedWorkflow_EmptyUID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Archive requires Argo Server mode
	if cluster.ConnectionMode != ModeArgoServer {
		t.Skip("Skipping test: archive requires Argo Server mode")
	}

	clientCtx := cluster.ArgoClient.Context()

	// Test delete_archived_workflow with empty UID
	t.Log("Testing delete_archived_workflow with empty UID...")
	deleteHandler := tools.DeleteArchivedWorkflowHandler(cluster.ArgoClient)
	deleteInput := tools.DeleteArchivedWorkflowInput{
		UID: "",
	}

	_, _, err := deleteHandler(clientCtx, nil, deleteInput)
	assert.Error(t, err, "delete_archived_workflow should return error for empty UID")
	assert.Contains(t, err.Error(), "cannot be empty", "Error should mention empty UID")
}

// TestArchive_ResubmitArchivedWorkflow_EmptyUID tests resubmit with empty UID.
func TestArchive_ResubmitArchivedWorkflow_EmptyUID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Archive requires Argo Server mode
	if cluster.ConnectionMode != ModeArgoServer {
		t.Skip("Skipping test: archive requires Argo Server mode")
	}

	clientCtx := cluster.ArgoClient.Context()

	// Test resubmit_archived_workflow with empty UID
	t.Log("Testing resubmit_archived_workflow with empty UID...")
	resubmitHandler := tools.ResubmitArchivedWorkflowHandler(cluster.ArgoClient)
	resubmitInput := tools.ResubmitArchivedWorkflowInput{
		UID: "",
	}

	_, _, err := resubmitHandler(clientCtx, nil, resubmitInput)
	assert.Error(t, err, "resubmit_archived_workflow should return error for empty UID")
	assert.Contains(t, err.Error(), "cannot be empty", "Error should mention empty UID")
}

// TestArchive_RetryArchivedWorkflow_EmptyUID tests retry with empty UID.
func TestArchive_RetryArchivedWorkflow_EmptyUID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Archive requires Argo Server mode
	if cluster.ConnectionMode != ModeArgoServer {
		t.Skip("Skipping test: archive requires Argo Server mode")
	}

	clientCtx := cluster.ArgoClient.Context()

	// Test retry_archived_workflow with empty UID
	t.Log("Testing retry_archived_workflow with empty UID...")
	retryHandler := tools.RetryArchivedWorkflowHandler(cluster.ArgoClient)
	retryInput := tools.RetryArchivedWorkflowInput{
		UID: "",
	}

	_, _, err := retryHandler(clientCtx, nil, retryInput)
	assert.Error(t, err, "retry_archived_workflow should return error for empty UID")
	assert.Contains(t, err.Error(), "cannot be empty", "Error should mention empty UID")
}
