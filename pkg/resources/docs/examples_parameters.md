# Parameters Example

Demonstrates how to use parameters in Argo Workflows for passing data and configuration.

## Complete Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: parameters-
spec:
  entrypoint: main

  # Workflow-level input parameters
  arguments:
    parameters:
    # Required parameter (no default)
    - name: message
      value: "Hello, World!"

    # Optional parameter with default value
    - name: repeat-count
      value: "3"

    # Parameter with description
    - name: environment
      value: "dev"
      description: "Deployment environment (dev, staging, prod)"

  templates:
  - name: main
    inputs:
      parameters:
      # Receive workflow arguments
      - name: message
      - name: repeat-count
      - name: environment
    steps:
    # First step - print the message
    - - name: print-message
        template: print
        arguments:
          parameters:
          - name: text
            value: "{{inputs.parameters.message}}"
          - name: env
            value: "{{inputs.parameters.environment}}"

    # Second step - repeat the message
    - - name: repeat
        template: repeat-message
        arguments:
          parameters:
          - name: message
            value: "{{inputs.parameters.message}}"
          - name: count
            value: "{{inputs.parameters.repeat-count}}"

    # Third step - process output from previous step
    - - name: summarize
        template: summary
        arguments:
          parameters:
          - name: result
            value: "{{steps.repeat.outputs.parameters.output}}"

  # Template that prints a message
  - name: print
    inputs:
      parameters:
      - name: text
      - name: env
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo '[{{inputs.parameters.env}}] {{inputs.parameters.text}}'"]

  # Template that repeats a message
  - name: repeat-message
    inputs:
      parameters:
      - name: message
      - name: count
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          for i in $(seq 1 {{inputs.parameters.count}}); do
            echo "{{inputs.parameters.message}}"
          done > /tmp/output.txt
          cat /tmp/output.txt
    outputs:
      parameters:
      - name: output
        valueFrom:
          path: /tmp/output.txt

  # Template that summarizes results
  - name: summary
    inputs:
      parameters:
      - name: result
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo 'Summary:' && echo '{{inputs.parameters.result}}' | wc -l"]
```

## Key Concepts

- **arguments.parameters**: Define workflow-level parameters
- **inputs.parameters**: Declare parameters a template accepts
- **outputs.parameters**: Capture data from a template execution
- **{{inputs.parameters.NAME}}**: Access input parameters
- **{{workflow.parameters.NAME}}**: Access workflow-level parameters from any template

## Submitting with Parameters

```bash
# Submit with default parameters
argo submit parameters.yaml

# Override parameters at submission time
argo submit parameters.yaml \
  -p message="Custom message" \
  -p repeat-count=5 \
  -p environment=prod

# Using parameter file
argo submit parameters.yaml --parameter-file params.yaml
```

## Common Variations

### Global Parameters Available Everywhere

```yaml
spec:
  arguments:
    parameters:
    - name: global-config
      value: "config-value"

  templates:
  - name: any-template
    container:
      image: alpine
      command: [echo]
      # Access global parameter without inputs
      args: ["{{workflow.parameters.global-config}}"]
```

### Enum-Style Parameters

```yaml
arguments:
  parameters:
  - name: log-level
    value: "info"
    enum:
    - "debug"
    - "info"
    - "warning"
    - "error"
```

### JSON Parameters

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
        "cache": {
          "ttl": 300
        }
      }

templates:
- name: use-json
  script:
    image: python:alpine
    command: [python]
    source: |
      import json
      config = json.loads('''{{inputs.parameters.config}}''')
      print(f"DB Host: {config['database']['host']}")
```

### Dynamic Parameters from Previous Steps

```yaml
steps:
# Generate dynamic values
- - name: get-config
    template: fetch-config

# Use generated values in next step
- - name: process
    template: processor
    arguments:
      parameters:
      - name: db-host
        value: "{{steps.get-config.outputs.parameters.host}}"
      - name: db-port
        value: "{{steps.get-config.outputs.parameters.port}}"
```

### Optional Parameters with Conditional Logic

```yaml
- name: conditional-param
  inputs:
    parameters:
    - name: optional-value
      value: ""  # Default empty
  container:
    image: alpine
    command: [sh, -c]
    args:
    - |
      if [ -n "{{inputs.parameters.optional-value}}" ]; then
        echo "Using: {{inputs.parameters.optional-value}}"
      else
        echo "Using default behavior"
      fi
```

## Parameter Value Sources

### From ConfigMaps

```yaml
arguments:
  parameters:
  - name: config-value
    valueFrom:
      configMapKeyRef:
        name: my-config
        key: config.json
```

### From Secrets

```yaml
arguments:
  parameters:
  - name: api-key
    valueFrom:
      secretKeyRef:
        name: my-secret
        key: api-key
```

## Next Steps

- See `argo://examples/artifacts` for passing files instead of parameters
- Explore `argo://examples/loops` for iterating over parameter lists
- Check `argo://examples/conditionals` for parameter-based conditional execution
