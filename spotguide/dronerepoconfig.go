package spotguide

import (
	"fmt"
	"strconv"

	libcompose "github.com/docker/libcompose/yaml"
	"gopkg.in/yaml.v2"
)

// nolint
// droneRepoConfig defines a pipeline configuration.
type droneRepoConfig struct {
	Cache     libcompose.Stringorslice
	Platform  string
	Branches  droneConstraint
	Workspace droneWorkspace
	Cluster   *droneKubernetesCluster
	Clone     droneContainers
	Pipeline  droneContainers
	Services  droneContainers
	Networks  droneNetworks
	Volumes   droneVolumes
	Labels    libcompose.SliceorMap
}

// nolint
// droneKubernetesCluster defines a cluster that has to be created before executing the rest of the Pipeline.
type droneKubernetesCluster struct {
	Name     string `yaml:"name"`
	Provider string `yaml:"provider"`
	SecretID string `yaml:"secret_id"`

	GoogleProject    string `yaml:"google_project,omitempty"`
	GoogleNodeCount  int    `yaml:"google_node_count,omitempty"`
	GoogleGKEVersion string `yaml:"google_gke_version,omitempty"`
}

// nolint
// droneWorkspace defines a pipeline workspace.
type droneWorkspace struct {
	Base string
	Path string
}

// nolint
// droneAuthConfig defines registry authentication credentials.
type droneAuthConfig struct {
	Username string
	Password string
	Email    string
}

// nolint
// droneContainers denotes an ordered collection of containers.
type droneContainers struct {
	Containers []*droneContainer
}

// nolint
// droneContainer defines a container.
type droneContainer struct {
	AuthConfig    droneAuthConfig           `yaml:"auth_config,omitempty"`
	CapAdd        []string                  `yaml:"cap_add,omitempty"`
	CapDrop       []string                  `yaml:"cap_drop,omitempty"`
	Command       libcompose.Command        `yaml:"command,omitempty"`
	Commands      libcompose.Stringorslice  `yaml:"commands,omitempty"`
	CpuQuota      libcompose.StringorInt    `yaml:"cpu_quota,omitempty"`
	CpuSet        string                    `yaml:"cpuset,omitempty"`
	CpuShares     libcompose.StringorInt    `yaml:"cpu_shares,omitempty"`
	Detached      bool                      `yaml:"detach,omitempty"`
	Devices       []string                  `yaml:"devices,omitempty"`
	Tmpfs         []string                  `yaml:"tmpfs,omitempty"`
	Dns           libcompose.Stringorslice  `yaml:"dns,omitempty"`
	DnsSearch     libcompose.Stringorslice  `yaml:"dns_search,omitempty"`
	Entrypoint    libcompose.Command        `yaml:"entrypoint,omitempty"`
	Environment   libcompose.SliceorMap     `yaml:"environment,omitempty"`
	ExtraHosts    []string                  `yaml:"extra_hosts,omitempty"`
	Group         string                    `yaml:"group,omitempty"`
	Image         string                    `yaml:"image,omitempty"`
	Isolation     string                    `yaml:"isolation,omitempty"`
	Labels        libcompose.SliceorMap     `yaml:"labels,omitempty"`
	MemLimit      libcompose.MemStringorInt `yaml:"mem_limit,omitempty"`
	MemSwapLimit  libcompose.MemStringorInt `yaml:"memswap_limit,omitempty"`
	MemSwappiness libcompose.MemStringorInt `yaml:"mem_swappiness,omitempty"`
	Name          string                    `yaml:"name,omitempty"`
	NetworkMode   string                    `yaml:"network_mode,omitempty"`
	IpcMode       string                    `yaml:"ipc_mode,omitempty"`
	Networks      libcompose.Networks       `yaml:"networks,omitempty"`
	Ports         []int32                   `yaml:"ports,omitempty"`
	Privileged    bool                      `yaml:"privileged,omitempty"`
	Pull          bool                      `yaml:"pull,omitempty"`
	ShmSize       libcompose.MemStringorInt `yaml:"shm_size,omitempty"`
	Ulimits       libcompose.Ulimits        `yaml:"ulimits,omitempty"`
	Volumes       libcompose.Volumes        `yaml:"volumes,omitempty"`
	Secrets       droneSecrets              `yaml:"secrets,omitempty"`
	Sysctls       libcompose.SliceorMap     `yaml:"sysctls,omitempty"`
	Constraints   droneConstraints          `yaml:"when,omitempty"`
	Vargs         map[string]interface{}    `yaml:",inline"`
}

// nolint
// droneConstraints defines a set of runtime constraints.
type droneConstraints struct {
	Ref         droneConstraint
	Repo        droneConstraint
	Instance    droneConstraint
	Platform    droneConstraint
	Environment droneConstraint
	Event       droneConstraint
	Branch      droneConstraint
	Status      droneConstraint
	Matrix      droneConstraintMap
	Local       droneBoolTrue
}

// nolint
// droneConstraint defines a runtime constraint.
type droneConstraint struct {
	Include []string
	Exclude []string
}

// nolint
// droneConstraintMap defines a runtime constraint map.
type droneConstraintMap struct {
	Include map[string]string
	Exclude map[string]string
}

// nolint
// droneNetworks defines a collection of networks.
type droneNetworks struct {
	Networks []*droneNetwork
}

// nolint
// droneNetwork defines a container network.
type droneNetwork struct {
	Name       string            `yaml:"name,omitempty"`
	Driver     string            `yaml:"driver,omitempty"`
	DriverOpts map[string]string `yaml:"driver_opts,omitempty"`
}

// nolint
// droneVolumes defines a collection of volumes.
type droneVolumes struct {
	Volumes []*droneVolume
}

// nolint
// droneVolume defines a container volume.
type droneVolume struct {
	Name       string            `yaml:"name,omitempty"`
	Driver     string            `yaml:"driver,omitempty"`
	DriverOpts map[string]string `yaml:"driver_opts,omitempty"`
}

// nolint
// droneSecrets defines a collection of secrets.
type droneSecrets struct {
	Secrets []*droneSecret
}

// nolint
// droneSecret defines a container secret.
type droneSecret struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

// droneBoolTrue is a custom Yaml boolean type that defaults to true.
type droneBoolTrue struct {
	Value bool
}

// UnmarshalYAML implements custom Yaml unmarshaling.
func (b *droneBoolTrue) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err != nil {
		return err
	}

	value, err := strconv.ParseBool(s)
	if err == nil {
		b.Value = !value
	}
	return nil
}

// MarshalYAML implements custom Yaml marshaling.
func (b *droneBoolTrue) MarshalYAML() (interface{}, error) {
	value := strconv.FormatBool(b.Bool())
	return value, nil
}

// Bool returns the bool value.
func (b droneBoolTrue) Bool() bool {
	return !b.Value
}

// UnmarshalYAML unmarshals the constraint.
func (c *droneConstraint) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var out1 = struct {
		include libcompose.Stringorslice
		exclude libcompose.Stringorslice
	}{}

	var out2 libcompose.Stringorslice

	unmarshal(&out1)
	unmarshal(&out2)

	c.Exclude = out1.exclude
	c.Include = append(
		out1.include,
		out2...,
	)
	return nil
}

// UnmarshalYAML unmarshals the constraint map.
func (c *droneConstraintMap) UnmarshalYAML(unmarshal func(interface{}) error) error {
	out1 := struct {
		include map[string]string
		exclude map[string]string
	}{
		include: map[string]string{},
		exclude: map[string]string{},
	}

	out2 := map[string]string{}

	unmarshal(&out1)
	unmarshal(&out2)

	c.Include = out1.include
	c.Exclude = out1.exclude
	for k, v := range out2 {
		c.Include[k] = v
	}
	return nil
}

// UnmarshalYAML implements the Unmarshaller interface.
func (c *droneContainers) UnmarshalYAML(unmarshal func(interface{}) error) error {
	slice := yaml.MapSlice{}
	if err := unmarshal(&slice); err != nil {
		return err
	}

	for _, s := range slice {
		container := droneContainer{}
		out, _ := yaml.Marshal(s.Value)

		if err := yaml.Unmarshal(out, &container); err != nil {
			return err
		}
		if container.Name == "" {
			container.Name = fmt.Sprintf("%v", s.Key)
		}
		c.Containers = append(c.Containers, &container)
	}
	return nil
}

// UnmarshalYAML implements the Unmarshaller interface.
func (n *droneNetworks) UnmarshalYAML(unmarshal func(interface{}) error) error {
	slice := yaml.MapSlice{}
	err := unmarshal(&slice)
	if err != nil {
		return err
	}

	for _, s := range slice {
		nn := droneNetwork{}
		out, _ := yaml.Marshal(s.Value)

		err = yaml.Unmarshal(out, &nn)
		if err != nil {
			return err
		}
		if nn.Name == "" {
			nn.Name = fmt.Sprintf("%v", s.Key)
		}
		if nn.Driver == "" {
			nn.Driver = "bridge"
		}
		n.Networks = append(n.Networks, &nn)
	}
	return err
}

// UnmarshalYAML implements the Unmarshaller interface.
func (s *droneSecrets) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var strslice []string
	err := unmarshal(&strslice)
	if err == nil {
		for _, str := range strslice {
			s.Secrets = append(s.Secrets, &droneSecret{
				Source: str,
				Target: str,
			})
		}
		return nil
	}
	return unmarshal(&s.Secrets)
}

// UnmarshalYAML implements the Unmarshaller interface.
func (v *droneVolumes) UnmarshalYAML(unmarshal func(interface{}) error) error {
	slice := yaml.MapSlice{}
	err := unmarshal(&slice)
	if err != nil {
		return err
	}

	for _, s := range slice {
		vv := droneVolume{}
		out, _ := yaml.Marshal(s.Value)

		err = yaml.Unmarshal(out, &vv)
		if err != nil {
			return err
		}
		if vv.Name == "" {
			vv.Name = fmt.Sprintf("%v", s.Key)
		}
		if vv.Driver == "" {
			vv.Driver = "local"
		}
		v.Volumes = append(v.Volumes, &vv)
	}
	return err
}
