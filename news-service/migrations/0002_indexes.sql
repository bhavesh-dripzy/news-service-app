-- Create indexes for better performance

-- Articles table indexes
CREATE INDEX IF NOT EXISTS idx_articles_pubdate_desc ON articles (publication_date DESC);
CREATE INDEX IF NOT EXISTS idx_articles_source ON articles (source_name);
CREATE INDEX IF NOT EXISTS idx_articles_category_gin ON articles USING GIN (category);
CREATE INDEX IF NOT EXISTS idx_articles_tsv_gin ON articles USING GIN (tsv);
CREATE INDEX IF NOT EXISTS idx_articles_title_trgm ON articles USING GIN (title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_articles_relevance_score ON articles (relevance_score DESC);

-- User events table indexes
CREATE INDEX IF NOT EXISTS idx_user_events_article_time ON user_events (article_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_events_time ON user_events (occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_events_location ON user_events (user_lat, user_lon) WHERE user_lat IS NOT NULL AND user_lon IS NOT NULL;

-- Article summaries table indexes
CREATE INDEX IF NOT EXISTS idx_article_summaries_generated ON article_summaries (generated_at DESC);
CREATE INDEX IF NOT EXISTS idx_article_summaries_model ON article_summaries (model);

