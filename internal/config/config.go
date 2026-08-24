// Package config handles configuration parsing and validation.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/pipekit/mcp-for-argo-workflows/pkg/argo"
)

// Valid transport modes.
const (
	TransportStdio = "stdio"
	// TransportHTTP is the HTTP+SSE transport, which streamable HTTP supersedes
	// in the MCP specification. Kept for existing clients.
	TransportHTTP = "http"
	// TransportStreamableHTTP is the streamable HTTP transport. Prefer it for
	// remote deployments: it needs no long-lived server-to-client channel, so a
	// reverse proxy that closes idle streams cannot strand a session.
	TransportStreamableHTTP = "streamable-http"
)

// isValidTransport reports whether name is a supported transport mode.
func isValidTransport(name string) bool {
	switch name {
	case TransportStdio, TransportHTTP, TransportStreamableHTTP:
		return true
	default:
		return false
	}
}

// Config holds the combined configuration for the MCP server.
// Fields are ordered for memory alignment rather than by topic.
type Config struct {
	// Context is the Kubernetes context to use (direct K8s mode only).
	Context string
	// Transport is the MCP transport mode: "stdio", "http" or "streamable-http".
	Transport string
	// ArgoServer is the Argo Server host:port (empty = direct K8s).
	ArgoServer string
	// ArgoToken is the bearer token for Argo Server auth.
	ArgoToken string
	// Namespace is the default namespace for operations.
	Namespace string
	// Kubeconfig is the path to the kubeconfig file (direct K8s mode only).
	Kubeconfig string
	// HTTPAddr is the HTTP listen address (e.g., ":8080").
	HTTPAddr string
	// AllowedContexts restricts which kubeconfig contexts may be used when
	// multi-context is enabled. Empty means all contexts are allowed.
	AllowedContexts []string
	// InsecureSkipVerify skips TLS certificate verification.
	InsecureSkipVerify bool
	// HTTP1 forces HTTP/1.1 (REST) instead of gRPC for Argo Server.
	HTTP1 bool
	// ReadOnly disables all mutating tools when enabled.
	ReadOnly bool
	// MultiContext allows tools to select a kubeconfig context per call.
	// Only effective in direct Kubernetes mode with stdio transport.
	MultiContext bool
	// Secure enables TLS when connecting to Argo Server.
	Secure bool
	// multiContextExplicit records whether MultiContext was set explicitly
	// via flag or environment rather than defaulted, so Validate can reject
	// an explicit enable in modes where it cannot take effect.
	multiContextExplicit bool
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Transport:    TransportStdio,
		HTTPAddr:     ":8080",
		Namespace:    "default",
		Secure:       true,
		ReadOnly:     false,
		MultiContext: true,
	}
}

// Validate returns an error if the configuration is invalid.
func (c *Config) Validate() error {
	if !isValidTransport(c.Transport) {
		return fmt.Errorf("invalid transport %q, must be %q, %q or %q",
			c.Transport, TransportStdio, TransportHTTP, TransportStreamableHTTP)
	}

	// Validate HTTP address has a port when using an HTTP-based transport
	if c.UsesHTTPListener() && c.HTTPAddr == "" {
		return fmt.Errorf("http-addr is required when using the %q transport", c.Transport)
	}

	if err := c.validateMultiContext(); err != nil {
		return err
	}

	return nil
}

// validateMultiContext normalizes the allowed-contexts list and rejects
// multi-context settings that cannot take effect, so an operator never runs
// believing a restriction is active when the feature is off.
func (c *Config) validateMultiContext() error {
	if len(c.AllowedContexts) > 0 {
		normalized := make([]string, 0, len(c.AllowedContexts))
		for _, name := range c.AllowedContexts {
			if name = strings.TrimSpace(name); name != "" {
				normalized = append(normalized, name)
			}
		}
		if len(normalized) == 0 {
			return fmt.Errorf("allowed-contexts is set but contains no context names")
		}
		c.AllowedContexts = normalized
	}

	if c.MultiContextEnabled() {
		return nil
	}
	if len(c.AllowedContexts) > 0 {
		return fmt.Errorf("allowed-contexts requires multi-context mode: direct Kubernetes mode (no argo-server), stdio transport, and multi-context enabled")
	}
	if c.multiContextExplicit && c.MultiContext {
		return fmt.Errorf("multi-context requires direct Kubernetes mode (no argo-server) and stdio transport")
	}
	return nil
}

// NewFromFlags creates a Config from CLI flags and environment variables.
// Precedence: CLI flags > Environment variables > Default values.
func NewFromFlags() (*Config, error) {
	cfg := DefaultConfig()

	// Define CLI flags
	pflag.StringVar(&cfg.Transport, "transport", cfg.Transport, "MCP transport mode: stdio, http (SSE) or streamable-http")
	pflag.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP listen address")
	pflag.StringVar(&cfg.ArgoServer, "argo-server", cfg.ArgoServer, "Argo Server host:port (empty = direct K8s)")
	pflag.StringVar(&cfg.ArgoToken, "argo-token", cfg.ArgoToken, "Bearer token for Argo Server auth")
	pflag.StringVar(&cfg.Namespace, "namespace", cfg.Namespace, "Default namespace for operations")
	pflag.BoolVar(&cfg.Secure, "argo-secure", cfg.Secure, "Use TLS when connecting to Argo Server")
	pflag.BoolVar(&cfg.InsecureSkipVerify, "argo-insecure-skip-verify", cfg.InsecureSkipVerify, "Skip TLS certificate verification")
	pflag.BoolVar(&cfg.HTTP1, "argo-http1", cfg.HTTP1, "Use HTTP/1.1 (REST) instead of gRPC for Argo Server")
	pflag.BoolVar(&cfg.ReadOnly, "read-only", cfg.ReadOnly, "Run in read-only mode, disabling all mutating tools")
	pflag.StringVar(&cfg.Kubeconfig, "kubeconfig", cfg.Kubeconfig, "Path to kubeconfig file")
	pflag.StringVar(&cfg.Context, "context", cfg.Context, "Kubernetes context to use")
	pflag.BoolVar(&cfg.MultiContext, "multi-context", cfg.MultiContext, "Allow tools to select a kubeconfig context per call (direct K8s mode with stdio transport only)")
	pflag.StringSliceVar(&cfg.AllowedContexts, "allowed-contexts", cfg.AllowedContexts, "Kubeconfig contexts permitted for per-call selection (empty = all)")

	// Parse CLI flags
	pflag.Parse()

	// Apply environment variables for values not set via CLI flags
	applyEnvOverrides(cfg)

	return cfg, nil
}

// getEnvIfNotSet returns the environment variable value if the flag was not explicitly set.
// Accepts a FlagSet for testability; pass pflag.CommandLine for normal usage.
func getEnvIfNotSet(fs *pflag.FlagSet, flagName, envKey, current string) string {
	if !fs.Changed(flagName) {
		if v := os.Getenv(envKey); v != "" {
			return v
		}
	}
	return current
}

// getEnvBoolIfNotSet returns the boolean environment variable value if the flag was not explicitly set.
// Accepts a FlagSet for testability; pass pflag.CommandLine for normal usage.
func getEnvBoolIfNotSet(fs *pflag.FlagSet, flagName, envKey string, current bool) bool {
	if !fs.Changed(flagName) {
		if v := os.Getenv(envKey); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				return b
			}
			slog.Warn("invalid boolean env var, using default",
				"env", envKey, "value", strconv.Quote(v), "default", current)
		}
	}
	return current
}

// applyEnvOverrides applies environment variable values for unset flags.
func applyEnvOverrides(cfg *Config) {
	applyEnvOverridesWithFlagSet(pflag.CommandLine, cfg)
}

// applyEnvOverridesWithFlagSet applies environment variable values for unset flags.
// Accepts a FlagSet for testability.
func applyEnvOverridesWithFlagSet(fs *pflag.FlagSet, cfg *Config) {
	// Transport with validation
	if !fs.Changed("transport") {
		if v := os.Getenv("MCP_TRANSPORT"); v != "" {
			v = strings.ToLower(strings.TrimSpace(v))
			if !isValidTransport(v) {
				slog.Warn("invalid MCP_TRANSPORT value, using default",
					"value", strconv.Quote(v), "default", cfg.Transport)
			} else {
				cfg.Transport = v
			}
		}
	}

	cfg.HTTPAddr = getEnvIfNotSet(fs, "http-addr", "MCP_HTTP_ADDR", cfg.HTTPAddr)
	cfg.ArgoServer = getEnvIfNotSet(fs, "argo-server", "ARGO_SERVER", cfg.ArgoServer)
	cfg.ArgoToken = getEnvIfNotSet(fs, "argo-token", "ARGO_TOKEN", cfg.ArgoToken)
	cfg.Namespace = getEnvIfNotSet(fs, "namespace", "ARGO_NAMESPACE", cfg.Namespace)
	cfg.Kubeconfig = getEnvIfNotSet(fs, "kubeconfig", "KUBECONFIG", cfg.Kubeconfig)

	cfg.Secure = getEnvBoolIfNotSet(fs, "argo-secure", "ARGO_SECURE", cfg.Secure)
	cfg.InsecureSkipVerify = getEnvBoolIfNotSet(fs, "argo-insecure-skip-verify", "ARGO_INSECURE_SKIP_VERIFY", cfg.InsecureSkipVerify)
	cfg.HTTP1 = getEnvBoolIfNotSet(fs, "argo-http1", "ARGO_HTTP1", cfg.HTTP1)
	cfg.ReadOnly = getEnvBoolIfNotSet(fs, "read-only", "MCP_READ_ONLY", cfg.ReadOnly)

	cfg.MultiContext = getEnvBoolIfNotSet(fs, "multi-context", "MCP_MULTI_CONTEXT", cfg.MultiContext)
	cfg.multiContextExplicit = fs.Changed("multi-context") || os.Getenv("MCP_MULTI_CONTEXT") != ""
	if !fs.Changed("allowed-contexts") {
		if v := os.Getenv("MCP_ALLOWED_CONTEXTS"); v != "" {
			cfg.AllowedContexts = strings.Split(v, ",")
		}
	}

	// Note: There's no standard env var for Kubernetes context,
	// so --context is CLI-only
}

// MultiContextEnabled reports whether tools may select a kubeconfig context
// per call. This requires direct Kubernetes mode (no Argo Server), stdio
// transport, and multi-context not being disabled.
func (c *Config) MultiContextEnabled() bool {
	return c.MultiContext && c.ArgoServer == "" && c.Transport == TransportStdio
}

// ToArgoConfig converts the Config to an argo.Config for creating the Argo client.
func (c *Config) ToArgoConfig() *argo.Config {
	return &argo.Config{
		ArgoServer:         c.ArgoServer,
		ArgoToken:          c.ArgoToken,
		Namespace:          c.Namespace,
		Kubeconfig:         c.Kubeconfig,
		Context:            c.Context,
		Secure:             c.Secure,
		InsecureSkipVerify: c.InsecureSkipVerify,
		HTTP1:              c.HTTP1,
	}
}

// IsHTTPTransport returns true if the HTTP+SSE transport mode is configured.
func (c *Config) IsHTTPTransport() bool {
	return c.Transport == TransportHTTP
}

// IsStreamableHTTPTransport returns true if the streamable HTTP transport mode
// is configured.
func (c *Config) IsStreamableHTTPTransport() bool {
	return c.Transport == TransportStreamableHTTP
}

// UsesHTTPListener returns true if the configured transport listens on a TCP
// address rather than on stdio.
func (c *Config) UsesHTTPListener() bool {
	return c.IsHTTPTransport() || c.IsStreamableHTTPTransport()
}
