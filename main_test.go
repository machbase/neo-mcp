package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMCPToolAnnotationsAndTitles(t *testing.T) {
	mcpServer := NewMCPServer(nil, nil)

	tests := []struct {
		name        string
		title       string
		readOnly    bool
		destructive bool
		idempotent  bool
	}{
		{"db_query", "Machbase SQL query", false, true, false},
		{"tql_run", "Run Machbase TQL", false, true, false},
		{"tql_run_file", "Run server TQL file", false, true, false},
		{"tql_file_link", "Verify and link server TQL file", false, true, false},
		{"jsh_exec", "Run Machbase JSH script", false, true, false},
		{"jsh_run_file", "Run server JSH file", false, true, false},
		{"render_markdown", "Render and Execute Markdown", false, true, false},
		{"db_list_tables", "List Machbase tables", true, false, true},
		{"server_info", "Machbase server info", true, false, true},
		{"fs_list", "List server files", true, false, true},
		{"fs_read", "Read server file", true, false, true},
		{"fs_write", "Write server file", false, true, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tool := mcpServer.GetTool(test.name)
			require.NotNil(t, tool)
			require.Equal(t, test.title, tool.Tool.Annotations.Title)
			require.NotNil(t, tool.Tool.Annotations.ReadOnlyHint)
			require.Equal(t, test.readOnly, *tool.Tool.Annotations.ReadOnlyHint)
			require.NotNil(t, tool.Tool.Annotations.DestructiveHint)
			require.Equal(t, test.destructive, *tool.Tool.Annotations.DestructiveHint)
			require.NotNil(t, tool.Tool.Annotations.IdempotentHint)
			require.Equal(t, test.idempotent, *tool.Tool.Annotations.IdempotentHint)
		})
	}
}
