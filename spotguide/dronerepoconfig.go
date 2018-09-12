package spotguide

import (
	libcompose "github.com/docker/libcompose/yaml"
)

// nolint
// droneRepoConfig defines a pipeline configuration.
type droneRepoConfig struct {
	Cache     libcompose.Stringorslice   `yaml:"cache,omitempty"`
	Platform  *string                    `yaml:"platform,omitempty"`
	Branches  *droneConstraint           `yaml:"branches,omitempty"`
	Workspace *droneWorkspace            `yaml:"workspace,omitempty"`
	Cluster   *droneKubernetesCluster    `yaml:"cluster,omitempty"`
	Clone     map[string]*droneContainer `yaml:"clone,omitempty"`
	Pipeline  map[string]*droneContainer `yaml:"pipeline,omitempty"`
	Services  map[string]*droneContainer `yaml:"services,omitempty"`
	Networks  map[string]*droneNetwork   `yaml:"networks,omitempty"`
	Volumes   map[string]*droneVolume    `yaml:"volumes,omitempty"`
	Labels    libcompose.SliceorMap      `yaml:"labels,omitempty"`
}

// nolint
// droneKubernetesCluster defines a cluster that has to be created before executing the rest of the Pipeline.
type droneKubernetesCluster struct {
	Name     *string `yaml:"name,omitempty"`
	Provider *string `yaml:"provider,omitempty"`
	SecretID *string `yaml:"secret_id,omitempty"`

	GoogleProject    *string `yaml:"google_project,omitempty"`
	GoogleNodeCount  *int    `yaml:"google_node_count,omitempty"`
	GoogleGKEVersion *string `yaml:"google_gke_version,omitempty"`
}

// nolint
// droneWorkspace defines a pipeline workspace.
type droneWorkspace struct {
	Base *string `yaml:"base,omitempty"`
	Path *string `yaml:"path,omitempty"`
}

// nolint
// droneAuthConfig defines registry authentication credentials.
type droneAuthConfig struct {
	Username *string `yaml:"username,omitempty"`
	Password *string `yaml:"password,omitempty"`
	Email    *string `yaml:"email,omitempty"`
}

// nolint
// droneContainer defines a container.
type droneContainer struct {
	AuthConfig    *droneAuthConfig           `yaml:"auth_config,omitempty"`
	CapAdd        []string                   `yaml:"cap_add,omitempty"`
	CapDrop       []string                   `yaml:"cap_drop,omitempty"`
	Command       libcompose.Command         `yaml:"command,omitempty"`
	Commands      libcompose.Stringorslice   `yaml:"commands,omitempty"`
	CpuQuota      *libcompose.StringorInt    `yaml:"cpu_quota,omitempty"`
	CpuSet        *string                    `yaml:"cpuset,omitempty"`
	CpuShares     *libcompose.StringorInt    `yaml:"cpu_shares,omitempty"`
	Detached      *bool                      `yaml:"detach,omitempty"`
	Devices       []string                   `yaml:"devices,omitempty"`
	Tmpfs         []string                   `yaml:"tmpfs,omitempty"`
	Dns           libcompose.Stringorslice   `yaml:"dns,omitempty"`
	DnsSearch     libcompose.Stringorslice   `yaml:"dns_search,omitempty"`
	Entrypoint    libcompose.Command         `yaml:"entrypoint,omitempty"`
	Environment   libcompose.SliceorMap      `yaml:"environment,omitempty"`
	ExtraHosts    []string                   `yaml:"extra_hosts,omitempty"`
	Group         *string                    `yaml:"group,omitempty"`
	Image         *string                    `yaml:"image,omitempty"`
	Isolation     *string                    `yaml:"isolation,omitempty"`
	Labels        libcompose.SliceorMap      `yaml:"labels,omitempty"`
	MemLimit      *libcompose.MemStringorInt `yaml:"mem_limit,omitempty"`
	MemSwapLimit  *libcompose.MemStringorInt `yaml:"memswap_limit,omitempty"`
	MemSwappiness *libcompose.MemStringorInt `yaml:"mem_swappiness,omitempty"`
	Name          *string                    `yaml:"name,omitempty"`
	NetworkMode   *string                    `yaml:"network_mode,omitempty"`
	IpcMode       *string                    `yaml:"ipc_mode,omitempty"`
	Networks      *libcompose.Networks       `yaml:"networks,omitempty"`
	Ports         []int32                    `yaml:"ports,omitempty"`
	Privileged    *bool                      `yaml:"privileged,omitempty"`
	Pull          *bool                      `yaml:"pull,omitempty"`
	ShmSize       *libcompose.MemStringorInt `yaml:"shm_size,omitempty"`
	Ulimits       *libcompose.Ulimits        `yaml:"ulimits,omitempty"`
	Volumes       *libcompose.Volumes        `yaml:"volumes,omitempty"`
	Secrets       []string                   `yaml:"secrets,omitempty"`
	Sysctls       libcompose.SliceorMap      `yaml:"sysctls,omitempty"`
	Constraints   *droneConstraints          `yaml:"when,omitempty"`
	Vargs         map[string]interface{}     `yaml:",inline,omitempty"`
	Dockerfile    *string                    `yaml:"dockerfile,omitempty"`
	Repo          *string                    `yaml:"repo,omitempty"`
	Tags          *string                    `yaml:"tags,omitempty"`
	Log           *string                    `yaml:"log,omitempty"`
	Deployment    *droneDeploymentContainer  `yaml:"deployment,omitempty"`
}

type droneDeploymentContainer struct {
	Name        string                 `yaml:"name,omitempty"`
	ReleaseName string                 `yaml:"releaseName,omitempty"`
	Values      map[string]interface{} `yaml:"values,omitempty"`
}

// nolint
// droneConstraints defines a set of runtime constraints.
type droneConstraints struct {
	Ref         *droneConstraint    `yaml:"ref,omitempty"`
	Repo        *droneConstraint    `yaml:"repo,omitempty"`
	Instance    *droneConstraint    `yaml:"instance,omitempty"`
	Platform    *droneConstraint    `yaml:"platform,omitempty"`
	Environment *droneConstraint    `yaml:"environment,omitempty"`
	Event       *droneConstraint    `yaml:"event,omitempty"`
	Branch      *droneConstraint    `yaml:"branch,omitempty"`
	Status      *droneConstraint    `yaml:"status,omitempty"`
	Matrix      *droneConstraintMap `yaml:"matrix,omitempty"`
	Local       *bool               `yaml:"local,omitempty"`
}

// nolint
// droneConstraint defines a runtime constraint.
type droneConstraint struct {
	Include libcompose.Stringorslice `yaml:"include,omitempty"`
	Exclude libcompose.Stringorslice `yaml:"exclude,omitempty"`
}

// nolint
// droneConstraintMap defines a runtime constraint map.
type droneConstraintMap struct {
	Include map[string]string `yaml:"include,omitempty"`
	Exclude map[string]string `yaml:"exclude,omitempty"`
}

// nolint
// droneNetwork defines a container network.
type droneNetwork struct {
	Name       *string            `yaml:"name,omitempty"`
	Driver     *string            `yaml:"driver,omitempty"`
	DriverOpts *map[string]string `yaml:"driver_opts,omitempty"`
}

// nolint
// droneVolume defines a container volume.
type droneVolume struct {
	Name       *string           `yaml:"name,omitempty"`
	Driver     *string           `yaml:"driver,omitempty"`
	DriverOpts map[string]string `yaml:"driver_opts,omitempty"`
}
