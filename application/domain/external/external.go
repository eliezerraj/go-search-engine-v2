package external

import "github.com/go-semantic-engine-v2/application/domain/entity"

type VectorSearchRequest struct {
	Text string `json:"text" validate:"required"`
}

type VectorSearchResponse struct {
	Response    string	`json:"response"`
	Vector		any	`json:"vector,omitempty"`
}

type IntentDecomposeRequest struct {
	Intent string `json:"intent" validate:"required"`
}

type IntentDecomposeResponse struct {
	Response    string	`json:"response"`
	Intent		any	`json:"intent,omitempty"`
}

type OutboxEventRequest struct {
	AggregateType string          `json:"aggregate_type" validate:"required"`
	EventType     string          `json:"event_type"`
}

type OutBoxEventResponse struct {
	Response    string          `json:"response"`
	Event       any             `json:"event,omitempty"`
}

type Bm25SearchRequest struct {
	Query           string   `json:"query" validate:"required"`
	CapabilityTypes []string `json:"capabilityTypes,omitempty"`
	ServerIds       []string `json:"serverIds,omitempty"`
	Limit           int      `json:"limit,omitempty"`
}

type Bm25SearchResponse struct {
	Results []entity.Bm25SearchResult `json:"results"`
}
