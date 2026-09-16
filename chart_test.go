package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteChartHTMLCreatesWorkspaceLink(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/web/echarts/echarts.min.js":
			_, _ = writer.Write([]byte("window.echarts = {};"))
		case "/web/api/tql-assets/chart.js":
			_, _ = writer.Write([]byte("window.chartLoaded = true;"))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	workspaceDir := t.TempDir()
	client := NewClient(server.URL, "nt_test", server.Client())
	client.workspaceDir = workspaceDir
	result, err := client.WriteChartHTML(context.Background(), map[string]any{
		"chartID":      "chart_test",
		"jsAssets":     []any{"/web/echarts/echarts.min.js"},
		"jsCodeAssets": []any{"/web/api/tql-assets/chart.js"},
		"style":        map[string]any{"width": "600px", "height": "400px"},
		"theme":        "white",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result["link"] != "./.neo-mcp/charts/chart_test.html" {
		t.Fatalf("unexpected chart link: %#v", result["link"])
	}
	path := filepath.Join(workspaceDir, ".neo-mcp", "charts", "chart_test.html")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, `id="chart_test"`) || !strings.Contains(text, `#chart_test{width:600px;height:400px;}`) || !strings.Contains(text, `<script src="`+server.URL+`/web/echarts/echarts.min.js"></script>`) || !strings.Contains(text, "window.chartLoaded = true;") || strings.Contains(text, "tql-assets/chart.js") {
		t.Fatalf("chart assets were not embedded: %s", text)
	}
}

func TestRunTQLChartSmokeScript(t *testing.T) {
	const script = `SCRIPT({
    for (x = 0; x < 360; x += 3.6) {
        $.yield(x, Math.sin(x / 180 * Math.PI));
    }
})
CHART(
    chartOption({
        xAxis: { type: "category", data: column(0) },
        yAxis: {},
        series: [{ type: "line", data: column(1) }]
    })
)`

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path == "/db/tql" {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = writer.Write([]byte(`{"chartID":"chart_smoke","jsAssets":["/web/echarts/echarts.min.js"],"jsCodeAssets":["/web/api/tql-assets/chart_smoke.js"],"style":{"width":"600px","height":"400px"},"theme":"white"}`))
			return
		}
		if request.URL.Path == "/web/echarts/echarts.min.js" || request.URL.Path == "/web/api/tql-assets/chart_smoke.js" {
			_, _ = writer.Write([]byte("window.chartSmoke = true;"))
			return
		}
		http.NotFound(writer, request)
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	result, err := client.RunTQL(context.Background(), script)
	if err != nil {
		t.Fatal(err)
	}
	chart, ok := result.(map[string]any)
	if !ok || chart["chartID"] != "chart_smoke" {
		t.Fatalf("unexpected chart result: %#v", result)
	}
	client.workspaceDir = t.TempDir()
	fileResult, err := client.WriteChartHTML(context.Background(), chart)
	if err != nil {
		t.Fatal(err)
	}
	if fileResult["link"] != "./.neo-mcp/charts/chart_smoke.html" {
		t.Fatalf("unexpected chart file result: %#v", fileResult)
	}
}
