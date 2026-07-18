package argo

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"sync"

	"k8s.io/client-go/tools/clientcmd"
)

// MultiContextClient is a Client for the default kubeconfig context that can
// also serve other contexts per call via ForKubeContext, lazily building and
// caching one Client per context name.
//
// It is the single enforcement point for per-call context selection: names are
// validated against the kubeconfig contexts and the allowlist snapshotted at
// startup (editing the kubeconfig requires a server restart), and clients are
// only ever built for names that pass both checks. Unknown and disallowed
// names produce identical errors so the allowlist does not reveal which hidden
// contexts exist.
// Fields are ordered for memory alignment rather than by topic.
type MultiContextClient struct {
	// baseCtx is the startup context used to construct per-context clients, so
	// their lifetime is tied to the server process rather than a single call.
	baseCtx context.Context //nolint:containedctx // Mirrors Client's Argo SDK pattern

	*Client

	config *Config
	// available is the set of selectable context names: present in the
	// kubeconfig at startup and, when an allowlist is configured, listed in it.
	available map[string]struct{}
	// newClient builds a Client for a cloned per-context Config. Injectable so
	// tests can avoid contacting real clusters.
	newClient      func(context.Context, *Config) (*Client, error)
	cache          map[string]*Client
	defaultContext string
	// names is the sorted form of available, served by ListKubeContexts.
	names []string
	// mu guards cache. It is held while building a client so concurrent first
	// calls for a context produce exactly one Client; that serializes builds,
	// which happen at most once per context.
	mu sync.Mutex
}

// Ensure MultiContextClient implements ClientInterface.
var _ ClientInterface = (*MultiContextClient)(nil)

// NewMultiContextClient creates a client for the default kubeconfig context
// that can also serve other contexts per call. When allowedContexts is
// non-empty it restricts the selectable contexts and must include the default
// context, otherwise construction fails.
func NewMultiContextClient(ctx context.Context, config *Config, allowedContexts []string) (*MultiContextClient, error) {
	base, err := NewClient(ctx, config)
	if err != nil {
		return nil, err
	}
	return newMultiContextClient(ctx, base, config, allowedContexts, NewClient)
}

func newMultiContextClient(ctx context.Context, base *Client, config *Config, allowedContexts []string, newClient func(context.Context, *Config) (*Client, error)) (*MultiContextClient, error) {
	rawConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(buildLoadingRules(config.Kubeconfig), &clientcmd.ConfigOverrides{}).RawConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig contexts: %w", err)
	}

	defaultContext := config.Context
	if defaultContext == "" {
		defaultContext = rawConfig.CurrentContext
	}

	allowed := make(map[string]struct{}, len(allowedContexts))
	for _, name := range allowedContexts {
		if name = strings.TrimSpace(name); name != "" {
			allowed[name] = struct{}{}
		}
	}
	if len(allowed) > 0 {
		if _, ok := allowed[defaultContext]; !ok {
			return nil, fmt.Errorf("default context %q is not in the allowed contexts list", defaultContext)
		}
	}

	available := make(map[string]struct{}, len(rawConfig.Contexts))
	names := make([]string, 0, len(rawConfig.Contexts))
	for name := range rawConfig.Contexts {
		if len(allowed) > 0 {
			if _, ok := allowed[name]; !ok {
				continue
			}
		}
		available[name] = struct{}{}
		names = append(names, name)
	}
	sort.Strings(names)

	return &MultiContextClient{
		Client:         base,
		baseCtx:        ctx,
		config:         config,
		defaultContext: defaultContext,
		available:      available,
		names:          names,
		newClient:      newClient,
		cache:          make(map[string]*Client),
	}, nil
}

// ForKubeContext returns a client bound to the named kubeconfig context,
// building and caching it on first use. An empty name or the default context
// name returns the receiver.
func (m *MultiContextClient) ForKubeContext(name string) (ClientInterface, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == m.defaultContext {
		return m, nil
	}

	if _, ok := m.available[name]; !ok {
		return nil, fmt.Errorf("context %q is not available (use list_contexts to see available contexts)", name)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if client, ok := m.cache[name]; ok {
		return client, nil
	}

	contextConfig := *m.config
	contextConfig.Context = name
	client, err := m.newClient(m.baseCtx, &contextConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for context %q: %w", name, err)
	}
	m.cache[name] = client
	slog.Info("created client for kubeconfig context", "context", name)
	return client, nil
}

// ListKubeContexts returns the selectable kubeconfig context names, sorted,
// along with the default context name.
func (m *MultiContextClient) ListKubeContexts() ([]string, string, error) {
	return slices.Clone(m.names), m.defaultContext, nil
}

// MultiContextEnabled reports true: this client supports per-call kubeconfig
// context selection.
func (m *MultiContextClient) MultiContextEnabled() bool {
	return true
}

// MergeContext returns a context that draws cancellation and deadline from
// request, and context values from values before falling back to request. The
// Argo SDK embeds each cluster's authentication in its client's context, so
// handlers switching context must run calls with the selected client's values
// while preserving the MCP request's cancellation — otherwise the call would
// silently execute against the default cluster.
func MergeContext(request, values context.Context) context.Context {
	return mergedContext{Context: request, values: values}
}

//nolint:containedctx // Exists to combine two contexts; holding them is the point.
type mergedContext struct {
	context.Context
	values context.Context
}

func (m mergedContext) Value(key any) any {
	if v := m.values.Value(key); v != nil {
		return v
	}
	return m.Context.Value(key)
}
