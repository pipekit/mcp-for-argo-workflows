// Package resources implements MCP resource handlers for Argo Workflows.
package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// ClusterResourceDefinition defines a dynamic MCP resource that queries the cluster.
type ClusterResourceDefinition struct {
	URI         string
	Name        string
	Title       string
	Description string
	MIMEType    string
}

// ClusterResourceTemplateDefinition defines a dynamic MCP resource template with parameters.
type ClusterResourceTemplateDefinition struct {
	URITemplate string
	Name        string
	Title       string
	Description string
	MIMEType    string
}

// WorkflowTemplateSummary represents a summary of a WorkflowTemplate for the resource listing.
type WorkflowTemplateSummary struct {
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace"`
	Description string   `json:"description,omitempty"`
	CreatedAt   string   `json:"createdAt,omitempty"`
	Parameters  []string `json:"parameters,omitempty"`
}

// ClusterWorkflowTemplateSummary represents a summary of a ClusterWorkflowTemplate.
type ClusterWorkflowTemplateSummary struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	CreatedAt   string   `json:"createdAt,omitempty"`
	Parameters  []string `json:"parameters,omitempty"`
}

// CronWorkflowSummary represents a summary of a CronWorkflow.
//
//nolint:govet // Field order optimized for readability over memory alignment
type CronWorkflowSummary struct {
	Name         string   `json:"name"`
	Namespace    string   `json:"namespace"`
	Schedules    []string `json:"schedules"`
	Suspended    bool     `json:"suspended"`
	LastRun      string   `json:"lastRun,omitempty"`
	NextRun      string   `json:"nextRun,omitempty"`
	CreatedAt    string   `json:"createdAt,omitempty"`
	ActiveCount  int      `json:"activeCount"`
	SuccessCount int      `json:"successCount"`
	FailedCount  int      `json:"failedCount"`
}

// buildArgumentsMap extracts arguments from workflow parameters into a map for JSON serialization.
func buildArgumentsMap(params []wfv1.Parameter) map[string]interface{} {
	if params == nil {
		return nil
	}
	arguments := map[string]interface{}{}
	paramsList := make([]map[string]interface{}, 0, len(params))
	for _, p := range params {
		param := map[string]interface{}{
			"name": p.Name,
		}
		if p.Value != nil {
			param["default"] = string(*p.Value)
		}
		if p.Description != nil {
			param["description"] = string(*p.Description)
		}
		if p.Enum != nil {
			param["enum"] = p.Enum
		}
		paramsList = append(paramsList, param)
	}
	arguments["parameters"] = paramsList
	return arguments
}

// AllClusterResources returns all static cluster resource definitions.
func AllClusterResources() []ClusterResourceDefinition {
	return []ClusterResourceDefinition{
		{
			URI:         "argo://cluster/workflow-templates",
			Name:        "cluster-workflow-templates-list",
			Title:       "Workflow Templates List",
			Description: "List all WorkflowTemplates available in the cluster",
			MIMEType:    "application/json",
		},
		{
			URI:         "argo://cluster/cluster-workflow-templates",
			Name:        "cluster-cluster-workflow-templates-list",
			Title:       "Cluster Workflow Templates List",
			Description: "List all ClusterWorkflowTemplates available in the cluster",
			MIMEType:    "application/json",
		},
		{
			URI:         "argo://cluster/cron-workflows",
			Name:        "cluster-cron-workflows-list",
			Title:       "Cron Workflows List",
			Description: "List all CronWorkflows in the cluster with schedule and status",
			MIMEType:    "application/json",
		},
	}
}

// AllClusterResourceTemplates returns all parameterized cluster resource template definitions.
func AllClusterResourceTemplates() []ClusterResourceTemplateDefinition {
	return []ClusterResourceTemplateDefinition{
		{
			URITemplate: "argo://cluster/workflow-templates/{namespace}/{name}",
			Name:        "cluster-workflow-template-detail",
			Title:       "Workflow Template Details",
			Description: "Get full details of a specific WorkflowTemplate",
			MIMEType:    "application/json",
		},
		{
			URITemplate: "argo://cluster/cluster-workflow-templates/{name}",
			Name:        "cluster-cluster-workflow-template-detail",
			Title:       "Cluster Workflow Template Details",
			Description: "Get full details of a specific ClusterWorkflowTemplate",
			MIMEType:    "application/json",
		},
	}
}

// RegisterClusterResources registers all dynamic cluster resources with the MCP server.
func RegisterClusterResources(s *mcp.Server, client argo.ClientInterface) {
	// Register static list resources
	for _, def := range AllClusterResources() {
		resource := &mcp.Resource{
			URI:         def.URI,
			Name:        def.Name,
			Title:       def.Title,
			Description: def.Description,
			MIMEType:    def.MIMEType,
		}
		s.AddResource(resource, clusterResourceHandler(def.URI, client))
	}

	// Register parameterized resource templates
	for _, def := range AllClusterResourceTemplates() {
		template := &mcp.ResourceTemplate{
			URITemplate: def.URITemplate,
			Name:        def.Name,
			Title:       def.Title,
			Description: def.Description,
			MIMEType:    def.MIMEType,
		}
		s.AddResourceTemplate(template, clusterResourceTemplateHandler(def.URITemplate, client))
	}
}

// clusterResourceHandler returns a handler for static cluster resources.
//
//nolint:unparam // uri kept for consistency with ResourceHandler signature and future use
func clusterResourceHandler(_ string, client argo.ClientInterface) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		requestURI := req.Params.URI

		// Parse query parameters if present (e.g., ?namespace=default)
		parsedURI, err := url.Parse(requestURI)
		if err != nil {
			return nil, fmt.Errorf("invalid URI: %w", err)
		}

		// Get namespace from query parameters or use default
		namespace := parsedURI.Query().Get("namespace")

		var content string

		switch {
		case strings.HasPrefix(requestURI, "argo://cluster/workflow-templates"):
			content, err = listWorkflowTemplatesContent(ctx, client, namespace)
		case strings.HasPrefix(requestURI, "argo://cluster/cluster-workflow-templates"):
			content, err = listClusterWorkflowTemplatesContent(ctx, client)
		case strings.HasPrefix(requestURI, "argo://cluster/cron-workflows"):
			content, err = listCronWorkflowsContent(ctx, client, namespace)
		default:
			return nil, mcp.ResourceNotFoundError(requestURI)
		}

		if err != nil {
			return nil, err
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      requestURI,
					MIMEType: "application/json",
					Text:     content,
				},
			},
		}, nil
	}
}

// clusterResourceTemplateHandler returns a handler for parameterized cluster resources.
//
//nolint:unparam // uriTemplate will be used in future extensions
func clusterResourceTemplateHandler(uriTemplate string, client argo.ClientInterface) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		requestURI := req.Params.URI

		var content string
		var err error

		switch {
		case strings.HasPrefix(requestURI, "argo://cluster/workflow-templates/"):
			// Parse namespace and name from URI: argo://cluster/workflow-templates/{namespace}/{name}
			parts := strings.TrimPrefix(requestURI, "argo://cluster/workflow-templates/")
			segments := strings.SplitN(parts, "/", 2)
			if len(segments) != 2 || segments[0] == "" || segments[1] == "" {
				return nil, fmt.Errorf("invalid URI format, expected argo://cluster/workflow-templates/{namespace}/{name}")
			}
			content, err = getWorkflowTemplateContent(ctx, client, segments[0], segments[1])

		case strings.HasPrefix(requestURI, "argo://cluster/cluster-workflow-templates/"):
			// Parse name from URI: argo://cluster/cluster-workflow-templates/{name}
			name := strings.TrimPrefix(requestURI, "argo://cluster/cluster-workflow-templates/")
			if name == "" {
				return nil, fmt.Errorf("invalid URI format, expected argo://cluster/cluster-workflow-templates/{name}")
			}
			content, err = getClusterWorkflowTemplateContent(ctx, client, name)

		default:
			return nil, mcp.ResourceNotFoundError(requestURI)
		}

		if err != nil {
			return nil, err
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      requestURI,
					MIMEType: "application/json",
					Text:     content,
				},
			},
		}, nil
	}
}

// listWorkflowTemplatesContent returns a JSON list of WorkflowTemplates.
func listWorkflowTemplatesContent(ctx context.Context, client argo.ClientInterface, namespace string) (string, error) {
	wftService, err := client.WorkflowTemplateService()
	if err != nil {
		return "", fmt.Errorf("failed to get workflow template service: %w", err)
	}

	// Use provided namespace or default
	if namespace == "" {
		namespace = client.DefaultNamespace()
	}

	listResp, err := wftService.ListWorkflowTemplates(ctx, &workflowtemplate.WorkflowTemplateListRequest{
		Namespace: namespace,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list workflow templates: %w", err)
	}

	summaries := make([]WorkflowTemplateSummary, 0, len(listResp.Items))
	for _, wft := range listResp.Items {
		summary := WorkflowTemplateSummary{
			Name:      wft.Name,
			Namespace: wft.Namespace,
		}

		// Get description from annotation
		if wft.Annotations != nil {
			if desc, ok := wft.Annotations["workflows.argoproj.io/description"]; ok {
				summary.Description = desc
			}
		}

		// Format timestamp
		if !wft.CreationTimestamp.IsZero() {
			summary.CreatedAt = wft.CreationTimestamp.Format(time.RFC3339)
		}

		// Extract parameter names
		if wft.Spec.Arguments.Parameters != nil {
			for _, p := range wft.Spec.Arguments.Parameters {
				summary.Parameters = append(summary.Parameters, p.Name)
			}
		}

		summaries = append(summaries, summary)
	}

	result := map[string]interface{}{
		"namespace": namespace,
		"count":     len(summaries),
		"templates": summaries,
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(data), nil
}

// listClusterWorkflowTemplatesContent returns a JSON list of ClusterWorkflowTemplates.
func listClusterWorkflowTemplatesContent(ctx context.Context, client argo.ClientInterface) (string, error) {
	cwftService, err := client.ClusterWorkflowTemplateService()
	if err != nil {
		return "", fmt.Errorf("failed to get cluster workflow template service: %w", err)
	}

	listResp, err := cwftService.ListClusterWorkflowTemplates(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateListRequest{})
	if err != nil {
		return "", fmt.Errorf("failed to list cluster workflow templates: %w", err)
	}

	summaries := make([]ClusterWorkflowTemplateSummary, 0, len(listResp.Items))
	for _, cwft := range listResp.Items {
		summary := ClusterWorkflowTemplateSummary{
			Name: cwft.Name,
		}

		// Get description from annotation
		if cwft.Annotations != nil {
			if desc, ok := cwft.Annotations["workflows.argoproj.io/description"]; ok {
				summary.Description = desc
			}
		}

		// Format timestamp
		if !cwft.CreationTimestamp.IsZero() {
			summary.CreatedAt = cwft.CreationTimestamp.Format(time.RFC3339)
		}

		// Extract parameter names
		if cwft.Spec.Arguments.Parameters != nil {
			for _, p := range cwft.Spec.Arguments.Parameters {
				summary.Parameters = append(summary.Parameters, p.Name)
			}
		}

		summaries = append(summaries, summary)
	}

	result := map[string]interface{}{
		"count":     len(summaries),
		"templates": summaries,
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(data), nil
}

// listCronWorkflowsContent returns a JSON list of CronWorkflows.
func listCronWorkflowsContent(ctx context.Context, client argo.ClientInterface, namespace string) (string, error) {
	cronService, err := client.CronWorkflowService()
	if err != nil {
		return "", fmt.Errorf("failed to get cron workflow service: %w", err)
	}

	// Use provided namespace or default
	if namespace == "" {
		namespace = client.DefaultNamespace()
	}

	listResp, err := cronService.ListCronWorkflows(ctx, &cronworkflow.ListCronWorkflowsRequest{
		Namespace: namespace,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list cron workflows: %w", err)
	}

	summaries := make([]CronWorkflowSummary, 0, len(listResp.Items))
	for _, cron := range listResp.Items {
		summary := CronWorkflowSummary{
			Name:      cron.Name,
			Namespace: cron.Namespace,
			Schedules: cron.Spec.GetSchedules(),
			Suspended: cron.Spec.Suspend,
		}

		// Format timestamps
		if !cron.CreationTimestamp.IsZero() {
			summary.CreatedAt = cron.CreationTimestamp.Format(time.RFC3339)
		}

		// Status information
		if cron.Status.LastScheduledTime != nil && !cron.Status.LastScheduledTime.IsZero() {
			summary.LastRun = cron.Status.LastScheduledTime.Format(time.RFC3339)
		}

		// Count active, successful, and failed workflows
		summary.ActiveCount = len(cron.Status.Active)
		summary.SuccessCount = int(cron.Status.Succeeded)
		summary.FailedCount = int(cron.Status.Failed)

		summaries = append(summaries, summary)
	}

	result := map[string]interface{}{
		"namespace":     namespace,
		"count":         len(summaries),
		"cronWorkflows": summaries,
	}

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(data), nil
}

// getWorkflowTemplateContent returns the full details of a specific WorkflowTemplate.
func getWorkflowTemplateContent(ctx context.Context, client argo.ClientInterface, namespace, name string) (string, error) {
	wftService, err := client.WorkflowTemplateService()
	if err != nil {
		return "", fmt.Errorf("failed to get workflow template service: %w", err)
	}

	wft, err := wftService.GetWorkflowTemplate(ctx, &workflowtemplate.WorkflowTemplateGetRequest{
		Namespace: namespace,
		Name:      name,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get workflow template: %w", err)
	}

	// Build a structured response with key information
	result := map[string]interface{}{
		"name":      wft.Name,
		"namespace": wft.Namespace,
	}

	// Add metadata
	if !wft.CreationTimestamp.IsZero() {
		result["createdAt"] = wft.CreationTimestamp.Format(time.RFC3339)
	}
	if wft.Labels != nil {
		result["labels"] = wft.Labels
	}
	if wft.Annotations != nil {
		result["annotations"] = wft.Annotations
	}

	// Add spec details
	spec := map[string]interface{}{
		"entrypoint": wft.Spec.Entrypoint,
	}

	// Arguments (parameters and artifacts)
	if args := buildArgumentsMap(wft.Spec.Arguments.Parameters); args != nil {
		spec["arguments"] = args
	}

	// Template names
	if len(wft.Spec.Templates) > 0 {
		templateNames := make([]string, 0, len(wft.Spec.Templates))
		for _, t := range wft.Spec.Templates {
			templateNames = append(templateNames, t.Name)
		}
		spec["templates"] = templateNames
	}

	result["spec"] = spec

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(data), nil
}

// getClusterWorkflowTemplateContent returns the full details of a specific ClusterWorkflowTemplate.
func getClusterWorkflowTemplateContent(ctx context.Context, client argo.ClientInterface, name string) (string, error) {
	cwftService, err := client.ClusterWorkflowTemplateService()
	if err != nil {
		return "", fmt.Errorf("failed to get cluster workflow template service: %w", err)
	}

	cwft, err := cwftService.GetClusterWorkflowTemplate(ctx, &clusterworkflowtemplate.ClusterWorkflowTemplateGetRequest{
		Name: name,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get cluster workflow template: %w", err)
	}

	// Build a structured response with key information
	result := map[string]interface{}{
		"name": cwft.Name,
	}

	// Add metadata
	if !cwft.CreationTimestamp.IsZero() {
		result["createdAt"] = cwft.CreationTimestamp.Format(time.RFC3339)
	}
	if cwft.Labels != nil {
		result["labels"] = cwft.Labels
	}
	if cwft.Annotations != nil {
		result["annotations"] = cwft.Annotations
	}

	// Add spec details
	spec := map[string]interface{}{
		"entrypoint": cwft.Spec.Entrypoint,
	}

	// Arguments (parameters and artifacts)
	if args := buildArgumentsMap(cwft.Spec.Arguments.Parameters); args != nil {
		spec["arguments"] = args
	}

	// Template names
	if len(cwft.Spec.Templates) > 0 {
		templateNames := make([]string, 0, len(cwft.Spec.Templates))
		for _, t := range cwft.Spec.Templates {
			templateNames = append(templateNames, t.Name)
		}
		spec["templates"] = templateNames
	}

	result["spec"] = spec

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(data), nil
}
