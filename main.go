package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	flags := flag.NewFlagSet("neo-mcp", flag.ExitOnError)
	serverURL := flags.String("server", "http://127.0.0.1:5654", "machbase-neo HTTP server URL")
	token := flags.String("token", "", "machbase-neo API token (required)")
	_ = flags.Parse(os.Args[1:])
	if strings.TrimSpace(*token) == "" {
		fmt.Fprintln(os.Stderr, "neo-mcp: -token is required")
		os.Exit(2)
	}

	client := NewClient(*serverURL, *token, http.DefaultClient)
	// The SSH (shell) service address is never configured directly; it is
	// discovered from the machbase-neo server via the service.port.list API.
	mcpServer := NewMCPServer(client, NewSSHClient(client, *token))
	if err := server.ServeStdio(mcpServer, server.WithErrorLogger(log.New(os.Stderr, "neo-mcp: ", log.LstdFlags))); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func NewMCPServer(client *Client, sshClient *SSHClient) *server.MCPServer {
	mcpServer := server.NewMCPServer("neo-mcp", "0.1.0", server.WithInstructions(
		"Before authoring a query or script, call manual_read with the relevant manual URI. For TQL use tql_run. For JSH use jsh_exec or jsh_run_command. For timer/subscriber/API token management use neo://manual/server and the timer_*/subscriber_*/token_* tools. Do not use terminal commands, curl, direct HTTP calls, or direct SSH calls instead.",
	))
	registerManualResources(mcpServer)
	registerManualTool(mcpServer)
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
