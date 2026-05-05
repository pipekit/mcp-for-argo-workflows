# Argo Workflows Artifacts Reference

Comprehensive guide to the artifact system in Argo Workflows for passing files between steps and storing outputs.

## Artifact Overview

Artifacts are files or directories that can be:
- Passed between workflow steps
- Downloaded from external sources
- Uploaded to artifact repositories
- Exported as workflow outputs

## Input Artifacts

### Basic Input Artifact

```yaml
templates:
- name: processor
  inputs:
    artifacts:
    - name: data-file
      path: /tmp/input.txt      # Where to mount the artifact
  container:
    image: alpine
    command: [cat]
    args: ["/tmp/input.txt"]
```

### Optional Artifacts

```yaml
inputs:
  artifacts:
  - name: optional-config
    path: /config/settings.yaml
    optional: true              # Won't fail if artifact doesn't exist
```

### Artifact from Previous Step

```yaml
arguments:
  artifacts:
  - name: input-data
    from: "{{steps.generate.outputs.artifacts.result}}"
```

### Artifact from Previous Task (DAG)

```yaml
arguments:
  artifacts:
  - name: input-data
    from: "{{tasks.generate.outputs.artifacts.result}}"
```

## Output Artifacts

### Basic Output Artifact

```yaml
outputs:
  artifacts:
  - name: result
    path: /tmp/output.txt       # File to capture as artifact
```

### Directory as Artifact

```yaml
outputs:
  artifacts:
  - name: results-dir
    path: /tmp/results/         # Entire directory as artifact
```

### Global Artifact (Workflow Output)

```yaml
outputs:
  artifacts:
  - name: report
    path: /tmp/report.pdf
    globalName: final-report    # Exported as workflow output
```

### Archive Settings

```yaml
outputs:
  artifacts:
  - name: logs
    path: /tmp/logs/
    archive:
      none: {}                  # No compression
```

```yaml
outputs:
  artifacts:
  - name: data
    path: /tmp/data/
    archive:
      tar:
        compressionLevel: 6     # gzip compression (1-9)
```

```yaml
outputs:
  artifacts:
  - name: binaries
    path: /tmp/bin/
    archive:
      zip: {}                   # ZIP compression
```

## Artifact Sources

### From Git Repository

```yaml
inputs:
  artifacts:
  - name: source-code
    path: /src
    git:
      repo: https://github.com/argoproj/argo-workflows.git
      revision: "main"
      # Optional: specific directory
      singleBranch: true
      depth: 1                  # Shallow clone
```

### From Git with SSH

```yaml
inputs:
  artifacts:
  - name: source-code
    path: /src
    git:
      repo: git@github.com:org/repo.git
      revision: "main"
      sshPrivateKeySecret:
        name: git-ssh-key
        key: ssh-privatekey
```

### From HTTP URL

```yaml
inputs:
  artifacts:
  - name: data
    path: /tmp/data.csv
    http:
      url: https://example.com/data.csv
      headers:
      - name: Authorization
        value: "Bearer {{workflow.parameters.token}}"
```

### From S3

```yaml
inputs:
  artifacts:
  - name: model
    path: /models/
    s3:
      bucket: my-bucket
      key: models/latest/
      endpoint: s3.amazonaws.com
      region: us-west-2
      accessKeySecret:
        name: s3-credentials
        key: accessKey
      secretKeySecret:
        name: s3-credentials
        key: secretKey
```

### From GCS

```yaml
inputs:
  artifacts:
  - name: data
    path: /data/
    gcs:
      bucket: my-gcs-bucket
      key: data/input/
      serviceAccountKeySecret:
        name: gcs-credentials
        key: serviceAccountKey
```

### From Azure Blob

```yaml
inputs:
  artifacts:
  - name: data
    path: /data/
    azure:
      container: my-container
      blob: data/input.tar
      accountKeySecret:
        name: azure-credentials
        key: accountKey
```

### From Artifactory

```yaml
inputs:
  artifacts:
  - name: artifact
    path: /tmp/artifact.jar
    artifactory:
      url: https://artifactory.example.com/repo/path/artifact.jar
      usernameSecret:
        name: artifactory-creds
        key: username
      passwordSecret:
        name: artifactory-creds
        key: password
```

### From HDFS

```yaml
inputs:
  artifacts:
  - name: data
    path: /data/
    hdfs:
      addresses:
      - namenode:8020
      path: /data/input/
      hdfsUser: "hdfs-user"
```

### Raw Inline Data

```yaml
inputs:
  artifacts:
  - name: config
    path: /config/settings.yaml
    raw:
      data: |
        database:
          host: localhost
          port: 5432
        cache:
          enabled: true
```

## Artifact Repository Configuration

### Default Repository

Configure a default repository for all artifacts:

```yaml
spec:
  artifactRepositoryRef:
    configMap: artifact-repositories
    key: default
```

### Repository ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: artifact-repositories
  annotations:
    # Mark as default repository
    workflows.argoproj.io/default-artifact-repository: "default"
data:
  default: |
    s3:
      bucket: workflow-artifacts
      endpoint: s3.amazonaws.com
      region: us-west-2
      accessKeySecret:
        name: s3-credentials
        key: accessKey
      secretKeySecret:
        name: s3-credentials
        key: secretKey

  minio: |
    s3:
      bucket: argo-artifacts
      endpoint: minio.default.svc.cluster.local:9000
      insecure: true
      accessKeySecret:
        name: minio-credentials
        key: accesskey
      secretKeySecret:
        name: minio-credentials
        key: secretkey
```

### Key Format Template

Customize artifact storage paths:

```yaml
s3:
  bucket: my-bucket
  keyFormat: "{{workflow.namespace}}/{{workflow.name}}/{{pod.name}}"
```

Available variables:
- `{{workflow.namespace}}`
- `{{workflow.name}}`
- `{{workflow.uid}}`
- `{{workflow.creationTimestamp}}`
- `{{pod.name}}`

## Artifact Garbage Collection

### Set Artifact GC Strategy

```yaml
outputs:
  artifacts:
  - name: temp-data
    path: /tmp/data.txt
    artifactGC:
      strategy: OnWorkflowDeletion  # Delete when workflow is deleted
```

Strategies:
- `Never` - Keep forever (default)
- `OnWorkflowDeletion` - Delete when workflow CR is deleted
- `OnWorkflowCompletion` - Delete when workflow completes

### Workflow-Level GC

```yaml
spec:
  artifactGC:
    strategy: OnWorkflowCompletion
    forceFinalizerRemoval: true
```

## Artifact Passing Patterns

### Steps: Sequential Passing

```yaml
templates:
- name: pipeline
  steps:
  - - name: generate
      template: generator
  - - name: process
      template: processor
      arguments:
        artifacts:
        - name: input
          from: "{{steps.generate.outputs.artifacts.data}}"
  - - name: finalize
      template: finalizer
      arguments:
        artifacts:
        - name: input
          from: "{{steps.process.outputs.artifacts.result}}"
```

### DAG: Parallel Fan-Out

```yaml
templates:
- name: dag-pipeline
  dag:
    tasks:
    - name: create-data
      template: generator

    - name: process-a
      dependencies: [create-data]
      template: processor
      arguments:
        artifacts:
        - name: input
          from: "{{tasks.create-data.outputs.artifacts.data}}"

    - name: process-b
      dependencies: [create-data]
      template: processor
      arguments:
        artifacts:
        - name: input
          from: "{{tasks.create-data.outputs.artifacts.data}}"
```

### Loop: Same Artifact to Multiple Iterations

```yaml
steps:
- - name: prepare
    template: data-prep
- - name: process
    template: worker
    arguments:
      artifacts:
      - name: shared-data
        from: "{{steps.prepare.outputs.artifacts.data}}"
      parameters:
      - name: id
        value: "{{item}}"
    withItems: ["1", "2", "3"]
```

## Artifact Mode and Permissions

### Set File Mode

```yaml
inputs:
  artifacts:
  - name: script
    path: /scripts/run.sh
    mode: 0755                  # Make executable
```

### Recursive Mode

```yaml
inputs:
  artifacts:
  - name: scripts
    path: /scripts/
    recurseMode: true           # Apply mode recursively
    mode: 0755
```

## Sub-Path Extraction

Extract specific files from an artifact:

```yaml
inputs:
  artifacts:
  - name: repo
    path: /work/
    git:
      repo: https://github.com/example/repo.git
    subPath: "src/main"         # Only extract src/main subdirectory
```

## Deleted Artifacts

Handle artifacts that may have been deleted:

```yaml
inputs:
  artifacts:
  - name: maybe-deleted
    path: /data/
    optional: true
    deleted: false              # Fail if artifact was GC'd
```

## Artifacts vs Parameters

| Aspect | Parameters | Artifacts |
|--------|-----------|-----------|
| Data Type | Small text/JSON | Files, binaries, datasets |
| Size Limit | ~256KB | Limited by storage backend |
| Storage | Kubernetes etcd | External (S3, GCS, etc.) |
| Access | Direct substitution | Mounted as files |
| Speed | Fast | Slower (network transfer) |
| Persistence | In workflow CR | In artifact repository |

## Common Gotchas

### Path Must Be Absolute

```yaml
# Correct
path: /tmp/data.txt

# Incorrect - will fail
path: data.txt
```

### Directory vs File

```yaml
# File artifact - specific file
path: /tmp/result.txt

# Directory artifact - entire directory
path: /tmp/results/           # Trailing slash optional but recommended
```

### Artifact Repository Required

Output artifacts require a configured artifact repository. Without one, artifacts can only be passed within the same workflow using `emptyDir` volumes.

### Archive Extraction

Input artifacts are automatically extracted if archived. The `path` is where the extracted contents appear.

## See Also

- `argo://docs/outputs` - Output capture patterns
- `argo://docs/parameters` - When to use parameters vs artifacts
- `argo://examples/artifacts` - Artifact usage examples
- `argo://examples/volumes` - Shared storage alternatives
