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
	"encoding/json"

	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgHelm "github.com/banzaicloud/pipeline/pkg/helm"
	libcompose "github.com/docker/libcompose/yaml"
	yaml2 "github.com/ghodss/yaml"
	yaml "gopkg.in/yaml.v2"
)

// nolint
// droneRepoConfig defines a pipeline configuration.
type droneRepoConfig struct {
	Cache     libcompose.Stringorslice   `yaml:"cache,omitempty"`
	Platform  *string                    `yaml:"platform,omitempty"`
	Branches  *droneConstraint           `yaml:"branches,omitempty"`
	Workspace *droneWorkspace            `yaml:"workspace,omitempty"`
	Clone     map[string]*droneContainer `yaml:"clone,omitempty"`
	Pipeline  yaml.MapSlice              `yaml:"pipeline,omitempty"` // map[string]*droneContainer
	Services  map[string]*droneContainer `yaml:"services,omitempty"`
	Networks  map[string]*droneNetwork   `yaml:"networks,omitempty"`
	Volumes   map[string]*droneVolume    `yaml:"volumes,omitempty"`
	Labels    libcompose.SliceorMap      `yaml:"labels,omitempty"`
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
	AuthConfig    *droneAuthConfig                       `yaml:"auth_config,omitempty"`
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
	Constraints   *droneConstraints                      `yaml:"when,omitempty"`
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

// Functions to transform droneContainer vs. yaml.MapSlice and vice-versa
func copyToDroneContainer(v interface{}) (*droneContainer, error) {
	bytes, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	var config droneContainer
	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func droneContainerToMapSlice(container *droneContainer) (yaml.MapSlice, error) {
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

func copyToMap(v interface{}) (map[string]interface{}, error) {
	bytes, err := yaml.Marshal(v)
	if err != nil {
		return nil, err
	}
	var config map[string]interface{}
	err = yaml2.Unmarshal(bytes, &config)
	if err != nil {
		return nil, err
	}
	bytes, err = json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var config2 map[string]interface{}
	err = json.Unmarshal(bytes, &config2)
	if err != nil {
		return nil, err
	}
	return config2, nil
}

func mapToMapSlice(container map[string]interface{}) (yaml.MapSlice, error) {
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
