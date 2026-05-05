package tools

import (
	"strings"
	"testing"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestRenderWorkflowGraphTool(t *testing.T) {
	tool := RenderWorkflowGraphTool()

	assert.Equal(t, "render_workflow_graph", tool.Name)
	assert.NotEmpty(t, tool.Description)
}

func TestRenderWorkflowGraphHandler(t *testing.T) {
	tests := []struct {
		workflow      *wfv1.Workflow
		validate      func(t *testing.T, output *RenderWorkflowGraphOutput)
		input         RenderWorkflowGraphInput
		name          string
		expectedError string
	}{
		{
			name: "simple DAG workflow with mermaid format",
			input: RenderWorkflowGraphInput{
				Name:   "test-workflow",
				Format: "mermaid",
			},
			workflow: createDAGWorkflow("test-workflow", "default"),
			validate: func(t *testing.T, output *RenderWorkflowGraphOutput) {
				assert.Equal(t, "mermaid", output.Format)
				assert.Equal(t, 3, output.NodeCount)
				assert.Contains(t, output.Graph, "flowchart TD")
				assert.Contains(t, output.Graph, "build-image")
				assert.Contains(t, output.Graph, "run-tests")
				assert.Contains(t, output.Graph, "deploy")
				assert.Contains(t, output.Graph, "-->")
			},
		},
		{
			name: "workflow with ASCII format",
			input: RenderWorkflowGraphInput{
				Name:   "test-workflow",
				Format: "ascii",
			},
			workflow: createDAGWorkflow("test-workflow", "default"),
			validate: func(t *testing.T, output *RenderWorkflowGraphOutput) {
				assert.Equal(t, "ascii", output.Format)
				assert.Contains(t, output.Graph, "build-image")
				assert.Contains(t, output.Graph, "✓")
				assert.Contains(t, output.Graph, "├──")
			},
		},
		{
			name: "workflow with DOT format",
			input: RenderWorkflowGraphInput{
				Name:   "test-workflow",
				Format: "dot",
			},
			workflow: createDAGWorkflow("test-workflow", "default"),
			validate: func(t *testing.T, output *RenderWorkflowGraphOutput) {
				assert.Equal(t, "dot", output.Format)
				assert.Contains(t, output.Graph, "digraph workflow")
				assert.Contains(t, output.Graph, "build-image")
				assert.Contains(t, output.Graph, "->")
			},
		},
		{
			name: "workflow with SVG format",
			input: RenderWorkflowGraphInput{
				Name:   "test-workflow",
				Format: "svg",
			},
			workflow: createDAGWorkflow("test-workflow", "default"),
			validate: func(t *testing.T, output *RenderWorkflowGraphOutput) {
				assert.Equal(t, "svg", output.Format)
				assert.True(t, strings.HasPrefix(output.Graph, "<?xml") || strings.Contains(output.Graph, "<svg"))
				assert.Contains(t, output.Graph, "</svg>")
				// Note: SVG may HTML-encode dashes as &#45; so we check for "build" instead
				assert.Contains(t, output.Graph, "build")
			},
		},
		{
			name: "workflow without status colors",
			input: RenderWorkflowGraphInput{
				Name:          "test-workflow",
				Format:        "mermaid",
				IncludeStatus: boolPtr(false),
			},
			workflow: createDAGWorkflow("test-workflow", "default"),
			validate: func(t *testing.T, output *RenderWorkflowGraphOutput) {
				assert.Equal(t, "mermaid", output.Format)
				assert.NotContains(t, output.Graph, "classDef")
				assert.NotContains(t, output.Graph, ":::")
			},
		},
		{
			name: "empty workflow",
			input: RenderWorkflowGraphInput{
				Name:   "empty-workflow",
				Format: "mermaid",
			},
			workflow: createEmptyWorkflow("empty-workflow", "default"),
			validate: func(t *testing.T, output *RenderWorkflowGraphOutput) {
				assert.Equal(t, "mermaid", output.Format)
				assert.Equal(t, 0, output.NodeCount)
				assert.Contains(t, output.Graph, "No nodes in workflow")
			},
		},
		{
			name: "invalid format",
			input: RenderWorkflowGraphInput{
				Name:   "test-workflow",
				Format: "invalid",
			},
			workflow:      createDAGWorkflow("test-workflow", "default"),
			expectedError: "invalid format",
		},
		{
			name: "empty workflow name",
			input: RenderWorkflowGraphInput{
				Name:   "",
				Format: "mermaid",
			},
			expectedError: "workflow name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock client
			mockClient := newMockClient(t, "default", false)
			mockWfService := newMockWorkflowService(t)
			mockClient.SetWorkflowService(mockWfService)

			if tt.expectedError == "" && tt.workflow != nil {
				mockWfService.On("GetWorkflow", mock.Anything, mock.MatchedBy(func(req *workflow.WorkflowGetRequest) bool {
					return req.Name == tt.input.Name
				})).Return(tt.workflow, nil)
			}

			// Always verify mock expectations
			defer mockWfService.AssertExpectations(t)

			// Create handler and execute
			handler := RenderWorkflowGraphHandler(mockClient)
			result, output, err := handler(t.Context(), nil, tt.input)

			// Validate
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
				assert.Contains(t, textContent.Text, "Rendered workflow")
				require.NotNil(t, output)
				tt.validate(t, output)
			}
		})
	}
}

func TestRenderMermaidGraph(t *testing.T) {
	tests := []struct {
		workflow      *wfv1.Workflow
		validate      func(t *testing.T, graph string)
		name          string
		includeStatus bool
	}{
		{
			name:          "DAG workflow with status",
			workflow:      createDAGWorkflow("test", "default"),
			includeStatus: true,
			validate: func(t *testing.T, graph string) {
				assert.Contains(t, graph, "flowchart TD")
				assert.Contains(t, graph, ":::succeeded")
				assert.Contains(t, graph, ":::running")
				assert.Contains(t, graph, "classDef succeeded")
				assert.Contains(t, graph, "classDef running")
			},
		},
		{
			name:          "DAG workflow without status",
			workflow:      createDAGWorkflow("test", "default"),
			includeStatus: false,
			validate: func(t *testing.T, graph string) {
				assert.Contains(t, graph, "flowchart TD")
				assert.NotContains(t, graph, ":::")
				assert.NotContains(t, graph, "classDef")
			},
		},
		{
			name:          "empty workflow",
			workflow:      createEmptyWorkflow("test", "default"),
			includeStatus: true,
			validate: func(t *testing.T, graph string) {
				assert.Contains(t, graph, "No nodes in workflow")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := renderMermaidGraph(tt.workflow, tt.includeStatus)
			require.NotEmpty(t, graph)
			tt.validate(t, graph)
		})
	}
}

func TestRenderASCIIGraph(t *testing.T) {
	tests := []struct {
		workflow      *wfv1.Workflow
		validate      func(t *testing.T, graph string)
		name          string
		includeStatus bool
	}{
		{
			name:          "DAG workflow with status",
			workflow:      createDAGWorkflow("test", "default"),
			includeStatus: true,
			validate: func(t *testing.T, graph string) {
				assert.Contains(t, graph, "build-image")
				assert.Contains(t, graph, "✓") // succeeded symbol
				assert.Contains(t, graph, "◉") // running symbol
				lines := strings.Split(graph, "\n")
				assert.GreaterOrEqual(t, len(lines), 2)
			},
		},
		{
			name:          "DAG workflow without status",
			workflow:      createDAGWorkflow("test", "default"),
			includeStatus: false,
			validate: func(t *testing.T, graph string) {
				assert.Contains(t, graph, "build-image")
				assert.NotContains(t, graph, "✓")
			},
		},
		{
			name:          "empty workflow",
			workflow:      createEmptyWorkflow("test", "default"),
			includeStatus: true,
			validate: func(t *testing.T, graph string) {
				assert.Equal(t, "No nodes in workflow", graph)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := renderASCIIGraph(tt.workflow, tt.includeStatus)
			require.NotEmpty(t, graph)
			tt.validate(t, graph)
		})
	}
}

func TestRenderDOTGraph(t *testing.T) {
	tests := []struct {
		workflow      *wfv1.Workflow
		validate      func(t *testing.T, graph string)
		name          string
		includeStatus bool
	}{
		{
			name:          "DAG workflow with status",
			workflow:      createDAGWorkflow("test", "default"),
			includeStatus: true,
			validate: func(t *testing.T, graph string) {
				assert.Contains(t, graph, "digraph workflow")
				assert.Contains(t, graph, "fillcolor")
				assert.Contains(t, graph, "#22c55e") // green for succeeded
			},
		},
		{
			name:          "DAG workflow without status",
			workflow:      createDAGWorkflow("test", "default"),
			includeStatus: false,
			validate: func(t *testing.T, graph string) {
				assert.Contains(t, graph, "digraph workflow")
				assert.NotContains(t, graph, "fillcolor")
			},
		},
		{
			name:          "empty workflow",
			workflow:      createEmptyWorkflow("test", "default"),
			includeStatus: true,
			validate: func(t *testing.T, graph string) {
				assert.Contains(t, graph, "No nodes in workflow")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := renderDOTGraph(tt.workflow, tt.includeStatus)
			require.NotEmpty(t, graph)
			tt.validate(t, graph)
		})
	}
}

func TestIsRenderableNode(t *testing.T) {
	tests := []struct {
		node     *wfv1.NodeStatus
		name     string
		expected bool
	}{
		{
			name:     "nil node",
			node:     nil,
			expected: false,
		},
		{
			name: "pod node",
			node: &wfv1.NodeStatus{
				Type: wfv1.NodeTypePod,
			},
			expected: true,
		},
		{
			name: "DAG node",
			node: &wfv1.NodeStatus{
				Type: wfv1.NodeTypeDAG,
			},
			expected: true,
		},
		{
			name: "retry node (virtual)",
			node: &wfv1.NodeStatus{
				Type: wfv1.NodeTypeRetry,
			},
			expected: false,
		},
		{
			name: "step group node (virtual)",
			node: &wfv1.NodeStatus{
				Type: wfv1.NodeTypeStepGroup,
			},
			expected: false,
		},
		{
			name: "task group node (virtual)",
			node: &wfv1.NodeStatus{
				Type: wfv1.NodeTypeTaskGroup,
			},
			expected: false,
		},
		{
			name: "container node",
			node: &wfv1.NodeStatus{
				Type: wfv1.NodeTypeContainer,
			},
			expected: true,
		},
		{
			name: "steps node",
			node: &wfv1.NodeStatus{
				Type: wfv1.NodeTypeSteps,
			},
			expected: true,
		},
		{
			name: "suspend node",
			node: &wfv1.NodeStatus{
				Type: wfv1.NodeTypeSuspend,
			},
			expected: true,
		},
		{
			name: "HTTP node",
			node: &wfv1.NodeStatus{
				Type: wfv1.NodeTypeHTTP,
			},
			expected: true,
		},
		{
			name: "plugin node",
			node: &wfv1.NodeStatus{
				Type: wfv1.NodeTypePlugin,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRenderableNode(tt.node)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetStatusSymbol(t *testing.T) {
	tests := []struct {
		name     string
		phase    wfv1.NodePhase
		expected string
	}{
		{"succeeded", wfv1.NodeSucceeded, "✓"},
		{"failed", wfv1.NodeFailed, "✗"},
		{"running", wfv1.NodeRunning, "◉"},
		{"pending", wfv1.NodePending, "○"},
		{"error", wfv1.NodeError, "⚠"},
		{"skipped", wfv1.NodeSkipped, "⊘"},
		{"omitted", wfv1.NodeOmitted, "⊗"},
		{"unknown", wfv1.NodePhase("unknown"), "○"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStatusSymbol(tt.phase)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetNodeClassName(t *testing.T) {
	tests := []struct {
		name     string
		phase    wfv1.NodePhase
		expected string
	}{
		{"succeeded", wfv1.NodeSucceeded, "succeeded"},
		{"failed", wfv1.NodeFailed, "failed"},
		{"running", wfv1.NodeRunning, "running"},
		{"pending", wfv1.NodePending, "pending"},
		{"error", wfv1.NodeError, "error"},
		{"skipped", wfv1.NodeSkipped, "skipped"},
		{"omitted", wfv1.NodeOmitted, "omitted"},
		{"unknown", wfv1.NodePhase("unknown"), "pending"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNodeClassName(tt.phase)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetNodeColor(t *testing.T) {
	tests := []struct {
		name     string
		phase    wfv1.NodePhase
		expected string
	}{
		{"succeeded", wfv1.NodeSucceeded, "#22c55e"},
		{"failed", wfv1.NodeFailed, "#ef4444"},
		{"running", wfv1.NodeRunning, "#3b82f6"},
		{"pending", wfv1.NodePending, "#9ca3af"},
		{"error", wfv1.NodeError, "#dc2626"},
		{"skipped", wfv1.NodeSkipped, "#d1d5db"},
		{"omitted", wfv1.NodeOmitted, "#e5e7eb"},
		{"unknown", wfv1.NodePhase("unknown"), "#9ca3af"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNodeColor(tt.phase)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMakeSafeID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		expected string
	}{
		{"simple", "node1", "node1"},
		{"with dots", "node.1.test", "node_1_test"},
		{"with dashes", "node-1-test", "node_1_test"},
		{"with parentheses", "node(1)", "node_1_"},
		{"with spaces", "node 1 test", "node_1_test"},
		{"with colons", "node:1:test", "node_1_test"},
		{"with brackets", "node[1]", "node_1_"},
		{"complex", "node.1(test)-2", "node_1_test__2"},
		{"all special chars", "node.1-2(3) 4:5[6]", "node_1_2_3__4_5_6_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := makeSafeID(tt.id)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetNodeDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		node     *wfv1.NodeStatus
		expected string
	}{
		{
			name: "with display name",
			node: &wfv1.NodeStatus{
				DisplayName:  "Build Image",
				TemplateName: "build",
				Name:         "test-workflow-123",
			},
			expected: "Build Image",
		},
		{
			name: "with template name only",
			node: &wfv1.NodeStatus{
				TemplateName: "build",
				Name:         "test-workflow-123",
			},
			expected: "build",
		},
		{
			name: "with name only",
			node: &wfv1.NodeStatus{
				Name: "test-workflow-123",
			},
			expected: "test-workflow-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getNodeDisplayName(tt.node)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions for creating test workflows

func createDAGWorkflow(name, namespace string) *wfv1.Workflow {
	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       types.UID("test-uid"),
		},
		Status: wfv1.WorkflowStatus{
			Phase: wfv1.WorkflowRunning,
			Nodes: wfv1.Nodes{
				"node-1": wfv1.NodeStatus{
					ID:           "node-1",
					Name:         name + ".build-image",
					DisplayName:  "build-image",
					Type:         wfv1.NodeTypePod,
					Phase:        wfv1.NodeSucceeded,
					TemplateName: "build",
					Children:     []string{"node-2", "node-3"},
				},
				"node-2": wfv1.NodeStatus{
					ID:           "node-2",
					Name:         name + ".run-tests",
					DisplayName:  "run-tests",
					Type:         wfv1.NodeTypePod,
					Phase:        wfv1.NodeSucceeded,
					TemplateName: "test",
					Children:     []string{},
				},
				"node-3": wfv1.NodeStatus{
					ID:           "node-3",
					Name:         name + ".deploy",
					DisplayName:  "deploy",
					Type:         wfv1.NodeTypePod,
					Phase:        wfv1.NodeRunning,
					TemplateName: "deploy",
					Children:     []string{},
				},
			},
		},
	}
}

func createEmptyWorkflow(name, namespace string) *wfv1.Workflow {
	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			UID:       types.UID("test-uid"),
		},
		Status: wfv1.WorkflowStatus{
			Phase: wfv1.WorkflowPending,
			Nodes: wfv1.Nodes{},
		},
	}
}

func boolPtr(b bool) *bool {
	return &b
}
