package azure

// ResponseWithValue describes an Azure cluster
type ResponseWithValue struct {
	StatusCode int   `json:"status_code"`
	Value      Value `json:"message,omitempty"`
}

// Values describes a list of Azure clusters
type Values struct {
	Value []Value `json:"value"`
}

// Value describes an Azure cluster
type Value struct {
	Id         string     `json:"id"`
	Location   string     `json:"location"`
	Name       string     `json:"name"`
	Properties Properties `json:"properties"`
}

// Properties describes an Azure cluster properties
type Properties struct {
	ProvisioningState string    `json:"provisioningState"`
	AgentPoolProfiles []Profile `json:"agentPoolProfiles"`
	Fqdn              string    `json:"fqdn"`
}

// Profile describes an Azure agent pool
type Profile struct {
	Name        string `json:"name"`
	Autoscaling bool   `json:"autoscaling"`
	MinCount    int    `json:"minCount"`
	MaxCount    int    `json:"maxCount"`
	Count       int    `json:"count"`
	VmSize      string `json:"vmSize"`
}

// Config describes an Azure kubeconfig
type Config struct {
	Location   string `json:"location"`
	Name       string `json:"name"`
	Properties struct {
		KubeConfig string `json:"kubeConfig"`
	} `json:"properties"`
}

// ListResponse describes an Azure cluster list
type ListResponse struct {
	StatusCode int    `json:"status_code"`
	Value      Values `json:"message"`
}

// Update updates `ResponseWithValue` with the given response code and value
func (r *ResponseWithValue) Update(code int, Value Value) {
	r.Value = Value
	r.StatusCode = code
}
