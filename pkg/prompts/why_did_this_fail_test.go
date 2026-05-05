package prompts

import (
	"testing"
	"time"
	"unicode/utf8"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWhyDidThisFailPrompt(t *testing.T) {
	prompt := WhyDidThisFailPrompt()

	assert.Equal(t, "why_did_this_fail", prompt.Name)
	assert.Equal(t, "Diagnose Workflow Failure", prompt.Title)
	assert.NotEmpty(t, prompt.Description)

	require.Len(t, prompt.Arguments, 2)
	assert.Equal(t, "workflow", prompt.Arguments[0].Name)
	assert.True(t, prompt.Arguments[0].Required)
	assert.Equal(t, "namespace", prompt.Arguments[1].Name)
	assert.False(t, prompt.Arguments[1].Required)
}

func TestFindFailedNodes(t *testing.T) {
	startTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 1, 15, 10, 5, 30, 0, time.UTC)

	//nolint:govet // Test struct field order optimized for readability
	tests := []struct {
		name          string
		workflow      *wfv1.Workflow
		expectedCount int
		validateFirst func(t *testing.T, node failedNodeInfo)
	}{
		{
			name: "workflow with failed and succeeded nodes",
			workflow: &wfv1.Workflow{
				Status: wfv1.WorkflowStatus{
					Nodes: wfv1.Nodes{
						"node1": wfv1.NodeStatus{
							ID:           "node1",
							Name:         "test-wf.step1",
							DisplayName:  "step1",
							TemplateName: "step1-template",
							Phase:        wfv1.NodeSucceeded,
						},
						"node2": wfv1.NodeStatus{
							ID:           "node2",
							Name:         "test-wf.step2",
							DisplayName:  "step2",
							TemplateName: "step2-template",
							Phase:        wfv1.NodeFailed,
							Message:      "Error: exit status 1",
							StartedAt:    metav1.Time{Time: startTime},
							FinishedAt:   metav1.Time{Time: endTime},
							Outputs: &wfv1.Outputs{
								ExitCode: strPtr("1"),
							},
						},
					},
				},
			},
			expectedCount: 1,
			validateFirst: func(t *testing.T, node failedNodeInfo) {
				assert.Equal(t, "node2", node.ID)
				assert.Equal(t, "test-wf.step2", node.Name)
				assert.Equal(t, "step2", node.DisplayName)
				assert.Equal(t, "Failed", node.Phase)
				assert.Equal(t, "Error: exit status 1", node.Message)
				assert.Equal(t, "1", node.ExitCode)
			},
		},
		{
			name: "workflow with error node",
			workflow: &wfv1.Workflow{
				Status: wfv1.WorkflowStatus{
					Nodes: wfv1.Nodes{
						"node1": wfv1.NodeStatus{
							ID:      "node1",
							Name:    "test-wf.step1",
							Phase:   wfv1.NodeError,
							Message: "PodGC error",
						},
					},
				},
			},
			expectedCount: 1,
			validateFirst: func(t *testing.T, node failedNodeInfo) {
				assert.Equal(t, "Error", node.Phase)
			},
		},
		{
			name: "workflow with no failed nodes",
			workflow: &wfv1.Workflow{
				Status: wfv1.WorkflowStatus{
					Nodes: wfv1.Nodes{
						"node1": wfv1.NodeStatus{
							ID:    "node1",
							Phase: wfv1.NodeSucceeded,
						},
						"node2": wfv1.NodeStatus{
							ID:    "node2",
							Phase: wfv1.NodeRunning,
						},
					},
				},
			},
			expectedCount: 0,
		},
		{
			name: "workflow with inputs and outputs",
			workflow: &wfv1.Workflow{
				Status: wfv1.WorkflowStatus{
					Nodes: wfv1.Nodes{
						"node1": wfv1.NodeStatus{
							ID:    "node1",
							Name:  "test-wf.process",
							Phase: wfv1.NodeFailed,
							Inputs: &wfv1.Inputs{
								Parameters: []wfv1.Parameter{
									{Name: "input-param", Value: wfv1.AnyStringPtr("test-value")},
								},
								Artifacts: []wfv1.Artifact{
									{Name: "input-artifact", From: "{{steps.download.outputs.artifacts.data}}"},
								},
							},
							Outputs: &wfv1.Outputs{
								Parameters: []wfv1.Parameter{
									{Name: "output-param", Value: wfv1.AnyStringPtr("result")},
								},
								ExitCode: strPtr("1"),
							},
						},
					},
				},
			},
			expectedCount: 1,
			validateFirst: func(t *testing.T, node failedNodeInfo) {
				require.Len(t, node.Inputs, 2)
				assert.Equal(t, "input-param", node.Inputs[0].Name)
				assert.Equal(t, "test-value", node.Inputs[0].Value)
				assert.Equal(t, "parameter", node.Inputs[0].Type)
				assert.Equal(t, "input-artifact", node.Inputs[1].Name)
				assert.Equal(t, "artifact", node.Inputs[1].Type)

				require.Len(t, node.Outputs, 1)
				assert.Equal(t, "output-param", node.Outputs[0].Name)
				assert.Equal(t, "result", node.Outputs[0].Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findFailedNodes(tt.workflow)
			assert.Len(t, result, tt.expectedCount)
			if tt.expectedCount > 0 && tt.validateFirst != nil {
				tt.validateFirst(t, result[0])
			}
		})
	}
}

func TestFindRootCauses(t *testing.T) {
	startTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	middleTime := time.Date(2025, 1, 15, 10, 1, 0, 0, time.UTC)
	endTime := time.Date(2025, 1, 15, 10, 2, 0, 0, time.UTC)

	tests := []struct {
		name              string
		workflow          *wfv1.Workflow
		failedNodes       []failedNodeInfo
		expectedRootCount int
	}{
		{
			name: "single failed pod node is root cause",
			workflow: &wfv1.Workflow{
				Status: wfv1.WorkflowStatus{
					Nodes: wfv1.Nodes{
						"node1": wfv1.NodeStatus{
							ID:    "node1",
							Type:  wfv1.NodeTypePod,
							Phase: wfv1.NodeFailed,
						},
					},
				},
			},
			failedNodes: []failedNodeInfo{
				{ID: "node1", StartedAt: startTime, FinishedAt: endTime},
			},
			expectedRootCount: 1,
		},
		{
			name: "DAG node with failed child is not root cause",
			workflow: &wfv1.Workflow{
				Status: wfv1.WorkflowStatus{
					Nodes: wfv1.Nodes{
						"dag": wfv1.NodeStatus{
							ID:       "dag",
							Type:     wfv1.NodeTypeDAG,
							Phase:    wfv1.NodeFailed,
							Children: []string{"pod1"},
						},
						"pod1": wfv1.NodeStatus{
							ID:    "pod1",
							Type:  wfv1.NodeTypePod,
							Phase: wfv1.NodeFailed,
						},
					},
				},
			},
			failedNodes: []failedNodeInfo{
				{ID: "dag", Children: []string{"pod1"}, StartedAt: startTime, FinishedAt: endTime},
				{ID: "pod1", StartedAt: middleTime, FinishedAt: endTime},
			},
			expectedRootCount: 1, // Only pod1 should be root cause
		},
		{
			name: "multiple independent failures are all root causes",
			workflow: &wfv1.Workflow{
				Status: wfv1.WorkflowStatus{
					Nodes: wfv1.Nodes{
						"pod1": wfv1.NodeStatus{
							ID:    "pod1",
							Type:  wfv1.NodeTypePod,
							Phase: wfv1.NodeFailed,
						},
						"pod2": wfv1.NodeStatus{
							ID:    "pod2",
							Type:  wfv1.NodeTypePod,
							Phase: wfv1.NodeFailed,
						},
					},
				},
			},
			failedNodes: []failedNodeInfo{
				{ID: "pod1", StartedAt: startTime, FinishedAt: middleTime},
				{ID: "pod2", StartedAt: startTime, FinishedAt: endTime},
			},
			expectedRootCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findRootCauses(tt.workflow, tt.failedNodes)
			assert.Len(t, result, tt.expectedRootCount)
			for _, rc := range result {
				assert.True(t, rc.IsRootCause)
			}
		})
	}
}

func TestDetectErrorPattern(t *testing.T) {
	tests := []struct {
		name            string
		node            *failedNodeInfo
		expectedPattern string
		expectNil       bool
	}{
		{
			name:            "exit code 137 (OOM)",
			node:            &failedNodeInfo{ExitCode: "137"},
			expectedPattern: "Exit code 137 (OOMKilled or SIGKILL)",
		},
		{
			name:            "exit code 139 (segfault)",
			node:            &failedNodeInfo{ExitCode: "139"},
			expectedPattern: "Exit code 139 (Segmentation fault)",
		},
		{
			name:            "exit code 143 (SIGTERM)",
			node:            &failedNodeInfo{ExitCode: "143"},
			expectedPattern: "Exit code 143 (SIGTERM)",
		},
		{
			name:            "OOMKilled in message",
			node:            &failedNodeInfo{Message: "Container was OOMKilled"},
			expectedPattern: "OOMKilled",
		},
		{
			name:            "ImagePullBackOff",
			node:            &failedNodeInfo{Message: "ImagePullBackOff: image not found"},
			expectedPattern: "ImagePullBackOff",
		},
		{
			name:            "timeout in message",
			node:            &failedNodeInfo{Message: "deadline exceeded"},
			expectedPattern: "Timeout/Deadline exceeded",
		},
		{
			name:            "permission denied in logs",
			node:            &failedNodeInfo{Logs: "Error: permission denied when accessing /data"},
			expectedPattern: "Permission denied",
		},
		{
			name:            "disk space in logs",
			node:            &failedNodeInfo{Logs: "write error: no space left on device"},
			expectedPattern: "No space left on device",
		},
		{
			name:            "python traceback",
			node:            &failedNodeInfo{Logs: "Traceback (most recent call last):\n  File 'test.py'\nError: ValueError"},
			expectedPattern: "Python exception",
		},
		{
			name:            "connection refused",
			node:            &failedNodeInfo{Logs: "dial tcp: connection refused"},
			expectedPattern: "Network connection error",
		},
		{
			name:      "no recognized pattern",
			node:      &failedNodeInfo{Message: "Unknown error", Logs: "Some random log output"},
			expectNil: true,
		},
		{
			name:      "empty node",
			node:      &failedNodeInfo{},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectErrorPattern(tt.node)
			if tt.expectNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tt.expectedPattern, result.Pattern)
				assert.NotEmpty(t, result.Suggestion)
			}
		})
	}
}

func TestBuildPromptText(t *testing.T) {
	startTime := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 1, 15, 10, 5, 30, 0, time.UTC)

	d := &diagnosis{
		WorkflowName: "test-pipeline",
		Namespace:    "default",
		Phase:        "Failed",
		Message:      "Step failed",
		StartedAt:    startTime,
		FinishedAt:   endTime,
		Duration:     5*time.Minute + 30*time.Second,
		Parameters: []parameterInfo{
			{Name: "input", Value: "test-value"},
		},
		RootCauses: []failedNodeInfo{
			{
				ID:           "node1",
				Name:         "test-pipeline.process",
				DisplayName:  "process",
				TemplateName: "process-template",
				Phase:        "Failed",
				Message:      "Error: exit status 1",
				ExitCode:     "1",
				StartedAt:    startTime,
				FinishedAt:   endTime,
				IsRootCause:  true,
				Logs:         "Processing failed\nError: invalid input",
				ErrorPattern: &errorPattern{
					Pattern:    "Exit code 1",
					Suggestion: "Check the logs for details",
				},
			},
		},
		FailedNodes: []failedNodeInfo{
			{
				ID:          "node1",
				Name:        "test-pipeline.process",
				DisplayName: "process",
				Phase:       "Failed",
				IsRootCause: true,
			},
			{
				ID:          "node2",
				Name:        "test-pipeline.cleanup",
				DisplayName: "cleanup",
				Phase:       "Failed",
				IsRootCause: false,
			},
		},
	}

	result := buildPromptText(d)

	// Verify header is present
	assert.Contains(t, result, "You are diagnosing a failed Argo Workflow")
	assert.Contains(t, result, "What failed and why")

	// Verify workflow info
	assert.Contains(t, result, "## Workflow: test-pipeline")
	assert.Contains(t, result, "Namespace: default")
	assert.Contains(t, result, "Status: Failed")
	assert.Contains(t, result, "Duration: 5m30s")

	// Verify parameters
	assert.Contains(t, result, "### Workflow Parameters")
	assert.Contains(t, result, "- input: test-value")

	// Verify root cause section
	assert.Contains(t, result, "## Root Cause Node(s)")
	assert.Contains(t, result, "### Node: process (ROOT CAUSE)")
	assert.Contains(t, result, "Template: process-template")
	assert.Contains(t, result, "Phase: Failed")
	assert.Contains(t, result, "Exit Code: 1")
	assert.Contains(t, result, "Logs (last lines):")
	assert.Contains(t, result, "Processing failed")

	// Verify other failed nodes section
	assert.Contains(t, result, "## Other Failed Nodes (Cascading Failures)")
	assert.Contains(t, result, "### Node: cleanup")

	// Verify error patterns section
	assert.Contains(t, result, "## Detected Error Patterns")
	assert.Contains(t, result, "**Pattern**: Exit code 1")
	assert.Contains(t, result, "**Suggestion**: Check the logs for details")

	// Verify conclusion
	assert.Contains(t, result, "Based on this information, explain the failure and suggest fixes.")
}

func TestDescribeValueFrom(t *testing.T) {
	tests := []struct {
		name     string
		vf       *wfv1.ValueFrom
		expected string
	}{
		{
			name:     "nil value from",
			vf:       nil,
			expected: "",
		},
		{
			name:     "path",
			vf:       &wfv1.ValueFrom{Path: "/tmp/output.txt"},
			expected: "path: /tmp/output.txt",
		},
		{
			name:     "jsonpath",
			vf:       &wfv1.ValueFrom{JSONPath: "{.status.phase}"},
			expected: "jsonpath: {.status.phase}",
		},
		{
			name:     "jq filter",
			vf:       &wfv1.ValueFrom{JQFilter: ".data | keys"},
			expected: "jq: .data | keys",
		},
		{
			name:     "parameter",
			vf:       &wfv1.ValueFrom{Parameter: "steps.process.outputs.result"},
			expected: "parameter: steps.process.outputs.result",
		},
		{
			name:     "expression",
			vf:       &wfv1.ValueFrom{Expression: "inputs.parameters.count > 5"},
			expected: "expression: inputs.parameters.count > 5",
		},
		{
			name:     "default value",
			vf:       &wfv1.ValueFrom{Default: wfv1.AnyStringPtr("default-value")},
			expected: "default: default-value",
		},
		{
			name: "multiple fields",
			vf: &wfv1.ValueFrom{
				Path:    "/tmp/out.txt",
				Default: wfv1.AnyStringPtr("fallback"),
			},
			expected: "path: /tmp/out.txt, default: fallback",
		},
		{
			name:     "empty value from",
			vf:       &wfv1.ValueFrom{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := describeValueFrom(tt.vf)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDescribeArtifactSource(t *testing.T) {
	tests := []struct {
		name     string
		artifact *wfv1.Artifact
		expected string
	}{
		{
			name:     "from reference",
			artifact: &wfv1.Artifact{From: "{{steps.download.outputs.artifacts.data}}"},
			expected: "from: {{steps.download.outputs.artifacts.data}}",
		},
		{
			name: "s3 source",
			artifact: &wfv1.Artifact{
				ArtifactLocation: wfv1.ArtifactLocation{
					S3: &wfv1.S3Artifact{
						S3Bucket: wfv1.S3Bucket{Bucket: "my-bucket"},
						Key:      "data/input.csv",
					},
				},
			},
			expected: "s3: my-bucket/data/input.csv",
		},
		{
			name: "gcs source",
			artifact: &wfv1.Artifact{
				ArtifactLocation: wfv1.ArtifactLocation{
					GCS: &wfv1.GCSArtifact{
						GCSBucket: wfv1.GCSBucket{Bucket: "gcs-bucket"},
						Key:       "path/to/file",
					},
				},
			},
			expected: "gcs: gcs-bucket/path/to/file",
		},
		{
			name: "http source",
			artifact: &wfv1.Artifact{
				ArtifactLocation: wfv1.ArtifactLocation{
					HTTP: &wfv1.HTTPArtifact{URL: "https://example.com/data.zip"},
				},
			},
			expected: "http: https://example.com/data.zip",
		},
		{
			name: "git source",
			artifact: &wfv1.Artifact{
				ArtifactLocation: wfv1.ArtifactLocation{
					Git: &wfv1.GitArtifact{Repo: "https://github.com/org/repo"},
				},
			},
			expected: "git: https://github.com/org/repo",
		},
		{
			name:     "no source",
			artifact: &wfv1.Artifact{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := describeArtifactSource(tt.artifact)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	//nolint:govet // Test struct field order optimized for readability
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			expected: "45s",
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
			name:     "zero",
			duration: 0,
			expected: "0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateString(t *testing.T) {
	//nolint:govet // Test struct field order optimized for readability
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than max",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "string equal to max",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "string longer than max",
			input:    "hello world",
			maxLen:   5,
			expected: "hello...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTruncateStringBytes(t *testing.T) {
	tests := []struct { //nolint:govet // Field order optimized for readability
		name     string
		input    string
		maxBytes int
		expected string
	}{
		{
			name:     "string shorter than max",
			input:    "hello",
			maxBytes: 10,
			expected: "hello",
		},
		{
			name:     "string equal to max",
			input:    "hello",
			maxBytes: 5,
			expected: "hello",
		},
		{
			name:     "string longer than max ASCII",
			input:    "hello world",
			maxBytes: 5,
			expected: "hello",
		},
		{
			name:     "empty string",
			input:    "",
			maxBytes: 10,
			expected: "",
		},
		{
			name:     "truncate at UTF-8 boundary - 2 byte char",
			input:    "hello é world", // é is 2 bytes (0xC3 0xA9)
			maxBytes: 6,               // "hello " is 6 bytes, next is start of é
			expected: "hello ",
		},
		{
			name:     "truncate mid UTF-8 char backs up - 2 byte",
			input:    "helloé", // é is 2 bytes
			maxBytes: 6,        // would cut é in half (5 + first byte of é)
			expected: "hello",  // backs up to valid boundary
		},
		{
			name:     "truncate mid UTF-8 char backs up - 3 byte",
			input:    "hello世界", // 世 is 3 bytes (0xE4 0xB8 0x96)
			maxBytes: 6,         // would cut 世 in half
			expected: "hello",   // backs up to valid boundary
		},
		{
			name:     "truncate preserves complete multi-byte chars",
			input:    "日本語", // each is 3 bytes
			maxBytes: 6,     // exactly 2 chars
			expected: "日本",
		},
	}

	for _, tt := range tests { //nolint:govet // Field order optimized for readability
		t.Run(tt.name, func(t *testing.T) {
			result := truncateStringBytes(tt.input, tt.maxBytes)
			assert.Equal(t, tt.expected, result)
			// Verify result is valid UTF-8
			assert.True(t, utf8.ValidString(result), "result should be valid UTF-8")
		})
	}
}

func TestFilterOutRootCauses(t *testing.T) {
	failed := []failedNodeInfo{
		{ID: "node1"},
		{ID: "node2"},
		{ID: "node3"},
	}
	rootCauses := []failedNodeInfo{
		{ID: "node1"},
	}

	result := filterOutRootCauses(failed, rootCauses)
	assert.Len(t, result, 2)
	assert.Equal(t, "node2", result[0].ID)
	assert.Equal(t, "node3", result[1].ID)
}

func TestHasErrorPatterns(t *testing.T) {
	tests := []struct {
		name       string
		rootCauses []failedNodeInfo
		expected   bool
	}{
		{
			name: "has error pattern",
			rootCauses: []failedNodeInfo{
				{ErrorPattern: &errorPattern{Pattern: "OOM"}},
			},
			expected: true,
		},
		{
			name: "no error pattern",
			rootCauses: []failedNodeInfo{
				{ErrorPattern: nil},
			},
			expected: false,
		},
		{
			name:       "empty slice",
			rootCauses: []failedNodeInfo{},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasErrorPatterns(tt.rootCauses)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// strPtr is a helper to create string pointers.
func strPtr(s string) *string {
	return &s
}
