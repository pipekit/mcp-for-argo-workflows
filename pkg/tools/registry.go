// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"fmt"
	"slices"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/pipekit/mcp-for-argo-workflows/pkg/argo"
)

// ToolRegistrar is a function that registers a tool with the MCP server.
// Each tool provides its own registrar that calls mcp.AddTool with the correct types.
type ToolRegistrar func(s *mcp.Server, client argo.ClientInterface)

// ToolDefinition is a function that returns the MCP tool definition.
type ToolDefinition func() *mcp.Tool

type toolSpec struct {
	register ToolRegistrar
	tool     ToolDefinition
	// multiContextOnly marks tools that only exist when per-call kubeconfig
	// context selection is available.
	multiContextOnly bool
}

func allToolSpecs() []toolSpec {
	return []toolSpec{
		{register: RegisterSubmitWorkflow, tool: SubmitWorkflowTool},
		{register: RegisterListWorkflows, tool: ListWorkflowsTool},
		{register: RegisterGetWorkflow, tool: GetWorkflowTool},
		{register: RegisterDeleteWorkflow, tool: DeleteWorkflowTool},
		{register: RegisterWatchWorkflow, tool: WatchWorkflowTool},
		{register: RegisterLogsWorkflow, tool: LogsWorkflowTool},
		{register: RegisterWaitWorkflow, tool: WaitWorkflowTool},
		{register: RegisterLintWorkflow, tool: LintWorkflowTool},
		{register: RegisterLintWorkflowTemplate, tool: LintWorkflowTemplateTool},
		{register: RegisterLintClusterWorkflowTemplate, tool: LintClusterWorkflowTemplateTool},
		{register: RegisterLintCronWorkflow, tool: LintCronWorkflowTool},
		{register: RegisterRetryWorkflow, tool: RetryWorkflowTool},
		{register: RegisterResubmitWorkflow, tool: ResubmitWorkflowTool},
		{register: RegisterSuspendWorkflow, tool: SuspendWorkflowTool},
		{register: RegisterResumeWorkflow, tool: ResumeWorkflowTool},
		{register: RegisterStopWorkflow, tool: StopWorkflowTool},
		{register: RegisterTerminateWorkflow, tool: TerminateWorkflowTool},
		{register: RegisterRenderWorkflowGraph, tool: RenderWorkflowGraphTool},
		{register: RegisterRenderManifestGraph, tool: RenderManifestGraphTool},
		{register: RegisterListWorkflowTemplates, tool: ListWorkflowTemplatesTool},
		{register: RegisterGetWorkflowTemplate, tool: GetWorkflowTemplateTool},
		{register: RegisterCreateWorkflowTemplate, tool: CreateWorkflowTemplateTool},
		{register: RegisterDeleteWorkflowTemplate, tool: DeleteWorkflowTemplateTool},
		{register: RegisterListClusterWorkflowTemplates, tool: ListClusterWorkflowTemplatesTool},
		{register: RegisterGetClusterWorkflowTemplate, tool: GetClusterWorkflowTemplateTool},
		{register: RegisterCreateClusterWorkflowTemplate, tool: CreateClusterWorkflowTemplateTool},
		{register: RegisterDeleteClusterWorkflowTemplate, tool: DeleteClusterWorkflowTemplateTool},
		{register: RegisterListCronWorkflows, tool: ListCronWorkflowsTool},
		{register: RegisterGetCronWorkflow, tool: GetCronWorkflowTool},
		{register: RegisterCreateCronWorkflow, tool: CreateCronWorkflowTool},
		{register: RegisterDeleteCronWorkflow, tool: DeleteCronWorkflowTool},
		{register: RegisterSuspendCronWorkflow, tool: SuspendCronWorkflowTool},
		{register: RegisterResumeCronWorkflow, tool: ResumeCronWorkflowTool},
		{register: RegisterGetWorkflowNode, tool: GetWorkflowNodeTool},
		{register: RegisterDeleteArchivedWorkflow, tool: DeleteArchivedWorkflowTool},
		{register: RegisterResubmitArchivedWorkflow, tool: ResubmitArchivedWorkflowTool},
		{register: RegisterRetryArchivedWorkflow, tool: RetryArchivedWorkflowTool},
		{register: RegisterConvertWorkflow, tool: ConvertWorkflowTool},
		{register: RegisterListContexts, tool: ListContextsTool, multiContextOnly: true},
	}
}

func isReadOnlyTool(t *mcp.Tool) bool {
	return t.Annotations != nil && t.Annotations.ReadOnlyHint
}

func toolRegistrars(readOnly, multiContext bool) []ToolRegistrar {
	registrars := make([]ToolRegistrar, 0, len(allToolSpecs()))
	for _, spec := range allToolSpecs() {
		if readOnly && !isReadOnlyTool(spec.tool()) {
			continue
		}
		if spec.multiContextOnly && !multiContext {
			continue
		}
		registrars = append(registrars, spec.register)
	}
	return registrars
}

// RegisterAll registers all tools with the MCP server. Whether per-call
// kubeconfig context selection is available travels on the client.
func RegisterAll(s *mcp.Server, client argo.ClientInterface, readOnly bool) {
	for _, register := range toolRegistrars(readOnly, client.MultiContextEnabled()) {
		register(s, client)
	}
}

// addTool registers a tool, stripping the kubeconfig context parameter from
// its input schema when per-call context selection is unavailable. The
// stripped schema is presentation plus first-line validation (the schema
// rejects unknown properties); the enforcing control is the client's
// ForKubeContext rejecting any non-empty context name.
func addTool[In, Out any](s *mcp.Server, client argo.ClientInterface, t *mcp.Tool, h mcp.ToolHandlerFor[In, Out]) {
	if !client.MultiContextEnabled() {
		t.InputSchema = schemaWithoutKubeContext[In](t.Name)
	}
	mcp.AddTool(s, t, h)
}

// schemaWithoutKubeContext infers the input schema for In and removes the
// "context" property so it is neither advertised nor accepted.
func schemaWithoutKubeContext[In any](toolName string) *jsonschema.Schema {
	schema, err := jsonschema.For[In](&jsonschema.ForOptions{})
	if err != nil {
		// mcp.AddTool panics on schema inference failure; mirror that.
		panic(fmt.Sprintf("tool %q: inferring input schema: %v", toolName, err))
	}
	delete(schema.Properties, "context")
	schema.PropertyOrder = slices.DeleteFunc(schema.PropertyOrder, func(p string) bool { return p == "context" })
	return schema
}

// Individual tool registrars - these wrap mcp.AddTool with the correct type parameters.

// RegisterSubmitWorkflow registers the submit_workflow tool.
func RegisterSubmitWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, SubmitWorkflowTool(), SubmitWorkflowHandler(client))
}

// RegisterListWorkflows registers the list_workflows tool.
func RegisterListWorkflows(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, ListWorkflowsTool(), ListWorkflowsHandler(client))
}

// RegisterGetWorkflow registers the get_workflow tool.
func RegisterGetWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, GetWorkflowTool(), GetWorkflowHandler(client))
}

// RegisterDeleteWorkflow registers the delete_workflow tool.
func RegisterDeleteWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, DeleteWorkflowTool(), DeleteWorkflowHandler(client))
}

// RegisterWatchWorkflow registers the watch_workflow tool.
func RegisterWatchWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, WatchWorkflowTool(), WatchWorkflowHandler(client))
}

// RegisterLogsWorkflow registers the logs_workflow tool.
func RegisterLogsWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, LogsWorkflowTool(), LogsWorkflowHandler(client))
}

// RegisterWaitWorkflow registers the wait_workflow tool.
func RegisterWaitWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, WaitWorkflowTool(), WaitWorkflowHandler(client))
}

// RegisterLintWorkflow registers the lint_workflow tool.
func RegisterLintWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, LintWorkflowTool(), LintWorkflowHandler(client))
}

// RegisterLintWorkflowTemplate registers the lint_workflow_template tool.
func RegisterLintWorkflowTemplate(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, LintWorkflowTemplateTool(), LintWorkflowTemplateHandler(client))
}

// RegisterLintClusterWorkflowTemplate registers the lint_cluster_workflow_template tool.
func RegisterLintClusterWorkflowTemplate(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, LintClusterWorkflowTemplateTool(), LintClusterWorkflowTemplateHandler(client))
}

// RegisterLintCronWorkflow registers the lint_cron_workflow tool.
func RegisterLintCronWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, LintCronWorkflowTool(), LintCronWorkflowHandler(client))
}

// RegisterRetryWorkflow registers the retry_workflow tool.
func RegisterRetryWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, RetryWorkflowTool(), RetryWorkflowHandler(client))
}

// RegisterResubmitWorkflow registers the resubmit_workflow tool.
func RegisterResubmitWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, ResubmitWorkflowTool(), ResubmitWorkflowHandler(client))
}

// RegisterSuspendWorkflow registers the suspend_workflow tool.
func RegisterSuspendWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, SuspendWorkflowTool(), SuspendWorkflowHandler(client))
}

// RegisterResumeWorkflow registers the resume_workflow tool.
func RegisterResumeWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, ResumeWorkflowTool(), ResumeWorkflowHandler(client))
}

// RegisterStopWorkflow registers the stop_workflow tool.
func RegisterStopWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, StopWorkflowTool(), StopWorkflowHandler(client))
}

// RegisterTerminateWorkflow registers the terminate_workflow tool.
func RegisterTerminateWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, TerminateWorkflowTool(), TerminateWorkflowHandler(client))
}

// RegisterListWorkflowTemplates registers the list_workflow_templates tool.
func RegisterListWorkflowTemplates(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, ListWorkflowTemplatesTool(), ListWorkflowTemplatesHandler(client))
}

// RegisterGetWorkflowTemplate registers the get_workflow_template tool.
func RegisterGetWorkflowTemplate(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, GetWorkflowTemplateTool(), GetWorkflowTemplateHandler(client))
}

// RegisterCreateWorkflowTemplate registers the create_workflow_template tool.
func RegisterCreateWorkflowTemplate(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, CreateWorkflowTemplateTool(), CreateWorkflowTemplateHandler(client))
}

// RegisterDeleteWorkflowTemplate registers the delete_workflow_template tool.
func RegisterDeleteWorkflowTemplate(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, DeleteWorkflowTemplateTool(), DeleteWorkflowTemplateHandler(client))
}

// RegisterListClusterWorkflowTemplates registers the list_cluster_workflow_templates tool.
func RegisterListClusterWorkflowTemplates(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, ListClusterWorkflowTemplatesTool(), ListClusterWorkflowTemplatesHandler(client))
}

// RegisterRenderWorkflowGraph registers the render_workflow_graph tool.
func RegisterRenderWorkflowGraph(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, RenderWorkflowGraphTool(), RenderWorkflowGraphHandler(client))
}

// RegisterRenderManifestGraph registers the render_manifest_graph tool.
func RegisterRenderManifestGraph(s *mcp.Server, _ argo.ClientInterface) {
	mcp.AddTool(s, RenderManifestGraphTool(), RenderManifestGraphHandler())
}

// RegisterGetClusterWorkflowTemplate registers the get_cluster_workflow_template tool.
func RegisterGetClusterWorkflowTemplate(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, GetClusterWorkflowTemplateTool(), GetClusterWorkflowTemplateHandler(client))
}

// RegisterCreateClusterWorkflowTemplate registers the create_cluster_workflow_template tool.
func RegisterCreateClusterWorkflowTemplate(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, CreateClusterWorkflowTemplateTool(), CreateClusterWorkflowTemplateHandler(client))
}

// RegisterDeleteClusterWorkflowTemplate registers the delete_cluster_workflow_template tool.
func RegisterDeleteClusterWorkflowTemplate(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, DeleteClusterWorkflowTemplateTool(), DeleteClusterWorkflowTemplateHandler(client))
}

// RegisterListCronWorkflows registers the list_cron_workflows tool.
func RegisterListCronWorkflows(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, ListCronWorkflowsTool(), ListCronWorkflowsHandler(client))
}

// RegisterGetCronWorkflow registers the get_cron_workflow tool.
func RegisterGetCronWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, GetCronWorkflowTool(), GetCronWorkflowHandler(client))
}

// RegisterCreateCronWorkflow registers the create_cron_workflow tool.
func RegisterCreateCronWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, CreateCronWorkflowTool(), CreateCronWorkflowHandler(client))
}

// RegisterDeleteCronWorkflow registers the delete_cron_workflow tool.
func RegisterDeleteCronWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, DeleteCronWorkflowTool(), DeleteCronWorkflowHandler(client))
}

// RegisterSuspendCronWorkflow registers the suspend_cron_workflow tool.
func RegisterSuspendCronWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, SuspendCronWorkflowTool(), SuspendCronWorkflowHandler(client))
}

// RegisterResumeCronWorkflow registers the resume_cron_workflow tool.
func RegisterResumeCronWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, ResumeCronWorkflowTool(), ResumeCronWorkflowHandler(client))
}

// RegisterGetWorkflowNode registers the get_workflow_node tool.
func RegisterGetWorkflowNode(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, GetWorkflowNodeTool(), GetWorkflowNodeHandler(client))
}

// RegisterDeleteArchivedWorkflow registers the delete_archived_workflow tool.
func RegisterDeleteArchivedWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, DeleteArchivedWorkflowTool(), DeleteArchivedWorkflowHandler(client))
}

// RegisterResubmitArchivedWorkflow registers the resubmit_archived_workflow tool.
func RegisterResubmitArchivedWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, ResubmitArchivedWorkflowTool(), ResubmitArchivedWorkflowHandler(client))
}

// RegisterRetryArchivedWorkflow registers the retry_archived_workflow tool.
func RegisterRetryArchivedWorkflow(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, RetryArchivedWorkflowTool(), RetryArchivedWorkflowHandler(client))
}

// RegisterConvertWorkflow registers the convert_workflow tool.
func RegisterConvertWorkflow(s *mcp.Server, _ argo.ClientInterface) {
	mcp.AddTool(s, ConvertWorkflowTool(), ConvertWorkflowHandler())
}

// RegisterListContexts registers the list_contexts tool.
func RegisterListContexts(s *mcp.Server, client argo.ClientInterface) {
	addTool(s, client, ListContextsTool(), ListContextsHandler(client))
}
