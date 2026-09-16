package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerRpcTools wires MCP tools to the JSON-RPC methods allowlisted for
// API-token auth on /db/rpc (see dbRpcAllowedMethods in
// neo-server/mods/server/http.go). Each method there is scoped to the calling
// token's user, so these tools only ever see/manage the token owner's own
// timers, subscribers, and API tokens.
func registerRpcTools(mcpServer *server.MCPServer, client *Client) {
	mcpServer.AddTool(
		mcp.NewTool("render_markdown",
			mcp.WithTitleAnnotation("Render and Execute Markdown"),
			mcp.WithDescription("Render Markdown text to HTML and execute its executable fenced blocks. SQL and JSH require {execute=true}; HTTP blocks execute by their established Markdown contract. Blocks can have side effects."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithString("markdown", mcp.Required(), mcp.Description("Markdown source text")),
			mcp.WithBoolean("darkMode", mcp.Description("Render using dark mode styling (default false)")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			markdown, err := requireArgument(request.Params.Arguments, "markdown")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			darkMode, _ := boolArgument(request.Params.Arguments, "darkMode")
			result, err := client.CallDbRpc(ctx, "markdown.render", []any{markdown, darkMode, ""})
			return toolResult(result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("server_info",
			mcp.WithTitleAnnotation("Machbase server info"),
			mcp.WithDescription("Return machbase-neo version and runtime info (pid, uptime, memory)."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := client.CallDbRpc(ctx, "server.info.get", []any{})
			return toolResult(result, err)
		},
	)

	registerTimerTools(mcpServer, client)
	registerSubscriberTools(mcpServer, client)
	registerTokenTools(mcpServer, client)
}

func registerTimerTools(mcpServer *server.MCPServer, client *Client) {
	mcpServer.AddTool(
		mcp.NewTool("timer_list",
			mcp.WithTitleAnnotation("List timer schedules"),
			mcp.WithDescription("List the caller's own timer schedules."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := client.CallDbRpc(ctx, "timer.list", []any{})
			return toolResult(result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("timer_get",
			mcp.WithTitleAnnotation("Get a timer schedule"),
			mcp.WithDescription("Return one of the caller's own timer schedules by ID."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("Timer ID")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id, err := requireNumberArgument(request.Params.Arguments, "id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result, err := client.CallDbRpc(ctx, "timer.get", []any{id})
			return toolResult(result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("timer_add",
			mcp.WithTitleAnnotation("Add a timer schedule"),
			mcp.WithDescription("Create a new timer schedule owned by the caller."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithString("name", mcp.Required(), mcp.Description("Timer name")),
			mcp.WithString("spec", mcp.Required(), mcp.Description("Cron schedule spec")),
			mcp.WithString("command", mcp.Required(), mcp.Description("Command to run on schedule")),
			mcp.WithBoolean("autoStart", mcp.Description("Start the timer immediately after creation (default false)")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, ok := request.Params.Arguments.(map[string]any)
			if !ok {
				return mcp.NewToolResultError("arguments must be an object"), nil
			}
			name, _ := arguments["name"].(string)
			spec, _ := arguments["spec"].(string)
			command, _ := arguments["command"].(string)
			if name == "" || spec == "" || command == "" {
				return mcp.NewToolResultError("missing required argument name, spec, or command"), nil
			}
			autoStart, _ := boolArgument(request.Params.Arguments, "autoStart")
			req := map[string]any{"name": name, "spec": spec, "command": command, "autoStart": autoStart}
			result, err := client.CallDbRpc(ctx, "timer.add", []any{req})
			return toolResult(result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("timer_update",
			mcp.WithTitleAnnotation("Update a timer schedule"),
			mcp.WithDescription("Update one of the caller's own timer schedules by ID."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("Timer ID")),
			mcp.WithString("spec", mcp.Required(), mcp.Description("Cron schedule spec")),
			mcp.WithString("command", mcp.Required(), mcp.Description("Command to run on schedule")),
			mcp.WithBoolean("autoStart", mcp.Description("Start the timer immediately after update (default false)")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id, err := requireNumberArgument(request.Params.Arguments, "id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			arguments, _ := request.Params.Arguments.(map[string]any)
			spec, _ := arguments["spec"].(string)
			command, _ := arguments["command"].(string)
			if spec == "" || command == "" {
				return mcp.NewToolResultError("missing required argument spec or command"), nil
			}
			autoStart, _ := boolArgument(request.Params.Arguments, "autoStart")
			req := map[string]any{"id": id, "spec": spec, "command": command, "autoStart": autoStart}
			result, err := client.CallDbRpc(ctx, "timer.update", []any{req})
			return toolResult(result, err)
		},
	)

	registerIDOnlyTool(mcpServer, client, "timer_delete", "Delete a timer schedule", "timer.delete", true, "Timer ID")
	registerIDOnlyTool(mcpServer, client, "timer_start", "Start a timer schedule", "timer.start", false, "Timer ID")
	registerIDOnlyTool(mcpServer, client, "timer_stop", "Stop a timer schedule", "timer.stop", false, "Timer ID")
}

func registerSubscriberTools(mcpServer *server.MCPServer, client *Client) {
	mcpServer.AddTool(
		mcp.NewTool("subscriber_list",
			mcp.WithTitleAnnotation("List subscribers"),
			mcp.WithDescription("List the caller's own bridge subscriber schedules."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := client.CallDbRpc(ctx, "subscriber.list", []any{})
			return toolResult(result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("subscriber_get",
			mcp.WithTitleAnnotation("Get a subscriber"),
			mcp.WithDescription("Return one of the caller's own bridge subscriber schedules by ID."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("Subscriber ID")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id, err := requireNumberArgument(request.Params.Arguments, "id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result, err := client.CallDbRpc(ctx, "subscriber.get", []any{id})
			return toolResult(result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("subscriber_add",
			mcp.WithTitleAnnotation("Add a subscriber"),
			mcp.WithDescription("Create a new MQTT bridge subscriber owned by the caller. NATS subscribers are not supported by this tool; use jsh_run_command for those."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithString("name", mcp.Required(), mcp.Description("Subscriber name")),
			mcp.WithString("bridge", mcp.Required(), mcp.Description("Bridge name to subscribe through")),
			mcp.WithString("command", mcp.Required(), mcp.Description("Command to run on each message")),
			mcp.WithString("topic", mcp.Required(), mcp.Description("MQTT topic filter")),
			mcp.WithNumber("qos", mcp.Description("MQTT QoS level (default 0)")),
			mcp.WithBoolean("autoStart", mcp.Description("Start the subscriber immediately after creation (default false)")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, ok := request.Params.Arguments.(map[string]any)
			if !ok {
				return mcp.NewToolResultError("arguments must be an object"), nil
			}
			name, _ := arguments["name"].(string)
			bridgeName, _ := arguments["bridge"].(string)
			command, _ := arguments["command"].(string)
			topic, _ := arguments["topic"].(string)
			if name == "" || bridgeName == "" || command == "" || topic == "" {
				return mcp.NewToolResultError("missing required argument name, bridge, command, or topic"), nil
			}
			qos, _ := numberArgument(request.Params.Arguments, "qos")
			autoStart, _ := boolArgument(request.Params.Arguments, "autoStart")
			req := map[string]any{
				"name": name, "bridge": bridgeName, "command": command, "autoStart": autoStart,
				"mqtt": map[string]any{"topic": topic, "qos": qos},
			}
			result, err := client.CallDbRpc(ctx, "subscriber.add", []any{req})
			return toolResult(result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("subscriber_update",
			mcp.WithTitleAnnotation("Update a subscriber"),
			mcp.WithDescription("Update one of the caller's own MQTT bridge subscribers by ID."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithNumber("id", mcp.Required(), mcp.Description("Subscriber ID")),
			mcp.WithString("bridge", mcp.Required(), mcp.Description("Bridge name to subscribe through")),
			mcp.WithString("command", mcp.Required(), mcp.Description("Command to run on each message")),
			mcp.WithString("topic", mcp.Required(), mcp.Description("MQTT topic filter")),
			mcp.WithNumber("qos", mcp.Description("MQTT QoS level (default 0)")),
			mcp.WithBoolean("autoStart", mcp.Description("Start the subscriber immediately after update (default false)")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id, err := requireNumberArgument(request.Params.Arguments, "id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			arguments, _ := request.Params.Arguments.(map[string]any)
			bridgeName, _ := arguments["bridge"].(string)
			command, _ := arguments["command"].(string)
			topic, _ := arguments["topic"].(string)
			if bridgeName == "" || command == "" || topic == "" {
				return mcp.NewToolResultError("missing required argument bridge, command, or topic"), nil
			}
			qos, _ := numberArgument(request.Params.Arguments, "qos")
			autoStart, _ := boolArgument(request.Params.Arguments, "autoStart")
			req := map[string]any{
				"id": id, "bridge": bridgeName, "command": command, "autoStart": autoStart,
				"mqtt": map[string]any{"topic": topic, "qos": qos},
			}
			result, err := client.CallDbRpc(ctx, "subscriber.update", []any{req})
			return toolResult(result, err)
		},
	)

	registerIDOnlyTool(mcpServer, client, "subscriber_delete", "Delete a subscriber", "subscriber.delete", true, "Subscriber ID")
	registerIDOnlyTool(mcpServer, client, "subscriber_start", "Start a subscriber", "subscriber.start", false, "Subscriber ID")
	registerIDOnlyTool(mcpServer, client, "subscriber_stop", "Stop a subscriber", "subscriber.stop", false, "Subscriber ID")
}

func registerTokenTools(mcpServer *server.MCPServer, client *Client) {
	mcpServer.AddTool(
		mcp.NewTool("token_list",
			mcp.WithTitleAnnotation("List API tokens"),
			mcp.WithDescription("List the caller's own machbase-neo API tokens (hints only, not the secret values)."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			result, err := client.CallDbRpc(ctx, "token.list", []any{})
			return toolResult(result, err)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("token_generate",
			mcp.WithTitleAnnotation("Generate an API token"),
			mcp.WithDescription("Generate a new API token owned by the caller. The plaintext token value is only ever returned once, in this response."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithString("name", mcp.Required(), mcp.Description("Label for the token")),
			mcp.WithNumber("notAfter", mcp.Description("Expiration as a Unix epoch in seconds (default: 10 years from now)")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			name, err := requireArgument(request.Params.Arguments, "name")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			notAfter, _ := numberArgument(request.Params.Arguments, "notAfter")
			result, err := client.CallDbRpc(ctx, "token.generate", []any{name, notAfter})
			return toolResult(result, err)
		},
	)

	registerIDOnlyTool(mcpServer, client, "token_delete", "Delete an API token", "token.delete", true, "API token ID")
}

// registerIDOnlyTool registers a tool that calls a JSON-RPC method taking a
// single numeric ID parameter (delete/start/stop style methods).
func registerIDOnlyTool(mcpServer *server.MCPServer, client *Client, toolName, title, method string, destructive bool, idDescription string) {
	mcpServer.AddTool(
		mcp.NewTool(toolName,
			mcp.WithTitleAnnotation(title),
			mcp.WithDescription(title+"."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(destructive),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithNumber("id", mcp.Required(), mcp.Description(idDescription)),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			id, err := requireNumberArgument(request.Params.Arguments, "id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			result, err := client.CallDbRpc(ctx, method, []any{id})
			return toolResult(result, err)
		},
	)
}

func numberArgument(requestArguments any, name string) (float64, error) {
	arguments, ok := requestArguments.(map[string]any)
	if !ok {
		return 0, fmt.Errorf("arguments must be an object")
	}
	value, ok := arguments[name]
	if !ok {
		return 0, nil
	}
	number, ok := value.(float64)
	if !ok {
		return 0, fmt.Errorf("argument %q must be a number", name)
	}
	return number, nil
}

func requireNumberArgument(requestArguments any, name string) (float64, error) {
	arguments, ok := requestArguments.(map[string]any)
	if !ok {
		return 0, fmt.Errorf("arguments must be an object")
	}
	value, ok := arguments[name]
	if !ok {
		return 0, fmt.Errorf("missing required argument %q", name)
	}
	number, ok := value.(float64)
	if !ok {
		return 0, fmt.Errorf("argument %q must be a number", name)
	}
	return number, nil
}

func boolArgument(requestArguments any, name string) (bool, error) {
	arguments, ok := requestArguments.(map[string]any)
	if !ok {
		return false, fmt.Errorf("arguments must be an object")
	}
	value, ok := arguments[name]
	if !ok {
		return false, nil
	}
	flag, ok := value.(bool)
	if !ok {
		return false, fmt.Errorf("argument %q must be a boolean", name)
	}
	return flag, nil
}
