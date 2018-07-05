package common

// BanzaiResponse describes Pipeline's responses
type BanzaiResponse struct {
	StatusCode int    `json:"status_code,omitempty"`
	Message    string `json:"message,omitempty"`
}

// ErrorResponse describes Pipeline's responses when an error occurred
type ErrorResponse struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// CreatorBaseFields describes all field which contains info about who created the cluster/application etc
type CreatorBaseFields struct {
	CreatedAt   string `json:"createdAt,omitempty"`
	CreatorName string `json:"creatorName,omitempty"`
	CreatorId   uint   `json:"creatorId,omitempty"`
}

// NodeNames describes node names
type NodeNames map[string][]string

// ### [ Constants to common cluster default values ] ### //
const (
	DefaultNodeMinCount = 1
	DefaultNodeMaxCount = 2
)

// Constants for labeling cluster nodes
const (
	LabelKey = "pipeline-nodepool-name"
)
