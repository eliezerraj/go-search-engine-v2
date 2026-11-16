package entity

import (
	"encoding/json"
	"time"
)

type VectorSearch struct {
	Text string `json:"text,omitempty"`
    Embedding []float64 `json:"embedding,omitempty"`
    Capabilities []Capability `json:"capabilities,omitempty"`
}

type Capability struct {
    Text string `json:"text,omitempty"`
	Name string `json:"name,omitempty"`
    Type string `json:"type,omitempty"`
    Endpoint string `json:"endpoint,omitempty"`
    Uri string `json:"uri,omitempty"`
}

type IntentDecompose struct {
	Intent string `json:"intent,omitempty"`
}

type TeiModel struct {
	Inputs 				string `json:"inputs,omitempty"`
    Normalize 			bool `json:"normalize,omitempty"`
    Truncate 			bool `json:"truncate,omitempty"`
    TruncationDirection string `json:"truncation_direction,omitempty"`
    Embedding 			[]float64 `json:"embedding,omitempty"`
}

type OutboxEvent struct {
	ID            int64           `json:"id"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     time.Time       `json:"created_at"`
	PublishedAt   *time.Time      `json:"published_at,omitempty"`
	Attempts      int32           `json:"attempts"`
	LastError     *string         `json:"last_error,omitempty"`
}

type Bm25SearchQuery struct {
	Query           string   `json:"query" validate:"required"`
	CapabilityTypes []string `json:"capabilityTypes,omitempty"`
	ServerIds       []string `json:"serverIds,omitempty"`
	Limit           int      `json:"limit,omitempty"`
}

type Bm25SearchResult struct {
	CapabilityID string  `json:"capabilityId"`
	Type         string  `json:"type"`
	Name         string  `json:"name"`
	Score        float64 `json:"score"`
}