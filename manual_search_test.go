package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseManualSearchOptions(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		want    manualSearchOptions
		wantErr string
	}{
		{
			name: "defaults",
			args: map[string]any{"query": "how to configure"},
			want: manualSearchOptions{
				query: "how to configure",
				topN:  defaultManualSearchTopN,
				model: defaultManualSearchModel,
			},
		},
		{
			name: "explicit options",
			args: map[string]any{
				"query":    "설정 방법",
				"top_n":    float64(5),
				"model":    "base2m",
				"language": "KR",
			},
			want: manualSearchOptions{
				query:    "설정 방법",
				topN:     5,
				model:    "BASE2M",
				language: "kr",
			},
		},
		{name: "missing query", args: map[string]any{}, wantErr: `missing required argument "query"`},
		{name: "fractional top_n", args: map[string]any{"query": "x", "top_n": 1.5}, wantErr: `argument "top_n" must be an integer`},
		{name: "top_n over maximum", args: map[string]any{"query": "x", "top_n": float64(maxManualSearchTopN + 1)}, wantErr: `argument "top_n" must be an integer`},
		{name: "unknown model", args: map[string]any{"query": "x", "model": "UNKNOWN"}, wantErr: `unsupported embedding model "UNKNOWN"`},
		{name: "unsupported language", args: map[string]any{"query": "x", "language": "fr"}, wantErr: `argument "language" must be one of en, kr, or ja`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseManualSearchOptions(test.args)
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestPadManualSearchEmbedding(t *testing.T) {
	got, err := padManualSearchEmbedding([]float32{0.25, -0.5}, 2)
	require.NoError(t, err)
	require.Len(t, got, manualVectorDimension)
	require.Equal(t, float32(0.25), got[0])
	require.Equal(t, float32(-0.5), got[1])
	require.Zero(t, got[2])

	_, err = padManualSearchEmbedding([]float32{1, 2}, 3)
	require.ErrorContains(t, err, "does not match model dimension")
}

func TestDetectManualSearchLanguage(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{query: "How do I configure the server?", want: "en"},
		{query: "서버 설정 방법을 알려주세요", want: "kr"},
		{query: "サーバーの設定方法を教えてください", want: "ja"},
	}
	for _, test := range tests {
		got, err := detectManualSearchLanguage(test.query)
		require.NoError(t, err)
		require.Equal(t, test.want, got)
	}
}

func TestSearchManualChunksBindsVectorAndReturnsSourceMetadata(t *testing.T) {
	vector := make([]float32, manualVectorDimension)
	vector[0] = 0.75
	vector[1] = -0.25
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		require.Equal(t, "/db/query", request.URL.Path)
		require.Equal(t, "Bearer nt_test", request.Header.Get("Authorization"))
		sql := request.URL.Query().Get("q")
		require.Contains(t, sql, "VECTOR_DISTANCE(EMBEDDING, TO_VECTOR(?, 512), 'COSINE')")
		require.Contains(t, sql, "MODEL_NAME = ? AND LANGUAGE = ?")
		require.Contains(t, sql, "ORDER BY DISTANCE ASC")
		require.Contains(t, sql, "LIMIT ?")

		var parameters []any
		require.NoError(t, json.Unmarshal([]byte(request.URL.Query().Get("p")), &parameters))
		require.Len(t, parameters, 4)
		var gotVector []float32
		require.NoError(t, json.Unmarshal([]byte(parameters[0].(string)), &gotVector))
		require.Equal(t, vector, gotVector)
		require.Equal(t, "RETRIEVAL32M", parameters[1])
		require.Equal(t, "kr", parameters[2])
		require.Equal(t, float64(3), parameters[3])

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"success":true,"data":{"columns":["CHUNK_ID","CONTENT","URL","SECTIONS","META_JSON","DISTANCE"],"rows":[["neo/guide.kr.md_0","설정 안내","https://docs.machbase.com/neo/guide/","guide",{"H1":"설정","language":"kr"},0.17]]}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	result, err := searchManualChunks(context.Background(), client, "설정 방법", "RETRIEVAL32M", "kr", 3, vector)
	require.NoError(t, err)
	require.Equal(t, "설정 방법", result["query"])
	require.Equal(t, "kr", result["language"])
	hits := result["results"].([]map[string]any)
	require.Len(t, hits, 1)
	require.Equal(t, "https://docs.machbase.com/neo/guide/", hits[0]["url"])
	require.Equal(t, "설정 안내", hits[0]["content"])
	require.Equal(t, map[string]any{"H1": "설정", "language": "kr"}, hits[0]["metadata"])
}

func TestParseManualSearchRowsReportsDatabaseAndShapeErrors(t *testing.T) {
	_, err := parseManualSearchRows(map[string]any{"success": false, "reason": "table does not exist"})
	require.ErrorContains(t, err, "table does not exist")

	_, err = parseManualSearchRows(map[string]any{"success": true, "data": map[string]any{"columns": []any{"CHUNK_ID"}, "rows": []any{[]any{"id", "extra"}}}})
	require.ErrorContains(t, err, "row 0 does not match columns")

	_, err = parseManualSearchRows("unexpected")
	require.ErrorContains(t, err, "expected an object")
}
