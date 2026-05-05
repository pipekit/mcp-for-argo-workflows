# Argo Workflows Template Types Overview

Templates are the building blocks of Argo Workflows. Each template defines a unit of work and must specify exactly ONE template type.

## Available Resources

For detailed documentation on each template type, use these resources:

- **argo://docs/template-types/container** - Container template documentation
- **argo://docs/template-types/script** - Script template documentation
- **argo://docs/template-types/dag** - DAG template documentation
- **argo://docs/template-types/steps** - Steps template documentation
- **argo://docs/template-types/suspend** - Suspend template documentation
- **argo://docs/template-types/resource** - Resource template documentation
- **argo://docs/template-types/http** - HTTP template documentation

## Template Types Quick Reference

Argo Workflows supports several template types:

### Execution Templates
- **Container** - Run containers with specific images and commands
- **Script** - Execute inline code/scripts
- **Resource** - Manage Kubernetes resources
- **HTTP** - Make HTTP requests
- **Suspend** - Pause workflow execution

### Orchestration Templates
- **Steps** - Sequential and parallel step groups
- **DAG** - Tasks with explicit dependencies

## Choosing the Right Template Type

- Use **Container** for running existing Docker images
- Use **Script** for inline code that doesn't need a custom image
- Use **Resource** for creating/managing Kubernetes resources
- Use **HTTP** for calling REST APIs
- Use **Suspend** for approval gates or delays
- Use **Steps** for simple sequential/parallel workflows
- Use **DAG** for complex dependencies and maximum parallelism