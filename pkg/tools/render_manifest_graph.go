// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"sigs.k8s.io/yaml"
)

// Kind constants for manifest types.
const (
	// KindWorkflow is the Workflow manifest kind.
	KindWorkflow = "Workflow"
	// KindWorkflowTemplate is the WorkflowTemplate manifest kind.
	KindWorkflowTemplate = "WorkflowTemplate"
	// KindClusterWorkflowTemplate is the ClusterWorkflowTemplate manifest kind.
	KindClusterWorkflowTemplate = "ClusterWorkflowTemplate"
	// KindCronWorkflow is the CronWorkflow manifest kind.
	KindCronWorkflow = "CronWorkflow"
)

// RenderManifestGraphInput defines the input parameters for the render_manifest_graph tool.
type RenderManifestGraphInput struct {
	// Manifest is the workflow YAML manifest to render.
	Manifest string `json:"manifest" jsonschema:"Workflow, WorkflowTemplate, ClusterWorkflowTemplate, or CronWorkflow YAML manifest,required"`

	// Format is the output format (mermaid, ascii, dot, or svg).
	Format string `json:"format,omitempty" jsonschema:"Output format: mermaid (default), ascii, dot, or svg,enum=mermaid,enum=ascii,enum=dot,enum=svg"`
}

// RenderManifestGraphOutput defines the output for the render_manifest_graph tool.
//
//nolint:govet // Field order optimized for readability over memory alignment
type RenderManifestGraphOutput struct {
	Graph     string `json:"graph"`
	Format    string `json:"format"`
	Kind      string `json:"kind"`
	Name      string `json:"name,omitempty"`
	NodeCount int    `json:"nodeCount"`
}

// RenderManifestGraphTool returns the MCP tool definition for render_manifest_graph.
func RenderManifestGraphTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "render_manifest_graph",
		Description: "Render an Argo Workflow manifest (YAML) as a graph showing the DAG structure and dependencies, without submitting it. Supports Workflow, WorkflowTemplate, ClusterWorkflowTemplate, and CronWorkflow manifests. Supports Mermaid, ASCII, DOT, and SVG formats.",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// RenderManifestGraphHandler returns a handler function for the render_manifest_graph tool.
// Note: This tool doesn't require the Argo client since it works purely from YAML.
func RenderManifestGraphHandler() func(context.Context, *mcp.CallToolRequest, RenderManifestGraphInput) (*mcp.CallToolResult, *RenderManifestGraphOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input RenderManifestGraphInput) (*mcp.CallToolResult, *RenderManifestGraphOutput, error) {
		// Validate manifest is provided
		if strings.TrimSpace(input.Manifest) == "" {
			return nil, nil, fmt.Errorf("manifest cannot be empty")
		}

		// Guard against oversized manifests (DoS hardening)
		const maxManifestBytes = 1 << 20 // 1 MiB
		if len(input.Manifest) > maxManifestBytes {
			return nil, nil, fmt.Errorf("manifest too large (%d bytes), max %d", len(input.Manifest), maxManifestBytes)
		}

		// Determine format
		format := strings.ToLower(strings.TrimSpace(input.Format))
		if format == "" {
			format = FormatMermaid
		}
		if format != FormatMermaid && format != FormatASCII && format != FormatDOT && format != FormatSVG {
			return nil, nil, fmt.Errorf("invalid format: %s (must be %s, %s, %s, or %s)", format, FormatMermaid, FormatASCII, FormatDOT, FormatSVG)
		}

		// Try to determine the kind of manifest
		spec, kind, name, err := extractWorkflowSpec(input.Manifest)
		if err != nil {
			return nil, nil, err
		}

		// Build the graph structure from the spec
		nodes, err := buildGraphFromSpec(spec)
		if err != nil {
			return nil, nil, err
		}

		// Render the graph
		var graph string
		var renderErr error
		switch format {
		case FormatMermaid:
			graph = renderManifestMermaid(nodes)
		case FormatASCII:
			graph = renderManifestASCII(nodes)
		case FormatDOT:
			graph = renderManifestDOT(nodes)
		case FormatSVG:
			dot := renderManifestDOT(nodes)
			graph, renderErr = dotToSVG(ctx, dot)
			if renderErr != nil {
				return nil, nil, fmt.Errorf("failed to render SVG: %w", renderErr)
			}
		}

		output := &RenderManifestGraphOutput{
			Graph:     graph,
			Format:    format,
			NodeCount: len(nodes),
			Kind:      kind,
			Name:      name,
		}

		// Build human-readable result
		resultText := fmt.Sprintf("Rendered %s %q as %s graph with %d nodes", kind, name, format, output.NodeCount)

		result := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: resultText},
			},
		}

		return result, output, nil
	}
}

// manifestNode represents a node in the manifest graph.
//
//nolint:govet // Field order optimized for readability over memory alignment
type manifestNode struct {
	Name         string
	TemplateName string
	Type         string
	When         string
	Dependencies []string
	WithItems    bool
	WithParam    bool
}

// extractWorkflowSpec extracts the WorkflowSpec from various manifest types.
func extractWorkflowSpec(manifest string) (*wfv1.WorkflowSpec, string, string, error) {
	// First, try to determine the kind by parsing as a generic object
	var generic struct {
		Kind     string `json:"kind"`
		Metadata struct {
			Name         string `json:"name"`
			GenerateName string `json:"generateName"`
		} `json:"metadata"`
	}
	if err := yaml.Unmarshal([]byte(manifest), &generic); err != nil {
		return nil, "", "", fmt.Errorf("failed to parse manifest: %w", err)
	}

	kind := generic.Kind
	name := generic.Metadata.Name
	if name == "" {
		name = generic.Metadata.GenerateName
	}

	switch kind {
	case KindWorkflow, "":
		var wf wfv1.Workflow
		if err := yaml.Unmarshal([]byte(manifest), &wf); err != nil {
			return nil, "", "", fmt.Errorf("failed to parse %s manifest: %w", KindWorkflow, err)
		}
		if name == "" {
			name = wf.Name
			if name == "" {
				name = wf.GenerateName
			}
		}
		if kind == "" {
			kind = KindWorkflow
		}
		return &wf.Spec, kind, name, nil

	case KindWorkflowTemplate:
		var wft wfv1.WorkflowTemplate
		if err := yaml.Unmarshal([]byte(manifest), &wft); err != nil {
			return nil, "", "", fmt.Errorf("failed to parse %s manifest: %w", KindWorkflowTemplate, err)
		}
		if name == "" {
			name = wft.Name
		}
		return &wft.Spec, kind, name, nil

	case KindClusterWorkflowTemplate:
		var cwft wfv1.ClusterWorkflowTemplate
		if err := yaml.Unmarshal([]byte(manifest), &cwft); err != nil {
			return nil, "", "", fmt.Errorf("failed to parse %s manifest: %w", KindClusterWorkflowTemplate, err)
		}
		if name == "" {
			name = cwft.Name
		}
		return &cwft.Spec, kind, name, nil

	case KindCronWorkflow:
		var cronWf wfv1.CronWorkflow
		if err := yaml.Unmarshal([]byte(manifest), &cronWf); err != nil {
			return nil, "", "", fmt.Errorf("failed to parse %s manifest: %w", KindCronWorkflow, err)
		}
		if name == "" {
			name = cronWf.Name
		}
		return &cronWf.Spec.WorkflowSpec, kind, name, nil

	default:
		return nil, "", "", fmt.Errorf("unsupported manifest kind: %s (must be %s, %s, %s, or %s)", kind, KindWorkflow, KindWorkflowTemplate, KindClusterWorkflowTemplate, KindCronWorkflow)
	}
}

// buildGraphFromSpec builds the graph structure from a WorkflowSpec.
// Returns an error if the entrypoint template is not found.
func buildGraphFromSpec(spec *wfv1.WorkflowSpec) (map[string]*manifestNode, error) {
	nodes := make(map[string]*manifestNode)

	// Build a map of templates for lookup
	templateMap := make(map[string]*wfv1.Template)
	for i := range spec.Templates {
		templateMap[spec.Templates[i].Name] = &spec.Templates[i]
	}

	// Find the entrypoint template
	entrypoint := spec.Entrypoint
	if entrypoint == "" && len(spec.Templates) > 0 {
		entrypoint = spec.Templates[0].Name
	}

	// Handle case where no templates exist
	if entrypoint == "" {
		return nodes, nil
	}

	entryTemplate := templateMap[entrypoint]
	if entryTemplate == nil {
		return nil, fmt.Errorf("entrypoint template %q not found in manifest", entrypoint)
	}

	// Process based on template type
	switch {
	case entryTemplate.DAG != nil:
		// DAG template - extract tasks
		for _, task := range entryTemplate.DAG.Tasks {
			node := &manifestNode{
				Name:         task.Name,
				TemplateName: task.Template,
				Type:         "dag-task",
				Dependencies: task.Dependencies,
				When:         task.When,
				WithItems:    len(task.WithItems) > 0,
				WithParam:    task.WithParam != "",
			}
			nodes[task.Name] = node
		}
	case len(entryTemplate.Steps) > 0:
		// Steps template - extract step groups
		var prevGroupNodes []string
		for groupIdx, stepGroup := range entryTemplate.Steps {
			var currentGroupNodes []string
			for _, step := range stepGroup.Steps {
				nodeName := fmt.Sprintf("step-%d-%s", groupIdx, step.Name)
				node := &manifestNode{
					Name:         step.Name,
					TemplateName: step.Template,
					Type:         "step",
					Dependencies: prevGroupNodes, // Each step depends on all steps in previous group
					When:         step.When,
					WithItems:    len(step.WithItems) > 0,
					WithParam:    step.WithParam != "",
				}
				nodes[nodeName] = node
				currentGroupNodes = append(currentGroupNodes, nodeName)
			}
			prevGroupNodes = currentGroupNodes
		}
	default:
		// Simple template (container, script, etc.)
		node := &manifestNode{
			Name:         entrypoint,
			TemplateName: entrypoint,
			Type:         getTemplateType(entryTemplate),
		}
		nodes[entrypoint] = node
	}

	return nodes, nil
}

// getTemplateType returns the type of a template.
func getTemplateType(tmpl *wfv1.Template) string {
	switch {
	case tmpl.Container != nil:
		return "container"
	case tmpl.Script != nil:
		return "script"
	case tmpl.Resource != nil:
		return "resource"
	case tmpl.Suspend != nil:
		return "suspend"
	case tmpl.HTTP != nil:
		return "http"
	case tmpl.Plugin != nil:
		return "plugin"
	case tmpl.DAG != nil:
		return "dag"
	case len(tmpl.Steps) > 0:
		return "steps"
	default:
		return "unknown"
	}
}

// renderManifestMermaid renders the manifest graph as Mermaid.
func renderManifestMermaid(nodes map[string]*manifestNode) string {
	if len(nodes) == 0 {
		return "flowchart TD\n    Start[No templates in workflow]"
	}

	var sb strings.Builder
	sb.WriteString("flowchart TD\n")

	// Sort node names for consistent output
	nodeNames := make([]string, 0, len(nodes))
	for name := range nodes {
		nodeNames = append(nodeNames, name)
	}
	sort.Strings(nodeNames)

	// Track edges to avoid duplicates
	edges := make(map[string]bool)

	// Process each node
	for _, nodeName := range nodeNames {
		node := nodes[nodeName]
		safeID := makeSafeID(nodeName)
		displayName := node.Name
		if node.TemplateName != "" && node.TemplateName != node.Name {
			displayName = fmt.Sprintf("%s\\n(%s)", node.Name, node.TemplateName)
		}

		// Add annotations for loops/conditionals
		annotations := ""
		if node.WithItems {
			annotations += " [loop]"
		}
		if node.WithParam {
			annotations += " [param-loop]"
		}
		if node.When != "" {
			annotations += " [conditional]"
		}
		if annotations != "" {
			displayName += annotations
		}

		// Add node definition
		fmt.Fprintf(&sb, "    %s[%s]\n", safeID, displayName)

		// Add edges from dependencies
		for _, dep := range node.Dependencies {
			safeDep := makeSafeID(dep)
			edge := fmt.Sprintf("%s -> %s", safeDep, safeID)
			if !edges[edge] {
				edges[edge] = true
				if node.When != "" {
					// Dashed line for conditional edges
					fmt.Fprintf(&sb, "    %s -.-> %s\n", safeDep, safeID)
				} else {
					fmt.Fprintf(&sb, "    %s --> %s\n", safeDep, safeID)
				}
			}
		}
	}

	// Add style classes
	sb.WriteString("\n")
	sb.WriteString("    classDef default fill:#e5e7eb,color:#374151,stroke:#9ca3af\n")

	return sb.String()
}

// renderManifestASCII renders the manifest graph as ASCII.
func renderManifestASCII(nodes map[string]*manifestNode) string {
	if len(nodes) == 0 {
		return "No templates in workflow"
	}

	var sb strings.Builder

	// Find root nodes (nodes with no dependencies)
	rootNodes := findManifestRootNodes(nodes)

	// Build child map for tree traversal
	childMap := make(map[string][]string)
	for name, node := range nodes {
		for _, dep := range node.Dependencies {
			childMap[dep] = append(childMap[dep], name)
		}
	}

	// Render each root node tree
	visited := make(map[string]bool)
	for i, rootName := range rootNodes {
		isLast := i == len(rootNodes)-1
		renderManifestASCIINode(rootName, nodes, childMap, &sb, "", isLast, visited)
	}

	return sb.String()
}

// renderManifestASCIINode renders a single node and its children in ASCII format.
func renderManifestASCIINode(nodeName string, nodes map[string]*manifestNode, childMap map[string][]string, sb *strings.Builder, prefix string, isLast bool, visited map[string]bool) {
	if visited[nodeName] {
		return
	}
	visited[nodeName] = true

	node := nodes[nodeName]
	if node == nil {
		return
	}

	// Determine the branch character
	branch := "├── "
	if isLast {
		branch = "└── "
	}

	// Build display string
	displayName := node.Name
	if node.TemplateName != "" && node.TemplateName != node.Name {
		displayName = fmt.Sprintf("%s (%s)", node.Name, node.TemplateName)
	}

	// Add annotations
	if node.WithItems || node.WithParam {
		displayName += " ↻"
	}
	if node.When != "" {
		displayName += " ?"
	}

	sb.WriteString(prefix + branch + displayName + "\n")

	// Get children
	children := childMap[nodeName]
	sort.Strings(children)

	// Render children
	for i, childName := range children {
		childIsLast := i == len(children)-1
		newPrefix := prefix
		if isLast {
			newPrefix += "    "
		} else {
			newPrefix += "│   "
		}
		renderManifestASCIINode(childName, nodes, childMap, sb, newPrefix, childIsLast, visited)
	}
}

// renderManifestDOT renders the manifest graph as DOT (Graphviz).
func renderManifestDOT(nodes map[string]*manifestNode) string {
	if len(nodes) == 0 {
		return "digraph workflow {\n    \"No templates in workflow\"\n}"
	}

	var sb strings.Builder
	sb.WriteString("digraph workflow {\n")
	sb.WriteString("    rankdir=TB;\n")
	sb.WriteString("    node [shape=box, style=filled, fillcolor=\"#e5e7eb\"];\n\n")

	// Sort node names for consistent output
	nodeNames := make([]string, 0, len(nodes))
	for name := range nodes {
		nodeNames = append(nodeNames, name)
	}
	sort.Strings(nodeNames)

	// Track edges to avoid duplicates
	edges := make(map[string]bool)

	// Process each node
	for _, nodeName := range nodeNames {
		node := nodes[nodeName]
		safeID := makeSafeID(nodeName)
		displayName := node.Name
		if node.TemplateName != "" && node.TemplateName != node.Name {
			displayName = fmt.Sprintf("%s\\n(%s)", node.Name, node.TemplateName)
		}

		// Add node definition
		fmt.Fprintf(&sb, "    \"%s\" [label=\"%s\"];\n", safeID, displayName)

		// Add edges from dependencies
		for _, dep := range node.Dependencies {
			safeDep := makeSafeID(dep)
			edge := fmt.Sprintf("%s -> %s", safeDep, safeID)
			if !edges[edge] {
				edges[edge] = true
				if node.When != "" {
					// Dashed line for conditional edges
					fmt.Fprintf(&sb, "    \"%s\" -> \"%s\" [style=dashed];\n", safeDep, safeID)
				} else {
					fmt.Fprintf(&sb, "    \"%s\" -> \"%s\";\n", safeDep, safeID)
				}
			}
		}
	}

	sb.WriteString("}\n")
	return sb.String()
}

// findManifestRootNodes finds nodes that have no dependencies.
func findManifestRootNodes(nodes map[string]*manifestNode) []string {
	var rootNodes []string
	for name, node := range nodes {
		if len(node.Dependencies) == 0 {
			rootNodes = append(rootNodes, name)
		}
	}
	sort.Strings(rootNodes)
	return rootNodes
}
