# Kubernetes Deployment

This directory contains example Kubernetes manifests for deploying MCP for Argo Workflows as a remote HTTP/SSE server.

## Overview

When deployed as a Kubernetes service, the MCP server runs in HTTP/SSE transport mode, allowing remote MCP clients to connect over the network.

## Files

- [`deployment.yaml`](deployment.yaml) - Full deployment with Service and ServiceAccount
- [`rbac.yaml`](rbac.yaml) - RBAC rules for the service account

## Prerequisites

1. A running Kubernetes cluster
2. Argo Workflows installed in the cluster
3. `kubectl` configured to access the cluster

## Quick Start

### Deploy to Kubernetes

```bash
# Create the namespace (if not exists)
kubectl create namespace mcp-argo

# Apply the RBAC configuration
kubectl apply -f rbac.yaml

# Apply the deployment
kubectl apply -f deployment.yaml
```

### Verify the Deployment

```bash
# Check the pod is running
kubectl get pods -n mcp-argo

# Check the service
kubectl get svc -n mcp-argo

# View logs
kubectl logs -n mcp-argo -l app=mcp-for-argo-workflows
```

### Access the Server

**Port-Forward (for testing)**:

```bash
kubectl port-forward -n mcp-argo svc/mcp-for-argo-workflows 8080:8080
```

Then connect your MCP client to `http://localhost:8080`.

**Ingress (for production)**:

Create an Ingress resource to expose the service externally. Example with NGINX Ingress:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: mcp-for-argo-workflows
  namespace: mcp-argo
spec:
  ingressClassName: nginx
  rules:
  - host: mcp-argo.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: mcp-for-argo-workflows
            port:
              number: 8080
```

## Configuration

### Environment Variables

The deployment can be configured via environment variables in `deployment.yaml`:

| Variable | Description | Default |
|----------|-------------|---------|
| `MCP_TRANSPORT` | Transport mode | `http` |
| `MCP_HTTP_ADDR` | HTTP listen address | `:8080` |
| `ARGO_NAMESPACE` | Default namespace | `argo` |
| `ARGO_SERVER` | Argo Server address (optional) | - |
| `ARGO_TOKEN` | Auth token (optional) | - |
| `MCP_READ_ONLY` | Only expose read only tools (optional) | `false` |

### Using Argo Server

To connect via Argo Server instead of direct Kubernetes API:

```yaml
env:
  - name: ARGO_SERVER
    value: "argo-server.argo:2746"
  - name: ARGO_TOKEN
    valueFrom:
      secretKeyRef:
        name: argo-token
        key: token
```

### Resource Limits

Adjust resource limits based on your usage:

```yaml
resources:
  requests:
    memory: "64Mi"
    cpu: "100m"
  limits:
    memory: "256Mi"
    cpu: "500m"
```

## Security Considerations

### RBAC Permissions

The provided `rbac.yaml` uses a **ClusterRole** with full CRUD permissions on Argo Workflows resources. This is intentional as the MCP server needs to:

- Create and submit workflows
- List, get, and watch workflow status
- Delete, retry, and resubmit workflows
- Manage workflow templates and cron workflows

**For read-only deployments**, modify the ClusterRole to only include `get`, `list`, `watch` verbs:

```yaml
rules:
  - apiGroups: ["argoproj.io"]
    resources: ["workflows", "workflowtemplates", "clusterworkflowtemplates", "cronworkflows"]
    verbs: ["get", "list", "watch"]  # Remove create, update, patch, delete
```

**For namespace-scoped access**, replace the ClusterRoleBinding with a RoleBinding in each namespace:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: mcp-for-argo-workflows
  namespace: argo  # Target namespace
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: mcp-for-argo-workflows
subjects:
  - kind: ServiceAccount
    name: mcp-for-argo-workflows
    namespace: mcp-argo
```

### Other Security Recommendations

1. **Network Policy**: Consider adding NetworkPolicy to restrict access.
2. **TLS**: Use TLS termination at the Ingress level for production.
3. **Authentication**: Consider adding authentication middleware for production deployments.
4. **Image Pinning**: Pin to a specific image version or digest for production (see comments in deployment.yaml).

## Troubleshooting

### Pod not starting

```bash
kubectl describe pod -n mcp-argo -l app=mcp-for-argo-workflows
kubectl logs -n mcp-argo -l app=mcp-for-argo-workflows
```

### Permission errors

Check the RBAC configuration and ensure the service account has the required permissions:

```bash
kubectl auth can-i list workflows --as=system:serviceaccount:mcp-argo:mcp-for-argo-workflows -n argo
```

### Connection issues

Verify the service is accessible:

```bash
kubectl run test --rm -it --image=curlimages/curl -- curl http://mcp-for-argo-workflows.mcp-argo:8080/health
```
