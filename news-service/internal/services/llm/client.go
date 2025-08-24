package llm

import (
	"context"
)

// Extraction represents the structured output from LLM
type Extraction struct {
	Entities struct {
		People     []string `json:"people"`
		Organizations []string `json:"orgs"`
		Locations  []string `json:"locations"`
	} `json:"entities"`
	Concepts []string `json:"concepts"`
	Intent   []Intent `json:"intent"`
	RadiusKm *float64 `json:"radius_km,omitempty"`
	SourceNames []string `json:"source_names,omitempty"`
	Categories []string `json:"categories,omitempty"`
}

type Intent struct {
	Type       string  `json:"type"`
	Confidence float64 `json:"confidence"`
}

// LLMClient interface for different LLM providers
type LLMClient interface {
	// Extract entities, concepts, and intent from a query
	Extract(ctx context.Context, query string) (*Extraction, error)
	
	// Summarize an article in 2-3 sentences
	Summarize(ctx context.Context, title, description, sourceName, publicationDate string) (string, error)
}

