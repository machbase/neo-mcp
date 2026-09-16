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
	pathpkg "path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var tableIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$]*$`)

const DefaultMaxResponseBytes int64 = 2 * 1024 * 1024

type Client struct {
	baseURL          string
	token            string
	httpClient       *http.Client
	workspaceDir     string
	maxResponseBytes int64
}

func NewClient(baseURL, token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		baseURL:          strings.TrimRight(baseURL, "/"),
		token:            token,
		httpClient:       httpClient,
		workspaceDir:     currentWorkspaceDir(),
		maxResponseBytes: DefaultMaxResponseBytes,
	}
}

func (c *Client) SetMaxResponseBytes(size int64) {
	if size > 0 {
		c.maxResponseBytes = size
	}
}

func (c *Client) readResponseBody(reader io.Reader) ([]byte, error) {
	limit := c.maxResponseBytes
	if limit <= 0 {
		limit = DefaultMaxResponseBytes
	}
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("response exceeds maximum size of %d bytes", limit)
	}
	return body, nil
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
	responseBody, err := c.readResponseBody(response.Body)
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
	return c.QueryWithOptions(ctx, sql, nil)
}

func (c *Client) QueryWithOptions(ctx context.Context, sql string, options map[string]any) (any, error) {
	query := url.Values{"q": []string{sql}}
	for key, value := range options {
		if value == nil {
			continue
		}
		if key == "p" {
			encoded, err := json.Marshal(value)
			if err != nil {
				return nil, fmt.Errorf("invalid query parameters: %w", err)
			}
			query.Set(key, string(encoded))
			continue
		}
		switch typed := value.(type) {
		case string:
			if typed != "" {
				query.Set(key, typed)
			}
		case bool:
			query.Set(key, strconv.FormatBool(typed))
		case float64:
			query.Set(key, strconv.FormatFloat(typed, 'f', -1, 64))
		default:
			return nil, fmt.Errorf("unsupported query option %q", key)
		}
	}
	result, _, err := c.doJSON(ctx, http.MethodGet, "/db/query", query, nil)
	return result, err
}

func (c *Client) RunTQL(ctx context.Context, script string) (any, error) {
	return c.doTQL(ctx, script)
}

func (c *Client) ListFiles(ctx context.Context, remotePath, filter string, recursive bool) (any, error) {
	path, err := normalizeServerFilePath(remotePath)
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	if filter != "" {
		query.Set("filter", filter)
	}
	if recursive {
		query.Set("recursive", "true")
	}
	result, _, err := c.doJSON(ctx, http.MethodGet, "/db/files"+path, query, nil)
	return result, err
}

func (c *Client) ReadFile(ctx context.Context, remotePath string) (any, error) {
	path, err := normalizeServerFilePath(remotePath)
	if err != nil {
		return nil, err
	}
	result, _, err := c.doJSON(ctx, http.MethodGet, "/db/files"+path, nil, nil)
	return result, err
}

func (c *Client) WriteFile(ctx context.Context, remotePath, content string) (any, error) {
	path, err := normalizeServerFilePath(remotePath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/db/files"+path, strings.NewReader(content))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "text/plain")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	responseBody, err := c.readResponseBody(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("machbase API returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	if json.Valid(responseBody) {
		var result any
		if err := json.Unmarshal(responseBody, &result); err != nil {
			return nil, err
		}
		return result, nil
	}
	return string(responseBody), nil
}

func (c *Client) RunTQLFile(ctx context.Context, remotePath string) (any, error) {
	content, err := c.ReadFile(ctx, remotePath)
	if err != nil {
		return nil, err
	}
	script, ok := content.(string)
	if !ok {
		return nil, fmt.Errorf("server file %q did not return text content", remotePath)
	}
	if !strings.HasSuffix(strings.ToLower(remotePath), ".tql") {
		return nil, fmt.Errorf("TQL file must have .tql extension: %q", remotePath)
	}
	return c.RunTQL(ctx, script)
}

func normalizeServerFilePath(rawPath string) (string, error) {
	if strings.ContainsRune(rawPath, 0) {
		return "", fmt.Errorf("invalid server file path")
	}
	cleanPath := pathpkg.Clean("/" + strings.TrimSpace(rawPath))
	if cleanPath == "/" {
		return cleanPath, nil
	}
	parts := strings.Split(strings.TrimPrefix(cleanPath, "/"), "/")
	escaped := make([]string, len(parts))
	for i, part := range parts {
		escaped[i] = url.PathEscape(part)
	}
	return "/" + strings.Join(escaped, "/"), nil
}

func (c *Client) ListDatabases(ctx context.Context) (any, error) {
	return c.RunTQL(ctx, "SQL(`show databases`)\nJSON()\n")
}

func (c *Client) CurrentSession(ctx context.Context) (any, error) {
	return c.RunTQL(ctx, "SQL(`select current_database(), current_user()`)\nJSON()\n")
}

func (c *Client) ListTablesScoped(ctx context.Context, database, prefix string) (any, error) {
	if database != "" && !tableIdentifierPattern.MatchString(database) {
		return nil, fmt.Errorf("invalid database name %q", database)
	}
	query := "show tables"
	if database != "" {
		query += " from " + database
	}
	if prefix != "" {
		query += fmt.Sprintf(" like '%s%%'", escapeSQLString(prefix))
	}
	return c.RunTQL(ctx, "SQL(`"+query+"`)\nJSON()\n")
}

func (c *Client) DescribeTableTQL(ctx context.Context, table string) (any, error) {
	if !tableIdentifierPattern.MatchString(table) {
		return nil, fmt.Errorf("invalid table name %q", table)
	}
	query := fmt.Sprintf("show table %s", table)
	return c.RunTQL(ctx, "SQL(`"+query+"`)\nJSON()\n")
}

func escapeSQLString(value string) string {
	return strings.ReplaceAll(value, "'", "''")
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
	body, err := c.readResponseBody(response.Body)
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
	responseBody, err := c.readResponseBody(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("machbase API returned HTTP %d", response.StatusCode)
	}

	if json.Valid(responseBody) {
		var result any
		if err := json.Unmarshal(responseBody, &result); err != nil {
			return nil, err
		}
		return result, nil
	}
	return string(responseBody), nil
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
