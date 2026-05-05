// Package prompts implements MCP prompt handlers for Argo Workflows operations.
package prompts

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	corev1 "k8s.io/api/core/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

const (
	// defaultLogTailLines is the default number of log lines to include per failed node.
	defaultLogTailLines = 50

	// maxLogBytes is the maximum total bytes of logs to include in the prompt.
	maxLogBytes = 50000
)

// WhyDidThisFailPrompt returns the MCP prompt definition for why_did_this_fail.
func WhyDidThisFailPrompt() *mcp.Prompt {
	return &mcp.Prompt{
		Name:        "why_did_this_fail",
		Title:       "Diagnose Workflow Failure",
		Description: "Diagnose why an Argo Workflow failed by analysing node statuses, logs, inputs, and data flow",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "workflow",
				Title:       "Workflow Name",
				Description: "Workflow name to diagnose",
				Required:    true,
			},
			{
				Name:        "namespace",
				Title:       "Namespace",
				Description: "Kubernetes namespace (uses default if not specified)",
				Required:    false,
			},
		},
	}
}

// WhyDidThisFailHandler returns a handler function for the why_did_this_fail prompt.
func WhyDidThisFailHandler(client argo.ClientInterface) mcp.PromptHandler {
	return func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		// Extract arguments
		workflowName := req.Params.Arguments["workflow"]
		if workflowName == "" {
			return nil, fmt.Errorf("workflow name is required")
		}

		namespace := req.Params.Arguments["namespace"]
		if namespace == "" {
			namespace = client.DefaultNamespace()
		}

		// Gather diagnostic information
		diagnosis, err := gatherDiagnostics(ctx, client, namespace, workflowName)
		if err != nil {
			return nil, fmt.Errorf("failed to gather diagnostics: %w", err)
		}

		// Build the prompt message
		promptText := buildPromptText(diagnosis)

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Diagnosis for failed workflow %s/%s", namespace, workflowName),
			Messages: []*mcp.PromptMessage{
				{
					Role:    "user",
					Content: &mcp.TextContent{Text: promptText},
				},
			},
		}, nil
	}
}

// diagnosis holds all gathered diagnostic information.
//
//nolint:govet // Field order optimized for readability over memory alignment
type diagnosis struct {
	WorkflowName string
	Namespace    string
	Phase        string
	Message      string
	StartedAt    time.Time
	FinishedAt   time.Time
	Duration     time.Duration
	Parameters   []parameterInfo
	FailedNodes  []failedNodeInfo
	RootCauses   []failedNodeInfo
}

// parameterInfo represents a workflow parameter.
type parameterInfo struct {
	Name  string
	Value string
}

// failedNodeInfo holds information about a failed node.
//
//nolint:govet // Field order optimized for readability over memory alignment
type failedNodeInfo struct {
	ID           string
	Name         string
	DisplayName  string
	TemplateName string
	Phase        string
	Message      string
	StartedAt    time.Time
	FinishedAt   time.Time
	ExitCode     string
	Inputs       []inputInfo
	Outputs      []outputInfo
	Logs         string
	Children     []string
	IsRootCause  bool
	ErrorPattern *errorPattern
}

// inputInfo represents a node input.
type inputInfo struct {
	Name   string
	Value  string
	Source string // Where this input came from
	Type   string // "parameter" or "artifact"
}

// outputInfo represents a node output.
type outputInfo struct {
	Name  string
	Value string
	Type  string // "parameter" or "artifact"
}

// errorPattern represents a recognized error pattern.
type errorPattern struct {
	Pattern    string
	Suggestion string
}

// gatherDiagnostics collects all relevant information for diagnosing a workflow failure.
func gatherDiagnostics(ctx context.Context, client argo.ClientInterface, namespace, workflowName string) (*diagnosis, error) {
	wfService := client.WorkflowService()

	// Get the workflow
	wf, err := wfService.GetWorkflow(ctx, &workflow.WorkflowGetRequest{
		Namespace: namespace,
		Name:      workflowName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}

	d := &diagnosis{
		WorkflowName: wf.Name,
		Namespace:    wf.Namespace,
		Phase:        string(wf.Status.Phase),
		Message:      wf.Status.Message,
	}

	// Set timing information
	if !wf.Status.StartedAt.Time.IsZero() {
		d.StartedAt = wf.Status.StartedAt.Time
	}
	if !wf.Status.FinishedAt.Time.IsZero() {
		d.FinishedAt = wf.Status.FinishedAt.Time
	}

	// Calculate duration
	if !d.StartedAt.IsZero() {
		endTime := d.FinishedAt
		if endTime.IsZero() {
			endTime = time.Now()
		}
		d.Duration = endTime.Sub(d.StartedAt)
	}

	// Extract parameters
	for _, p := range wf.Spec.Arguments.Parameters {
		param := parameterInfo{Name: p.Name}
		if p.Value != nil {
			param.Value = string(*p.Value)
		}
		d.Parameters = append(d.Parameters, param)
	}

	// Find failed nodes
	d.FailedNodes = findFailedNodes(wf)

	// Identify root causes (nodes that failed without upstream failures)
	d.RootCauses = findRootCauses(wf, d.FailedNodes)

	// Gather logs for root cause nodes
	totalLogBytes := 0
	for i := range d.RootCauses {
		if totalLogBytes >= maxLogBytes {
			break
		}
		logs, err := getNodeLogs(ctx, client, namespace, workflowName, d.RootCauses[i].Name)
		if err == nil && logs != "" {
			// Truncate if needed (using rune-safe truncation)
			remaining := maxLogBytes - totalLogBytes
			if len(logs) > remaining {
				logs = truncateStringBytes(logs, remaining) + "\n... (logs truncated)"
			}
			d.RootCauses[i].Logs = logs
			totalLogBytes += len(logs)
		}
	}

	// Detect error patterns
	for i := range d.RootCauses {
		d.RootCauses[i].ErrorPattern = detectErrorPattern(&d.RootCauses[i])
	}

	return d, nil
}

// findFailedNodes extracts all failed or error nodes from a workflow.
func findFailedNodes(wf *wfv1.Workflow) []failedNodeInfo {
	failed := make([]failedNodeInfo, 0, len(wf.Status.Nodes))

	for _, node := range wf.Status.Nodes {
		if node.Phase != wfv1.NodeFailed && node.Phase != wfv1.NodeError {
			continue
		}

		info := failedNodeInfo{
			ID:           node.ID,
			Name:         node.Name,
			DisplayName:  node.DisplayName,
			TemplateName: node.TemplateName,
			Phase:        string(node.Phase),
			Message:      node.Message,
			Children:     node.Children,
		}

		if !node.StartedAt.Time.IsZero() {
			info.StartedAt = node.StartedAt.Time
		}
		if !node.FinishedAt.Time.IsZero() {
			info.FinishedAt = node.FinishedAt.Time
		}

		// Extract exit code from outputs
		if node.Outputs != nil && node.Outputs.ExitCode != nil {
			info.ExitCode = *node.Outputs.ExitCode
		}

		// Extract inputs
		if node.Inputs != nil {
			for _, p := range node.Inputs.Parameters {
				input := inputInfo{
					Name: p.Name,
					Type: "parameter",
				}
				if p.Value != nil {
					input.Value = string(*p.Value)
				}
				// Try to identify source from value reference
				if p.ValueFrom != nil {
					input.Source = describeValueFrom(p.ValueFrom)
				}
				info.Inputs = append(info.Inputs, input)
			}
			for _, a := range node.Inputs.Artifacts {
				input := inputInfo{
					Name:   a.Name,
					Type:   "artifact",
					Source: describeArtifactSource(&a),
				}
				info.Inputs = append(info.Inputs, input)
			}
		}

		// Extract outputs
		if node.Outputs != nil {
			for _, p := range node.Outputs.Parameters {
				output := outputInfo{
					Name: p.Name,
					Type: "parameter",
				}
				if p.Value != nil {
					output.Value = string(*p.Value)
				}
				info.Outputs = append(info.Outputs, output)
			}
			for _, a := range node.Outputs.Artifacts {
				output := outputInfo{
					Name: a.Name,
					Type: "artifact",
				}
				info.Outputs = append(info.Outputs, output)
			}
		}

		failed = append(failed, info)
	}

	// Sort by start time to show failures in order
	sort.Slice(failed, func(i, j int) bool {
		return failed[i].StartedAt.Before(failed[j].StartedAt)
	})

	return failed
}

// findRootCauses identifies nodes that are true root causes (first failures in their chain).
func findRootCauses(wf *wfv1.Workflow, failedNodes []failedNodeInfo) []failedNodeInfo {
	// Build a map of failed node IDs
	failedIDs := make(map[string]bool)
	for _, node := range failedNodes {
		failedIDs[node.ID] = true
	}

	var rootCauses []failedNodeInfo

	for _, node := range failedNodes {
		// A node is a root cause if none of its children failed
		// (children represent downstream nodes that depend on this one)
		isRoot := true

		// Check if this node has any failed dependencies by looking at the workflow structure
		// We consider a node a root cause if:
		// 1. It's a Pod/Container type (leaf node that actually ran)
		// 2. Or it has no children that are also failed
		wfNode := wf.Status.Nodes[node.ID]
		if wfNode.Type == wfv1.NodeTypePod || wfNode.Type == wfv1.NodeTypeContainer {
			// This is a leaf node that actually executed - check if it's the first failure
			// by checking if any of its boundary/parent nodes had children that failed before this one
			for _, otherNode := range failedNodes {
				if otherNode.ID != node.ID &&
					otherNode.FinishedAt.Before(node.StartedAt) &&
					isUpstream(wf, otherNode.ID, node.ID) {
					isRoot = false
					break
				}
			}
		} else {
			// Non-leaf nodes (DAG, Steps) - they fail because their children fail
			// Don't mark these as root causes, unless they have no failed children
			for _, childID := range node.Children {
				if failedIDs[childID] {
					isRoot = false
					break
				}
			}
		}

		if isRoot {
			nodeCopy := node
			nodeCopy.IsRootCause = true
			rootCauses = append(rootCauses, nodeCopy)
		}
	}

	// If no root causes found, return the first failed node
	if len(rootCauses) == 0 && len(failedNodes) > 0 {
		first := failedNodes[0]
		first.IsRootCause = true
		rootCauses = append(rootCauses, first)
	}

	return rootCauses
}

// isUpstream checks if nodeA is upstream of nodeB in the workflow graph.
func isUpstream(wf *wfv1.Workflow, nodeAID, nodeBID string) bool {
	// Simple check: A is upstream of B if B depends on A's outputs or A is in B's boundary
	nodeB := wf.Status.Nodes[nodeBID]

	// Check direct parent relationship
	if nodeB.BoundaryID == nodeAID {
		return true
	}

	// Check through children relationships (if A has B as descendant)
	visited := make(map[string]bool)
	return hasDescendant(wf, nodeAID, nodeBID, visited)
}

// hasDescendant recursively checks if targetID is a descendant of nodeID.
func hasDescendant(wf *wfv1.Workflow, nodeID, targetID string, visited map[string]bool) bool {
	if visited[nodeID] {
		return false
	}
	visited[nodeID] = true

	node := wf.Status.Nodes[nodeID]
	for _, childID := range node.Children {
		if childID == targetID {
			return true
		}
		if hasDescendant(wf, childID, targetID, visited) {
			return true
		}
	}
	return false
}

// getNodeLogs retrieves logs for a specific node.
// Returns whatever logs were collected even if streaming ends with an error (including EOF).
//
//nolint:nilerr // Intentionally returning nil error - we want partial logs even if stream errors
func getNodeLogs(ctx context.Context, client argo.ClientInterface, namespace, workflowName, nodeName string) (string, error) {
	wfService := client.WorkflowService()

	tailLines := int64(defaultLogTailLines)
	logOptions := &corev1.PodLogOptions{
		TailLines: &tailLines,
		Container: "main",
	}

	req := &workflow.WorkflowLogRequest{
		Namespace:  namespace,
		Name:       workflowName,
		PodName:    nodeName, // In Argo, the pod name is typically the node name
		LogOptions: logOptions,
	}

	stream, err := wfService.WorkflowLogs(ctx, req)
	if err != nil {
		return "", err
	}

	var logs strings.Builder
	for {
		entry, recvErr := stream.Recv()
		if recvErr != nil {
			// EOF or other error means end of stream - return what we have
			break
		}
		logs.WriteString(entry.Content)
		logs.WriteString("\n")
	}

	return logs.String(), nil
}

// describeValueFrom returns a description of a parameter's value source.
func describeValueFrom(vf *wfv1.ValueFrom) string {
	if vf == nil {
		return ""
	}

	parts := []string{}
	if vf.Path != "" {
		parts = append(parts, fmt.Sprintf("path: %s", vf.Path))
	}
	if vf.JSONPath != "" {
		parts = append(parts, fmt.Sprintf("jsonpath: %s", vf.JSONPath))
	}
	if vf.JQFilter != "" {
		parts = append(parts, fmt.Sprintf("jq: %s", vf.JQFilter))
	}
	if vf.Parameter != "" {
		parts = append(parts, fmt.Sprintf("parameter: %s", vf.Parameter))
	}
	if vf.Expression != "" {
		parts = append(parts, fmt.Sprintf("expression: %s", vf.Expression))
	}
	if vf.Default != nil {
		parts = append(parts, fmt.Sprintf("default: %s", string(*vf.Default)))
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ", ")
}

// describeArtifactSource returns a description of an artifact's source.
func describeArtifactSource(a *wfv1.Artifact) string {
	if a.From != "" {
		return fmt.Sprintf("from: %s", a.From)
	}
	if a.S3 != nil {
		return fmt.Sprintf("s3: %s/%s", a.S3.Bucket, a.S3.Key)
	}
	if a.GCS != nil {
		return fmt.Sprintf("gcs: %s/%s", a.GCS.Bucket, a.GCS.Key)
	}
	if a.HTTP != nil {
		return fmt.Sprintf("http: %s", a.HTTP.URL)
	}
	if a.Git != nil {
		return fmt.Sprintf("git: %s", a.Git.Repo)
	}
	return ""
}

// detectErrorPattern analyzes a failed node and identifies common error patterns.
func detectErrorPattern(node *failedNodeInfo) *errorPattern {
	message := strings.ToLower(node.Message)
	logs := strings.ToLower(node.Logs)

	// Check exit code patterns
	switch node.ExitCode {
	case "137":
		return &errorPattern{
			Pattern:    "Exit code 137 (OOMKilled or SIGKILL)",
			Suggestion: "The container was killed, likely due to out-of-memory (OOM). Consider increasing memory limits in the template's resources.limits.memory field.",
		}
	case "139":
		return &errorPattern{
			Pattern:    "Exit code 139 (Segmentation fault)",
			Suggestion: "The container crashed with a segmentation fault. This is typically a bug in the code or a native library issue.",
		}
	case "143":
		return &errorPattern{
			Pattern:    "Exit code 143 (SIGTERM)",
			Suggestion: "The container received SIGTERM, typically due to timeout or workflow termination. Check activeDeadlineSeconds settings.",
		}
	}

	// Check message patterns
	if strings.Contains(message, "oomkilled") || strings.Contains(logs, "oomkilled") {
		return &errorPattern{
			Pattern:    "OOMKilled",
			Suggestion: "The container ran out of memory. Increase memory limits in the template's resources.limits.memory field.",
		}
	}

	if strings.Contains(message, "imagepullbackoff") || strings.Contains(message, "errimagepull") {
		return &errorPattern{
			Pattern:    "ImagePullBackOff",
			Suggestion: "Failed to pull container image. Check: 1) Image name and tag are correct, 2) Image exists in the registry, 3) imagePullSecrets are configured if using private registry.",
		}
	}

	if strings.Contains(message, "deadline exceeded") || strings.Contains(message, "timeout") {
		return &errorPattern{
			Pattern:    "Timeout/Deadline exceeded",
			Suggestion: "The workflow or step exceeded its time limit. Consider increasing activeDeadlineSeconds at the workflow or template level.",
		}
	}

	if strings.Contains(message, "permission denied") || strings.Contains(logs, "permission denied") {
		return &errorPattern{
			Pattern:    "Permission denied",
			Suggestion: "Permission error detected. Check: 1) ServiceAccount has required RBAC permissions, 2) File/directory permissions in container, 3) PodSecurityPolicy/SecurityContext settings.",
		}
	}

	if strings.Contains(logs, "no space left on device") {
		return &errorPattern{
			Pattern:    "No space left on device",
			Suggestion: "Disk space exhausted. Consider: 1) Increasing volume size, 2) Using ephemeral volumes, 3) Cleaning up artifacts between steps.",
		}
	}

	// Python-specific patterns
	if strings.Contains(logs, "traceback") && strings.Contains(logs, "error") {
		return &errorPattern{
			Pattern:    "Python exception",
			Suggestion: "A Python exception occurred. Check the traceback in the logs for the specific error type and location.",
		}
	}

	// Connection/Network patterns
	if strings.Contains(logs, "connection refused") || strings.Contains(logs, "connection timed out") {
		return &errorPattern{
			Pattern:    "Network connection error",
			Suggestion: "Network connectivity issue. Check: 1) Target service is running and accessible, 2) Network policies allow the connection, 3) DNS resolution is working.",
		}
	}

	return nil
}

// buildPromptText constructs the prompt text from diagnostic information.
func buildPromptText(d *diagnosis) string {
	var sb strings.Builder

	// Header and instructions
	sb.WriteString("You are diagnosing a failed Argo Workflow. Analyse the following information and explain:\n")
	sb.WriteString("1. What failed and why\n")
	sb.WriteString("2. The root cause (trace back to the original failure)\n")
	sb.WriteString("3. Any suspicious inputs or upstream issues\n")
	sb.WriteString("4. Recommended fixes\n\n")

	// Workflow overview
	fmt.Fprintf(&sb, "## Workflow: %s\n", d.WorkflowName)
	fmt.Fprintf(&sb, "Namespace: %s\n", d.Namespace)
	fmt.Fprintf(&sb, "Status: %s\n", d.Phase)
	if d.Message != "" {
		fmt.Fprintf(&sb, "Message: %s\n", d.Message)
	}
	if !d.StartedAt.IsZero() {
		fmt.Fprintf(&sb, "Started: %s\n", d.StartedAt.Format(time.RFC3339))
	}
	if !d.FinishedAt.IsZero() {
		fmt.Fprintf(&sb, "Finished: %s\n", d.FinishedAt.Format(time.RFC3339))
	}
	if d.Duration > 0 {
		fmt.Fprintf(&sb, "Duration: %s\n", formatDuration(d.Duration))
	}

	// Workflow parameters
	if len(d.Parameters) > 0 {
		sb.WriteString("\n### Workflow Parameters\n")
		for _, p := range d.Parameters {
			fmt.Fprintf(&sb, "- %s: %s\n", p.Name, truncateString(p.Value, 200))
		}
	}

	// Root cause nodes (primary focus)
	if len(d.RootCauses) > 0 {
		sb.WriteString("\n## Root Cause Node(s)\n")
		sb.WriteString("These are the nodes that appear to be the original source of failure:\n\n")
		for _, node := range d.RootCauses {
			writeNodeInfo(&sb, &node, true)
		}
	}

	// Other failed nodes (for context)
	otherFailed := filterOutRootCauses(d.FailedNodes, d.RootCauses)
	if len(otherFailed) > 0 {
		sb.WriteString("\n## Other Failed Nodes (Cascading Failures)\n")
		sb.WriteString("These nodes failed as a result of upstream failures:\n\n")
		for _, node := range otherFailed {
			writeNodeInfo(&sb, &node, false)
		}
	}

	// Summary of suggested actions
	if hasErrorPatterns(d.RootCauses) {
		sb.WriteString("\n## Detected Error Patterns\n")
		for _, node := range d.RootCauses {
			if node.ErrorPattern != nil {
				fmt.Fprintf(&sb, "\n### %s\n", node.DisplayName)
				fmt.Fprintf(&sb, "**Pattern**: %s\n", node.ErrorPattern.Pattern)
				fmt.Fprintf(&sb, "**Suggestion**: %s\n", node.ErrorPattern.Suggestion)
			}
		}
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString("Based on this information, explain the failure and suggest fixes.\n")

	return sb.String()
}

// writeNodeInfo writes detailed information about a node to the string builder.
func writeNodeInfo(sb *strings.Builder, node *failedNodeInfo, includeFullDetails bool) {
	displayName := node.DisplayName
	if displayName == "" {
		displayName = node.Name
	}

	fmt.Fprintf(sb, "### Node: %s", displayName)
	if node.IsRootCause {
		sb.WriteString(" (ROOT CAUSE)")
	}
	sb.WriteString("\n")

	if node.TemplateName != "" {
		fmt.Fprintf(sb, "Template: %s\n", node.TemplateName)
	}
	fmt.Fprintf(sb, "Phase: %s\n", node.Phase)
	if node.Message != "" {
		fmt.Fprintf(sb, "Message: %s\n", node.Message)
	}
	if node.ExitCode != "" {
		fmt.Fprintf(sb, "Exit Code: %s\n", node.ExitCode)
	}
	if !node.StartedAt.IsZero() {
		fmt.Fprintf(sb, "Started: %s\n", node.StartedAt.Format(time.RFC3339))
	}
	if !node.FinishedAt.IsZero() {
		fmt.Fprintf(sb, "Finished: %s\n", node.FinishedAt.Format(time.RFC3339))
	}

	// Inputs (always show for root causes, abbreviated for others)
	if len(node.Inputs) > 0 {
		sb.WriteString("\nInputs:\n")
		for _, input := range node.Inputs {
			if input.Type == "parameter" {
				value := truncateString(input.Value, 100)
				if input.Source != "" {
					fmt.Fprintf(sb, "  - %s [param]: %s (source: %s)\n", input.Name, value, input.Source)
				} else {
					fmt.Fprintf(sb, "  - %s [param]: %s\n", input.Name, value)
				}
			} else {
				if input.Source != "" {
					fmt.Fprintf(sb, "  - %s [artifact]: %s\n", input.Name, input.Source)
				} else {
					fmt.Fprintf(sb, "  - %s [artifact]\n", input.Name)
				}
			}
		}
	}

	// Outputs (show for context nodes that succeeded before root cause)
	if len(node.Outputs) > 0 && !includeFullDetails {
		sb.WriteString("\nOutputs:\n")
		for _, output := range node.Outputs {
			if output.Type == "parameter" {
				fmt.Fprintf(sb, "  - %s [param]: %s\n", output.Name, truncateString(output.Value, 100))
			} else {
				fmt.Fprintf(sb, "  - %s [artifact]\n", output.Name)
			}
		}
	}

	// Logs (only for root causes with full details)
	if includeFullDetails && node.Logs != "" {
		sb.WriteString("\nLogs (last lines):\n```\n")
		sb.WriteString(node.Logs)
		sb.WriteString("```\n")
	}

	sb.WriteString("\n")
}

// filterOutRootCauses returns failed nodes that are not root causes.
func filterOutRootCauses(failed, rootCauses []failedNodeInfo) []failedNodeInfo {
	rootIDs := make(map[string]bool)
	for _, r := range rootCauses {
		rootIDs[r.ID] = true
	}

	var other []failedNodeInfo
	for _, node := range failed {
		if !rootIDs[node.ID] {
			other = append(other, node)
		}
	}
	return other
}

// hasErrorPatterns checks if any root cause has a detected error pattern.
func hasErrorPatterns(rootCauses []failedNodeInfo) bool {
	for _, node := range rootCauses {
		if node.ErrorPattern != nil {
			return true
		}
	}
	return false
}

// formatDuration formats a duration in a human-readable format.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
}

// truncateString truncates a string to a maximum number of runes (unicode-safe).
func truncateString(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}

// truncateStringBytes truncates a string to approximately maxBytes while respecting UTF-8 boundaries.
// It finds the largest valid UTF-8 boundary at or before maxBytes.
func truncateStringBytes(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	// Find a valid UTF-8 boundary at or before maxBytes
	// UTF-8 continuation bytes have the pattern 10xxxxxx (0x80-0xBF)
	// So we back up until we find a byte that's not a continuation byte
	end := maxBytes
	for end > 0 && (s[end]&0xC0) == 0x80 {
		end--
	}
	return s[:end]
}
