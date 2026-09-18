package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadManualSupportsRootAndNestedURI(t *testing.T) {
	root, err := readManual("neo://manual/tql")
	if err != nil {
		t.Fatal(err)
	}
	if len(root) == 0 {
		t.Fatal("expected TQL manual content")
	}

	sql, err := readManual("neo://manual/sql")
	if err != nil {
		t.Fatal(err)
	}
	if len(sql) == 0 {
		t.Fatal("expected SQL manual content")
	}
	if !strings.Contains(sql, "Each `CREATE TAG TABLE` consumes TAG cache memory") {
		t.Fatal("expected SQL manual to document TAG cache lifecycle")
	}

	machcli, err := readManual("neo://manual/jsh/machcli")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(machcli, "MACHCLI-ERR-1423, TAG cache exhausted") {
		t.Fatal("expected machcli manual to document TAG cache exhaustion")
	}

	nested, err := readManual("neo://manual/tql/overview")
	if err != nil {
		t.Fatal(err)
	}
	if len(nested) == 0 {
		t.Fatal("expected nested TQL manual content")
	}
}

func TestReadManualRejectsTraversal(t *testing.T) {
	if _, err := readManual("neo://manual/../go.mod"); err == nil {
		t.Fatal("expected traversal URI to be rejected")
	}
}

func TestTableListResourceParts(t *testing.T) {
	tests := []struct {
		uri         string
		hasDatabase bool
		database    string
		prefix      string
	}{
		{"neo://machbase/tables", false, "", ""},
		{"neo://machbase/tables/sensor", false, "", "sensor"},
		{"neo://machbase/tables/otherdb", true, "otherdb", ""},
		{"neo://machbase/tables/otherdb/sensor", true, "otherdb", "sensor"},
	}

	for _, test := range tests {
		database, prefix, err := tableListResourceParts(test.uri, test.hasDatabase)
		require.NoError(t, err, test.uri)
		require.Equal(t, test.database, database, test.uri)
		require.Equal(t, test.prefix, prefix, test.uri)
	}
}

func TestTableResourcePart(t *testing.T) {
	table, err := tableResourcePart("neo://machbase/table/sensor_data")
	require.NoError(t, err)
	require.Equal(t, "sensor_data", table)

	_, err = tableResourcePart("neo://machbase/table/otherdb/sensor_data")
	require.Error(t, err)
}
