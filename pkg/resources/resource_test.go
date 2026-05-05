package resources

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllDefinitions(t *testing.T) {
	// Should have exactly 29 resource definitions (4 schema + 5 data flow + 8 template types + 12 examples)
	assert.Len(t, AllDefinitions(), 29, "Expected 29 resource definitions")

	// Verify all definitions have required fields
	for _, def := range AllDefinitions() {
		t.Run(def.Name, func(t *testing.T) {
			assert.NotEmpty(t, def.URI, "URI should not be empty")
			assert.NotEmpty(t, def.Name, "Name should not be empty")
			assert.NotEmpty(t, def.Title, "Title should not be empty")
			assert.NotEmpty(t, def.Description, "Description should not be empty")
			assert.NotEmpty(t, def.DocFile, "DocFile should not be empty")
		})
	}
}

func TestDefinitionResource(t *testing.T) {
	for _, def := range AllDefinitions() {
		t.Run(def.Name, func(t *testing.T) {
			resource := def.Resource()

			assert.Equal(t, def.URI, resource.URI)
			assert.Equal(t, def.Name, resource.Name)
			assert.Equal(t, def.Title, resource.Title)
			assert.Equal(t, def.Description, resource.Description)
			assert.Equal(t, "text/markdown", resource.MIMEType)
		})
	}
}

func TestDefinitionHandler(t *testing.T) {
	for _, def := range AllDefinitions() {
		t.Run(def.Name+"_valid_uri", func(t *testing.T) {
			ctx := t.Context()
			handler := def.Handler()
			req := &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: def.URI,
				},
			}

			result, err := handler(ctx, req)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Contents, 1)

			content := result.Contents[0]
			assert.Equal(t, def.URI, content.URI)
			assert.Equal(t, "text/markdown", content.MIMEType)
			assert.NotEmpty(t, content.Text)
		})

		t.Run(def.Name+"_invalid_uri", func(t *testing.T) {
			ctx := t.Context()
			handler := def.Handler()
			req := &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: "argo://invalid/uri",
				},
			}

			result, err := handler(ctx, req)
			require.Error(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestRegisterAll(t *testing.T) {
	implementation := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}
	server := mcp.NewServer(implementation, nil)

	// Register all resources - should not panic
	RegisterAll(server)
}

// Content validation tests for specific resources

func TestWorkflowSchemaContent(t *testing.T) {
	def := findDefinition("workflow-schema")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "# Workflow CRD Schema")
	assert.Contains(t, content, "apiVersion")
	assert.Contains(t, content, "kind")
	assert.Contains(t, content, "spec")
}

func TestWorkflowTemplateSchemaContent(t *testing.T) {
	def := findDefinition("workflow-template-schema")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "# WorkflowTemplate CRD Schema")
	assert.Contains(t, content, "No Status")
	assert.Contains(t, content, "Reusable")
}

func TestClusterWorkflowTemplateSchemaContent(t *testing.T) {
	def := findDefinition("cluster-workflow-template-schema")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "# ClusterWorkflowTemplate CRD Schema")
	assert.Contains(t, content, "Cluster-Scoped")
	assert.Contains(t, content, "RBAC")
}

func TestCronWorkflowSchemaContent(t *testing.T) {
	def := findDefinition("cron-workflow-schema")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "# CronWorkflow CRD Schema")
	assert.Contains(t, content, "schedule")
	assert.Contains(t, content, "timezone")
	assert.Contains(t, content, "concurrencyPolicy")
}

func TestTemplateTypesOverviewContent(t *testing.T) {
	def := findDefinition("template-types-overview")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "Template Types Overview")
	assert.Contains(t, content, "Container")
	assert.Contains(t, content, "Script")
	assert.Contains(t, content, "DAG")
	assert.Contains(t, content, "Steps")
}

func TestTemplateTypesContainerContent(t *testing.T) {
	def := findDefinition("template-types-container")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "Container Template Type")
	assert.Contains(t, content, "image")
	assert.Contains(t, content, "command")
	assert.Contains(t, content, "args")
}

func TestTemplateTypesScriptContent(t *testing.T) {
	def := findDefinition("template-types-script")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "Script Template Type")
	assert.Contains(t, content, "source")
	assert.Contains(t, content, "inline")
}

func TestTemplateTypesDAGContent(t *testing.T) {
	def := findDefinition("template-types-dag")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "DAG Template Type")
	assert.Contains(t, content, "dependencies")
	assert.Contains(t, content, "tasks")
}

func TestTemplateTypesStepsContent(t *testing.T) {
	def := findDefinition("template-types-steps")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "Steps Template Type")
	assert.Contains(t, content, "sequential")
	assert.Contains(t, content, "parallel")
}

func TestTemplateTypesSuspendContent(t *testing.T) {
	def := findDefinition("template-types-suspend")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "Suspend Template Type")
	assert.Contains(t, content, "duration")
	assert.Contains(t, content, "approval")
}

func TestTemplateTypesResourceContent(t *testing.T) {
	def := findDefinition("template-types-resource")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "Resource Template Type")
	assert.Contains(t, content, "manifest")
	assert.Contains(t, content, "action")
	assert.Contains(t, content, "Kubernetes")
}

func TestTemplateTypesHTTPContent(t *testing.T) {
	def := findDefinition("template-types-http")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "HTTP Template Type")
	assert.Contains(t, content, "url")
	assert.Contains(t, content, "method")
	assert.Contains(t, content, "HTTP requests")
}

func TestExamplesHelloWorldContent(t *testing.T) {
	def := findDefinition("examples-hello-world")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "Hello World")
	assert.Contains(t, content, "apiVersion")
	assert.Contains(t, content, "kind: Workflow")
}

func TestExamplesMultiStepContent(t *testing.T) {
	def := findDefinition("examples-multi-step")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "Multi-Step")
	assert.Contains(t, content, "steps")
	assert.Contains(t, content, "outputs")
}

func TestExamplesDAGDiamondContent(t *testing.T) {
	def := findDefinition("examples-dag-diamond")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "diamond")
	assert.Contains(t, content, "dag")
	assert.Contains(t, content, "dependencies")
}

func TestExamplesParametersContent(t *testing.T) {
	def := findDefinition("examples-parameters")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "parameters")
	assert.Contains(t, content, "arguments")
	assert.Contains(t, content, "inputs")
}

func TestExamplesArtifactsContent(t *testing.T) {
	def := findDefinition("examples-artifacts")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "artifacts")
	assert.Contains(t, content, "S3")
	assert.Contains(t, content, "path")
}

func TestExamplesLoopsContent(t *testing.T) {
	def := findDefinition("examples-loops")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "withItems")
	assert.Contains(t, content, "withParam")
	assert.Contains(t, content, "withSequence")
}

func TestExamplesConditionalsContent(t *testing.T) {
	def := findDefinition("examples-conditionals")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "when")
	assert.Contains(t, content, "conditional")
}

func TestExamplesRetriesContent(t *testing.T) {
	def := findDefinition("examples-retries")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "retryStrategy")
	assert.Contains(t, content, "backoff")
	assert.Contains(t, content, "limit")
}

func TestExamplesTimeoutLimitsContent(t *testing.T) {
	def := findDefinition("examples-timeout-limits")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "activeDeadlineSeconds")
	assert.Contains(t, content, "timeout")
}

func TestExamplesResourceManagementContent(t *testing.T) {
	def := findDefinition("examples-resource-management")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "resources")
	assert.Contains(t, content, "requests")
	assert.Contains(t, content, "limits")
	assert.Contains(t, content, "cpu")
	assert.Contains(t, content, "memory")
}

func TestExamplesVolumesContent(t *testing.T) {
	def := findDefinition("examples-volumes")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "volumes")
	assert.Contains(t, content, "volumeMounts")
	assert.Contains(t, content, "PersistentVolumeClaim")
}

func TestExamplesExitHandlersContent(t *testing.T) {
	def := findDefinition("examples-exit-handlers")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "onExit")
	assert.Contains(t, content, "exit")
	assert.Contains(t, content, "workflow.status")
}

// Data Flow Resources Tests

func TestDocsVariablesContent(t *testing.T) {
	def := findDefinition("docs-variables")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "Variables Reference")
	assert.Contains(t, content, "inputs.parameters")
	assert.Contains(t, content, "outputs.parameters")
	assert.Contains(t, content, "workflow.name")
	assert.Contains(t, content, "steps.")
	assert.Contains(t, content, "tasks.")
	assert.Contains(t, content, "{{item}}")
}

func TestDocsExpressionsContent(t *testing.T) {
	def := findDefinition("docs-expressions")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "Expressions Reference")
	assert.Contains(t, content, "{{=")
	assert.Contains(t, content, "sprig")
	assert.Contains(t, content, "jsonpath")
	assert.Contains(t, content, "fromJson")
	assert.Contains(t, content, "asInt")
}

func TestDocsParametersContent(t *testing.T) {
	def := findDefinition("docs-parameters")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "Parameters Reference")
	assert.Contains(t, content, "arguments")
	assert.Contains(t, content, "inputs")
	assert.Contains(t, content, "valueFrom")
	assert.Contains(t, content, "globalName")
	assert.Contains(t, content, "enum")
}

func TestDocsArtifactsContent(t *testing.T) {
	def := findDefinition("docs-artifacts")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "Artifacts Reference")
	assert.Contains(t, content, "inputs")
	assert.Contains(t, content, "outputs")
	assert.Contains(t, content, "path")
	assert.Contains(t, content, "s3:")
	assert.Contains(t, content, "gcs:")
	assert.Contains(t, content, "git:")
}

func TestDocsOutputsContent(t *testing.T) {
	def := findDefinition("docs-outputs")
	require.NotNil(t, def)

	content := getContent(t, def)
	assert.Contains(t, content, "Outputs Reference")
	assert.Contains(t, content, "valueFrom")
	assert.Contains(t, content, "path")
	assert.Contains(t, content, "outputs.result")
	assert.Contains(t, content, "globalName")
	assert.Contains(t, content, "exitCode")
}

// Helper functions

func findDefinition(name string) *Definition {
	defs := AllDefinitions()
	for i := range defs {
		if defs[i].Name == name {
			return &defs[i]
		}
	}
	return nil
}

func getContent(t *testing.T, def *Definition) string {
	t.Helper()
	ctx := t.Context()
	handler := def.Handler()
	req := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: def.URI,
		},
	}
	result, err := handler(ctx, req)
	require.NoError(t, err)
	return result.Contents[0].Text
}
