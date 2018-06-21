package common

// BanzaiResponse describes Pipeline's responses
type BanzaiResponse struct {
	StatusCode int    `json:"status_code,omitempty"`
	Message    string `json:"message,omitempty"`
}

// ErrorResponse describes Pipeline's responses when an error occurred
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

// ### [ Constants to common cluster default values ] ### //
const (
	DefaultNodeMinCount = 1
	DefaultNodeMaxCount = 2
)
