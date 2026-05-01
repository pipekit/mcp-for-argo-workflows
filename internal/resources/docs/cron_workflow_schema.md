# CronWorkflow CRD Schema

CronWorkflow is a scheduled workflow that runs on a cron schedule, similar to Kubernetes CronJobs but for Argo Workflows.

## API Version and Kind

- **apiVersion**: argoproj.io/v1alpha1
- **kind**: CronWorkflow

## Key Features

1. **Scheduled Execution**: Workflows run automatically on a cron schedule
2. **Workflow Template**: Embeds a complete WorkflowSpec
3. **Concurrency Control**: Manages overlapping workflow executions
4. **Timezone Support**: Schedule in specific timezones
5. **Suspend/Resume**: Can be temporarily disabled without deletion

---

## Structure

A CronWorkflow consists of three main sections:

1. **metadata**: Standard Kubernetes object metadata
2. **spec**: The cron workflow specification (schedule + workflow spec)
3. **status**: Runtime status information (read-only, managed by controller)

---

## Metadata Fields

Standard Kubernetes ObjectMeta fields:

- **name** (string, required): CronWorkflow name
- **namespace** (string): Kubernetes namespace. Defaults to "default" if not specified
- **labels** (map[string]string): Key-value pairs for organizing cron workflows
- **annotations** (map[string]string): Non-identifying metadata

### Common Labels

- **workflows.argoproj.io/cron-workflow**: Auto-added to created Workflows
- Custom labels for categorization and discovery

---

## Spec Fields

### Schedule Configuration

- **schedules** ([]string, recommended): List of cron schedule expressions
  - Modern format supporting multiple schedules (Argo Workflows v3.6+)
  - Format: Standard 5-field cron format (minute, hour, day, month, weekday)
  - Examples:
    - ` + "`" + `["0 9 * * *"]` + "`" + ` - Daily at 9:00 AM
    - ` + "`" + `["0 9 * * *", "0 17 * * *"]` + "`" + ` - Twice daily at 9 AM and 5 PM
    - ` + "`" + `["*/15 * * * *"]` + "`" + ` - Every 15 minutes
    - ` + "`" + `["0 0 1 * *", "0 0 15 * *"]` + "`" + ` - 1st and 15th of month at midnight

- **schedule** (string, deprecated): Single cron schedule expression
  - Legacy format, replaced by ` + "`" + `schedules` + "`" + `
  - Still supported for backward compatibility
  - If both are specified, ` + "`" + `schedules` + "`" + ` takes precedence

- **timezone** (string): IANA timezone for schedule interpretation
  - Default: UTC
  - Examples: "America/New_York", "Europe/London", "Asia/Tokyo"
  - Use ` + "`" + `"Local"` + "`" + ` for server's local timezone (not recommended)

### Concurrency Control

- **concurrencyPolicy** (string): How to handle overlapping workflow executions
  - **"Allow"** (default): Allow concurrent workflows to run
  - **"Forbid"**: Skip new execution if previous is still running
  - **"Replace"**: Terminate running workflow and start new one

### Workflow Specification

- **workflowSpec** (WorkflowSpec, required): Complete workflow specification
  - Same as Workflow.spec
  - See Workflow schema for all available fields
  - Can include templates, arguments, volumes, etc.

### Alternative: Template Reference

Instead of defining templates inline in `workflowSpec`, you can reference an existing template using `workflowSpec.workflowTemplateRef`:

- **workflowSpec.workflowTemplateRef** (WorkflowTemplateRef): Reference to WorkflowTemplate or ClusterWorkflowTemplate
  - **name** (string): Template name
  - **clusterScope** (bool): If true, references ClusterWorkflowTemplate

Note: Within a single `workflowSpec`, specify either inline templates OR `workflowTemplateRef`, not both. `workflowTemplateRef` is a field of `workflowSpec` — not an alternative to it.

### Workflow Metadata

- **workflowMetadata** (ObjectMeta): Metadata to apply to created Workflows
  - **labels** (map[string]string): Labels for created workflows
  - **annotations** (map[string]string): Annotations for created workflows

### History Management

- **successfulJobsHistoryLimit** (int32): Number of successful workflows to keep
  - Default: 3
  - Older workflows are automatically deleted

- **failedJobsHistoryLimit** (int32): Number of failed workflows to keep
  - Default: 1
  - Older workflows are automatically deleted

### Suspension

- **suspend** (bool): If true, prevents new workflow creation
  - Default: false
  - Useful for temporarily disabling without deletion
  - Does not affect already-running workflows

### Starting Deadline

- **startingDeadlineSeconds** (int64): Deadline in seconds for starting missed schedules
  - If a schedule is missed and the deadline passes, skip that execution
  - Useful for avoiding backlog when system is down

---

## Status Fields (Read-Only)

The status section is managed by the controller:

- **active** ([]ObjectReference): Currently running workflows created by this CronWorkflow
- **lastScheduledTime** (Time): When the last workflow was scheduled
- **conditions** ([]Condition): Current conditions
  - **type**: "SubmissionError" indicates scheduling problems

---

## Complete Examples

### Simple Scheduled Workflow

```yaml
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: hello-world-cron
spec:
  schedules:
    - "0 9 * * *"  # Daily at 9:00 AM UTC
  timezone: "America/New_York"
  workflowSpec:
    entrypoint: main
    templates:
    - name: main
      container:
        image: alpine:latest
        command: [echo]
        args: ["Hello, World! The time is $(date)"]
```

### Parameterized Cron Workflow

```yaml
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: daily-report
  labels:
    app: reporting
spec:
  schedules:
    - "0 0 * * *"  # Daily at midnight
  timezone: "UTC"
  concurrencyPolicy: "Forbid"  # Don't run if previous is still running
  successfulJobsHistoryLimit: 7  # Keep 7 days of history
  failedJobsHistoryLimit: 3

  workflowSpec:
    entrypoint: generate-report
    arguments:
      parameters:
      - name: report-date
        value: "{{workflow.creationTimestamp}}"

    templates:
    - name: generate-report
      inputs:
        parameters:
        - name: report-date
      steps:
      - - name: fetch-data
          template: fetch
          arguments:
            parameters:
            - name: date
              value: "{{inputs.parameters.report-date}}"

      - - name: process
          template: process-data
          arguments:
            artifacts:
            - name: input
              from: "{{steps.fetch-data.outputs.artifacts.data}}"

      - - name: send-email
          template: send-report
          arguments:
            artifacts:
            - name: report
              from: "{{steps.process.outputs.artifacts.report}}"

    - name: fetch
      inputs:
        parameters:
        - name: date
      outputs:
        artifacts:
        - name: data
          path: /tmp/data.csv
      container:
        image: postgres:15
        command: [sh, -c]
        args:
        - |
          psql -c "COPY (SELECT * FROM events WHERE date = '{{inputs.parameters.date}}') TO '/tmp/data.csv' CSV"

    - name: process-data
      inputs:
        artifacts:
        - name: input
          path: /tmp/input.csv
      outputs:
        artifacts:
        - name: report
          path: /tmp/report.html
      script:
        image: python:3.11
        command: [python]
        source: |
          import pandas as pd
          df = pd.read_csv('/tmp/input.csv')
          # Generate report
          with open('/tmp/report.html', 'w') as f:
              f.write(df.to_html())

    - name: send-report
      inputs:
        artifacts:
        - name: report
          path: /tmp/report.html
      container:
        image: alpine:latest
        command: [sh, -c]
        args: ["cat /tmp/report.html && echo 'Report sent'"]
```

### Using WorkflowTemplate Reference

```yaml
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: scheduled-backup
spec:
  schedules:
    - "0 2 * * *"  # 2 AM daily
  timezone: "UTC"
  concurrencyPolicy: "Forbid"

  workflowSpec:
    # Reference existing template instead of defining templates inline
    workflowTemplateRef:
      name: backup-template

    # Pass arguments to the referenced template
    arguments:
      parameters:
      - name: backup-type
        value: "full"
      - name: retention-days
        value: "30"
```

### High-Frequency Monitoring

```yaml
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: health-check
spec:
  schedules:
    - "*/5 * * * *"  # Every 5 minutes
  concurrencyPolicy: "Forbid"
  successfulJobsHistoryLimit: 12  # Keep 1 hour of history
  failedJobsHistoryLimit: 6

  workflowSpec:
    entrypoint: check-health
    templates:
    - name: check-health
      steps:
      - - name: check-api
          template: http-check
          arguments:
            parameters:
            - name: url
              value: "https://api.example.com/health"

      - - name: check-database
          template: db-check

      - - name: alert
          template: send-alert
          when: "{{steps.check-api.status}} == Failed || {{steps.check-database.status}} == Failed"

    - name: http-check
      inputs:
        parameters:
        - name: url
      container:
        image: curlimages/curl:latest
        command: [curl]
        args: ["-f", "{{inputs.parameters.url}}"]

    - name: db-check
      container:
        image: postgres:15
        command: [pg_isready]
        args: ["-h", "postgres.default.svc.cluster.local"]

    - name: send-alert
      container:
        image: alpine:latest
        command: [sh, -c]
        args: ["echo 'ALERT: Health check failed!'"]
```

---

## Cron Schedule Format

### Standard 5-Field Format (minute hour day month weekday)

```
┌───────────── minute (0 - 59)
│ ┌───────────── hour (0 - 23)
│ │ ┌───────────── day of month (1 - 31)
│ │ │ ┌───────────── month (1 - 12)
│ │ │ │ ┌───────────── day of week (0 - 6) (Sunday to Saturday)
│ │ │ │ │
* * * * *
```

### Special Characters

- ` + "`" + `*` + "`" + ` - Any value
- ` + "`" + `,` + "`" + ` - Value list separator (e.g., "1,15" = 1st and 15th)
- ` + "`" + `-` + "`" + ` - Range (e.g., "1-5" = 1 through 5)
- ` + "`" + `/` + "`" + ` - Step values (e.g., "*/10" = every 10 units)

### Common Examples

- ` + "`" + `"0 0 * * *"` + "`" + ` - Daily at midnight
- ` + "`" + `"0 */6 * * *"` + "`" + ` - Every 6 hours
- ` + "`" + `"0 9 * * 1-5"` + "`" + ` - Weekdays at 9 AM
- ` + "`" + `"0 0 1,15 * *"` + "`" + ` - 1st and 15th of month
- ` + "`" + `"0 0 * * 0"` + "`" + ` - Every Sunday at midnight
- ` + "`" + `"*/30 * * * *"` + "`" + ` - Every 30 minutes

---

## Timezone Handling

### IANA Timezone Examples

- **UTC**: ` + "`" + `"UTC"` + "`" + ` - Coordinated Universal Time
- **US**: ` + "`" + `"America/New_York"` + "`" + `, ` + "`" + `"America/Chicago"` + "`" + `, ` + "`" + `"America/Los_Angeles"` + "`" + `
- **Europe**: ` + "`" + `"Europe/London"` + "`" + `, ` + "`" + `"Europe/Paris"` + "`" + `, ` + "`" + `"Europe/Berlin"` + "`" + `
- **Asia**: ` + "`" + `"Asia/Tokyo"` + "`" + `, ` + "`" + `"Asia/Shanghai"` + "`" + `, ` + "`" + `"Asia/Singapore"` + "`" + `

### Daylight Saving Time

The cron controller automatically handles DST transitions:
- Spring forward: Schedule may skip an hour
- Fall back: Schedule may run twice in one day

---

## Concurrency Policies

### Allow (Default)

```yaml
spec:
  concurrencyPolicy: "Allow"
  schedules:
    - "*/1 * * * *"  # Every minute
```

Multiple workflows can run simultaneously. Use when:
- Workflows are short and don't overlap
- Concurrent execution is safe
- You want maximum throughput

### Forbid

```yaml
spec:
  concurrencyPolicy: "Forbid"
  schedules:
    - "*/5 * * * *"  # Every 5 minutes
```

Skip new execution if previous is still running. Use when:
- Concurrent execution could cause conflicts
- Workflows might take longer than the schedule interval
- You want to prevent resource exhaustion

### Replace

```yaml
spec:
  concurrencyPolicy: "Replace"
  schedules:
    - "0 * * * *"  # Every hour
```

Terminate running workflow and start new one. Use when:
- Latest data is more important than completing old runs
- Long-running workflows should be interrupted
- You want guaranteed fresh starts

---

## Managing CronWorkflows

### Suspend/Resume

Temporarily disable:

```bash
# Suspend
kubectl patch cronworkflow my-cron -p '{"spec":{"suspend":true}}'

# Resume
kubectl patch cronworkflow my-cron -p '{"spec":{"suspend":false}}'
```

Or set in YAML:

```yaml
spec:
  suspend: true
  schedule: "0 9 * * *"
  # ... rest of spec
```

### View Created Workflows

```bash
# List workflows created by a CronWorkflow
kubectl get workflows -l workflows.argoproj.io/cron-workflow=my-cron

# Watch for new workflows
kubectl get workflows -l workflows.argoproj.io/cron-workflow=my-cron --watch
```

### Manual Trigger

Manually create a workflow from the CronWorkflow:

```bash
# Using Argo CLI
argo submit --from cronworkflow/my-cron
```

---

## Required Fields Summary

**Minimum viable CronWorkflow:**
- metadata.name
- spec.schedules (or spec.schedule for legacy support)
- spec.workflowSpec.entrypoint OR spec.workflowTemplateRef.name
- spec.workflowSpec.templates (if using workflowSpec)

---

## Best Practices

1. **Use Forbid for Long Workflows**: Set concurrencyPolicy to "Forbid" if workflows might overlap
2. **Set History Limits**: Configure successfulJobsHistoryLimit and failedJobsHistoryLimit to prevent unbounded growth
3. **Specify Timezone**: Always set timezone explicitly to avoid confusion
4. **Use WorkflowTemplateRef**: Reference templates for easier maintenance and reuse
5. **Monitor Failed Runs**: Set up alerts for failed workflows
6. **Add Resource Limits**: Prevent runaway resource usage in scheduled workflows
7. **Use Workflow Metadata**: Add labels/annotations to created workflows for tracking
8. **Test Schedule**: Use [crontab.guru](https://crontab.guru) to verify cron expressions
9. **Handle Timezones Carefully**: Be aware of DST transitions in your timezone
10. **Set Starting Deadline**: Use startingDeadlineSeconds for non-critical schedules

---

## Common Use Cases

- **Batch Processing**: Nightly ETL jobs, data aggregation
- **Reporting**: Daily/weekly report generation
- **Backups**: Regular database or application backups
- **Monitoring**: Periodic health checks, metric collection
- **Cleanup**: Removing old data, pruning resources
- **Synchronization**: Regular data sync between systems
- **Testing**: Periodic integration or smoke tests
- **Certification Renewal**: Certificate rotation workflows

---

## Troubleshooting

### Schedule Not Running

1. Check if suspended: ` + "`" + `kubectl get cronwf my-cron -o jsonpath='{.spec.suspend}'` + "`" + `
2. Verify schedule syntax: Use crontab.guru or similar
3. Check controller logs: ` + "`" + `kubectl logs -n argo deploy/workflow-controller` + "`" + `
4. Verify timezone: Ensure timezone is valid IANA timezone

### Workflows Piling Up

1. Set concurrencyPolicy to "Forbid" or "Replace"
2. Adjust history limits
3. Increase workflow timeout (activeDeadlineSeconds)
4. Review workflow performance

### Missing Executions

1. Check startingDeadlineSeconds
2. Verify controller was running during scheduled time
3. Check for SubmissionError conditions in status

---

## References

- [Argo Workflows Documentation](https://argo-workflows.readthedocs.io/)
- [Cron Expression Format](https://en.wikipedia.org/wiki/Cron)
- [Crontab Guru](https://crontab.guru) - Cron expression testing
- [IANA Time Zone Database](https://www.iana.org/time-zones)
- [Workflow Schema](argo://schemas/workflow) - For workflowSpec field documentation