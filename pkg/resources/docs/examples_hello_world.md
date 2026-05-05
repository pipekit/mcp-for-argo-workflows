# Hello World Workflow Example

The simplest possible Argo Workflow - runs a single container that prints "Hello, World!".

## Complete Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  # Entry point - which template to run first
  entrypoint: whalesay

  templates:
  - name: whalesay
    container:
      # Container image to run
      image: docker/whalesay:latest
      # Command to execute in the container
      command: [cowsay]
      # Arguments passed to the command
      args: ["Hello, World!"]
```

## Key Concepts

- **apiVersion**: Always `argoproj.io/v1alpha1` for Workflows
- **kind**: `Workflow` for a single workflow execution
- **generateName**: Generates unique names (e.g., `hello-world-abc123`)
- **entrypoint**: The name of the template to start with
- **templates**: List of template definitions that can be invoked

## Running This Example

```bash
# Submit the workflow
argo submit hello-world.yaml

# Watch the workflow
argo watch hello-world-xxxxx

# Get workflow logs
argo logs hello-world-xxxxx
```

## Common Variations

### Using a Simple Image

```yaml
templates:
- name: hello
  container:
    image: alpine:latest
    command: [echo]
    args: ["Hello, World!"]
```

### With Environment Variables

```yaml
templates:
- name: hello-env
  container:
    image: alpine:latest
    command: [sh, -c]
    args: ["echo Hello, $NAME!"]
    env:
    - name: NAME
      value: "Argo"
```

### With Resource Limits

```yaml
templates:
- name: hello-limited
  container:
    image: alpine:latest
    command: [echo, "Hello, World!"]
    resources:
      limits:
        memory: "128Mi"
        cpu: "200m"
      requests:
        memory: "64Mi"
        cpu: "100m"
```

## Next Steps

- Explore `argo://examples/multi-step` for sequential workflows
- See `argo://examples/parameters` for passing data to workflows
- Check `argo://examples/dag-diamond` for parallel execution
