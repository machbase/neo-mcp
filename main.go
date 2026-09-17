package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	flags := flag.NewFlagSet("neo-mcp", flag.ExitOnError)
	serverURL := flags.String("server", "http://127.0.0.1:5654", "machbase-neo HTTP server URL")
	token := flags.String("token", "", "machbase-neo API token (required)")
	maxResponseBytes := flags.String("max-response-bytes", formatByteSize(DefaultMaxResponseBytes), "maximum HTTP response size (bytes or KB/MB/GB)")
	dataDir := flags.String("data-dir", defaultDataDir(), "root directory for generated neo-mcp artifacts")
	_ = flags.Parse(os.Args[1:])
	apiToken := resolveAPIToken(*token, os.Getenv("NEO_MCP_TOKEN"))
	if apiToken == "" {
		fmt.Fprintln(os.Stderr, "neo-mcp: provide -token or NEO_MCP_TOKEN")
		os.Exit(2)
	}
	maxResponseBytesValue, err := parseByteSize(*maxResponseBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "neo-mcp: invalid -max-response-bytes: %v\n", err)
		os.Exit(2)
	}

	client := NewClient(*serverURL, apiToken, http.DefaultClient)
	defer client.Close()
	client.SetMaxResponseBytes(maxResponseBytesValue)
	client.SetDataDir(*dataDir)
	// The SSH (shell) service address is never configured directly; it is
	// discovered from the machbase-neo server via the service.port.list API.
	mcpServer := NewMCPServer(client, NewSSHClient(client, apiToken))
	if err := server.ServeStdio(mcpServer, server.WithErrorLogger(log.New(os.Stderr, "neo-mcp: ", log.LstdFlags))); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseByteSize(value string) (int64, error) {
	raw := strings.ToUpper(strings.TrimSpace(value))
	multiplier := int64(1)
	for _, unit := range []struct {
		suffix string
		value  int64
	}{
		{"GIB", 1024 * 1024 * 1024},
		{"GB", 1024 * 1024 * 1024},
		{"MIB", 1024 * 1024},
		{"MB", 1024 * 1024},
		{"KIB", 1024},
		{"KB", 1024},
		{"B", 1},
	} {
		if strings.HasSuffix(raw, unit.suffix) {
			raw = strings.TrimSpace(strings.TrimSuffix(raw, unit.suffix))
			multiplier = unit.value
			break
		}
	}
	if raw == "" {
		return 0, fmt.Errorf("size is empty")
	}
	number, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || number <= 0 {
		return 0, fmt.Errorf("expected a positive integer with optional B, KB, MB, or GB suffix")
	}
	if number > int64(^uint64(0)>>1)/multiplier {
		return 0, fmt.Errorf("size is too large")
	}
	return number * multiplier, nil
}

func formatByteSize(value int64) string {
	return strconv.FormatInt(value, 10)
}

func resolveAPIToken(flagValue, environmentValue string) string {
	if token := strings.TrimSpace(flagValue); token != "" {
		return token
	}
	return strings.TrimSpace(environmentValue)
}

func NewMCPServer(client *Client, sshClient *SSHClient) *server.MCPServer {
	mcpServer := server.NewMCPServer("neo-mcp", "0.1.0", server.WithInstructions(
		"Before authoring a query or script, call manual_read with the relevant manual URI. Use the MCP /project namespace for server files: fs_write('/project/example.tql', ...), then tql_run_file or tql_file_link; for JavaScript use jsh_run_file('/project/example.js'). Do not pass internal /work paths. For database context, read neo://machbase/session for the current logical database and user, neo://machbase/databases for logical and mounted databases, and the neo://machbase/tables resources for table metadata. Table URIs support current-database prefixes and logical database selection. For TQL use tql_run. For JSH use jsh_exec, jsh_run_file, or jsh_run_command. For timer/subscriber/API token management use neo://manual/server and the timer_*/subscriber_*/token_* tools. Do not use terminal commands, curl, direct HTTP calls, or direct SSH calls instead.",
	))
	registerManualResources(mcpServer)
	registerDatabaseResources(mcpServer, client)
	registerManualTool(mcpServer)
	registerFileTools(mcpServer, client)
	registerReadOnlyTools(mcpServer, client)
	registerRpcTools(mcpServer, client)
	registerJSHTools(mcpServer, sshClient)
	return mcpServer
}

func requireArgument(requestArguments any, name string) (string, error) {
	arguments, ok := requestArguments.(map[string]any)
	if !ok {
		return "", fmt.Errorf("arguments must be an object")
	}
	value, ok := arguments[name].(string)
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("missing required argument %q", name)
	}
	return value, nil
}
