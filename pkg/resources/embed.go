// Package resources implements MCP resource handlers for Argo Workflows schema documentation.
package resources

import (
	"embed"
	"fmt"
)

//go:embed docs/*.md
var docsFS embed.FS

// readDoc reads a markdown document from the embedded filesystem.
func readDoc(filename string) (string, error) {
	content, err := docsFS.ReadFile("docs/" + filename)
	if err != nil {
		return "", fmt.Errorf("failed to read embedded file %s: %w", filename, err)
	}
	return string(content), nil
}
