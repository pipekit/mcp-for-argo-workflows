# Loops Example

Demonstrates iteration patterns in Argo Workflows using withItems, withParam, and withSequence.

## Complete Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: loops-
spec:
  entrypoint: main

  templates:
  - name: main
    steps:
    # Loop over static list of items
    - - name: loop-static
        template: process-item
        arguments:
          parameters:
          - name: item
            value: "{{item}}"
        withItems:
        - "apple"
        - "banana"
        - "cherry"

    # Loop over numeric sequence
    - - name: loop-sequence
        template: process-number
        arguments:
          parameters:
          - name: num
            value: "{{item}}"
        withSequence:
          count: "5"  # Creates: 0, 1, 2, 3, 4

    # Loop over dynamic JSON list
    - - name: generate-list
        template: generate-json

    - - name: loop-dynamic
        template: process-item
        arguments:
          parameters:
          - name: item
            value: "{{item}}"
        withParam: "{{steps.generate-list.outputs.result}}"

  # Template that processes a single item
  - name: process-item
    inputs:
      parameters:
      - name: item
    container:
      image: alpine:latest
      command: [echo]
      args: ["Processing: {{inputs.parameters.item}}"]

  # Template that processes a number
  - name: process-number
    inputs:
      parameters:
      - name: num
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo 'Number: {{inputs.parameters.num}}' && echo 'Square: $(({{inputs.parameters.num}} * {{inputs.parameters.num}}))' "]

  # Template that generates a JSON list dynamically
  - name: generate-json
    script:
      image: python:alpine
      command: [python]
      source: |
        import json
        items = ["red", "green", "blue", "yellow"]
        print(json.dumps(items))
```

## Key Concepts

- **withItems**: Loop over a static list of values
- **withParam**: Loop over a dynamic JSON array (from parameter or step output)
- **withSequence**: Loop over numeric ranges
- **{{item}}**: Access the current iteration value

## Loop Types

### 1. withItems - Static List

```yaml
steps:
- - name: process-users
    template: process
    arguments:
      parameters:
      - name: user
        value: "{{item}}"
    withItems:
    - "alice"
    - "bob"
    - "charlie"
```

### 2. withItems - Objects

```yaml
steps:
- - name: process-configs
    template: deploy
    arguments:
      parameters:
      - name: env
        value: "{{item.environment}}"
      - name: replicas
        value: "{{item.replicas}}"
    withItems:
    - environment: "dev"
      replicas: "1"
    - environment: "staging"
      replicas: "2"
    - environment: "prod"
      replicas: "5"
```

### 3. withSequence - Numeric Range

```yaml
# Simple count
withSequence:
  count: "10"  # 0, 1, 2, ..., 9

# With start, end, and format
withSequence:
  start: "1"
  end: "5"
  format: "batch-%02d"  # batch-01, batch-02, ..., batch-05
```

### 4. withParam - Dynamic JSON

```yaml
steps:
# Generate dynamic list
- - name: list-files
    template: list-s3-files

# Process each file
- - name: process-files
    template: download
    arguments:
      parameters:
      - name: filename
        value: "{{item}}"
    withParam: "{{steps.list-files.outputs.result}}"
```

## Common Variations

### Nested Loops

```yaml
steps:
# Outer loop - environments
- - name: deploy-env
    template: deploy-region
    arguments:
      parameters:
      - name: env
        value: "{{item}}"
    withItems:
    - "dev"
    - "prod"

# Inner template with its own loop
- name: deploy-region
  inputs:
    parameters:
    - name: env
  steps:
  - - name: deploy
      template: deploy-app
      arguments:
        parameters:
        - name: env
          value: "{{inputs.parameters.env}}"
        - name: region
          value: "{{item}}"
      withItems:
      - "us-east-1"
      - "us-west-2"
      - "eu-west-1"
```

### Loop with DAG

```yaml
- name: parallel-processing
  dag:
    tasks:
    # Each task in the loop runs in parallel
    - name: process-item
      template: process
      arguments:
        parameters:
        - name: item
          value: "{{item}}"
      withItems:
      - "task1"
      - "task2"
      - "task3"
```

### Limiting Parallelism

```yaml
steps:
- - name: batch-process
    template: heavy-task
    arguments:
      parameters:
      - name: item
        value: "{{item}}"
    withItems:
    - "item1"
    - "item2"
    - "item3"
    - "item4"
    - "item5"
    # Only run 2 items in parallel at a time
    parallelism: 2
```

### Loop with Artifacts

```yaml
- name: process-files
  inputs:
    artifacts:
    - name: input-file
      path: /input/file.txt
  container:
    image: alpine
    command: [cat, /input/file.txt]
  withParam: "{{inputs.parameters.files}}"
```

## Complex Example: Matrix Build

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: matrix-build-
spec:
  entrypoint: matrix

  templates:
  - name: matrix
    steps:
    - - name: build
        template: build-and-test
        arguments:
          parameters:
          - name: os
            value: "{{item.os}}"
          - name: version
            value: "{{item.version}}"
        withItems:
        - os: "ubuntu"
          version: "20.04"
        - os: "ubuntu"
          version: "22.04"
        - os: "alpine"
          version: "3.18"
        - os: "alpine"
          version: "3.19"

  - name: build-and-test
    inputs:
      parameters:
      - name: os
      - name: version
    container:
      image: "{{inputs.parameters.os}}:{{inputs.parameters.version}}"
      command: [sh, -c]
      args: ["echo 'Building on {{inputs.parameters.os}}:{{inputs.parameters.version}}'"]
```

## Performance Considerations

- **Default Parallelism**: All loop iterations run in parallel by default
- **Controlling Parallelism**: Use `parallelism` to limit concurrent executions
- **Resource Usage**: Be mindful of cluster resources with large loops
- **Pod Creation**: Each iteration creates a new pod

## Next Steps

- See `argo://examples/conditionals` for selective execution based on conditions
- Explore `argo://examples/dag-diamond` for complex parallel patterns
- Check `argo://examples/parameters` for parameter-based iteration
