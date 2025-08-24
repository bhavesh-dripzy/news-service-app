package ingest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"news-system/internal/repo"
	"news-system/internal/services/news"
)

// Loader handles data ingestion from JSON files
type Loader struct {
	repo repo.Repository
}

// NewLoader creates a new Loader instance
func NewLoader(repo repo.Repository) *Loader {
	return &Loader{repo: repo}
}

// LoadFromDirectory loads all JSON files from a directory
func (l *Loader) LoadFromDirectory(ctx context.Context, dirPath string) error {
	return filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".json") {
			return nil
		}
		
		fmt.Printf("Loading file: %s\n", path)
		return l.LoadFromFile(ctx, path)
	})
}

// LoadFromFile loads articles from a single JSON file
func (l *Loader) LoadFromFile(ctx context.Context, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	var articles []news.ArticleDTO
	if err := json.NewDecoder(file).Decode(&articles); err != nil {
		return fmt.Errorf("failed to decode JSON from %s: %w", filePath, err)
	}

	fmt.Printf("Found %d articles in %s\n", len(articles), filePath)
	
	for i, article := range articles {
		if err := l.LoadArticle(ctx, article); err != nil {
			fmt.Printf("Failed to load article %d: %v\n", i, err)
			continue
		}
		fmt.Printf("Loaded article: %s\n", article.Title)
	}
	
	return nil
}

// LoadArticle loads a single article into the database
func (l *Loader) LoadArticle(ctx context.Context, article news.ArticleDTO) error {
	// Generate a unique ID for the article
	id := generateID()
	
	// Convert DTO to database model
	dbArticle := repo.CreateArticleParams{
		ID:              id,
		Title:           article.Title,
		Description:     article.Description,
		URL:             article.URL,
		PublicationDate: article.PublicationDate,
		SourceName:      article.SourceName,
		Category:        article.Category,
		RelevanceScore:  article.RelevanceScore,
		Latitude:        article.Latitude,
		Longitude:       article.Longitude,
	}

	// Create the article
	_, err := l.repo.CreateArticle(ctx, dbArticle)
	if err != nil {
		return fmt.Errorf("failed to create article: %w", err)
	}

	return nil
}

// generateID generates a simple unique ID
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// GenerateSampleData generates 20 sample articles for testing
func (l *Loader) GenerateSampleData(ctx context.Context) error {
	sampleArticles := []news.ArticleDTO{
		{
			Title:           "Tech Giants Announce AI Partnership",
			Description:     stringPtr("Major technology companies have announced a collaborative partnership to advance artificial intelligence research and development..."),
			URL:             "https://example.com/tech-ai-partnership",
			SourceName:      "TechNews",
			Category:        []string{"Technology", "AI"},
			PublicationDate: time.Now().Add(-2 * time.Hour),
			Latitude:        float64Ptr(37.7749),
			Longitude:       float64Ptr(-122.4194),
			RelevanceScore:  0.9,
		},
		{
			Title:           "Climate Change Summit in Paris",
			Description:     stringPtr("World leaders gather in Paris for the annual climate change summit to discuss global warming solutions..."),
			URL:             "https://example.com/climate-summit-paris",
			SourceName:      "GlobalNews",
			Category:        []string{"Environment", "Politics"},
			PublicationDate: time.Now().Add(-4 * time.Hour),
			Latitude:        float64Ptr(48.8566),
			Longitude:       float64Ptr(2.3522),
			RelevanceScore:  0.95,
		},
		{
			Title:           "Stock Market Reaches New Highs",
			Description:     stringPtr("Global stock markets have reached unprecedented levels as investors show confidence in economic recovery..."),
			URL:             "https://example.com/stock-market-highs",
			SourceName:      "FinanceDaily",
			Category:        []string{"Business", "Finance"},
			PublicationDate: time.Now().Add(-6 * time.Hour),
			Latitude:        float64Ptr(40.7128),
			Longitude:       float64Ptr(-74.0060),
			RelevanceScore:  0.85,
		},
		{
			Title:           "New Medical Breakthrough in Cancer Research",
			Description:     stringPtr("Scientists have discovered a promising new treatment approach for certain types of cancer..."),
			URL:             "https://example.com/cancer-research-breakthrough",
			SourceName:      "HealthScience",
			Category:        []string{"Health", "Science"},
			PublicationDate: time.Now().Add(-8 * time.Hour),
			Latitude:        float64Ptr(43.6532),
			Longitude:       float64Ptr(-79.3832),
			RelevanceScore:  0.98,
		},
		{
			Title:           "SpaceX Launches New Satellite Constellation",
			Description:     stringPtr("SpaceX successfully launched another batch of satellites for its global internet constellation..."),
			URL:             "https://example.com/spacex-satellite-launch",
			SourceName:      "SpaceNews",
			Category:        []string{"Science", "Technology"},
			PublicationDate: time.Now().Add(-10 * time.Hour),
			Latitude:        float64Ptr(28.5729),
			Longitude:       float64Ptr(-80.6490),
			RelevanceScore:  0.92,
		},
		{
			Title:           "Olympic Games Opening Ceremony",
			Description:     stringPtr("The world's greatest athletes gather for the opening ceremony of the Olympic Games..."),
			URL:             "https://example.com/olympics-opening",
			SourceName:      "SportsCentral",
			Category:        []string{"Sports"},
			PublicationDate: time.Now().Add(-12 * time.Hour),
			Latitude:        float64Ptr(35.6762),
			Longitude:       float64Ptr(139.6503),
			RelevanceScore:  0.88,
		},
		{
			Title:           "New Electric Vehicle Factory Opens",
			Description:     stringPtr("A major automaker has opened a new factory dedicated to electric vehicle production..."),
			URL:             "https://example.com/ev-factory-opens",
			SourceName:      "AutoIndustry",
			Category:        []string{"Technology", "Business"},
			PublicationDate: time.Now().Add(-14 * time.Hour),
			Latitude:        float64Ptr(52.5200),
			Longitude:       float64Ptr(13.4050),
			RelevanceScore:  0.87,
		},
		{
			Title:           "Global Food Security Conference",
			Description:     stringPtr("Experts discuss solutions to global food security challenges at international conference..."),
			URL:             "https://example.com/food-security-conference",
			SourceName:      "WorldFood",
			Category:        []string{"World", "Environment"},
			PublicationDate: time.Now().Add(-16 * time.Hour),
			Latitude:        float64Ptr(41.9028),
			Longitude:       float64Ptr(12.4964),
			RelevanceScore:  0.83,
		},
		{
			Title:           "Cybersecurity Threat Alert",
			Description:     stringPtr("Government agencies issue warning about new sophisticated cyber attack patterns..."),
			URL:             "https://example.com/cybersecurity-alert",
			SourceName:      "CyberNews",
			Category:        []string{"Technology", "Security"},
			PublicationDate: time.Now().Add(-18 * time.Hour),
			Latitude:        float64Ptr(51.5074),
			Longitude:       float64Ptr(-0.1278),
			RelevanceScore:  0.96,
		},
		{
			Title:           "Renewable Energy Milestone Reached",
			Description:     stringPtr("Global renewable energy production has reached a new milestone, surpassing fossil fuels..."),
			URL:             "https://example.com/renewable-energy-milestone",
			SourceName:      "GreenEnergy",
			Category:        []string{"Environment", "Technology"},
			PublicationDate: time.Now().Add(-20 * time.Hour),
			Latitude:        float64Ptr(55.6761),
			Longitude:       float64Ptr(12.5683),
			RelevanceScore:  0.91,
		},
		{
			Title:           "Major Sports League Expansion",
			Description:     stringPtr("Professional sports league announces expansion to new cities and markets..."),
			URL:             "https://example.com/sports-league-expansion",
			SourceName:      "SportsBiz",
			Category:        []string{"Sports", "Business"},
			PublicationDate: time.Now().Add(-22 * time.Hour),
			Latitude:        float64Ptr(34.0522),
			Longitude:       float64Ptr(-118.2437),
			RelevanceScore:  0.84,
		},
		{
			Title:           "New Quantum Computing Breakthrough",
			Description:     stringPtr("Researchers achieve quantum supremacy in solving complex computational problems..."),
			URL:             "https://example.com/quantum-computing-breakthrough",
			SourceName:      "QuantumTech",
			Category:        []string{"Science", "Technology"},
			PublicationDate: time.Now().Add(-24 * time.Hour),
			Latitude:        float64Ptr(55.7558),
			Longitude:       float64Ptr(37.6176),
			RelevanceScore:  0.97,
		},
		{
			Title:           "Global Trade Agreement Signed",
			Description:     stringPtr("Major economies sign comprehensive trade agreement to boost international commerce..."),
			URL:             "https://example.com/global-trade-agreement",
			SourceName:      "TradeNews",
			Category:        []string{"Business", "World"},
			PublicationDate: time.Now().Add(-26 * time.Hour),
			Latitude:        float64Ptr(46.9479),
			Longitude:       float64Ptr(7.4474),
			RelevanceScore:  0.89,
		},
		{
			Title:           "New Archaeological Discovery",
			Description:     stringPtr("Archaeologists uncover ancient ruins that could rewrite human history..."),
			URL:             "https://example.com/archaeological-discovery",
			SourceName:      "AncientWorld",
			Category:        []string{"Science", "History"},
			PublicationDate: time.Now().Add(-28 * time.Hour),
			Latitude:        float64Ptr(30.0444),
			Longitude:       float64Ptr(31.2357),
			RelevanceScore:  0.86,
		},
		{
			Title:           "Digital Currency Adoption Surges",
			Description:     stringPtr("Central banks worldwide accelerate digital currency development and testing..."),
			URL:             "https://example.com/digital-currency-adoption",
			SourceName:      "CryptoNews",
			Category:        []string{"Finance", "Technology"},
			PublicationDate: time.Now().Add(-30 * time.Hour),
			Latitude:        float64Ptr(1.3521),
			Longitude:       float64Ptr(103.8198),
			RelevanceScore:  0.93,
		},
		{
			Title:           "New Movie Franchise Announced",
			Description:     stringPtr("Major studio announces new blockbuster movie franchise based on popular books..."),
			URL:             "https://example.com/movie-franchise-announced",
			SourceName:      "EntertainmentNow",
			Category:        []string{"Entertainment"},
			PublicationDate: time.Now().Add(-32 * time.Hour),
			Latitude:        float64Ptr(34.0522),
			Longitude:       float64Ptr(-118.2437),
			RelevanceScore:  0.82,
		},
		{
			Title:           "Global Internet Connectivity Initiative",
			Description:     stringPtr("International consortium launches project to provide internet access to remote areas..."),
			URL:             "https://example.com/internet-connectivity-initiative",
			SourceName:      "TechGlobal",
			Category:        []string{"Technology", "World"},
			PublicationDate: time.Now().Add(-34 * time.Hour),
			Latitude:        float64Ptr(28.7041),
			Longitude:       float64Ptr(77.1025),
			RelevanceScore:  0.88,
		},
		{
			Title:           "New Educational Technology Platform",
			Description:     stringPtr("Revolutionary online learning platform launches with AI-powered personalized education..."),
			URL:             "https://example.com/educational-tech-platform",
			SourceName:      "EduTech",
			Category:        []string{"Education", "Technology"},
			PublicationDate: time.Now().Add(-36 * time.Hour),
			Latitude:        float64Ptr(-33.8688),
			Longitude:       float64Ptr(151.2093),
			RelevanceScore:  0.90,
		},
		{
			Title:           "Sustainable Fashion Revolution",
			Description:     stringPtr("Major fashion brands commit to sustainable practices and eco-friendly materials..."),
			URL:             "https://example.com/sustainable-fashion-revolution",
			SourceName:      "FashionForward",
			Category:        []string{"Lifestyle", "Environment"},
			PublicationDate: time.Now().Add(-38 * time.Hour),
			Latitude:        float64Ptr(59.3293),
			Longitude:       float64Ptr(18.0686),
			RelevanceScore:  0.85,
		},
		{
			Title:           "New Gaming Console Launch",
			Description:     stringPtr("Next-generation gaming console launches with revolutionary graphics and AI features..."),
			URL:             "https://example.com/gaming-console-launch",
			SourceName:      "GameTech",
			Category:        []string{"Technology", "Entertainment"},
			PublicationDate: time.Now().Add(-40 * time.Hour),
			Latitude:        float64Ptr(35.6762),
			Longitude:       float64Ptr(139.6503),
			RelevanceScore:  0.94,
		},
	}

	fmt.Printf("Generating %d sample articles...\n", len(sampleArticles))
	
	for i, article := range sampleArticles {
		if err := l.LoadArticle(ctx, article); err != nil {
			fmt.Printf("Failed to load sample article %d: %v\n", i, err)
			continue
		}
		fmt.Printf("Generated sample article: %s\n", article.Title)
	}
	
	fmt.Printf("Successfully generated %d sample articles\n", len(sampleArticles))
	return nil
}

// Helper functions for creating pointers to primitive types
func stringPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}
