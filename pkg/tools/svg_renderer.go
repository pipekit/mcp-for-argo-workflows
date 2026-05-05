// Package tools implements MCP tool handlers for Argo Workflows operations.
package tools

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/goccy/go-graphviz"
)

// dotToSVG converts a DOT graph string to SVG format using graphviz.
func dotToSVG(ctx context.Context, dot string) (string, error) {
	g, err := graphviz.New(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create graphviz instance: %w", err)
	}

	graph, err := graphviz.ParseBytes([]byte(dot))
	if err != nil {
		closeErr := g.Close()
		return "", errors.Join(fmt.Errorf("failed to parse DOT graph: %w", err), closeErr)
	}

	var buf bytes.Buffer
	if renderErr := g.Render(ctx, graph, graphviz.SVG, &buf); renderErr != nil {
		graphCloseErr := graph.Close()
		gCloseErr := g.Close()
		return "", errors.Join(fmt.Errorf("failed to render SVG: %w", renderErr), graphCloseErr, gCloseErr)
	}

	// Clean up resources
	graphCloseErr := graph.Close()
	gCloseErr := g.Close()
	if graphCloseErr != nil || gCloseErr != nil {
		return buf.String(), errors.Join(graphCloseErr, gCloseErr)
	}

	return buf.String(), nil
}
