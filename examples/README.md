# Example Configurations

This directory contains example configurations for running MCP for Argo Workflows with various MCP clients and deployment methods.

## MCP Client Configurations

### [Claude Desktop](./claude-desktop/)

Configuration examples for [Claude Desktop](https://claude.ai/download), Anthropic's desktop application.

- Direct Kubernetes API configuration
- Argo Server configuration with token authentication

### [Cursor](./cursor/)

Configuration examples for [Cursor](https://cursor.sh/), the AI-powered code editor.

- MCP server settings for direct Kubernetes access
- Argo Server configuration

## Deployment Examples

### [Kubernetes](./kubernetes/)

Deploy MCP for Argo Workflows as an HTTP/SSE server in Kubernetes.

- Deployment and Service manifests
- RBAC configuration
- Production-ready with health checks and security contexts

### [Docker Compose](./docker-compose/)

Run MCP for Argo Workflows locally using Docker Compose.

- Quick setup for development and testing
- Support for both kubeconfig and Argo Server modes

## Quick Reference

| Client/Method | Transport | Best For |
|---------------|-----------|----------|
| Claude Desktop | stdio | Local development with Claude Desktop |
| Cursor | stdio | Local development with Cursor IDE |
| Kubernetes | HTTP/SSE | Production deployments, remote access |
| Docker Compose | HTTP/SSE | Local testing, development |

## Getting Started

1. Choose your deployment method based on your use case
2. Follow the README in the corresponding directory
3. Adjust configurations as needed for your environment

## Configuration Options

All examples support these configuration options:

| Option | CLI Flag | Environment Variable | Description |
|--------|----------|---------------------|-------------|
| Namespace | `--namespace` | `ARGO_NAMESPACE` | Default Kubernetes namespace |
| Argo Server | `--argo-server` | `ARGO_SERVER` | Argo Server host:port |
| Token | `--argo-token` | `ARGO_TOKEN` | Bearer token for auth |
| Transport | `--transport` | `MCP_TRANSPORT` | `stdio` or `http` |
| HTTP Address | `--http-addr` | `MCP_HTTP_ADDR` | HTTP listen address |
| TLS | `--argo-secure` | `ARGO_SECURE` | Use TLS (default: true) |
| Skip TLS Verify | `--argo-insecure-skip-verify` | `ARGO_INSECURE_SKIP_VERIFY` | Skip cert verification |
| Kubeconfig | `--kubeconfig` | `KUBECONFIG` | Path to kubeconfig |
| Context | `--context` | - | Kubernetes context |

## Need Help?

- See the main [README](../README.md) for complete documentation
- Check [Troubleshooting](../README.md#troubleshooting) for common issues
- Open an issue on [GitHub](https://github.com/pipekit/mcp-for-argo-workflows/issues)
