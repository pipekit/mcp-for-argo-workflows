# Volumes Example

Demonstrates volume mounting patterns including PersistentVolumeClaims, ConfigMaps, Secrets, and shared volumes between workflow steps.

## Complete Example

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: volumes-
spec:
  entrypoint: main

  # Define volumes available to all templates
  volumes:
  # PersistentVolumeClaim for shared storage
  - name: workdir
    persistentVolumeClaim:
      claimName: my-workflow-pvc

  # ConfigMap volume
  - name: config
    configMap:
      name: workflow-config

  # Secret volume
  - name: credentials
    secret:
      secretName: api-credentials

  # EmptyDir for temporary storage
  - name: scratch
    emptyDir: {}

  templates:
  - name: main
    steps:
    # Step 1: Write to shared volume
    - - name: generate-data
        template: writer

    # Step 2: Read from shared volume
    - - name: process-data
        template: processor

    # Step 3: Use ConfigMap and Secret
    - - name: use-config
        template: configured-task

  # Template that writes to shared volume
  - name: writer
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          echo "Writing data at $(date)" > /work/data.txt
          echo "Line 1" >> /work/data.txt
          echo "Line 2" >> /work/data.txt
          cat /work/data.txt
      volumeMounts:
      - name: workdir
        mountPath: /work

  # Template that reads from shared volume
  - name: processor
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          echo "Reading shared data:"
          cat /work/data.txt
          echo "Processing..."
          cat /work/data.txt | wc -l > /work/result.txt
      volumeMounts:
      - name: workdir
        mountPath: /work

  # Template using ConfigMap and Secret
  - name: configured-task
    container:
      image: alpine:latest
      command: [sh, -c]
      args:
        - |
          echo "Config file:"
          cat /config/app-config.yaml
          echo "Using API key from secret..."
          # Secret mounted as file
          if [ -f /secrets/api-key ]; then
            echo "API key loaded successfully"
          fi
      volumeMounts:
      - name: config
        mountPath: /config
      - name: credentials
        mountPath: /secrets
        readOnly: true
      - name: scratch
        mountPath: /tmp
```

## Supporting Resources

### PersistentVolumeClaim

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-workflow-pvc
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: standard
```

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: workflow-config
data:
  app-config.yaml: |
    database:
      host: localhost
      port: 5432
    cache:
      ttl: 300
```

### Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: api-credentials
type: Opaque
data:
  # base64-encoded placeholder values for documentation
  api-key: <BASE64_ENCODED_API_KEY>
  username: <BASE64_ENCODED_USERNAME>
  password: <BASE64_ENCODED_PASSWORD>
```

## Volume Types

### 1. PersistentVolumeClaim (PVC)

```yaml
volumes:
- name: data-volume
  persistentVolumeClaim:
    claimName: my-pvc

# Mount in template
volumeMounts:
- name: data-volume
  mountPath: /data
```

**Use Case**: Persistent storage that survives pod restarts, shared data between workflow runs.

### 2. EmptyDir

```yaml
volumes:
- name: scratch-space
  emptyDir: {}

# Or with size limit
- name: limited-scratch
  emptyDir:
    sizeLimit: "1Gi"

# Or memory-backed
- name: memory-scratch
  emptyDir:
    medium: Memory
    sizeLimit: "512Mi"
```

**Use Case**: Temporary storage, scratch space for processing, shared between containers in same pod.

### 3. ConfigMap

```yaml
volumes:
- name: config
  configMap:
    name: my-config
    items:  # Optional: select specific keys
    - key: config.json
      path: application/config.json

volumeMounts:
- name: config
  mountPath: /etc/config
```

**Use Case**: Configuration files, application settings.

### 4. Secret

```yaml
volumes:
- name: secrets
  secret:
    secretName: my-secret
    defaultMode: 0400  # Read-only by owner

volumeMounts:
- name: secrets
  mountPath: /secrets
  readOnly: true
```

**Use Case**: Sensitive data, credentials, certificates.

### 5. HostPath (Use with Caution)

```yaml
volumes:
- name: host-volume
  hostPath:
    path: /data
    type: Directory

volumeMounts:
- name: host-volume
  mountPath: /host-data
```

**Use Case**: Access node filesystem (security implications).

## Sharing Data Between Steps

### Using Workflow-Level Volume

```yaml
spec:
  volumes:
  - name: shared-data
    emptyDir: {}

  templates:
  - name: pipeline
    steps:
    # Step 1: Create data
    - - name: create
        template: create-files

    # Step 2: Transform data
    - - name: transform
        template: transform-files

    # Step 3: Upload data
    - - name: upload
        template: upload-files

  - name: create-files
    container:
      image: alpine
      command: [sh, -c, "echo 'data' > /shared/input.txt"]
      volumeMounts:
      - name: shared-data
        mountPath: /shared

  - name: transform-files
    container:
      image: alpine
      command: [sh, -c, "cat /shared/input.txt | tr '[:lower:]' '[:upper:]' > /shared/output.txt"]
      volumeMounts:
      - name: shared-data
        mountPath: /shared

  - name: upload-files
    container:
      image: alpine
      command: [sh, -c, "cat /shared/output.txt"]
      volumeMounts:
      - name: shared-data
        mountPath: /shared
```

### Dynamic Volume Provisioning

```yaml
spec:
  # Create PVC dynamically for this workflow
  volumeClaimTemplates:
  - metadata:
      name: workdir
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 5Gi

  templates:
  - name: task
    container:
      image: alpine
      command: [sh, -c]
      args: ["echo 'data' > /work/file.txt"]
      volumeMounts:
      - name: workdir
        mountPath: /work
```

## DAG with Shared Volumes

```yaml
- name: dag-with-volume
  dag:
    tasks:
    - name: init
      template: initialize

    # Parallel tasks sharing same volume
    - name: process-a
      dependencies: [init]
      template: processor-a

    - name: process-b
      dependencies: [init]
      template: processor-b

    # Final task reads all results
    - name: finalize
      dependencies: [process-a, process-b]
      template: finalizer

- name: initialize
  container:
    image: alpine
    command: [sh, -c, "mkdir -p /shared/results"]
    volumeMounts:
    - name: workdir
      mountPath: /shared

- name: processor-a
  container:
    image: alpine
    command: [sh, -c, "echo 'result-a' > /shared/results/a.txt"]
    volumeMounts:
    - name: workdir
      mountPath: /shared

- name: processor-b
  container:
    image: alpine
    command: [sh, -c, "echo 'result-b' > /shared/results/b.txt"]
    volumeMounts:
    - name: workdir
      mountPath: /shared

- name: finalizer
  container:
    image: alpine
    command: [sh, -c, "cat /shared/results/*.txt"]
    volumeMounts:
    - name: workdir
      mountPath: /shared
```

## Advanced Patterns

### Multiple Volume Mounts

```yaml
- name: multi-volume-task
  container:
    image: myapp:latest
    volumeMounts:
    - name: data
      mountPath: /data
    - name: config
      mountPath: /etc/config
    - name: secrets
      mountPath: /var/secrets
    - name: cache
      mountPath: /tmp/cache
```

### Read-Only Mounts

```yaml
volumeMounts:
- name: source-code
  mountPath: /src
  readOnly: true  # Prevent modifications
```

### SubPath Mounts

```yaml
# Mount only a specific directory from PVC
volumeMounts:
- name: shared-pvc
  mountPath: /app/data
  subPath: my-workflow/data  # Only mount this subdirectory
```

### ConfigMap as Environment Variables

```yaml
container:
  image: alpine
  env:
  - name: DATABASE_HOST
    valueFrom:
      configMapKeyRef:
        name: app-config
        key: db-host
```

### Secret as Environment Variables

```yaml
container:
  image: alpine
  env:
  - name: API_KEY
    valueFrom:
      secretKeyRef:
        name: api-credentials
        key: api-key
```

## Volume Lifecycle

### Workflow-Scoped Volume

```yaml
# Volume lives for duration of workflow
spec:
  volumes:
  - name: temp-data
    emptyDir: {}
# Deleted when workflow completes
```

### External PVC (Persistent)

```yaml
# Volume exists before and after workflow
volumes:
- name: persistent-data
  persistentVolumeClaim:
    claimName: existing-pvc
# Must be created before workflow runs
```

### Dynamic PVC (Workflow-Owned)

```yaml
# Created with workflow, deleted with workflow
volumeClaimTemplates:
- metadata:
    name: workflow-storage
  spec:
    accessModes: ["ReadWriteOnce"]
    resources:
      requests:
        storage: 10Gi
```

## Best Practices

1. **Use PVC for Shared Data**: When steps need to share files
2. **EmptyDir for Temporary Data**: For scratch space within workflow
3. **ConfigMaps for Configuration**: Keep config separate from code
4. **Secrets for Sensitive Data**: Never hardcode credentials
5. **Read-Only Mounts**: When data shouldn't be modified
6. **Size Limits on EmptyDir**: Prevent disk exhaustion
7. **Clean Up Dynamic PVCs**: Ensure proper cleanup policies

## Common Issues

### PVC Not Bound

```bash
# Check PVC status
kubectl get pvc my-workflow-pvc

# Check if PV is available
kubectl get pv
```

### Volume Mount Conflicts

```yaml
# Issue: Two mounts to same path
volumeMounts:
- name: volume-a
  mountPath: /data
- name: volume-b
  mountPath: /data  # Conflict!

# Fix: Use different paths
volumeMounts:
- name: volume-a
  mountPath: /data/a
- name: volume-b
  mountPath: /data/b
```

### Permission Issues

```yaml
# Fix: Set appropriate fsGroup
spec:
  securityContext:
    fsGroup: 2000  # Files owned by group 2000
    runAsUser: 1000
```

## Next Steps

- See `argo://examples/artifacts` for automatic artifact storage
- Explore `argo://examples/multi-step` for data passing between steps
- Check `argo://examples/resource-management` for storage resource limits
