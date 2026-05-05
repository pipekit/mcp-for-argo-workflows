# Artifacts Example

Demonstrates passing files (artifacts) between workflow steps using artifact repositories.

## Complete Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: artifacts-
spec:
  entrypoint: main

  # Optional: Configure artifact repository (can also be configured globally)
  artifactRepositoryRef:
    configMap: artifact-repositories
    key: default-v1

  templates:
  - name: main
    steps:
    # Step 1: Generate a file
    - - name: generate-artifact
        template: generate-file

    # Step 2: Process the file from step 1
    - - name: process-artifact
        template: process-file
        arguments:
          artifacts:
          # Pass the artifact from previous step
          - name: input-file
            from: "{{steps.generate-artifact.outputs.artifacts.result-file}}"

    # Step 3: Consume the processed file
    - - name: consume-artifact
        template: consume-file
        arguments:
          artifacts:
          - name: processed-file
            from: "{{steps.process-artifact.outputs.artifacts.output-file}}"

  # Template that generates a file
  - name: generate-file
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          echo "Generating data at $(date)" > /tmp/data.txt
          echo "Line 1: Important data" >> /tmp/data.txt
          echo "Line 2: More data" >> /tmp/data.txt
          cat /tmp/data.txt
    outputs:
      artifacts:
      # Save the file as an artifact
      - name: result-file
        path: /tmp/data.txt

  # Template that processes the input file
  - name: process-file
    inputs:
      artifacts:
      # Receive artifact and mount at this path
      - name: input-file
        path: /tmp/input.txt
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          echo "Processing file..."
          cat /tmp/input.txt
          echo "---PROCESSED---" >> /tmp/input.txt
          cat /tmp/input.txt | tr '[:lower:]' '[:upper:]' > /tmp/output.txt
          cat /tmp/output.txt
    outputs:
      artifacts:
      - name: output-file
        path: /tmp/output.txt

  # Template that consumes the final artifact
  - name: consume-file
    inputs:
      artifacts:
      - name: processed-file
        path: /tmp/final.txt
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo 'Final output:' && cat /tmp/final.txt"]
```

## Key Concepts

- **outputs.artifacts**: Define files to save from a template execution
- **inputs.artifacts**: Declare artifacts a template receives
- **path**: File system path where artifact is saved or loaded
- **from**: Reference to an artifact from a previous step
- **Artifact Repository**: Backend storage (S3, GCS, MinIO) for artifact persistence

## Artifact Repository Configuration

### Using S3

```yaml
spec:
  # S3 artifact repository
  artifactRepositoryRef:
    configMap: artifact-repositories
    key: s3-repository

# ConfigMap definition (applied separately)
apiVersion: v1
kind: ConfigMap
metadata:
  name: artifact-repositories
data:
  s3-repository: |
    s3:
      bucket: my-workflow-artifacts
      endpoint: s3.amazonaws.com
      region: us-west-2
      # Optional: custom path prefix
      keyFormat: "{{workflow.name}}/{{pod.name}}"
      # Authentication via serviceAccountKeySecret or IAM roles
      accessKeySecret:
        name: my-s3-credentials
        key: accessKey
      secretKeySecret:
        name: my-s3-credentials
        key: secretKey
```

### Using GCS

```yaml
data:
  gcs-repository: |
    gcs:
      bucket: my-workflow-artifacts
      keyFormat: "{{workflow.name}}/{{pod.name}}"
      serviceAccountKeySecret:
        name: my-gcs-credentials
        key: serviceAccountKey
```

### Using MinIO

```yaml
data:
  minio-repository: |
    s3:
      bucket: my-bucket
      endpoint: minio.default.svc.cluster.local:9000
      insecure: true
      accessKeySecret:
        name: my-minio-cred
        key: accesskey
      secretKeySecret:
        name: my-minio-cred
        key: secretkey
```

## Common Variations

### Multiple Artifacts

```yaml
outputs:
  artifacts:
  - name: logs
    path: /tmp/logs.txt
  - name: results
    path: /tmp/results.json
  - name: report
    path: /tmp/report.html
```

### Archive/Tar Artifacts

```yaml
outputs:
  artifacts:
  - name: archive
    path: /tmp/output
    archive:
      # Archive entire directory
      tar:
        compressionLevel: 6
```

### Artifact from Git Repository

```yaml
inputs:
  artifacts:
  - name: source-code
    path: /src
    git:
      repo: https://github.com/argoproj/argo-workflows.git
      revision: "v3.5.0"
```

### Artifact from HTTP URL

```yaml
inputs:
  artifacts:
  - name: data
    path: /tmp/data.csv
    http:
      url: https://example.com/data.csv
```

### Optional Artifacts

```yaml
inputs:
  artifacts:
  - name: optional-config
    path: /config/settings.json
    optional: true  # Won't fail if artifact doesn't exist
```

### Artifact in DAG

```yaml
- name: artifact-dag
  dag:
    tasks:
    - name: create
      template: generate-file

    - name: process-a
      dependencies: [create]
      template: process-file
      arguments:
        artifacts:
        - name: input-file
          from: "{{tasks.create.outputs.artifacts.result-file}}"

    - name: process-b
      dependencies: [create]
      template: process-file
      arguments:
        artifacts:
        - name: input-file
          from: "{{tasks.create.outputs.artifacts.result-file}}"
```

## Artifacts vs Parameters

| Feature | Parameters | Artifacts |
|---------|-----------|-----------|
| Use Case | Small text/JSON data | Large files, binaries, datasets |
| Size Limit | ~256KB | Limited by storage backend |
| Storage | Kubernetes API | S3/GCS/MinIO |
| Access | Direct value in YAML | Mounted as files |

## Next Steps

- See `argo://examples/parameters` for passing small data values
- Explore `argo://examples/volumes` for shared persistent storage
- Check `argo://examples/multi-step` for sequential artifact passing
