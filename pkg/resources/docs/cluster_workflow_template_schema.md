# ClusterWorkflowTemplate CRD Schema

ClusterWorkflowTemplate is a cluster-scoped reusable workflow definition. It can be referenced from any namespace, making it ideal for shared, organization-wide workflow templates.

## API Version and Kind

- **apiVersion**: argoproj.io/v1alpha1
- **kind**: ClusterWorkflowTemplate

## Key Differences from WorkflowTemplate

1. **Cluster-Scoped**: Not confined to a single namespace - accessible from all namespaces
2. **Requires RBAC**: Needs cluster-level permissions to create/update/delete
3. **No Namespace Field**: metadata.namespace is not applicable (always cluster-scoped)
4. **Global Reuse**: Can be referenced by Workflows in any namespace

---

## When to Use ClusterWorkflowTemplate vs WorkflowTemplate

**Use ClusterWorkflowTemplate when:**
- Template should be shared across multiple namespaces
- You want centralized management of common workflows
- Building organization-wide standards (e.g., security scans, compliance checks)
- Creating templates for platform/infrastructure teams

**Use WorkflowTemplate when:**
- Template is specific to a team or project
- You want namespace-level isolation
- Users don't have cluster-level permissions

---

## Structure

A ClusterWorkflowTemplate consists of two main sections:

1. **metadata**: Standard Kubernetes object metadata (without namespace)
2. **spec**: The workflow specification (identical to WorkflowTemplate.spec)

---

## Metadata Fields

Standard Kubernetes ObjectMeta fields (cluster-scoped):

- **name** (string, required): ClusterWorkflowTemplate name (must be unique cluster-wide)
- **labels** (map[string]string): Key-value pairs for organizing templates
- **annotations** (map[string]string): Non-identifying metadata

### Common Labels

- **workflows.argoproj.io/template-type**: Categorize templates (e.g., "ci", "ml", "data-processing")
- **app.kubernetes.io/managed-by**: Indicate management tool
- **team**: Owning team name
- **environment**: Target environment (e.g., "production", "development")

---

## Spec Fields

The spec section is identical to WorkflowTemplate.spec and Workflow.spec. See the Workflow schema documentation for complete details.

### Key Spec Fields

- **templates** ([]Template, required): List of template definitions
- **entrypoint** (string, required): Name of the template to start execution
- **arguments** (Arguments): Default input parameters and artifacts
- **serviceAccountName** (string): ServiceAccount for workflow pods
- **volumes** ([]Volume): Volumes available to templates
- **volumeClaimTemplates** ([]PersistentVolumeClaim): Dynamic PVC creation
- **parallelism** (int64): Max parallel pods
- **ttlStrategy** (TTLStrategy): Workflow deletion strategy
- **podMetadata** (Metadata): Metadata for all pods
- All other fields from Workflow.spec

---

## RBAC Requirements

To work with ClusterWorkflowTemplates, users need cluster-level permissions:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cluster-workflow-template-admin
rules:
- apiGroups: ["argoproj.io"]
  resources: ["clusterworkflowtemplates"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

For read-only access:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cluster-workflow-template-viewer
rules:
- apiGroups: ["argoproj.io"]
  resources: ["clusterworkflowtemplates"]
  verbs: ["get", "list", "watch"]
```

---

## Using ClusterWorkflowTemplates

### Referencing from a Workflow

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: from-cluster-template-
  namespace: team-a  # Can be any namespace
spec:
  workflowTemplateRef:
    name: global-ci-template
    clusterScope: true  # Important: indicates cluster-scoped reference
  arguments:
    parameters:
    - name: repo-url
      value: "https://github.com/example/repo"
```

### Submitting via CLI

```bash
# Submit from any namespace
argo submit --from clusterworkflowtemplate/global-ci-template \
  -n team-a \
  -p repo-url="https://github.com/example/repo"

# List all cluster templates
kubectl get clusterworkflowtemplates

# Get specific template
kubectl get clusterworkflowtemplate/global-ci-template -o yaml
```

---

## Complete Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: security-scan-template
  labels:
    workflows.argoproj.io/template-type: security
    team: platform
spec:
  entrypoint: security-scan
  arguments:
    parameters:
    - name: image
      value: ""
    - name: severity-threshold
      value: "HIGH"

  templates:
  - name: security-scan
    inputs:
      parameters:
      - name: image
      - name: severity-threshold
    steps:
    - - name: trivy-scan
        template: run-trivy
        arguments:
          parameters:
          - name: image
            value: "{{inputs.parameters.image}}"
          - name: severity
            value: "{{inputs.parameters.severity-threshold}}"

    - - name: report
        template: generate-report
        arguments:
          artifacts:
          - name: scan-results
            from: "{{steps.trivy-scan.outputs.artifacts.results}}"

  - name: run-trivy
    inputs:
      parameters:
      - name: image
      - name: severity
    outputs:
      artifacts:
      - name: results
        path: /tmp/trivy-results.json
    container:
      image: aquasec/trivy:latest
      command: [trivy]
      args:
      - "image"
      - "--format=json"
      - "--severity={{inputs.parameters.severity}}"
      - "--output=/tmp/trivy-results.json"
      - "{{inputs.parameters.image}}"

  - name: generate-report
    inputs:
      artifacts:
      - name: scan-results
        path: /tmp/results.json
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["cat /tmp/results.json && echo 'Report generated'"]
```

---

## Advanced Patterns

### Multi-Stage CI/CD Template

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: ci-cd-pipeline
  labels:
    workflows.argoproj.io/template-type: ci-cd
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: git-url
    - name: git-revision
      value: "main"
    - name: image-tag

  templates:
  - name: main
    dag:
      tasks:
      - name: clone
        template: git-clone
        arguments:
          parameters:
          - name: url
            value: "{{workflow.parameters.git-url}}"
          - name: revision
            value: "{{workflow.parameters.git-revision}}"

      - name: test
        template: run-tests
        dependencies: [clone]
        arguments:
          artifacts:
          - name: source
            from: "{{tasks.clone.outputs.artifacts.source}}"

      - name: build
        template: build-image
        dependencies: [test]
        arguments:
          parameters:
          - name: tag
            value: "{{workflow.parameters.image-tag}}"
          artifacts:
          - name: source
            from: "{{tasks.clone.outputs.artifacts.source}}"

      - name: deploy
        template: deploy-app
        dependencies: [build]
        arguments:
          parameters:
          - name: image
            value: "myregistry/myapp:{{workflow.parameters.image-tag}}"

  - name: git-clone
    inputs:
      parameters:
      - name: url
      - name: revision
    outputs:
      artifacts:
      - name: source
        path: /src
    container:
      image: alpine/git:latest
      command: [sh, -c]
      args:
      - >
        git clone {{inputs.parameters.url}} /src &&
        cd /src &&
        git checkout {{inputs.parameters.revision}}

  - name: run-tests
    inputs:
      artifacts:
      - name: source
        path: /src
    container:
      image: node:18
      workingDir: /src
      command: [npm]
      args: ["test"]

  - name: build-image
    inputs:
      parameters:
      - name: tag
      artifacts:
      - name: source
        path: /workspace
    container:
      image: gcr.io/kaniko-project/executor:latest
      args:
      - "--dockerfile=/workspace/Dockerfile"
      - "--context=/workspace"
      - "--destination=myregistry/myapp:{{inputs.parameters.tag}}"

  - name: deploy-app
    inputs:
      parameters:
      - name: image
    resource:
      action: apply
      manifest: |
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: myapp
        spec:
          replicas: 3
          selector:
            matchLabels:
              app: myapp
          template:
            metadata:
              labels:
                app: myapp
            spec:
              containers:
              - name: myapp
                image: {{inputs.parameters.image}}
```

### Data Processing Template

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: etl-pipeline
  labels:
    workflows.argoproj.io/template-type: data-processing
spec:
  entrypoint: etl
  arguments:
    parameters:
    - name: source-table
    - name: dest-table
    - name: date

  templates:
  - name: etl
    steps:
    - - name: extract
        template: extract-data
        arguments:
          parameters:
          - name: table
            value: "{{workflow.parameters.source-table}}"
          - name: date
            value: "{{workflow.parameters.date}}"

    - - name: transform
        template: transform-data
        arguments:
          artifacts:
          - name: input
            from: "{{steps.extract.outputs.artifacts.data}}"

    - - name: load
        template: load-data
        arguments:
          parameters:
          - name: table
            value: "{{workflow.parameters.dest-table}}"
          artifacts:
          - name: data
            from: "{{steps.transform.outputs.artifacts.output}}"

  - name: extract-data
    inputs:
      parameters:
      - name: table
      - name: date
    outputs:
      artifacts:
      - name: data
        path: /tmp/extracted.csv
    container:
      image: postgres:15
      command: [psql]
      args: ["-c", "COPY {{inputs.parameters.table}} TO '/tmp/extracted.csv' CSV"]

  - name: transform-data
    inputs:
      artifacts:
      - name: input
        path: /tmp/input.csv
    outputs:
      artifacts:
      - name: output
        path: /tmp/output.csv
    script:
      image: python:3.11
      command: [python]
      source: |
        import pandas as pd
        df = pd.read_csv('/tmp/input.csv')
        # Transformation logic here
        df.to_csv('/tmp/output.csv', index=False)

  - name: load-data
    inputs:
      parameters:
      - name: table
      artifacts:
      - name: data
        path: /tmp/data.csv
    container:
      image: postgres:15
      command: [psql]
      args: ["-c", "COPY {{inputs.parameters.table}} FROM '/tmp/data.csv' CSV"]
```

---

## Required Fields Summary

**Minimum viable ClusterWorkflowTemplate:**
- metadata.name (cluster-wide unique)
- spec.entrypoint
- spec.templates (with at least one template matching entrypoint)

**Each template must have:**
- name
- ONE execution type (container, script, resource, suspend, steps, or dag)

---

## Best Practices

1. **Namespace-Aware**: Don't hardcode namespace-specific values; use parameters
2. **Document Well**: Use comprehensive descriptions since templates are shared
3. **Version Carefully**: Changes affect all users; consider versioning in names (e.g., "ci-v2")
4. **Security First**: Review security implications since templates run in multiple namespaces
5. **Resource Limits**: Always set appropriate limits to prevent resource exhaustion
6. **Use ServiceAccountName**: Don't assume default SA has necessary permissions
7. **Test Thoroughly**: Test in multiple namespaces before promoting to production

---

## Common Use Cases

- **Organization-Wide CI/CD**: Standard build/test/deploy pipelines
- **Security & Compliance**: Centralized security scans, policy checks
- **Data Platform**: Standard ETL/ELT workflows
- **ML Platforms**: Common training and inference pipelines
- **Infrastructure Operations**: Backup, restore, migration workflows
- **Monitoring & Observability**: Standard metric collection and analysis

---

## Migration from WorkflowTemplate

To convert a WorkflowTemplate to ClusterWorkflowTemplate:

1. Change kind from WorkflowTemplate to ClusterWorkflowTemplate
2. Remove metadata.namespace field
3. Update references to use clusterScope: true
4. Review and parameterize any namespace-specific values
5. Update RBAC permissions for cluster-level access

---

## References

- [Argo Workflows Documentation](https://argo-workflows.readthedocs.io/)
- [Workflow Schema](argo://schemas/workflow) - For complete spec field documentation
- [WorkflowTemplate Schema](argo://schemas/workflow-template) - For namespace-scoped alternative