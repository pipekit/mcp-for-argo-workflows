# Resource Management Example

Demonstrates CPU and memory resource management, pod priority, and optimization strategies for Argo Workflows.

## Complete Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: resource-management-
spec:
  entrypoint: main

  # Workflow-level pod metadata
  podMetadata:
    labels:
      app: resource-demo
      tier: compute

  # Optional: Set priority class for all pods
  priorityClassName: high-priority

  templates:
  - name: main
    steps:
    # Small, lightweight task
    - - name: lightweight-task
        template: small-task

    # Memory-intensive task
    - - name: memory-intensive
        template: large-memory-task

    # CPU-intensive task
    - - name: cpu-intensive
        template: compute-heavy-task

    # Burstable task (requests < limits)
    - - name: burstable-task
        template: burstable

  # Small task with minimal resources
  - name: small-task
    container:
      image: alpine:latest
      command: [echo, "Hello, World!"]
      resources:
        requests:
          memory: "32Mi"
          cpu: "50m"      # 50 millicores = 0.05 CPU
        limits:
          memory: "64Mi"
          cpu: "100m"

  # Memory-intensive task
  - name: large-memory-task
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          echo "Starting memory-intensive task..."
          # Allocate some memory
          dd if=/dev/zero of=/tmp/output bs=1M count=512
          ls -lh /tmp/output
      resources:
        requests:
          memory: "512Mi"
          cpu: "100m"
        limits:
          memory: "1Gi"
          cpu: "500m"

  # CPU-intensive task
  - name: compute-heavy-task
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          echo "Starting CPU-intensive task..."
          # Simulate CPU work
          for i in $(seq 1 10); do
            echo "Iteration $i"
            yes > /dev/null &
            sleep 2
            killall yes
          done
      resources:
        requests:
          memory: "128Mi"
          cpu: "1000m"    # 1 full CPU
        limits:
          memory: "256Mi"
          cpu: "2000m"    # Up to 2 CPUs

  # Burstable task - can use more than requested
  - name: burstable
    container:
      image: alpine:latest
      command: [sh, -c, "echo 'Burstable task' && sleep 10"]
      resources:
        requests:
          memory: "64Mi"
          cpu: "100m"
        limits:
          memory: "256Mi"   # Can burst to 4x memory
          cpu: "500m"       # Can burst to 5x CPU
```

## Key Concepts

- **requests**: Minimum guaranteed resources (used for scheduling)
- **limits**: Maximum resources the container can use
- **QoS Classes**: Guaranteed, Burstable, BestEffort based on requests/limits
- **priorityClassName**: Determines pod scheduling priority

## Resource Units

### CPU

```yaml
resources:
  requests:
    cpu: "100m"   # 100 millicores = 0.1 CPU
    cpu: "1"      # 1 full CPU core
    cpu: "2.5"    # 2.5 CPU cores
```

### Memory

```yaml
resources:
  requests:
    memory: "128Mi"   # 128 Mebibytes
    memory: "1Gi"     # 1 Gibibyte
    memory: "512M"    # 512 Megabytes (decimal)
```

## QoS Classes

### 1. Guaranteed (Best Performance)

```yaml
# Requests = Limits for both CPU and memory
container:
  resources:
    requests:
      memory: "1Gi"
      cpu: "1"
    limits:
      memory: "1Gi"
      cpu: "1"
```

### 2. Burstable (Flexible)

```yaml
# Requests < Limits OR only requests specified
container:
  resources:
    requests:
      memory: "512Mi"
      cpu: "500m"
    limits:
      memory: "1Gi"
      cpu: "2"
```

### 3. BestEffort (Lowest Priority)

```yaml
# No requests or limits specified
container:
  image: alpine
  command: [echo, "hello"]
  # No resources field = BestEffort
```

## Pod Priority

### Using PriorityClass

```yaml
spec:
  # Reference a PriorityClass
  priorityClassName: high-priority
  entrypoint: main

# PriorityClass definition (applied separately)
---
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: high-priority
value: 1000000
globalDefault: false
description: "High priority for critical workflows"
```

### Common Priority Patterns

```yaml
# Production critical workflows
priorityClassName: critical-priority      # value: 1000000

# Regular production workflows
priorityClassName: high-priority          # value: 100000

# Development/testing workflows
priorityClassName: low-priority           # value: 10000

# Background/batch jobs
priorityClassName: best-effort-priority   # value: 0
```

## Resource Optimization Patterns

### Right-Sizing Resources

```yaml
templates:
# Over-provisioned (wasteful)
- name: wasteful
  container:
    image: alpine
    command: [sleep, "10"]
    resources:
      requests:
        memory: "4Gi"    # Only uses 100Mi
        cpu: "2"         # Only uses 0.1 CPU

# Right-sized (efficient)
- name: efficient
  container:
    image: alpine
    command: [sleep, "10"]
    resources:
      requests:
        memory: "128Mi"
        cpu: "100m"
      limits:
        memory: "256Mi"
        cpu: "200m"
```

### Resource Templates for Consistency

```yaml
spec:
  # Define common resource profiles
  templates:
  - name: small-task
    metadata:
      labels:
        size: small
    container:
      image: "{{inputs.parameters.image}}"
      command: "{{inputs.parameters.command}}"
      resources:
        requests:
          memory: "64Mi"
          cpu: "100m"
        limits:
          memory: "128Mi"
          cpu: "200m"

  - name: medium-task
    metadata:
      labels:
        size: medium
    container:
      image: "{{inputs.parameters.image}}"
      command: "{{inputs.parameters.command}}"
      resources:
        requests:
          memory: "256Mi"
          cpu: "500m"
        limits:
          memory: "512Mi"
          cpu: "1"

  - name: large-task
    metadata:
      labels:
        size: large
    container:
      image: "{{inputs.parameters.image}}"
      command: "{{inputs.parameters.command}}"
      resources:
        requests:
          memory: "1Gi"
          cpu: "2"
        limits:
          memory: "2Gi"
          cpu: "4"
```

## GPU Resources

```yaml
- name: gpu-task
  container:
    image: tensorflow/tensorflow:latest-gpu
    command: [python, train.py]
    resources:
      requests:
        memory: "8Gi"
        cpu: "4"
        nvidia.com/gpu: "1"  # Request 1 GPU
      limits:
        memory: "16Gi"
        cpu: "8"
        nvidia.com/gpu: "1"
```

## Node Selection

### Node Affinity

```yaml
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: node.kubernetes.io/instance-type
            operator: In
            values:
            - c5.4xlarge  # CPU-optimized instances
```

### Node Selector

```yaml
spec:
  nodeSelector:
    workload-type: compute-intensive
    availability-zone: us-west-2a
```

### Tolerations

```yaml
spec:
  tolerations:
  - key: "high-memory"
    operator: "Equal"
    value: "true"
    effect: "NoSchedule"
```

## Resource Quotas and Limits

### Template-Level Resource Constraints

```yaml
- name: constrained-workflow
  steps:
  - - name: task1
      template: compute-task
  - - name: task2
      template: compute-task
  # Limit total parallel resource usage
  parallelism: 3  # Only 3 pods at once
```

## Monitoring Resources

### Resource Usage in Status

```bash
# View workflow resource usage
argo get workflow-name

# Check pod resources
kubectl top pod -l workflows.argoproj.io/workflow=workflow-name

# View resource metrics
kubectl describe pod pod-name
```

## Best Practices

1. **Always Set Requests**: Ensures proper scheduling
2. **Set Reasonable Limits**: Prevents OOMKilled and resource exhaustion
3. **Use Guaranteed QoS for Critical Tasks**: Ensures resources are reserved
4. **Profile Your Workloads**: Measure actual usage to right-size
5. **Consider Burstable for Variable Workloads**: Allows efficient resource sharing
6. **Use Priority Classes**: Ensure critical workflows are scheduled first
7. **Monitor and Adjust**: Track resource usage and optimize over time

## Common Issues

### OOMKilled (Out of Memory)

```yaml
# Fix: Increase memory limits
resources:
  limits:
    memory: "2Gi"  # Increase from 1Gi
```

### CPU Throttling

```yaml
# Fix: Increase CPU limits
resources:
  limits:
    cpu: "2"  # Increase from 1
```

### Pod Stuck in Pending

```yaml
# Issue: Insufficient cluster resources
# Fix: Reduce requests or add nodes
resources:
  requests:
    memory: "512Mi"  # Reduce from 4Gi
    cpu: "500m"      # Reduce from 4
```

## Next Steps

- See `argo://examples/timeout-limits` for execution time limits
- Explore `argo://examples/volumes` for storage resources
- Check `argo://examples/retries` for handling resource-related failures
