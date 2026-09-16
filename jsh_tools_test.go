package main

import (
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestJSHApplicationResultPreservesStdoutStderrAndExitCode(t *testing.T) {
	result, err := jshToolResult(JSHResult{
		Stdout:   "rows: 1000\n",
		Stderr:   "warning: one skipped\n",
		ExitCode: 0,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Content) != 1 {
		t.Fatalf("unexpected content count: %d", len(result.Content))
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("unexpected content: %#v", result.Content[0])
	}
	if !strings.Contains(text.Text, `"stdout":"rows: 1000\n"`) ||
		!strings.Contains(text.Text, `"stderr":"warning: one skipped\n"`) ||
		!strings.Contains(text.Text, `"exitCode":0`) {
		t.Fatalf("JSH result lost process fields: %s", text.Text)
	}
}

func TestJSHApplicationFailureIncludesOutput(t *testing.T) {
	result, err := jshToolResult(JSHResult{
		Stdout:   "processed: 10\n",
		Stderr:   "database error\n",
		ExitCode: 1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok || !strings.Contains(text.Text, `"exitCode":1`) || !strings.Contains(text.Text, "database error") {
		t.Fatalf("JSH failure result lost diagnostics: %#v", result.Content)
	}
}
