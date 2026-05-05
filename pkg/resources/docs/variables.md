# Argo Workflows Variables Reference

Complete reference guide for template variables available in Argo Workflows. Variables use the `{{variable}}` syntax and are evaluated at runtime.

## Input Parameters

Access input parameters passed to a template.

| Variable | Description |
|----------|-------------|
| `{{inputs.parameters.NAME}}` | Value of input parameter NAME |
| `{{inputs.parameters}}` | JSON object of all input parameters |

```yaml
templates:
- name: my-template
  inputs:
    parameters:
    - name: message
    - name: count
  container:
    image: alpine
    command: [echo]
    args: ["{{inputs.parameters.message}} (count: {{inputs.parameters.count}})"]
```

## Output Variables

Access outputs from templates.

| Variable | Description |
|----------|-------------|
| `{{outputs.parameters.NAME}}` | Value of output parameter NAME |
| `{{outputs.result}}` | stdout result from script/container templates |
| `{{outputs.exitCode}}` | Exit code from the container |

```yaml
outputs:
  parameters:
  - name: result
    valueFrom:
      path: /tmp/result.txt
  - name: status
    valueFrom:
      expression: "outputs.result == 'success' ? 'completed' : 'failed'"
```

## Step Outputs (Steps Template)

Reference outputs from previous steps within a steps template.

| Variable | Description |
|----------|-------------|
| `{{steps.STEP_NAME.outputs.parameters.NAME}}` | Parameter from step STEP_NAME |
| `{{steps.STEP_NAME.outputs.result}}` | stdout from step STEP_NAME |
| `{{steps.STEP_NAME.outputs.exitCode}}` | Exit code from step STEP_NAME |
| `{{steps.STEP_NAME.outputs.artifacts.NAME}}` | Artifact reference from step |
| `{{steps.STEP_NAME.ip}}` | IP address of the step's pod |
| `{{steps.STEP_NAME.status}}` | Status of step (Succeeded, Failed, etc.) |
| `{{steps.STEP_NAME.id}}` | Node ID of the step |
| `{{steps.STEP_NAME.startedAt}}` | Timestamp when step started |
| `{{steps.STEP_NAME.finishedAt}}` | Timestamp when step finished |

```yaml
templates:
- name: sequential-steps
  steps:
  - - name: generate
      template: generate-data
  - - name: process
      template: process-data
      arguments:
        parameters:
        - name: data
          value: "{{steps.generate.outputs.result}}"
      when: "{{steps.generate.status}} == Succeeded"
```

## Task Outputs (DAG Template)

Reference outputs from dependency tasks within a DAG template.

| Variable | Description |
|----------|-------------|
| `{{tasks.TASK_NAME.outputs.parameters.NAME}}` | Parameter from task TASK_NAME |
| `{{tasks.TASK_NAME.outputs.result}}` | stdout from task TASK_NAME |
| `{{tasks.TASK_NAME.outputs.exitCode}}` | Exit code from task TASK_NAME |
| `{{tasks.TASK_NAME.outputs.artifacts.NAME}}` | Artifact reference from task |
| `{{tasks.TASK_NAME.ip}}` | IP address of the task's pod |
| `{{tasks.TASK_NAME.status}}` | Status of task |
| `{{tasks.TASK_NAME.id}}` | Node ID of the task |
| `{{tasks.TASK_NAME.startedAt}}` | Timestamp when task started |
| `{{tasks.TASK_NAME.finishedAt}}` | Timestamp when task finished |

```yaml
templates:
- name: dag-example
  dag:
    tasks:
    - name: task-a
      template: generate
    - name: task-b
      dependencies: [task-a]
      template: process
      arguments:
        parameters:
        - name: input
          value: "{{tasks.task-a.outputs.parameters.output}}"
```

## Workflow Variables

Global workflow-level variables available anywhere in the workflow.

| Variable | Description |
|----------|-------------|
| `{{workflow.name}}` | Workflow name |
| `{{workflow.namespace}}` | Workflow namespace |
| `{{workflow.uid}}` | Workflow UID |
| `{{workflow.serviceAccountName}}` | Service account name |
| `{{workflow.parameters.NAME}}` | Workflow-level parameter value |
| `{{workflow.outputs.parameters.NAME}}` | Workflow output parameter (in onExit) |
| `{{workflow.outputs.artifacts.NAME}}` | Workflow output artifact (in onExit) |
| `{{workflow.annotations.KEY}}` | Workflow annotation value |
| `{{workflow.labels.KEY}}` | Workflow label value |
| `{{workflow.creationTimestamp}}` | Workflow creation time |
| `{{workflow.creationTimestamp.RFC3339}}` | Creation time in RFC3339 format |
| `{{workflow.duration}}` | Workflow duration in seconds |
| `{{workflow.priority}}` | Workflow priority |
| `{{workflow.status}}` | Workflow status (in onExit handlers) |
| `{{workflow.failures}}` | JSON list of failed nodes (in onExit) |

```yaml
spec:
  arguments:
    parameters:
    - name: environment
      value: "prod"

  templates:
  - name: any-template
    container:
      image: alpine
      command: [echo]
      args:
      - "Running in {{workflow.namespace}}"
      - "Workflow: {{workflow.name}}"
      - "Environment: {{workflow.parameters.environment}}"
```

## Pod and Node Variables

Information about the running pod and Kubernetes node.

| Variable | Description |
|----------|-------------|
| `{{pod.name}}` | Name of the pod |
| `{{node.name}}` | Name of the Kubernetes node |

```yaml
templates:
- name: node-info
  container:
    image: alpine
    command: [sh, -c]
    args:
    - |
      echo "Pod: {{pod.name}}"
      echo "Node: {{node.name}}"
```

## Loop Variables

Variables available inside loop iterations (withItems, withParam, withSequence).

| Variable | Description |
|----------|-------------|
| `{{item}}` | Current item value (simple values) |
| `{{item.FIELD}}` | Field from current item (objects) |

### Simple Items

```yaml
- name: loop-simple
  steps:
  - - name: process
      template: worker
      arguments:
        parameters:
        - name: value
          value: "{{item}}"
      withItems:
      - "one"
      - "two"
      - "three"
```

### Object Items

```yaml
- name: loop-objects
  steps:
  - - name: process
      template: worker
      arguments:
        parameters:
        - name: name
          value: "{{item.name}}"
        - name: version
          value: "{{item.version}}"
      withItems:
      - { name: "app-a", version: "1.0" }
      - { name: "app-b", version: "2.0" }
```

### Sequence Variables

```yaml
- name: loop-sequence
  steps:
  - - name: process
      template: worker
      arguments:
        parameters:
        - name: index
          value: "{{item}}"
      withSequence:
        count: "5"
        start: "1"
```

## Retry Variables

Variables available when using retry strategies.

| Variable | Description |
|----------|-------------|
| `{{retries}}` | Current retry attempt number (0-indexed) |

```yaml
templates:
- name: retry-template
  retryStrategy:
    limit: 3
    backoff:
      duration: "1s"
      factor: 2
  container:
    image: alpine
    command: [sh, -c]
    args:
    - |
      echo "Attempt number: {{retries}}"
      # Your logic here
```

## Artifact Variables

Reference artifacts in templates.

| Variable | Description |
|----------|-------------|
| `{{inputs.artifacts.NAME.path}}` | Path where input artifact is mounted |
| `{{outputs.artifacts.NAME.path}}` | Path where output artifact should be written |

## Template Variables

Variables related to the template itself.

| Variable | Description |
|----------|-------------|
| `{{template.name}}` | Name of the current template |
| `{{template.baseName}}` | Base name (without random suffix) |

## Special Values

### Null and Empty Handling

| Variable | Description |
|----------|-------------|
| `{{=}}` | Expression evaluation (see expressions doc) |

### Escaping

To include literal `{{` or `}}` in output:
- Use `{{='{{'}}` to output `{{`
- Use `{{='}}'}}` to output `}}`

## Variable Resolution Order

When multiple sources could provide a variable:

1. Explicit template inputs
2. Step/task outputs (within scope)
3. Workflow-level parameters
4. Default values

## Common Gotchas

### String vs Number

All template variables are strings. For numeric operations:

```yaml
# Use expression syntax for arithmetic
value: "{{=inputs.parameters.count + 1}}"
```

### Undefined Variables

Referencing undefined variables results in an empty string unless you use `default`:

```yaml
value: "{{inputs.parameters.optional | default('fallback')}}"
```

### Nested References

Variables cannot contain other variables:

```yaml
# This will NOT work
value: "{{inputs.parameters.{{item.param_name}}}}"

# Instead, use expressions
value: "{{=inputs.parameters[item.param_name]}}"
```

## See Also

- `argo://docs/expressions` - Expression syntax for complex variable manipulation
- `argo://docs/parameters` - Parameter definition and passing
- `argo://docs/outputs` - Capturing and using outputs
