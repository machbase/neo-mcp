package main

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerFileTools(mcpServer *server.MCPServer, client *Client) {
	mcpServer.AddTool(
		mcp.NewTool("fs_list",
			mcp.WithTitleAnnotation("List server files"),
			mcp.WithDescription("List supported files and directories on the machbase-neo server. Paths are server-side SSFS paths, not local workspace paths."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithString("path", mcp.Description("Server-side directory path; defaults to /")),
			mcp.WithString("filter", mcp.Description("Optional server file extension filter, for example .tql or .md")),
			mcp.WithBoolean("recursive", mcp.Description("Include recursive directory entries when supported")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, _ := request.Params.Arguments.(map[string]any)
			remotePath := stringValue(arguments["path"], "/")
			filter := stringValue(arguments["filter"], "")
			recursive, _ := boolArgument(request.Params.Arguments, "recursive")
			result, err := client.ListFiles(ctx, remotePath, filter, recursive)
			return toolResult(result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("fs_read",
			mcp.WithTitleAnnotation("Read server file"),
			mcp.WithDescription("Read a supported file from the machbase-neo server-side file system. Paths are server-side SSFS paths, not local workspace paths."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithString("path", mcp.Required(), mcp.Description("Server-side file path")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			remotePath, err := requireArgument(request.Params.Arguments, "path")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result, err := client.ReadFile(ctx, remotePath)
			return toolResult(result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("fs_write",
			mcp.WithTitleAnnotation("Write server file"),
			mcp.WithDescription("Write content to a supported file on the machbase-neo server-side file system. This changes server state; paths are server-side SSFS paths."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithString("path", mcp.Required(), mcp.Description("Server-side file path")),
			mcp.WithString("content", mcp.Required(), mcp.Description("Complete file content")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			path, err := requireArgument(request.Params.Arguments, "path")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			content, err := requireArgument(request.Params.Arguments, "content")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result, err := client.WriteFile(ctx, path, content)
			return toolResult(result, err)
		},
	)
}
