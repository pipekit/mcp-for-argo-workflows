//go:build e2e

// Package e2e contains end-to-end tests for the MCP server.
// Note: gosec security warnings are disabled for this test package as it intentionally
// uses exec.Command and file operations with test data.
package e2e

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/k3s"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// ConnectionMode specifies how the E2E tests connect to Argo Workflows.
type ConnectionMode string

const (
	// ModeKubernetesAPI uses direct Kubernetes API access (default).
	ModeKubernetesAPI ConnectionMode = "kubernetes"

	// ModeArgoServer connects via Argo Server API.
	ModeArgoServer ConnectionMode = "argo-server"
)

const (
	// ArgoVersion is the Argo Workflows version to install for E2E tests.
	ArgoVersion = "v3.7.6"

	// ArgoNamespace is the namespace where Argo Workflows is installed.
	ArgoNamespace = "argo"

	// ArgoQuickStartMinimalURL is the minimal quick-start manifest (no archiving).
	// Used for kubernetes mode where we don't need archive functionality.
	ArgoQuickStartMinimalURL = "https://github.com/argoproj/argo-workflows/releases/download/" + ArgoVersion + "/quick-start-minimal.yaml"

	// ArgoQuickStartPostgresURL enables workflow archiving with PostgreSQL.
	// Used for argo-server mode to test archive tools.
	ArgoQuickStartPostgresURL = "https://github.com/argoproj/argo-workflows/releases/download/" + ArgoVersion + "/quick-start-postgres.yaml"

	// ArgoServerPort is the port where Argo Server listens.
	ArgoServerPort = 2746
)

// Shared cluster state for all E2E tests.
// Connection mode is determined by E2E_MODE env var at process startup.
// To test both modes in parallel, run separate test processes with different E2E_MODE values.
var (
	sharedCluster     *E2ECluster
	sharedClusterOnce sync.Once
	sharedClusterErr  error
)

// GetConnectionMode returns the connection mode from the E2E_MODE environment variable.
// Defaults to ModeKubernetesAPI if not set or invalid.
func GetConnectionMode() ConnectionMode {
	mode := os.Getenv("E2E_MODE")
	switch ConnectionMode(mode) {
	case ModeArgoServer:
		return ModeArgoServer
	case ModeKubernetesAPI:
		return ModeKubernetesAPI
	default:
		return ModeKubernetesAPI
	}
}

// getProjectRoot returns the project root directory.
func getProjectRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller information")
	}
	// Go up from test/e2e to project root
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// buildBinary builds the MCP server binary and returns the path.
func buildBinary(t *testing.T) string {
	t.Helper()
	projectRoot := getProjectRoot()
	// Use test name to avoid conflicts with parallel test execution
	binaryPath := filepath.Join(projectRoot, "dist", fmt.Sprintf("mcp-for-argo-workflows-e2e-test-%s", t.Name()))

	//nolint:gosec // Building binaries in tests is expected
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/mcp-for-argo-workflows")
	buildCmd.Dir = projectRoot
	buildOutput, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build binary: %s", string(buildOutput))

	t.Cleanup(func() {
		if err := os.Remove(binaryPath); err != nil && !os.IsNotExist(err) {
			t.Logf("Failed to remove test binary: %v", err)
		}
	})

	return binaryPath
}

// E2ECluster represents a test cluster with Argo Workflows installed.
//
//nolint:revive // E2ECluster is clearer than Cluster for this test package
type E2ECluster struct {
	// ArgoClient is a configured Argo client for the cluster.
	ArgoClient *argo.Client

	// container is the k3s container (for cleanup).
	container *k3s.K3sContainer

	// Kubeconfig is the raw kubeconfig content.
	Kubeconfig string

	// KubeconfigPath is the path to the temporary kubeconfig file.
	KubeconfigPath string

	// ArgoNamespace is the namespace where Argo Workflows is installed.
	ArgoNamespace string

	// ConnectionMode indicates how the client connects to Argo Workflows.
	ConnectionMode ConnectionMode

	// ArgoServerURL is the URL to the Argo Server (only set in ModeArgoServer).
	ArgoServerURL string

	// portForwardCmd is the kubectl port-forward process (only set in ModeArgoServer).
	portForwardCmd *exec.Cmd
}

// SetupE2ECluster returns the shared E2E cluster, creating it on first call.
// All tests share the same cluster to speed up test execution.
// The cluster is terminated when the test binary exits.
func SetupE2ECluster(ctx context.Context, t *testing.T) *E2ECluster {
	t.Helper()

	sharedClusterOnce.Do(func() {
		sharedCluster, sharedClusterErr = createSharedCluster(ctx, t)
	})

	require.NoError(t, sharedClusterErr, "Failed to create shared E2E cluster")
	require.NotNil(t, sharedCluster, "Shared cluster is nil")

	return sharedCluster
}

// createSharedCluster creates the shared k3s cluster with Argo Workflows.
func createSharedCluster(ctx context.Context, t *testing.T) (*E2ECluster, error) {
	mode := GetConnectionMode()
	t.Logf("Starting shared k3s container for all E2E tests (mode: %s)...", mode)

	// Start k3s container
	k3sContainer, err := k3s.Run(ctx, "rancher/k3s:v1.31.2-k3s1")
	if err != nil {
		return nil, fmt.Errorf("failed to start k3s container: %w", err)
	}

	// Get kubeconfig from container
	kubeconfig, err := k3sContainer.GetKubeConfig(ctx)
	if err != nil {
		_ = k3sContainer.Terminate(context.Background())
		return nil, fmt.Errorf("failed to get kubeconfig from k3s: %w", err)
	}

	// Write kubeconfig to temp file
	kubeconfigFile, err := os.CreateTemp("", "e2e-kubeconfig-*.yaml")
	if err != nil {
		_ = k3sContainer.Terminate(context.Background())
		return nil, fmt.Errorf("failed to create temp kubeconfig file: %w", err)
	}

	kubeconfigPath := kubeconfigFile.Name()

	_, err = kubeconfigFile.Write(kubeconfig)
	if err != nil {
		_ = os.Remove(kubeconfigPath)
		_ = k3sContainer.Terminate(context.Background())
		return nil, fmt.Errorf("failed to write kubeconfig: %w", err)
	}
	err = kubeconfigFile.Close()
	if err != nil {
		_ = os.Remove(kubeconfigPath)
		_ = k3sContainer.Terminate(context.Background())
		return nil, fmt.Errorf("failed to close kubeconfig file: %w", err)
	}

	t.Logf("Kubeconfig written to: %s", kubeconfigPath)

	// Install Argo Workflows
	t.Log("Installing Argo Workflows...")
	if err := installArgoWorkflowsShared(t, kubeconfigPath, mode); err != nil {
		_ = os.Remove(kubeconfigPath)
		_ = k3sContainer.Terminate(context.Background())
		return nil, fmt.Errorf("failed to install Argo Workflows: %w", err)
	}

	// Wait for Argo controller to be ready
	t.Log("Waiting for Argo controller to be ready...")
	if err := waitForArgoControllerShared(t, kubeconfigPath); err != nil {
		_ = os.Remove(kubeconfigPath)
		_ = k3sContainer.Terminate(context.Background())
		return nil, fmt.Errorf("argo controller not ready: %w", err)
	}

	cluster := &E2ECluster{
		Kubeconfig:     string(kubeconfig),
		KubeconfigPath: kubeconfigPath,
		ArgoNamespace:  ArgoNamespace,
		container:      k3sContainer,
		ConnectionMode: mode,
	}

	// Set up connection based on mode
	if mode == ModeArgoServer {
		// Wait for Argo Server to be ready
		t.Log("Waiting for Argo Server to be ready...")
		if err := waitForArgoServerShared(t, kubeconfigPath); err != nil {
			_ = os.Remove(kubeconfigPath)
			_ = k3sContainer.Terminate(context.Background())
			return nil, fmt.Errorf("argo server not ready: %w", err)
		}

		// Start port-forward to Argo Server
		t.Log("Starting port-forward to Argo Server...")
		portForwardCmd, localPort, err := startPortForward(t, kubeconfigPath)
		if err != nil {
			_ = os.Remove(kubeconfigPath)
			_ = k3sContainer.Terminate(context.Background())
			return nil, fmt.Errorf("failed to start port-forward: %w", err)
		}

		cluster.portForwardCmd = portForwardCmd
		cluster.ArgoServerURL = fmt.Sprintf("localhost:%d", localPort)

		// Note: The port-forward process is intentionally not cleaned up here.
		// Since we use a shared cluster pattern, t.Cleanup() would be scoped to
		// the first test and would kill the port-forward when that test finishes,
		// breaking subsequent tests. The port-forward will be terminated when:
		// 1. The test binary process exits (parent dies)
		// 2. The k3s container is terminated (kills the pod)

		t.Logf("Argo Server available at: %s", cluster.ArgoServerURL)

		// Get a service account token for authentication
		argoToken, err := getArgoServerToken(t, kubeconfigPath)
		if err != nil {
			stopPortForward(portForwardCmd)
			_ = os.Remove(kubeconfigPath)
			_ = k3sContainer.Terminate(context.Background())
			return nil, fmt.Errorf("failed to get argo server token: %w", err)
		}

		// Create Argo client with server mode
		// We patched argo-server to use HTTP (--secure=false) to avoid gRPC ALPN issues
		argoClient, err := argo.NewClient(ctx, &argo.Config{
			ArgoServer:         cluster.ArgoServerURL,
			ArgoToken:          argoToken,
			Namespace:          ArgoNamespace,
			Secure:             false, // We patched argo-server to use HTTP
			InsecureSkipVerify: true,
		})
		if err != nil {
			stopPortForward(portForwardCmd)
			_ = os.Remove(kubeconfigPath)
			_ = k3sContainer.Terminate(context.Background())
			return nil, fmt.Errorf("failed to create Argo client (server mode): %w", err)
		}
		cluster.ArgoClient = argoClient
	} else {
		// Direct Kubernetes API mode
		argoClient, err := argo.NewClient(ctx, &argo.Config{
			Kubeconfig: kubeconfigPath,
			Namespace:  ArgoNamespace,
		})
		if err != nil {
			_ = os.Remove(kubeconfigPath)
			_ = k3sContainer.Terminate(context.Background())
			return nil, fmt.Errorf("failed to create Argo client (kubernetes mode): %w", err)
		}
		cluster.ArgoClient = argoClient
	}

	t.Logf("Shared E2E cluster setup complete (mode: %s)", mode)

	return cluster, nil
}

// installArgoWorkflowsShared installs Argo Workflows in the k3s cluster (non-fatal version).
// Uses quick-start-postgres.yaml for argo-server mode (archive support) and quick-start-minimal.yaml
// for kubernetes mode (no archive sidecar that breaks logs tests).
func installArgoWorkflowsShared(t *testing.T, kubeconfigPath string, mode ConnectionMode) error {
	t.Helper()

	// Select manifest based on mode
	var manifestURL string
	if mode == ModeArgoServer {
		manifestURL = ArgoQuickStartPostgresURL
		t.Log("Using quick-start-postgres.yaml for archive support")
	} else {
		manifestURL = ArgoQuickStartMinimalURL
		t.Log("Using quick-start-minimal.yaml")
	}

	// Create the argo namespace first (quick-start manifest expects it to exist)
	//nolint:gosec // Using kubectl in tests is expected
	nsCmd := exec.Command("kubectl", "create", "namespace", ArgoNamespace)
	nsCmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
	output, err := nsCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create argo namespace: %s: %w", string(output), err)
	}

	// Download the quick-start manifest to a temp file
	manifestFile, err := os.CreateTemp("", "argo-install-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp manifest file: %w", err)
	}
	manifestPath := manifestFile.Name()

	// Close file before curl writes to it
	err = manifestFile.Close()
	if err != nil {
		return fmt.Errorf("failed to close manifest file: %w", err)
	}

	defer func() {
		_ = os.Remove(manifestPath) //nolint:errcheck // Cleanup is best-effort
	}()

	// Download manifest
	//nolint:gosec // Using curl to download manifests in tests is expected
	downloadCmd := exec.Command("curl", "-sSL", "-o", manifestPath, manifestURL)
	output, err = downloadCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to download Argo quick-start manifest: %s: %w", string(output), err)
	}

	// Apply the manifest with -n argo to set default namespace for resources without explicit namespace.
	// Some resources in quick-start-postgres.yaml (like postgres Deployment/Service) don't have namespace
	// fields, so they need the -n flag to be created in the argo namespace.
	//nolint:gosec // Using kubectl in tests is expected
	applyCmd := exec.Command("kubectl", "apply", "-n", ArgoNamespace, "-f", manifestPath)
	applyCmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
	output, err = applyCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply Argo manifest: %s: %w", string(output), err)
	}
	t.Logf("kubectl apply output: %s", string(output))

	t.Logf("Argo Workflows %s installed", ArgoVersion)
	return nil
}

// waitForArgoControllerShared waits for the Argo controller deployment to be ready (non-fatal version).
// Uses 5 minute timeout to allow for PostgreSQL initialization with quick-start-postgres.yaml.
func waitForArgoControllerShared(t *testing.T, kubeconfigPath string) error {
	t.Helper()

	// Wait for the argo namespace to exist
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for Argo controller to be ready")
		case <-ticker.C:
			// Check if the deployment is ready
			cmd := exec.Command("kubectl", "wait", "--for=condition=available",
				"--timeout=5s",
				"-n", ArgoNamespace,
				"deployment/workflow-controller")
			cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
			output, err := cmd.CombinedOutput()

			if err == nil {
				t.Log("Argo controller is ready")
				return nil
			}

			// Log the error but continue waiting
			t.Logf("Waiting for Argo controller... %s", string(output))
		}
	}
}

// waitForArgoServerShared waits for the Argo Server deployment to be ready (non-fatal version).
func waitForArgoServerShared(t *testing.T, kubeconfigPath string) error {
	t.Helper()

	// Wait for PostgreSQL to be ready first (required by quick-start-postgres.yaml)
	// PostgreSQL must be running before argo-server can connect to it
	t.Log("Waiting for PostgreSQL to be ready...")
	if err := waitForDeploymentAvailable(t, kubeconfigPath, "postgres"); err != nil {
		t.Log("PostgreSQL deployment not found or not ready (unexpected with quick-start-postgres.yaml)")
		// Continue anyway - postgres might not exist if using minimal install
	} else {
		t.Log("PostgreSQL is ready")
	}

	// Now wait for the argo-server deployment to be available
	if err := waitForDeploymentAvailable(t, kubeconfigPath, "argo-server"); err != nil {
		return fmt.Errorf("timeout waiting for Argo Server initial deployment: %w", err)
	}
	t.Log("Argo Server initial deployment is ready")

	// Patch the argo-server deployment to disable TLS (use HTTP instead of HTTPS)
	// This avoids gRPC ALPN issues with newer grpc-go versions
	// We need to patch both:
	// 1. Add --secure=false to the container args
	// 2. Change the readinessProbe scheme from HTTPS to HTTP
	t.Log("Patching Argo Server to use HTTP (disable TLS)...")
	//nolint:gosec // Using kubectl in tests is expected
	patchCmd := exec.Command("kubectl", "patch", "deployment", "argo-server",
		"-n", ArgoNamespace,
		"--type=json",
		"-p", `[
			{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--secure=false"},
			{"op": "replace", "path": "/spec/template/spec/containers/0/readinessProbe/httpGet/scheme", "value": "HTTP"}
		]`)
	patchCmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
	patchOutput, patchErr := patchCmd.CombinedOutput()
	if patchErr != nil {
		t.Logf("Warning: Failed to patch argo-server (may already be patched): %s", string(patchOutput))
	} else {
		t.Log("Argo Server patched to use HTTP")
	}

	// Force delete old pods to speed up rollout (k3s can be slow to terminate pods)
	t.Log("Force deleting old argo-server pods to speed up rollout...")
	//nolint:gosec // Using kubectl in tests is expected
	deletePodsCmd := exec.Command("kubectl", "delete", "pods",
		"-n", ArgoNamespace,
		"-l", "app=argo-server",
		"--grace-period=0",
		"--force")
	deletePodsCmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
	deleteOutput, _ := deletePodsCmd.CombinedOutput()
	t.Logf("Deleted old pods: %s", string(deleteOutput))

	// Wait for the new deployment to be ready
	t.Log("Waiting for new Argo Server pods to be ready...")
	if err := waitForDeploymentAvailable(t, kubeconfigPath, "argo-server"); err != nil {
		return fmt.Errorf("failed to wait for argo-server after patch: %w", err)
	}

	// Verify the patch was applied by checking the container args
	//nolint:gosec // Using kubectl in tests is expected
	verifyCmd := exec.Command("kubectl", "get", "deployment", "argo-server",
		"-n", ArgoNamespace,
		"-o", "jsonpath={.spec.template.spec.containers[0].args}")
	verifyCmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
	verifyOutput, _ := verifyCmd.CombinedOutput()
	t.Logf("Argo Server container args after patch: %s", string(verifyOutput))

	// Also check the readinessProbe scheme
	//nolint:gosec // Using kubectl in tests is expected
	probeCmd := exec.Command("kubectl", "get", "deployment", "argo-server",
		"-n", ArgoNamespace,
		"-o", "jsonpath={.spec.template.spec.containers[0].readinessProbe.httpGet.scheme}")
	probeCmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
	probeOutput, _ := probeCmd.CombinedOutput()
	t.Logf("Argo Server readinessProbe scheme after patch: %s", string(probeOutput))

	t.Log("Argo Server is ready (HTTP mode)")
	return nil
}

// waitForDeploymentAvailable waits for a deployment to be available.
// Uses 5 minute timeout to allow for PostgreSQL initialization with quick-start-postgres.yaml.
func waitForDeploymentAvailable(t *testing.T, kubeconfigPath, deploymentName string) error {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	attempts := 0
	for {
		select {
		case <-ctx.Done():
			// Get final deployment status for debugging
			//nolint:gosec // Using kubectl in tests is expected
			statusCmd := exec.Command("kubectl", "get", "deployment", deploymentName,
				"-n", ArgoNamespace, "-o", "wide")
			statusCmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
			statusOutput, _ := statusCmd.CombinedOutput()
			t.Logf("Final deployment status for %s:\n%s", deploymentName, string(statusOutput))

			// Get pods status
			//nolint:gosec // Using kubectl in tests is expected
			podsCmd := exec.Command("kubectl", "get", "pods",
				"-n", ArgoNamespace, "-l", "app="+deploymentName, "-o", "wide")
			podsCmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
			podsOutput, _ := podsCmd.CombinedOutput()
			t.Logf("Pods for %s:\n%s", deploymentName, string(podsOutput))

			return fmt.Errorf("timeout waiting for deployment %s", deploymentName)
		case <-ticker.C:
			attempts++
			//nolint:gosec // Using kubectl in tests is expected
			cmd := exec.Command("kubectl", "wait", "--for=condition=available",
				"--timeout=5s",
				"-n", ArgoNamespace,
				"deployment/"+deploymentName)
			cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
			output, err := cmd.CombinedOutput()

			if err == nil {
				return nil
			}

			// Log progress every 6th attempt (30 seconds)
			if attempts%6 == 0 {
				t.Logf("Still waiting for deployment %s (attempt %d): %s", deploymentName, attempts, string(output))
			}
		}
	}
}

// startPortForward starts a kubectl port-forward to the Argo Server and returns the command and local port.
func startPortForward(t *testing.T, kubeconfigPath string) (*exec.Cmd, int, error) {
	t.Helper()

	// First, check that the argo-server pod is ready
	//nolint:gosec // Using kubectl in tests is expected
	checkPodCmd := exec.Command("kubectl", "get", "pods", "-n", ArgoNamespace,
		"-l", "app=argo-server", "-o", "jsonpath={.items[0].status.phase}")
	checkPodCmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
	podStatus, _ := checkPodCmd.CombinedOutput()
	t.Logf("Argo Server pod status: %s", string(podStatus))

	// Check what port the service is using
	//nolint:gosec // Using kubectl in tests is expected
	checkSvcCmd := exec.Command("kubectl", "get", "svc", "argo-server", "-n", ArgoNamespace,
		"-o", "jsonpath={.spec.ports[0].port}")
	checkSvcCmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
	svcPort, _ := checkSvcCmd.CombinedOutput()
	t.Logf("Argo Server service port: %s", string(svcPort))

	// Find an available port by binding to :0
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find available port: %w", err)
	}
	localPort := listener.Addr().(*net.TCPAddr).Port
	// Close the listener so kubectl can use the port
	if err := listener.Close(); err != nil {
		return nil, 0, fmt.Errorf("failed to close listener: %w", err)
	}

	t.Logf("Using local port %d for port-forward", localPort)

	//nolint:gosec // Using kubectl in tests is expected
	cmd := exec.Command("kubectl", "port-forward",
		"-n", ArgoNamespace,
		"svc/argo-server",
		fmt.Sprintf("%d:%d", localPort, ArgoServerPort))
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)

	// Capture stdout and stderr for debugging
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the port-forward in the background
	if err := cmd.Start(); err != nil {
		return nil, 0, fmt.Errorf("failed to start port-forward: %w", err)
	}

	// Wait for the port-forward to be ready by checking if we can connect
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			stderrOutput := stderr.String()
			stdoutOutput := stdout.String()
			stopPortForward(cmd)
			return nil, 0, fmt.Errorf("timeout waiting for port-forward to be ready (stdout: %s, stderr: %s)", stdoutOutput, stderrOutput)
		case <-ticker.C:
			// Try to connect to the port to verify it's ready
			// We patched argo-server to use HTTP (--secure=false)
			//nolint:gosec // Using curl in tests is expected
			checkCmd := exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}",
				"--max-time", "2",
				fmt.Sprintf("http://localhost:%d/api/v1/info", localPort))
			output, err := checkCmd.CombinedOutput()
			httpStatus := string(output)
			if err == nil && (httpStatus == "200" || httpStatus == "401" || httpStatus == "403") {
				// 200, 401, or 403 means the server is responding
				t.Logf("Port-forward is ready (HTTP status: %s)", httpStatus)
				return cmd, localPort, nil
			}
			t.Logf("Waiting for port-forward... (status: %s, err: %v, stdout: %s, stderr: %s)", httpStatus, err, stdout.String(), stderr.String())
		}
	}
}

// stopPortForward stops the kubectl port-forward process.
func stopPortForward(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

// getArgoServerToken creates a service account token for Argo Server authentication.
func getArgoServerToken(t *testing.T, kubeconfigPath string) (string, error) {
	t.Helper()

	// Create a token for the argo-server service account using kubectl
	//nolint:gosec // Using kubectl in tests is expected
	cmd := exec.Command("kubectl", "create", "token", "argo-server",
		"-n", ArgoNamespace,
		"--duration=1h")
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create token: %w: %s", err, string(output))
	}

	token := strings.TrimSpace(string(output))
	t.Logf("Created Argo Server token (length: %d)", len(token))
	return "Bearer " + token, nil
}

// installArgoWorkflows installs Argo Workflows in the k3s cluster.
func installArgoWorkflows(t *testing.T, kubeconfigPath string, mode ConnectionMode) {
	t.Helper()
	err := installArgoWorkflowsShared(t, kubeconfigPath, mode)
	require.NoError(t, err, "Failed to install Argo Workflows")
}

// waitForArgoController waits for the Argo controller deployment to be ready.
func waitForArgoController(t *testing.T, kubeconfigPath string) {
	t.Helper()
	err := waitForArgoControllerShared(t, kubeconfigPath)
	require.NoError(t, err, "Argo controller not ready")
}

// LoadTestDataFile reads a test data file from the testdata directory.
func LoadTestDataFile(t *testing.T, filename string) string {
	t.Helper()

	projectRoot := getProjectRoot()
	path := filepath.Join(projectRoot, "test", "e2e", "testdata", filename)

	//nolint:gosec // Reading test data files is expected
	data, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read test data file %s", filename)

	return string(data)
}

// WaitForWorkflowPhase polls the workflow status until it reaches one of the expected phases.
// Returns the final phase or fails the test if timeout is reached.
func (c *E2ECluster) WaitForWorkflowPhase(t *testing.T, namespace, name string, timeout time.Duration, phases ...string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	phaseSet := make(map[string]bool)
	for _, p := range phases {
		phaseSet[p] = true
	}

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("Timeout waiting for workflow %s/%s to reach phase %v", namespace, name, phases)
		case <-ticker.C:
			// Get workflow status
			cmd := exec.Command("kubectl", "get", "workflow", name,
				"-n", namespace,
				"-o", "jsonpath={.status.phase}")
			cmd.Env = append(os.Environ(), "KUBECONFIG="+c.KubeconfigPath)
			output, err := cmd.CombinedOutput()

			if err != nil {
				t.Logf("Error getting workflow status: %s", string(output))
				continue
			}

			phase := string(output)
			if phaseSet[phase] {
				t.Logf("Workflow %s/%s reached phase: %s", namespace, name, phase)
				return phase
			}

			t.Logf("Workflow %s/%s current phase: %s (waiting for %v)", namespace, name, phase, phases)
		}
	}
}

// WorkflowExists checks if a workflow exists in the cluster.
func (c *E2ECluster) WorkflowExists(t *testing.T, namespace, name string) bool {
	t.Helper()

	cmd := exec.Command("kubectl", "get", "workflow", name, "-n", namespace)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+c.KubeconfigPath)
	err := cmd.Run()

	return err == nil
}

// WorkflowTemplateExists checks if a workflow template exists in the cluster.
func (c *E2ECluster) WorkflowTemplateExists(t *testing.T, namespace, name string) bool {
	t.Helper()

	cmd := exec.Command("kubectl", "get", "workflowtemplate", name, "-n", namespace)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+c.KubeconfigPath)
	err := cmd.Run()

	return err == nil
}

// CronWorkflowExists checks if a cron workflow exists in the cluster.
func (c *E2ECluster) CronWorkflowExists(t *testing.T, namespace, name string) bool {
	t.Helper()

	cmd := exec.Command("kubectl", "get", "cronworkflow", name, "-n", namespace)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+c.KubeconfigPath)
	err := cmd.Run()

	return err == nil
}

// ClusterWorkflowTemplateExists checks if a cluster workflow template exists.
func (c *E2ECluster) ClusterWorkflowTemplateExists(t *testing.T, name string) bool {
	t.Helper()

	cmd := exec.Command("kubectl", "get", "clusterworkflowtemplate", name)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+c.KubeconfigPath)
	err := cmd.Run()

	return err == nil
}

// GetWorkflowPhase returns the current phase of a workflow.
func (c *E2ECluster) GetWorkflowPhase(t *testing.T, namespace, name string) (string, error) {
	t.Helper()

	cmd := exec.Command("kubectl", "get", "workflow", name,
		"-n", namespace,
		"-o", "jsonpath={.status.phase}")
	cmd.Env = append(os.Environ(), "KUBECONFIG="+c.KubeconfigPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return "", fmt.Errorf("failed to get workflow phase: %w: %s", err, string(output))
	}

	return string(output), nil
}
