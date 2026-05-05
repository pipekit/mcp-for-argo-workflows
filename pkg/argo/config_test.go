package argo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfigFromEnv(t *testing.T) {
	tests := []struct {
		envVars  map[string]string
		expected *Config
		name     string
	}{
		{
			name:    "default values when no env vars set",
			envVars: map[string]string{},
			expected: &Config{
				ArgoServer:         "",
				ArgoToken:          "",
				Namespace:          "default",
				Kubeconfig:         "",
				Secure:             true,
				InsecureSkipVerify: false,
			},
		},
		{
			name: "all env vars set",
			envVars: map[string]string{
				"ARGO_SERVER":               "localhost:2746",
				"ARGO_TOKEN":                "test-token",
				"ARGO_NAMESPACE":            "test-namespace",
				"KUBECONFIG":                "/path/to/kubeconfig",
				"ARGO_SECURE":               "false",
				"ARGO_INSECURE_SKIP_VERIFY": "true",
			},
			expected: &Config{
				ArgoServer:         "localhost:2746",
				ArgoToken:          "test-token",
				Namespace:          "test-namespace",
				Kubeconfig:         "/path/to/kubeconfig",
				Secure:             false,
				InsecureSkipVerify: true,
			},
		},
		{
			name: "argo server mode with secure connection",
			envVars: map[string]string{
				"ARGO_SERVER":    "argo.example.com:443",
				"ARGO_TOKEN":     "bearer-token-123",
				"ARGO_NAMESPACE": "production",
				"ARGO_SECURE":    "true",
			},
			expected: &Config{
				ArgoServer:         "argo.example.com:443",
				ArgoToken:          "bearer-token-123",
				Namespace:          "production",
				Kubeconfig:         "",
				Secure:             true,
				InsecureSkipVerify: false,
			},
		},
		{
			name: "direct kubernetes mode",
			envVars: map[string]string{
				"KUBECONFIG":     "/home/user/.kube/config",
				"ARGO_NAMESPACE": "workflows",
			},
			expected: &Config{
				ArgoServer:         "",
				ArgoToken:          "",
				Namespace:          "workflows",
				Kubeconfig:         "/home/user/.kube/config",
				Secure:             true,
				InsecureSkipVerify: false,
			},
		},
		{
			name: "invalid ARGO_SECURE falls back to default",
			envVars: map[string]string{
				"ARGO_SECURE": "not-a-bool",
			},
			expected: &Config{
				ArgoServer:         "",
				ArgoToken:          "",
				Namespace:          "default",
				Kubeconfig:         "",
				Secure:             true,
				InsecureSkipVerify: false,
			},
		},
		{
			name: "invalid ARGO_INSECURE_SKIP_VERIFY falls back to default",
			envVars: map[string]string{
				"ARGO_INSECURE_SKIP_VERIFY": "invalid",
			},
			expected: &Config{
				ArgoServer:         "",
				ArgoToken:          "",
				Namespace:          "default",
				Kubeconfig:         "",
				Secure:             true,
				InsecureSkipVerify: false,
			},
		},
		{
			name: "various boolean formats for ARGO_SECURE",
			envVars: map[string]string{
				"ARGO_SECURE": "0",
			},
			expected: &Config{
				ArgoServer:         "",
				ArgoToken:          "",
				Namespace:          "default",
				Kubeconfig:         "",
				Secure:             false,
				InsecureSkipVerify: false,
			},
		},
		{
			name: "various boolean formats for ARGO_INSECURE_SKIP_VERIFY",
			envVars: map[string]string{
				"ARGO_INSECURE_SKIP_VERIFY": "1",
			},
			expected: &Config{
				ArgoServer:         "",
				ArgoToken:          "",
				Namespace:          "default",
				Kubeconfig:         "",
				Secure:             true,
				InsecureSkipVerify: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all relevant env vars
			clearEnvVars := []string{
				"ARGO_SERVER",
				"ARGO_TOKEN",
				"ARGO_NAMESPACE",
				"KUBECONFIG",
				"ARGO_SECURE",
				"ARGO_INSECURE_SKIP_VERIFY",
			}
			for _, key := range clearEnvVars {
				t.Setenv(key, "")
			}

			// Set test env vars
			for key, value := range tt.envVars {
				t.Setenv(key, value)
			}

			config := NewConfigFromEnv()

			assert.Equal(t, tt.expected.ArgoServer, config.ArgoServer, "ArgoServer mismatch")
			assert.Equal(t, tt.expected.ArgoToken, config.ArgoToken, "ArgoToken mismatch")
			assert.Equal(t, tt.expected.Namespace, config.Namespace, "Namespace mismatch")
			assert.Equal(t, tt.expected.Kubeconfig, config.Kubeconfig, "Kubeconfig mismatch")
			assert.Equal(t, tt.expected.Secure, config.Secure, "Secure mismatch")
			assert.Equal(t, tt.expected.InsecureSkipVerify, config.InsecureSkipVerify, "InsecureSkipVerify mismatch")
		})
	}
}

func TestNewConfigFromEnv_BooleanParsing(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		value    string
		expected bool
	}{
		{"true lowercase", "ARGO_SECURE", "true", true},
		{"True mixed case", "ARGO_SECURE", "True", true},
		{"TRUE uppercase", "ARGO_SECURE", "TRUE", true},
		{"1 as true", "ARGO_SECURE", "1", true},
		{"false lowercase", "ARGO_SECURE", "false", false},
		{"False mixed case", "ARGO_SECURE", "False", false},
		{"FALSE uppercase", "ARGO_SECURE", "FALSE", false},
		{"0 as false", "ARGO_SECURE", "0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env vars
			t.Setenv("ARGO_SERVER", "")
			t.Setenv("ARGO_TOKEN", "")
			t.Setenv("ARGO_NAMESPACE", "")
			t.Setenv("KUBECONFIG", "")
			t.Setenv("ARGO_SECURE", "")
			t.Setenv("ARGO_INSECURE_SKIP_VERIFY", "")

			// Set the test value
			t.Setenv(tt.envVar, tt.value)

			config := NewConfigFromEnv()

			if tt.envVar == "ARGO_SECURE" {
				assert.Equal(t, tt.expected, config.Secure, "Secure value mismatch for %s=%s", tt.envVar, tt.value)
			}
		})
	}
}
