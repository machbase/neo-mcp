package main

import "testing"

func TestReadManualSupportsRootAndNestedURI(t *testing.T) {
	root, err := readManual("neo://manual/tql")
	if err != nil {
		t.Fatal(err)
	}
	if len(root) == 0 {
		t.Fatal("expected TQL manual content")
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
