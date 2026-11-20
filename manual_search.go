package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/pemistahl/lingua-go"
	potion "github.com/trengrj/go-potion"
)

const (
	defaultManualSearchModel = "RETRIEVAL32M"
	defaultManualSearchTopN  = 3
	maxManualSearchTopN      = 20
	manualVectorDimension    = 512
)

var (
	manualSearchModels = map[string]int{
		"BASE2M":           64,
		"BASE4M":           128,
		"BASE8M":           256,
		"BASE32M":          512,
		"RETRIEVAL32M":     512,
		"SCIENCE32M":       256,
		"CODE16M":          256,
		"CODE16MV2":        256,
		"MULTILINGUAL128M": 256,
	}
	manualSearchModelNames = []string{
		"BASE2M",
		"BASE4M",
		"BASE8M",
		"BASE32M",
		"RETRIEVAL32M",
		"SCIENCE32M",
		"CODE16M",
		"CODE16MV2",
		"MULTILINGUAL128M",
	}
	manualLanguageDetector struct {
		once     sync.Once
		detector lingua.LanguageDetector
	}
)

type manualSearchOptions struct {
	query    string
	topN     int
	model    string
	language string
}

type manualSearchService struct {
	mu       sync.Mutex
	encoders map[string]*potion.Potion
}

func newManualSearchService() *manualSearchService {
	return &manualSearchService{encoders: make(map[string]*potion.Potion)}
}

func parseManualSearchOptions(arguments any) (manualSearchOptions, error) {
	values, ok := arguments.(map[string]any)
	if !ok {
		return manualSearchOptions{}, fmt.Errorf("arguments must be an object")
	}
	query, ok := values["query"].(string)
	if !ok || strings.TrimSpace(query) == "" {
		return manualSearchOptions{}, fmt.Errorf("missing required argument %q", "query")
	}
	options := manualSearchOptions{
		query: strings.TrimSpace(query),
		topN:  defaultManualSearchTopN,
		model: defaultManualSearchModel,
	}
	if raw, exists := values["top_n"]; exists {
		number, ok := raw.(float64)
		if !ok || math.Trunc(number) != number || number < 1 || number > maxManualSearchTopN {
			return manualSearchOptions{}, fmt.Errorf("argument %q must be an integer from 1 to %d", "top_n", maxManualSearchTopN)
		}
		options.topN = int(number)
	}
	if raw, exists := values["model"]; exists {
		model, ok := raw.(string)
		if !ok {
			return manualSearchOptions{}, fmt.Errorf("argument %q must be a string", "model")
		}
		options.model = strings.ToUpper(strings.TrimSpace(model))
	}
	if _, ok := manualSearchModels[options.model]; !ok {
		return manualSearchOptions{}, fmt.Errorf("unsupported embedding model %q", options.model)
	}
	if raw, exists := values["language"]; exists {
		language, ok := raw.(string)
		if !ok {
			return manualSearchOptions{}, fmt.Errorf("argument %q must be a string", "language")
		}
		options.language = strings.ToLower(strings.TrimSpace(language))
		if options.language != "en" && options.language != "kr" && options.language != "ja" {
			return manualSearchOptions{}, fmt.Errorf("argument %q must be one of en, kr, or ja", "language")
		}
	}
	return options, nil
}

func (s *manualSearchService) search(ctx context.Context, client *Client, options manualSearchOptions) (map[string]any, error) {
	encoder, err := s.encoder(ctx, options.model)
	if err != nil {
		return nil, err
	}
	vector, err := encoder.Encode(options.query)
	if err != nil {
		return nil, fmt.Errorf("encode manual search query: %w", err)
	}
	vector, err = padManualSearchEmbedding(vector, manualSearchModels[options.model])
	if err != nil {
		return nil, fmt.Errorf("prepare manual search query embedding: %w", err)
	}
	if vector == nil {
		return nil, fmt.Errorf("model %s produced a zero query embedding", options.model)
	}

	language := options.language
	if language == "" {
		language, err = detectManualSearchLanguage(options.query)
		if err != nil {
			return nil, err
		}
	}
	return searchManualChunks(ctx, client, options.query, options.model, language, options.topN, vector)
}

func (s *manualSearchService) encoder(ctx context.Context, modelName string) (*potion.Potion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if encoder := s.encoders[modelName]; encoder != nil {
		return encoder, nil
	}
	encoder, err := potion.New(ctx, potion.Model(modelName))
	if err != nil {
		return nil, fmt.Errorf("load embedding model %s: %w", modelName, err)
	}
	s.encoders[modelName] = encoder
	return encoder, nil
}

func padManualSearchEmbedding(values []float32, expectedDimension int) ([]float32, error) {
	if expectedDimension <= 0 || expectedDimension > manualVectorDimension {
		return nil, fmt.Errorf("model dimension %d is outside supported range 1..%d", expectedDimension, manualVectorDimension)
	}
	if len(values) != expectedDimension {
		return nil, fmt.Errorf("embedding dimension %d does not match model dimension %d", len(values), expectedDimension)
	}
	nonzero := false
	for index, value := range values {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return nil, fmt.Errorf("embedding element %d is not finite", index)
		}
		nonzero = nonzero || value != 0
	}
	if !nonzero {
		return nil, nil
	}
	padded := make([]float32, manualVectorDimension)
	copy(padded, values)
	return padded, nil
}

func detectManualSearchLanguage(query string) (string, error) {
	manualLanguageDetector.once.Do(func() {
		languages := []lingua.Language{lingua.English, lingua.Korean, lingua.Japanese}
		manualLanguageDetector.detector = lingua.NewLanguageDetectorBuilder().FromLanguages(languages...).Build()
	})
	language, exists := manualLanguageDetector.detector.DetectLanguageOf(query)
	if exists {
		switch language {
		case lingua.English:
			return "en", nil
		case lingua.Korean:
			return "kr", nil
		case lingua.Japanese:
			return "ja", nil
		}
	}
	return "", fmt.Errorf("could not detect manual language; specify language as en, kr, or ja")
}

func searchManualChunks(ctx context.Context, client *Client, query, model, language string, topN int, vector []float32) (map[string]any, error) {
	embeddingJSON, err := json.Marshal(vector)
	if err != nil {
		return nil, fmt.Errorf("marshal query embedding: %w", err)
	}
	const sql = `SELECT CHUNK_ID, CONTENT, URL, SECTIONS, META_JSON,
		VECTOR_DISTANCE(EMBEDDING, TO_VECTOR(?, 512), 'COSINE') AS DISTANCE
		FROM NEO_MCP_DOC_CHUNKS
		WHERE MODEL_NAME = ? AND LANGUAGE = ? AND EMBEDDING IS NOT NULL
		ORDER BY DISTANCE ASC
		LIMIT ?`
	result, err := client.QueryWithOptions(ctx, sql, map[string]any{
		"format": "json",
		"p":      []any{string(embeddingJSON), model, language, topN},
	})
	if err != nil {
		return nil, fmt.Errorf("query manual vector index: %w", err)
	}
	hits, err := parseManualSearchRows(result)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"query":    query,
		"model":    model,
		"language": language,
		"top_n":    topN,
		"results":  hits,
	}, nil
}

func parseManualSearchRows(result any) ([]map[string]any, error) {
	response, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected manual search response: expected an object, got %T", result)
	}
	if success, exists := response["success"].(bool); exists && !success {
		message, _ := response["reason"].(string)
		if message == "" {
			message, _ = response["error"].(string)
		}
		if message == "" {
			message = "database query failed"
		}
		return nil, fmt.Errorf("manual vector search failed: %s", message)
	}
	data, ok := response["data"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected manual search response: missing data object")
	}
	columns, ok := data["columns"].([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected manual search response: missing columns")
	}
	rows, ok := data["rows"].([]any)
	if !ok {
		if data["rows"] == nil {
			return []map[string]any{}, nil
		}
		return nil, fmt.Errorf("unexpected manual search response: rows must be an array")
	}
	hits := make([]map[string]any, 0, len(rows))
	for rowIndex, rawRow := range rows {
		values, ok := rawRow.([]any)
		if !ok || len(values) != len(columns) {
			return nil, fmt.Errorf("unexpected manual search response: row %d does not match columns", rowIndex)
		}
		hit := make(map[string]any, len(columns))
		for columnIndex, rawColumn := range columns {
			column, ok := rawColumn.(string)
			if !ok {
				return nil, fmt.Errorf("unexpected manual search response: column %d is not a string", columnIndex)
			}
			key := strings.ToLower(column)
			if key == "meta_json" {
				key = "metadata"
			}
			hit[key] = values[columnIndex]
		}
		hits = append(hits, hit)
	}
	return hits, nil
}
