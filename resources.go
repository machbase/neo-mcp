package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

//go:embed manual
var manualFS embed.FS

func registerManualResources(mcpServer *server.MCPServer) {
	_ = fs.WalkDir(manualFS, "manual", func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || !strings.HasSuffix(filePath, ".md") {
			return walkErr
		}
		text, err := fs.ReadFile(manualFS, filePath)
		if err != nil {
			return err
		}
		relativePath := strings.TrimPrefix(filePath, "manual/")
		uri := "neo://manual/" + strings.TrimSuffix(relativePath, ".md")
		name := strings.TrimSuffix(path.Base(relativePath), ".md")
		description := fmt.Sprintf("Maintained Machbase LLM guidance: %s", relativePath)
		resource := mcp.NewResource(uri, name,
			mcp.WithResourceDescription(description),
			mcp.WithMIMEType("text/markdown"),
		)
		resourceText := string(text)
		mcpServer.AddResource(
			resource,
			func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
				return []mcp.ResourceContents{
					mcp.TextResourceContents{URI: request.Params.URI, MIMEType: "text/markdown", Text: resourceText},
				}, nil
			},
		)
		return nil
	})
}

func readManual(uri string) (string, error) {
	if !strings.HasPrefix(uri, "neo://manual/") {
		return "", fmt.Errorf("unsupported manual URI %q", uri)
	}
	relativePath := strings.TrimPrefix(uri, "neo://manual/")
	if relativePath == "" || strings.Contains(relativePath, "..") {
		return "", fmt.Errorf("invalid manual URI %q", uri)
	}
	if !strings.HasSuffix(relativePath, ".md") {
		relativePath += ".md"
	}
	filePath := path.Join("manual", relativePath)
	if !strings.HasPrefix(filePath, "manual/") {
		return "", fmt.Errorf("invalid manual URI %q", uri)
	}
	content, err := fs.ReadFile(manualFS, filePath)
	if err != nil {
		return "", fmt.Errorf("manual %q not found: %w", uri, err)
	}
	return string(content), nil
}
