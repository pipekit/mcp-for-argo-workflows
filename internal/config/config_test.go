package config

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiContextEnabled(t *testing.T) {
	tests := []struct {
		name         string
		transport    string
		argoServer   string
		multiContext bool
		want         bool
	}{
		{name: "stdio direct k8s enabled", transport: TransportStdio, multiContext: true, want: true},
		{name: "stdio direct k8s disabled", transport: TransportStdio, multiContext: false, want: false},
		{name: "http transport", transport: TransportHTTP, multiContext: true, want: false},
		{name: "argo server mode", transport: TransportStdio, argoServer: "localhost:2746", multiContext: true, want: false},
		{name: "argo server and http", transport: TransportHTTP, argoServer: "localhost:2746", multiContext: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Transport = tt.transport
			cfg.ArgoServer = tt.argoServer
			cfg.MultiContext = tt.multiContext
			assert.Equal(t, tt.want, cfg.MultiContextEnabled())
		})
	}
}

func TestValidate_MultiContext(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name:   "defaults are valid",
			mutate: func(_ *Config) {},
		},
		{
			name: "allowlist in eligible mode",
			mutate: func(c *Config) {
				c.AllowedContexts = []string{"alpha", "beta"}
			},
		},
		{
			name: "allowlist entries are normalized",
			mutate: func(c *Config) {
				c.AllowedContexts = []string{" alpha ", "", "beta"}
			},
		},
		{
			name: "allowlist with only empty entries",
			mutate: func(c *Config) {
				c.AllowedContexts = []string{" ", ""}
			},
			wantErr: "contains no context names",
		},
		{
			name: "allowlist in argo server mode",
			mutate: func(c *Config) {
				c.ArgoServer = "localhost:2746"
				c.AllowedContexts = []string{"alpha"}
			},
			wantErr: "allowed-contexts requires multi-context mode",
		},
		{
			name: "allowlist with http transport",
			mutate: func(c *Config) {
				c.Transport = TransportHTTP
				c.AllowedContexts = []string{"alpha"}
			},
			wantErr: "allowed-contexts requires multi-context mode",
		},
		{
			name: "allowlist with multi-context disabled",
			mutate: func(c *Config) {
				c.MultiContext = false
				c.AllowedContexts = []string{"alpha"}
			},
			wantErr: "allowed-contexts requires multi-context mode",
		},
		{
			name: "explicit enable in argo server mode",
			mutate: func(c *Config) {
				c.ArgoServer = "localhost:2746"
				c.multiContextExplicit = true
			},
			wantErr: "multi-context requires direct Kubernetes mode",
		},
		{
			name: "explicit enable with http transport",
			mutate: func(c *Config) {
				c.Transport = TransportHTTP
				c.multiContextExplicit = true
			},
			wantErr: "multi-context requires direct Kubernetes mode",
		},
		{
			name: "explicit disable in argo server mode is fine",
			mutate: func(c *Config) {
				c.ArgoServer = "localhost:2746"
				c.MultiContext = false
				c.multiContextExplicit = true
			},
		},
		{
			name: "implicit default in argo server mode is fine",
			mutate: func(c *Config) {
				c.ArgoServer = "localhost:2746"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.mutate(cfg)
			err := cfg.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestValidate_NormalizesAllowedContexts(t *testing.T) {
	cfg := DefaultConfig()
	cfg.AllowedContexts = []string{" alpha ", "", "beta", "  "}
	require.NoError(t, cfg.Validate())
	assert.Equal(t, []string{"alpha", "beta"}, cfg.AllowedContexts)
}

func TestApplyEnvOverrides_MultiContext(t *testing.T) {
	t.Run("MCP_MULTI_CONTEXT disables", func(t *testing.T) {
		t.Setenv("MCP_MULTI_CONTEXT", "false")
		cfg := DefaultConfig()
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		applyEnvOverridesWithFlagSet(fs, cfg)
		assert.False(t, cfg.MultiContext)
		assert.True(t, cfg.multiContextExplicit)
	})

	t.Run("MCP_ALLOWED_CONTEXTS splits on commas", func(t *testing.T) {
		t.Setenv("MCP_ALLOWED_CONTEXTS", "alpha, beta ,gamma")
		cfg := DefaultConfig()
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		applyEnvOverridesWithFlagSet(fs, cfg)
		require.NoError(t, cfg.Validate())
		assert.Equal(t, []string{"alpha", "beta", "gamma"}, cfg.AllowedContexts)
	})

	t.Run("unset leaves defaults", func(t *testing.T) {
		cfg := DefaultConfig()
		fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
		applyEnvOverridesWithFlagSet(fs, cfg)
		assert.True(t, cfg.MultiContext)
		assert.False(t, cfg.multiContextExplicit)
		assert.Empty(t, cfg.AllowedContexts)
	})
}
