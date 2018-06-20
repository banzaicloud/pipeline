package catalog

// SpotguideFile to parse spotguide.yaml
type SpotguideFile struct {
	Resources *ApplicationResources   `json:"resources" yaml:"resources"`
	Options   []ApplicationOptions    `json:"options" yaml:"options"`
	Depends   []ApplicationDependency `json:"depends" yaml:"depends"`
	Secrets   []ApplicationSecret     `json:"secrets" yaml:"secrets"`
}

// ApplicationSecret for API response
type ApplicationSecret struct {
	Name     string                     `json:"name" yaml:"name" binding:"required"`
	Htaccess *ApplicationSecretHtaccess `json:"htaccess,omitempty" yaml:"htaccess,omitempty"`
	Password *ApplicationSecretPassword `json:"password,omitempty" yaml:"password,omitempty"`
	TLS      *ApplicationSecretTLS      `json:"tls,omitempty" yaml:"tls,omitempty"`
}

// ApplicationSecretHtaccess to parse spotguide.yaml
type ApplicationSecretHtaccess struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

// ApplicationSecretPassword to parse spotguide.yaml
type ApplicationSecretPassword struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}

// ApplicationSecretTLS to parse spotguide.yaml
type ApplicationSecretTLS struct {
	Hosts      string `json:"hosts" yaml:"hosts" binding:"required"`
	Validity string `json:"validity,omitempty" yaml:"validity,omitempty"`
}

// ApplicationOptions for API response
type ApplicationOptions struct {
	Name     string   `json:"name" yaml:"name"`
	Type     string   `json:"type" yaml:"type"`
	Default  string   `json:"default" yaml:"default"`
	Label    string   `json:"label" yaml:"label"`
	Info     string   `json:"info" yaml:"info"`
	Readonly bool     `json:"readonly" yaml:"readonly"`
	Key      string   `json:"key" yaml:"key"`
	Value    string   `json:"value" yaml:"value"`
	Enum     []string `json:"enum" yaml:"enum"`
}

// ApplicationDependency for spotguide.yaml
type ApplicationDependency struct {
	Info      string           `json:"info" yaml:"info"`
	Name      string           `json:"name" yaml:"name"`
	Type      string           `json:"type" yaml:"type"`
	Values    []string         `json:"values" yaml:"values"`
	Namespace string           `json:"namespace" yaml:"namespace"`
	Chart     ApplicationChart `json:"chart" yaml:"chart"`
	Timeout   int              `json:"timeout" yaml:"timeout"`
	Retry     int              `json:"retry" yaml:"retry"`
}

// ApplicationChart for spotguide.yaml
type ApplicationChart struct {
	Name       string `json:"name" yaml:"name"`
	Repository string `json:"repository" yaml:"repository"`
	Version    string `json:"version" yaml:"version"`
}

// ApplicationResources to parse spotguide.yaml
type ApplicationResources struct {
	SumCpu      float64  `json:"sumCpu" yaml:"sumCpu"`
	SumMem      float64  `json:"sumMem" yaml:"sumMem"`
	MinNodes    int      `json:"minNodes,omitempty" yaml:"minNodes,omitempty"`
	MaxNodes    int      `json:"maxNodes,omitempty" yaml:"maxNodes,omitempty"`
	SameSize    bool     `json:"sameSize,omitempty" yaml:"sameSize,omitempty"`
	OnDemandPct int      `json:"onDemandPct,omitempty" yaml:"onDemandPct,omitempty" binding:"min=1,max=100"`
	Zones       []string `json:"zones,omitempty" yaml:"zones,omitempty" binding:"dive,zone"`
	SumGpu      int      `json:"sumGpu,omitempty" yaml:"sumGpu,omitempty"`
	AllowBurst  *bool    `json:"allowBurst,omitempty" yaml:"allowBurst,omitempty"`
	NetworkPerf *string  `json:"networkPerf,omitempty" yaml:"networkPerf,omitempty"`
}
