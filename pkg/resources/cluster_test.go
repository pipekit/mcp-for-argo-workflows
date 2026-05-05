package resources

import (
	"errors"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo/mocks"
)

func TestAllClusterResources(t *testing.T) {
	resources := AllClusterResources()

	// Verify we have all expected resources
	assert.Len(t, resources, 3)

	// Verify each resource has required fields
	for _, r := range resources {
		assert.NotEmpty(t, r.URI, "URI should not be empty")
		assert.NotEmpty(t, r.Name, "Name should not be empty")
		assert.NotEmpty(t, r.Title, "Title should not be empty")
		assert.NotEmpty(t, r.Description, "Description should not be empty")
		assert.NotEmpty(t, r.MIMEType, "MIMEType should not be empty")
	}

	// Check specific resources exist
	uris := make([]string, 0, len(resources))
	for _, r := range resources {
		uris = append(uris, r.URI)
	}

	assert.Contains(t, uris, "argo://cluster/workflow-templates")
	assert.Contains(t, uris, "argo://cluster/cluster-workflow-templates")
	assert.Contains(t, uris, "argo://cluster/cron-workflows")
}

func TestAllClusterResourceTemplates(t *testing.T) {
	templates := AllClusterResourceTemplates()

	// Verify we have all expected templates
	assert.Len(t, templates, 2)

	// Verify each template has required fields
	for _, tmpl := range templates {
		assert.NotEmpty(t, tmpl.URITemplate, "URITemplate should not be empty")
		assert.NotEmpty(t, tmpl.Name, "Name should not be empty")
		assert.NotEmpty(t, tmpl.Title, "Title should not be empty")
		assert.NotEmpty(t, tmpl.Description, "Description should not be empty")
		assert.NotEmpty(t, tmpl.MIMEType, "MIMEType should not be empty")
	}

	// Check specific templates exist
	uriTemplates := make([]string, 0, len(templates))
	for _, tmpl := range templates {
		uriTemplates = append(uriTemplates, tmpl.URITemplate)
	}

	assert.Contains(t, uriTemplates, "argo://cluster/workflow-templates/{namespace}/{name}")
	assert.Contains(t, uriTemplates, "argo://cluster/cluster-workflow-templates/{name}")
}

func TestRegisterClusterResources(t *testing.T) {
	mockClient := mocks.NewMockClient("default", true)

	implementation := &mcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}
	server := mcp.NewServer(implementation, nil)

	// Should not panic
	RegisterClusterResources(server, mockClient)
}

func TestListWorkflowTemplatesContent(t *testing.T) {
	tests := []struct {
		setupMock  func(*mocks.MockWorkflowTemplateServiceClient)
		assertions func(*testing.T, string)
		name       string
		namespace  string
		wantErr    bool
	}{
		{
			name:      "success - list templates",
			namespace: "default",
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("ListWorkflowTemplates", mock.Anything, mock.MatchedBy(func(req *workflowtemplate.WorkflowTemplateListRequest) bool {
					return req.Namespace == "default"
				})).Return(
					&wfv1.WorkflowTemplateList{
						Items: []wfv1.WorkflowTemplate{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "template-1",
									Namespace:         "default",
									CreationTimestamp: metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
									Annotations: map[string]string{
										"workflows.argoproj.io/description": "Test template 1",
									},
								},
								Spec: wfv1.WorkflowSpec{
									Arguments: wfv1.Arguments{
										Parameters: []wfv1.Parameter{
											{Name: "param1"},
											{Name: "param2"},
										},
									},
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "template-2",
									Namespace: "default",
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			assertions: func(t *testing.T, content string) {
				assert.Contains(t, content, "template-1")
				assert.Contains(t, content, "template-2")
				assert.Contains(t, content, "default")
				assert.Contains(t, content, "param1")
				assert.Contains(t, content, "param2")
				assert.Contains(t, content, "Test template 1")
			},
		},
		{
			name:      "success - empty list",
			namespace: "empty-ns",
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("ListWorkflowTemplates", mock.Anything, mock.Anything).Return(
					&wfv1.WorkflowTemplateList{Items: []wfv1.WorkflowTemplate{}},
					nil,
				)
			},
			wantErr: false,
			assertions: func(t *testing.T, content string) {
				assert.Contains(t, content, `"count": 0`)
				assert.Contains(t, content, `"templates": []`)
			},
		},
		{
			name:      "error - service failure",
			namespace: "default",
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("ListWorkflowTemplates", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("connection refused"),
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockClient("default", true)
			mockService := &mocks.MockWorkflowTemplateServiceClient{}
			mockService.Test(t)
			mockClient.SetWorkflowTemplateService(mockService)

			tt.setupMock(mockService)

			content, err := listWorkflowTemplatesContent(t.Context(), mockClient, tt.namespace)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, content)

			if tt.assertions != nil {
				tt.assertions(t, content)
			}
		})
	}
}

func TestListClusterWorkflowTemplatesContent(t *testing.T) {
	tests := []struct {
		setupMock  func(*mocks.MockClusterWorkflowTemplateServiceClient)
		assertions func(*testing.T, string)
		name       string
		wantErr    bool
	}{
		{
			name: "success - list cluster templates",
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("ListClusterWorkflowTemplates", mock.Anything, mock.Anything).Return(
					&wfv1.ClusterWorkflowTemplateList{
						Items: []wfv1.ClusterWorkflowTemplate{
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:              "cluster-template-1",
									CreationTimestamp: metav1.Time{Time: time.Now().Add(-1 * time.Hour)},
								},
								Spec: wfv1.WorkflowSpec{
									Arguments: wfv1.Arguments{
										Parameters: []wfv1.Parameter{
											{Name: "global-param"},
										},
									},
								},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			assertions: func(t *testing.T, content string) {
				assert.Contains(t, content, "cluster-template-1")
				assert.Contains(t, content, "global-param")
			},
		},
		{
			name: "error - service failure",
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("ListClusterWorkflowTemplates", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("forbidden"),
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockClient("default", true)
			mockService := &mocks.MockClusterWorkflowTemplateServiceClient{}
			mockService.Test(t)
			mockClient.SetClusterWorkflowTemplateService(mockService)

			tt.setupMock(mockService)

			content, err := listClusterWorkflowTemplatesContent(t.Context(), mockClient)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, content)

			if tt.assertions != nil {
				tt.assertions(t, content)
			}
		})
	}
}

func TestListCronWorkflowsContent(t *testing.T) {
	tests := []struct {
		setupMock  func(*mocks.MockCronWorkflowServiceClient)
		assertions func(*testing.T, string)
		name       string
		namespace  string
		wantErr    bool
	}{
		{
			name:      "success - list cron workflows",
			namespace: "default",
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
									CreationTimestamp: metav1.Time{Time: time.Now().Add(-24 * time.Hour)},
								},
								Spec: wfv1.CronWorkflowSpec{
									Schedules: []string{"0 * * * *"},
									Suspend:   false,
								},
								Status: wfv1.CronWorkflowStatus{
									Active:    []corev1.ObjectReference{},
									Succeeded: 10,
									Failed:    2,
								},
							},
							{
								ObjectMeta: metav1.ObjectMeta{
									Name:      "cron-2",
									Namespace: "default",
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
			assertions: func(t *testing.T, content string) {
				assert.Contains(t, content, "cron-1")
				assert.Contains(t, content, "cron-2")
				assert.Contains(t, content, "0 * * * *")
				assert.Contains(t, content, "*/5 * * * *")
				assert.Contains(t, content, `"suspended": true`)
			},
		},
		{
			name:      "error - service failure",
			namespace: "default",
			setupMock: func(m *mocks.MockCronWorkflowServiceClient) {
				m.On("ListCronWorkflows", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("connection refused"),
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockClient("default", true)
			mockService := &mocks.MockCronWorkflowServiceClient{}
			mockService.Test(t)
			mockClient.SetCronWorkflowService(mockService)

			tt.setupMock(mockService)

			content, err := listCronWorkflowsContent(t.Context(), mockClient, tt.namespace)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, content)

			if tt.assertions != nil {
				tt.assertions(t, content)
			}
		})
	}
}

func TestGetWorkflowTemplateContent(t *testing.T) {
	tests := []struct {
		setupMock  func(*mocks.MockWorkflowTemplateServiceClient)
		assertions func(*testing.T, string)
		name       string
		namespace  string
		tmplName   string
		wantErr    bool
	}{
		{
			name:      "success - get template",
			namespace: "default",
			tmplName:  "my-template",
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("GetWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *workflowtemplate.WorkflowTemplateGetRequest) bool {
					return req.Namespace == "default" && req.Name == "my-template"
				})).Return(
					&wfv1.WorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-template",
							Namespace: "default",
							Labels: map[string]string{
								"app": "test",
							},
						},
						Spec: wfv1.WorkflowSpec{
							Entrypoint: "main",
							Arguments: wfv1.Arguments{
								Parameters: []wfv1.Parameter{
									{Name: "input", Value: wfv1.AnyStringPtr("default-value")},
								},
							},
							Templates: []wfv1.Template{
								{Name: "main"},
								{Name: "worker"},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			assertions: func(t *testing.T, content string) {
				assert.Contains(t, content, "my-template")
				assert.Contains(t, content, "default")
				assert.Contains(t, content, "main")
				assert.Contains(t, content, "worker")
				assert.Contains(t, content, "input")
				assert.Contains(t, content, "default-value")
			},
		},
		{
			name:      "error - not found",
			namespace: "default",
			tmplName:  "nonexistent",
			setupMock: func(m *mocks.MockWorkflowTemplateServiceClient) {
				m.On("GetWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("not found"),
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockClient("default", true)
			mockService := &mocks.MockWorkflowTemplateServiceClient{}
			mockService.Test(t)
			mockClient.SetWorkflowTemplateService(mockService)

			tt.setupMock(mockService)

			content, err := getWorkflowTemplateContent(t.Context(), mockClient, tt.namespace, tt.tmplName)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, content)

			if tt.assertions != nil {
				tt.assertions(t, content)
			}
		})
	}
}

func TestGetClusterWorkflowTemplateContent(t *testing.T) {
	tests := []struct {
		setupMock  func(*mocks.MockClusterWorkflowTemplateServiceClient)
		assertions func(*testing.T, string)
		name       string
		tmplName   string
		wantErr    bool
	}{
		{
			name:     "success - get cluster template",
			tmplName: "cluster-tmpl",
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("GetClusterWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *clusterworkflowtemplate.ClusterWorkflowTemplateGetRequest) bool {
					return req.Name == "cluster-tmpl"
				})).Return(
					&wfv1.ClusterWorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster-tmpl",
						},
						Spec: wfv1.WorkflowSpec{
							Entrypoint: "start",
							Templates: []wfv1.Template{
								{Name: "start"},
							},
						},
					},
					nil,
				)
			},
			wantErr: false,
			assertions: func(t *testing.T, content string) {
				assert.Contains(t, content, "cluster-tmpl")
				assert.Contains(t, content, "start")
			},
		},
		{
			name:     "error - not found",
			tmplName: "nonexistent",
			setupMock: func(m *mocks.MockClusterWorkflowTemplateServiceClient) {
				m.On("GetClusterWorkflowTemplate", mock.Anything, mock.Anything).Return(
					nil,
					errors.New("not found"),
				)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockClient("default", true)
			mockService := &mocks.MockClusterWorkflowTemplateServiceClient{}
			mockService.Test(t)
			mockClient.SetClusterWorkflowTemplateService(mockService)

			tt.setupMock(mockService)

			content, err := getClusterWorkflowTemplateContent(t.Context(), mockClient, tt.tmplName)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotEmpty(t, content)

			if tt.assertions != nil {
				tt.assertions(t, content)
			}
		})
	}
}

func TestClusterResourceHandler(t *testing.T) {
	tests := []struct {
		setupMock  func(*mocks.MockClient)
		assertions func(*testing.T, *mcp.ReadResourceResult)
		name       string
		uri        string
		wantErr    bool
	}{
		{
			name: "workflow-templates list",
			uri:  "argo://cluster/workflow-templates",
			setupMock: func(m *mocks.MockClient) {
				mockService := &mocks.MockWorkflowTemplateServiceClient{}
				mockService.On("ListWorkflowTemplates", mock.Anything, mock.Anything).Return(
					&wfv1.WorkflowTemplateList{Items: []wfv1.WorkflowTemplate{}},
					nil,
				)
				m.SetWorkflowTemplateService(mockService)
			},
			wantErr: false,
			assertions: func(t *testing.T, result *mcp.ReadResourceResult) {
				require.Len(t, result.Contents, 1)
				assert.Contains(t, result.Contents[0].Text, "templates")
			},
		},
		{
			name: "workflow-templates with namespace query",
			uri:  "argo://cluster/workflow-templates?namespace=custom-ns",
			setupMock: func(m *mocks.MockClient) {
				mockService := &mocks.MockWorkflowTemplateServiceClient{}
				mockService.On("ListWorkflowTemplates", mock.Anything, mock.MatchedBy(func(req *workflowtemplate.WorkflowTemplateListRequest) bool {
					return req.Namespace == "custom-ns"
				})).Return(
					&wfv1.WorkflowTemplateList{Items: []wfv1.WorkflowTemplate{}},
					nil,
				)
				m.SetWorkflowTemplateService(mockService)
			},
			wantErr: false,
		},
		{
			name: "cluster-workflow-templates list",
			uri:  "argo://cluster/cluster-workflow-templates",
			setupMock: func(m *mocks.MockClient) {
				mockService := &mocks.MockClusterWorkflowTemplateServiceClient{}
				mockService.On("ListClusterWorkflowTemplates", mock.Anything, mock.Anything).Return(
					&wfv1.ClusterWorkflowTemplateList{Items: []wfv1.ClusterWorkflowTemplate{}},
					nil,
				)
				m.SetClusterWorkflowTemplateService(mockService)
			},
			wantErr: false,
		},
		{
			name: "cron-workflows list",
			uri:  "argo://cluster/cron-workflows",
			setupMock: func(m *mocks.MockClient) {
				mockService := &mocks.MockCronWorkflowServiceClient{}
				mockService.On("ListCronWorkflows", mock.Anything, mock.Anything).Return(
					&wfv1.CronWorkflowList{Items: []wfv1.CronWorkflow{}},
					nil,
				)
				m.SetCronWorkflowService(mockService)
			},
			wantErr: false,
		},
		{
			name: "unknown resource",
			uri:  "argo://cluster/unknown",
			setupMock: func(_ *mocks.MockClient) {
				// No setup needed
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockClient("default", true)
			tt.setupMock(mockClient)

			handler := clusterResourceHandler("", mockClient)

			req := &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: tt.uri,
				},
			}

			result, err := handler(t.Context(), req)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			if tt.assertions != nil {
				tt.assertions(t, result)
			}
		})
	}
}

func TestClusterResourceTemplateHandler(t *testing.T) {
	tests := []struct {
		setupMock  func(*mocks.MockClient)
		assertions func(*testing.T, *mcp.ReadResourceResult)
		name       string
		uri        string
		wantErr    bool
	}{
		{
			name: "get workflow template",
			uri:  "argo://cluster/workflow-templates/default/my-template",
			setupMock: func(m *mocks.MockClient) {
				mockService := &mocks.MockWorkflowTemplateServiceClient{}
				mockService.On("GetWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *workflowtemplate.WorkflowTemplateGetRequest) bool {
					return req.Namespace == "default" && req.Name == "my-template"
				})).Return(
					&wfv1.WorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "my-template",
							Namespace: "default",
						},
						Spec: wfv1.WorkflowSpec{
							Entrypoint: "main",
						},
					},
					nil,
				)
				m.SetWorkflowTemplateService(mockService)
			},
			wantErr: false,
			assertions: func(t *testing.T, result *mcp.ReadResourceResult) {
				require.Len(t, result.Contents, 1)
				assert.Contains(t, result.Contents[0].Text, "my-template")
			},
		},
		{
			name: "get cluster workflow template",
			uri:  "argo://cluster/cluster-workflow-templates/cluster-tmpl",
			setupMock: func(m *mocks.MockClient) {
				mockService := &mocks.MockClusterWorkflowTemplateServiceClient{}
				mockService.On("GetClusterWorkflowTemplate", mock.Anything, mock.MatchedBy(func(req *clusterworkflowtemplate.ClusterWorkflowTemplateGetRequest) bool {
					return req.Name == "cluster-tmpl"
				})).Return(
					&wfv1.ClusterWorkflowTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster-tmpl",
						},
						Spec: wfv1.WorkflowSpec{
							Entrypoint: "start",
						},
					},
					nil,
				)
				m.SetClusterWorkflowTemplateService(mockService)
			},
			wantErr: false,
			assertions: func(t *testing.T, result *mcp.ReadResourceResult) {
				require.Len(t, result.Contents, 1)
				assert.Contains(t, result.Contents[0].Text, "cluster-tmpl")
			},
		},
		{
			name: "invalid workflow template URI format - missing name",
			uri:  "argo://cluster/workflow-templates/default/",
			setupMock: func(_ *mocks.MockClient) {
				// No setup needed - should fail before calling service
			},
			wantErr: true,
		},
		{
			name: "invalid workflow template URI format - only namespace",
			uri:  "argo://cluster/workflow-templates/default",
			setupMock: func(_ *mocks.MockClient) {
				// No setup needed - should fail before calling service
			},
			wantErr: true,
		},
		{
			name: "unknown template URI",
			uri:  "argo://cluster/unknown/test",
			setupMock: func(_ *mocks.MockClient) {
				// No setup needed
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := mocks.NewMockClient("default", true)
			tt.setupMock(mockClient)

			handler := clusterResourceTemplateHandler("", mockClient)

			req := &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: tt.uri,
				},
			}

			result, err := handler(t.Context(), req)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			if tt.assertions != nil {
				tt.assertions(t, result)
			}
		})
	}
}
