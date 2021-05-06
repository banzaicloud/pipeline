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

package ark

import (
	"encoding/json"
	"fmt"

	"emperror.dev/errors"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1/backup"
	v1 "k8s.io/api/core/v1"

	"github.com/banzaicloud/pipeline/internal/ark/client"
	"github.com/banzaicloud/pipeline/internal/ark/providers/amazon"
	"github.com/banzaicloud/pipeline/internal/ark/providers/azure"
	"github.com/banzaicloud/pipeline/internal/ark/providers/google"
	"github.com/banzaicloud/pipeline/internal/global"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/src/secret"
)

// ChartConfig describes an ARK deployment chart config
type ChartConfig struct {
	Namespace      string
	Chart          string
	Name           string
	Version        string
	ValueOverrides []byte
}

// ConfigRequest describes an ARK config request
type ConfigRequest struct {
	Cluster       clusterConfig
	ClusterSecret *secret.SecretItemResponse
	Bucket        bucketConfig
	BucketSecret  *secret.SecretItemResponse
	SecretName    string

	UseClusterSecret      bool
	ServiceAccountRoleARN string
	UseProviderSecret     bool
	RestoreMode           bool
}

type clusterConfig struct {
	Name         string
	Provider     string
	Distribution string
	Location     string
	RBACEnabled  bool

	azureClusterConfig
}

type azureClusterConfig struct {
	ResourceGroup string
}

type bucketConfig struct {
	Name     string
	Prefix   string
	Provider string
	Location string

	azureBucketConfig
}

type azureBucketConfig struct {
	StorageAccount string
	ResourceGroup  string
}

type HelmValueOverrides struct {
	backup.ValueOverrides
	Image          Image          `json:"image,omitempty"`
	InitContainers []v1.Container `json:"initContainers,omitempty"`
}

type Image struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	PullPolicy string `json:"pullPolicy"`
}

// GetChartConfig get a ChartConfig
func GetChartConfig() ChartConfig {
	return ChartConfig{
		Name:      "velero",
		Namespace: global.Config.Cluster.DisasterRecovery.Namespace,
		Chart:     global.Config.Cluster.DisasterRecovery.Charts.Ark.Chart,
		Version:   global.Config.Cluster.DisasterRecovery.Charts.Ark.Version,
	}
}

func (req ConfigRequest) getInitContainers(bsp backup.BackupStorageLocation, vsl backup.VolumeSnapshotLocation) []v1.Container {
	initContainers := make([]v1.Container, 0, 2)

	if bsp.Provider == amazon.BackupStorageProvider || vsl.Provider == amazon.PersistentVolumeProvider {
		pluginImage := fmt.Sprintf("%s:%s", global.Config.Cluster.DisasterRecovery.Charts.Ark.Values.AwsPluginImage.Repository,
			global.Config.Cluster.DisasterRecovery.Charts.Ark.Values.AwsPluginImage.Tag)

		initContainers = append(initContainers, v1.Container{
			Name:            "velero-plugin-for-aws",
			Image:           pluginImage,
			ImagePullPolicy: getPullPolicy(global.Config.Cluster.DisasterRecovery.Charts.Ark.Values.AwsPluginImage.PullPolicy),
			VolumeMounts: []v1.VolumeMount{
				{
					Name:      "plugins",
					MountPath: "/target",
				},
			},
		})
	}

	if bsp.Provider == google.BackupStorageProvider || vsl.Provider == google.PersistentVolumeProvider {
		pluginImage := fmt.Sprintf("%s:%s", global.Config.Cluster.DisasterRecovery.Charts.Ark.Values.GcpPluginImage.Repository,
			global.Config.Cluster.DisasterRecovery.Charts.Ark.Values.GcpPluginImage.Tag)

		initContainers = append(initContainers, v1.Container{
			Name:            "velero-plugin-for-gcp",
			Image:           pluginImage,
			ImagePullPolicy: getPullPolicy(global.Config.Cluster.DisasterRecovery.Charts.Ark.Values.GcpPluginImage.PullPolicy),
			VolumeMounts: []v1.VolumeMount{
				{
					Name:      "plugins",
					MountPath: "/target",
				},
			},
		})
	}

	if bsp.Provider == azure.BackupStorageProvider || vsl.Provider == azure.PersistentVolumeProvider {
		pluginImage := fmt.Sprintf("%s:%s", global.Config.Cluster.DisasterRecovery.Charts.Ark.Values.AzurePluginImage.Repository,
			global.Config.Cluster.DisasterRecovery.Charts.Ark.Values.AzurePluginImage.Tag)

		initContainers = append(initContainers, v1.Container{
			Name:            "velero-plugin-for-azure",
			Image:           pluginImage,
			ImagePullPolicy: getPullPolicy(global.Config.Cluster.DisasterRecovery.Charts.Ark.Values.AzurePluginImage.PullPolicy),
			VolumeMounts: []v1.VolumeMount{
				{
					Name:      "plugins",
					MountPath: "/target",
				},
			},
		})
	}
	return initContainers
}

// Get gets helm deployment value overrides
func (req ConfigRequest) Get() (values backup.ValueOverrides, err error) {
	var provider backup.Provider
	switch req.Bucket.Provider {
	case providers.Amazon:
		provider = backup.AWSProvider
	case providers.Azure:
		provider = backup.AzureProvider
	case providers.Google:
		provider = backup.GCPProvider
	default:
		return values, pkgErrors.ErrorNotSupportedCloudType
	}

	vsl, err := req.getVolumeSnapshotLocation()
	if err != nil {
		return values, err
	}

	bsp, err := req.getBackupStorageLocation()
	if err != nil {
		return values, err
	}

	values = backup.ValueOverrides{
		Configuration: backup.Configuration{
			Provider:               provider,
			VolumeSnapshotLocation: vsl,
			BackupStorageLocation:  bsp,
			RestoreOnlyMode:        req.RestoreMode,
			LogLevel:               "debug",
		},
		RBAC: backup.Rbac{
			Create: req.Cluster.RBACEnabled,
		},
		Credentials: backup.Credentials{
			ExistingSecret: req.SecretName,
		},
		// cleanup crd's only in restore mode
		CleanUpCRDs: req.RestoreMode,
		ServiceAccount: backup.ServiceAccount{
			Server: backup.Server{
				Create: true,
			},
		},
	}

	if vsl.Provider == amazon.PersistentVolumeProvider && req.ServiceAccountRoleARN != "" {
		values.ServiceAccount = backup.ServiceAccount{
			Server: backup.Server{
				Create: true,
				Name:   "velero-sa",
				Annotations: map[string]string{
					"eks.amazonaws.com/role-arn": req.ServiceAccountRoleARN,
				},
			},
		}
		values.SecurityContext = backup.SecurityContext{
			FsGroup: 1337,
		}
	}

	return values, nil
}

func (req *ConfigRequest) getChartConfig() (config ChartConfig, err error) {
	config = GetChartConfig()

	arkConfig, err := req.Get()
	if err != nil {
		err = errors.Wrap(err, "error getting config")
		return
	}

	helmConfig := HelmValueOverrides{
		ValueOverrides: arkConfig,
	}

	// in case of deploying as an integrated service, image versions below are set by IS operator
	helmConfig.Image = Image{
		Repository: global.Config.Cluster.DisasterRecovery.Charts.Ark.Values.Image.Repository,
		Tag:        global.Config.Cluster.DisasterRecovery.Charts.Ark.Values.Image.Tag,
		PullPolicy: global.Config.Cluster.DisasterRecovery.Charts.Ark.Values.Image.PullPolicy,
	}
	helmConfig.InitContainers = req.getInitContainers(helmConfig.Configuration.BackupStorageLocation,
		helmConfig.Configuration.VolumeSnapshotLocation)

	json, err := json.Marshal(helmConfig)
	if err != nil {
		err = errors.Wrap(err, "json convert failed")
		return
	}

	config.ValueOverrides = json

	return
}

func (req ConfigRequest) getVolumeSnapshotLocation() (backup.VolumeSnapshotLocation, error) {
	var config backup.VolumeSnapshotLocation
	var vslconfig backup.VolumeSnapshotLocationConfig
	var pvcProvider backup.Provider

	switch req.Cluster.Provider {
	case providers.Amazon:
		pvcProvider = amazon.PersistentVolumeProvider
		vslconfig.Region = req.Cluster.Location
		if req.UseProviderSecret {
			vslconfig.Profile = "bucket"
		}
	case providers.Azure:
		pvcProvider = azure.PersistentVolumeProvider
		vslconfig.ApiTimeout = "3m0s"
		vslconfig.ResourceGroup = azure.GetAzureClusterResourceGroupName(req.Cluster.Distribution, req.Cluster.ResourceGroup, req.Cluster.Name, req.Cluster.Location)
	case providers.Google:
		pvcProvider = google.PersistentVolumeProvider
	default:
		return config, pkgErrors.ErrorNotSupportedCloudType
	}

	return backup.VolumeSnapshotLocation{
		Name:     client.DefaultVolumeSnapshotLocationName,
		Provider: pvcProvider,
		Config:   vslconfig,
	}, nil
}

func (req ConfigRequest) getBackupStorageLocation() (backup.BackupStorageLocation, error) {
	config := backup.BackupStorageLocation{
		Name:   client.DefaultBackupStorageLocationName,
		Bucket: req.Bucket.Name,
		Prefix: req.Bucket.Prefix,
	}

	switch req.Bucket.Provider {
	case providers.Amazon:
		config.Provider = amazon.BackupStorageProvider
		config.Config.Region = req.Bucket.Location
		config.Config.Profile = "bucket"

	case providers.Azure:
		config.Provider = azure.BackupStorageProvider
		config.Config.StorageAccount = req.Bucket.StorageAccount
		config.Config.ResourceGroup = req.Bucket.ResourceGroup
		config.Config.StorageAccountKeyEnvVar = "AZURE_STORAGE_KEY"

	case providers.Google:
		config.Provider = google.BackupStorageProvider

	default:
		return config, pkgErrors.ErrorNotSupportedCloudType
	}

	return config, nil
}

func getPullPolicy(pullPolicy string) v1.PullPolicy {
	switch pullPolicy {
	case string(v1.PullAlways), string(v1.PullIfNotPresent), string(v1.PullNever): // Note: known values.
		return v1.PullPolicy(pullPolicy)
	default:
		return v1.PullIfNotPresent
	}
}
