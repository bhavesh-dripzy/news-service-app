package news

// SearchRequest represents a search query request
type SearchRequest struct {
	Query string `json:"query" validate:"required,min=1,max=500"`
	Limit int    `json:"limit" validate:"min=1,max=50"`
}

// CategoryRequest represents a category filter request
type CategoryRequest struct {
	Name  string `json:"name" validate:"required,min=1,max=100"`
	Limit int    `json:"limit" validate:"min=1,max=50"`
}

// SourceRequest represents a source filter request
type SourceRequest struct {
	Name  string `json:"name" validate:"required,min=1,max=100"`
	Limit int    `json:"limit" validate:"min=1,max=50"`
}

// ScoreRequest represents a score filter request
type ScoreRequest struct {
	Min   float64 `json:"min" validate:"min=0,max=1"`
	Limit int     `json:"limit" validate:"min=1,max=50"`
}

// NearbyRequest represents a nearby search request
type NearbyRequest struct {
	Lat    float64 `json:"lat" validate:"min=-90,max=90"`
	Lon    float64 `json:"lon" validate:"min=-180,max=180"`
	Radius float64 `json:"radius_km" validate:"min=0.1,max=200"`
	Limit  int     `json:"limit" validate:"min=1,max=50"`
}

// TrendingRequest represents a trending request
type TrendingRequest struct {
	Lat   float64 `json:"lat" validate:"min=-90,max=90"`
	Lon   float64 `json:"lon" validate:"min=-180,max=180"`
	Limit int     `json:"limit" validate:"min=1,max=50"`
}

// AutoRequest represents an automatic intent routing request
type AutoRequest struct {
	Query string  `json:"query" validate:"required,min=1,max=500"`
	Lat   *float64 `json:"lat,omitempty" validate:"omitempty,min=-90,max=90"`
	Lon   *float64 `json:"lon,omitempty" validate:"omitempty,min=-180,max=180"`
	Limit int     `json:"limit" validate:"min=1,max=50"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorInfo `json:"error"`
}

// ErrorInfo represents error details
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Common error codes
const (
	ErrCodeValidation     = "VALIDATION_ERROR"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeInternal       = "INTERNAL_ERROR"
	ErrCodeRateLimit      = "RATE_LIMIT"
	ErrCodeBadRequest     = "BAD_REQUEST"
	ErrCodeUnauthorized   = "UNAUTHORIZED"
)

// NewErrorResponse creates a new error response
func NewErrorResponse(code, message string) *ErrorResponse {
	return &ErrorResponse{
		Error: ErrorInfo{
			Code:    code,
			Message: message,
		},
	}
}

