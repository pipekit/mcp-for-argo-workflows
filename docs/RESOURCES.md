# MCP Resources

This document describes the MCP resources provided by mcp-for-argo-workflows.

## Overview

Resources in MCP are static or dynamic content that can be read by clients. The mcp-for-argo-workflows server provides schema documentation resources for all Argo Workflows CRDs.

## Available Resources

### 1. Workflow Schema

- **URI**: `argo://schemas/workflow`
- **Name**: `workflow-schema`
- **Title**: Argo Workflow CRD Schema
- **Description**: Complete schema documentation for the Workflow custom resource definition
- **MIME Type**: `text/markdown`

The Workflow schema resource provides comprehensive documentation about the Workflow CRD including:
- API version and kind
- Metadata fields
- Spec fields (templates, entrypoint, arguments, execution control, pod configuration, etc.)
- Template schema (container, script, resource, suspend, steps, dag)
- Status fields
- Common patterns and examples
- Required fields summary

### 2. WorkflowTemplate Schema

- **URI**: `argo://schemas/workflow-template`
- **Name**: `workflow-template-schema`
- **Title**: Argo WorkflowTemplate CRD Schema
- **Description**: Complete schema documentation for the WorkflowTemplate custom resource definition
- **MIME Type**: `text/markdown`

The WorkflowTemplate schema resource provides documentation about WorkflowTemplates including:
- Key differences from Workflow (no status, reusable, namespace-scoped)
- Metadata and spec fields
- Usage patterns (referencing from workflows, CLI submission)
- Complete examples with DAG, artifacts, resource limits
- Best practices

### 3. ClusterWorkflowTemplate Schema

- **URI**: `argo://schemas/cluster-workflow-template`
- **Name**: `cluster-workflow-template-schema`
- **Title**: Argo ClusterWorkflowTemplate CRD Schema
- **Description**: Complete schema documentation for the ClusterWorkflowTemplate custom resource definition
- **MIME Type**: `text/markdown`

The ClusterWorkflowTemplate schema resource provides documentation about cluster-scoped templates including:
- Key differences from WorkflowTemplate (cluster-scoped, requires RBAC)
- When to use ClusterWorkflowTemplate vs WorkflowTemplate
- RBAC requirements
- Usage patterns and examples
- Multi-stage CI/CD and data processing examples
- Migration guide from WorkflowTemplate

### 4. CronWorkflow Schema

- **URI**: `argo://schemas/cron-workflow`
- **Name**: `cron-workflow-schema`
- **Title**: Argo CronWorkflow CRD Schema
- **Description**: Complete schema documentation for the CronWorkflow custom resource definition
- **MIME Type**: `text/markdown`

The CronWorkflow schema resource provides documentation about scheduled workflows including:
- Schedule configuration (cron format, timezone)
- Concurrency control (Allow, Forbid, Replace)
- Workflow specification (embedded or template reference)
- History management
- Cron schedule format reference
- Timezone handling
- Complete examples (simple scheduled, parameterized, high-frequency monitoring)
- Management operations (suspend/resume, manual trigger)

## Using Resources in Claude

When using Claude Code or other MCP clients, these resources are automatically available. Claude can reference them to provide accurate information about Argo Workflows CRDs.

### Example Queries

1. "What are the required fields for a Workflow?"
   - Claude can read `argo://schemas/workflow` to provide accurate information

2. "How do I create a CronWorkflow that runs daily at 9 AM?"
   - Claude can reference `argo://schemas/cron-workflow` for schedule syntax and examples

3. "What's the difference between WorkflowTemplate and ClusterWorkflowTemplate?"
   - Claude can compare the two schema resources to explain the differences

4. "Show me an example of a DAG workflow"
   - Claude can extract examples from `argo://schemas/workflow`

## Implementation Details

Resources are implemented in `/pkg/resources/` with the following structure:

- `workflow_schema.go` - Workflow schema resource and handler
- `workflow_template_schema.go` - WorkflowTemplate schema resource and handler
- `cluster_workflow_template_schema.go` - ClusterWorkflowTemplate schema resource and handler
- `cron_workflow_schema.go` - CronWorkflow schema resource and handler
- `registry.go` - Resource registration logic

Each resource is defined as:
1. A resource definition (metadata about the resource)
2. A handler function that returns the resource contents when requested

All resources are registered with the MCP server on startup via `server.RegisterResources()`.

## Benefits

1. **Always Available**: Schema documentation is embedded in the binary, no external dependencies
2. **Accurate**: Documentation is version-controlled with the code
3. **Structured**: Markdown format is easy for LLMs to parse and understand
4. **Comprehensive**: Covers all major CRD types and their fields
5. **Practical**: Includes examples and common patterns

## Future Enhancements

Potential future resource additions:
- Individual field reference (e.g., `argo://schemas/workflow/spec/templates`)
- Common pattern library (e.g., `argo://patterns/ci-cd`)
- Troubleshooting guides (e.g., `argo://troubleshooting/failed-workflows`)
- API version migration guides
