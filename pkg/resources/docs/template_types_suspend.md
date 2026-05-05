# Suspend Template Type

Pause workflow execution for approval gates or timed delays.

## Key Fields

- **duration** - How long to suspend (e.g., "10s", "5m", "1h")
  - If omitted, suspends indefinitely until manually resumed

## Example

```yaml
templates:
  - name: approve
    suspend:
      duration: "0"  # Suspend indefinitely until resumed

  - name: delay
    suspend:
      duration: "20s"  # Auto-resume after 20 seconds
```

## When to Use

Suspend templates are ideal for manual approval gates or introducing delays in your workflow.

## See Full Documentation

For complete examples and best practices, refer to the Argo Workflows documentation.