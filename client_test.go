package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
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

func TestClientQueryWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("q") != "select * from example limit ?" ||
			request.URL.Query().Get("format") != "box" ||
			request.URL.Query().Get("db") != "otherdb" ||
			request.URL.Query().Get("precision") != "2" ||
			request.URL.Query().Get("rownum") != "true" ||
			request.URL.Query().Get("p") != "[10]" {
			t.Fatalf("unexpected query parameters: %s", request.URL.RawQuery)
		}
		writer.Header().Set("Content-Type", "text/plain")
		_, _ = writer.Write([]byte("+-----+\n| 10  |\n+-----+\n"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	result, err := client.QueryWithOptions(context.Background(), "select * from example limit ?", map[string]any{
		"format":    "box",
		"db":        "otherdb",
		"precision": float64(2),
		"rownum":    true,
		"p":         []any{10},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result != "+-----+\n| 10  |\n+-----+\n" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestClientResponseLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/plain")
		_, _ = writer.Write([]byte("1234567890"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	client.SetMaxResponseBytes(5)
	_, err := client.Query(context.Background(), "select 1")
	if err == nil || !strings.Contains(err.Error(), "response exceeds maximum size") {
		t.Fatalf("expected response limit error, got %v", err)
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

func TestClientListDatabasesAndScopedTables(t *testing.T) {
	var scripts []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		scripts = append(scripts, string(body))
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"columns":["NAME"],"rows":[]}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	_, err := client.ListDatabases(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListTablesScoped(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListTablesScoped(context.Background(), "", "sensor")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListTablesScoped(context.Background(), "otherdb", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListTablesScoped(context.Background(), "otherdb", "sensor")
	if err != nil {
		t.Fatal(err)
	}

	if len(scripts) != 5 {
		t.Fatalf("unexpected script count: %d", len(scripts))
	}
	want := []string{
		"SQL(`show databases`)\nJSON()\n",
		"SQL(`show tables`)\nJSON()\n",
		"SQL(`show tables like 'sensor%'`)\nJSON()\n",
		"SQL(`show tables from otherdb`)\nJSON()\n",
		"SQL(`show tables from otherdb like 'sensor%'`)\nJSON()\n",
	}
	for i := range want {
		if scripts[i] != want[i] {
			t.Errorf("script %d: got %q, want %q", i, scripts[i], want[i])
		}
	}
}

func TestClientCurrentSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "SQL(`select current_database(), current_user()`)\nJSON()\n" {
			t.Fatalf("unexpected TQL: %q", body)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"columns":["CURRENT_DATABASE","CURRENT_USER"],"rows":[["MACHBASEDB","sys"]]}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	result, err := client.CurrentSession(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	data := result.(map[string]any)["data"].(map[string]any)
	if data["rows"].([]any)[0].([]any)[0] != "MACHBASEDB" {
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

func TestClientRunTQLPreservesTextOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/plain")
		_, _ = writer.Write([]byte("+-----+\n| 1   |\n+-----+\n"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	result, err := client.RunTQL(context.Background(), "BOX()\n")
	if err != nil {
		t.Fatal(err)
	}
	if result != "+-----+\n| 1   |\n+-----+\n" {
		t.Fatalf("unexpected TQL text result: %#v", result)
	}
}

func TestClientServerFileAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPost && request.URL.Path == "/db/files/hello-world.tql" {
			if request.Header.Get("Authorization") != "Bearer nt_test" {
				t.Fatalf("unexpected authorization: %s", request.Header.Get("Authorization"))
			}
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != "FAKE(linspace(1, 2, 2))\nJSON()\n" {
				t.Fatalf("unexpected written content: %q", body)
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"success":true,"reason":"success"}`))
			return
		}
		if request.URL.Path == "/db/files/" {
			if request.URL.Query().Get("filter") != ".tql" || request.URL.Query().Get("recursive") != "true" {
				t.Fatalf("unexpected file query: %s", request.URL.RawQuery)
			}
			if request.Header.Get("Authorization") != "Bearer nt_test" {
				t.Fatalf("unexpected authorization: %s", request.Header.Get("Authorization"))
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"success":true,"data":{"isDir":true,"name":"work"}}`))
			return
		}
		if request.URL.Path == "/db/files/query.tql" {
			writer.Header().Set("Content-Type", "text/plain")
			_, _ = writer.Write([]byte("FAKE(linspace(1, 2, 2))\nJSON()\n"))
			return
		}
		if request.Method == http.MethodGet && request.URL.Path == "/db/tql/query.tql" {
			if request.Header.Get("Authorization") != "Bearer nt_test" {
				t.Fatalf("unexpected authorization: %s", request.Header.Get("Authorization"))
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"success":true,"data":{"columns":["x"],"rows":[[1],[2]]}}`))
			return
		}
		if request.URL.Path == "/db/tql" {
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != "FAKE(linspace(1, 2, 2))\nJSON()\n" {
				t.Fatalf("unexpected TQL file content: %q", body)
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"success":true,"data":{"columns":["x"],"rows":[[1],[2]]}}`))
			return
		}
		http.NotFound(writer, request)
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	result, err := client.ListFiles(context.Background(), "/project", ".tql", true)
	if err != nil {
		t.Fatal(err)
	}
	if result.(map[string]any)["success"] != true {
		t.Fatalf("unexpected list result: %#v", result)
	}
	content, err := client.ReadFile(context.Background(), "/project/query.tql")
	if err != nil {
		t.Fatal(err)
	}
	if content != "FAKE(linspace(1, 2, 2))\nJSON()\n" {
		t.Fatalf("unexpected file content: %#v", content)
	}
	_, err = client.WriteFile(context.Background(), "/project/hello-world.tql", "FAKE(linspace(1, 2, 2))\nJSON()\n")
	if err != nil {
		t.Fatal(err)
	}
	result, err = client.RunTQLFile(context.Background(), "/project/query.tql")
	if err != nil {
		t.Fatal(err)
	}
	if result.(map[string]any)["success"] != true {
		t.Fatalf("unexpected TQL file result: %#v", result)
	}
	result, contentType, err := client.VerifyTQLFile(context.Background(), "/project/query.tql")
	if err != nil {
		t.Fatal(err)
	}
	if contentType != "application/json" || result.(map[string]any)["success"] != true {
		t.Fatalf("unexpected external TQL result: type=%q result=%#v", contentType, result)
	}
	url, err := client.TQLFileURL("/project/query.tql")
	if err != nil {
		t.Fatal(err)
	}
	if url != server.URL+"/db/tql/query.tql" {
		t.Fatalf("unexpected TQL file URL: %s", url)
	}
}

func TestNormalizeServerFilePath(t *testing.T) {
	path, err := normalizeServerFilePath("work/a file.tql")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/work/a%20file.tql" {
		t.Fatalf("unexpected normalized path: %s", path)
	}
}

func TestNormalizeMCPFilePathMapsProjectNamespace(t *testing.T) {
	path, err := normalizeMCPFilePath("/project/a file.tql")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/a%20file.tql" {
		t.Fatalf("unexpected API path: %s", path)
	}
	if _, err := normalizeMCPFilePath("/work/a.tql"); err == nil {
		t.Fatal("expected internal /work path to be rejected")
	}
}

func TestBrowserTQLURLProxiesMachbaseToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/db/tql/query.tql" || request.URL.Query().Get("x") != "1" {
			t.Fatalf("unexpected proxied request: %s", request.URL.RequestURI())
		}
		if request.Header.Get("Authorization") != "Bearer nt_test" {
			t.Fatalf("unexpected authorization header: %s", request.Header.Get("Authorization"))
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	client.SetDataDir(t.TempDir())
	defer client.Close()
	link, err := client.BrowserTQLFileURL("/project/query.tql")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(link, "nt_test") || !strings.Contains(link, "/db/tql/query.tql") {
		t.Fatalf("unexpected browser link: %s", link)
	}
	request, err := http.NewRequest(http.MethodGet, link+"?x=1", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer browser-token")
	response, err := client.httpClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected proxy status: %d", response.StatusCode)
	}
}

func TestBrowserServerSeparatesMCPFilesFromProxy(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	dataDir := t.TempDir()
	chartDir := dataDir + "/charts"
	if err := os.MkdirAll(chartDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chartDir+"/sample.html", []byte("chart"), 0o644); err != nil {
		t.Fatal(err)
	}
	client := NewClient(server.URL, "nt_test", server.Client())
	client.SetDataDir(dataDir)
	defer client.Close()
	link, err := client.BrowserTQLFileURL("/project/query.tql")
	if err != nil {
		t.Fatal(err)
	}
	base := strings.TrimSuffix(link, "/db/tql/query.tql")
	response, err := http.Get(base + "/mcp/charts/sample.html")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected chart status: %d", response.StatusCode)
	}
	response, err = http.Get(base + "/other")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("unexpected unregistered path status: %d", response.StatusCode)
	}
}

func TestClientRunTQLCancellationClosesRequest(t *testing.T) {
	requestStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	handlerDone := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer close(handlerDone)
		close(requestStarted)
		<-releaseHandler
	}))

	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan error, 1)
	client := NewClient(server.URL, "nt_test", server.Client())
	go func() {
		_, err := client.RunTQL(ctx, "FAKE(linspace(1, 2, 2))\nJSON()\n")
		resultCh <- err
	}()

	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("TQL request was not received")
	}
	cancel()
	select {
	case err := <-resultCh:
		if err == nil {
			t.Fatal("expected TQL cancellation error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("TQL client did not return after cancellation")
	}
	close(releaseHandler)
	select {
	case <-handlerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("TQL fixture handler did not exit")
	}
	server.Close()
}
