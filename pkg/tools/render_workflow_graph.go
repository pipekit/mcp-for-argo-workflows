// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

const (
	// FormatMermaid is the mermaid output format.
	FormatMermaid = "mermaid"
	// FormatASCII is the ASCII output format.
	FormatASCII = "ascii"
	// FormatDOT is the DOT (Graphviz) output format.
	FormatDOT = "dot"
	// FormatSVG is the SVG output format (rendered via Graphviz).
	FormatSVG = "svg"
)

// RenderWorkflowGraphInput defines the input parameters for the render_workflow_graph tool.
type RenderWorkflowGraphInput struct {
	// IncludeStatus indicates whether to include node execution status with colors.
	IncludeStatus *bool `json:"includeStatus,omitempty" jsonschema:"Include node execution status with colours (default: true)"`

	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// Name is the workflow name.
	Name string `json:"name" jsonschema:"Workflow name,required"`

	// Format is the output format (mermaid, ascii, dot, or svg).
	Format string `json:"format,omitempty" jsonschema:"Output format: mermaid (default), ascii, dot, or svg,enum=mermaid,enum=ascii,enum=dot,enum=svg"`
}

// RenderWorkflowGraphOutput defines the output for the render_workflow_graph tool.
type RenderWorkflowGraphOutput struct {
	// Graph is the rendered workflow graph.
	Graph string `json:"graph"`

	// Format is the format used for rendering.
	Format string `json:"format"`

	// NodeCount is the number of nodes in the graph.
	NodeCount int `json:"nodeCount"`
}

// RenderWorkflowGraphTool returns the MCP tool definition for render_workflow_graph.
func RenderWorkflowGraphTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "render_workflow_graph",
		Description: "Render an Argo Workflow as a graph showing the DAG structure, step dependencies, and node statuses. Supports Mermaid, ASCII, DOT, and SVG formats.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// RenderWorkflowGraphHandler returns a handler function for the render_workflow_graph tool.
func RenderWorkflowGraphHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, RenderWorkflowGraphInput) (*mcp.CallToolResult, *RenderWorkflowGraphOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input RenderWorkflowGraphInput) (*mcp.CallToolResult, *RenderWorkflowGraphOutput, error) {
		// Validate name is provided
		if strings.TrimSpace(input.Name) == "" {
			return nil, nil, fmt.Errorf("workflow name cannot be empty")
		}

		// Determine namespace
		namespace := ResolveNamespace(input.Namespace, client)

		// Determine format
		format := strings.ToLower(strings.TrimSpace(input.Format))
		if format == "" {
			format = FormatMermaid
		}
		if format != FormatMermaid && format != FormatASCII && format != FormatDOT && format != FormatSVG {
			return nil, nil, fmt.Errorf("invalid format: %s (must be %s, %s, %s, or %s)", format, FormatMermaid, FormatASCII, FormatDOT, FormatSVG)
		}

		// Determine includeStatus
		includeStatus := true
		if input.IncludeStatus != nil {
			includeStatus = *input.IncludeStatus
		}

		// Get the workflow service client
		wfService := client.WorkflowService()

		// Get the workflow
		wf, err := wfService.GetWorkflow(ctx, &workflow.WorkflowGetRequest{
			Namespace: namespace,
			Name:      input.Name,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get workflow: %w", err)
		}

		// Render the graph
		var graph string
		var renderErr error
		switch format {
		case FormatMermaid:
			graph = renderMermaidGraph(wf, includeStatus)
		case FormatASCII:
			graph = renderASCIIGraph(wf, includeStatus)
		case FormatDOT:
			graph = renderDOTGraph(wf, includeStatus)
		case FormatSVG:
			dot := renderDOTGraph(wf, includeStatus)
			graph, renderErr = dotToSVG(ctx, dot)
			if renderErr != nil {
				return nil, nil, fmt.Errorf("failed to render SVG: %w", renderErr)
			}
		}

		output := &RenderWorkflowGraphOutput{
			Graph:     graph,
			Format:    format,
			NodeCount: len(wf.Status.Nodes),
		}

		// Build human-readable result
		resultText := fmt.Sprintf("Rendered workflow %q as %s graph with %d nodes", input.Name, format, output.NodeCount)

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}

		return result, output, nil
	}
}

// renderMermaidGraph renders a workflow as a Mermaid flowchart.
func renderMermaidGraph(wf *wfv1.Workflow, includeStatus bool) string {
	if len(wf.Status.Nodes) == 0 {
		return "flowchart TD\n    Start[No nodes in workflow]"
	}

	var sb strings.Builder
	sb.WriteString("flowchart TD\n")

	// Build a map of nodes for easy lookup
	nodeMap := make(map[string]*wfv1.NodeStatus)
	for id, node := range wf.Status.Nodes {
		nodeCopy := node
		nodeMap[id] = &nodeCopy
	}

	// Find renderable nodes (excluding virtual/container nodes)
	renderableNodes := getRenderableNodes(wf.Status.Nodes)

	// Track edges to avoid duplicates
	edges := make(map[string]bool)

	// Process each renderable node
	for _, nodeID := range renderableNodes {
		node := nodeMap[nodeID]
		displayName := getNodeDisplayName(node)
		safeID := makeSafeID(nodeID)

		// Add node definition
		if includeStatus {
			className := getNodeClassName(node.Phase)
			fmt.Fprintf(&sb, "    %s[%s]:::%s\n", safeID, displayName, className)
		} else {
			fmt.Fprintf(&sb, "    %s[%s]\n", safeID, displayName)
		}

		// Add edges to children
		for _, childID := range node.Children {
			if isRenderableNode(nodeMap[childID]) {
				safeChildID := makeSafeID(childID)
				edge := fmt.Sprintf("%s -> %s", safeID, safeChildID)
				if !edges[edge] {
					edges[edge] = true
					fmt.Fprintf(&sb, "    %s --> %s\n", safeID, safeChildID)
				}
			}
		}
	}

	// Add class definitions if includeStatus is enabled
	if includeStatus {
		sb.WriteString("\n")
		sb.WriteString("    classDef succeeded fill:#22c55e,color:#fff,stroke:#16a34a\n")
		sb.WriteString("    classDef failed fill:#ef4444,color:#fff,stroke:#dc2626\n")
		sb.WriteString("    classDef running fill:#3b82f6,color:#fff,stroke:#2563eb\n")
		sb.WriteString("    classDef pending fill:#9ca3af,color:#fff,stroke:#6b7280\n")
		sb.WriteString("    classDef error fill:#dc2626,color:#fff,stroke:#b91c1c\n")
		sb.WriteString("    classDef skipped fill:#d1d5db,color:#374151,stroke:#9ca3af\n")
		sb.WriteString("    classDef omitted fill:#e5e7eb,color:#6b7280,stroke:#d1d5db\n")
	}

	return sb.String()
}

// renderASCIIGraph renders a workflow as an ASCII tree.
func renderASCIIGraph(wf *wfv1.Workflow, includeStatus bool) string {
	if len(wf.Status.Nodes) == 0 {
		return "No nodes in workflow"
	}

	var sb strings.Builder

	// Build a map of nodes for easy lookup
	nodeMap := make(map[string]*wfv1.NodeStatus)
	for id, node := range wf.Status.Nodes {
		nodeCopy := node
		nodeMap[id] = &nodeCopy
	}

	// Find root nodes (nodes with no parents among renderable nodes)
	rootNodes := findRootNodes(wf.Status.Nodes, nodeMap)

	// Render each root node tree
	visited := make(map[string]bool)
	for i, rootID := range rootNodes {
		isLast := i == len(rootNodes)-1
		renderASCIINode(nodeMap[rootID], nodeMap, &sb, "", isLast, includeStatus, visited)
	}

	return sb.String()
}

// renderASCIINode renders a single node and its children in ASCII format.
func renderASCIINode(node *wfv1.NodeStatus, nodeMap map[string]*wfv1.NodeStatus, sb *strings.Builder, prefix string, isLast bool, includeStatus bool, visited map[string]bool) {
	if node == nil || visited[node.ID] {
		return
	}
	visited[node.ID] = true

	// Determine the branch character
	branch := "├── "
	if isLast {
		branch = "└── "
	}

	// Get status indicator
	statusIndicator := ""
	if includeStatus {
		statusIndicator = getStatusSymbol(node.Phase) + " "
	}

	// Write the node
	displayName := getNodeDisplayName(node)
	sb.WriteString(prefix + branch + displayName + " " + statusIndicator + "\n")

	// Get renderable children
	var renderableChildren []string
	for _, childID := range node.Children {
		if childNode := nodeMap[childID]; childNode != nil && isRenderableNode(childNode) {
			renderableChildren = append(renderableChildren, childID)
		}
	}

	// Render children
	for i, childID := range renderableChildren {
		childIsLast := i == len(renderableChildren)-1
		newPrefix := prefix
		if isLast {
			newPrefix += "    "
		} else {
			newPrefix += "│   "
		}
		renderASCIINode(nodeMap[childID], nodeMap, sb, newPrefix, childIsLast, includeStatus, visited)
	}
}

// renderDOTGraph renders a workflow as a DOT (Graphviz) graph.
func renderDOTGraph(wf *wfv1.Workflow, includeStatus bool) string {
	if len(wf.Status.Nodes) == 0 {
		return "digraph workflow {\n    \"No nodes in workflow\"\n}"
	}

	var sb strings.Builder
	sb.WriteString("digraph workflow {\n")
	sb.WriteString("    rankdir=TB;\n")
	sb.WriteString("    node [shape=box];\n\n")

	// Build a map of nodes for easy lookup
	nodeMap := make(map[string]*wfv1.NodeStatus)
	for id, node := range wf.Status.Nodes {
		nodeCopy := node
		nodeMap[id] = &nodeCopy
	}

	// Find renderable nodes
	renderableNodes := getRenderableNodes(wf.Status.Nodes)

	// Track edges to avoid duplicates
	edges := make(map[string]bool)

	// Process each renderable node
	for _, nodeID := range renderableNodes {
		node := nodeMap[nodeID]
		displayName := getNodeDisplayName(node)
		safeID := makeSafeID(nodeID)

		// Add node definition with styling
		if includeStatus {
			color := getNodeColor(node.Phase)
			fmt.Fprintf(&sb, "    \"%s\" [label=\"%s\", fillcolor=\"%s\", style=filled];\n",
				safeID, displayName, color)
		} else {
			fmt.Fprintf(&sb, "    \"%s\" [label=\"%s\"];\n", safeID, displayName)
		}

		// Add edges to children
		for _, childID := range node.Children {
			if isRenderableNode(nodeMap[childID]) {
				safeChildID := makeSafeID(childID)
				edge := fmt.Sprintf("%s -> %s", safeID, safeChildID)
				if !edges[edge] {
					edges[edge] = true
					fmt.Fprintf(&sb, "    \"%s\" -> \"%s\";\n", safeID, safeChildID)
				}
			}
		}
	}

	sb.WriteString("}\n")
	return sb.String()
}

// getRenderableNodes returns a sorted list of renderable node IDs.
func getRenderableNodes(nodes wfv1.Nodes) []string {
	var renderableNodes []string
	for id, node := range nodes {
		if isRenderableNode(&node) {
			renderableNodes = append(renderableNodes, id)
		}
	}
	sort.Strings(renderableNodes)
	return renderableNodes
}

// isRenderableNode checks if a node should be rendered in the graph.
// Excludes virtual nodes like Retry, StepGroup, and TaskGroup.
func isRenderableNode(node *wfv1.NodeStatus) bool {
	if node == nil {
		return false
	}
	// Exclude virtual node types
	switch node.Type {
	case wfv1.NodeTypeRetry, wfv1.NodeTypeStepGroup, wfv1.NodeTypeTaskGroup:
		return false
	case wfv1.NodeTypePod, wfv1.NodeTypeContainer, wfv1.NodeTypeSteps, wfv1.NodeTypeDAG,
		wfv1.NodeTypeSkipped, wfv1.NodeTypeSuspend, wfv1.NodeTypeHTTP, wfv1.NodeTypePlugin:
		return true
	default:
		// For any unknown node types, render them by default
		return true
	}
}

// findRootNodes finds nodes that have no parents among renderable nodes.
func findRootNodes(nodes wfv1.Nodes, nodeMap map[string]*wfv1.NodeStatus) []string {
	// Build a set of all children
	childSet := make(map[string]bool)
	for _, node := range nodes {
		if isRenderableNode(&node) {
			for _, childID := range node.Children {
				if childNode := nodeMap[childID]; childNode != nil && isRenderableNode(childNode) {
					childSet[childID] = true
				}
			}
		}
	}

	// Find nodes that are not in the child set (i.e., root nodes)
	var rootNodes []string
	for id, node := range nodes {
		if isRenderableNode(&node) && !childSet[id] {
			rootNodes = append(rootNodes, id)
		}
	}

	sort.Strings(rootNodes)
	return rootNodes
}

// getNodeDisplayName returns the display name for a node.
func getNodeDisplayName(node *wfv1.NodeStatus) string {
	if node.DisplayName != "" {
		return node.DisplayName
	}
	// Fallback to the template name or node name
	if node.TemplateName != "" {
		return node.TemplateName
	}
	return node.Name
}

// makeSafeID creates a safe identifier for Mermaid and DOT formats.
func makeSafeID(id string) string {
	// Replace characters that might cause issues
	safe := strings.ReplaceAll(id, ".", "_")
	safe = strings.ReplaceAll(safe, "-", "_")
	safe = strings.ReplaceAll(safe, "(", "_")
	safe = strings.ReplaceAll(safe, ")", "_")
	safe = strings.ReplaceAll(safe, " ", "_")
	safe = strings.ReplaceAll(safe, ":", "_")
	safe = strings.ReplaceAll(safe, "[", "_")
	safe = strings.ReplaceAll(safe, "]", "_")
	return safe
}

// getNodeClassName returns the CSS class name for a node based on its phase.
func getNodeClassName(phase wfv1.NodePhase) string {
	switch phase {
	case wfv1.NodeSucceeded:
		return "succeeded"
	case wfv1.NodeFailed:
		return "failed"
	case wfv1.NodeRunning:
		return "running"
	case wfv1.NodePending:
		return "pending"
	case wfv1.NodeError:
		return "error"
	case wfv1.NodeSkipped:
		return "skipped"
	case wfv1.NodeOmitted:
		return "omitted"
	default:
		return "pending"
	}
}

// getStatusSymbol returns an ASCII symbol for a node's phase.
func getStatusSymbol(phase wfv1.NodePhase) string {
	switch phase {
	case wfv1.NodeSucceeded:
		return "✓"
	case wfv1.NodeFailed:
		return "✗"
	case wfv1.NodeRunning:
		return "◉"
	case wfv1.NodePending:
		return "○"
	case wfv1.NodeError:
		return "⚠"
	case wfv1.NodeSkipped:
		return "⊘"
	case wfv1.NodeOmitted:
		return "⊗"
	default:
		return "○"
	}
}

// getNodeColor returns a color for a node based on its phase (for DOT format).
func getNodeColor(phase wfv1.NodePhase) string {
	switch phase {
	case wfv1.NodeSucceeded:
		return "#22c55e"
	case wfv1.NodeFailed:
		return "#ef4444"
	case wfv1.NodeRunning:
		return "#3b82f6"
	case wfv1.NodePending:
		return "#9ca3af"
	case wfv1.NodeError:
		return "#dc2626"
	case wfv1.NodeSkipped:
		return "#d1d5db"
	case wfv1.NodeOmitted:
		return "#e5e7eb"
	default:
		return "#9ca3af"
	}
}
