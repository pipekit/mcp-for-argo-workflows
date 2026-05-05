package tools

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderManifestGraphTool(t *testing.T) {
	tool := RenderManifestGraphTool()

	assert.Equal(t, "render_manifest_graph", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "YAML")
}

func TestRenderManifestGraphHandler(t *testing.T) {
	//nolint:govet // fieldalignment - test struct readability over alignment
	tests := []struct {
		validate      func(t *testing.T, output *RenderManifestGraphOutput)
		input         RenderManifestGraphInput
		name          string
		expectedError string
	}{
		{
			name: "simple DAG workflow with mermaid format",
			input: RenderManifestGraphInput{
				Manifest: dagWorkflowManifest,
				Format:   "mermaid",
			},
			validate: func(t *testing.T, output *RenderManifestGraphOutput) {
				assert.Equal(t, "mermaid", output.Format)
				assert.Equal(t, "Workflow", output.Kind)
				assert.Equal(t, "dag-example", output.Name)
				assert.Equal(t, 3, output.NodeCount)
				assert.Contains(t, output.Graph, "flowchart TD")
				assert.Contains(t, output.Graph, "build")
				assert.Contains(t, output.Graph, "test")
				assert.Contains(t, output.Graph, "deploy")
				assert.Contains(t, output.Graph, "-->")
			},
		},
		{
			name: "DAG workflow with ASCII format",
			input: RenderManifestGraphInput{
				Manifest: dagWorkflowManifest,
				Format:   "ascii",
			},
			validate: func(t *testing.T, output *RenderManifestGraphOutput) {
				assert.Equal(t, "ascii", output.Format)
				assert.Contains(t, output.Graph, "build")
			},
		},
		{
			name: "DAG workflow with DOT format",
			input: RenderManifestGraphInput{
				Manifest: dagWorkflowManifest,
				Format:   "dot",
			},
			validate: func(t *testing.T, output *RenderManifestGraphOutput) {
				assert.Equal(t, "dot", output.Format)
				assert.Contains(t, output.Graph, "digraph workflow")
				assert.Contains(t, output.Graph, "build")
				assert.Contains(t, output.Graph, "->")
			},
		},
		{
			name: "DAG workflow with SVG format",
			input: RenderManifestGraphInput{
				Manifest: dagWorkflowManifest,
				Format:   "svg",
			},
			validate: func(t *testing.T, output *RenderManifestGraphOutput) {
				assert.Equal(t, "svg", output.Format)
				assert.True(t, strings.HasPrefix(output.Graph, "<?xml") || strings.Contains(output.Graph, "<svg"))
				assert.Contains(t, output.Graph, "</svg>")
				// Note: SVG may HTML-encode special chars, so we check for the basic word
				assert.Contains(t, output.Graph, "build")
			},
		},
		{
			name: "steps workflow",
			input: RenderManifestGraphInput{
				Manifest: stepsWorkflowManifest,
				Format:   "mermaid",
			},
			validate: func(t *testing.T, output *RenderManifestGraphOutput) {
				assert.Equal(t, "mermaid", output.Format)
				assert.Equal(t, "Workflow", output.Kind)
				assert.Equal(t, "steps-example", output.Name)
				assert.GreaterOrEqual(t, output.NodeCount, 2)
				assert.Contains(t, output.Graph, "flowchart TD")
			},
		},
		{
			name: "WorkflowTemplate manifest",
			input: RenderManifestGraphInput{
				Manifest: workflowTemplateManifest,
				Format:   "mermaid",
			},
			validate: func(t *testing.T, output *RenderManifestGraphOutput) {
				assert.Equal(t, "mermaid", output.Format)
				assert.Equal(t, "WorkflowTemplate", output.Kind)
				assert.Equal(t, "my-template", output.Name)
			},
		},
		{
			name: "ClusterWorkflowTemplate manifest",
			input: RenderManifestGraphInput{
				Manifest: clusterWorkflowTemplateManifest,
				Format:   "mermaid",
			},
			validate: func(t *testing.T, output *RenderManifestGraphOutput) {
				assert.Equal(t, "mermaid", output.Format)
				assert.Equal(t, "ClusterWorkflowTemplate", output.Kind)
				assert.Equal(t, "cluster-template", output.Name)
			},
		},
		{
			name: "CronWorkflow manifest",
			input: RenderManifestGraphInput{
				Manifest: cronWorkflowManifest,
				Format:   "mermaid",
			},
			validate: func(t *testing.T, output *RenderManifestGraphOutput) {
				assert.Equal(t, "mermaid", output.Format)
				assert.Equal(t, "CronWorkflow", output.Kind)
				assert.Equal(t, "my-cron", output.Name)
			},
		},
		{
			name: "default format is mermaid",
			input: RenderManifestGraphInput{
				Manifest: dagWorkflowManifest,
			},
			validate: func(t *testing.T, output *RenderManifestGraphOutput) {
				assert.Equal(t, "mermaid", output.Format)
				assert.Contains(t, output.Graph, "flowchart TD")
			},
		},
		{
			name: "empty manifest",
			input: RenderManifestGraphInput{
				Manifest: "",
				Format:   "mermaid",
			},
			expectedError: "manifest cannot be empty",
		},
		{
			name: "whitespace-only manifest",
			input: RenderManifestGraphInput{
				Manifest: "   \n\t  ",
				Format:   "mermaid",
			},
			expectedError: "manifest cannot be empty",
		},
		{
			name: "invalid YAML",
			input: RenderManifestGraphInput{
				Manifest: "this is not: valid: yaml: [",
				Format:   "mermaid",
			},
			expectedError: "failed to parse manifest",
		},
		{
			name: "invalid format",
			input: RenderManifestGraphInput{
				Manifest: dagWorkflowManifest,
				Format:   "invalid",
			},
			expectedError: "invalid format",
		},
		{
			name: "unsupported kind",
			input: RenderManifestGraphInput{
				Manifest: unsupportedKindManifest,
				Format:   "mermaid",
			},
			expectedError: "unsupported manifest kind",
		},
		{
			name: "missing entrypoint template",
			input: RenderManifestGraphInput{
				Manifest: missingEntrypointManifest,
				Format:   "mermaid",
			},
			expectedError: "entrypoint template",
		},
		{
			name: "workflow with conditional steps",
			input: RenderManifestGraphInput{
				Manifest: conditionalWorkflowManifest,
				Format:   "mermaid",
			},
			validate: func(t *testing.T, output *RenderManifestGraphOutput) {
				assert.Equal(t, "mermaid", output.Format)
				assert.Contains(t, output.Graph, "flowchart TD")
				// Conditional edges should use dashed lines
				assert.Contains(t, output.Graph, "-.->")
			},
		},
		{
			name: "workflow with loops",
			input: RenderManifestGraphInput{
				Manifest: loopWorkflowManifest,
				Format:   "mermaid",
			},
			validate: func(t *testing.T, output *RenderManifestGraphOutput) {
				assert.Equal(t, "mermaid", output.Format)
				assert.Contains(t, output.Graph, "[loop]")
			},
		},
		{
			name: "simple container template",
			input: RenderManifestGraphInput{
				Manifest: simpleContainerManifest,
				Format:   "mermaid",
			},
			validate: func(t *testing.T, output *RenderManifestGraphOutput) {
				assert.Equal(t, "mermaid", output.Format)
				assert.Equal(t, 1, output.NodeCount)
				assert.Contains(t, output.Graph, "hello-world")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := RenderManifestGraphHandler()
			result, output, err := handler(t.Context(), nil, tt.input)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
				assert.Nil(t, output)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				textContent, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok, "expected TextContent")
				assert.Contains(t, textContent.Text, "Rendered")
				require.NotNil(t, output)
				tt.validate(t, output)
			}
		})
	}
}

func TestExtractWorkflowSpec(t *testing.T) {
	tests := []struct {
		name          string
		manifest      string
		expectedKind  string
		expectedName  string
		expectedError string
	}{
		{
			name:         "Workflow",
			manifest:     dagWorkflowManifest,
			expectedKind: "Workflow",
			expectedName: "dag-example",
		},
		{
			name:         "WorkflowTemplate",
			manifest:     workflowTemplateManifest,
			expectedKind: "WorkflowTemplate",
			expectedName: "my-template",
		},
		{
			name:         "ClusterWorkflowTemplate",
			manifest:     clusterWorkflowTemplateManifest,
			expectedKind: "ClusterWorkflowTemplate",
			expectedName: "cluster-template",
		},
		{
			name:         "CronWorkflow",
			manifest:     cronWorkflowManifest,
			expectedKind: "CronWorkflow",
			expectedName: "my-cron",
		},
		{
			name: "workflow without kind defaults to Workflow",
			manifest: `
metadata:
  name: no-kind
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: alpine
`,
			expectedKind: "Workflow",
			expectedName: "no-kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec, kind, name, err := extractWorkflowSpec(tt.manifest)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, spec)
				assert.Equal(t, tt.expectedKind, kind)
				assert.Equal(t, tt.expectedName, name)
			}
		})
	}
}

func TestBuildGraphFromSpec(t *testing.T) {
	// Test DAG workflow
	spec, _, _, err := extractWorkflowSpec(dagWorkflowManifest)
	require.NoError(t, err)

	nodes, err := buildGraphFromSpec(spec)
	require.NoError(t, err)
	assert.Len(t, nodes, 3)
	assert.Contains(t, nodes, "build")
	assert.Contains(t, nodes, "test")
	assert.Contains(t, nodes, "deploy")

	// Check dependencies
	buildNode := nodes["build"]
	assert.Empty(t, buildNode.Dependencies)

	testNode := nodes["test"]
	assert.Contains(t, testNode.Dependencies, "build")

	deployNode := nodes["deploy"]
	assert.Contains(t, deployNode.Dependencies, "test")
}

func TestRenderManifestMermaid(t *testing.T) {
	nodes := map[string]*manifestNode{
		"A": {Name: "A", TemplateName: "tmpl-a", Type: "dag-task"},
		"B": {Name: "B", TemplateName: "tmpl-b", Type: "dag-task", Dependencies: []string{"A"}},
		"C": {Name: "C", TemplateName: "tmpl-c", Type: "dag-task", Dependencies: []string{"A"}, When: "{{tasks.A.outputs.result}} == 'success'"},
	}

	graph := renderManifestMermaid(nodes)

	assert.Contains(t, graph, "flowchart TD")
	assert.Contains(t, graph, "A[A")
	assert.Contains(t, graph, "B[B")
	assert.Contains(t, graph, "C[C")
	assert.Contains(t, graph, "-->")  // Normal edge
	assert.Contains(t, graph, "-.->") // Conditional edge
	assert.Contains(t, graph, "[conditional]")
}

func TestRenderManifestASCII(t *testing.T) {
	nodes := map[string]*manifestNode{
		"root":  {Name: "root", TemplateName: "root-tmpl", Type: "dag-task"},
		"child": {Name: "child", TemplateName: "child-tmpl", Type: "dag-task", Dependencies: []string{"root"}},
	}

	graph := renderManifestASCII(nodes)

	assert.Contains(t, graph, "root")
	assert.Contains(t, graph, "child")
}

func TestRenderManifestDOT(t *testing.T) {
	nodes := map[string]*manifestNode{
		"A": {Name: "A", TemplateName: "tmpl-a", Type: "dag-task"},
		"B": {Name: "B", TemplateName: "tmpl-b", Type: "dag-task", Dependencies: []string{"A"}},
	}

	graph := renderManifestDOT(nodes)

	assert.Contains(t, graph, "digraph workflow")
	assert.Contains(t, graph, "rankdir=TB")
	assert.Contains(t, graph, "->")
	assert.True(t, strings.HasSuffix(strings.TrimSpace(graph), "}"))
}

func TestFindManifestRootNodes(t *testing.T) {
	nodes := map[string]*manifestNode{
		"A": {Name: "A", Dependencies: []string{}},
		"B": {Name: "B", Dependencies: []string{"A"}},
		"C": {Name: "C", Dependencies: []string{}},
		"D": {Name: "D", Dependencies: []string{"B", "C"}},
	}

	roots := findManifestRootNodes(nodes)

	assert.Len(t, roots, 2)
	assert.Contains(t, roots, "A")
	assert.Contains(t, roots, "C")
}

func TestManifestTooLarge(t *testing.T) {
	// Create a manifest larger than 1 MiB
	largeManifest := strings.Repeat("x", 1<<20+1)

	handler := RenderManifestGraphHandler()
	_, _, err := handler(t.Context(), nil, RenderManifestGraphInput{
		Manifest: largeManifest,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "manifest too large")
}

// Test manifests

const dagWorkflowManifest = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: dag-example
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: build
            template: echo
          - name: test
            template: echo
            dependencies: [build]
          - name: deploy
            template: echo
            dependencies: [test]
    - name: echo
      container:
        image: alpine
        command: [echo, hello]
`

const stepsWorkflowManifest = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: steps-example
spec:
  entrypoint: main
  templates:
    - name: main
      steps:
        - - name: step1
            template: echo
        - - name: step2a
            template: echo
          - name: step2b
            template: echo
    - name: echo
      container:
        image: alpine
`

const workflowTemplateManifest = `
apiVersion: argoproj.io/v1alpha1
kind: WorkflowTemplate
metadata:
  name: my-template
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: task1
            template: echo
    - name: echo
      container:
        image: alpine
`

const clusterWorkflowTemplateManifest = `
apiVersion: argoproj.io/v1alpha1
kind: ClusterWorkflowTemplate
metadata:
  name: cluster-template
spec:
  entrypoint: main
  templates:
    - name: main
      container:
        image: alpine
`

const cronWorkflowManifest = `
apiVersion: argoproj.io/v1alpha1
kind: CronWorkflow
metadata:
  name: my-cron
spec:
  schedule: "0 * * * *"
  workflowSpec:
    entrypoint: main
    templates:
      - name: main
        container:
          image: alpine
`

const unsupportedKindManifest = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: test
data:
  key: value
`

const missingEntrypointManifest = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: missing-entrypoint
spec:
  entrypoint: nonexistent-template
  templates:
    - name: actual-template
      container:
        image: alpine
`

const conditionalWorkflowManifest = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: conditional-example
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: check
            template: echo
          - name: deploy
            template: echo
            dependencies: [check]
            when: "{{tasks.check.outputs.result}} == 'success'"
    - name: echo
      container:
        image: alpine
`

//nolint:gosec // G101 false positive - this is a YAML manifest, not credentials
const loopWorkflowManifest = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: loop-example
spec:
  entrypoint: main
  templates:
    - name: main
      dag:
        tasks:
          - name: process-items
            template: echo
            withItems:
              - item1
              - item2
    - name: echo
      container:
        image: alpine
`

const simpleContainerManifest = `
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: hello-world
spec:
  entrypoint: hello-world
  templates:
    - name: hello-world
      container:
        image: alpine
        command: [echo, hello]
`
