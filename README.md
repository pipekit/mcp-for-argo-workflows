# MCP for Argo Workflows

MCP (Model Context Protocol) server for [Argo Workflows](https://argoproj.github.io/argo-workflows/), enabling AI assistants like Claude to interact with Argo Workflows via standardized tools.

This server is based on Argo v4, and will want your semaphores, mutexes and schedules to be pluralised lists.
Using it with v3.5 or earlier may mean you need to argue with it about this.

## What is MCP?

The [Model Context Protocol](https://modelcontextprotocol.io/) is an open standard that allows AI assistants to securely interact with external tools and data sources. This server exposes Argo Workflows operations as MCP tools, enabling AI assistants to:

- Submit and manage workflows
- Monitor workflow status and logs
- Manage workflow templates and cron workflows
- Query and operate on the workflow archive
- Visualize workflow graphs

## Features

### Connection Modes

- **Direct Kubernetes API** — Connect directly to the Kubernetes API using kubeconfig. Best for local development or when Argo Server is not deployed.
- **Argo Server** — Connect via Argo Server for full feature support including workflow archive operations, large workflow support, and centralized authentication.

### Transport Modes

- **stdio** (default) — For local clients like Claude Desktop and Cursor
- **HTTP/SSE** — For remote client connections

### Supported MCP Clients

- [Claude Desktop](https://claude.ai/download)
- [Cursor](https://cursor.sh/)
- [Claude Code](https://claude.ai/code) (Claude's CLI)
- Any MCP-compatible client

## Installation

### Download Pre-built Binaries

Download the latest release from the [GitHub Releases](https://github.com/pipekit/mcp-for-argo-workflows/releases) page.

```bash
# Linux (amd64)
curl -Lo mcp-for-argo-workflows https://github.com/pipekit/mcp-for-argo-workflows/releases/latest/download/mcp-for-argo-workflows-linux-amd64
chmod +x mcp-for-argo-workflows
sudo mv mcp-for-argo-workflows /usr/local/bin/

# macOS (Apple Silicon)
curl -Lo mcp-for-argo-workflows https://github.com/pipekit/mcp-for-argo-workflows/releases/latest/download/mcp-for-argo-workflows-darwin-arm64
chmod +x mcp-for-argo-workflows
sudo mv mcp-for-argo-workflows /usr/local/bin/

# macOS (Intel)
curl -Lo mcp-for-argo-workflows https://github.com/pipekit/mcp-for-argo-workflows/releases/latest/download/mcp-for-argo-workflows-darwin-amd64
chmod +x mcp-for-argo-workflows
sudo mv mcp-for-argo-workflows /usr/local/bin/
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/pipekit/mcp-for-argo-workflows.git
cd mcp-for-argo-workflows

# Build the binary
make build

# The binary is created at bin/mcp-for-argo-workflows
```

### Docker

```bash
docker pull ghcr.io/pipekit/mcp-for-argo-workflows:latest

# Run with kubeconfig mounted
docker run -v ~/.kube:/root/.kube ghcr.io/pipekit/mcp-for-argo-workflows:latest
```

## Quick Start

### Claude Desktop Configuration

Add the following to your Claude Desktop configuration (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS or `%APPDATA%\Claude\claude_desktop_config.json` on Windows):

**Using Direct Kubernetes API:**

```json
{
  "mcpServers": {
    "argo-workflows": {
      "command": "/usr/local/bin/mcp-for-argo-workflows",
      "args": ["--namespace", "argo"]
    }
  }
}
```

**Using Argo Server:**

```json
{
  "mcpServers": {
    "argo-workflows": {
      "command": "/usr/local/bin/mcp-for-argo-workflows",
      "args": [
        "--argo-server", "argo-server.argo:2746",
        "--namespace", "argo"
      ],
      "env": {
        "ARGO_TOKEN": "Bearer eyJhbGciOiJSUzI1NiIs..."
      }
    }
  }
}
```

### Claude Code Configuration

Add to your Claude Code settings (`~/.claude.json`):

```json
{
  "mcpServers": {
    "argo-workflows": {
      "command": "/usr/local/bin/mcp-for-argo-workflows",
      "args": ["--namespace", "argo"]
    }
  }
}
```

### Cursor Configuration

Add to Cursor settings:

```json
{
  "mcp.servers": {
    "argo-workflows": {
      "command": "/usr/local/bin/mcp-for-argo-workflows",
      "args": ["--namespace", "argo"]
    }
  }
}
```

## Configuration

### Configuration Options

| Environment Variable | CLI Flag | Default | Description |
|---------------------|----------|---------|-------------|
| `MCP_TRANSPORT` | `--transport` | `stdio` | MCP transport mode: `stdio` or `http` |
| `MCP_HTTP_ADDR` | `--http-addr` | `:8080` | HTTP listen address (when using HTTP transport) |
| `ARGO_SERVER` | `--argo-server` | | Argo Server host:port (omit for direct K8s API) |
| `ARGO_TOKEN` | `--argo-token` | | Bearer token for Argo Server authentication |
| `ARGO_NAMESPACE` | `--namespace` | `default` | Default namespace for operations |
| `KUBECONFIG` | `--kubeconfig` | | Path to kubeconfig file. Multiple files may be joined with the OS path-list separator (`:` on Unix, `;` on Windows), matching the kubectl convention |
| | `--context` | | Kubeconfig context to use. Defaults to the kubeconfig's `current-context` (CLI only) |
| `ARGO_SECURE` | `--argo-secure` | `true` | Use TLS when connecting to Argo Server |
| `ARGO_INSECURE_SKIP_VERIFY` | `--argo-insecure-skip-verify` | `false` | Skip TLS certificate verification |
| `ARGO_HTTP1` | `--argo-http1` | `false` | Use HTTP/1.1 (REST) instead of gRPC for Argo Server. Required when the server is behind a reverse proxy (e.g. nginx ingress) that does not support gRPC |

**Precedence:** CLI flags > Environment variables > Default values

### Example Configurations

#### Local Development (Direct K8s API)

```bash
# Uses your current kubeconfig context
mcp-for-argo-workflows --namespace argo

# Pin to a specific context (e.g. when KUBECONFIG merges several clusters)
mcp-for-argo-workflows --context eks-internal --namespace argo

# Multiple kubeconfig files, kubectl-style (':' on Unix)
KUBECONFIG=~/.kube/configs/eks.yaml:~/.kube/configs/k3d.yaml \
  mcp-for-argo-workflows --context k3d-pipeline-mono --namespace argo
```

The active context and cluster are logged at startup so you can confirm which
cluster the server is bound to before running any tools.

#### Argo Server with Token Auth

```bash
export ARGO_TOKEN="Bearer $(kubectl get secret -n argo argo-server-token -o jsonpath='{.data.token}' | base64 -d)"
mcp-for-argo-workflows \
  --argo-server argo-server.argo:2746 \
  --namespace argo
```

#### HTTP Transport for Remote Access

```bash
mcp-for-argo-workflows \
  --transport http \
  --http-addr :8080 \
  --namespace argo
```

#### Port-forwarded Argo Server

```bash
# In one terminal
kubectl port-forward svc/argo-server -n argo 2746:2746

# In another terminal
mcp-for-argo-workflows \
  --argo-server localhost:2746 \
  --argo-insecure-skip-verify \
  --namespace argo
```

#### Argo Server behind Reverse Proxy (e.g. nginx ingress)

```bash
mcp-for-argo-workflows \
  --argo-server argo-workflows.example.com:443 \
  --argo-http1 \
  --argo-token "Bearer dummy" \
  --namespace argo
```

## Available Tools

### Workflow Lifecycle

| Tool | Description |
|------|-------------|
| `submit_workflow` | Submit a workflow from a YAML manifest |
| `list_workflows` | List workflows with optional filtering by status/labels |
| `get_workflow` | Get detailed workflow information |
| `delete_workflow` | Delete a workflow |
| `logs_workflow` | Get workflow or pod logs |
| `watch_workflow` | Stream workflow status updates |
| `wait_workflow` | Wait for workflow completion |
| `lint_workflow` | Validate a workflow manifest before submission |

### Workflow Control

| Tool | Description |
|------|-------------|
| `suspend_workflow` | Suspend a running workflow |
| `resume_workflow` | Resume a suspended workflow |
| `stop_workflow` | Stop a workflow (allows exit handlers to run) |
| `terminate_workflow` | Immediately terminate a workflow |
| `retry_workflow` | Retry a failed workflow from the failed step |
| `resubmit_workflow` | Create a new workflow from an existing one |

### Visualisation

| Tool | Description |
|------|-------------|
| `render_workflow_graph` | Render a workflow as Mermaid, ASCII, DOT, or SVG diagram |
| `render_manifest_graph` | Preview workflow structure from YAML without submitting |

### WorkflowTemplates

| Tool | Description |
|------|-------------|
| `list_workflow_templates` | List workflow templates in a namespace |
| `get_workflow_template` | Get workflow template details |
| `create_workflow_template` | Create a workflow template from YAML |
| `delete_workflow_template` | Delete a workflow template |

### ClusterWorkflowTemplates

| Tool | Description |
|------|-------------|
| `list_cluster_workflow_templates` | List cluster-scoped workflow templates |
| `get_cluster_workflow_template` | Get cluster workflow template details |
| `create_cluster_workflow_template` | Create a cluster workflow template from YAML |
| `delete_cluster_workflow_template` | Delete a cluster workflow template |

### CronWorkflows

| Tool | Description |
|------|-------------|
| `list_cron_workflows` | List cron workflows (scheduled workflows) |
| `get_cron_workflow` | Get cron workflow details including schedule |
| `create_cron_workflow` | Create a cron workflow from YAML |
| `delete_cron_workflow` | Delete a cron workflow |
| `suspend_cron_workflow` | Suspend a cron workflow's schedule |
| `resume_cron_workflow` | Resume a suspended cron workflow |

### Archived Workflows (Argo Server only)

| Tool | Description |
|------|-------------|
| `delete_archived_workflow` | Delete a workflow from the archive |
| `resubmit_archived_workflow` | Resubmit an archived workflow |
| `retry_archived_workflow` | Retry a failed archived workflow |

> **Note:** When connected via Argo Server, `list_workflows` and `get_workflow` automatically include archived workflows. Separate list/get tools for archived workflows are not needed.

### Node Operations

| Tool | Description |
|------|-------------|
| `get_workflow_node` | Get details of a specific node within a workflow |

## Usage Examples

### Submitting a Workflow

Ask Claude: "Submit this workflow to the argo namespace"

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: alpine:latest
      command: [echo, "Hello World"]
```

### Monitoring Workflows

- "List all running workflows in the argo namespace"
- "Show me the logs for workflow hello-world-abc123"
- "Wait for workflow hello-world-abc123 to complete"
- "What's the status of workflow hello-world-abc123?"

### Workflow Control

- "Suspend workflow data-pipeline-xyz"
- "Resume the suspended workflow"
- "Retry the failed workflow from where it failed"
- "Stop the workflow gracefully"

### Visualizing Workflows

- "Show me a diagram of workflow complex-dag-123"
- "Render this workflow YAML as a Mermaid diagram"
- "Give me an ASCII visualization of the workflow graph"

### Managing Templates

- "List all workflow templates in the argo namespace"
- "Create a workflow template from this YAML"
- "Delete the workflow template named data-pipeline"

### Working with CronWorkflows

- "Show me all scheduled workflows"
- "Suspend the daily-backup cron workflow"
- "What's the schedule for cron workflow nightly-cleanup?"

## Troubleshooting

### Connection Issues

**"Failed to create Argo client"**

- Verify your kubeconfig is valid: `kubectl cluster-info`
- Check RBAC permissions: `kubectl auth can-i list workflows`
- For Argo Server, verify the server is accessible: `curl https://argo-server.argo:2746/api/v1/info`

**"No workflows found" when workflows exist**

- Check the namespace: workflows are namespace-scoped
- Verify label selectors if filtering

### Authentication Issues

**"Unauthorized" errors with Argo Server**

- Ensure `ARGO_TOKEN` is set correctly
- Tokens may expire; regenerate if needed
- Check token has required RBAC permissions

**Token generation:**

```bash
# For service account token
kubectl create token argo-server -n argo

# Or from a secret
kubectl get secret -n argo argo-server-token -o jsonpath='{.data.token}' | base64 -d
```

### TLS Issues

**"Certificate verification failed"**

For development/testing with self-signed certificates:

```bash
mcp-for-argo-workflows --argo-insecure-skip-verify
```

> **Warning:** Don't use `--argo-insecure-skip-verify` in production.

### Reverse Proxy / gRPC Issues

**"unexpected content-type text/html" or gRPC errors behind nginx ingress**

When your Argo Server is behind a reverse proxy that does not support gRPC (e.g. nginx ingress without gRPC backend-protocol), use HTTP/1.1 mode:

```bash
mcp-for-argo-workflows --argo-http1 --argo-server argo.example.com:443
```

### Debug Logging

The server logs to stderr. For verbose output, check stderr in your MCP client's logs or run manually:

```bash
mcp-for-argo-workflows --namespace argo 2>&1 | tee debug.log
```

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/pipekit/mcp-for-argo-workflows.git
cd mcp-for-argo-workflows

# Install development tools
make tools

# Run all checks (fmt, vet, lint, test)
make all

# Run only tests
make test

# Run only linter
make lint

# Run E2E tests (requires Docker for testcontainers)
make test-e2e
```

### Code Style

- Follow standard Go conventions
- Use `gofmt` and `goimports` for formatting
- Pass `golangci-lint` checks

### Pull Request Process

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Ensure `make all` passes
5. Submit a pull request

## License

Apache License 2.0 - see [LICENSE](LICENSE) for details.
