# Retries Example

Demonstrates retry strategies for handling transient failures in Argo Workflows.

## Complete Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: retries-
spec:
  entrypoint: main

  templates:
  - name: main
    steps:
    # Step with basic retry
    - - name: flaky-task
        template: flaky-service

    # Step with custom retry strategy
    - - name: api-call
        template: external-api

    # Step with exponential backoff
    - - name: database-operation
        template: db-task

  # Template with basic retry configuration
  - name: flaky-service
    retryStrategy:
      # Retry up to 3 times
      limit: "3"
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          # Simulates a flaky service that fails randomly
          if [ $((RANDOM % 3)) -eq 0 ]; then
            echo "Success!"
            exit 0
          else
            echo "Failed, will retry..."
            exit 1
          fi

  # Template with exponential backoff
  - name: external-api
    retryStrategy:
      limit: "5"
      # Exponential backoff policy
      backoff:
        duration: "10s"      # Initial wait time
        factor: 2            # Multiply by 2 each retry
        maxDuration: "5m"    # Maximum wait time between retries
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["curl -f https://api.example.com/data || exit 1"]

  # Template with conditional retry
  - name: db-task
    retryStrategy:
      limit: "4"
      # Only retry on specific exit codes
      retryPolicy: "OnError"  # Options: Always, OnError, OnFailure, OnTransientError
      backoff:
        duration: "5s"
        factor: 1.5
        maxDuration: "2m"
    container:
      image: postgres:alpine
      command: [sh, -c]
      args:
        - |
          psql -h db.example.com -c "SELECT 1" || exit 1
```

## Key Concepts

- **retryStrategy**: Configuration for retry behavior
- **limit**: Maximum number of retry attempts
- **backoff**: Wait time between retries
- **retryPolicy**: Conditions under which to retry

## Retry Policies

### Available Policies

```yaml
retryStrategy:
  retryPolicy: "OnFailure"  # Default
```

Options:
- **Always**: Retry on any failure (including errors)
- **OnFailure**: Retry when container exits with non-zero code
- **OnError**: Retry on Kubernetes errors (pod scheduling issues, etc.)
- **OnTransientError**: Retry only on transient errors (OOMKilled, etc.)

## Backoff Strategies

### Exponential Backoff

```yaml
retryStrategy:
  limit: "5"
  backoff:
    duration: "10s"       # Start with 10s
    factor: 2             # Double each time: 10s, 20s, 40s, 80s, 160s
    maxDuration: "5m"     # Cap at 5 minutes
```

Retry delays: 10s → 20s → 40s → 80s → 160s

### Fixed Backoff

```yaml
retryStrategy:
  limit: "3"
  backoff:
    duration: "30s"   # Wait 30s between each retry
    factor: 1         # No increase
```

Retry delays: 30s → 30s → 30s

### Linear Backoff

```yaml
retryStrategy:
  limit: "4"
  backoff:
    duration: "10s"   # Start with 10s
    factor: 1.5       # Increase by 50% each time
```

Retry delays: 10s → 15s → 22.5s → 33.75s

## Common Patterns

### Retry with Timeout

```yaml
- name: task-with-timeout
  retryStrategy:
    limit: "3"
    backoff:
      duration: "5s"
      factor: 2
  # Timeout per attempt
  activeDeadlineSeconds: 60  # Each attempt times out after 60s
  container:
    image: alpine
    command: [sh, -c, "sleep 30 && echo 'done'"]
```

### Selective Retry Based on Exit Code

```yaml
- name: selective-retry
  retryStrategy:
    limit: "3"
    # Only retry for transient errors
    retryPolicy: "OnTransientError"
  script:
    image: python:alpine
    command: [python]
    source: |
      import sys
      import random

      result = random.choice([0, 1, 137])  # 0=success, 1=error, 137=OOMKilled
      print(f"Exit code: {result}")
      sys.exit(result)
```

### Retry in Loops

```yaml
steps:
- - name: retry-each-item
    template: process-item
    arguments:
      parameters:
      - name: item
        value: "{{item}}"
    withItems:
    - "item1"
    - "item2"
    - "item3"

- name: process-item
  inputs:
    parameters:
    - name: item
  retryStrategy:
    limit: "2"  # Each item can retry twice
  container:
    image: alpine
    command: [sh, -c]
    args: ["echo 'Processing {{inputs.parameters.item}}'"]
```

### DAG with Retries

```yaml
- name: dag-with-retries
  dag:
    tasks:
    - name: task-a
      template: flaky-task
      # Each task can have its own retry strategy
      retryStrategy:
        limit: "3"
        backoff:
          duration: "10s"

    - name: task-b
      dependencies: [task-a]
      template: another-task
      retryStrategy:
        limit: "2"
```

## Advanced Configurations

### Global Retry Strategy

```yaml
spec:
  entrypoint: main
  # Apply to all templates unless overridden
  retryStrategy:
    limit: "2"
    backoff:
      duration: "5s"
      factor: 2

  templates:
  - name: main
    steps:
    - - name: step1
        template: task  # Inherits global retry strategy

  - name: task
    # Can override global strategy
    retryStrategy:
      limit: "5"  # Override
    container:
      image: alpine
      command: [echo, "hello"]
```

### No Retry

```yaml
- name: no-retry-task
  retryStrategy:
    limit: "0"  # Explicitly disable retries
  container:
    image: alpine
    command: [sh, -c, "exit 1"]
```

### Retry with Status Reporting

```yaml
- name: retry-with-status
  retryStrategy:
    limit: "3"
  script:
    image: python:alpine
    command: [python]
    source: |
      import os
      import sys

      # Check which retry attempt this is
      retry_num = os.getenv('ARGO_RETRY_COUNT', '0')
      print(f"Attempt {int(retry_num) + 1}")

      # Fail first 2 attempts, succeed on 3rd
      if int(retry_num) < 2:
        sys.exit(1)
      else:
        print("Success!")
        sys.exit(0)
```

## Best Practices

1. **Set Appropriate Limits**: Avoid infinite retries, cap at reasonable number
2. **Use Backoff**: Prevent overwhelming external services
3. **Choose Right Policy**: Match retry behavior to failure type
4. **Combine with Timeouts**: Prevent individual attempts from hanging
5. **Monitor Retry Metrics**: Track retry rates to identify systemic issues

## Monitoring Retries

Check retry attempts in workflow status:

```bash
# View workflow with retry information
argo get workflow-name

# Check specific node retry count
kubectl get workflow workflow-name -o jsonpath='{.status.nodes["node-id"].retries}'
```

## Next Steps

- See `argo://examples/timeout-limits` for deadline configurations
- Explore `argo://examples/exit-handlers` for cleanup after failures
- Check `argo://examples/conditionals` for fallback strategies
