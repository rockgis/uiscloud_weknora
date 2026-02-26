package types

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// WebSearchConfig represents the web search configuration for a tenant
type WebSearchConfig struct {
	Provider          string   `json:"provider"`
	APIKey            string   `json:"api_key"`
	MaxResults        int      `json:"max_results"`
	IncludeDate       bool     `json:"include_date"`
	CompressionMethod string   `json:"compression_method"`
	Blacklist         []string `json:"blacklist"`
	EmbeddingModelID   string `json:"embedding_model_id,omitempty"`
	EmbeddingDimension int    `json:"embedding_dimension,omitempty"`
	RerankModelID      string `json:"rerank_model_id,omitempty"`
	DocumentFragments  int    `json:"document_fragments,omitempty"`
}

// Value implements driver.Valuer interface for WebSearchConfig
func (c WebSearchConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements sql.Scanner interface for WebSearchConfig
func (c *WebSearchConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(b, c)
}

// WebSearchResult represents a single web search result
type WebSearchResult struct {
	Title       string     `json:"title"`
	URL         string     `json:"url"`
	Snippet     string     `json:"snippet"`
	Content     string     `json:"content"`
	Source      string     `json:"source"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
}

// WebSearchProviderInfo represents information about a web search provider
type WebSearchProviderInfo struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Free           bool   `json:"free"`
	RequiresAPIKey bool   `json:"requires_api_key"`
	Description    string `json:"description"`
	APIURL         string `json:"api_url,omitempty"`
}
