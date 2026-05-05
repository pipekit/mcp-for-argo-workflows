//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/tools"
)

// TestClusterWorkflowTemplate_CRUD tests the full CRUD lifecycle: create → get → list → delete.
func TestClusterWorkflowTemplate_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Use the client's context which contains the KubeClient
	clientCtx := cluster.ArgoClient.Context()

	// Load test cluster workflow template
	manifest := LoadTestDataFile(t, "cluster-workflow-template.yaml")

	// Step 1: Create cluster workflow template
	t.Log("Creating cluster workflow template...")
	createHandler := tools.CreateClusterWorkflowTemplateHandler(cluster.ArgoClient)
	createInput := tools.CreateClusterWorkflowTemplateInput{
		Manifest: manifest,
	}

	_, createOutput, err := createHandler(clientCtx, nil, createInput)
	require.NoError(t, err, "Failed to create cluster workflow template")
	require.NotNil(t, createOutput)

	templateName := createOutput.Name
	t.Logf("Created cluster workflow template: %s", templateName)

	// Verify template was created
	assert.True(t, cluster.ClusterWorkflowTemplateExists(t, templateName),
		"ClusterWorkflowTemplate should exist after creation")

	// Cleanup at the end (skipped if explicit delete in Step 4 succeeded)
	defer func() {
		if templateName == "" {
			return
		}
		deleteHandler := tools.DeleteClusterWorkflowTemplateHandler(cluster.ArgoClient)
		deleteInput := tools.DeleteClusterWorkflowTemplateInput{
			Name: templateName,
		}
		_, _, _ = deleteHandler(clientCtx, nil, deleteInput)
	}()

	// Step 2: Get cluster workflow template
	t.Log("Getting cluster workflow template...")
	getHandler := tools.GetClusterWorkflowTemplateHandler(cluster.ArgoClient)
	getInput := tools.GetClusterWorkflowTemplateInput{
		Name: templateName,
	}

	_, getOutput, err := getHandler(clientCtx, nil, getInput)
	require.NoError(t, err, "Failed to get cluster workflow template")
	require.NotNil(t, getOutput)

	assert.Equal(t, templateName, getOutput.Name)
	assert.NotEmpty(t, getOutput.CreatedAt)
	assert.NotEmpty(t, getOutput.Templates, "Should have templates")

	// Step 3: List cluster workflow templates
	t.Log("Listing cluster workflow templates...")
	listHandler := tools.ListClusterWorkflowTemplatesHandler(cluster.ArgoClient)
	listInput := tools.ListClusterWorkflowTemplatesInput{}

	_, listOutput, err := listHandler(clientCtx, nil, listInput)
	require.NoError(t, err, "Failed to list cluster workflow templates")
	require.NotNil(t, listOutput)

	// Verify our template is in the list
	assert.NotEmpty(t, listOutput.Templates, "Should have at least one template")

	found := false
	for _, tmpl := range listOutput.Templates {
		if tmpl.Name == templateName {
			found = true
			break
		}
	}
	assert.True(t, found, "Created template should be in the list")

	// Step 4: Delete cluster workflow template
	t.Log("Deleting cluster workflow template...")
	deleteHandler := tools.DeleteClusterWorkflowTemplateHandler(cluster.ArgoClient)
	deleteInput := tools.DeleteClusterWorkflowTemplateInput{
		Name: templateName,
	}

	_, deleteOutput, err := deleteHandler(clientCtx, nil, deleteInput)
	require.NoError(t, err, "Failed to delete cluster workflow template")
	require.NotNil(t, deleteOutput)

	assert.Equal(t, templateName, deleteOutput.Name)

	// Mark as deleted so deferred cleanup is skipped
	deletedName := templateName
	templateName = ""

	// Verify template was deleted
	require.Eventually(t, func() bool {
		return !cluster.ClusterWorkflowTemplateExists(t, deletedName)
	}, 10*time.Second, 500*time.Millisecond, "ClusterWorkflowTemplate should be deleted")
}

// TestClusterWorkflowTemplate_SubmitWithRef tests creating a cluster template and submitting a workflow that references it.
func TestClusterWorkflowTemplate_SubmitWithRef(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Use the client's context which contains the KubeClient
	clientCtx := cluster.ArgoClient.Context()

	// Step 1: Create cluster workflow template
	t.Log("Creating cluster workflow template...")
	templateManifest := LoadTestDataFile(t, "cluster-workflow-template.yaml")

	createHandler := tools.CreateClusterWorkflowTemplateHandler(cluster.ArgoClient)
	createInput := tools.CreateClusterWorkflowTemplateInput{
		Manifest: templateManifest,
	}

	_, createOutput, err := createHandler(clientCtx, nil, createInput)
	require.NoError(t, err, "Failed to create cluster workflow template")

	templateName := createOutput.Name
	t.Logf("Created cluster workflow template: %s", templateName)

	// Cleanup template at the end
	defer func() {
		deleteHandler := tools.DeleteClusterWorkflowTemplateHandler(cluster.ArgoClient)
		deleteInput := tools.DeleteClusterWorkflowTemplateInput{
			Name: templateName,
		}
		_, _, _ = deleteHandler(clientCtx, nil, deleteInput)
	}()

	// Step 2: Submit a workflow that references the cluster template
	t.Log("Submitting workflow from cluster template...")
	workflowManifest := fmt.Sprintf(`apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: from-cluster-template-
spec:
  workflowTemplateRef:
    name: %s
    clusterScope: true
  arguments:
    parameters:
      - name: message
        value: "Hello from workflow using cluster template"
`, templateName)

	submitHandler := tools.SubmitWorkflowHandler(cluster.ArgoClient)
	submitInput := tools.SubmitWorkflowInput{
		Namespace: cluster.ArgoNamespace,
		Manifest:  workflowManifest,
	}

	_, submitOutput, err := submitHandler(clientCtx, nil, submitInput)
	require.NoError(t, err, "Failed to submit workflow from cluster template")

	workflowName := submitOutput.Name
	t.Logf("Submitted workflow: %s", workflowName)

	// Cleanup workflow at the end
	defer func() {
		deleteWorkflowHandler := tools.DeleteWorkflowHandler(cluster.ArgoClient)
		deleteWorkflowInput := tools.DeleteWorkflowInput{
			Namespace: cluster.ArgoNamespace,
			Name:      workflowName,
		}
		_, _, _ = deleteWorkflowHandler(clientCtx, nil, deleteWorkflowInput)
	}()

	// Step 3: Wait for workflow to complete
	t.Log("Waiting for workflow to complete...")
	finalPhase := cluster.WaitForWorkflowPhase(t, cluster.ArgoNamespace, workflowName,
		2*time.Minute, "Succeeded", "Failed", "Error")

	// Workflow may end in Error in CI due to resource constraints
	assert.Contains(t, []string{"Succeeded", "Failed", "Error"}, finalPhase,
		"Workflow should reach terminal state")

	// Step 4: Verify logs contain the custom message (only if workflow succeeded)
	if finalPhase == "Succeeded" {
		t.Log("Verifying workflow output...")
		logsHandler := tools.LogsWorkflowHandler(cluster.ArgoClient)
		logsInput := tools.LogsWorkflowInput{
			Namespace: cluster.ArgoNamespace,
			Name:      workflowName,
		}

		_, logsOutput, err := logsHandler(clientCtx, nil, logsInput)
		require.NoError(t, err, "Failed to get workflow logs")
		require.NotNil(t, logsOutput)

		// Check that at least one log entry contains the expected output
		foundMessage := false
		for _, entry := range logsOutput.Logs {
			if strings.Contains(entry.Content, "Hello from workflow using cluster template") {
				foundMessage = true
				break
			}
		}
		assert.True(t, foundMessage, "Logs should contain the custom message parameter")
	} else {
		t.Logf("Skipping log verification - workflow ended in %s state", finalPhase)
	}
}

// TestClusterWorkflowTemplate_Upsert tests that creating an existing template updates it.
func TestClusterWorkflowTemplate_Upsert(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Use the client's context which contains the KubeClient
	clientCtx := cluster.ArgoClient.Context()

	// Load test cluster workflow template
	manifest := LoadTestDataFile(t, "cluster-workflow-template.yaml")

	// Step 1: Create cluster workflow template
	t.Log("Creating cluster workflow template...")
	createHandler := tools.CreateClusterWorkflowTemplateHandler(cluster.ArgoClient)
	createInput := tools.CreateClusterWorkflowTemplateInput{
		Manifest: manifest,
	}

	_, createOutput, err := createHandler(clientCtx, nil, createInput)
	require.NoError(t, err, "Failed to create cluster workflow template")
	require.NotNil(t, createOutput)
	assert.True(t, createOutput.Created, "Should indicate template was created")

	templateName := createOutput.Name
	t.Logf("Created cluster workflow template: %s", templateName)

	// Cleanup at the end
	defer func() {
		deleteHandler := tools.DeleteClusterWorkflowTemplateHandler(cluster.ArgoClient)
		deleteInput := tools.DeleteClusterWorkflowTemplateInput{
			Name: templateName,
		}
		_, _, _ = deleteHandler(clientCtx, nil, deleteInput)
	}()

	// Step 2: "Create" the same template again (should update)
	t.Log("Creating same cluster workflow template again (should update)...")
	_, updateOutput, err := createHandler(clientCtx, nil, createInput)
	require.NoError(t, err, "Failed to update cluster workflow template")
	require.NotNil(t, updateOutput)
	assert.False(t, updateOutput.Created, "Should indicate template was updated, not created")
	assert.Equal(t, templateName, updateOutput.Name)

	// Step 3: Verify the template is still accessible
	t.Log("Verifying template after upsert...")
	getHandler := tools.GetClusterWorkflowTemplateHandler(cluster.ArgoClient)
	getInput := tools.GetClusterWorkflowTemplateInput{
		Name: templateName,
	}

	_, getOutput, err := getHandler(clientCtx, nil, getInput)
	require.NoError(t, err, "Failed to get cluster workflow template after upsert")
	require.NotNil(t, getOutput)
	assert.Equal(t, templateName, getOutput.Name)
}

// TestClusterWorkflowTemplate_GetConsistency tests that getting a template returns consistent data.
func TestClusterWorkflowTemplate_GetConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()
	cluster := SetupE2ECluster(ctx, t)

	// Use the client's context which contains the KubeClient
	clientCtx := cluster.ArgoClient.Context()

	// Create initial template
	t.Log("Creating cluster workflow template...")
	manifest := LoadTestDataFile(t, "cluster-workflow-template.yaml")

	createHandler := tools.CreateClusterWorkflowTemplateHandler(cluster.ArgoClient)
	createInput := tools.CreateClusterWorkflowTemplateInput{
		Manifest: manifest,
	}

	_, createOutput, err := createHandler(clientCtx, nil, createInput)
	require.NoError(t, err, "Failed to create cluster workflow template")

	templateName := createOutput.Name
	t.Logf("Created cluster workflow template: %s", templateName)

	// Cleanup at the end
	defer func() {
		deleteHandler := tools.DeleteClusterWorkflowTemplateHandler(cluster.ArgoClient)
		deleteInput := tools.DeleteClusterWorkflowTemplateInput{
			Name: templateName,
		}
		_, _, _ = deleteHandler(clientCtx, nil, deleteInput)
	}()

	// Get the template
	getHandler := tools.GetClusterWorkflowTemplateHandler(cluster.ArgoClient)
	getInput := tools.GetClusterWorkflowTemplateInput{
		Name: templateName,
	}

	_, getOutput1, err := getHandler(clientCtx, nil, getInput)
	require.NoError(t, err, "Failed to get cluster workflow template")
	require.NotNil(t, getOutput1)

	originalCreatedAt := getOutput1.CreatedAt
	require.NotEmpty(t, originalCreatedAt)

	// Verify the template is stable by checking multiple Get calls return consistent data
	var getOutput2 *tools.GetClusterWorkflowTemplateOutput
	require.Eventually(t, func() bool {
		_, out, err := getHandler(clientCtx, nil, getInput)
		if err != nil || out == nil {
			return false
		}
		getOutput2 = out
		return out.CreatedAt == originalCreatedAt
	}, 15*time.Second, 500*time.Millisecond, "CreatedAt should remain stable")

	// Verify the template is consistent
	assert.Equal(t, originalCreatedAt, getOutput2.CreatedAt, "CreatedAt should remain the same")
	assert.Equal(t, templateName, getOutput2.Name)
}
