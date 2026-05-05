# Argo Workflows Parameters Reference

Comprehensive guide to defining, passing, and using parameters in Argo Workflows.

## Parameter Types

### Workflow Arguments (Global Parameters)

Parameters defined at the workflow level, accessible from any template using `{{workflow.parameters.NAME}}`.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: example-
spec:
  arguments:
    parameters:
    - name: environment
      value: "dev"
    - name: log-level
      value: "info"

  templates:
  - name: any-template
    container:
      image: alpine
      command: [echo]
      args: ["Env: {{workflow.parameters.environment}}"]
```

### Template Input Parameters

Parameters a template explicitly declares as inputs.

```yaml
templates:
- name: my-template
  inputs:
    parameters:
    - name: required-param        # Required: no default
    - name: optional-param        # Optional: has default
      value: "default-value"
  container:
    image: alpine
    command: [echo]
    args: ["{{inputs.parameters.required-param}} / {{inputs.parameters.optional-param}}"]
```

## Parameter Definition Options

### Basic Parameter

```yaml
parameters:
- name: message
  value: "Hello"
```

### Parameter with Description

```yaml
parameters:
- name: environment
  value: "dev"
  description: "Target deployment environment (dev, staging, prod)"
```

### Parameter with Enum Constraint

```yaml
parameters:
- name: log-level
  value: "info"
  enum:
  - "debug"
  - "info"
  - "warning"
  - "error"
```

### Parameter from ConfigMap

```yaml
parameters:
- name: config-data
  valueFrom:
    configMapKeyRef:
      name: my-config
      key: settings.json
```

### Parameter from Secret

```yaml
parameters:
- name: api-key
  valueFrom:
    secretKeyRef:
      name: my-secret
      key: api-key
```

### Parameter with Default Expression

```yaml
parameters:
- name: timestamp
  valueFrom:
    default: "{{workflow.creationTimestamp}}"
```

### Global Name for Workflow Output

```yaml
parameters:
- name: result
  globalName: workflow-result  # Exported as workflow output
  valueFrom:
    path: /tmp/result.txt
```

## Parameter Passing Patterns

### From Workflow to Entry Template

```yaml
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: message
      value: "Hello"

  templates:
  - name: main
    inputs:
      parameters:
      - name: message       # Automatically receives workflow argument
    container:
      image: alpine
      command: [echo]
      args: ["{{inputs.parameters.message}}"]
```

### Between Templates (Steps)

```yaml
templates:
- name: orchestrator
  steps:
  - - name: step-one
      template: generator
  - - name: step-two
      template: processor
      arguments:
        parameters:
        - name: input
          value: "{{steps.step-one.outputs.parameters.data}}"
```

### Between Templates (DAG)

```yaml
templates:
- name: dag-orchestrator
  dag:
    tasks:
    - name: task-a
      template: generator
    - name: task-b
      dependencies: [task-a]
      template: processor
      arguments:
        parameters:
        - name: input
          value: "{{tasks.task-a.outputs.parameters.data}}"
```

### Loop Variables to Parameters

```yaml
steps:
- - name: process-items
    template: worker
    arguments:
      parameters:
      - name: item-name
        value: "{{item.name}}"
      - name: item-value
        value: "{{item.value}}"
    withItems:
    - { name: "first", value: "100" }
    - { name: "second", value: "200" }
```

## Submission Time Parameters

### CLI Override

```bash
# Override single parameter
argo submit workflow.yaml -p environment=prod

# Override multiple parameters
argo submit workflow.yaml -p environment=prod -p log-level=debug

# Using parameter file
argo submit workflow.yaml --parameter-file params.yaml
```

### Parameter File Format

```yaml
# params.yaml
environment: prod
log-level: debug
config: |
  {
    "database": "production-db",
    "cache": true
  }
```

### API Submission

```yaml
# When submitting via API
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: my-workflow-
spec:
  arguments:
    parameters:
    - name: environment
      value: "prod"        # This overrides any default
```

## Output Parameters

### From File

```yaml
outputs:
  parameters:
  - name: result
    valueFrom:
      path: /tmp/result.txt
```

### From Expression

```yaml
outputs:
  parameters:
  - name: status
    valueFrom:
      expression: "steps.check.status == 'Succeeded' ? 'passed' : 'failed'"
```

### From Default with Fallback

```yaml
outputs:
  parameters:
  - name: result
    valueFrom:
      path: /tmp/result.txt
      default: "no-result"  # If file doesn't exist
```

### From JSON Path

```yaml
outputs:
  parameters:
  - name: id
    valueFrom:
      path: /tmp/response.json
      jsonPath: '$.data.id'
```

### From Script Result

```yaml
- name: script-template
  script:
    image: python:alpine
    command: [python]
    source: |
      print("computed-value")
  outputs:
    parameters:
    - name: result
      valueFrom:
        path: /dev/stdout
```

## Parameter Value Sources

### Literal Values

```yaml
arguments:
  parameters:
  - name: count
    value: "10"
```

### From Previous Step

```yaml
arguments:
  parameters:
  - name: data
    value: "{{steps.generate.outputs.parameters.result}}"
```

### From Previous Task

```yaml
arguments:
  parameters:
  - name: data
    value: "{{tasks.generate.outputs.parameters.result}}"
```

### From Item (Loop)

```yaml
arguments:
  parameters:
  - name: current
    value: "{{item}}"
```

### From Workflow Parameter

```yaml
arguments:
  parameters:
  - name: env
    value: "{{workflow.parameters.environment}}"
```

### From Supplied (WorkflowTemplates)

```yaml
# WorkflowTemplate definition
spec:
  arguments:
    parameters:
    - name: user-input  # No default - must be supplied at submit time

# Submitting from the template
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: from-template-
spec:
  workflowTemplateRef:
    name: my-template
  arguments:
    parameters:
    - name: user-input
      value: "supplied-value"
```

## JSON Parameters

### Passing JSON Data

```yaml
arguments:
  parameters:
  - name: config
    value: |
      {
        "database": {
          "host": "localhost",
          "port": 5432
        },
        "features": ["a", "b", "c"]
      }
```

### Accessing JSON Fields

```yaml
# Using expressions
value: "{{=fromJson(inputs.parameters.config).database.host}}"

# Using jsonpath
value: "{{=jsonpath(inputs.parameters.config, '$.database.port')}}"
```

### Dynamic JSON from Step Output

```yaml
steps:
- - name: get-config
    template: fetch-config
- - name: use-config
    template: processor
    arguments:
      parameters:
      - name: host
        value: "{{=fromJson(steps.get-config.outputs.result).host}}"
```

## Parameter Validation

### Required Parameters

Parameters without a default value are required:

```yaml
inputs:
  parameters:
  - name: required-field  # No value = required
```

### Enum Validation

```yaml
arguments:
  parameters:
  - name: size
    value: "medium"
    enum:
    - "small"
    - "medium"
    - "large"
```

### Description for Documentation

```yaml
arguments:
  parameters:
  - name: count
    value: "1"
    description: "Number of parallel workers (1-10 recommended)"
```

## Common Patterns

### Environment-Specific Configuration

```yaml
spec:
  arguments:
    parameters:
    - name: env
      value: "dev"
      enum: ["dev", "staging", "prod"]

  templates:
  - name: deploy
    container:
      image: deployer:latest
      env:
      - name: TARGET_ENV
        value: "{{workflow.parameters.env}}"
      - name: REPLICAS
        value: "{{=workflow.parameters.env == 'prod' ? '3' : '1'}}"
```

### Parameterized Resource Names

```yaml
metadata:
  generateName: "{{workflow.parameters.app-name}}-build-"
```

### Conditional Template Selection

```yaml
steps:
- - name: deploy
    template: "deploy-{{workflow.parameters.env}}"
    # Results in deploy-dev, deploy-staging, or deploy-prod
```

### Parameter Aggregation from Parallel Steps

```yaml
steps:
- - name: parallel-work
    template: worker
    arguments:
      parameters:
      - name: item
        value: "{{item}}"
    withItems: ["a", "b", "c"]
- - name: aggregate
    template: collector
    arguments:
      parameters:
      - name: results
        value: "{{steps.parallel-work.outputs.parameters.result}}"
```

## Gotchas and Best Practices

### All Parameters Are Strings

```yaml
# Even numbers are strings
- name: count
  value: "10"    # String "10", not integer 10

# Convert in expressions when needed
value: "{{=asInt(inputs.parameters.count) + 1}}"
```

### Parameter Name Restrictions

- Must be valid DNS label (lowercase, alphanumeric, hyphens)
- Cannot start or end with hyphen
- Maximum 63 characters

### Large Parameters

- Parameters are stored in etcd via the Workflow CR
- Keep parameters under 256KB total
- For large data, use artifacts instead

### Sensitive Data

- Avoid putting secrets directly in parameter values
- Use `valueFrom.secretKeyRef` instead
- Secrets in parameters appear in logs and workflow status

## See Also

- `argo://docs/variables` - Variable reference syntax
- `argo://docs/expressions` - Expression evaluation
- `argo://docs/outputs` - Output parameter capture
- `argo://examples/parameters` - Parameter usage examples
