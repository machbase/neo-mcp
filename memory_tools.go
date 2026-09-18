package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const (
	memoryTableName        = "_NEO_AGENT_MEMORY"
	memorySchemaProbeTQL   = "SQL(`show tables with all`)\nJSON()\n"
	memoryStoreReadbackSQL = "SELECT _arrival_time, * FROM _NEO_AGENT_MEMORY WHERE MEMORY_ID = ? ORDER BY _arrival_time DESC LIMIT 1"
	memoryReadbackAttempts = 20
	memoryReadbackInterval = 50 * time.Millisecond
	memoryMaxContent       = 64 * 1024
	memoryMaxLimit         = 100
)

func registerMemoryTools(mcpServer *server.MCPServer, client *Client) {
	mcpServer.AddTool(
		mcp.NewTool("memory_store",
			mcp.WithTitleAnnotation("Store Agent memory"),
			mcp.WithDescription("Store a durable Agent memory record in Machbase. Store important facts, decisions, observations, and verified procedures, not every conversational turn or secrets. Read neo://manual/memory first."),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithString("content", mcp.Required(), mcp.Description("Memory content or summary")),
			mcp.WithString("tenant_id", mcp.Required(), mcp.Description("Logical metadata used for memory filtering; not a security boundary")),
			mcp.WithString("team_id", mcp.Required(), mcp.Description("Logical metadata used for memory filtering; not a security boundary")),
			mcp.WithString("memory_type", mcp.Required(), mcp.Description("conversation, observation, incident, procedure, decision, or result")),
			mcp.WithString("source", mcp.Required(), mcp.Description("Provenance such as agent run, issue, file, or query")),
			mcp.WithString("author_id", mcp.Description("Original author identifier")),
			mcp.WithString("agent_id", mcp.Description("Agent identifier")),
			mcp.WithString("session_id", mcp.Description("Conversation or work session identifier")),
			mcp.WithBoolean("verified", mcp.Description("Whether the memory has been verified; default false")),
			mcp.WithNumber("importance", mcp.Description("Optional application priority from 0 to 1; default 0.5")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := memoryArguments(request.Params.Arguments)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			content, err := requiredString(arguments, "content")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if len(content) > memoryMaxContent {
				return mcp.NewToolResultError(fmt.Sprintf("content exceeds maximum size of %d bytes", memoryMaxContent)), nil
			}
			tenantID, err := requiredString(arguments, "tenant_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			teamID, err := requiredString(arguments, "team_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			memoryType, err := requiredString(arguments, "memory_type")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if !validMemoryType(memoryType) {
				return mcp.NewToolResultError("memory_type must be conversation, observation, incident, procedure, decision, or result"), nil
			}
			source, err := requiredString(arguments, "source")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			authorID := stringValue(arguments["author_id"], "")
			agentID := stringValue(arguments["agent_id"], "")
			sessionID := stringValue(arguments["session_id"], "")
			verified, _ := boolArgument(arguments, "verified")
			importance, err := memoryNumberArgument(arguments, "importance", 0.5)
			if err != nil || importance < 0 || importance > 1 {
				return mcp.NewToolResultError("importance must be a number from 0 to 1"), nil
			}
			memoryID, err := newMemoryID()
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := ensureMemorySchema(ctx, client, true); err != nil {
				return toolResult(nil, err)
			}
			verifiedValue := 0
			if verified {
				verifiedValue = 1
			}
			contentHash := sha256.Sum256([]byte(content))
			query := "INSERT INTO _NEO_AGENT_MEMORY (MEMORY_ID, TENANT_ID, TEAM_ID, AUTHOR_ID, AGENT_ID, SESSION_ID, MEMORY_TYPE, CONTENT, SOURCE, VERIFIED, IMPORTANCE, CONTENT_HASH) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
			_, err = client.QueryWithOptions(ctx, query, map[string]any{"p": []any{
				memoryID, tenantID, teamID, authorID, agentID, sessionID, memoryType, content, source, verifiedValue, importance, hex.EncodeToString(contentHash[:]),
			}})
			if err != nil {
				return toolResult(nil, err)
			}
			arrivalTime, err := readMemoryArrivalTime(ctx, client, memoryID)
			if err != nil {
				return toolResult(nil, err)
			}
			return toolResult(map[string]any{
				"memory_id":       memoryID,
				"tenant_id":       tenantID,
				"team_id":         teamID,
				"memory_type":     memoryType,
				"arrival_time":    arrivalTime,
				"schema":          memoryTableName,
				"content_index":   "_NEO_AGENT_MEMORY_CONTENT_IDX",
				"memory_id_index": "_NEO_AGENT_MEMORY_MEMORY_ID_IDX",
			}, nil)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("memory_search",
			mcp.WithTitleAnnotation("Search Agent memory"),
			mcp.WithDescription("Search Agent memory using Machbase keyword retrieval. The default multi-term operator is OR for recall. Results are lexical candidates, not semantic similarity results. Read neo://manual/memory first."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithString("query", mcp.Required(), mcp.Description("One or more search terms")),
			mcp.WithString("tenant_id", mcp.Description("Optional logical metadata filter")),
			mcp.WithString("team_id", mcp.Description("Optional logical metadata filter")),
			mcp.WithString("memory_type", mcp.Description("Optional memory type filter")),
			mcp.WithString("since", mcp.Description("Optional _arrival_time lower bound")),
			mcp.WithString("until", mcp.Description("Optional _arrival_time upper bound")),
			mcp.WithBoolean("verified_only", mcp.Description("Return only verified memories")),
			mcp.WithNumber("limit", mcp.Description("Maximum results, from 1 to 100; default 10")),
			mcp.WithString("mode", mcp.Description("search, esearch, or exact; default search")),
			mcp.WithString("operator", mcp.Description("OR by default; AND is supported for narrower retrieval")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := memoryArguments(request.Params.Arguments)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			queryText, err := requiredString(arguments, "query")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			terms := strings.Fields(queryText)
			if len(terms) == 0 {
				return mcp.NewToolResultError("query must contain at least one term"), nil
			}
			mode := stringValue(arguments["mode"], "search")
			if mode != "search" && mode != "esearch" && mode != "exact" {
				return mcp.NewToolResultError("mode must be search, esearch, or exact"), nil
			}
			operator := strings.ToUpper(stringValue(arguments["operator"], "OR"))
			if operator != "OR" && operator != "AND" {
				return mcp.NewToolResultError("operator must be OR or AND"), nil
			}
			limit, err := memoryNumberArgument(arguments, "limit", 10)
			if err != nil || limit < 1 || limit > memoryMaxLimit {
				return mcp.NewToolResultError(fmt.Sprintf("limit must be from 1 to %d", memoryMaxLimit)), nil
			}
			where := []string{}
			params := []any{}
			addFilter := func(column, value string) {
				if value != "" {
					where = append(where, column+" = ?")
					params = append(params, value)
				}
			}
			addFilter("TENANT_ID", stringValue(arguments["tenant_id"], ""))
			addFilter("TEAM_ID", stringValue(arguments["team_id"], ""))
			addFilter("MEMORY_TYPE", stringValue(arguments["memory_type"], ""))
			if verified, _ := boolArgument(arguments, "verified_only"); verified {
				where = append(where, "VERIFIED = ?")
				params = append(params, 1)
			}
			if since := stringValue(arguments["since"], ""); since != "" {
				where = append(where, "_arrival_time >= ?")
				params = append(params, since)
			}
			if until := stringValue(arguments["until"], ""); until != "" {
				where = append(where, "_arrival_time <= ?")
				params = append(params, until)
			}
			termClauses := make([]string, len(terms))
			for i, term := range terms {
				if mode == "exact" {
					termClauses[i] = "CONTENT LIKE ?"
					params = append(params, "%"+term+"%")
				} else {
					termClauses[i] = "CONTENT " + strings.ToUpper(mode) + " ?"
					params = append(params, term)
				}
			}
			where = append(where, "("+strings.Join(termClauses, " "+operator+" ")+")")
			sql := "SELECT _arrival_time, MEMORY_ID, TENANT_ID, TEAM_ID, AUTHOR_ID, AGENT_ID, SESSION_ID, MEMORY_TYPE, CONTENT, SOURCE, VERIFIED, IMPORTANCE FROM _NEO_AGENT_MEMORY WHERE " + strings.Join(where, " AND ") + " ORDER BY _arrival_time DESC LIMIT ?"
			params = append(params, int(limit))
			if err := ensureMemorySchema(ctx, client, false); err != nil {
				return toolResult(nil, err)
			}
			result, err := client.QueryWithOptions(ctx, sql, map[string]any{"p": params})
			if err != nil {
				return toolResult(nil, err)
			}
			rows, err := memoryRows(result)
			if err != nil {
				return toolResult(nil, err)
			}
			return toolResult(map[string]any{
				"memories": rows,
				"retrieval": map[string]any{
					"method":            "keyword",
					"mode":              mode,
					"operator":          operator,
					"query_terms":       terms,
					"time_filtered":     stringValue(arguments["since"], "") != "" || stringValue(arguments["until"], "") != "",
					"metadata_filtered": stringValue(arguments["tenant_id"], "") != "" || stringValue(arguments["team_id"], "") != "" || stringValue(arguments["memory_type"], "") != "",
				},
			}, nil)
		},
	)

	mcpServer.AddTool(
		mcp.NewTool("memory_get",
			mcp.WithTitleAnnotation("Get Agent memory"),
			mcp.WithDescription("Get one Agent memory record and its provenance by memory_id. Read neo://manual/memory first."),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithString("memory_id", mcp.Required(), mcp.Description("Memory identifier")),
		),
		func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			arguments, err := memoryArguments(request.Params.Arguments)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			memoryID, err := requiredString(arguments, "memory_id")
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			if err := ensureMemorySchema(ctx, client, false); err != nil {
				return toolResult(nil, err)
			}
			result, err := client.QueryWithOptions(ctx, "SELECT _arrival_time, MEMORY_ID, TENANT_ID, TEAM_ID, AUTHOR_ID, AGENT_ID, SESSION_ID, MEMORY_TYPE, CONTENT, SOURCE, VERIFIED, IMPORTANCE FROM _NEO_AGENT_MEMORY WHERE MEMORY_ID = ? ORDER BY _arrival_time DESC LIMIT 2", map[string]any{"p": []any{memoryID}})
			if err != nil {
				return toolResult(nil, err)
			}
			rows, err := memoryRows(result)
			if err != nil {
				return toolResult(nil, err)
			}
			if len(rows) == 0 {
				return mcp.NewToolResultError("memory_id not found"), nil
			}
			if len(rows) > 1 {
				return mcp.NewToolResultError("memory_id is not unique"), nil
			}
			return toolResult(rows[0], nil)
		},
	)
}

func ensureMemorySchema(ctx context.Context, client *Client, create bool) error {
	if client == nil {
		return fmt.Errorf("memory client is not configured")
	}
	result, err := client.RunTQL(ctx, memorySchemaProbeTQL)
	if err != nil {
		return fmt.Errorf("list memory tables: %w", err)
	}
	rows, err := memoryRows(result)
	if err != nil {
		return fmt.Errorf("list memory tables: %w", err)
	}
	for _, row := range rows {
		if row["TABLE_NAME"] == memoryTableName {
			return nil
		}
	}
	if !create {
		return fmt.Errorf("memory table %s is not available", memoryTableName)
	}
	if _, err := client.Query(ctx, "CREATE LOG TABLE _NEO_AGENT_MEMORY (MEMORY_ID VARCHAR(64), TENANT_ID VARCHAR(64), TEAM_ID VARCHAR(64), AUTHOR_ID VARCHAR(64), AGENT_ID VARCHAR(64), SESSION_ID VARCHAR(128), MEMORY_TYPE VARCHAR(32), CONTENT TEXT, SOURCE VARCHAR(256), VERIFIED SHORT, IMPORTANCE DOUBLE, CONTENT_HASH VARCHAR(64))"); err != nil {
		return fmt.Errorf("create memory table: %w", err)
	}
	if _, err := client.Query(ctx, "CREATE INDEX _NEO_AGENT_MEMORY_CONTENT_IDX ON _NEO_AGENT_MEMORY(CONTENT) INDEX_TYPE KEYWORD"); err != nil {
		return fmt.Errorf("create memory content index: %w", err)
	}
	if _, err := client.Query(ctx, "CREATE INDEX _NEO_AGENT_MEMORY_MEMORY_ID_IDX ON _NEO_AGENT_MEMORY(MEMORY_ID)"); err != nil {
		return fmt.Errorf("create memory id index: %w", err)
	}
	return nil
}

func readMemoryArrivalTime(ctx context.Context, client *Client, memoryID string) (any, error) {
	for attempt := 0; attempt < memoryReadbackAttempts; attempt++ {
		result, err := client.QueryWithOptions(ctx, memoryStoreReadbackSQL, map[string]any{"p": []any{memoryID}})
		if err != nil {
			return nil, fmt.Errorf("read stored memory: %w", err)
		}
		rows, err := memoryRows(result)
		if err != nil {
			return nil, fmt.Errorf("read stored memory: %w", err)
		}
		if len(rows) == 1 {
			return rows[0]["_arrival_time"], nil
		}
		if attempt+1 == memoryReadbackAttempts {
			break
		}
		timer := time.NewTimer(memoryReadbackInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, fmt.Errorf("stored memory %s is not visible after %d attempts", memoryID, memoryReadbackAttempts)
}

func memoryRows(result any) ([]map[string]any, error) {
	envelope, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected memory query response: %T", result)
	}
	data, ok := envelope["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected memory query data: %T", envelope["data"])
	}
	columns, ok := data["columns"].([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected memory query columns: %T", data["columns"])
	}
	rows, ok := data["rows"].([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected memory query rows: %T", data["rows"])
	}
	resultRows := make([]map[string]any, 0, len(rows))
	for _, rawRow := range rows {
		values, ok := rawRow.([]any)
		if !ok || len(values) != len(columns) {
			return nil, fmt.Errorf("unexpected memory row: %T", rawRow)
		}
		row := make(map[string]any, len(columns))
		for i, column := range columns {
			name, ok := column.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected memory column: %T", column)
			}
			row[name] = values[i]
		}
		resultRows = append(resultRows, row)
	}
	return resultRows, nil
}

func memoryArguments(arguments any) (map[string]any, error) {
	values, ok := arguments.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("arguments must be an object")
	}
	return values, nil
}

func requiredString(arguments map[string]any, name string) (string, error) {
	value := strings.TrimSpace(stringValue(arguments[name], ""))
	if value == "" {
		return "", fmt.Errorf("missing required argument %q", name)
	}
	return value, nil
}

func memoryNumberArgument(arguments map[string]any, name string, fallback float64) (float64, error) {
	value, ok := arguments[name]
	if !ok || value == nil {
		return fallback, nil
	}
	switch typed := value.(type) {
	case float64:
		return typed, nil
	case int:
		return float64(typed), nil
	case string:
		parsed, err := strconv.ParseFloat(typed, 64)
		return parsed, err
	default:
		return 0, fmt.Errorf("%s must be a number", name)
	}
}

func validMemoryType(value string) bool {
	switch value {
	case "conversation", "observation", "incident", "procedure", "decision", "result":
		return true
	default:
		return false
	}
}

func newMemoryID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate memory_id: %w", err)
	}
	return "mem-" + hex.EncodeToString(bytes[:]), nil
}
