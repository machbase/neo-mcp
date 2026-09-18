package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerReadOnlyTools(mcpServer *server.MCPServer, client *Client) {
	mcpServer.AddTool(
		mcp.NewTool("tql_run",
			mcp.WithTitleAnnotation("Run Machbase TQL"),
			mcp.WithDescription("Execute an LLM-authored TQL script through machbase-neo. This is the required execution Tool for TQL; call manual_read with neo://manual/tql first, then call tql_run. Do not use curl, terminal commands, or direct HTTP calls instead."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithString("script", mcp.Required(), mcp.Description("TQL script to execute")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			script, err := requireArgument(request.Params.Arguments, "script")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result, err := client.RunTQL(ctx, script)
			return tqlToolResult(ctx, client, result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("tql_run_file",
			mcp.WithTitleAnnotation("Run server TQL file"),
			mcp.WithDescription("Read and execute a .tql file from the machbase-neo server-side file system. Use fs_list first to discover server paths."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithString("path", mcp.Required(), mcp.Description("Server-side .tql file path")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			remotePath, err := requireArgument(request.Params.Arguments, "path")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result, err := client.RunTQLFile(ctx, remotePath)
			return tqlToolResult(ctx, client, result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("tql_file_link",
			mcp.WithTitleAnnotation("Verify and link server TQL file"),
			mcp.WithDescription("Execute a server-side .tql file through the configured /db/tql/<path>.tql reading API, then return a browser URL and verification result. The loopback browser URL proxies /db, /web, /metrics, and /debug requests with the configured MCP token; the token is not included in the URL."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithString("path", mcp.Required(), mcp.Description("Server-side .tql file path")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			remotePath, err := requireArgument(request.Params.Arguments, "path")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result, contentType, err := client.VerifyTQLFile(ctx, remotePath)
			if err != nil {
				return toolResult(nil, err)
			}
			url, err := client.BrowserTQLFileURL(remotePath)
			if err != nil {
				return toolResult(nil, err)
			}
			return tqlFileLinkResult(remotePath, url, contentType, result), nil
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("db_query",
			mcp.WithTitleAnnotation("Machbase SQL query"),
			mcp.WithDescription("Execute a SQL query through the configured machbase-neo HTTP API. Read neo://manual/sql first. Every temporary CREATE TAG TABLE consumes TAG cache memory and must have deterministic DROP TABLE cleanup; also avoid SQL keywords such as ROWS as aliases."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithString("q", mcp.Required(), mcp.Description("SQL query")),
			mcp.WithString("format", mcp.Description("Result format: json, csv, box, or ndjson")),
			mcp.WithString("timeformat", mcp.Description("Datetime format: s, ms, us, or ns")),
			mcp.WithString("tz", mcp.Description("Timezone, for example UTC, Local, or Asia/Seoul")),
			mcp.WithString("binaryformat", mcp.Description("Binary format: hex, base64, bytes, or preview")),
			mcp.WithString("header", mcp.Description("Use skip to omit CSV/BOX headers")),
			mcp.WithNumber("precision", mcp.Description("Floating-point precision; -1 disables rounding")),
			mcp.WithBoolean("rownum", mcp.Description("Include row numbers")),
			mcp.WithString("db", mcp.Description("Target logical database")),
			mcp.WithAny("p", mcp.Description("Positional JSON array or named JSON object bind parameters")),
			mcp.WithBoolean("transpose", mcp.Description("JSON-only column-oriented output")),
			mcp.WithBoolean("rowsFlatten", mcp.Description("JSON-only flattened rows")),
			mcp.WithBoolean("rowsArray", mcp.Description("JSON-only array of row objects")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sql, err := requireArgument(request.Params.Arguments, "q")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			arguments, _ := request.Params.Arguments.(map[string]any)
			options := map[string]any{}
			for _, key := range []string{"format", "timeformat", "tz", "binaryformat", "header", "precision", "rownum", "db", "p", "transpose", "rowsFlatten", "rowsArray"} {
				if value, ok := arguments[key]; ok {
					options[key] = value
				}
			}
			result, err := client.QueryWithOptions(ctx, sql, options)
			if format, _ := arguments["format"].(string); format != "" && format != "json" {
				if text, ok := result.(string); ok && err == nil {
					return mcp.NewToolResultText(text), nil
				}
			}
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

func tqlToolResult(ctx context.Context, client *Client, result any, err error) (*mcp.CallToolResult, error) {
	if err != nil {
		return toolResult(result, err)
	}
	if chart, ok := result.(map[string]any); ok && chart["chartID"] != nil && chart["jsCodeAssets"] != nil {
		chartFile, renderErr := client.WriteChartHTML(ctx, chart)
		if renderErr != nil {
			return mcp.NewToolResultError(renderErr.Error()), nil
		}
		return chartToolResult(chartFile), nil
	}
	if text, ok := result.(string); ok {
		return mcp.NewToolResultText(text), nil
	}
	return toolResult(result, nil)
}

func chartToolResult(chartFile map[string]any) *mcp.CallToolResult {
	link := stringValue(chartFile["uri"], stringValue(chartFile["link"], ""))
	return mcp.NewToolResultText(fmt.Sprintf("Interactive chart: [Open chart](%s)", link))
}

func tqlFileLinkResult(path, link, contentType string, result any) *mcp.CallToolResult {
	verification, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(err.Error())
	}
	return mcp.NewToolResultText(fmt.Sprintf("Verified TQL file `%s`: [Open TQL](%s)\n\nContent-Type: `%s`\n\nVerification result:\n```json\n%s\n```", path, link, contentType, verification))
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
		return mcp.NewToolResultError(mapMCPError(err)), nil
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
