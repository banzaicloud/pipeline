package catalog

// SpotguideFile to parse spotguide.yaml
type SpotguideFile struct {
	Resources *ApplicationResources   `json:"resources"`
	Options   []ApplicationOptions    `json:"options"`
	Depends   []ApplicationDependency `json:"depends"`
}

// ApplicationOptions for API response
type ApplicationOptions struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Default  bool   `json:"default"`
	Info     string `json:"info"`
	Readonly bool   `json:"readonly"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

// ApplicationDependency for spotguide.yaml
type ApplicationDependency struct {
	Name      string           `json:"name"`
	Type      string           `json:"type"`
	Values    []string         `json:"values"`
	Namespace string           `json:"namespace"`
	Chart     ApplicationChart `json:"chart"`
	Timeout		int							 `json:"timeout"`
	Retry 		int 						 `json:"retry"`
}

// ApplicationChart for spotguide.yaml
type ApplicationChart struct {
	Name       string `json:"name"`
	Repository string `json:"repository"`
	Version    string `json:"version"`
}

// ApplicationResources to parse spotguide.yaml
type ApplicationResources struct {
	VCPU               int      `json:"vcpu"`
	Memory             int      `json:"memory"`
	Filters            []string `json:"filters"`
	OnDemandPercentage int      `json:"onDemandPercentage"`
	SameSize           bool     `json:"sameSize"`
}
