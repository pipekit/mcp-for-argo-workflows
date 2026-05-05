# Workflow CRD Schema

The Workflow is the primary resource in Argo Workflows, representing an executable workflow definition.

## API Version and Kind

- **apiVersion**: argoproj.io/v1alpha1
- **kind**: Workflow

## Structure

A Workflow consists of three main sections:

1. **metadata**: Standard Kubernetes object metadata
2. **spec**: The workflow specification (what to execute)
3. **status**: Runtime status information (read-only, managed by controller)

---

## Metadata Fields

Standard Kubernetes ObjectMeta fields:

- **name** (string, required): Workflow name. If using generateName, this will be auto-generated.
- **generateName** (string): Prefix for auto-generated names (e.g., "my-workflow-" → "my-workflow-abc123")
- **namespace** (string): Kubernetes namespace. Defaults to "default" if not specified.
- **labels** (map[string]string): Key-value pairs for organizing and selecting workflows
- **annotations** (map[string]string): Non-identifying metadata

### Common Labels

- **workflows.argoproj.io/workflow-template**: Name of the WorkflowTemplate (if created from template)
- **workflows.argoproj.io/cron-workflow**: Name of the CronWorkflow (if created by cron)
- Custom labels for organization-specific purposes

---

## Spec Fields

### Core Template Fields

- **templates** ([]Template, required): List of template definitions
  - Each template defines a step, DAG node, or container to execute
  - Referenced by name from entrypoint or other templates
  - See Template Schema section below

- **entrypoint** (string, required): Name of the template to start execution
  - Must match one of the template names in the templates list

### Arguments

- **arguments** (Arguments): Input parameters and artifacts for the workflow
  - **parameters** ([]Parameter): Input parameters
    - **name** (string, required): Parameter name
    - **value** (string): Default value
    - **valueFrom** (ValueFrom): Dynamic value source
  - **artifacts** ([]Artifact): Input artifacts
    - **name** (string, required): Artifact name
    - **path** (string): Where to load the artifact in containers
    - **from** (string): Reference to another artifact
    - **s3**, **gcs**, **http**, **git**, etc.: Artifact location

### Execution Control

- **suspend** (bool): If true, workflow starts in suspended state
- **parallelism** (int64): Max number of parallel pods (limits concurrency)
- **activeDeadlineSeconds** (int64): Maximum workflow duration in seconds
- **ttlStrategy** (TTLStrategy): When to delete completed workflows
  - **secondsAfterCompletion** (int32): Delete N seconds after completion
  - **secondsAfterSuccess** (int32): Delete N seconds after success
  - **secondsAfterFailure** (int32): Delete N seconds after failure

### Pod Configuration

- **serviceAccountName** (string): ServiceAccount for all workflow pods
- **automountServiceAccountToken** (bool): Whether to mount SA token
- **podMetadata** (Metadata): Metadata to apply to all workflow pods
  - **labels** (map[string]string): Labels for all pods
  - **annotations** (map[string]string): Annotations for all pods
- **podGC** (PodGCStrategy): Pod garbage collection strategy
  - **strategy** (string): "OnPodCompletion", "OnPodSuccess", "OnWorkflowCompletion", "OnWorkflowSuccess"
  - **labelSelector** (LabelSelector): Select which pods to GC

### Scheduling

- **nodeSelector** (map[string]string): Node labels for pod scheduling
- **affinity** (Affinity): Pod affinity/anti-affinity rules
- **tolerations** ([]Toleration): Pod tolerations for node taints
- **schedulerName** (string): Custom scheduler name
- **priorityClassName** (string): Priority class for workflow pods
- **priority** (int32): Priority value

### Storage

- **volumes** ([]Volume): Volumes available to all templates
- **volumeClaimTemplates** ([]PersistentVolumeClaim): Dynamic PVC creation
  - Created at workflow start, deleted at workflow end
  - Can be shared across workflow steps

### Artifact Repository

- **artifactRepositoryRef** (ArtifactRepositoryRef): Reference to artifact repo config
  - **configMap** (string): ConfigMap name
  - **key** (string): Key within ConfigMap

### Workflow Template References

- **workflowTemplateRef** (WorkflowTemplateRef): Reference to a WorkflowTemplate
  - **name** (string): WorkflowTemplate name
  - **clusterScope** (bool): If true, references a ClusterWorkflowTemplate

### Hooks

- **hooks** (map[string]LifecycleHook): Workflow-level lifecycle hooks
  - Keys: "running", "succeeded", "failed", "error", "exit"
  - Values: Template reference to execute at lifecycle point

### Synchronization

- **synchronization** (Synchronization): Workflow synchronization
  - **semaphore** (SemaphoreRef): Limit concurrent workflows
    - **configMapKeyRef** (ConfigMapKeySelector): ConfigMap with semaphore config
  - **mutex** (Mutex): Mutual exclusion lock
    - **name** (string): Mutex name

### Metrics

- **metrics** (Metrics): Custom Prometheus metrics
  - **prometheus** ([]Prometheus): List of Prometheus metrics to emit

### Other Fields

- **onExit** (string): Template to execute when workflow exits (regardless of success/failure)
- **hostNetwork** (bool): Enable host networking for pods
- **dnsPolicy** (string): DNS policy ("ClusterFirst", "Default", "ClusterFirstWithHostNet", "None")
- **dnsConfig** (PodDNSConfig): Custom DNS configuration
- **imagePullSecrets** ([]LocalObjectReference): Secrets for pulling private images
- **securityContext** (PodSecurityContext): Pod-level security context
- **podSpecPatch** (string): JSON patch to apply to pod specs
- **podPriority** (int32): Pod priority value
- **retryStrategy** (RetryStrategy): Workflow-level retry configuration

---

## Template Schema

Templates are the building blocks of workflows. Each template must have ONE of these types:

### Template Types

1. **container** (Container): Run a container
   - **image** (string, required): Container image
   - **command** ([]string): Entrypoint command
   - **args** ([]string): Arguments to command
   - **env** ([]EnvVar): Environment variables
   - **resources** (ResourceRequirements): CPU/memory requests and limits
   - **volumeMounts** ([]VolumeMount): Volumes to mount

2. **script** (ScriptTemplate): Run a script in a container
   - **image** (string, required): Container image
   - **source** (string, required): Script source code
   - **command** ([]string): Script interpreter (e.g., ["python"])

3. **resource** (ResourceTemplate): Create/apply Kubernetes resource
   - **action** (string, required): "create", "apply", "delete", "patch"
   - **manifest** (string): Resource manifest (can use templates)
   - **successCondition** (string): Success criteria
   - **failureCondition** (string): Failure criteria

4. **suspend** (SuspendTemplate): Pause execution
   - **duration** (string): How long to suspend (e.g., "10s", "5m")

5. **steps** ([]ParallelSteps): Sequential steps with parallelism
   - Each item is a list of steps that run in parallel
   - Steps run sequentially across items

6. **dag** (DAGTemplate): Directed Acyclic Graph of tasks
   - **tasks** ([]DAGTask): List of DAG tasks
     - **name** (string, required): Task name
     - **template** (string, required): Template to execute
     - **dependencies** ([]string): Task names this depends on
     - **arguments** (Arguments): Arguments to pass

### Template Fields

- **name** (string, required): Template name
- **inputs** (Inputs): Input parameters and artifacts
  - **parameters** ([]Parameter): Input parameters
  - **artifacts** ([]Artifact): Input artifacts
- **outputs** (Outputs): Output parameters and artifacts
  - **parameters** ([]Parameter): Output parameters
    - **valueFrom** (ValueFrom): How to extract value
      - **path** (string): File path to read
      - **jsonPath** (string): JSONPath expression
  - **artifacts** ([]Artifact): Output artifacts
    - **path** (string): File path to save
- **metadata** (Metadata): Template-level metadata
- **activeDeadlineSeconds** (int64): Template timeout
- **retryStrategy** (RetryStrategy): Retry configuration
  - **limit** (int32): Max retry attempts
  - **backoff** (Backoff): Backoff strategy
- **parallelism** (int64): Limit parallel execution (for steps/dag)

---

## Status Fields (Read-Only)

The status section is managed by the workflow controller:

- **phase** (string): Current workflow phase
  - Values: "Pending", "Running", "Succeeded", "Failed", "Error"
- **message** (string): Human-readable status message
- **startedAt** (Time): When workflow started
- **finishedAt** (Time): When workflow completed
- **progress** (string): Progress indicator (e.g., "3/5")
- **nodes** (map[string]NodeStatus): Status of each node
  - Key: Node ID
  - Value: NodeStatus with phase, timing, outputs, etc.

---

## Common Patterns

### Simple Container Workflow

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: main
  templates:
  - name: main
    container:
      image: alpine:latest
      command: [echo]
      args: ["Hello, World!"]
```

### Parameterized Workflow

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: parameterized-
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
      - name: message
    container:
      image: alpine:latest
      command: [echo]
      args: ["{{inputs.parameters.message}}"]
```

### DAG Workflow

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: dag-example-
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

---

## Required Fields Summary

**Minimum viable Workflow:**
- metadata.generateName OR metadata.name
- spec.entrypoint
- spec.templates (with at least one template matching entrypoint)

**Each template must have:**
- name
- ONE execution type (container, script, resource, suspend, steps, or dag)

---

## References

- [Argo Workflows Documentation](https://argo-workflows.readthedocs.io/)
- [Workflow Examples](https://github.com/argoproj/argo-workflows/tree/main/examples)