// Copyright © 2020 Banzai Cloud
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

package helm

import (
	"context"
	"strings"
	"time"

	"emperror.dev/errors"
)

// ReleaseInfo copy of the struct form the helm library
type ReleaseInfo struct {
	// FirstDeployed is when the release was first deployed.
	FirstDeployed time.Time `json:"first_deployed,omitempty"`
	// LastDeployed is when the release was last deployed.
	LastDeployed time.Time `json:"last_deployed,omitempty"`
	// Deleted tracks when this object was deleted.
	Deleted time.Time `json:"deleted"`
	// Description is human-friendly "log entry" about this release.
	Description string `json:"description,omitempty"`
	// Status is the current state of the release
	Status string
	// Contains the rendered templates/NOTES.txt if available
	Notes string
	// Contains override values provided to the release
	Values map[string]interface{}
}

type ReleaseResource struct {
	Name string `json:"name" yaml:"name"`
	Kind string `json:"kind" yaml:"kind"`
}

//  Release represents information related to a helm chart release
type Release struct {
	// ReleaseInput struct encapsulating information about the release to be created
	ReleaseName      string
	ChartName        string
	Namespace        string
	Values           map[string]interface{} // json representation
	Version          string
	ReleaseInfo      ReleaseInfo
	ReleaseVersion   int32
	ReleaseResources []ReleaseResource
}

type KubeConfigBytes = []byte

// ReleaseFilter struct for release filter data
type ReleaseFilter struct {
	TagFilter string  `json:"tag" mapstructure:"tag"`
	Filter    *string `json:"filter,omitempty" mapstructure:"filter"`
}

// releaser collects and groups helm release related operations
// it's intended to be embedded in the "Helm Facade"
// implementers are in charge to produce input for the Releaser component
type releaser interface {
	// Install installs the release to the cluster with the given identifier
	InstallRelease(ctx context.Context, organizationID uint, clusterID uint, releaseInput Release, options Options) (release Release, err error)
	// Delete deletes the  specified release
	DeleteRelease(ctx context.Context, organizationID uint, clusterID uint, releaseName string, options Options) error
	// List retrieves  releases in a given namespace, eventually applies the passed in filters
	ListReleases(ctx context.Context, organizationID uint, clusterID uint, filters ReleaseFilter, options Options) ([]Release, error)
	// Get retrieves the release details for the given  release
	GetRelease(ctx context.Context, organizationID uint, clusterID uint, releaseName string, options Options) (Release, error)
	// Upgrade upgrades the given release
	UpgradeRelease(ctx context.Context, organizationID uint, clusterID uint, releaseInput Release, options Options) (release Release, err error)
	// CheckRelease
	CheckRelease(ctx context.Context, organizationID uint, clusterID uint, releaseName string, options Options) (string, error)
	// ReleaseResources retrieves resources belonging to the release
	GetReleaseResources(ctx context.Context, organizationID uint, clusterID uint, release Release, options Options) ([]ReleaseResource, error)
}

// utility for providing input arguments ...
func (ri Release) NameAndChartSlice() []string {
	if ri.ReleaseName == "" {
		return []string{ri.ChartName}
	}
	return []string{ri.ReleaseName, ri.ChartName}
}

// Releaser interface collecting operations related to releases
// It manages releases on the cluster
type Releaser interface {
	// Install installs the specified chart using to a cluster identified by the kubeConfig  argument
	Install(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, releaseInput Release, options Options) (Release, error)
	// Uninstall removes the  specified release from the cluster
	Uninstall(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, releaseName string, options Options) error
	// List lists releases
	List(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, options Options) ([]Release, error)
	// Get gets the given release details
	Get(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, releaseInput Release, options Options) (Release, error)
	// Upgrade upgrades the given release
	Upgrade(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, releaseInput Release, options Options) (release Release, err error)
	// Resources retrieves the kubernetes resources belonging to the release
	Resources(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, releaseInput Release, options Options) ([]ReleaseResource, error)
}

func ErrReleaseNotFound(err error) bool {
	return strings.Contains(errors.Cause(err).Error(), "not found")
}

// ReleaseDeleter abstraction of the operations
type ReleaseDeleter interface {
	// DeleteReleases deletes all releases in the provided namespaces; all namespaces are considered when no namespaces provided
	DeleteReleases(ctx context.Context, orgID uint, kubeConfig []byte, namespaces []string) error
}

type ListerUninstaller interface {
	Uninstall(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, releaseName string, options Options) error
	List(ctx context.Context, helmEnv HelmEnv, kubeConfig KubeConfigBytes, options Options) ([]Release, error)
}
