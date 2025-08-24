-- name: CreateArticle :one
INSERT INTO articles (
    id, title, description, url, publication_date, source_name, 
    category, relevance_score, latitude, longitude
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) ON CONFLICT (id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    url = EXCLUDED.url,
    publication_date = EXCLUDED.publication_date,
    source_name = EXCLUDED.source_name,
    category = EXCLUDED.category,
    relevance_score = EXCLUDED.relevance_score,
    latitude = EXCLUDED.latitude,
    longitude = EXCLUDED.longitude
RETURNING *;

-- name: GetArticleByID :one
SELECT * FROM articles WHERE id = $1;

-- name: GetArticlesByCategory :many
SELECT * FROM articles 
WHERE $1 = ANY(category)
ORDER BY publication_date DESC
LIMIT $2;

-- name: GetArticlesBySource :many
SELECT * FROM articles 
WHERE source_name = $1
ORDER BY publication_date DESC
LIMIT $2;

-- name: GetArticlesByScore :many
SELECT * FROM articles 
WHERE relevance_score >= $1
ORDER BY relevance_score DESC, publication_date DESC
LIMIT $2;

-- name: SearchArticles :many
SELECT 
    *,
    (0.6 * ts_rank(tsv, plainto_tsquery('english', $1)) + 0.4 * relevance_score) as search_score
FROM articles 
WHERE tsv @@ plainto_tsquery('english', $1)
ORDER BY search_score DESC, publication_date DESC
LIMIT $2;

-- name: GetNearbyArticles :many
SELECT 
    *,
    earth_distance(
        ll_to_earth($1, $2), 
        ll_to_earth(latitude, longitude)
    ) as distance_meters
FROM articles 
WHERE latitude IS NOT NULL 
    AND longitude IS NOT NULL
    AND earth_distance(
        ll_to_earth($1, $2), 
        ll_to_earth(latitude, longitude)
    ) <= $3 * 1000
ORDER BY distance_meters ASC
LIMIT $4;

-- name: GetRecentEventsByGeohash :many
SELECT 
    ue.*,
    a.latitude,
    a.longitude
FROM user_events ue
JOIN articles a ON ue.article_id = a.id
WHERE 
    a.latitude IS NOT NULL 
    AND a.longitude IS NOT NULL
    AND ue.occurred_at >= $1
    AND ue.user_lat IS NOT NULL 
    AND ue.user_lon IS NOT NULL
ORDER BY ue.occurred_at DESC;

-- name: CreateArticleSummary :one
INSERT INTO article_summaries (
    article_id, llm_summary, model
) VALUES (
    $1, $2, $3
) ON CONFLICT (article_id) DO UPDATE SET
    llm_summary = EXCLUDED.llm_summary,
    model = EXCLUDED.model,
    generated_at = now()
RETURNING *;

-- name: GetArticleSummary :one
SELECT * FROM article_summaries WHERE article_id = $1;

-- name: CreateUserEvent :one
INSERT INTO user_events (
    article_id, event, user_lat, user_lon
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetArticlesWithoutSummary :many
SELECT a.* FROM articles a
LEFT JOIN article_summaries s ON a.id = s.article_id
WHERE s.article_id IS NULL
LIMIT $1;

