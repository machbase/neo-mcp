package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var tableIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$]*$`)

type Client struct {
	baseURL      string
	token        string
	httpClient   *http.Client
	workspaceDir string
}

func NewClient(baseURL, token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		token:        token,
		httpClient:   httpClient,
		workspaceDir: currentWorkspaceDir(),
	}
}

func currentWorkspaceDir() string {
	workspaceDir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return workspaceDir
}

func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, body any) (any, string, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, "", err
		}
		reader = bytes.NewReader(payload)
	}

	target := c.baseURL + path
	if encoded := query.Encode(); encoded != "" {
		target += "?" + encoded
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return nil, "", err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, "", err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, string(responseBody), fmt.Errorf("machbase API returned HTTP %d", response.StatusCode)
	}

	contentType := response.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") || json.Valid(responseBody) {
		var result any
		if err := json.Unmarshal(responseBody, &result); err != nil {
			return nil, string(responseBody), err
		}
		return result, contentType, nil
	}
	return string(responseBody), contentType, nil
}

func (c *Client) Query(ctx context.Context, sql string) (any, error) {
	result, _, err := c.doJSON(ctx, http.MethodGet, "/db/query", url.Values{"q": []string{sql}}, nil)
	return result, err
}

func (c *Client) RunTQL(ctx context.Context, script string) (any, error) {
	return c.doTQL(ctx, script)
}

func (c *Client) WriteChartHTML(ctx context.Context, chart map[string]any) (map[string]any, error) {
	chartID, ok := chart["chartID"].(string)
	if !ok || !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(chartID) {
		return nil, fmt.Errorf("chart response has invalid chartID")
	}
	jsAssets, err := stringSlice(chart["jsAssets"])
	if err != nil {
		return nil, fmt.Errorf("chart response jsAssets: %w", err)
	}
	jsCodeAssets, err := stringSlice(chart["jsCodeAssets"])
	if err != nil {
		return nil, fmt.Errorf("chart response jsCodeAssets: %w", err)
	}

	var scripts strings.Builder
	for _, asset := range jsAssets {
		assetURL, err := c.assetURL(asset)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&scripts, "<script src=\"%s\"></script>\n", html.EscapeString(assetURL))
	}
	for _, asset := range jsCodeAssets {
		content, err := c.fetchAsset(ctx, asset)
		if err != nil {
			return nil, err
		}
		scripts.WriteString("<script>\n")
		scripts.WriteString(strings.ReplaceAll(content, "</script", "<\\/script"))
		scripts.WriteString("\n</script>\n")
	}

	style := mapValue(chart["style"])
	width := stringValue(style["width"], "600px")
	height := stringValue(style["height"], "400px")
	theme := stringValue(chart["theme"], "white")
	html := fmt.Sprintf(`<!doctype html>
<html><head><meta charset="utf-8"><title>Machbase TQL chart %s</title>
<style>html,body,#%s{margin:0;width:100%%;height:100%%;}#%s{width:%s;height:%s;}</style>
</head><body><div id="%s" data-theme="%s"></div>%s</body></html>
`, chartID, chartID, chartID, width, height, chartID, theme, scripts.String())

	chartDir := filepath.Join(c.workspaceDir, ".neo-mcp", "charts")
	if err := os.MkdirAll(chartDir, 0o755); err != nil {
		return nil, fmt.Errorf("create chart directory: %w", err)
	}
	chartPath := filepath.Join(chartDir, chartID+".html")
	if err := os.WriteFile(chartPath, []byte(html), 0o644); err != nil {
		return nil, fmt.Errorf("write chart HTML: %w", err)
	}
	relativePath, err := filepath.Rel(c.workspaceDir, chartPath)
	if err != nil {
		return nil, fmt.Errorf("resolve chart link: %w", err)
	}
	return map[string]any{
		"type":    "chart",
		"chartID": chartID,
		"file":    filepath.ToSlash(relativePath),
		"link":    "./" + filepath.ToSlash(relativePath),
		"width":   width,
		"height":  height,
	}, nil
}

func (c *Client) fetchAsset(ctx context.Context, asset string) (string, error) {
	assetURL, err := c.assetURL(asset)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		return "", err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	response, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("chart asset %s returned HTTP %d", asset, response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *Client) assetURL(asset string) (string, error) {
	parsed, err := url.Parse(asset)
	if err != nil || parsed.IsAbs() || parsed.Host != "" {
		return "", fmt.Errorf("chart asset must be a relative server path: %q", asset)
	}
	assetPath := asset
	if !strings.HasPrefix(assetPath, "/") {
		assetPath = "/" + assetPath
	}
	return c.baseURL + assetPath, nil
}

func stringSlice(value any) ([]string, error) {
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("expected string array")
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("expected string asset")
		}
		result = append(result, text)
	}
	return result, nil
}

func mapValue(value any) map[string]any {
	if result, ok := value.(map[string]any); ok {
		return result
	}
	return map[string]any{}
}

func stringValue(value any, fallback string) string {
	if text, ok := value.(string); ok && text != "" {
		return text
	}
	return fallback
}

func (c *Client) ListTables(ctx context.Context) (any, error) {
	return c.doTQL(ctx, "SQL(`show tables`)\nJSON()\n")
}

func (c *Client) doTQL(ctx context.Context, script string) (any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/db/tql", strings.NewReader(script))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Tql-Output", "json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("machbase API returned HTTP %d", response.StatusCode)
	}

	var result any
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) DescribeTable(ctx context.Context, table string) (any, error) {
	if !tableIdentifierPattern.MatchString(table) {
		return nil, fmt.Errorf("invalid table name %q", table)
	}
	return c.Query(ctx, "DESC "+table)
}

func (c *Client) ListTags(ctx context.Context, table string) (any, error) {
	result, _, err := c.doJSON(ctx, http.MethodGet, "/web/api/tables/"+url.PathEscape(table)+"/tags", nil, nil)
	return result, err
}

func (c *Client) TagStat(ctx context.Context, table, tag string) (any, error) {
	result, _, err := c.doJSON(ctx, http.MethodGet, "/web/api/tables/"+url.PathEscape(table)+"/tags/"+url.PathEscape(tag)+"/stat", nil, nil)
	return result, err
}

// ServicePort describes a machbase-neo listener address as reported by the
// "service.port.list" JSON-RPC method.
type ServicePort struct {
	Service string `json:"Service"`
	Address string `json:"Address"`
}

// CallDbRpc invokes a JSON-RPC method through the API-token authenticated
// /db/rpc endpoint. Only methods allowlisted server-side in
// dbRpcAllowedMethods (neo-server/mods/server/http.go) are reachable; all
// other methods return a JSON-RPC "Method not found" error.
func (c *Client) CallDbRpc(ctx context.Context, method string, params []any) (any, error) {
	payload := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}
	result, _, err := c.doJSON(ctx, http.MethodPost, "/db/rpc", nil, payload)
	if err != nil {
		return nil, fmt.Errorf("%s request failed: %w", method, err)
	}
	response, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s returned an unexpected response", method)
	}
	if rpcErr, hasErr := response["error"]; hasErr && rpcErr != nil {
		return nil, fmt.Errorf("%s failed: %v", method, rpcErr)
	}
	return response["result"], nil
}

// ServicePorts discovers listener addresses for the given service name (e.g.
// "shell" for the SSH service) by calling the machbase-neo JSON-RPC endpoint.
// An empty svc returns addresses for all services. Callers must not hardcode
// or configure service addresses separately; this API is the source of truth.
func (c *Client) ServicePorts(ctx context.Context, svc string) ([]ServicePort, error) {
	result, err := c.CallDbRpc(ctx, "service.port.list", []any{svc})
	if err != nil {
		return nil, err
	}
	items, ok := result.([]any)
	if !ok {
		return nil, fmt.Errorf("service.port.list did not return a result array")
	}
	ports := make([]ServicePort, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ports = append(ports, ServicePort{
			Service: stringValue(entry["Service"], ""),
			Address: stringValue(entry["Address"], ""),
		})
	}
	return ports, nil
}
