package tools

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDotToSVG(t *testing.T) {
	tests := []struct {
		name          string
		dot           string
		validate      func(t *testing.T, svg string)
		expectedError string
	}{
		{
			name: "simple graph",
			dot: `digraph G {
    A -> B;
    B -> C;
}`,
			validate: func(t *testing.T, svg string) {
				assert.True(t, strings.HasPrefix(svg, "<?xml") || strings.Contains(svg, "<svg"))
				assert.Contains(t, svg, "</svg>")
			},
		},
		{
			name: "workflow-like graph with styling",
			dot: `digraph workflow {
    rankdir=TB;
    node [shape=box, style=filled];

    "build" [label="build", fillcolor="#22c55e"];
    "test" [label="test", fillcolor="#3b82f6"];
    "deploy" [label="deploy", fillcolor="#9ca3af"];

    "build" -> "test";
    "test" -> "deploy";
}`,
			validate: func(t *testing.T, svg string) {
				assert.True(t, strings.HasPrefix(svg, "<?xml") || strings.Contains(svg, "<svg"))
				assert.Contains(t, svg, "</svg>")
				// The SVG should contain the node labels
				assert.Contains(t, svg, "build")
				assert.Contains(t, svg, "test")
				assert.Contains(t, svg, "deploy")
			},
		},
		{
			name: "empty graph",
			dot: `digraph G {
    "No nodes"
}`,
			validate: func(t *testing.T, svg string) {
				assert.True(t, strings.HasPrefix(svg, "<?xml") || strings.Contains(svg, "<svg"))
				assert.Contains(t, svg, "</svg>")
			},
		},
		{
			name:          "invalid DOT syntax",
			dot:           "not valid dot syntax {{{",
			expectedError: "failed to parse DOT graph",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svg, err := dotToSVG(t.Context(), tt.dot)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				require.NotEmpty(t, svg)
				tt.validate(t, svg)
			}
		})
	}
}
