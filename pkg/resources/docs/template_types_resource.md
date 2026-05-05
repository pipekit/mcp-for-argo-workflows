# Resource Template Type

Create, apply, patch, or delete Kubernetes resources.

## Key Fields

- **action** (required) - create, apply, patch, delete, or get
- **manifest** (required) - Resource YAML/JSON
- **successCondition** - Condition for success
- **failureCondition** - Condition for failure

## Example

```yaml
templates:
  - name: create-configmap
    resource:
      action: create
      manifest: |
        apiVersion: v1
        kind: ConfigMap
        metadata:
          name: my-config
        data:
          key: value
```

## When to Use

Resource templates are ideal for managing Kubernetes resources as part of your workflow.

## See Full Documentation

For complete examples and best practices, refer to the Argo Workflows documentation.