// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package spotguide

import (
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	libcompose "github.com/docker/libcompose/yaml"
	yaml2 "github.com/ghodss/yaml"
	yaml "gopkg.in/yaml.v2"
)

// nolint
// cicdRepoConfig defines a pipeline configuration.
type cicdRepoConfig struct {
	Cache     libcompose.Stringorslice  `yaml:"cache,omitempty"`
	Platform  *string                   `yaml:"platform,omitempty"`
	Branches  *cicdConstraint           `yaml:"branches,omitempty"`
	Workspace *cicdWorkspace            `yaml:"workspace,omitempty"`
	Clone     map[string]*cicdContainer `yaml:"clone,omitempty"`
	Cluster   map[string]interface{}    `yaml:"cluster,omitempty"`
	Pipeline  yaml.MapSlice             `yaml:"pipeline,omitempty"` // map[string]*cicdContainer
	Services  map[string]*cicdContainer `yaml:"services,omitempty"`
	Networks  map[string]*cicdNetwork   `yaml:"networks,omitempty"`
	Volumes   map[string]*cicdVolume    `yaml:"volumes,omitempty"`
	Labels    libcompose.SliceorMap     `yaml:"labels,omitempty"`
}

// nolint
// cicdWorkspace defines a pipeline workspace.
type cicdWorkspace struct {
	Base *string `yaml:"base,omitempty"`
	Path *string `yaml:"path,omitempty"`
}

// nolint
// cicdAuthConfig defines registry authentication credentials.
type cicdAuthConfig struct {
	Username *string `yaml:"username,omitempty"`
	Password *string `yaml:"password,omitempty"`
	Email    *string `yaml:"email,omitempty"`
}

// nolint
// cicdContainer defines a container.
type cicdContainer struct {
	AuthConfig    *cicdAuthConfig                        `yaml:"auth_config,omitempty"`
	CapAdd        []string                               `yaml:"cap_add,omitempty"`
	CapDrop       []string                               `yaml:"cap_drop,omitempty"`
	Command       libcompose.Command                     `yaml:"command,omitempty"`
	Commands      libcompose.Stringorslice               `yaml:"commands,omitempty"`
	CpuQuota      *libcompose.StringorInt                `yaml:"cpu_quota,omitempty"`
	CpuSet        *string                                `yaml:"cpuset,omitempty"`
	CpuShares     *libcompose.StringorInt                `yaml:"cpu_shares,omitempty"`
	Detached      *bool                                  `yaml:"detach,omitempty"`
	Devices       []string                               `yaml:"devices,omitempty"`
	Tmpfs         []string                               `yaml:"tmpfs,omitempty"`
	Dns           libcompose.Stringorslice               `yaml:"dns,omitempty"`
	DnsSearch     libcompose.Stringorslice               `yaml:"dns_search,omitempty"`
	Entrypoint    libcompose.Command                     `yaml:"entrypoint,omitempty"`
	Environment   libcompose.SliceorMap                  `yaml:"environment,omitempty"`
	ExtraHosts    []string                               `yaml:"extra_hosts,omitempty"`
	Group         *string                                `yaml:"group,omitempty"`
	Image         *string                                `yaml:"image,omitempty"`
	Isolation     *string                                `yaml:"isolation,omitempty"`
	Labels        libcompose.SliceorMap                  `yaml:"labels,omitempty"`
	MemLimit      *libcompose.MemStringorInt             `yaml:"mem_limit,omitempty"`
	MemSwapLimit  *libcompose.MemStringorInt             `yaml:"memswap_limit,omitempty"`
	MemSwappiness *libcompose.MemStringorInt             `yaml:"mem_swappiness,omitempty"`
	Name          *string                                `yaml:"name,omitempty"`
	NetworkMode   *string                                `yaml:"network_mode,omitempty"`
	IpcMode       *string                                `yaml:"ipc_mode,omitempty"`
	Networks      *libcompose.Networks                   `yaml:"networks,omitempty"`
	Ports         []int32                                `yaml:"ports,omitempty"`
	Privileged    *bool                                  `yaml:"privileged,omitempty"`
	Pull          *bool                                  `yaml:"pull,omitempty"`
	ShmSize       *libcompose.MemStringorInt             `yaml:"shm_size,omitempty"`
	Ulimits       *libcompose.Ulimits                    `yaml:"ulimits,omitempty"`
	Volumes       *libcompose.Volumes                    `yaml:"volumes,omitempty"`
	Secrets       []string                               `yaml:"secrets,omitempty"`
	Sysctls       libcompose.SliceorMap                  `yaml:"sysctls,omitempty"`
	Constraints   *cicdConstraints                       `yaml:"when,omitempty"`
	Vargs         map[string]interface{}                 `yaml:",inline,omitempty"`
	Dockerfile    *string                                `yaml:"dockerfile,omitempty"`
	Repo          *string                                `yaml:"repo,omitempty"`
	Tags          *string                                `yaml:"tags,omitempty"`
	Log           *string                                `yaml:"log,omitempty"`
	Cluster       *pkgCluster.CreateClusterRequest       `yaml:"cluster,omitempty"`
	Deployment    *pkgHelm.CreateUpdateDeploymentRequest `yaml:"deployment,omitempty"`
	ClusterSecret map[string]interface{}                 `yaml:"cluster_secret,omitempty"`
}

// nolint
// cicdConstraints defines a set of runtime constraints.
type cicdConstraints struct {
	Ref         *cicdConstraint    `yaml:"ref,omitempty"`
	Repo        *cicdConstraint    `yaml:"repo,omitempty"`
	Instance    *cicdConstraint    `yaml:"instance,omitempty"`
	Platform    *cicdConstraint    `yaml:"platform,omitempty"`
	Environment *cicdConstraint    `yaml:"environment,omitempty"`
	Event       *cicdConstraint    `yaml:"event,omitempty"`
	Branch      *cicdConstraint    `yaml:"branch,omitempty"`
	Status      *cicdConstraint    `yaml:"status,omitempty"`
	Matrix      *cicdConstraintMap `yaml:"matrix,omitempty"`
	Local       *bool              `yaml:"local,omitempty"`
}

// nolint
// cicdConstraint defines a runtime constraint.
type cicdConstraint struct {
	Include libcompose.Stringorslice `yaml:"include,omitempty"`
	Exclude libcompose.Stringorslice `yaml:"exclude,omitempty"`
}

// nolint
// cicdConstraintMap defines a runtime constraint map.
type cicdConstraintMap struct {
	Include map[string]string `yaml:"include,omitempty"`
	Exclude map[string]string `yaml:"exclude,omitempty"`
}

// nolint
// cicdNetwork defines a container network.
type cicdNetwork struct {
	Name       *string            `yaml:"name,omitempty"`
	Driver     *string            `yaml:"driver,omitempty"`
	DriverOpts *map[string]string `yaml:"driver_opts,omitempty"`
}

// nolint
// cicdVolume defines a container volume.
type cicdVolume struct {
	Name       *string           `yaml:"name,omitempty"`
	Driver     *string           `yaml:"driver,omitempty"`
	DriverOpts map[string]string `yaml:"driver_opts,omitempty"`
}

// Functions to transform cicdContainer vs. yaml.MapSlice and vice-versa
func copyToCICDContainer(v interface{}) (*cicdContainer, error) {
	bytes, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	var config cicdContainer
	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func cicdContainerToMapSlice(container *cicdContainer) (yaml.MapSlice, error) {
	bytes, err := yaml.Marshal(container)
	if err != nil {
		return nil, err
	}
	var mapSlice yaml.MapSlice
	err = yaml.Unmarshal(bytes, &mapSlice)
	if err != nil {
		return nil, err
	}
	return mapSlice, nil
}

func yamlMapSliceToMap(v interface{}) (map[string]interface{}, error) {
	bytes, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	var config map[string]interface{}
	err = yaml2.Unmarshal(bytes, &config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func mapToYamlMapSlice(container interface{}) (yaml.MapSlice, error) {
	bytes, err := yaml.Marshal(container)
	if err != nil {
		return nil, err
	}
	var mapSlice yaml.MapSlice
	err = yaml.Unmarshal(bytes, &mapSlice)
	if err != nil {
		return nil, err
	}
	return mapSlice, nil
}
