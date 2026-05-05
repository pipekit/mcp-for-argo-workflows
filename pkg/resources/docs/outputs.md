# Argo Workflows Outputs Reference

Comprehensive guide to capturing and using outputs from workflow templates in Argo Workflows.

## Output Types

Argo Workflows supports three types of outputs:

1. **Output Parameters** - Small text values captured from files, expressions, or stdout
2. **Output Artifacts** - Files or directories stored in artifact repositories
3. **Exit Code** - The container's exit code

## Output Parameters

### From File Path

```yaml
outputs:
  parameters:
  - name: result
    valueFrom:
      path: /tmp/result.txt     # Read entire file as parameter value
```

### From Stdout (result)

Script templates automatically capture stdout:

```yaml
- name: generate
  script:
    image: python:alpine
    command: [python]
    source: |
      print("hello-world")      # This becomes outputs.result
```

Access via:
```yaml
value: "{{steps.generate.outputs.result}}"
```

### From Expression

```yaml
outputs:
  parameters:
  - name: status
    valueFrom:
      expression: "steps.validate.status == 'Succeeded' ? 'passed' : 'failed'"
```

### From JSONPath

Extract specific fields from JSON files:

```yaml
outputs:
  parameters:
  - name: id
    valueFrom:
      path: /tmp/response.json
      jsonPath: '$.data.id'       # Extract just the id field

  - name: all-names
    valueFrom:
      path: /tmp/response.json
      jsonPath: '$.items[*].name' # Extract all names as JSON array
```

### With Default Value

```yaml
outputs:
  parameters:
  - name: result
    valueFrom:
      path: /tmp/result.txt
      default: "no-result"        # Used if file doesn't exist
```

### Global Parameters (Workflow Outputs)

Export a parameter as a workflow-level output:

```yaml
outputs:
  parameters:
  - name: result
    valueFrom:
      path: /tmp/result.txt
    globalName: workflow-result   # Becomes workflow.outputs.parameters.workflow-result
```

Access in exit handler:
```yaml
value: "{{workflow.outputs.parameters.workflow-result}}"
```

## Output Artifacts

### Basic Output Artifact

```yaml
outputs:
  artifacts:
  - name: report
    path: /tmp/report.pdf
```

### Directory Artifact

```yaml
outputs:
  artifacts:
  - name: logs
    path: /var/log/app/           # Entire directory
```

### With Archive Settings

```yaml
outputs:
  artifacts:
  - name: data
    path: /tmp/data/
    archive:
      tar:
        compressionLevel: 6
```

### Global Artifacts (Workflow Outputs)

```yaml
outputs:
  artifacts:
  - name: final-report
    path: /tmp/report.pdf
    globalName: workflow-report   # Becomes workflow.outputs.artifacts.workflow-report
```

## Accessing Outputs

### In Steps Template

```yaml
steps:
- - name: step-one
    template: producer
- - name: step-two
    template: consumer
    arguments:
      parameters:
      - name: data
        value: "{{steps.step-one.outputs.parameters.result}}"
      - name: stdout
        value: "{{steps.step-one.outputs.result}}"
      - name: exitcode
        value: "{{steps.step-one.outputs.exitCode}}"
      artifacts:
      - name: input
        from: "{{steps.step-one.outputs.artifacts.data}}"
```

### In DAG Template

```yaml
dag:
  tasks:
  - name: task-a
    template: producer
  - name: task-b
    dependencies: [task-a]
    template: consumer
    arguments:
      parameters:
      - name: data
        value: "{{tasks.task-a.outputs.parameters.result}}"
      - name: stdout
        value: "{{tasks.task-a.outputs.result}}"
      artifacts:
      - name: input
        from: "{{tasks.task-a.outputs.artifacts.data}}"
```

### In Exit Handlers

```yaml
spec:
  onExit: exit-handler

  templates:
  - name: exit-handler
    container:
      image: alpine
      command: [sh, -c]
      args:
      - |
        echo "Workflow status: {{workflow.status}}"
        echo "Result: {{workflow.outputs.parameters.result}}"
```

## Output Aggregation

### From Parallel Steps

When a step runs multiple times (withItems/withParam), outputs are aggregated:

```yaml
steps:
- - name: parallel-work
    template: worker
    arguments:
      parameters:
      - name: item
        value: "{{item}}"
    withItems: ["a", "b", "c"]
- - name: collect
    template: aggregator
    arguments:
      parameters:
      - name: results
        # Returns JSON array: ["result-a", "result-b", "result-c"]
        value: "{{steps.parallel-work.outputs.parameters.result}}"
```

### From Fan-Out Tasks

```yaml
dag:
  tasks:
  - name: fan-out
    template: worker
    arguments:
      parameters:
      - name: id
        value: "{{item}}"
    withItems: ["1", "2", "3"]
  - name: fan-in
    dependencies: [fan-out]
    template: aggregator
    arguments:
      parameters:
      - name: all-results
        value: "{{tasks.fan-out.outputs.result}}"
```

## Exit Code

Access the container exit code:

```yaml
# In conditional execution
when: "{{steps.check.outputs.exitCode}} == 0"

# As parameter value
arguments:
  parameters:
  - name: exit-status
    value: "{{steps.run.outputs.exitCode}}"
```

## Script Template Outputs

Script templates have implicit outputs:

```yaml
- name: compute
  script:
    image: python:alpine
    command: [python]
    source: |
      import json
      result = {"value": 42, "status": "ok"}
      print(json.dumps(result))

# Access outputs
value: "{{steps.compute.outputs.result}}"           # The JSON string
value: "{{=fromJson(steps.compute.outputs.result).value}}"  # Extract field
```

## Container Template Outputs

Container templates require explicit output definitions:

```yaml
- name: producer
  container:
    image: alpine
    command: [sh, -c]
    args: ["echo 'hello' > /tmp/result.txt"]
  outputs:
    parameters:
    - name: result
      valueFrom:
        path: /tmp/result.txt
```

## Multiple Outputs

### Multiple Parameters

```yaml
outputs:
  parameters:
  - name: status
    valueFrom:
      path: /tmp/status.txt
  - name: count
    valueFrom:
      path: /tmp/count.txt
  - name: summary
    valueFrom:
      expression: "outputs.parameters.status + ': ' + outputs.parameters.count"
```

### Multiple Artifacts

```yaml
outputs:
  artifacts:
  - name: logs
    path: /var/log/
  - name: results
    path: /tmp/results/
  - name: metrics
    path: /tmp/metrics.json
```

### Mixed Parameters and Artifacts

```yaml
outputs:
  parameters:
  - name: record-count
    valueFrom:
      path: /tmp/count.txt
  artifacts:
  - name: processed-data
    path: /tmp/data/
```

## Conditional Outputs

### Optional Output

```yaml
outputs:
  parameters:
  - name: optional-result
    valueFrom:
      path: /tmp/maybe-exists.txt
      default: ""                 # Empty string if file doesn't exist
```

### Output Based on Success

```yaml
outputs:
  parameters:
  - name: result
    valueFrom:
      expression: |
        steps.main.status == 'Succeeded'
          ? steps.main.outputs.parameters.data
          : 'FAILED'
```

## Workflow Outputs

Export outputs at workflow level for external access:

```yaml
spec:
  templates:
  - name: main
    steps:
    - - name: final
        template: producer

    outputs:
      parameters:
      - name: workflow-result
        valueFrom:
          parameter: "{{steps.final.outputs.parameters.result}}"
      artifacts:
      - name: workflow-artifact
        from: "{{steps.final.outputs.artifacts.data}}"
```

Or using globalName:

```yaml
outputs:
  parameters:
  - name: result
    valueFrom:
      path: /tmp/result.txt
    globalName: final-result      # workflow.outputs.parameters.final-result
```

## Output Validation

### Check Output Exists

```yaml
when: "{{steps.check.outputs.parameters.result}} != ''"
```

### Validate JSON Output

```yaml
# Using expression to check structure
when: "{{=fromJson(steps.fetch.outputs.result).success == true}}"
```

## Common Patterns

### Chain Processing Results

```yaml
steps:
- - name: fetch
    template: data-fetcher
- - name: transform
    template: transformer
    arguments:
      parameters:
      - name: raw-data
        value: "{{steps.fetch.outputs.result}}"
- - name: store
    template: storer
    arguments:
      parameters:
      - name: processed-data
        value: "{{steps.transform.outputs.result}}"
```

### Error Message Capture

```yaml
- name: may-fail
  container:
    image: alpine
    command: [sh, -c]
    args:
    - |
      if ! some-command; then
        echo "Error: command failed" > /tmp/error.txt
        exit 1
      fi
      echo "success" > /tmp/result.txt
  outputs:
    parameters:
    - name: result
      valueFrom:
        path: /tmp/result.txt
        default: ""
    - name: error
      valueFrom:
        path: /tmp/error.txt
        default: ""
```

### Status with Details

```yaml
- name: job
  script:
    image: python:alpine
    command: [python]
    source: |
      import json
      import sys

      result = {
        "status": "success",
        "records_processed": 1000,
        "errors": []
      }
      print(json.dumps(result))

# Consumer can parse details:
value: "{{=fromJson(steps.job.outputs.result).records_processed}}"
```

## Gotchas

### Output Files Must Exist

Unless using `default`, output files must exist when the container exits:

```yaml
# This will fail if /tmp/result.txt doesn't exist
outputs:
  parameters:
  - name: result
    valueFrom:
      path: /tmp/result.txt

# Safe version:
outputs:
  parameters:
  - name: result
    valueFrom:
      path: /tmp/result.txt
      default: "no-output"
```

### Large Outputs

- Parameters are stored in the workflow CR
- Keep parameter values small (< 256KB)
- Use artifacts for large data

### Script stdout vs File

Script templates:
- `outputs.result` = stdout (automatic)
- `outputs.parameters.X` = from file (explicit)

```yaml
- name: script-template
  script:
    image: python:alpine
    command: [python]
    source: |
      print("stdout-value")                    # -> outputs.result
      with open('/tmp/file.txt', 'w') as f:
        f.write("file-value")                  # -> outputs.parameters.from-file
  outputs:
    parameters:
    - name: from-file
      valueFrom:
        path: /tmp/file.txt
```

## See Also

- `argo://docs/parameters` - Parameter handling reference
- `argo://docs/artifacts` - Artifact system reference
- `argo://docs/variables` - Variable reference for accessing outputs
- `argo://examples/multi-step` - Output passing examples
