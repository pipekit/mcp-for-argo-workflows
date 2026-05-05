# Exit Handlers Example

Demonstrates OnExit handlers for cleanup, notifications, and status-specific actions that run after workflow completion.

## Complete Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: exit-handler-
spec:
  entrypoint: main

  # Define exit handler - runs after workflow completes (success or failure)
  onExit: exit-handler

  templates:
  # Main workflow logic
  - name: main
    steps:
    - - name: step1
        template: task-a

    - - name: step2
        template: task-b

  # Task that succeeds
  - name: task-a
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo 'Task A completed successfully'"]

  # Task that might fail
  - name: task-b
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          echo "Task B running..."
          # Simulate random failure
          if [ $((RANDOM % 2)) -eq 0 ]; then
            echo "Task B succeeded"
            exit 0
          else
            echo "Task B failed"
            exit 1
          fi

  # Exit handler - always runs
  - name: exit-handler
    steps:
    # Always run cleanup
    - - name: cleanup
        template: cleanup-resources

    # Send notification based on status
    - - name: notify-success
        template: send-success-notification
        when: "{{workflow.status}} == Succeeded"

    - - name: notify-failure
        template: send-failure-notification
        when: "{{workflow.status}} != Succeeded"

    # Archive logs
    - - name: archive-logs
        template: archive-workflow-logs

  # Cleanup template
  - name: cleanup-resources
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          echo "Cleaning up resources..."
          echo "Workflow: {{workflow.name}}"
          echo "Status: {{workflow.status}}"
          echo "Cleanup completed"

  # Success notification
  - name: send-success-notification
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          echo "SUCCESS: Workflow {{workflow.name}} completed successfully"
          echo "Duration: {{workflow.duration}} seconds"
          # In real scenario: send to Slack/email/webhook
          # curl -X POST https://hooks.slack.com/... -d '{"text":"Success"}'

  # Failure notification
  - name: send-failure-notification
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          echo "FAILURE: Workflow {{workflow.name}} failed"
          echo "Status: {{workflow.status}}"
          echo "Message: {{workflow.message}}"
          # In real scenario: send alert
          # curl -X POST https://api.pagerduty.com/...

  # Archive logs
  - name: archive-workflow-logs
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          echo "Archiving logs for {{workflow.name}}"
          echo "Created: {{workflow.creationTimestamp}}"
          echo "Duration: {{workflow.duration}}s"
```

## Key Concepts

- **onExit**: Specifies a template to run after workflow completes
- **Always Runs**: Exit handler executes regardless of workflow success/failure
- **Status Variables**: Access workflow status in exit handler
- **Conditional Steps**: Use `when` to run steps based on workflow outcome

## Workflow Status Variables

```yaml
# Available in exit handlers:
{{workflow.name}}              # Workflow name
{{workflow.namespace}}         # Namespace
{{workflow.status}}            # Succeeded, Failed, Error
{{workflow.message}}           # Error message if failed
{{workflow.duration}}          # Total duration in seconds
{{workflow.creationTimestamp}} # When workflow started
{{workflow.uid}}               # Unique identifier
{{workflow.parameters.X}}      # Access workflow parameters
```

## Common Patterns

### Cleanup Resources

```yaml
- name: cleanup
  onExit: cleanup-handler

- name: cleanup-handler
  steps:
  - - name: delete-temp-files
      template: cleanup-temp

  - - name: release-locks
      template: release-distributed-locks

  - - name: notify-completion
      template: send-notification
```

### Status-Based Actions

```yaml
- name: status-handler
  steps:
  # On success: Deploy
  - - name: deploy
      template: deploy-application
      when: "{{workflow.status}} == Succeeded"

  # On failure: Rollback
  - - name: rollback
      template: rollback-changes
      when: "{{workflow.status}} == Failed"

  # On error: Alert
  - - name: alert
      template: send-alert
      when: "{{workflow.status}} == Error"
```

### Multi-Step Exit Handler

```yaml
- name: comprehensive-exit
  steps:
  # Step 1: Collect metrics
  - - name: collect-metrics
      template: gather-workflow-metrics

  # Step 2: Cleanup (parallel)
  - - name: cleanup-storage
      template: cleanup-pvc
    - name: cleanup-secrets
      template: cleanup-temp-secrets
    - name: cleanup-configmaps
      template: cleanup-temp-configs

  # Step 3: Notify based on status
  - - name: notify
      template: notification-handler

  # Step 4: Update external systems
  - - name: update-cicd
      template: update-cicd-status
```

## Nested Exit Handlers

```yaml
spec:
  entrypoint: main
  onExit: workflow-exit

  templates:
  - name: main
    steps:
    - - name: deploy-stage
        template: deploy-with-exit
        onExit: stage-exit  # Stage-specific exit handler

  # Workflow-level exit handler
  - name: workflow-exit
    container:
      image: alpine
      command: [echo, "Workflow {{workflow.status}}"]

  # Step-level exit handler
  - name: stage-exit
    container:
      image: alpine
      command: [echo, "Stage completed"]

  - name: deploy-with-exit
    container:
      image: alpine
      command: [echo, "Deploying..."]
```

## Exit Handler with DAG

```yaml
- name: dag-exit-handler
  dag:
    tasks:
    # Parallel cleanup tasks
    - name: cleanup-resources
      template: cleanup

    - name: send-metrics
      template: metrics

    # Final notification depends on both
    - name: notify
      dependencies: [cleanup-resources, send-metrics]
      template: notification
```

## Real-World Examples

### CI/CD Pipeline Cleanup

```yaml
- name: cicd-exit-handler
  steps:
  # Clean up Docker images
  - - name: cleanup-docker
      template: docker-cleanup

  # Update build status
  - - name: update-github-status
      template: github-status
      arguments:
        parameters:
        - name: status
          value: "{{workflow.status}}"

  # Send metrics to monitoring system
  - - name: send-metrics
      template: push-metrics
      arguments:
        parameters:
        - name: duration
          value: "{{workflow.duration}}"
        - name: result
          value: "{{workflow.status}}"

- name: github-status
  inputs:
    parameters:
    - name: status
  container:
    image: curlimages/curl:latest
    command: [sh, -c]
    args:
      - |
        curl -X POST \
          -H "Authorization: token ${GITHUB_TOKEN}" \
          https://api.github.com/repos/owner/repo/statuses/${COMMIT_SHA} \
          -d '{"state":"{{inputs.parameters.status}}","context":"argo-workflow"}'
```

### Resource Cleanup

```yaml
- name: resource-cleanup
  steps:
  # Delete temporary PVC
  - - name: delete-pvc
      template: kubectl-delete
      arguments:
        parameters:
        - name: resource
          value: "pvc/temp-storage-{{workflow.name}}"

  # Delete temporary secrets
  - - name: delete-secrets
      template: kubectl-delete
      arguments:
        parameters:
        - name: resource
          value: "secret/temp-creds-{{workflow.name}}"

  # Remove temporary files from S3
  - - name: cleanup-s3
      template: s3-cleanup
      arguments:
        parameters:
        - name: prefix
          value: "temp/{{workflow.name}}/"
```

### Conditional Notifications

```yaml
- name: smart-notifications
  steps:
  # Always send summary
  - - name: send-summary
      template: workflow-summary

  # Only alert on failure
  - - name: send-alert
      template: pagerduty-alert
      when: "{{workflow.status}} == Failed"

  # Only update dashboard on success
  - - name: update-dashboard
      template: dashboard-update
      when: "{{workflow.status}} == Succeeded"

  # Send detailed report for certain workflows
  - - name: detailed-report
      template: generate-report
      when: "{{workflow.parameters.send-report}} == true"
```

## Exit Handler with Artifacts

```yaml
- name: exit-with-artifacts
  steps:
  # Collect all workflow outputs
  - - name: collect-outputs
      template: gather-artifacts

  # Upload to archive
  - - name: archive
      template: upload-archive
      arguments:
        artifacts:
        - name: workflow-outputs
          from: "{{steps.collect-outputs.outputs.artifacts.bundle}}"

- name: gather-artifacts
  outputs:
    artifacts:
    - name: bundle
      path: /tmp/outputs
  container:
    image: alpine
    command: [sh, -c]
    args:
      - |
        mkdir -p /tmp/outputs
        echo "Workflow: {{workflow.name}}" > /tmp/outputs/metadata.txt
        echo "Status: {{workflow.status}}" >> /tmp/outputs/metadata.txt
```

## Best Practices

1. **Always Include Exit Handlers**: For cleanup and notifications
2. **Keep Exit Handlers Simple**: They should be reliable and fast
3. **Handle All Status Types**: Success, failure, and error
4. **Use Conditional Steps**: Don't waste resources on unnecessary actions
5. **Idempotent Cleanup**: Exit handlers should be safe to run multiple times
6. **Timeout Exit Handlers**: Set reasonable timeouts
7. **Don't Fail in Exit Handlers**: Exit handler failure doesn't fail workflow

## Exit Handler Behavior

### Workflow States

- **Succeeded**: All steps completed successfully
- **Failed**: One or more steps failed
- **Error**: Kubernetes error (not step failure)

### Exit Handler Execution

```yaml
# Exit handler runs even if:
- Workflow fails
- Workflow is terminated
- Workflow times out
- Workflow is deleted (if hooks configured)
```

## Debugging Exit Handlers

```bash
# View workflow with exit handler status
argo get workflow-name

# Check exit handler logs
argo logs workflow-name -c exit-handler

# See exit handler node status
kubectl get workflow workflow-name -o jsonpath='{.status.nodes}'
```

## Common Issues

### Exit Handler Not Running

```yaml
# Issue: Typo in template name
onExit: exit-handlerr  # Wrong name

# Fix:
onExit: exit-handler  # Correct template name
```

### Exit Handler Failing

```yaml
# Exit handlers should not fail the workflow
# But they can timeout
- name: exit-handler
  activeDeadlineSeconds: 300  # Set timeout
  steps:
  - - name: cleanup
      template: safe-cleanup
```

## Next Steps

- See `argo://examples/conditionals` for conditional execution patterns
- Explore `argo://examples/timeout-limits` for deadline configurations
- Check `argo://examples/retries` for retry strategies
