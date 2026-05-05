# Container Template Type

Run containers with specified images, commands, and arguments.

## Key Fields

- **image** (required) - Container image to run
- **command** - Override container entrypoint
- **args** - Arguments to pass to command
- **env** - Environment variables
- **resources** - CPU/memory requests and limits
- **volumeMounts** - Volumes to mount

## Example

```yaml
templates:
  - name: hello
    container:
      image: alpine:latest
      command: [echo]
      args: ["Hello, World!"]
      resources:
        requests:
          memory: "64Mi"
          cpu: "100m"
        limits:
          memory: "128Mi"
          cpu: "200m"
      env:
        - name: MY_VAR
          value: "my-value"
```

## When to Use

Container templates are ideal when you need to run a specific Docker image with custom commands or arguments.

## See Full Documentation

For complete examples and best practices, refer to the Argo Workflows documentation.