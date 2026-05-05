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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestListCronWorkflowsTool(t *testing.T) {
	tool := ListCronWorkflowsTool()

	assert.Equal(t, "list_cron_workflows", tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.Contains(t, tool.Description, "CronWorkflows")
}

func TestListCronWorkflowsInput(t *testing.T) {
	// Test default values
	input := ListCronWorkflowsInput{}
	assert.Empty(t, input.Namespace)
	assert.Empty(t, input.Labels)

	// Test with values
	input2 := ListCronWorkflowsInput{
		Namespace: "test-namespace",
		Labels:    "app=test",
	}
	assert.Equal(t, "test-namespace", input2.Namespace)
	assert.Equal(t, "app=test", input2.Labels)
}

func TestCronWorkflowSummary(t *testing.T) {
	summary := CronWorkflowSummary{
		Name:              "test-cron",
		Namespace:         "default",
		Schedules:         []string{"0 * * * *"},
		Suspended:         false,
		LastScheduledTime: "2025-01-01T12:00:00Z",
		CreatedAt:         "2025-01-01T00:00:00Z",
	}

	assert.Equal(t, "test-cron", summary.Name)
	assert.Equal(t, "default", summary.Namespace)
	assert.Equal(t, []string{"0 * * * *"}, summary.Schedules)
	assert.False(t, summary.Suspended)
	assert.NotEmpty(t, summary.LastScheduledTime)
	assert.NotEmpty(t, summary.CreatedAt)
}

func TestListCronWorkflowsOutput(t *testing.T) {
	output := ListCronWorkflowsOutput{
		CronWorkflows: []CronWorkflowSummary{
			{Name: "cron-1", Namespace: "default", Schedules: []string{"0 * * * *"}},
			{Name: "cron-2", Namespace: "default", Schedules: []string{"*/5 * * * *"}},
		},
		Total: 2,
	}

	assert.Len(t, output.CronWorkflows, 2)
	assert.Equal(t, 2, output.Total)
	assert.Equal(t, "cron-1", output.CronWorkflows[0].Name)
	assert.Equal(t, "cron-2", output.CronWorkflows[1].Name)
}

// newMockCronWorkflowService creates a new mock cron workflow service client.
func newMockCronWorkflowService(t *testing.T) *mocks.MockCronWorkflowServiceClient {
	t.Helper()
	m := &mocks.MockCronWorkflowServiceClient{}
	m.Test(t)
	return m
}

func TestListCronWorkflowsHandler(t *testing.T) {
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	lastScheduledTime := metav1.Time{Time: testTime.Add(time.Hour)}

	tests := []struct {
		setupMock func(*mocks.MockCronWorkflowServiceClient)
		validate  func(*testing.T, *ListCronWorkflowsOutput, *mcp.CallToolResult)
		name      string
		input     ListCronWorkflowsInput
		wantErr   bool
	}{
		{
			name: "success - list cron workflows in namespace",
			input: ListCronWorkflowsInput{
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("ListCronWorkflows", mock.Anything, mock.MatchedBy(func(req *cronworkflow.ListCronWorkflowsRequest) bool {
					return req.Namespace == "default"
				})).Return(
					&wfv1.CronWorkflowList{
						Items: []wfv1.CronWorkflow{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "cron-1",
									Namespace:         "default",
									CreationTimestamp: metav1.Time{Time: testTime},
								},
								Spec: wfv1.CronWorkflowSpec{
									Schedules: []string{"0 * * * *"},
									Suspend:   false,
								},
								Status: wfv1.CronWorkflowStatus{
									LastScheduledTime: &lastScheduledTime,
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "cron-2",
									Namespace:         "default",
									CreationTimestamp: metav1.Time{Time: testTime.Add(time.Hour)},
								},
								Spec: wfv1.CronWorkflowSpec{
									Schedules: []string{"*/5 * * * *"},
									Suspend:   true,
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *ListCronWorkflowsOutput, result *mcp.CallToolResult) {
				assert.Equal(t, 2, output.Total)
				assert.Len(t, output.CronWorkflows, 2)
				assert.Equal(t, "cron-1", output.CronWorkflows[0].Name)
				assert.Equal(t, "default", output.CronWorkflows[0].Namespace)
				assert.Equal(t, []string{"0 * * * *"}, output.CronWorkflows[0].Schedules)
				assert.False(t, output.CronWorkflows[0].Suspended)
				assert.NotEmpty(t, output.CronWorkflows[0].LastScheduledTime)
				assert.Equal(t, "cron-2", output.CronWorkflows[1].Name)
				assert.Equal(t, []string{"*/5 * * * *"}, output.CronWorkflows[1].Schedules)
				assert.True(t, output.CronWorkflows[1].Suspended)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "Found 2 cron workflow(s)")
				assert.Contains(t, text.Text, "default")
			},
		},
		{
			name: "success - list cron workflows with labels",
			input: ListCronWorkflowsInput{
				Namespace: "default",
				Labels:    "app=myapp",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("ListCronWorkflows", mock.Anything, mock.MatchedBy(func(req *cronworkflow.ListCronWorkflowsRequest) bool {
					return req.Namespace == "default" && req.ListOptions != nil && req.ListOptions.LabelSelector == "app=myapp"
				})).Return(
					&wfv1.CronWorkflowList{
						Items: []wfv1.CronWorkflow{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "cron-myapp",
									Namespace:         "default",
									CreationTimestamp: metav1.Time{Time: testTime},
								},
								Spec: wfv1.CronWorkflowSpec{
									Schedules: []string{"0 0 * * *"},
									Suspend:   false,
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *ListCronWorkflowsOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, 1, output.Total)
				assert.Equal(t, "cron-myapp", output.CronWorkflows[0].Name)
			},
		},
		{
			name:  "success - uses default namespace",
			input: ListCronWorkflowsInput{},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("ListCronWorkflows", mock.Anything, mock.MatchedBy(func(req *cronworkflow.ListCronWorkflowsRequest) bool {
					return req.Namespace == "argo" // default namespace from mock
				})).Return(
					&wfv1.CronWorkflowList{
						Items: []wfv1.CronWorkflow{},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *ListCronWorkflowsOutput, result *mcp.CallToolResult) {
				assert.Equal(t, 0, output.Total)
				assert.Empty(t, output.CronWorkflows)
				require.NotNil(t, result)
				require.Len(t, result.Content, 1)
				text, ok := result.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				assert.Contains(t, text.Text, "Found 0 cron workflow(s)")
			},
		},
		{
			name: "success - empty result",
			input: ListCronWorkflowsInput{
				Namespace: "empty-namespace",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("ListCronWorkflows", mock.Anything, mock.Anything).Return(
					&wfv1.CronWorkflowList{
						Items: []wfv1.CronWorkflow{},
					},
					nil,
				)
			},
			wantErr: false,
			validate: func(t *testing.T, output *ListCronWorkflowsOutput, _ *mcp.CallToolResult) {
				assert.Equal(t, 0, output.Total)
				assert.Empty(t, output.CronWorkflows)
			},
		},
		{
			name: "error - API failure",
			input: ListCronWorkflowsInput{
				Namespace: "default",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("ListCronWorkflows", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("connection refused"),
				)
			},
			wantErr: true,
		},
		{
			name: "error - namespace not found",
			input: ListCronWorkflowsInput{
				Namespace: "nonexistent",
			},
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("ListCronWorkflows", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("namespaces \"nonexistent\" not found"),
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

			// Verify mock expectations even on error paths
			defer mockService.AssertExpectations(t)

			// Create handler and call it
			handler := ListCronWorkflowsHandler(mockClient)
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
		})
	}
}
