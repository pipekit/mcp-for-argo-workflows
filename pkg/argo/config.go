package argo

import (
	"log/slog"
	"os"
	"strconv"
)

// Config holds the configuration for connecting to Argo Workflows.
type Config struct {
	// ArgoServer is the Argo Server host:port (e.g., "localhost:2746").
	// When empty, the client will use direct Kubernetes API access.
	ArgoServer string

	// ArgoToken is the bearer token for authentication with Argo Server.
	ArgoToken string

	// Namespace is the default namespace for operations.
	Namespace string

	// Kubeconfig is the path to the kubeconfig file.
	// Used for direct Kubernetes API access when ArgoServer is empty.
	Kubeconfig string

	// Secure indicates whether to use TLS when connecting to Argo Server.
	// Only applies when ArgoServer is set.
	Secure bool

	// InsecureSkipVerify skips TLS certificate verification.
	// Only applies when ArgoServer is set and Secure is true.
	InsecureSkipVerify bool

	// HTTP1 forces the use of HTTP/1.1 (REST API) instead of gRPC when
	// connecting to Argo Server. This is required when the Argo Server is
	// behind a reverse proxy (e.g., nginx ingress) that does not support gRPC.
	HTTP1 bool
}

// NewConfigFromEnv creates a Config from environment variables.
func NewConfigFromEnv() *Config {
	config := &Config{
		ArgoServer: os.Getenv("ARGO_SERVER"),
		ArgoToken:  os.Getenv("ARGO_TOKEN"),
		Namespace:  os.Getenv("ARGO_NAMESPACE"),
		Kubeconfig: os.Getenv("KUBECONFIG"),
		Secure:     true, // Default to secure
	}

	// Parse ARGO_SECURE if set
	if secureStr := os.Getenv("ARGO_SECURE"); secureStr != "" {
		secure, err := strconv.ParseBool(secureStr)
		if err != nil {
			slog.Warn("invalid ARGO_SECURE value, using default",
				"value", strconv.Quote(secureStr), "default", true)
		} else {
			config.Secure = secure
		}
	}

	// Parse ARGO_INSECURE_SKIP_VERIFY if set
	if skipVerifyStr := os.Getenv("ARGO_INSECURE_SKIP_VERIFY"); skipVerifyStr != "" {
		skipVerify, err := strconv.ParseBool(skipVerifyStr)
		if err != nil {
			slog.Warn("invalid ARGO_INSECURE_SKIP_VERIFY value, using default",
				"value", strconv.Quote(skipVerifyStr), "default", false)
		} else {
			config.InsecureSkipVerify = skipVerify
		}
	}

	// Parse ARGO_HTTP1 if set
	if http1Str := os.Getenv("ARGO_HTTP1"); http1Str != "" {
		http1, err := strconv.ParseBool(http1Str)
		if err != nil {
			slog.Warn("invalid ARGO_HTTP1 value, using default",
				"value", strconv.Quote(http1Str), "default", false)
		} else {
			config.HTTP1 = http1
		}
	}

	// Default namespace if not set
	if config.Namespace == "" {
		config.Namespace = "default"
	}

	return config
}
