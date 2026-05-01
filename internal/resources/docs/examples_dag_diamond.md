# DAG Diamond Pattern Example

The classic diamond DAG pattern demonstrates fan-out (one task spawning multiple parallel tasks) and fan-in (multiple tasks converging into one).

## Complete Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-diamond-
spec:
  entrypoint: diamond

  templates:
  - name: diamond
    dag:
      tasks:
      # Task A - the root/start task
      - name: A
        template: echo
        arguments:
          parameters:
          - name: message
            value: "A"

      # Task B - depends on A, runs in parallel with C
      - name: B
        dependencies: [A]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "B"

      # Task C - depends on A, runs in parallel with B
      - name: C
        dependencies: [A]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "C"

      # Task D - depends on both B and C (fan-in)
      - name: D
        dependencies: [B, C]
        template: echo
        arguments:
          parameters:
          - name: message
            value: "D"

  # Simple echo template used by all tasks
  - name: echo
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:latest
      command: [echo]
      args: ["{{inputs.parameters.message}}"]
```

## Execution Flow

```text
       A (Start)
      / \
     /   \
    B     C  (Parallel execution)
     \   /
      \ /
       D (Fan-in - waits for both B and C)
```

## Key Concepts

- **dag**: Defines tasks with explicit dependencies instead of sequential steps
- **dependencies**: Array of task names that must complete before this task starts
- **Parallel Execution**: Tasks B and C run simultaneously after A completes
- **Fan-in**: Task D waits for both B and C to complete before starting

## Common Variations

### Passing Data Through DAG

```yaml
- name: diamond-with-data
  dag:
    tasks:
    - name: generate
      template: gen-data

    - name: process-a
      dependencies: [generate]
      template: process
      arguments:
        parameters:
        - name: input
          value: "{{tasks.generate.outputs.parameters.data}}"

    - name: process-b
      dependencies: [generate]
      template: process
      arguments:
        parameters:
        - name: input
          value: "{{tasks.generate.outputs.parameters.data}}"

    - name: combine
      dependencies: [process-a, process-b]
      template: merge
      arguments:
        parameters:
        - name: result-a
          value: "{{tasks.process-a.outputs.parameters.result}}"
        - name: result-b
          value: "{{tasks.process-b.outputs.parameters.result}}"
```

### Complex Dependencies

```yaml
dag:
  tasks:
  - name: A
    template: task

  - name: B
    dependencies: [A]
    template: task

  - name: C
    dependencies: [A]
    template: task

  - name: D
    dependencies: [A]
    template: task

  # E waits for B and C (but not D)
  - name: E
    dependencies: [B, C]
    template: task

  # F waits for D and E
  - name: F
    dependencies: [D, E]
    template: task
```

### Conditional Dependencies

```yaml
- name: conditional-task
  dependencies: [check]
  template: cleanup
  # Only run if the check task succeeded
  when: "{{tasks.check.outputs.parameters.status}} == success"
```

## Benefits of DAG over Steps

- **Explicit Dependencies**: Clear visualization of task relationships
- **Maximum Parallelism**: Automatically runs tasks in parallel when possible
- **Flexible Execution**: Easy to add/remove dependencies without restructuring

## Next Steps

- See `argo://examples/multi-step` for sequential execution patterns
- Explore `argo://examples/conditionals` for conditional task execution
- Check `argo://examples/artifacts` for passing files between DAG tasks
