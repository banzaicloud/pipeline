package spotguide

import (
	libcompose "github.com/docker/libcompose/yaml"
)

// nolint
// droneRepoConfig defines a pipeline configuration.
type droneRepoConfig struct {
	Cache     libcompose.Stringorslice  `json:"cache,omitempty"`
	Platform  *string                   `json:"platform,omitempty"`
	Branches  *droneConstraint          `json:"branches,omitempty"`
	Workspace *droneWorkspace           `json:"workspace,omitempty"`
	Cluster   *droneKubernetesCluster   `json:"cluster,omitempty"`
	Clone     map[string]*droneContainer `json:"clone,omitempty"`
	Pipeline  map[string]*droneContainer `json:"pipeline,omitempty"`
	Services  map[string]*droneContainer `json:"services,omitempty"`
	Networks  map[string]*droneNetwork   `json:"networks,omitempty"`
	Volumes   map[string]*droneVolume    `json:"volumes,omitempty"`
	Labels    libcompose.SliceorMap     `json:"labels,omitempty"`
}

// nolint
// droneKubernetesCluster defines a cluster that has to be created before executing the rest of the Pipeline.
type droneKubernetesCluster struct {
	Name     *string `json:"name,omitempty"`
	Provider *string `json:"provider,omitempty"`
	SecretID *string `json:"secret_id,omitempty"`

	GoogleProject    *string `json:"google_project,omitempty"`
	GoogleNodeCount  int     `json:"google_node_count,omitempty"`
	GoogleGKEVersion *string `json:"google_gke_version,omitempty"`
}

// nolint
// droneWorkspace defines a pipeline workspace.
type droneWorkspace struct {
	Base *string `json:"base,omitempty"`
	Path *string `json:"path,omitempty"`
}

// nolint
// droneAuthConfig defines registry authentication credentials.
type droneAuthConfig struct {
	Username *string `json:"username,omitempty"`
	Password *string `json:"password,omitempty"`
	Email    *string `json:"email,omitempty"`
}

// nolint
// droneContainer defines a container.
type droneContainer struct {
	AuthConfig    *droneAuthConfig           `json:"auth_config,omitempty"`
	CapAdd        []string                   `json:"cap_add,omitempty"`
	CapDrop       []string                   `json:"cap_drop,omitempty"`
	Command       libcompose.Command         `json:"command,omitempty"`
	Commands      libcompose.Stringorslice   `json:"commands,omitempty"`
	CpuQuota      *libcompose.StringorInt    `json:"cpu_quota,omitempty"`
	CpuSet        *string                    `json:"cpuset,omitempty"`
	CpuShares     *libcompose.StringorInt    `json:"cpu_shares,omitempty"`
	Detached      *bool                      `json:"detach,omitempty"`
	Devices       []string                   `json:"devices,omitempty"`
	Tmpfs         []string                   `json:"tmpfs,omitempty"`
	Dns           libcompose.Stringorslice   `json:"dns,omitempty"`
	DnsSearch     libcompose.Stringorslice   `json:"dns_search,omitempty"`
	Entrypoint    libcompose.Command         `json:"entrypoint,omitempty"`
	Environment   libcompose.SliceorMap      `json:"environment,omitempty"`
	ExtraHosts    []string                   `json:"extra_hosts,omitempty"`
	Group         *string                    `json:"group,omitempty"`
	Image         *string                    `json:"image,omitempty"`
	Isolation     *string                    `json:"isolation,omitempty"`
	Labels        libcompose.SliceorMap      `json:"labels,omitempty"`
	MemLimit      *libcompose.MemStringorInt `json:"mem_limit,omitempty"`
	MemSwapLimit  *libcompose.MemStringorInt `json:"memswap_limit,omitempty"`
	MemSwappiness *libcompose.MemStringorInt `json:"mem_swappiness,omitempty"`
	Name          *string                    `json:"name,omitempty"`
	NetworkMode   *string                    `json:"network_mode,omitempty"`
	IpcMode       *string                    `json:"ipc_mode,omitempty"`
	Networks      *libcompose.Networks       `json:"networks,omitempty"`
	Ports         []int32                    `json:"ports,omitempty"`
	Privileged    *bool                      `json:"privileged,omitempty"`
	Pull          *bool                      `json:"pull,omitempty"`
	ShmSize       *libcompose.MemStringorInt `json:"shm_size,omitempty"`
	Ulimits       *libcompose.Ulimits        `json:"ulimits,omitempty"`
	Volumes       *libcompose.Volumes        `json:"volumes,omitempty"`
	Secrets       []string                   `json:"secrets,omitempty"`
	Sysctls       libcompose.SliceorMap      `json:"sysctls,omitempty"`
	Constraints   *droneConstraints          `json:"when,omitempty"`
	Vargs         map[string]interface{}     `json:",inline,omitempty"`
	Dockerfile    *string                    `json:"dockerfile,omitempty"`
	Repo          *string                    `json:"repo,omitempty"`
	Tags          *string                    `json:"tags,dockerfile,omitempty"`
	Log           *string                    `json:"log,omitempty"`
}

// nolint
// droneConstraints defines a set of runtime constraints.
type droneConstraints struct {
	Ref         *droneConstraint    `json:"ref,omitempty"`
	Repo        *droneConstraint    `json:"repo,omitempty"`
	Instance    *droneConstraint    `json:"instance,omitempty"`
	Platform    *droneConstraint    `json:"platform,omitempty"`
	Environment *droneConstraint    `json:"environment,omitempty"`
	Event       *droneConstraint    `json:"event,omitempty"`
	Branch      *droneConstraint    `json:"branch,omitempty"`
	Status      *droneConstraint    `json:"status,omitempty"`
	Matrix      *droneConstraintMap `json:"matrix,omitempty"`
	Local       *bool               `json:"local,omitempty"`
}

// nolint
// droneConstraint defines a runtime constraint.
type droneConstraint struct {
	Include libcompose.Stringorslice `json:"include,omitempty"`
	Exclude libcompose.Stringorslice `json:"exclude,omitempty"`
}

// nolint
// droneConstraintMap defines a runtime constraint map.
type droneConstraintMap struct {
	Include map[string]string `json:"include,omitempty"`
	Exclude map[string]string `json:"exclude,omitempty"`
}

// nolint
// droneNetwork defines a container network.
type droneNetwork struct {
	Name       *string            `json:"name,omitempty"`
	Driver     *string            `json:"driver,omitempty"`
	DriverOpts *map[string]string `json:"driver_opts,omitempty"`
}

// nolint
// droneVolume defines a container volume.
type droneVolume struct {
	Name       *string           `json:"name,omitempty"`
	Driver     *string           `json:"driver,omitempty"`
	DriverOpts map[string]string `json:"driver_opts,omitempty"`
}
