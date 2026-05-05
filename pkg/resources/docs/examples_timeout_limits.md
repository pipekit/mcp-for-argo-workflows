# Timeout and Limits Example

Demonstrates timeout configuration at workflow and template levels to prevent runaway executions.

## Complete Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: timeout-limits-
spec:
  entrypoint: main

  # Workflow-level timeout - entire workflow must complete within this time
  activeDeadlineSeconds: 300  # 5 minutes total

  templates:
  - name: main
    steps:
    # Step with template-level timeout
    - - name: quick-task
        template: fast-task

    # Step with longer timeout
    - - name: slow-task
        template: slow-task

    # Step with no timeout (inherits workflow timeout)
    - - name: final-task
        template: final-task

  # Template with 30-second timeout
  - name: fast-task
    activeDeadlineSeconds: 30
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          echo "Starting fast task..."
          sleep 10
          echo "Fast task completed!"

  # Template with 2-minute timeout
  - name: slow-task
    activeDeadlineSeconds: 120
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          echo "Starting slow task..."
          for i in $(seq 1 60); do
            echo "Progress: $i/60"
            sleep 1
          done
          echo "Slow task completed!"

  # Template without explicit timeout (uses workflow-level timeout)
  - name: final-task
    container:
      image: alpine:latest
      command: [echo]
      args: ["Final task - uses workflow timeout"]
```

## Key Concepts

- **activeDeadlineSeconds**: Maximum execution time in seconds
- **Workflow-level timeout**: Applies to entire workflow execution
- **Template-level timeout**: Applies to individual template execution
- **Timeout Hierarchy**: Template timeout takes precedence over workflow timeout

## Timeout Levels

### 1. Workflow-Level Timeout

```yaml
spec:
  # Entire workflow fails if not complete in 1 hour
  activeDeadlineSeconds: 3600
  entrypoint: main
```

### 2. Template-Level Timeout

```yaml
templates:
- name: task-with-timeout
  # This template fails if not complete in 5 minutes
  activeDeadlineSeconds: 300
  container:
    image: alpine
    command: [sleep, "600"]  # Will timeout at 300s
```

### 3. Step-Level Timeout (via template)

```yaml
steps:
- - name: limited-step
    template: task
    # Cannot directly set timeout here, must be in template definition
```

## Common Patterns

### Different Timeouts for Different Stages

```yaml
templates:
- name: pipeline
  steps:
  # Quick validation - 30 seconds
  - - name: validate
      template: validation

  # Build phase - 10 minutes
  - - name: build
      template: build-app

  # Long-running tests - 30 minutes
  - - name: test
      template: integration-tests

- name: validation
  activeDeadlineSeconds: 30
  container:
    image: alpine
    command: [sh, -c, "echo 'Validating...'"]

- name: build-app
  activeDeadlineSeconds: 600
  container:
    image: golang:alpine
    command: [sh, -c, "go build ./..."]

- name: integration-tests
  activeDeadlineSeconds: 1800
  container:
    image: test-runner
    command: [sh, -c, "npm test"]
```

### Timeout with Retries

```yaml
- name: retry-with-timeout
  # Each retry attempt gets 60 seconds
  activeDeadlineSeconds: 60
  retryStrategy:
    limit: "3"
    backoff:
      duration: "10s"
  container:
    image: alpine
    command: [sh, -c]
    args: ["sleep 30 && echo 'done'"]
# Total possible time: 3 attempts × (60s timeout + backoff) ≈ 3-4 minutes
```

### Global Timeout Protection

```yaml
spec:
  # Safety net: workflow cannot run longer than 2 hours
  activeDeadlineSeconds: 7200

  templates:
  - name: main
    steps:
    # Individual steps have their own tighter timeouts
    - - name: step1
        template: task1  # 5 min timeout
    - - name: step2
        template: task2  # 10 min timeout
    - - name: step3
        template: task3  # 15 min timeout

  - name: task1
    activeDeadlineSeconds: 300
    container:
      image: alpine
      command: [echo, "task1"]

  - name: task2
    activeDeadlineSeconds: 600
    container:
      image: alpine
      command: [echo, "task2"]

  - name: task3
    activeDeadlineSeconds: 900
    container:
      image: alpine
      command: [echo, "task3"]
```

## DAG Timeout Behavior

```yaml
- name: dag-with-timeouts
  # DAG template timeout applies to entire DAG execution
  activeDeadlineSeconds: 600
  dag:
    tasks:
    # Task A: 2 minutes
    - name: task-a
      template: task-2min

    # Task B: 3 minutes (runs parallel with C)
    - name: task-b
      dependencies: [task-a]
      template: task-3min

    # Task C: 3 minutes (runs parallel with B)
    - name: task-c
      dependencies: [task-a]
      template: task-3min

    # Task D: must complete before DAG timeout
    - name: task-d
      dependencies: [task-b, task-c]
      template: task-1min

- name: task-2min
  activeDeadlineSeconds: 120
  container:
    image: alpine
    command: [sleep, "60"]

- name: task-3min
  activeDeadlineSeconds: 180
  container:
    image: alpine
    command: [sleep, "90"]

- name: task-1min
  activeDeadlineSeconds: 60
  container:
    image: alpine
    command: [sleep, "30"]
```

## Timeout with Suspend

```yaml
- name: workflow-with-approval
  # Workflow timeout includes suspend time
  activeDeadlineSeconds: 3600
  steps:
  - - name: build
      template: build-task

  # Manual approval with timeout
  - - name: approval
      suspend:
        duration: "1h"  # Auto-resume after 1 hour

  - - name: deploy
      template: deploy-task
```

## Handling Timeout Events

### Cleanup After Timeout

```yaml
spec:
  activeDeadlineSeconds: 300
  entrypoint: main

  # Run cleanup even if workflow times out
  onExit: cleanup

  templates:
  - name: main
    steps:
    - - name: task
        template: long-running-task

  - name: long-running-task
    container:
      image: alpine
      command: [sleep, "600"]  # Will timeout

  - name: cleanup
    container:
      image: alpine
      command: [sh, -c]
      args: ["echo 'Cleaning up after timeout or completion'"]
```

## Pod-Level Timeout vs Template Timeout

```yaml
- name: task-with-pod-timeout
  activeDeadlineSeconds: 120  # Template timeout: 2 minutes
  container:
    image: alpine
    command: [sleep, "180"]
  # Pod will be terminated at 120 seconds
  # Exit status will show timeout/deadline exceeded
```

## Common Time Formats

```yaml
# All times in seconds
activeDeadlineSeconds: 30      # 30 seconds
activeDeadlineSeconds: 300     # 5 minutes
activeDeadlineSeconds: 3600    # 1 hour
activeDeadlineSeconds: 86400   # 1 day
```

## Best Practices

1. **Set Workflow-Level Timeout**: Always have a safety net
2. **Be Generous with Timeouts**: Account for system load and variability
3. **Template-Level Timeouts**: Set tighter timeouts on known operations
4. **Monitor Timeout Rates**: Track how often timeouts occur
5. **Combine with Retries**: Timeouts and retries work together
6. **Consider External Timeouts**: Cloud provider or cluster timeouts may apply

## Troubleshooting Timeouts

```bash
# Check if workflow timed out
argo get workflow-name

# Look for deadline exceeded in status
kubectl get workflow workflow-name -o jsonpath='{.status.message}'

# View node that timed out
argo logs workflow-name --follow
```

## Next Steps

- See `argo://examples/retries` for handling timeouts with retries
- Explore `argo://examples/exit-handlers` for cleanup after timeouts
- Check `argo://examples/resource-management` for pod resource limits
