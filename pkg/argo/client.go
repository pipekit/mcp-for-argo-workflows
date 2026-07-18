package argo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient"
	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/clusterworkflowtemplate"
	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/info"
	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowarchive"
	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// ErrArchivedWorkflowsNotSupported is returned when trying to access archived workflows
	// in direct Kubernetes API mode.
	ErrArchivedWorkflowsNotSupported = errors.New("archived workflows are only supported with Argo Server connection")

	// ErrMultiContextUnavailable is returned when a kubeconfig context is requested
	// but per-call context selection is not available.
	ErrMultiContextUnavailable = errors.New("selecting a kubeconfig context per call is not available (requires direct Kubernetes mode, stdio transport, and multi-context enabled)")
)

// ClientInterface defines the interface for interacting with Argo Workflows.
// This interface is implemented by both the real Client and mock implementations.
type ClientInterface interface {
	// WorkflowService returns the workflow service client.
	WorkflowService() workflow.WorkflowServiceClient

	// CronWorkflowService returns the cron workflow service client.
	CronWorkflowService() (cronworkflow.CronWorkflowServiceClient, error)

	// WorkflowTemplateService returns the workflow template service client.
	WorkflowTemplateService() (workflowtemplate.WorkflowTemplateServiceClient, error)

	// ClusterWorkflowTemplateService returns the cluster workflow template service client.
	ClusterWorkflowTemplateService() (clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient, error)

	// ArchivedWorkflowService returns the archived workflow service client.
	ArchivedWorkflowService() (workflowarchive.ArchivedWorkflowServiceClient, error)

	// InfoService returns the info service client.
	InfoService() (info.InfoServiceClient, error)

	// IsArgoServerMode returns true if connected via Argo Server.
	IsArgoServerMode() bool

	// DefaultNamespace returns the default namespace configured for this client.
	DefaultNamespace() string

	// Context returns the context associated with this client.
	Context() context.Context

	// ForKubeContext returns a client bound to the named kubeconfig context.
	// An empty name returns the receiver. It returns ErrMultiContextUnavailable
	// when per-call context selection is not available.
	ForKubeContext(name string) (ClientInterface, error)

	// ListKubeContexts returns the kubeconfig context names available for
	// per-call selection, along with the default context name. It returns
	// ErrMultiContextUnavailable when per-call context selection is not available.
	ListKubeContexts() (names []string, defaultContext string, err error)

	// MultiContextEnabled reports whether per-call kubeconfig context selection
	// is available on this client.
	MultiContextEnabled() bool
}

// Ensure Client implements ClientInterface.
var _ ClientInterface = (*Client)(nil)

// Client wraps the Argo Workflows API client and provides service client getters.
type Client struct {
	config    *Config
	apiClient apiclient.Client
	// ctx is returned by apiclient.NewClientFromOpts and contains authentication
	// metadata required for API calls. This is the standard Argo SDK pattern.
	ctx context.Context //nolint:containedctx // Required by Argo SDK design
}

// NewClient creates a new Argo Workflows client based on the provided configuration.
// It supports two connection modes:
//   - Argo Server mode: When config.ArgoServer is set, connects via gRPC
//   - Direct K8s mode: When config.ArgoServer is empty, uses kubeconfig
//
// The provided context is used as the base context for the client. Cancelling this
// context will affect all operations using the client.
func NewClient(ctx context.Context, config *Config) (*Client, error) {
	if config == nil {
		return nil, errors.New("config cannot be nil")
	}

	var opts apiclient.Opts

	if config.ArgoServer != "" {
		// Argo Server mode
		opts = apiclient.Opts{
			ArgoServerOpts: apiclient.ArgoServerOpts{
				URL:                config.ArgoServer,
				Secure:             config.Secure,
				InsecureSkipVerify: config.InsecureSkipVerify,
				HTTP1:              config.HTTP1,
			},
		}

		// Add auth supplier if token is provided
		if config.ArgoToken != "" {
			opts.AuthSupplier = func() string {
				return config.ArgoToken
			}
		}
	} else {
		// Direct Kubernetes API mode
		opts = apiclient.Opts{}

		loadingRules := buildLoadingRules(config.Kubeconfig)
		overrides := &clientcmd.ConfigOverrides{CurrentContext: config.Context}

		logResolvedContext(loadingRules, overrides)

		opts.ClientConfigSupplier = func() clientcmd.ClientConfig {
			return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
		}
	}

	clientCtx, apiClient, err := apiclient.NewClientFromOptsWithContext(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create Argo API client: %w", err)
	}

	return &Client{
		config:    config,
		apiClient: apiClient,
		ctx:       clientCtx,
	}, nil
}

// WorkflowService returns the workflow service client.
func (c *Client) WorkflowService() workflow.WorkflowServiceClient {
	return c.apiClient.NewWorkflowServiceClient(c.ctx)
}

// Note: The WorkflowServiceClient is returned directly without error since
// the Argo API client always returns a valid client for this service.

// CronWorkflowService returns the cron workflow service client.
func (c *Client) CronWorkflowService() (cronworkflow.CronWorkflowServiceClient, error) {
	client, err := c.apiClient.NewCronWorkflowServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create cron workflow service client: %w", err)
	}
	return client, nil
}

// WorkflowTemplateService returns the workflow template service client.
func (c *Client) WorkflowTemplateService() (workflowtemplate.WorkflowTemplateServiceClient, error) {
	client, err := c.apiClient.NewWorkflowTemplateServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow template service client: %w", err)
	}
	return client, nil
}

// ClusterWorkflowTemplateService returns the cluster workflow template service client.
func (c *Client) ClusterWorkflowTemplateService() (clusterworkflowtemplate.ClusterWorkflowTemplateServiceClient, error) {
	client, err := c.apiClient.NewClusterWorkflowTemplateServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster workflow template service client: %w", err)
	}
	return client, nil
}

// ArchivedWorkflowService returns the archived workflow service client.
// This service is only available when using Argo Server mode.
// Returns ErrArchivedWorkflowsNotSupported if not in Argo Server mode.
func (c *Client) ArchivedWorkflowService() (workflowarchive.ArchivedWorkflowServiceClient, error) {
	if !c.IsArgoServerMode() {
		return nil, ErrArchivedWorkflowsNotSupported
	}

	client, err := c.apiClient.NewArchivedWorkflowServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create archived workflow service client: %w", err)
	}
	return client, nil
}

// InfoService returns the info service client.
func (c *Client) InfoService() (info.InfoServiceClient, error) {
	client, err := c.apiClient.NewInfoServiceClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create info service client: %w", err)
	}
	return client, nil
}

// IsArgoServerMode returns true if the client is connected via Argo Server,
// false if using direct Kubernetes API access.
func (c *Client) IsArgoServerMode() bool {
	return c.config.ArgoServer != ""
}

// DefaultNamespace returns the default namespace configured for this client.
func (c *Client) DefaultNamespace() string {
	return c.config.Namespace
}

// Context returns the context associated with this client.
func (c *Client) Context() context.Context {
	return c.ctx
}

// ForKubeContext returns the receiver for an empty context name and
// ErrMultiContextUnavailable otherwise: a plain Client is bound to a single
// kubeconfig context for its lifetime. This rejection is the call-time
// enforcement that keeps context selection unavailable in Argo Server mode,
// HTTP transport, or when multi-context is disabled, regardless of what a
// caller sends.
func (c *Client) ForKubeContext(name string) (ClientInterface, error) {
	if strings.TrimSpace(name) == "" {
		return c, nil
	}
	return nil, ErrMultiContextUnavailable
}

// ListKubeContexts returns ErrMultiContextUnavailable: a plain Client does not
// support per-call context selection.
func (c *Client) ListKubeContexts() ([]string, string, error) {
	return nil, "", ErrMultiContextUnavailable
}

// MultiContextEnabled reports false: a plain Client is bound to a single
// kubeconfig context.
func (c *Client) MultiContextEnabled() bool {
	return false
}

// buildLoadingRules constructs kubeconfig loading rules from a path string.
// The path may contain multiple files joined by os.PathListSeparator (matching
// the kubectl KUBECONFIG convention). When empty, default discovery is used.
func buildLoadingRules(kubeconfig string) *clientcmd.ClientConfigLoadingRules {
	if kubeconfig == "" {
		return clientcmd.NewDefaultClientConfigLoadingRules()
	}
	return &clientcmd.ClientConfigLoadingRules{Precedence: filepath.SplitList(kubeconfig)}
}

// logResolvedContext resolves and logs the kubeconfig context the client will
// use, so operators can confirm which cluster the server is bound to.
func logResolvedContext(loadingRules *clientcmd.ClientConfigLoadingRules, overrides *clientcmd.ConfigOverrides) {
	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides).RawConfig()
	if err != nil {
		slog.Warn("could not resolve kubeconfig context for logging", "error", err)
		return
	}

	contextName := overrides.CurrentContext
	if contextName == "" {
		contextName = rawConfig.CurrentContext
	}

	cluster := ""
	if kctx, ok := rawConfig.Contexts[contextName]; ok {
		cluster = kctx.Cluster
	}

	slog.Info("using kubeconfig context",
		"context", contextName,
		"cluster", cluster,
		"sources", loadingRules.Precedence,
	)
}
