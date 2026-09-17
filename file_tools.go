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
			mcp.WithDescription("List files and directories in the MCP /project namespace. For example, /project lists the server work area; do not use internal /work paths."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithString("path", mcp.Description("MCP server directory path under /project; defaults to /project")),
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
			mcp.WithDescription("Read a supported file from the MCP /project namespace. Do not use the internal JSH /work path."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithString("path", mcp.Required(), mcp.Description("MCP server file path under /project")),
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
			mcp.WithDescription("Write content to a supported file in the MCP /project namespace, creating any missing parent directories automatically (mkdir -p semantics). This changes server state; do not use internal /work paths."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithString("path", mcp.Required(), mcp.Description("MCP server file path under /project")),
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
