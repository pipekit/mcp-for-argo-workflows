package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListWorkflowsTool(t *testing.T) {
	tool := ListWorkflowsTool()

	assert.Equal(t, "list_workflows", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "List")
}

func TestListWorkflowsInput(t *testing.T) {
	// Test default values
	input := ListWorkflowsInput{}
	assert.Nil(t, input.Namespace)
	assert.Empty(t, input.Status)
	assert.Empty(t, input.Labels)
	assert.Zero(t, input.Limit)

	// Test with values
	ns := "test-namespace"
	input2 := ListWorkflowsInput{
		Namespace: &ns,
		Status:    []string{"Running", "Pending"},
		Labels:    "app=test",
		Limit:     100,
	}
	assert.Equal(t, "test-namespace", *input2.Namespace)
	assert.Len(t, input2.Status, 2)
	assert.Equal(t, "app=test", input2.Labels)
	assert.Equal(t, int64(100), input2.Limit)
}

func TestWorkflowSummary(t *testing.T) {
	summary := WorkflowSummary{
		Name:       "test-workflow",
		Namespace:  "default",
		Phase:      "Running",
		CreatedAt:  "2025-01-01T00:00:00Z",
		FinishedAt: "",
		Message:    "Workflow is running",
	}

	assert.Equal(t, "test-workflow", summary.Name)
	assert.Equal(t, "default", summary.Namespace)
	assert.Equal(t, "Running", summary.Phase)
	assert.NotEmpty(t, summary.CreatedAt)
	assert.Empty(t, summary.FinishedAt)
	assert.NotEmpty(t, summary.Message)
}

func TestListWorkflowsOutput(t *testing.T) {
	output := ListWorkflowsOutput{
		Workflows: []WorkflowSummary{
			{Name: "wf-1", Namespace: "default", Phase: "Succeeded"},
			{Name: "wf-2", Namespace: "default", Phase: "Running"},
		},
		Total: 2,
	}

	assert.Len(t, output.Workflows, 2)
	assert.Equal(t, 2, output.Total)
	assert.Equal(t, "wf-1", output.Workflows[0].Name)
	assert.Equal(t, "wf-2", output.Workflows[1].Name)
}

func TestValidPhases(t *testing.T) {
	// These are the valid phases that the tool should accept
	validPhases := []string{"Pending", "Running", "Succeeded", "Failed", "Error"}

	for _, phase := range validPhases {
		// Use the exported ValidWorkflowPhases from the package
		assert.True(t, ValidWorkflowPhases[phase], "phase %s should be valid", phase)
	}

	// Invalid phases - case sensitive and must be exact
	invalidPhases := []string{"pending", "RUNNING", "Unknown", ""}
	for _, phase := range invalidPhases {
		assert.False(t, ValidWorkflowPhases[phase], "phase %q should be invalid", phase)
	}
}
