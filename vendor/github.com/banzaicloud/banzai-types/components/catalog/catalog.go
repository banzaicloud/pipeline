package catalog

// SpotguideFile to parse spotguide.yaml
type SpotguideFile struct {
	Resources *ApplicationResources   `json:"resources"`
	Options   []ApplicationOptions    `json:"options"`
	Depends   []ApplicationDependency `json:"depends"`
}

// ApplicationOptions for API response
type ApplicationOptions struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Default  bool     `json:"default"`
	Label    string   `json:"label"`
	Info     string   `json:"info"`
	Readonly bool     `json:"readonly"`
	Key      string   `json:"key"`
	Value    string   `json:"value"`
	Enum     []string `json:"enum"`
}

// ApplicationDependency for spotguide.yaml
type ApplicationDependency struct {
	Info      string           `json:"info"`
	Name      string           `json:"name"`
	Type      string           `json:"type"`
	Values    []string         `json:"values"`
	Namespace string           `json:"namespace"`
	Chart     ApplicationChart `json:"chart"`
	Timeout   int              `json:"timeout"`
	Retry     int              `json:"retry"`
}

// ApplicationChart for spotguide.yaml
type ApplicationChart struct {
	Name       string `json:"name"`
	Repository string `json:"repository"`
	Version    string `json:"version"`
}

// ApplicationResources to parse spotguide.yaml
type ApplicationResources struct {
	SumCpu      float64  `json:"sumCpu" binding:"min=1"`
	SumMem      float64  `json:"sumMem" binding:"min=1"`
	MinNodes    int      `json:"minNodes,omitempty" binding:"min=1,ltefield=MaxNodes"`
	MaxNodes    int      `json:"maxNodes,omitempty"`
	SameSize    bool     `json:"sameSize,omitempty"`
	OnDemandPct int      `json:"onDemandPct,omitempty" binding:"min=1,max=100"`
	Zones       []string `json:"zones,omitempty" binding:"dive,zone"`
	SumGpu      int      `json:"sumGpu,omitempty"`
	AllowBurst  *bool    `json:"allowBurst,omitempty"`
	NetworkPerf *string  `json:"networkPerf,omitempty"`
}
