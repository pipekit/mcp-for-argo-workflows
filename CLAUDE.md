# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MCP (Model Context Protocol) server for Argo Workflows, allowing AI assistants like Claude to interact with Argo Workflows via standardized tools.

**Repository**: `github.com/pipekit/mcp-for-argo-workflows`

## Goals

- Standalone Go binary using the official MCP Go SDK
- Support both stdio and HTTP/SSE transports
- Support both direct Kubernetes API access and Argo Server connections
- Cover all major CLI operations as MCP tools

## Key Dependencies

- `github.com/modelcontextprotocol/go-sdk` — Official MCP Go SDK
- `github.com/argoproj/argo-workflows/v3/pkg/apiclient` — Argo client libraries

## Build Commands

```bash
make all        # Run fmt, vet, lint, test and build the linux/amd64 binary
make build-all  # Cross-compile binaries for all platforms into dist/
make test       # Run tests with race detection and coverage
make lint       # Run golangci-lint
make lint-fix   # Run golangci-lint with auto-fix
make fmt        # Run gofmt and goimports
make vet        # Run go vet
make clean      # Remove build artifacts
```

Binaries land in `dist/` (e.g. `dist/mcp-for-argo-workflows-linux-amd64`). Run `make help` to see all targets.

## Directory Structure

```
cmd/mcp-for-argo-workflows/    # main.go entry point
pkg/                           # Reusable packages — importable by other MCP servers
  argo/                         # Argo client wrapper
  tools/                        # MCP tool implementations
  prompts/                      # MCP prompt implementations
  resources/                    # MCP resource implementations (embedded docs)
internal/                      # Binary-specific glue (not importable externally)
  server/                       # MCP server wiring
  config/                       # CLI flag / env parsing
  version/                      # Build-time version info
```

## Architecture

### Connection Modes

1. **Direct Kubernetes API** — When `ARGO_SERVER` is not set. Uses kubeconfig. Does not support large workflows or workflow archive.
2. **Argo Server** — When `ARGO_SERVER` is set. Full feature support via gRPC/HTTP.

### Transport Modes

- **stdio** (default) — For local clients like Claude Desktop, Cursor
- **HTTP/SSE** — For remote client connections

### Configuration

Environment variables / CLI flags:
- `ARGO_SERVER` / `--argo-server` — Argo Server host:port
- `ARGO_TOKEN` / `--argo-token` — Bearer token for auth
- `ARGO_NAMESPACE` / `--namespace` — Default namespace
- `MCP_TRANSPORT` / `--transport` — `stdio` (default) or `http`
- `MCP_HTTP_ADDR` / `--http-addr` — HTTP listen address (default `:8080`)
- `KUBECONFIG` / `--kubeconfig` — Path to kubeconfig (when not using Argo Server)
- `MCP_MULTI_CONTEXT` / `--multi-context` — Per-call kubeconfig context selection (default `true`; direct K8s + stdio only)
- `MCP_ALLOWED_CONTEXTS` / `--allowed-contexts` — Allowlist of selectable contexts (empty = all)

## MCP Tools

The server exposes these tool categories:

### Workflow Lifecycle
- `submit_workflow`, `list_workflows`, `get_workflow`, `delete_workflow`
- `logs_workflow`, `watch_workflow`, `wait_workflow`

### Workflow Control
- `suspend_workflow`, `resume_workflow`, `stop_workflow`, `terminate_workflow`
- `retry_workflow`, `resubmit_workflow`

### Validation
- `lint_workflow` — Validate Workflow manifests before submission
- `lint_workflow_template` — Validate WorkflowTemplate manifests before creation
- `lint_cluster_workflow_template` — Validate ClusterWorkflowTemplate manifests before creation
- `lint_cron_workflow` — Validate CronWorkflow manifests before creation

### WorkflowTemplates
- `list_workflow_templates`, `get_workflow_template`, `create_workflow_template`, `delete_workflow_template`

### ClusterWorkflowTemplates
- `list_cluster_workflow_templates`, `get_cluster_workflow_template`, `create_cluster_workflow_template`, `delete_cluster_workflow_template`

### CronWorkflows
- `list_cron_workflows`, `get_cron_workflow`, `create_cron_workflow`, `delete_cron_workflow`
- `suspend_cron_workflow`, `resume_cron_workflow`

### Archived Workflows (Argo Server only)
- `list_archived_workflows`, `get_archived_workflow`, `delete_archived_workflow`
- `resubmit_archived_workflow`, `retry_archived_workflow`

### Node Operations
- `get_workflow_node`, `set_workflow_node`

### Multi-Context (direct K8s mode + stdio transport only)
- `list_contexts` — List kubeconfig context names selectable per call
- Cluster-facing tools accept an optional `context` parameter (kubeconfig context name); on by default, disabled with `--multi-context=false`, restricted with `--allowed-contexts`

## Development Notes

- Use `/usr/bin/env bash` for shell scripts (not `/bin/bash`)
- Run the appropriate lint tool before any create/submit operation to validate manifests:
  - `lint_workflow` before `submit_workflow`
  - `lint_workflow_template` before `create_workflow_template`
  - `lint_cluster_workflow_template` before `create_cluster_workflow_template`
  - `lint_cron_workflow` before `create_cron_workflow`
- Archive operations require Argo Server connection (not available in direct K8s mode)
- Use `github.com/stretchr/testify` for test assertions

## Project Tracking

This project uses Linear for task management. Issues are prefixed with `PIP-` (e.g., PIP-5, PIP-10).

## Pull Request Workflow

When creating a PR:

1. **Monitor CI**: After creating a PR, watch for the CI checks to pass
2. **Address review feedback**:
   - Fix any issues reviewers identify
   - For nitpicks/optional suggestions, reply explaining the reasoning if not implementing
3. **Merge on approval**: Once CI is green and any review feedback is addressed, merge using `gh pr merge --squash`
4. **Update Linear**: After merging, update the corresponding Linear issue status to "Done"
