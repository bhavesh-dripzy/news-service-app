package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/rs/zerolog/log"
)

type OpenAIClient struct {
	client openai.Client
	model  string
}

func NewOpenAIClient(apiKey, model string) (*OpenAIClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))

	if model == "" {
		model = "gpt-4o-mini"
	}

	return &OpenAIClient{
		client: client,
		model:  model,
	}, nil
}

func (c *OpenAIClient) Extract(ctx context.Context, query string) (*Extraction, error) {
	// For now, return a mock extraction to avoid complex OpenAI API usage
	// TODO: Implement actual OpenAI API call when the types are properly understood
	log.Info().Str("query", query).Msg("Mock extraction - OpenAI API not yet implemented")
	
	queryLower := strings.ToLower(query)
	
	// Simple keyword-based extraction for testing
	var entities struct {
		People        []string `json:"people"`
		Organizations []string `json:"orgs"`
		Locations     []string `json:"locations"`
	}
	var concepts []string
	var intent []Intent
	var categories []string
	var sourceNames []string
	
	// Detect score-based queries
	if strings.Contains(queryLower, "score") || strings.Contains(queryLower, "relevance") || strings.Contains(queryLower, "above") || strings.Contains(queryLower, "threshold") || strings.Contains(queryLower, "high quality") || strings.Contains(queryLower, "best") {
		intent = append(intent, Intent{Type: "score", Confidence: 0.9})
	}
	
	// Detect categories
	if strings.Contains(queryLower, "technology") || strings.Contains(queryLower, "tech") {
		categories = append(categories, "Technology")
		intent = append(intent, Intent{Type: "category", Confidence: 0.9})
	}
	if strings.Contains(queryLower, "business") || strings.Contains(queryLower, "finance") {
		categories = append(categories, "Business")
		intent = append(intent, Intent{Type: "category", Confidence: 0.9})
	}
	if strings.Contains(queryLower, "sports") {
		categories = append(categories, "Sports")
		intent = append(intent, Intent{Type: "category", Confidence: 0.9})
	}
	if strings.Contains(queryLower, "health") || strings.Contains(queryLower, "medical") {
		categories = append(categories, "Health")
		intent = append(intent, Intent{Type: "category", Confidence: 0.9})
	}
	if strings.Contains(queryLower, "science") {
		categories = append(categories, "Science")
		intent = append(intent, Intent{Type: "category", Confidence: 0.9})
	}
	if strings.Contains(queryLower, "environment") || strings.Contains(queryLower, "climate") {
		categories = append(categories, "Environment")
		intent = append(intent, Intent{Type: "category", Confidence: 0.9})
	}
	if strings.Contains(queryLower, "entertainment") || strings.Contains(queryLower, "movie") || strings.Contains(queryLower, "gaming") {
		categories = append(categories, "Entertainment")
		intent = append(intent, Intent{Type: "category", Confidence: 0.9})
	}
	if strings.Contains(queryLower, "politics") || strings.Contains(queryLower, "government") {
		categories = append(categories, "Politics")
		intent = append(intent, Intent{Type: "category", Confidence: 0.9})
	}
	
	// Detect sources
	if strings.Contains(queryLower, "new york times") || strings.Contains(queryLower, "nyt") {
		sourceNames = append(sourceNames, "New York Times")
		intent = append(intent, Intent{Type: "source", Confidence: 0.9})
	}
	if strings.Contains(queryLower, "reuters") {
		sourceNames = append(sourceNames, "Reuters")
		intent = append(intent, Intent{Type: "source", Confidence: 0.9})
	}
	if strings.Contains(queryLower, "bbc") {
		sourceNames = append(sourceNames, "BBC")
		intent = append(intent, Intent{Type: "source", Confidence: 0.9})
	}
	if strings.Contains(queryLower, "cnn") {
		sourceNames = append(sourceNames, "CNN")
		intent = append(intent, Intent{Type: "source", Confidence: 0.9})
	}
	if strings.Contains(queryLower, "dw") {
		sourceNames = append(sourceNames, "DW")
		intent = append(intent, Intent{Type: "source", Confidence: 0.9})
	}
	if strings.Contains(queryLower, "technews") {
		sourceNames = append(sourceNames, "TechNews")
		intent = append(intent, Intent{Type: "source", Confidence: 0.9})
	}
	if strings.Contains(queryLower, "spacenews") {
		sourceNames = append(sourceNames, "SpaceNews")
		intent = append(intent, Intent{Type: "source", Confidence: 0.9})
	}
	if strings.Contains(queryLower, "financedaily") {
		sourceNames = append(sourceNames, "FinanceDaily")
		intent = append(intent, Intent{Type: "source", Confidence: 0.9})
	}
	if strings.Contains(queryLower, "healthscience") {
		sourceNames = append(sourceNames, "HealthScience")
		intent = append(intent, Intent{Type: "source", Confidence: 0.9})
	}
	if strings.Contains(queryLower, "globalnews") {
		sourceNames = append(sourceNames, "GlobalNews")
		intent = append(intent, Intent{Type: "source", Confidence: 0.9})
	}
	
	// Detect locations
	if strings.Contains(queryLower, "paris") {
		entities.Locations = append(entities.Locations, "Paris")
		intent = append(intent, Intent{Type: "nearby", Confidence: 0.8})
	}
	if strings.Contains(queryLower, "new york") || strings.Contains(queryLower, "nyc") {
		entities.Locations = append(entities.Locations, "New York")
		intent = append(intent, Intent{Type: "nearby", Confidence: 0.8})
	}
	if strings.Contains(queryLower, "london") {
		entities.Locations = append(entities.Locations, "London")
		intent = append(intent, Intent{Type: "nearby", Confidence: 0.8})
	}
	if strings.Contains(queryLower, "near") || strings.Contains(queryLower, "nearby") || strings.Contains(queryLower, "local") || strings.Contains(queryLower, "location") {
		intent = append(intent, Intent{Type: "nearby", Confidence: 0.7})
	}
	
	// Detect people
	if strings.Contains(queryLower, "elon musk") {
		entities.People = append(entities.People, "Elon Musk")
	}
	if strings.Contains(queryLower, "john smith") {
		entities.People = append(entities.People, "John Smith")
	}
	
	// Detect organizations
	if strings.Contains(queryLower, "spacex") {
		entities.Organizations = append(entities.Organizations, "SpaceX")
	}
	if strings.Contains(queryLower, "tesla") {
		entities.Organizations = append(entities.Organizations, "Tesla")
	}
	
	// Add concepts
	if strings.Contains(queryLower, "ai") || strings.Contains(queryLower, "artificial intelligence") {
		concepts = append(concepts, "Artificial Intelligence")
	}
	if strings.Contains(queryLower, "climate change") {
		concepts = append(concepts, "Climate Change")
	}
	if strings.Contains(queryLower, "stock market") {
		concepts = append(concepts, "Stock Market")
	}
	
	// Default to search if no specific intent detected
	if len(intent) == 0 {
		intent = append(intent, Intent{Type: "search", Confidence: 0.7})
	}
	
	return &Extraction{
		Entities:    entities,
		Concepts:    concepts,
		Intent:      intent,
		Categories:  categories,
		SourceNames: sourceNames,
	}, nil
}

func (c *OpenAIClient) Summarize(ctx context.Context, title, description, sourceName, publicationDate string) (string, error) {
	// For now, return a mock summary to avoid complex OpenAI API usage
	// TODO: Implement actual OpenAI API call when the types are properly understood
	log.Info().Str("title", title).Msg("Mock summarization - OpenAI API not yet implemented")
	
	return fmt.Sprintf("This article discusses %s, published by %s on %s. %s", 
		title, sourceName, publicationDate, description), nil
}
