package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerReadOnlyTools(mcpServer *server.MCPServer, client *Client) {
	mcpServer.AddTool(
		mcp.NewTool("tql_run",
			mcp.WithTitleAnnotation("Run Machbase TQL"),
			mcp.WithDescription("Execute an LLM-authored TQL script through machbase-neo. This is the required execution Tool for TQL; call manual_read with neo://manual/tql first, then call tql_run. Do not use curl, terminal commands, or direct HTTP calls instead."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithString("script", mcp.Required(), mcp.Description("TQL script to execute")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			script, err := requireArgument(request.Params.Arguments, "script")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result, err := client.RunTQL(ctx, script)
			if chart, ok := result.(map[string]any); ok && chart["chartID"] != nil && chart["jsCodeAssets"] != nil {
				chartFile, renderErr := client.WriteChartHTML(ctx, chart)
				if renderErr != nil {
					return mcp.NewToolResultError(renderErr.Error()), nil
				}
				return mcp.NewToolResultStructured(chartFile, fmt.Sprintf("Interactive chart: [Open chart](%s)", chartFile["link"])), nil
			}
			return toolResult(result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("db_query",
			mcp.WithTitleAnnotation("Machbase SQL query"),
			mcp.WithDescription("Execute a SQL query through the configured machbase-neo HTTP API. Read neo://manual/sql first for Machbase-specific SQL guidance."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithString("q", mcp.Required(), mcp.Description("SQL query")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sql, err := requireArgument(request.Params.Arguments, "q")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result, err := client.Query(ctx, sql)
			return toolResult(result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("db_list_tables",
			mcp.WithTitleAnnotation("List Machbase tables"),
			mcp.WithDescription("List tables visible to the configured machbase-neo user. Read neo://manual/sql before composing follow-up queries."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := client.ListTables(ctx)
			return toolResult(result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("db_describe_table",
			mcp.WithTitleAnnotation("Describe Machbase table"),
			mcp.WithDescription("Return metadata for one Machbase table. Read neo://manual/sql before composing queries against the table."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithString("table", mcp.Required(), mcp.Description("Table name")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			table, err := requireArgument(request.Params.Arguments, "table")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result, err := client.DescribeTable(ctx, table)
			return toolResult(result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("db_list_tags",
			mcp.WithTitleAnnotation("List Machbase tags"),
			mcp.WithDescription("List tags for one Machbase table."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithString("table", mcp.Required(), mcp.Description("Table name")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			table, err := requireArgument(request.Params.Arguments, "table")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result, err := client.ListTags(ctx, table)
			return toolResult(result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("db_tag_stat",
			mcp.WithTitleAnnotation("Read Machbase tag statistics"),
			mcp.WithDescription("Return statistics for one tag on one Machbase table."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithString("table", mcp.Required(), mcp.Description("Table name")),
			mcp.WithString("tag", mcp.Required(), mcp.Description("Tag name")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, ok := request.Params.Arguments.(map[string]any)
			if !ok {
				return mcp.NewToolResultError("arguments must be an object"), nil
			}
			table, tableOK := arguments["table"].(string)
			tag, tagOK := arguments["tag"].(string)
			if !tableOK || table == "" || !tagOK || tag == "" {
				return mcp.NewToolResultError("missing required argument table or tag"), nil
			}
			result, err := client.TagStat(ctx, table, tag)
			return toolResult(result, err)
		},
	)
}

func registerManualTool(mcpServer *server.MCPServer) {
	mcpServer.AddTool(
		mcp.NewTool("manual_read",
			mcp.WithTitleAnnotation("Read neo-mcp manual"),
			mcp.WithDescription("Read maintained Machbase SQL, TQL, or JSH guidance. Use URI neo://manual/sql, neo://manual/tql, or neo://manual/jsh; nested markdown resources are also supported."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithString("uri", mcp.Required(), mcp.Description("Manual URI, for example neo://manual/tql")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			uri, err := requireArgument(request.Params.Arguments, "uri")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			text, err := readManual(uri)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			return mcp.NewToolResultText(text), nil
		},
	)
}

func toolResult(result any, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("neo-mcp request failed: %v", err)), nil
	}
	if chart, ok := result.(map[string]any); ok {
		if _, hasChartID := chart["chartID"]; hasChartID {
			if _, hasCodeAssets := chart["jsCodeAssets"]; hasCodeAssets {
				return mcp.NewToolResultStructured(chart, "Machbase TQL chart envelope. Load jsAssets before jsCodeAssets in a webview renderer."), nil
			}
		}
	}
	response, err := mcp.NewToolResultJSON(result)
	if err != nil {
		return nil, err
	}
	return response, nil
}
