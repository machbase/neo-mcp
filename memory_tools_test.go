package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMemoryStoreReadbackIncludesArrivalTimeAndColumns(t *testing.T) {
	if !strings.HasPrefix(memoryStoreReadbackSQL, "SELECT _arrival_time, * FROM ") {
		t.Fatalf("memory store readback must explicitly select _arrival_time and table columns: %s", memoryStoreReadbackSQL)
	}
}

func TestEnsureMemorySchemaProbesHiddenLogTable(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
		if request.Method != http.MethodPost || request.URL.Path != "/db/tql" {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != memorySchemaProbeTQL {
			t.Fatalf("unexpected schema probe: %s", body)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"columns":["TABLE_NAME","TABLE_TYPE"],"rows":[["_NEO_AGENT_MEMORY","Log"]]}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	if err := ensureMemorySchema(context.Background(), client, true); err != nil {
		t.Fatal(err)
	}
	if requestCount != 1 {
		t.Fatalf("unexpected request count: %d", requestCount)
	}
}

func TestEnsureMemorySchemaDoesNotCreateWhenListingFails(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
		http.Error(writer, "list failed", http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	err := ensureMemorySchema(context.Background(), client, true)
	if err == nil || !strings.Contains(err.Error(), "list memory tables") {
		t.Fatalf("expected table listing error, got %v", err)
	}
	if requestCount != 1 {
		t.Fatalf("schema creation must not run after a listing failure; requests: %d", requestCount)
	}
}

func TestReadMemoryArrivalTimeRetriesUntilVisible(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestCount++
		writer.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			_, _ = writer.Write([]byte(`{"success":true,"data":{"columns":["_arrival_time"],"rows":[]}}`))
			return
		}
		_, _ = writer.Write([]byte(`{"success":true,"data":{"columns":["_arrival_time"],"rows":[[1789706028731817500]]}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	arrivalTime, err := readMemoryArrivalTime(context.Background(), client, "mem-test")
	if err != nil {
		t.Fatal(err)
	}
	if arrivalTime == nil {
		t.Fatal("expected arrival time")
	}
	if requestCount != 2 {
		t.Fatalf("unexpected request count: %d", requestCount)
	}
}
