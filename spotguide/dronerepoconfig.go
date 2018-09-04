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
	cache     libcompose.Stringorslice
	platform  string
	branches  droneConstraint
	workspace droneWorkspace
	cluster   *droneKubernetesCluster
	clone     droneContainers
	pipeline  droneContainers
	services  droneContainers
	networks  droneNetworks
	volumes   droneVolumes
	labels    libcompose.SliceorMap
}

// nolint
// droneKubernetesCluster defines a cluster that has to be created before executing the rest of the Pipeline.
type droneKubernetesCluster struct {
	name     string `yaml:"name"`
	provider string `yaml:"provider"`
	secretID string `yaml:"secret_id"`

	googleProject    string `yaml:"google_project,omitempty"`
	googleNodeCount  int    `yaml:"google_node_count,omitempty"`
	googleGKEVersion string `yaml:"google_gke_version,omitempty"`
}

// nolint
// droneWorkspace defines a pipeline workspace.
type droneWorkspace struct {
	base string
	path string
}

// nolint
// droneAuthConfig defines registry authentication credentials.
type droneAuthConfig struct {
	username string
	password string
	email    string
}

// nolint
// droneContainers denotes an ordered collection of containers.
type droneContainers struct {
	containers []*droneContainer
}

// nolint
// droneContainer defines a container.
type droneContainer struct {
	authConfig    droneAuthConfig           `yaml:"auth_config,omitempty"`
	capAdd        []string                  `yaml:"cap_add,omitempty"`
	capDrop       []string                  `yaml:"cap_drop,omitempty"`
	command       libcompose.Command        `yaml:"command,omitempty"`
	commands      libcompose.Stringorslice  `yaml:"commands,omitempty"`
	cpuQuota      libcompose.StringorInt    `yaml:"cpu_quota,omitempty"`
	cpuSet        string                    `yaml:"cpuset,omitempty"`
	cpuShares     libcompose.StringorInt    `yaml:"cpu_shares,omitempty"`
	detached      bool                      `yaml:"detach,omitempty"`
	devices       []string                  `yaml:"devices,omitempty"`
	tmpfs         []string                  `yaml:"tmpfs,omitempty"`
	dns           libcompose.Stringorslice  `yaml:"dns,omitempty"`
	dnsSearch     libcompose.Stringorslice  `yaml:"dns_search,omitempty"`
	entrypoint    libcompose.Command        `yaml:"entrypoint,omitempty"`
	environment   libcompose.SliceorMap     `yaml:"environment,omitempty"`
	extraHosts    []string                  `yaml:"extra_hosts,omitempty"`
	group         string                    `yaml:"group,omitempty"`
	image         string                    `yaml:"image,omitempty"`
	isolation     string                    `yaml:"isolation,omitempty"`
	labels        libcompose.SliceorMap     `yaml:"labels,omitempty"`
	memLimit      libcompose.MemStringorInt `yaml:"mem_limit,omitempty"`
	memSwapLimit  libcompose.MemStringorInt `yaml:"memswap_limit,omitempty"`
	memSwappiness libcompose.MemStringorInt `yaml:"mem_swappiness,omitempty"`
	name          string                    `yaml:"name,omitempty"`
	networkMode   string                    `yaml:"network_mode,omitempty"`
	ipcMode       string                    `yaml:"ipc_mode,omitempty"`
	networks      libcompose.Networks       `yaml:"networks,omitempty"`
	ports         []int32                   `yaml:"ports,omitempty"`
	privileged    bool                      `yaml:"privileged,omitempty"`
	pull          bool                      `yaml:"pull,omitempty"`
	shmSize       libcompose.MemStringorInt `yaml:"shm_size,omitempty"`
	ulimits       libcompose.Ulimits        `yaml:"ulimits,omitempty"`
	volumes       libcompose.Volumes        `yaml:"volumes,omitempty"`
	secrets       droneSecrets              `yaml:"secrets,omitempty"`
	sysctls       libcompose.SliceorMap     `yaml:"sysctls,omitempty"`
	constraints   droneConstraints          `yaml:"when,omitempty"`
	vargs         map[string]interface{}    `yaml:",inline"`
}

// nolint
// droneConstraints defines a set of runtime constraints.
type droneConstraints struct {
	ref         droneConstraint
	repo        droneConstraint
	instance    droneConstraint
	platform    droneConstraint
	environment droneConstraint
	event       droneConstraint
	branch      droneConstraint
	status      droneConstraint
	matrix      droneConstraintMap
	local       droneBoolTrue
}

// nolint
// droneConstraint defines a runtime constraint.
type droneConstraint struct {
	include []string
	exclude []string
}

// nolint
// droneConstraintMap defines a runtime constraint map.
type droneConstraintMap struct {
	include map[string]string
	exclude map[string]string
}

// nolint
// droneNetworks defines a collection of networks.
type droneNetworks struct {
	networks []*droneNetwork
}

// nolint
// droneNetwork defines a container network.
type droneNetwork struct {
	name       string            `yaml:"name,omitempty"`
	driver     string            `yaml:"driver,omitempty"`
	driverOpts map[string]string `yaml:"driver_opts,omitempty"`
}

// nolint
// droneVolumes defines a collection of volumes.
type droneVolumes struct {
	volumes []*droneVolume
}

// nolint
// droneVolume defines a container volume.
type droneVolume struct {
	name       string            `yaml:"name,omitempty"`
	driver     string            `yaml:"driver,omitempty"`
	driverOpts map[string]string `yaml:"driver_opts,omitempty"`
}

// nolint
// droneSecrets defines a collection of secrets.
type droneSecrets struct {
	secrets []*droneSecret
}

// nolint
// droneSecret defines a container secret.
type droneSecret struct {
	source string `yaml:"source"`
	target string `yaml:"target"`
}

// droneBoolTrue is a custom Yaml boolean type that defaults to true.
type droneBoolTrue struct {
	value bool
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
		b.value = !value
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
	return !b.value
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

	c.exclude = out1.exclude
	c.include = append(
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

	c.include = out1.include
	c.exclude = out1.exclude
	for k, v := range out2 {
		c.include[k] = v
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
		if container.name == "" {
			container.name = fmt.Sprintf("%v", s.Key)
		}
		c.containers = append(c.containers, &container)
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
		if nn.name == "" {
			nn.name = fmt.Sprintf("%v", s.Key)
		}
		if nn.driver == "" {
			nn.driver = "bridge"
		}
		n.networks = append(n.networks, &nn)
	}
	return err
}

// UnmarshalYAML implements the Unmarshaller interface.
func (s *droneSecrets) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var strslice []string
	err := unmarshal(&strslice)
	if err == nil {
		for _, str := range strslice {
			s.secrets = append(s.secrets, &droneSecret{
				source: str,
				target: str,
			})
		}
		return nil
	}
	return unmarshal(&s.secrets)
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
		if vv.name == "" {
			vv.name = fmt.Sprintf("%v", s.Key)
		}
		if vv.driver == "" {
			vv.driver = "local"
		}
		v.volumes = append(v.volumes, &vv)
	}
	return err
}
