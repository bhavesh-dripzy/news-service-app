-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS cube;
CREATE EXTENSION IF NOT EXISTS earthdistance;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Create articles table
CREATE TABLE IF NOT EXISTS articles (
  id               UUID PRIMARY KEY,
  title            TEXT NOT NULL,
  description      TEXT,
  url              TEXT NOT NULL,
  publication_date TIMESTAMPTZ NOT NULL,
  source_name      TEXT NOT NULL,
  category         TEXT[] NOT NULL,
  relevance_score  DOUBLE PRECISION NOT NULL,
  latitude         DOUBLE PRECISION,
  longitude        DOUBLE PRECISION,
  tsv              tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(description, '')), 'B')
  ) STORED
);

-- Create article summaries table
CREATE TABLE IF NOT EXISTS article_summaries (
  article_id   UUID PRIMARY KEY REFERENCES articles(id) ON DELETE CASCADE,
  llm_summary  TEXT NOT NULL,
  model        TEXT NOT NULL,
  generated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Create event type enum
DO $$ BEGIN
  CREATE TYPE event_type AS ENUM ('view','click');
EXCEPTION
  WHEN duplicate_object THEN null;
END $$;

-- Create user events table
CREATE TABLE IF NOT EXISTS user_events (
  id           BIGSERIAL PRIMARY KEY,
  article_id   UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
  event        event_type NOT NULL,
  occurred_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
  user_lat     DOUBLE PRECISION,
  user_lon     DOUBLE PRECISION
);

