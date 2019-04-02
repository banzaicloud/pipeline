// Copyright © 2019 Banzai Cloud
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

package adapter

import (
	"errors"
	"strings"

	"github.com/banzaicloud/pipeline/internal/cluster"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
)

const (
	GORMAzurePKEClustersTableName  = "azure_pke_clusters"
	GORMAzurePKENodePoolsTableName = "azure_pke_node_pools"
	GORMLabelSeparator             = ","
	GORMLabelKVSeparator           = ":"
	GORMRoleSeparator              = ","
	GORMZoneSeparator              = ","
)

type gormAzurePKEClusterStore struct {
	db *gorm.DB
}

func NewGORMAzurePKEClusterStore(db *gorm.DB) pke.AzurePKEClusterStore {
	return gormAzurePKEClusterStore{
		db: db,
	}
}

type gormAzurePKENodePoolModel struct {
	gorm.Model

	Autoscaling  bool
	ClusterID    uint
	CreatedBy    uint
	DesiredCount uint
	InstanceType string
	Labels       string
	Max          uint
	Min          uint
	Name         string
	Roles        string
	SubnetName   string
	Zones        string
}

func (gormAzurePKENodePoolModel) TableName() string {
	return GORMAzurePKENodePoolsTableName
}

type gormAzurePKEClusterModel struct {
	ID                     uint `gorm:"primary_key"`
	ClusterID              uint
	ResourceGroupName      string
	VirtualNetworkLocation string
	VirtualNetworkName     string

	Cluster   cluster.ClusterModel        `gorm:"foreignkey:ClusterID"`
	NodePools []gormAzurePKENodePoolModel `gorm:"foreignkey:ClusterID"`
}

func (gormAzurePKEClusterModel) TableName() string {
	return GORMAzurePKEClustersTableName
}

func fillClusterFromClusterModel(cl *pke.PKEOnAzureCluster, model cluster.ClusterModel) {
	cl.CreatedBy = model.CreatedBy
	cl.CreationTime = model.CreatedAt
	cl.ID = model.ID
	cl.K8sSecretID = model.ConfigSecretID
	cl.Name = model.Name
	cl.OrganizationID = model.OrganizationID
	cl.SecretID = model.SecretID
	cl.SSHSecretID = model.SSHSecretID
	cl.Status = model.Status
	cl.StatusMessage = model.StatusMessage
	cl.UID = model.UID

	cl.ScaleOptions.DesiredCpu = model.ScaleOptions.DesiredCpu
	cl.ScaleOptions.DesiredGpu = model.ScaleOptions.DesiredGpu
	cl.ScaleOptions.DesiredMem = model.ScaleOptions.DesiredMem
	cl.ScaleOptions.Enabled = model.ScaleOptions.Enabled
	cl.ScaleOptions.Excludes = deserializeExcludes(model.ScaleOptions.Excludes)
	cl.ScaleOptions.KeepDesiredCapacity = model.ScaleOptions.KeepDesiredCapacity
	cl.ScaleOptions.OnDemandPct = model.ScaleOptions.OnDemandPct
}

func deserializeExcludes(excludes string) []string {
	if excludes == "" {
		return nil
	}
	return strings.Split(excludes, cluster.InstanceTypeSeparator)
}

func serializeLabels(labels map[string]string) string {
	var b strings.Builder
	for k, v := range labels {
		if b.Len() != 0 {
			b.WriteString(GORMLabelSeparator)
		}
		b.WriteString(k)
		b.WriteString(GORMLabelKVSeparator)
		b.WriteString(v)
	}
	return b.String()
}

func deserializeLabels(labels string) map[string]string {
	res := make(map[string]string)
	for _, l := range strings.Split(labels, GORMLabelSeparator) {
		kv := strings.Split(l, GORMLabelKVSeparator)
		res[kv[0]] = kv[1]
	}
	return res
}

func fillClusterFromAzurePKEClusterModel(cluster *pke.PKEOnAzureCluster, model gormAzurePKEClusterModel) {
	fillClusterFromClusterModel(cluster, model.Cluster)

	cluster.ResourceGroup.Name = model.ResourceGroupName
	cluster.Location = model.VirtualNetworkLocation

	cluster.NodePools = make([]pke.NodePool, len(model.NodePools))
	for i, np := range model.NodePools {
		cluster.NodePools[i].Autoscaling = np.Autoscaling
		cluster.NodePools[i].CreatedBy = np.CreatedBy
		cluster.NodePools[i].DesiredCount = np.DesiredCount
		cluster.NodePools[i].InstanceType = np.InstanceType
		cluster.NodePools[i].Labels = deserializeLabels(np.Labels)
		cluster.NodePools[i].Max = np.Max
		cluster.NodePools[i].Min = np.Min
		cluster.NodePools[i].Name = np.Name
		cluster.NodePools[i].Roles = strings.Split(np.Roles, GORMRoleSeparator)
		cluster.NodePools[i].Subnet.Name = np.SubnetName
		cluster.NodePools[i].Zones = strings.Split(np.Zones, GORMZoneSeparator)
	}

	cluster.VirtualNetwork.Name = model.VirtualNetworkName
	cluster.VirtualNetwork.Location = model.VirtualNetworkLocation
}

func (s gormAzurePKEClusterStore) Create(params pke.CreateParams) (c pke.PKEOnAzureCluster, err error) {
	nodePools := make([]gormAzurePKENodePoolModel, len(params.NodePools))
	for i, np := range params.NodePools {
		nodePools[i].Autoscaling = np.Autoscaling
		nodePools[i].CreatedBy = np.CreatedBy
		nodePools[i].DesiredCount = np.DesiredCount
		nodePools[i].InstanceType = np.InstanceType
		nodePools[i].Labels = serializeLabels(np.Labels)
		nodePools[i].Max = np.Max
		nodePools[i].Min = np.Min
		nodePools[i].Name = np.Name
		nodePools[i].Roles = strings.Join(np.Roles, GORMRoleSeparator)
		nodePools[i].SubnetName = np.Subnet.Name
		nodePools[i].Zones = strings.Join(np.Zones, GORMZoneSeparator)
	}
	model := gormAzurePKEClusterModel{
		Cluster: cluster.ClusterModel{
			CreatedBy:      params.CreatedBy,
			Name:           params.Name,
			Location:       params.Location,
			Cloud:          providers.Azure,
			Distribution:   pke.PKEOnAzure,
			OrganizationID: params.OrganizationID,
			SecretID:       params.SecretID,
			SSHSecretID:    params.SSHSecretID,
			Status:         pkgCluster.Creating,
			RbacEnabled:    params.RBAC,
			ScaleOptions: model.ScaleOptions{
				Enabled:             params.ScaleOptions.Enabled,
				DesiredCpu:          params.ScaleOptions.DesiredCpu,
				DesiredMem:          params.ScaleOptions.DesiredMem,
				DesiredGpu:          params.ScaleOptions.DesiredGpu,
				OnDemandPct:         params.ScaleOptions.OnDemandPct,
				Excludes:            strings.Join(params.ScaleOptions.Excludes, cluster.InstanceTypeSeparator),
				KeepDesiredCapacity: params.ScaleOptions.KeepDesiredCapacity,
			},
		},
		ResourceGroupName:      params.ResourceGroupName,
		VirtualNetworkLocation: params.Location,
		VirtualNetworkName:     params.VirtualNetworkName,
		NodePools:              nodePools,
	}
	if err = emperror.Wrap(s.db.Preload("Cluster").Create(&model).Error, "failed to create cluster model"); err != nil {
		return
	}
	fillClusterFromAzurePKEClusterModel(&c, model)
	return
}

func (s gormAzurePKEClusterStore) GetByID(clusterID uint) (cluster pke.PKEOnAzureCluster, err error) {
	model := gormAzurePKEClusterModel{
		ClusterID: clusterID,
	}
	if err = emperror.Wrap(s.db.Preload("Cluster").Where(&model).First(&model).Error, "failed to load model from database"); err != nil {
		return
	}
	fillClusterFromAzurePKEClusterModel(&cluster, model)
	return
}

func (s gormAzurePKEClusterStore) SetStatus(clusterID uint, status, message string) error {
	if clusterID == 0 {
		return errors.New("cluster ID cannot be 0")
	}

	model := cluster.ClusterModel{
		ID: clusterID,
	}
	if err := emperror.Wrap(s.db.Where(&model).First(&model).Error, "failed to load cluster model"); err != nil {
		return err
	}

	if status != model.Status || message != model.StatusMessage {
		fields := map[string]interface{}{
			"status":        status,
			"statusMessage": message,
		}

		statusHistory := cluster.StatusHistoryModel{
			ClusterID:   model.ID,
			ClusterName: model.Name,

			FromStatus:        model.Status,
			FromStatusMessage: model.StatusMessage,
			ToStatus:          status,
			ToStatusMessage:   message,
		}
		if err := emperror.Wrap(s.db.Save(&statusHistory).Error, "failed to save status history"); err != nil {
			return err
		}

		return emperror.Wrap(s.db.Model(&model).Updates(fields).Error, "failed to update cluster model")
	}

	return nil
}
