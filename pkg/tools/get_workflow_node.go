// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// GetWorkflowNodeInput defines the input parameters for the get_workflow_node tool.
type GetWorkflowNodeInput struct {
	// Namespace is the Kubernetes namespace (uses default if not specified).
	Namespace string `json:"namespace,omitempty" jsonschema:"Kubernetes namespace (uses default if not specified)"`

	// WorkflowName is the workflow name.
	WorkflowName string `json:"workflowName" jsonschema:"Workflow name,required"`

	// NodeName is the node name, display name, or ID within the workflow.
	NodeName string `json:"nodeName" jsonschema:"Node name, display name, or ID,required"`
}

// GetWorkflowNodeOutput defines the output for the get_workflow_node tool.
//
//nolint:govet // Field order optimized for readability over memory alignment
type GetWorkflowNodeOutput struct {
	Inputs       *NodeInputsOutput  `json:"inputs,omitempty"`
	Outputs      *NodeOutputsOutput `json:"outputs,omitempty"`
	Phase        string             `json:"phase"`
	Message      string             `json:"message,omitempty"`
	Name         string             `json:"name"`
	DisplayName  string             `json:"displayName,omitempty"`
	Type         string             `json:"type"`
	TemplateName string             `json:"templateName,omitempty"`
	HostNodeName string             `json:"hostNodeName,omitempty"`
	ID           string             `json:"id"`
	StartedAt    string             `json:"startedAt,omitempty"`
	FinishedAt   string             `json:"finishedAt,omitempty"`
	Duration     string             `json:"duration,omitempty"`
	Progress     string             `json:"progress,omitempty"`
	PodIP        string             `json:"podIp,omitempty"`
	BoundaryID   string             `json:"boundaryId,omitempty"`
	Children     []string           `json:"children,omitempty"`
}

// NodeInputsOutput represents node inputs.
type NodeInputsOutput struct {
	// Parameters are the input parameters.
	Parameters []ParameterInfo `json:"parameters,omitempty"`

	// Artifacts are the input artifacts.
	Artifacts []ArtifactInfo `json:"artifacts,omitempty"`
}

// NodeOutputsOutput represents node outputs.
//
//nolint:govet // Field order optimized for readability over memory alignment
type NodeOutputsOutput struct {
	ExitCode   string          `json:"exitCode,omitempty"`
	Result     string          `json:"result,omitempty"`
	Parameters []ParameterInfo `json:"parameters,omitempty"`
	Artifacts  []ArtifactInfo  `json:"artifacts,omitempty"`
}

// ArtifactInfo represents an artifact.
type ArtifactInfo struct {
	// Name is the artifact name.
	Name string `json:"name"`

	// Path is the artifact path in the container.
	Path string `json:"path,omitempty"`
}

// GetWorkflowNodeTool returns the MCP tool definition for get_workflow_node.
func GetWorkflowNodeTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "get_workflow_node",
		Description: "Get details of a specific node within an Argo Workflow",
		Annotations: &mcp.ToolAnnotations{
			ReadOnlyHint: true,
		},
	}
}

// GetWorkflowNodeHandler returns a handler function for the get_workflow_node tool.
func GetWorkflowNodeHandler(client argo.ClientInterface) func(context.Context, *mcp.CallToolRequest, GetWorkflowNodeInput) (*mcp.CallToolResult, *GetWorkflowNodeOutput, error) {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input GetWorkflowNodeInput) (*mcp.CallToolResult, *GetWorkflowNodeOutput, error) {
		// Validate workflow name
		workflowName, err := ValidateName(input.WorkflowName)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid workflow name: %w", err)
		}

		// Validate node name
		nodeName, err := ValidateName(input.NodeName)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid node name: %w", err)
		}

		// Determine namespace
		namespace := ResolveNamespace(input.Namespace, client)

		// Get the workflow service client
		wfService := client.WorkflowService()

		// Get the workflow
		wf, err := wfService.GetWorkflow(ctx, &workflow.WorkflowGetRequest{
			Namespace: namespace,
			Name:      workflowName,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get workflow: %w", err)
		}

		// Find the node by name or ID
		node, err := findNode(wf.Status.Nodes, nodeName)
		if err != nil {
			return nil, nil, err
		}

		// Build the output
		output := buildNodeOutput(node)

		// Build human-readable result
		resultText := buildNodeResultText(output, workflowName, namespace)

		return TextResult(resultText), output, nil
	}
}

// findNode finds a node by name or ID in the workflow nodes.
func findNode(nodes wfv1.Nodes, nameOrID string) (*wfv1.NodeStatus, error) {
	// First try to find by ID (exact match)
	if node, exists := nodes[nameOrID]; exists {
		nodeCopy := node
		return &nodeCopy, nil
	}

	// Then try to find by name (exact match)
	for _, node := range nodes {
		if node.Name == nameOrID {
			nodeCopy := node
			return &nodeCopy, nil
		}
	}

	// Finally, try to find by display name (exact match)
	for _, node := range nodes {
		if node.DisplayName == nameOrID {
			nodeCopy := node
			return &nodeCopy, nil
		}
	}

	return nil, fmt.Errorf("node %q not found in workflow", nameOrID)
}

// buildNodeOutput constructs the output from a node status.
func buildNodeOutput(node *wfv1.NodeStatus) *GetWorkflowNodeOutput {
	output := &GetWorkflowNodeOutput{
		ID:           node.ID,
		Name:         node.Name,
		DisplayName:  node.DisplayName,
		Type:         string(node.Type),
		TemplateName: node.TemplateName,
		Phase:        string(node.Phase),
		Message:      node.Message,
		BoundaryID:   node.BoundaryID,
		PodIP:        node.PodIP,
		HostNodeName: node.HostNodeName,
	}

	// Set a default phase if empty
	if output.Phase == "" {
		output.Phase = PhasePending
	}

	// Set timing information
	if !node.StartedAt.Time.IsZero() {
		output.StartedAt = node.StartedAt.Format(time.RFC3339)
	}
	if !node.FinishedAt.Time.IsZero() {
		output.FinishedAt = node.FinishedAt.Format(time.RFC3339)
	}

	// Calculate duration
	if !node.StartedAt.Time.IsZero() {
		endTime := node.FinishedAt.Time
		if endTime.IsZero() {
			endTime = time.Now()
		}
		duration := endTime.Sub(node.StartedAt.Time)
		output.Duration = formatDuration(duration)
	}

	// Set progress
	if node.Progress != "" {
		output.Progress = string(node.Progress)
	}

	// Set children
	if len(node.Children) > 0 {
		output.Children = node.Children
	}

	// Set inputs
	if node.Inputs != nil {
		output.Inputs = buildNodeInputsOutput(node.Inputs)
	}

	// Set outputs
	if node.Outputs != nil {
		output.Outputs = buildNodeOutputsOutput(node.Outputs)
	}

	return output
}

// buildNodeInputsOutput builds the inputs output structure.
func buildNodeInputsOutput(inputs *wfv1.Inputs) *NodeInputsOutput {
	if inputs == nil {
		return nil
	}

	result := &NodeInputsOutput{}

	// Extract parameters
	if len(inputs.Parameters) > 0 {
		result.Parameters = make([]ParameterInfo, 0, len(inputs.Parameters))
		for _, p := range inputs.Parameters {
			param := ParameterInfo{Name: p.Name}
			if p.Value != nil {
				param.Value = string(*p.Value)
			}
			result.Parameters = append(result.Parameters, param)
		}
	}

	// Extract artifacts
	if len(inputs.Artifacts) > 0 {
		result.Artifacts = make([]ArtifactInfo, 0, len(inputs.Artifacts))
		for _, a := range inputs.Artifacts {
			artifact := ArtifactInfo{
				Name: a.Name,
				Path: a.Path,
			}
			result.Artifacts = append(result.Artifacts, artifact)
		}
	}

	// Return nil if both are empty
	if len(result.Parameters) == 0 && len(result.Artifacts) == 0 {
		return nil
	}

	return result
}

// buildNodeOutputsOutput builds the outputs output structure.
func buildNodeOutputsOutput(outputs *wfv1.Outputs) *NodeOutputsOutput {
	if outputs == nil {
		return nil
	}

	result := &NodeOutputsOutput{}

	// Extract parameters
	if len(outputs.Parameters) > 0 {
		result.Parameters = make([]ParameterInfo, 0, len(outputs.Parameters))
		for _, p := range outputs.Parameters {
			param := ParameterInfo{Name: p.Name}
			if p.Value != nil {
				param.Value = string(*p.Value)
			}
			result.Parameters = append(result.Parameters, param)
		}
	}

	// Extract artifacts
	if len(outputs.Artifacts) > 0 {
		result.Artifacts = make([]ArtifactInfo, 0, len(outputs.Artifacts))
		for _, a := range outputs.Artifacts {
			artifact := ArtifactInfo{
				Name: a.Name,
				Path: a.Path,
			}
			result.Artifacts = append(result.Artifacts, artifact)
		}
	}

	// Extract exit code
	if outputs.ExitCode != nil {
		result.ExitCode = *outputs.ExitCode
	}

	// Extract result
	if outputs.Result != nil {
		result.Result = *outputs.Result
	}

	// Return nil if all are empty
	if len(result.Parameters) == 0 && len(result.Artifacts) == 0 && result.ExitCode == "" && result.Result == "" {
		return nil
	}

	return result
}

// buildNodeResultText builds a human-readable text result.
func buildNodeResultText(output *GetWorkflowNodeOutput, workflowName, namespace string) string {
	result := fmt.Sprintf("Node %q in workflow %q (namespace: %s)\n", output.Name, workflowName, namespace)
	result += fmt.Sprintf("  Type: %s\n", output.Type)
	result += fmt.Sprintf("  Phase: %s\n", output.Phase)

	if output.DisplayName != "" && output.DisplayName != output.Name {
		result += fmt.Sprintf("  Display Name: %s\n", output.DisplayName)
	}

	if output.TemplateName != "" {
		result += fmt.Sprintf("  Template: %s\n", output.TemplateName)
	}

	if output.Message != "" {
		result += fmt.Sprintf("  Message: %s\n", output.Message)
	}

	if output.StartedAt != "" {
		result += fmt.Sprintf("  Started: %s\n", output.StartedAt)
	}

	if output.FinishedAt != "" {
		result += fmt.Sprintf("  Finished: %s\n", output.FinishedAt)
	}

	if output.Duration != "" {
		result += fmt.Sprintf("  Duration: %s\n", output.Duration)
	}

	if output.Progress != "" {
		result += fmt.Sprintf("  Progress: %s\n", output.Progress)
	}

	if output.PodIP != "" {
		result += fmt.Sprintf("  Pod IP: %s\n", output.PodIP)
	}

	if output.HostNodeName != "" {
		result += fmt.Sprintf("  Host Node: %s\n", output.HostNodeName)
	}

	if output.Inputs != nil {
		result += "  Inputs:\n"
		for _, p := range output.Inputs.Parameters {
			result += fmt.Sprintf("    - %s: %s\n", p.Name, p.Value)
		}
		for _, a := range output.Inputs.Artifacts {
			result += fmt.Sprintf("    - [artifact] %s: %s\n", a.Name, a.Path)
		}
	}

	if output.Outputs != nil {
		result += "  Outputs:\n"
		if output.Outputs.ExitCode != "" {
			result += fmt.Sprintf("    Exit Code: %s\n", output.Outputs.ExitCode)
		}
		if output.Outputs.Result != "" {
			result += fmt.Sprintf("    Result: %s\n", output.Outputs.Result)
		}
		for _, p := range output.Outputs.Parameters {
			result += fmt.Sprintf("    - %s: %s\n", p.Name, p.Value)
		}
		for _, a := range output.Outputs.Artifacts {
			result += fmt.Sprintf("    - [artifact] %s: %s\n", a.Name, a.Path)
		}
	}

	if len(output.Children) > 0 {
		result += fmt.Sprintf("  Children: %v\n", output.Children)
	}

	return result
}
