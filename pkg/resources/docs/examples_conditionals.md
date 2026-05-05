# Conditionals Example

Demonstrates conditional execution of steps and tasks using `when` expressions.

## Complete Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: conditionals-
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: environment
      value: "dev"
    - name: run-tests
      value: "true"

  templates:
  - name: main
    steps:
    # Step 1: Always runs - determine environment
    - - name: check-env
        template: check-environment
        arguments:
          parameters:
          - name: env
            value: "{{workflow.parameters.environment}}"

    # Step 2a: Only runs in production
    - - name: prod-security-scan
        template: security-scan
        when: "{{workflow.parameters.environment}} == prod"

    # Step 2b: Only runs in non-production
    - - name: dev-fast-build
        template: fast-build
        when: "{{workflow.parameters.environment}} != prod"

    # Step 3: Conditional based on previous step output
    - - name: run-tests
        template: test-suite
        when: "{{workflow.parameters.run-tests}} == true"

    - - name: conditional-deploy
        template: deploy
        when: "{{steps.run-tests.outputs.parameters.status}} == success"

    # Step 4: Always runs - cleanup
    - - name: cleanup
        template: cleanup-resources

  - name: check-environment
    inputs:
      parameters:
      - name: env
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo 'Environment: {{inputs.parameters.env}}'"]

  - name: security-scan
    container:
      image: alpine:latest
      command: [echo]
      args: ["Running production security scan..."]

  - name: fast-build
    container:
      image: alpine:latest
      command: [echo]
      args: ["Running fast development build..."]

  - name: test-suite
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo 'Running tests...' && echo 'success' > /tmp/status.txt"]
    outputs:
      parameters:
      - name: status
        valueFrom:
          path: /tmp/status.txt

  - name: deploy
    container:
      image: alpine:latest
      command: [echo]
      args: ["Deploying application..."]

  - name: cleanup-resources
    container:
      image: alpine:latest
      command: [echo]
      args: ["Cleaning up resources..."]
```

## Key Concepts

- **when**: Expression that determines if a step/task should execute
- **Comparison Operators**: `==`, `!=`, `>`, `<`, `>=`, `<=`
- **Logical Operators**: `&&` (and), `||` (or), `!` (not)
- **Expression Context**: Access parameters, step outputs, workflow status

## Common When Expressions

### Equality Checks

```yaml
# String equality
when: "{{inputs.parameters.env}} == prod"

# Numeric equality
when: "{{workflow.parameters.version}} == 2"

# Boolean check
when: "{{workflow.parameters.enabled}} == true"
```

### Inequality

```yaml
# Not equal
when: "{{inputs.parameters.status}} != failed"

# Numeric comparison
when: "{{workflow.parameters.count}} > 0"
when: "{{workflow.parameters.replicas}} >= 3"
```

### Logical Operators

```yaml
# AND condition
when: "{{workflow.parameters.env}} == prod && {{workflow.parameters.deploy}} == true"

# OR condition
when: "{{workflow.parameters.env}} == dev || {{workflow.parameters.env}} == test"

# NOT condition
when: "!({{workflow.parameters.skip}} == true)"
```

### Status-Based Conditions

```yaml
# Based on previous step status
when: "{{steps.build.status}} == Succeeded"

# Based on step output
when: "{{steps.test.outputs.parameters.result}} == pass"

# Based on task status (in DAG)
when: "{{tasks.validate.status}} == Succeeded"
```

## Conditionals in DAG

```yaml
- name: conditional-dag
  dag:
    tasks:
    # Always runs
    - name: validate
      template: validation

    # Only runs if validation succeeds
    - name: process
      dependencies: [validate]
      template: processing
      when: "{{tasks.validate.outputs.parameters.valid}} == true"

    # Only runs if validation fails
    - name: notify-failure
      dependencies: [validate]
      template: send-alert
      when: "{{tasks.validate.outputs.parameters.valid}} == false"

    # Runs after process (if it ran)
    - name: deploy
      dependencies: [process]
      template: deployment
```

## Advanced Patterns

### Multiple Conditions

```yaml
- name: complex-condition
  template: task
  when: |
    {{workflow.parameters.env}} == prod &&
    {{steps.tests.outputs.parameters.passed}} == true &&
    {{workflow.parameters.approval}} == granted
```

### Conditional Loops

```yaml
steps:
- - name: process-items
    template: process
    arguments:
      parameters:
      - name: item
        value: "{{item.name}}"
    withItems:
    - name: "item1"
      enabled: "true"
    - name: "item2"
      enabled: "false"
    - name: "item3"
      enabled: "true"
    # Only process enabled items
    when: "{{item.enabled}} == true"
```

### Workflow Parameter Patterns

```yaml
arguments:
  parameters:
  - name: skip-tests
    value: "false"
  - name: skip-deploy
    value: "false"
  - name: force-deploy
    value: "false"

templates:
- name: deploy-step
  template: deploy
  # Deploy if not skipped OR if forced
  when: "{{workflow.parameters.skip-deploy}} == false || {{workflow.parameters.force-deploy}} == true"
```

## Exit Condition Pattern

```yaml
- name: retry-pattern
  steps:
  # Try main task
  - - name: main-task
      template: task

  # Only run fallback if main failed
  - - name: fallback-task
      template: fallback
      when: "{{steps.main-task.status}} != Succeeded"
```

## Common Use Cases

### Environment-Specific Steps

```yaml
# Production-only steps
when: "{{workflow.parameters.environment}} == prod"

# Non-production steps
when: "{{workflow.parameters.environment}} != prod"
```

### Feature Flags

```yaml
# Run new feature if flag is enabled
when: "{{workflow.parameters.feature-flag-new-ui}} == true"
```

### Error Handling

```yaml
# Send notification only on failure
when: "{{steps.build.status}} == Failed"

# Retry alternative approach
when: "{{steps.primary.status}} != Succeeded"
```

### Approval Gates

```yaml
# Wait for manual approval in production
- name: manual-approval
  suspend: {}
  when: "{{workflow.parameters.environment}} == prod"

- name: deploy
  dependencies: [manual-approval]
  template: deployment
```

## Limitations

- Cannot use complex functions in `when` expressions
- Limited to simple comparison and logical operators
- Cannot access external systems in conditions
- Expressions are evaluated at schedule time, not runtime

## Next Steps

- See `argo://examples/loops` for iteration with conditions
- Explore `argo://examples/exit-handlers` for status-based cleanup
- Check `argo://examples/parameters` for passing conditional values
