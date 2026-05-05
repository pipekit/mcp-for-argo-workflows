package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeleteWorkflowTool(t *testing.T) {
	tool := DeleteWorkflowTool()

	assert.Equal(t, "delete_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
}

func TestDeleteWorkflowHandler_Validation(t *testing.T) {
	// Test handler validation directly - validation errors occur before client is used,
	// so we can pass nil and test that validation returns the expected errors.
	handler := DeleteWorkflowHandler(nil)

	tests := []struct {
		name        string
		errContains string
		input       DeleteWorkflowInput
		wantErr     bool
	}{
		{
			name: "empty name returns error",
			input: DeleteWorkflowInput{
				Name: "",
			},
			wantErr:     true,
			errContains: "workflow name cannot be empty",
		},
		{
			name: "whitespace-only name returns error",
			input: DeleteWorkflowInput{
				Name: "   ",
			},
			wantErr:     true,
			errContains: "workflow name cannot be empty",
		},
		{
			name: "whitespace-padded name returns error",
			input: DeleteWorkflowInput{
				Name: "  \t\n  ",
			},
			wantErr:     true,
			errContains: "workflow name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := handler(t.Context(), nil, tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			}
		})
	}
}

func TestDeleteWorkflowOutput(t *testing.T) {
	output := &DeleteWorkflowOutput{
		Name:      "test-workflow",
		Namespace: "default",
		Message:   "Workflow \"test-workflow\" deleted successfully",
	}

	assert.Equal(t, "test-workflow", output.Name)
	assert.Equal(t, "default", output.Namespace)
	assert.Contains(t, output.Message, "deleted successfully")
}
