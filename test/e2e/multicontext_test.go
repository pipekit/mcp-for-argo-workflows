//go:build e2e

package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/k3s"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/pipekit/mcp-for-argo-workflows/pkg/argo"
	"github.com/pipekit/mcp-for-argo-workflows/pkg/tools"
)

// TestMultiContext_RoutesToSelectedCluster is the cross-cluster routing proof
// for per-call kubeconfig context selection. It runs two k3s clusters under one
// kubeconfig (contexts "alpha" and "beta"), submits a workflow only to beta via
// the context parameter, and asserts each cluster sees exactly its own
// workflows. Without the MergeContext auth plumbing this fails: the beta call
// would silently execute against alpha.
func TestMultiContext_RoutesToSelectedCluster(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}
	if GetConnectionMode() != ModeKubernetesAPI {
		t.Skip("Multi-context only applies to direct Kubernetes mode")
	}

	ctx := context.Background()
	clusterAlpha := SetupE2ECluster(ctx, t)

	// Start a second cluster (beta) and install Argo on it. The controller
	// does not need to be ready: creating and listing Workflow objects only
	// requires the CRDs.
	t.Log("Starting second k3s container (beta)...")
	betaContainer, err := k3s.Run(ctx, "rancher/k3s:v1.31.2-k3s1")
	require.NoError(t, err, "Failed to start beta k3s container")
	t.Cleanup(func() {
		if termErr := betaContainer.Terminate(context.Background()); termErr != nil {
			t.Logf("Failed to terminate beta container: %v", termErr)
		}
	})

	betaKubeconfig, err := betaContainer.GetKubeConfig(ctx)
	require.NoError(t, err, "Failed to get beta kubeconfig")

	betaKubeconfigPath := filepath.Join(t.TempDir(), "beta-kubeconfig.yaml")
	require.NoError(t, os.WriteFile(betaKubeconfigPath, betaKubeconfig, 0o600))

	t.Log("Installing Argo Workflows on beta...")
	require.NoError(t, installArgoWorkflowsShared(t, betaKubeconfigPath, ModeKubernetesAPI))

	// Merge both kubeconfigs into one file with contexts "alpha" and "beta".
	mergedPath := writeMergedKubeconfig(t, []byte(clusterAlpha.Kubeconfig), betaKubeconfig)

	multiClient, err := argo.NewMultiContextClient(ctx, &argo.Config{
		Kubeconfig: mergedPath,
		Namespace:  ArgoNamespace,
	}, nil)
	require.NoError(t, err, "Failed to create multi-context client")

	// Discovery: both contexts visible, current-context is the default.
	names, defaultContext, err := multiClient.ListKubeContexts()
	require.NoError(t, err)
	assert.Equal(t, []string{"alpha", "beta"}, names)
	assert.Equal(t, "alpha", defaultContext)

	// The transport runs with the default client's context, exactly as main.go
	// wires it.
	clientCtx := multiClient.Context()
	manifest := LoadTestDataFile(t, "hello-world.yaml")

	// Submit a workflow to beta only, selecting it via the context parameter.
	submitHandler := tools.SubmitWorkflowHandler(multiClient)
	_, submitOutput, err := submitHandler(clientCtx, nil, tools.SubmitWorkflowInput{
		KubeContextInput: tools.KubeContextInput{KubeContext: "beta"},
		Namespace:        ArgoNamespace,
		Manifest:         manifest,
	})
	require.NoError(t, err, "Failed to submit workflow to beta")
	betaWorkflow := submitOutput.Name
	t.Logf("Submitted workflow %s to beta", betaWorkflow)

	listHandler := tools.ListWorkflowsHandler(multiClient)

	// Beta sees the workflow.
	_, betaList, err := listHandler(clientCtx, nil, tools.ListWorkflowsInput{
		KubeContextInput: tools.KubeContextInput{KubeContext: "beta"},
	})
	require.NoError(t, err, "Failed to list workflows on beta")
	assert.True(t, containsWorkflow(betaList, betaWorkflow),
		"workflow submitted with context=beta must be visible on beta")

	// Alpha (the default) must NOT see it — this is the assertion that fails
	// if per-context calls silently run against the default cluster.
	_, alphaList, err := listHandler(clientCtx, nil, tools.ListWorkflowsInput{})
	require.NoError(t, err, "Failed to list workflows on alpha")
	assert.False(t, containsWorkflow(alphaList, betaWorkflow),
		"workflow submitted with context=beta must not appear on the default cluster")

	// Selecting the default context explicitly matches the implicit default.
	_, alphaExplicit, err := listHandler(clientCtx, nil, tools.ListWorkflowsInput{
		KubeContextInput: tools.KubeContextInput{KubeContext: "alpha"},
	})
	require.NoError(t, err)
	assert.False(t, containsWorkflow(alphaExplicit, betaWorkflow))

	// Unknown contexts are rejected with the uniform not-available error.
	_, _, err = listHandler(clientCtx, nil, tools.ListWorkflowsInput{
		KubeContextInput: tools.KubeContextInput{KubeContext: "does-not-exist"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `context "does-not-exist" is not available`)
}

func containsWorkflow(output *tools.ListWorkflowsOutput, name string) bool {
	for _, wf := range output.Workflows {
		if wf.Name == name {
			return true
		}
	}
	return false
}

// writeMergedKubeconfig combines two single-context kubeconfigs into one file
// with contexts named "alpha" and "beta", current-context alpha.
func writeMergedKubeconfig(t *testing.T, alphaConfig, betaConfig []byte) string {
	t.Helper()

	merged := clientcmdapi.NewConfig()
	for name, raw := range map[string][]byte{"alpha": alphaConfig, "beta": betaConfig} {
		source, err := clientcmd.Load(raw)
		require.NoError(t, err, "Failed to parse %s kubeconfig", name)
		sourceContext, ok := source.Contexts[source.CurrentContext]
		require.True(t, ok, "%s kubeconfig has no current context", name)

		merged.Clusters[name] = source.Clusters[sourceContext.Cluster]
		merged.AuthInfos[name] = source.AuthInfos[sourceContext.AuthInfo]
		merged.Contexts[name] = &clientcmdapi.Context{Cluster: name, AuthInfo: name}
	}
	merged.CurrentContext = "alpha"

	path := filepath.Join(t.TempDir(), "merged-kubeconfig.yaml")
	require.NoError(t, clientcmd.WriteToFile(*merged, path))
	t.Logf("Merged kubeconfig written to: %s", path)
	return path
}
