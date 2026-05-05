# DAG Template Type

Define tasks with explicit dependencies for maximum parallelism.

## Key Fields

- **tasks** (required) - List of tasks to execute
- Each task has:
  - **name** - Task name
  - **template** - Template to execute
  - **dependencies** - Other tasks to wait for

## Example

```yaml
templates:
  - name: diamond
    dag:
      tasks:
        - name: A
          template: echo
          arguments:
            parameters: [{name: message, value: "A"}]
        - name: B
          dependencies: [A]
          template: echo
          arguments:
            parameters: [{name: message, value: "B"}]
        - name: C
          dependencies: [A]
          template: echo
          arguments:
            parameters: [{name: message, value: "C"}]
        - name: D
          dependencies: [B, C]
          template: echo
          arguments:
            parameters: [{name: message, value: "D"}]
```

## When to Use

DAG templates are ideal when tasks have complex dependencies and you want maximum parallelism.

## See Full Documentation

For complete examples and best practices, refer to the Argo Workflows documentation.