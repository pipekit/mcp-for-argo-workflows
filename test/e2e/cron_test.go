//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/tools"
)

// TestCronWorkflow_CRUD tests the full CRUD lifecycle: create → get → list → delete.
func TestCronWorkflow_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Use the client's context which contains the KubeClient
	clientCtx := cluster.ArgoClient.Context()

	// Load test cron workflow manifest
	manifest := LoadTestDataFile(t, "cron-workflow.yaml")

	// Step 1: Create cron workflow
	t.Log("Creating cron workflow...")
	createHandler := tools.CreateCronWorkflowHandler(cluster.ArgoClient)
	createInput := tools.CreateCronWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Manifest:  manifest,
	}

	_, createOutput, err := createHandler(clientCtx, nil, createInput)
	require.NoError(t, err, "Failed to create cron workflow")
	require.NotNil(t, createOutput)

	cronName := createOutput.Name
	t.Logf("Created cron workflow: %s", cronName)

	// Verify cron workflow was created
	assert.True(t, cluster.CronWorkflowExists(t, cluster.ArgoNamespace, cronName),
		"CronWorkflow should exist after creation")
	assert.Equal(t, []string{"0 0 * * *"}, createOutput.Schedules)
	assert.False(t, createOutput.Suspended, "CronWorkflow should not be suspended on creation")

	// Cleanup at the end (skipped if explicit delete in Step 4 succeeded)
	defer func() {
		if cronName == "" {
			return
		}
		deleteHandler := tools.DeleteCronWorkflowHandler(cluster.ArgoClient)
		deleteInput := tools.DeleteCronWorkflowInput{
			Namespace: cluster.ArgoNamespace,
			Name:      cronName,
		}
		if _, _, err := deleteHandler(clientCtx, nil, deleteInput); err != nil {
			t.Logf("Cleanup: failed to delete cron workflow %s: %v", cronName, err)
		}
	}()

	// Step 2: Get cron workflow
	t.Log("Getting cron workflow...")
	getHandler := tools.GetCronWorkflowHandler(cluster.ArgoClient)
	getInput := tools.GetCronWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Name:      cronName,
	}

	_, getOutput, err := getHandler(clientCtx, nil, getInput)
	require.NoError(t, err, "Failed to get cron workflow")
	require.NotNil(t, getOutput)

	assert.Equal(t, cronName, getOutput.Name)
	assert.Equal(t, cluster.ArgoNamespace, getOutput.Namespace)
	assert.Equal(t, []string{"0 0 * * *"}, getOutput.Schedules)
	assert.Equal(t, "America/Los_Angeles", getOutput.Timezone)
	assert.Equal(t, "Replace", getOutput.ConcurrencyPolicy)
	assert.NotEmpty(t, getOutput.CreatedAt)
	assert.False(t, getOutput.Suspended)

	// Step 3: List cron workflows
	t.Log("Listing cron workflows...")
	listHandler := tools.ListCronWorkflowsHandler(cluster.ArgoClient)
	listInput := tools.ListCronWorkflowsInput{
		Namespace: cluster.ArgoNamespace,
	}

	_, listOutput, err := listHandler(clientCtx, nil, listInput)
	require.NoError(t, err, "Failed to list cron workflows")
	require.NotNil(t, listOutput)

	// Verify our cron workflow is in the list
	assert.NotEmpty(t, listOutput.CronWorkflows, "Should have at least one cron workflow")

	found := false
	for _, cw := range listOutput.CronWorkflows {
		if cw.Name == cronName {
			found = true
			assert.Equal(t, cluster.ArgoNamespace, cw.Namespace)
			assert.Equal(t, []string{"0 0 * * *"}, cw.Schedules)
			assert.False(t, cw.Suspended)
			break
		}
	}
	assert.True(t, found, "Created cron workflow should be in the list")

	// Step 4: Delete cron workflow
	t.Log("Deleting cron workflow...")
	deleteHandler := tools.DeleteCronWorkflowHandler(cluster.ArgoClient)
	deleteInput := tools.DeleteCronWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Name:      cronName,
	}

	_, deleteOutput, err := deleteHandler(clientCtx, nil, deleteInput)
	require.NoError(t, err, "Failed to delete cron workflow")
	require.NotNil(t, deleteOutput)

	assert.Equal(t, cronName, deleteOutput.Name)

	// Mark as deleted so deferred cleanup is skipped
	deletedName := cronName
	cronName = ""

	// Verify cron workflow was deleted
	require.Eventually(t, func() bool {
		return !cluster.CronWorkflowExists(t, cluster.ArgoNamespace, deletedName)
	}, 10*time.Second, 500*time.Millisecond, "CronWorkflow should be deleted")
}

// TestCronWorkflow_SuspendResume tests suspending and resuming a cron workflow.
func TestCronWorkflow_SuspendResume(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Use the client's context which contains the KubeClient
	clientCtx := cluster.ArgoClient.Context()

	// Step 1: Create cron workflow
	t.Log("Creating cron workflow...")
	manifest := LoadTestDataFile(t, "cron-workflow.yaml")

	createHandler := tools.CreateCronWorkflowHandler(cluster.ArgoClient)
	createInput := tools.CreateCronWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Manifest:  manifest,
	}

	_, createOutput, err := createHandler(clientCtx, nil, createInput)
	require.NoError(t, err, "Failed to create cron workflow")

	cronName := createOutput.Name
	t.Logf("Created cron workflow: %s", cronName)

	// Verify initially not suspended
	assert.False(t, createOutput.Suspended, "CronWorkflow should not be suspended initially")

	// Cleanup at the end
	defer func() {
		deleteHandler := tools.DeleteCronWorkflowHandler(cluster.ArgoClient)
		deleteInput := tools.DeleteCronWorkflowInput{
			Namespace: cluster.ArgoNamespace,
			Name:      cronName,
		}
		if _, _, err := deleteHandler(clientCtx, nil, deleteInput); err != nil {
			t.Logf("Cleanup: failed to delete cron workflow %s: %v", cronName, err)
		}
	}()

	// Step 2: Suspend the cron workflow
	t.Log("Suspending cron workflow...")
	suspendHandler := tools.SuspendCronWorkflowHandler(cluster.ArgoClient)
	suspendInput := tools.SuspendCronWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Name:      cronName,
	}

	_, suspendOutput, err := suspendHandler(clientCtx, nil, suspendInput)
	require.NoError(t, err, "Failed to suspend cron workflow")
	require.NotNil(t, suspendOutput)

	assert.Equal(t, cronName, suspendOutput.Name)
	assert.Contains(t, suspendOutput.Message, "suspended")

	// Verify it's now suspended by getting it
	getHandler := tools.GetCronWorkflowHandler(cluster.ArgoClient)
	getInput := tools.GetCronWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Name:      cronName,
	}

	_, getOutput, err := getHandler(clientCtx, nil, getInput)
	require.NoError(t, err, "Failed to get cron workflow after suspend")
	assert.True(t, getOutput.Suspended, "CronWorkflow should be suspended")

	// Step 3: Resume the cron workflow
	t.Log("Resuming cron workflow...")
	resumeHandler := tools.ResumeCronWorkflowHandler(cluster.ArgoClient)
	resumeInput := tools.ResumeCronWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Name:      cronName,
	}

	_, resumeOutput, err := resumeHandler(clientCtx, nil, resumeInput)
	require.NoError(t, err, "Failed to resume cron workflow")
	require.NotNil(t, resumeOutput)

	assert.Equal(t, cronName, resumeOutput.Name)
	assert.Contains(t, resumeOutput.Message, "resumed")

	// Verify it's no longer suspended
	_, getOutput, err = getHandler(clientCtx, nil, getInput)
	require.NoError(t, err, "Failed to get cron workflow after resume")
	assert.False(t, getOutput.Suspended, "CronWorkflow should not be suspended after resume")
}

// TestCronWorkflow_Upsert tests that creating an existing cron workflow updates it.
func TestCronWorkflow_Upsert(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Use the client's context which contains the KubeClient
	clientCtx := cluster.ArgoClient.Context()

	// Load test cron workflow manifest
	manifest := LoadTestDataFile(t, "cron-workflow.yaml")

	// Step 1: Create cron workflow
	t.Log("Creating cron workflow...")
	createHandler := tools.CreateCronWorkflowHandler(cluster.ArgoClient)
	createInput := tools.CreateCronWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Manifest:  manifest,
	}

	_, createOutput, err := createHandler(clientCtx, nil, createInput)
	require.NoError(t, err, "Failed to create cron workflow")
	require.NotNil(t, createOutput)
	assert.True(t, createOutput.Created, "Should indicate cron workflow was created")

	cronName := createOutput.Name
	t.Logf("Created cron workflow: %s", cronName)

	// Cleanup at the end
	defer func() {
		deleteHandler := tools.DeleteCronWorkflowHandler(cluster.ArgoClient)
		deleteInput := tools.DeleteCronWorkflowInput{
			Namespace: cluster.ArgoNamespace,
			Name:      cronName,
		}
		_, _, _ = deleteHandler(clientCtx, nil, deleteInput)
	}()

	// Step 2: "Create" the same cron workflow again (should update)
	t.Log("Creating same cron workflow again (should update)...")
	_, updateOutput, err := createHandler(clientCtx, nil, createInput)
	require.NoError(t, err, "Failed to update cron workflow")
	require.NotNil(t, updateOutput)
	assert.False(t, updateOutput.Created, "Should indicate cron workflow was updated, not created")
	assert.Equal(t, cronName, updateOutput.Name)

	// Step 3: Verify the cron workflow is still accessible
	t.Log("Verifying cron workflow after upsert...")
	getHandler := tools.GetCronWorkflowHandler(cluster.ArgoClient)
	getInput := tools.GetCronWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Name:      cronName,
	}

	_, getOutput, err := getHandler(clientCtx, nil, getInput)
	require.NoError(t, err, "Failed to get cron workflow after upsert")
	require.NotNil(t, getOutput)
	assert.Equal(t, cronName, getOutput.Name)
	assert.Equal(t, []string{"0 0 * * *"}, getOutput.Schedules)
}

// TestCronWorkflow_GetConsistency tests that getting a cron workflow returns consistent data.
func TestCronWorkflow_GetConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Use the client's context which contains the KubeClient
	clientCtx := cluster.ArgoClient.Context()

	// Create cron workflow
	t.Log("Creating cron workflow...")
	manifest := LoadTestDataFile(t, "cron-workflow.yaml")

	createHandler := tools.CreateCronWorkflowHandler(cluster.ArgoClient)
	createInput := tools.CreateCronWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Manifest:  manifest,
	}

	_, createOutput, err := createHandler(clientCtx, nil, createInput)
	require.NoError(t, err, "Failed to create cron workflow")

	cronName := createOutput.Name
	t.Logf("Created cron workflow: %s", cronName)

	// Cleanup at the end
	defer func() {
		deleteHandler := tools.DeleteCronWorkflowHandler(cluster.ArgoClient)
		deleteInput := tools.DeleteCronWorkflowInput{
			Namespace: cluster.ArgoNamespace,
			Name:      cronName,
		}
		if _, _, err := deleteHandler(clientCtx, nil, deleteInput); err != nil {
			t.Logf("Cleanup: failed to delete cron workflow %s: %v", cronName, err)
		}
	}()

	// Get the cron workflow
	getHandler := tools.GetCronWorkflowHandler(cluster.ArgoClient)
	getInput := tools.GetCronWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Name:      cronName,
	}

	_, getOutput1, err := getHandler(clientCtx, nil, getInput)
	require.NoError(t, err, "Failed to get cron workflow")
	require.NotNil(t, getOutput1)

	originalCreatedAt := getOutput1.CreatedAt
	require.NotEmpty(t, originalCreatedAt)

	// Verify the cron workflow is stable by checking multiple Get calls return consistent data
	var getOutput2 *tools.GetCronWorkflowOutput
	require.Eventually(t, func() bool {
		_, out, err := getHandler(clientCtx, nil, getInput)
		if err != nil || out == nil {
			return false
		}
		getOutput2 = out
		return out.CreatedAt == originalCreatedAt
	}, 15*time.Second, 500*time.Millisecond, "CreatedAt should remain stable")

	// Verify the cron workflow is consistent
	assert.Equal(t, originalCreatedAt, getOutput2.CreatedAt, "CreatedAt should remain the same")
	assert.Equal(t, cronName, getOutput2.Name)
	assert.Equal(t, []string{"0 0 * * *"}, getOutput2.Schedules)
	assert.Equal(t, "America/Los_Angeles", getOutput2.Timezone)
}
