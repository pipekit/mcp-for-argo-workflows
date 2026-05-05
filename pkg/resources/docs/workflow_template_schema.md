# WorkflowTemplate CRD Schema

WorkflowTemplate is a reusable workflow definition. It defines templates that can be referenced by Workflows or other WorkflowTemplates.

## API Version and Kind

- **apiVersion**: argoproj.io/v1alpha1
- **kind**: WorkflowTemplate

## Key Differences from Workflow

1. **No Status**: WorkflowTemplates are definitions, not executions, so they have no status field
2. **Reusable**: Can be referenced by multiple Workflows via workflowTemplateRef
3. **Namespace-Scoped**: Each WorkflowTemplate exists in a single namespace
4. **No Immediate Execution**: Must be instantiated as a Workflow to execute

---

## Structure

A WorkflowTemplate consists of two main sections:

1. **metadata**: Standard Kubernetes object metadata
2. **spec**: The workflow specification (same as Workflow.spec)

---

## Metadata Fields

Standard Kubernetes ObjectMeta fields:

- **name** (string, required): WorkflowTemplate name
- **namespace** (string): Kubernetes namespace. Defaults to "default" if not specified
- **labels** (map[string]string): Key-value pairs for organizing templates
- **annotations** (map[string]string): Non-identifying metadata

### Common Labels

- **workflows.argoproj.io/template-type**: Custom label to categorize templates
- Organization-specific labels for grouping and discovery

---

## Spec Fields

The spec section is identical to Workflow.spec. See the Workflow schema documentation for complete details.

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
- **podGC** (PodGCStrategy): Pod garbage collection
- **nodeSelector** (map[string]string): Node selection
- **affinity** (Affinity): Pod affinity rules
- **tolerations** ([]Toleration): Node tolerations
- **synchronization** (Synchronization): Concurrency control
- **metrics** (Metrics): Custom metrics
- **hooks** (map[string]LifecycleHook): Lifecycle hooks
- **onExit** (string): Exit handler template

All fields from Workflow.spec are supported except those that only apply to running workflows.

---

## Using WorkflowTemplates

### Referencing from a Workflow

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: from-template-
spec:
  workflowTemplateRef:
    name: my-workflow-template
  arguments:
    parameters:
    - name: message
      value: "Hello from Workflow"
```

### Submitting via CLI

```bash
# Submit a WorkflowTemplate with parameters
argo submit --from workflowtemplate/my-template \
  -p message="Hello World"

# Submit with generateName
argo submit --from workflowtemplate/my-template \
  --generate-name "run-"
```

---

## Complete Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: hello-world-template
  namespace: default
  labels:
    workflows.argoproj.io/template-type: example
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: message
      value: "Default Message"
  templates:
  - name: main
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:latest
      command: [echo]
      args: ["{{inputs.parameters.message}}"]
```

---

## Advanced Patterns

### Template with DAG

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: dag-template
spec:
  entrypoint: main
  templates:
  - name: main
    dag:
      tasks:
      - name: task-a
        template: echo
        arguments:
          parameters:
          - name: message
            value: "Task A"
      - name: task-b
        template: echo
        dependencies: [task-a]
        arguments:
          parameters:
          - name: message
            value: "Task B"
  - name: echo
    inputs:
      parameters:
      - name: message
    container:
      image: alpine:latest
      command: [echo]
      args: ["{{inputs.parameters.message}}"]
```

### Template with Artifacts

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: artifact-template
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: generate
        template: gen-artifact
    - - name: consume
        template: use-artifact
        arguments:
          artifacts:
          - name: data
            from: "{{steps.generate.outputs.artifacts.result}}"

  - name: gen-artifact
    container:
      image: alpine:latest
      command: [sh, -c]
      args: ["echo 'Hello' > /tmp/result.txt"]
    outputs:
      artifacts:
      - name: result
        path: /tmp/result.txt

  - name: use-artifact
    inputs:
      artifacts:
      - name: data
        path: /tmp/input.txt
    container:
      image: alpine:latest
      command: [cat]
      args: [/tmp/input.txt]
```

### Template with Resource Requests

```yaml
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: resource-limits-template
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: python:3.9
      command: [python, -c]
      args: ["import time; print('Processing...'); time.sleep(10)"]
      resources:
        requests:
          memory: "64Mi"
          cpu: "250m"
        limits:
          memory: "128Mi"
          cpu: "500m"
```

---

## Required Fields Summary

**Minimum viable WorkflowTemplate:**
- metadata.name
- spec.entrypoint
- spec.templates (with at least one template matching entrypoint)

**Each template must have:**
- name
- ONE execution type (container, script, resource, suspend, steps, or dag)

---

## Best Practices

1. **Use Parameters**: Make templates reusable with parameters instead of hardcoded values
2. **Version Templates**: Use labels or naming conventions to version templates (e.g., "my-template-v2")
3. **Document Arguments**: Use descriptions in parameter definitions to document expected inputs
4. **Set Resource Limits**: Always specify resource requests/limits for production templates
5. **Use Exit Handlers**: Add onExit templates for cleanup operations
6. **Template TTL**: Configure ttlStrategy to automatically clean up completed workflows

---

## Common Use Cases

- **CI/CD Pipelines**: Define build, test, and deploy workflows
- **Data Processing**: ETL and batch processing jobs
- **Machine Learning**: Training and inference pipelines
- **Scheduled Tasks**: Templates invoked by CronWorkflows
- **Multi-Step Operations**: Complex operations with dependencies

---

## References

- [Argo Workflows Documentation](https://argo-workflows.readthedocs.io/)
- [WorkflowTemplate Examples](https://github.com/argoproj/argo-workflows/tree/main/examples)
- [Workflow Schema](argo://schemas/workflow) - For complete spec field documentation