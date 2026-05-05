package tools

import (
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestGetWorkflowTool(t *testing.T) {
	tool := GetWorkflowTool()

	assert.Equal(t, "get_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
}

func TestBuildGetWorkflowOutput(t *testing.T) {
	startTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 1, 15, 10, 5, 30, 0, time.UTC)

	tests := []struct {
		workflow *wfv1.Workflow
		validate func(t *testing.T, output *GetWorkflowOutput)
		name     string
	}{
		{
			name: "complete workflow with all fields",
			workflow: &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-workflow",
					Namespace: "default",
					UID:       types.UID("abc-123-def"),
				},
				Spec: wfv1.WorkflowSpec{
					Arguments: wfv1.Arguments{
						Parameters: []wfv1.Parameter{
							{Name: "message", Value: wfv1.AnyStringPtr("hello")},
							{Name: "count", Value: wfv1.AnyStringPtr("5")},
						},
					},
				},
				Status: wfv1.WorkflowStatus{
					Phase:      wfv1.WorkflowSucceeded,
					Message:    "Workflow completed successfully",
					StartedAt:  metav1.Time{Time: startTime},
					FinishedAt: metav1.Time{Time: endTime},
					Progress:   "3/3",
					Nodes: wfv1.Nodes{
						"node1": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded},
						"node2": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded},
						"node3": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded},
					},
				},
			},
			validate: func(t *testing.T, output *GetWorkflowOutput) {
				assert.Equal(t, "test-workflow", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Equal(t, "abc-123-def", output.UID)
				assert.Equal(t, "Succeeded", output.Phase)
				assert.Equal(t, "Workflow completed successfully", output.Message)
				assert.Equal(t, "2025-01-15T10:00:00Z", output.StartedAt)
				assert.Equal(t, "2025-01-15T10:05:30Z", output.FinishedAt)
				assert.Equal(t, "5m30s", output.Duration)
				assert.Equal(t, "3/3", output.Progress)

				require.Len(t, output.Parameters, 2)
				assert.Equal(t, "message", output.Parameters[0].Name)
				assert.Equal(t, "hello", output.Parameters[0].Value)
				assert.Equal(t, "count", output.Parameters[1].Name)
				assert.Equal(t, "5", output.Parameters[1].Value)

				require.NotNil(t, output.NodeSummary)
				assert.Equal(t, 3, output.NodeSummary.Total)
				assert.Equal(t, 3, output.NodeSummary.Succeeded)
			},
		},
		{
			name: "pending workflow with no status",
			workflow: &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pending-workflow",
					Namespace: "test-ns",
					UID:       types.UID("pending-uid"),
				},
				Spec: wfv1.WorkflowSpec{},
				Status: wfv1.WorkflowStatus{
					Phase: "",
				},
			},
			validate: func(t *testing.T, output *GetWorkflowOutput) {
				assert.Equal(t, "pending-workflow", output.Name)
				assert.Equal(t, "test-ns", output.Namespace)
				assert.Equal(t, "Pending", output.Phase)
				assert.Empty(t, output.StartedAt)
				assert.Empty(t, output.FinishedAt)
				assert.Empty(t, output.Duration)
				assert.Empty(t, output.Parameters)
				assert.Nil(t, output.NodeSummary)
			},
		},
		{
			name: "failed workflow with mixed node statuses",
			workflow: &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failed-workflow",
					Namespace: "default",
					UID:       types.UID("failed-uid"),
				},
				Spec: wfv1.WorkflowSpec{},
				Status: wfv1.WorkflowStatus{
					Phase:     wfv1.WorkflowFailed,
					Message:   "Step failed",
					StartedAt: metav1.Time{Time: startTime},
					Nodes: wfv1.Nodes{
						"node1": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded},
						"node2": wfv1.NodeStatus{Phase: wfv1.NodeFailed},
						"node3": wfv1.NodeStatus{Phase: wfv1.NodeSkipped},
						"node4": wfv1.NodeStatus{Phase: wfv1.NodePending},
					},
				},
			},
			validate: func(t *testing.T, output *GetWorkflowOutput) {
				assert.Equal(t, "Failed", output.Phase)
				assert.Equal(t, "Step failed", output.Message)

				require.NotNil(t, output.NodeSummary)
				assert.Equal(t, 4, output.NodeSummary.Total)
				assert.Equal(t, 1, output.NodeSummary.Succeeded)
				assert.Equal(t, 1, output.NodeSummary.Failed)
				assert.Equal(t, 1, output.NodeSummary.Skipped)
				assert.Equal(t, 1, output.NodeSummary.Pending)
			},
		},
		{
			name: "running workflow",
			workflow: &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "running-workflow",
					Namespace: "default",
					UID:       types.UID("running-uid"),
				},
				Spec: wfv1.WorkflowSpec{},
				Status: wfv1.WorkflowStatus{
					Phase:     wfv1.WorkflowRunning,
					StartedAt: metav1.Time{Time: startTime},
					Progress:  "1/3",
					Nodes: wfv1.Nodes{
						"node1": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded},
						"node2": wfv1.NodeStatus{Phase: wfv1.NodeRunning},
						"node3": wfv1.NodeStatus{Phase: wfv1.NodePending},
					},
				},
			},
			validate: func(t *testing.T, output *GetWorkflowOutput) {
				assert.Equal(t, "Running", output.Phase)
				assert.NotEmpty(t, output.StartedAt)
				assert.Empty(t, output.FinishedAt)
				assert.NotEmpty(t, output.Duration) // Should be calculated from now
				assert.Equal(t, "1/3", output.Progress)

				require.NotNil(t, output.NodeSummary)
				assert.Equal(t, 1, output.NodeSummary.Succeeded)
				assert.Equal(t, 1, output.NodeSummary.Running)
				assert.Equal(t, 1, output.NodeSummary.Pending)
			},
		},
		{
			name: "workflow with parameter without value",
			workflow: &wfv1.Workflow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "param-workflow",
					Namespace: "default",
					UID:       types.UID("param-uid"),
				},
				Spec: wfv1.WorkflowSpec{
					Arguments: wfv1.Arguments{
						Parameters: []wfv1.Parameter{
							{Name: "required-param", Value: nil},
							{Name: "optional-param", Value: wfv1.AnyStringPtr("value")},
						},
					},
				},
				Status: wfv1.WorkflowStatus{
					Phase: wfv1.WorkflowPending,
				},
			},
			validate: func(t *testing.T, output *GetWorkflowOutput) {
				require.Len(t, output.Parameters, 2)
				assert.Equal(t, "required-param", output.Parameters[0].Name)
				assert.Empty(t, output.Parameters[0].Value)
				assert.Equal(t, "optional-param", output.Parameters[1].Name)
				assert.Equal(t, "value", output.Parameters[1].Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := buildGetWorkflowOutput(tt.workflow)
			require.NotNil(t, output)
			tt.validate(t, output)
		})
	}
}

func TestBuildNodeSummary(t *testing.T) {
	tests := []struct {
		expected *NodeSummary
		nodes    wfv1.Nodes
		name     string
	}{
		{
			name: "all node phases",
			nodes: wfv1.Nodes{
				"succeeded": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded},
				"failed":    wfv1.NodeStatus{Phase: wfv1.NodeFailed},
				"running":   wfv1.NodeStatus{Phase: wfv1.NodeRunning},
				"pending":   wfv1.NodeStatus{Phase: wfv1.NodePending},
				"skipped":   wfv1.NodeStatus{Phase: wfv1.NodeSkipped},
				"error":     wfv1.NodeStatus{Phase: wfv1.NodeError},
				"omitted":   wfv1.NodeStatus{Phase: wfv1.NodeOmitted},
			},
			expected: &NodeSummary{
				Total:     7,
				Succeeded: 1,
				Failed:    1,
				Running:   1,
				Pending:   1,
				Skipped:   1,
				Error:     1,
				Omitted:   1,
			},
		},
		{
			name:  "empty nodes",
			nodes: wfv1.Nodes{},
			expected: &NodeSummary{
				Total: 0,
			},
		},
		{
			name: "multiple succeeded",
			nodes: wfv1.Nodes{
				"node1": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded},
				"node2": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded},
				"node3": wfv1.NodeStatus{Phase: wfv1.NodeSucceeded},
			},
			expected: &NodeSummary{
				Total:     3,
				Succeeded: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildNodeSummary(tt.nodes)
			assert.Equal(t, tt.expected.Total, result.Total)
			assert.Equal(t, tt.expected.Succeeded, result.Succeeded)
			assert.Equal(t, tt.expected.Failed, result.Failed)
			assert.Equal(t, tt.expected.Running, result.Running)
			assert.Equal(t, tt.expected.Pending, result.Pending)
			assert.Equal(t, tt.expected.Skipped, result.Skipped)
			assert.Equal(t, tt.expected.Error, result.Error)
			assert.Equal(t, tt.expected.Omitted, result.Omitted)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		duration time.Duration
	}{
		{
			name:     "less than a minute",
			duration: 45 * time.Second,
			expected: "45s",
		},
		{
			name:     "exactly one minute",
			duration: 60 * time.Second,
			expected: "1m0s",
		},
		{
			name:     "minutes and seconds",
			duration: 5*time.Minute + 30*time.Second,
			expected: "5m30s",
		},
		{
			name:     "hours minutes seconds",
			duration: 2*time.Hour + 15*time.Minute + 45*time.Second,
			expected: "2h15m45s",
		},
		{
			name:     "zero duration",
			duration: 0,
			expected: "0s",
		},
		{
			name:     "exactly one hour",
			duration: 1 * time.Hour,
			expected: "1h0m0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}
