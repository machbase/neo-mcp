package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
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

func registerDatabaseResources(mcpServer *server.MCPServer, client *Client) {
	mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"neo://machbase/session",
			"Machbase session context",
			mcp.WithTemplateTitle("Machbase current session"),
			mcp.WithTemplateDescription("Return the current logical database and authenticated user."),
			mcp.WithTemplateMIMEType("application/json"),
		),
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			result, err := client.CurrentSession(ctx)
			if err != nil {
				return nil, err
			}
			text, err := marshalResourceJSON(result)
			if err != nil {
				return nil, err
			}
			return []mcp.ResourceContents{mcp.TextResourceContents{URI: request.Params.URI, MIMEType: "application/json", Text: text}}, nil
		},
	)

	mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"neo://machbase/databases",
			"Machbase databases",
			mcp.WithTemplateTitle("Machbase database list"),
			mcp.WithTemplateDescription("List logical and mounted databases visible to the current user."),
			mcp.WithTemplateMIMEType("application/json"),
		),
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			result, err := client.ListDatabases(ctx)
			if err != nil {
				return nil, err
			}
			text, err := marshalResourceJSON(result)
			if err != nil {
				return nil, err
			}
			return []mcp.ResourceContents{mcp.TextResourceContents{URI: request.Params.URI, MIMEType: "application/json", Text: text}}, nil
		},
	)

	mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"neo://machbase/tables",
			"Machbase tables",
			mcp.WithTemplateTitle("Machbase table list"),
			mcp.WithTemplateDescription("List tables in the current database."),
			mcp.WithTemplateMIMEType("application/json"),
		),
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			result, err := client.ListTables(ctx)
			if err != nil {
				return nil, err
			}
			text, err := marshalResourceJSON(result)
			if err != nil {
				return nil, err
			}
			return []mcp.ResourceContents{mcp.TextResourceContents{URI: request.Params.URI, MIMEType: "application/json", Text: text}}, nil
		},
	)
	registerTableListTemplate(mcpServer, client, "neo://machbase/tables/{prefix}", "List tables in the current database whose names start with the requested prefix.", false)
	registerTableListTemplate(mcpServer, client, "neo://machbase/tables/{database}", "List tables in the requested logical database.", true)
	registerTableListTemplate(mcpServer, client, "neo://machbase/tables/{database}/{prefix}", "List tables in a logical database whose names start with the requested prefix.", true)

	mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate(
			"neo://machbase/table/{table}",
			"Machbase table schema",
			mcp.WithTemplateTitle("Machbase table schema"),
			mcp.WithTemplateDescription("Describe a Machbase table using the TQL SQL and JSON pipeline."),
			mcp.WithTemplateMIMEType("application/json"),
		),
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			table, err := tableResourcePart(request.Params.URI)
			if err != nil {
				return nil, err
			}
			result, err := client.DescribeTableTQL(ctx, table)
			if err != nil {
				return nil, err
			}
			text, err := marshalResourceJSON(result)
			if err != nil {
				return nil, err
			}
			return []mcp.ResourceContents{mcp.TextResourceContents{URI: request.Params.URI, MIMEType: "application/json", Text: text}}, nil
		},
	)
}

func registerTableListTemplate(mcpServer *server.MCPServer, client *Client, pattern, description string, hasDatabase bool) {
	mcpServer.AddResourceTemplate(
		mcp.NewResourceTemplate(
			pattern,
			"Machbase tables",
			mcp.WithTemplateTitle("Machbase table list"),
			mcp.WithTemplateDescription(description),
			mcp.WithTemplateMIMEType("application/json"),
		),
		func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			database, prefix, err := tableListResourceParts(request.Params.URI, hasDatabase)
			if err != nil {
				return nil, err
			}
			result, err := client.ListTablesScoped(ctx, database, prefix)
			if err != nil {
				return nil, err
			}
			text, err := marshalResourceJSON(result)
			if err != nil {
				return nil, err
			}
			return []mcp.ResourceContents{mcp.TextResourceContents{URI: request.Params.URI, MIMEType: "application/json", Text: text}}, nil
		},
	)
}

func tableListResourceParts(rawURI string, hasDatabase bool) (string, string, error) {
	u, err := url.Parse(rawURI)
	if err != nil || u.Scheme != "neo" || u.Host != "machbase" {
		return "", "", fmt.Errorf("invalid database resource URI %q", rawURI)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) == 1 && parts[0] == "tables" && !hasDatabase {
		return "", "", nil
	}
	if len(parts) < 2 || parts[0] != "tables" {
		return "", "", fmt.Errorf("invalid database resource URI %q", rawURI)
	}
	if hasDatabase {
		if len(parts) != 2 && len(parts) != 3 {
			return "", "", fmt.Errorf("invalid database resource URI %q", rawURI)
		}
		database, err := url.PathUnescape(parts[1])
		if err != nil {
			return "", "", err
		}
		prefix := ""
		if len(parts) == 3 {
			prefix, err = url.PathUnescape(parts[2])
			if err != nil {
				return "", "", err
			}
		}
		return database, prefix, nil
	}
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid database resource URI %q", rawURI)
	}
	prefix, err := url.PathUnescape(parts[1])
	return "", prefix, err
}

func tableResourcePart(rawURI string) (string, error) {
	u, err := url.Parse(rawURI)
	if err != nil || u.Scheme != "neo" || u.Host != "machbase" {
		return "", fmt.Errorf("invalid table resource URI %q", rawURI)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 || parts[0] != "table" {
		return "", fmt.Errorf("invalid table resource URI %q", rawURI)
	}
	return url.PathUnescape(parts[1])
}

func marshalResourceJSON(value any) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode database resource: %w", err)
	}
	return string(data), nil
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
