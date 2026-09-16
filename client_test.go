package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientQueryUsesBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/db/query" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer nt_test" {
			t.Fatalf("unexpected authorization header: %s", request.Header.Get("Authorization"))
		}
		if request.URL.Query().Get("q") != "select 1" {
			t.Fatalf("unexpected query: %s", request.URL.Query().Get("q"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"columns":["x"],"rows":[[1]]}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	result, err := client.Query(context.Background(), "select 1")
	if err != nil {
		t.Fatal(err)
	}
	response, ok := result.(map[string]any)
	if !ok || response["success"] != true {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestClientListTables(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/db/tql" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer nt_test" {
			t.Fatalf("unexpected authorization header: %s", request.Header.Get("Authorization"))
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "SQL(`show tables`)\nJSON()\n" {
			t.Fatalf("unexpected TQL: %q", body)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"columns":["NAME"],"rows":[["sensor_data"]]}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	result, err := client.ListTables(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	data := result.(map[string]any)["data"].(map[string]any)
	if data["rows"].([]any)[0].([]any)[0] != "sensor_data" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestClientServicePortsUsesRpcAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/db/rpc" {
			t.Fatalf("unexpected path: %s %s", request.Method, request.URL.Path)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), `"method":"service.port.list"`) || !strings.Contains(string(body), `"params":["shell"]`) {
			t.Fatalf("unexpected rpc payload: %s", body)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"result":[{"Service":"shell","Address":"tcp://127.0.0.1:5652"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	ports, err := client.ServicePorts(context.Background(), "shell")
	if err != nil {
		t.Fatal(err)
	}
	if len(ports) != 1 || ports[0].Service != "shell" || ports[0].Address != "tcp://127.0.0.1:5652" {
		t.Fatalf("unexpected ports: %#v", ports)
	}
}

func TestClientDescribeTableUsesQueryAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/db/query" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("q") != "DESC sensor_data" {
			t.Fatalf("unexpected query: %s", request.URL.Query().Get("q"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	if _, err := client.DescribeTable(context.Background(), "sensor_data"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.DescribeTable(context.Background(), "sensor_data; DROP TABLE x"); err == nil {
		t.Fatal("expected invalid table name error")
	}
}

func TestClientRunTQL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/db/tql" || request.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer nt_test" {
			t.Fatalf("unexpected authorization header: %s", request.Header.Get("Authorization"))
		}
		if request.Header.Get("X-Tql-Output") != "json" {
			t.Fatalf("unexpected TQL output header: %s", request.Header.Get("X-Tql-Output"))
		}
		body := make([]byte, request.ContentLength)
		_, _ = request.Body.Read(body)
		if string(body) != "FAKE(linspace(1, 2, 2))\nJSON()\n" {
			t.Fatalf("unexpected TQL script: %q", body)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"columns":["x"],"rows":[[1],[2]]}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	result, err := client.RunTQL(context.Background(), "FAKE(linspace(1, 2, 2))\nJSON()\n")
	if err != nil {
		t.Fatal(err)
	}
	if result.(map[string]any)["success"] != true {
		t.Fatalf("unexpected result: %#v", result)
	}
}
