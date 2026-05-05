package tools

import (
	"errors"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestGetCronWorkflowTool(t *testing.T) {
	tool := GetCronWorkflowTool()

	assert.Equal(t, "get_cron_workflow", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "CronWorkflow")
}

func TestGetCronWorkflowInput(t *testing.T) {
	// Test with required name only
	input := GetCronWorkflowInput{
		Name: "test-cron",
	}
	assert.Equal(t, "test-cron", input.Name)
	assert.Empty(t, input.Namespace)

	// Test with all values
	input2 := GetCronWorkflowInput{
		Name:      "test-cron-2",
		Namespace: "test-namespace",
	}
	assert.Equal(t, "test-cron-2", input2.Name)
	assert.Equal(t, "test-namespace", input2.Namespace)
}

func TestActiveWorkflowRef(t *testing.T) {
	ref := ActiveWorkflowRef{
		Name:      "active-workflow",
		Namespace: "default",
		UID:       "abc-123",
	}

	assert.Equal(t, "active-workflow", ref.Name)
	assert.Equal(t, "default", ref.Namespace)
	assert.Equal(t, "abc-123", ref.UID)
}

func TestGetCronWorkflowOutput(t *testing.T) {
	successfulLimit := int32(3)
	failedLimit := int32(1)
	deadlineSeconds := int64(60)

	output := GetCronWorkflowOutput{
		Name:                       "test-cron",
		Namespace:                  "default",
		Schedules:                  []string{"0 * * * *"},
		Timezone:                   "UTC",
		ConcurrencyPolicy:          "Forbid",
		Suspended:                  false,
		CreatedAt:                  "2025-01-01T00:00:00Z",
		LastScheduledTime:          "2025-01-01T12:00:00Z",
		Entrypoint:                 "main",
		SuccessfulJobsHistoryLimit: &successfulLimit,
		FailedJobsHistoryLimit:     &failedLimit,
		StartingDeadlineSeconds:    &deadlineSeconds,
		SucceededCount:             5,
		FailedCount:                2,
		ActiveWorkflows: []ActiveWorkflowRef{
			{Name: "workflow-1", Namespace: "default"},
		},
	}

	assert.Equal(t, "test-cron", output.Name)
	assert.Equal(t, "default", output.Namespace)
	assert.Equal(t, []string{"0 * * * *"}, output.Schedules)
	assert.Equal(t, "UTC", output.Timezone)
	assert.Equal(t, "Forbid", output.ConcurrencyPolicy)
	assert.False(t, output.Suspended)
	assert.NotEmpty(t, output.CreatedAt)
	assert.NotEmpty(t, output.LastScheduledTime)
	assert.Equal(t, "main", output.Entrypoint)
	assert.Equal(t, int32(3), *output.SuccessfulJobsHistoryLimit)
	assert.Equal(t, int32(1), *output.FailedJobsHistoryLimit)
	assert.Equal(t, int64(60), *output.StartingDeadlineSeconds)
	assert.Equal(t, int64(5), output.SucceededCount)
	assert.Equal(t, int64(2), output.FailedCount)
	assert.Len(t, output.ActiveWorkflows, 1)
}

func TestGetCronWorkflowHandler(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	lastScheduledTime := metav1.Time{Time: testTime.Add(time.Hour)}
	successfulLimit := int32(3)
	failedLimit := int32(1)

	tests := []struct {
		setupMock func(*mocks.MockCronWorkflowServiceClient)
		validate  func(*testing.T, *GetCronWorkflowOutput, *mcp.CallToolResult)
		name      string
		input     GetCronWorkflowInput
		wantErr   bool
	}{
		{
			name: "success - get cron workflow with all details",
			input: GetCronWorkflowInput{
				Name:      "test-cron",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("GetCronWorkflow", mock.Anything, mock.MatchedBy(func(req *cronworkflow.GetCronWorkflowRequest) bool {
					return req.Name == "test-cron" && req.Namespace == "default"
				})).Return(
					&wfv1.CronWorkflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "test-cron",
							Namespace:         "default",
							CreationTimestamp: metav1.Time{Time: testTime},
							Labels:            map[string]string{"app": "test"},
							Annotations:       map[string]string{"description": "test cron"},
						},
						Spec: wfv1.CronWorkflowSpec{
							Schedules:                  []string{"0 * * * *"},
							Timezone:                   "UTC",
							ConcurrencyPolicy:          wfv1.ForbidConcurrent,
							Suspend:                    false,
							SuccessfulJobsHistoryLimit: &successfulLimit,
							FailedJobsHistoryLimit:     &failedLimit,
							WorkflowSpec: wfv1.WorkflowSpec{
								Entrypoint: "main",
							},
						},
						Status: wfv1.CronWorkflowStatus{
							LastScheduledTime: &lastScheduledTime,
							Succeeded:         5,
							Failed:            2,
							Active: []corev1.ObjectReference{
								{
									Name:      "active-workflow-1",
									Namespace: "default",
									UID:       types.UID("abc-123"),
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetCronWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "test-cron", output.Name)
				assert.Equal(t, "default", output.Namespace)
				assert.Equal(t, []string{"0 * * * *"}, output.Schedules)
				assert.Equal(t, "UTC", output.Timezone)
				assert.Equal(t, "Forbid", output.ConcurrencyPolicy)
				assert.False(t, output.Suspended)
				assert.Equal(t, "main", output.Entrypoint)
				assert.NotEmpty(t, output.CreatedAt)
				assert.NotEmpty(t, output.LastScheduledTime)
				assert.Equal(t, map[string]string{"app": "test"}, output.Labels)
				assert.Equal(t, map[string]string{"description": "test cron"}, output.Annotations)
				assert.Equal(t, int64(5), output.SucceededCount)
				assert.Equal(t, int64(2), output.FailedCount)
				assert.Len(t, output.ActiveWorkflows, 1)
				assert.Equal(t, "active-workflow-1", output.ActiveWorkflows[0].Name)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "test-cron")
				assert.Contains(t, text.Text, "0 * * * *")
				assert.Contains(t, text.Text, "Active workflows: 1")
			},
		},
		{
			name: "success - suspended cron workflow",
			input: GetCronWorkflowInput{
				Name:      "suspended-cron",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("GetCronWorkflow", mock.Anything, mock.Anything).Return(
					&wfv1.CronWorkflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "suspended-cron",
							Namespace:         "default",
							CreationTimestamp: metav1.Time{Time: testTime},
						},
						Spec: wfv1.CronWorkflowSpec{
							Schedules: []string{"*/5 * * * *"},
							Suspend:   true,
							WorkflowSpec: wfv1.WorkflowSpec{
								Entrypoint: "main",
							},
						},
						Status: wfv1.CronWorkflowStatus{},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetCronWorkflowOutput, result *mcp.CallToolResult) {
				assert.Equal(t, "suspended-cron", output.Name)
				assert.True(t, output.Suspended)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "Suspended")
			},
		},
		{
			name: "success - uses default namespace",
			input: GetCronWorkflowInput{
				Name: "my-cron",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("GetCronWorkflow", mock.Anything, mock.MatchedBy(func(req *cronworkflow.GetCronWorkflowRequest) bool {
					return req.Namespace == "argo" // default namespace from mock
				})).Return(
					&wfv1.CronWorkflow{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "my-cron",
							Namespace:         "argo",
							CreationTimestamp: metav1.Time{Time: testTime},
						},
						Spec: wfv1.CronWorkflowSpec{
							Schedules: []string{"0 0 * * *"},
							WorkflowSpec: wfv1.WorkflowSpec{
								Entrypoint: "main",
							},
						},
						Status: wfv1.CronWorkflowStatus{},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *GetCronWorkflowOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, "my-cron", output.Name)
				assert.Equal(t, "argo", output.Namespace)
			},
		},
		{
			name: "error - name is required",
			input: GetCronWorkflowInput{
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				// No mock setup needed - validation fails before API call
			},
			wantErr: true,
		},
		{
			name: "error - cron workflow not found",
			input: GetCronWorkflowInput{
				Name:      "nonexistent",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("GetCronWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("cronworkflows.argoproj.io \"nonexistent\" not found"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - API failure",
			input: GetCronWorkflowInput{
				Name:      "test-cron",
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("GetCronWorkflow", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("connection refused"),
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client and service
			mockClient := newMockClient(t, "argo", true)
			mockService := newMockCronWorkflowService(t)
			mockClient.SetCronWorkflowService(mockService)

			// Setup mock expectations
			tt.setupMock(mockService)

			// Create handler and call it
			handler := GetCronWorkflowHandler(mockClient)
			ctx := t.Context()
			req := &mcp.CallToolRequest{}

			result, output, err := handler(ctx, req, tt.input)

			// Validate results
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, output)
			tt.validate(t, output, result)

			// Verify mock expectations
			mockService.AssertExpectations(t)
		})
	}
}
