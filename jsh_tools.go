package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerJSHTools(mcpServer *server.MCPServer, client *SSHClient) {
	mcpServer.AddTool(
		mcp.NewTool("jsh_exec",
			mcp.WithTitleAnnotation("Run Machbase JSH script"),
			mcp.WithDescription("Run an LLM-authored JavaScript script once via the raw jsh engine (SSH user neo-mcp:jsh). No DB session and no /usr/bin commands are available here; use jsh_run_command for those. Read neo://manual/jsh first."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithString("script", mcp.Required(), mcp.Description("JSH JavaScript source")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			script, err := requireArgument(request.Params.Arguments, "script")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result, err := client.RunJSH(ctx, script)
			return jshToolResult(result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("jsh_run_command",
			mcp.WithTitleAnnotation("Run Machbase JSH command"),
			mcp.WithDescription("Run an existing neo-shell/JSH command (e.g. sql, show, import, export) through full neo-shell (SSH user neo-mcp), with an already-authenticated DB session. Read neo://manual/jsh first."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithString("command", mcp.Required(), mcp.Description("Existing neo-shell or JSH command with its arguments")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			command, err := requireArgument(request.Params.Arguments, "command")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result, err := client.RunNeoShellCommand(ctx, command)
			return jshToolResult(result, err)
		},
	)
}

func jshToolResult(result JSHResult, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("neo-mcp SSH request failed: %v; stdout=%q; stderr=%q", err, result.Stdout, result.Stderr)), nil
	}
	return mcp.NewToolResultJSON(result)
}
