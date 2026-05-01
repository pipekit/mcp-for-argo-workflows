# Multi-Step Workflow Example

Demonstrates sequential step execution with data passing between steps using outputs and inputs.

## Complete Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: multi-step-
spec:
  entrypoint: main

  templates:
  # Main template that orchestrates the steps
  - name: main
    steps:
    # First step - generate a message
    - - name: generate-message
        template: generate

    # Second step - process the message (runs after first completes)
    - - name: process-message
        template: process
        arguments:
          parameters:
          # Pass output from first step to second step
          - name: message
            value: "{{steps.generate-message.outputs.parameters.result}}"

    # Third step - display the result
    - - name: display-result
        template: display
        arguments:
          parameters:
          - name: processed
            value: "{{steps.process-message.outputs.parameters.processed}}"

  # Template that generates a message
  - name: generate
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo 'Hello from step 1' > /tmp/message.txt && cat /tmp/message.txt"]
    outputs:
      parameters:
      # Capture output to pass to next step
      - name: result
        valueFrom:
          path: /tmp/message.txt

  # Template that processes the message
  - name: process
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo 'Processed: {{inputs.parameters.message}}' | tee /tmp/output.txt"]
    outputs:
      parameters:
      - name: processed
        valueFrom:
          path: /tmp/output.txt

  # Template that displays the final result
  - name: display
    inputs:
      parameters:
      - name: processed
    container:
      image: alpine:latest
      command: [echo]
      args: ["Final result: {{inputs.parameters.processed}}"]
```

## Key Concepts

- **steps**: Defines sequential execution - each array element is a step group
- **outputs.parameters**: Capture data from a step to pass to subsequent steps
- **inputs.parameters**: Receive data from previous steps or workflow arguments
- **valueFrom.path**: Read parameter value from a file in the container
- **{{steps.STEP_NAME.outputs.parameters.PARAM_NAME}}**: Reference output from previous steps

## Step Execution Order

```text
generate-message (Step 1)
        ↓
process-message (Step 2) - uses output from Step 1
        ↓
display-result (Step 3) - uses output from Step 2
```

## Common Variations

### Parallel Steps Within a Group

```yaml
steps:
# These two steps run in parallel
- - name: step-a
    template: task-a
  - name: step-b
    template: task-b

# This step waits for both above to complete
- - name: step-c
    template: task-c
```

### Using Script Template for Data Generation

```yaml
- name: generate-data
  script:
    image: python:alpine
    command: [python]
    source: |
      import json
      data = {"status": "success", "value": 42}
      print(json.dumps(data))
  outputs:
    parameters:
    - name: json-data
      valueFrom:
        path: /tmp/stdout
```

### Conditional Step Execution

```yaml
- name: conditional-step
  template: cleanup
  when: "{{steps.process-message.outputs.parameters.status}} == failed"
```

## Next Steps

- See `argo://examples/dag-diamond` for parallel execution with dependencies
- Explore `argo://examples/parameters` for more parameter passing patterns
- Check `argo://examples/artifacts` for passing files between steps
