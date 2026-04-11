package model

// HomeSection represents a curated section on the home screen.
type HomeSection struct {
	Title string      `json:"title"`
	Type  string      `json:"type"` // "songs", "artists", "albums"
	Items interface{} `json:"items"`
}

// HomeResponse is the full response for GET /v1/home.
type HomeResponse struct {
	Sections []HomeSection `json:"sections"`
}

// APIError is the standard error response shape.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse wraps an APIError.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// PaginatedResponse is a generic paginated response.
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	NextCursor *string     `json:"nextCursor"`
}
