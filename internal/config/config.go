// Package config handles configuration parsing and validation.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"

	"github.com/Joibel/mcp-for-argo-workflows/pkg/argo"
)

// Valid transport modes.
const (
	TransportStdio = "stdio"
	TransportHTTP  = "http"
)

// Config holds the combined configuration for the MCP server.
type Config struct {
	// Server settings
	Transport string // "stdio" or "http"
	HTTPAddr  string // HTTP listen address (e.g., ":8080")

	// Argo connection settings
	ArgoServer string // Argo Server host:port (empty = direct K8s)
	ArgoToken  string // Bearer token for Argo Server auth
	Namespace  string // Default namespace for operations

	// Kubernetes settings (when not using Argo Server)
	Kubeconfig string // Path to kubeconfig file
	Context    string // Kubernetes context to use

	// TLS settings (grouped together for alignment)
	Secure             bool // Use TLS when connecting to Argo Server
	InsecureSkipVerify bool // Skip TLS certificate verification

	// HTTP1 forces HTTP/1.1 (REST) instead of gRPC for Argo Server
	HTTP1 bool
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Transport: TransportStdio,
		HTTPAddr:  ":8080",
		Namespace: "default",
		Secure:    true,
	}
}

// Validate returns an error if the configuration is invalid.
func (c *Config) Validate() error {
	if c.Transport != TransportStdio && c.Transport != TransportHTTP {
		return fmt.Errorf("invalid transport %q, must be %q or %q", c.Transport, TransportStdio, TransportHTTP)
	}

	// Validate HTTP address has a port when using HTTP transport
	if c.Transport == TransportHTTP && c.HTTPAddr == "" {
		return fmt.Errorf("http-addr is required when using HTTP transport")
	}

	return nil
}

// NewFromFlags creates a Config from CLI flags and environment variables.
// Precedence: CLI flags > Environment variables > Default values.
func NewFromFlags() (*Config, error) {
	cfg := DefaultConfig()

	// Define CLI flags
	pflag.StringVar(&cfg.Transport, "transport", cfg.Transport, "MCP transport mode: stdio or http")
	pflag.StringVar(&cfg.HTTPAddr, "http-addr", cfg.HTTPAddr, "HTTP listen address")
	pflag.StringVar(&cfg.ArgoServer, "argo-server", cfg.ArgoServer, "Argo Server host:port (empty = direct K8s)")
	pflag.StringVar(&cfg.ArgoToken, "argo-token", cfg.ArgoToken, "Bearer token for Argo Server auth")
	pflag.StringVar(&cfg.Namespace, "namespace", cfg.Namespace, "Default namespace for operations")
	pflag.BoolVar(&cfg.Secure, "argo-secure", cfg.Secure, "Use TLS when connecting to Argo Server")
	pflag.BoolVar(&cfg.InsecureSkipVerify, "argo-insecure-skip-verify", cfg.InsecureSkipVerify, "Skip TLS certificate verification")
	pflag.BoolVar(&cfg.HTTP1, "argo-http1", cfg.HTTP1, "Use HTTP/1.1 (REST) instead of gRPC for Argo Server")
	pflag.StringVar(&cfg.Kubeconfig, "kubeconfig", cfg.Kubeconfig, "Path to kubeconfig file")
	pflag.StringVar(&cfg.Context, "context", cfg.Context, "Kubernetes context to use")

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
			if v != TransportStdio && v != TransportHTTP {
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

	// Note: There's no standard env var for Kubernetes context,
	// so --context is CLI-only
}

// ToArgoConfig converts the Config to an argo.Config for creating the Argo client.
func (c *Config) ToArgoConfig() *argo.Config {
	return &argo.Config{
		ArgoServer:         c.ArgoServer,
		ArgoToken:          c.ArgoToken,
		Namespace:          c.Namespace,
		Kubeconfig:         c.Kubeconfig,
		Secure:             c.Secure,
		InsecureSkipVerify: c.InsecureSkipVerify,
		HTTP1:              c.HTTP1,
	}
}

// IsHTTPTransport returns true if the HTTP transport mode is configured.
func (c *Config) IsHTTPTransport() bool {
	return c.Transport == TransportHTTP
}
