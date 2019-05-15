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

package deployment

import (
	"github.com/goph/emperror"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// CGDeploymentRepository
type CGDeploymentRepository struct {
	db     *gorm.DB
	logger logrus.FieldLogger
}

// FindByName returns a cluster group deployment by name.
func (g *CGDeploymentRepository) FindByName(clusterGroupID uint, deploymentName string) (*ClusterGroupDeploymentModel, error) {
	if len(deploymentName) == 0 {
		return nil, errors.New("deployment name is required")
	}
	var result ClusterGroupDeploymentModel
	err := g.db.Where(ClusterGroupDeploymentModel{
		ClusterGroupID:        clusterGroupID,
		DeploymentReleaseName: deploymentName,
	}).Preload("TargetClusters").First(&result).Error

	if gorm.IsRecordNotFoundError(err) {
		return nil, errors.WithStack(&deploymentNotFoundError{
			clusterGroupID: clusterGroupID,
			deploymentName: deploymentName,
		})
	}
	if err != nil {
		return nil, emperror.With(err,
			"clusterGroupID", clusterGroupID,
			"deploymentName", deploymentName,
		)
	}

	return &result, nil
}

// FindAll returns all cluster group deployments
func (g *CGDeploymentRepository) FindAll(clusterGroupID uint) ([]*ClusterGroupDeploymentModel, error) {
	var deployments []*ClusterGroupDeploymentModel

	err := g.db.Preload("TargetClusters").Where(&ClusterGroupDeploymentModel{
		ClusterGroupID: clusterGroupID,
	}).Find(&deployments).Error

	if err != nil && !gorm.IsRecordNotFoundError(err) {
		return nil, emperror.With(errors.Wrap(err, "could not fetch cluster group deployments"),
			"clusterGroupID", clusterGroupID,
		)
	}

	return deployments, nil
}

func (g *CGDeploymentRepository) Save(model *ClusterGroupDeploymentModel) error {
	return g.db.Save(model).Error
}

// Delete deletes a target cluster from deployment
func (g *CGDeploymentRepository) DeleteTargetCluster(model *TargetCluster) error {
	err := g.db.Delete(model).Error
	if err != nil {
		return err
	}
	return nil
}

// Delete deletes a cluster group deployment if every target cluster is deleted successfully
func (g *CGDeploymentRepository) Delete(model *ClusterGroupDeploymentModel, targetClustersStatus []TargetClusterStatus) error {
	targetClusterStatusMap := make(map[uint]TargetClusterStatus, 0)
	for _, cs := range targetClustersStatus {
		targetClusterStatusMap[cs.ClusterId] = cs
	}
	deletedCount := 0

	for _, v := range model.TargetClusters {
		if status, ok := targetClusterStatusMap[v.ClusterID]; ok && status.Status == DeletedStatus {
			err := g.db.Delete(v).Error
			if err != nil {
				return err
			}
			deletedCount++
		}
	}

	if deletedCount == len(model.TargetClusters) {
		err := g.db.Delete(model).Error
		if err != nil {
			return err
		}
	}

	return nil
}
