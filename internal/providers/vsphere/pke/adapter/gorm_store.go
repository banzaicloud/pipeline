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
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter/clustermodel"
	sqlJson "github.com/banzaicloud/pipeline/internal/database/sql/json"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"
	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

const (
	ClustersTableName  = "vsphere_pke_clusters"
	NodePoolsTableName = "vsphere_pke_node_pools"
)

type gormVspherePKEClusterStore struct {
	db *gorm.DB
}

func NewClusterStore(db *gorm.DB) pke.ClusterStore {
	return gormVspherePKEClusterStore{
		db: db,
	}
}

type rolesModel []string

func (m *rolesModel) Scan(v interface{}) error {
	return sqlJson.Scan(v, m)
}

func (m rolesModel) Value() (driver.Value, error) {
	return sqlJson.Value(m)
}

type nodePoolModel struct {
	ID            uint `gorm:"primary_key"`
	Autoscaling   bool
	ClusterID     uint `gorm:"unique_index:idx_vsphere_pke_np_cluster_id_name"`
	CreatedBy     uint
	Size          int
	MaxSize       uint
	MinSize       uint
	VCPU          int        `gorm:"column:vcpu"`
	RAM           int        // MiB
	Name          string     `gorm:"unique_index:idx_vsphere_pke_np_cluster_id_name"`
	Roles         rolesModel `gorm:"type:json"`
	AdminUsername string
	TemplateName  string
}

func (nodePoolModel) TableName() string {
	return NodePoolsTableName
}

type vspherePkeCluster struct {
	ID                  uint                      `gorm:"primary_key"`
	ClusterID           uint                      `gorm:"unique_index:idx_vsphere_pke_cluster_id"`
	Cluster             clustermodel.ClusterModel `gorm:"foreignkey:ClusterID"`
	Spec                ProviderSpec              `gorm:"type:json"`
	NodePools           []nodePoolModel           `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID"`
	StorageSecretID     string
	LoadBalancerIPRange string
}

func (vspherePkeCluster) TableName() string {
	return ClustersTableName
}

type ProviderSpec struct {
	Kubernetes       intPKE.Kubernetes
	ActiveWorkflowID string
	HTTPProxy        intPKE.HTTPProxy
	ResourcePoolName string
	FolderName       string
	DatastoreName    string
}

func (m *ProviderSpec) Scan(v interface{}) error {
	if s, ok := v.(string); ok {
		v = []byte(s)
	}
	return json.Unmarshal(v.([]byte), m)
}

func (m ProviderSpec) Value() (driver.Value, error) {
	return json.Marshal(m)
}

type recordNotFoundError struct{}

func (recordNotFoundError) Error() string {
	return "record was not found"
}

func (recordNotFoundError) NotFound() bool {
	return true
}

func fillClusterFromClusterModel(cl *pke.PKEOnVsphereCluster, model clustermodel.ClusterModel) {
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

	cl.Kubernetes.RBAC = model.RbacEnabled
	cl.Kubernetes.OIDC.Enabled = model.OidcEnabled
}

func fillClusterFromModel(cluster *pke.PKEOnVsphereCluster, model vspherePkeCluster) error {
	fillClusterFromClusterModel(cluster, model.Cluster)

	cluster.NodePools = make([]pke.NodePool, len(model.NodePools))
	for i, np := range model.NodePools {
		fillNodePoolFromModel(&cluster.NodePools[i], np)
	}

	cluster.Kubernetes = model.Spec.Kubernetes
	cluster.ActiveWorkflowID = model.Spec.ActiveWorkflowID
	cluster.Datastore = model.Spec.DatastoreName
	cluster.Folder = model.Spec.FolderName
	cluster.ResourcePool = model.Spec.ResourcePoolName
	cluster.HTTPProxy = model.Spec.HTTPProxy
	cluster.StorageSecretID = model.StorageSecretID
	cluster.LoadBalancerIPRange = model.LoadBalancerIPRange

	return nil
}

func fillNodePoolFromModel(nodePool *pke.NodePool, model nodePoolModel) {
	nodePool.CreatedBy = model.CreatedBy
	nodePool.Size = model.Size
	nodePool.VCPU = model.VCPU
	nodePool.RAM = model.RAM
	nodePool.Name = model.Name
	nodePool.Roles = model.Roles
	nodePool.TemplateName = model.TemplateName
	nodePool.AdminUsername = model.AdminUsername
}

func fillModelFromNodePool(model *nodePoolModel, nodePool pke.NodePool) {
	model.CreatedBy = nodePool.CreatedBy
	model.Size = nodePool.Size
	model.VCPU = nodePool.VCPU
	model.RAM = nodePool.RAM
	model.Name = nodePool.Name
	model.Roles = nodePool.Roles
	model.TemplateName = nodePool.TemplateName
	model.AdminUsername = nodePool.AdminUsername
}

func (s gormVspherePKEClusterStore) CreateNodePool(clusterID uint, nodePool pke.NodePool) error {
	var np nodePoolModel
	np.ClusterID = clusterID
	fillModelFromNodePool(&np, nodePool)
	return getError(s.db.Create(&np), "failed to create node pool model")
}

func (s gormVspherePKEClusterStore) Create(params pke.CreateParams) (c pke.PKEOnVsphereCluster, err error) {
	nodePools := make([]nodePoolModel, len(params.NodePools))
	for i, np := range params.NodePools {
		fillModelFromNodePool(&nodePools[i], np)
	}

	model := vspherePkeCluster{
		Cluster: clustermodel.ClusterModel{
			CreatedBy:      params.CreatedBy,
			Name:           params.Name,
			Cloud:          pkgCluster.Vsphere,
			Distribution:   pkgCluster.PKE,
			OrganizationID: params.OrganizationID,
			SecretID:       params.SecretID,
			SSHSecretID:    params.SSHSecretID,
			Status:         pkgCluster.Creating,
			StatusMessage:  pkgCluster.CreatingMessage,
			RbacEnabled:    params.RBAC,
			OidcEnabled:    params.OIDC,
		},
		Spec: ProviderSpec{
			ResourcePoolName: params.ResourcePoolName,
			FolderName:       params.FolderName,
			DatastoreName:    params.DatastoreName,
			Kubernetes:       params.Kubernetes,
			HTTPProxy:        params.HTTPProxy,
		},
		NodePools:           nodePools,
		StorageSecretID:     params.StorageSecretID,
		LoadBalancerIPRange: params.LoadBalancerIPRange,
	}

	if err = getError(s.db.Preload("Cluster").Create(&model), "failed to create cluster model"); err != nil {
		return
	}
	if err := fillClusterFromModel(&c, model); err != nil {
		return c, errors.WrapIf(err, "failed to fill cluster from model")
	}
	return
}

func (s gormVspherePKEClusterStore) DeleteNodePool(clusterID uint, nodePoolName string) error {
	if err := validateClusterID(clusterID); err != nil {
		return errors.WrapIf(err, "invalid cluster ID")
	}
	if nodePoolName == "" {
		return errors.New("empty node pool name")
	}

	model := nodePoolModel{
		ClusterID: clusterID,
		Name:      nodePoolName,
	}
	if err := getError(s.db.Where(model).First(&model), "failed to load model from database"); err != nil {
		return err
	}

	return getError(s.db.Delete(model), "failed to delete model from database")
}

func (s gormVspherePKEClusterStore) UpdateNodePoolSize(clusterID uint, nodePoolName string, size int) error {
	if err := validateClusterID(clusterID); err != nil {
		return errors.WrapIf(err, "invalid cluster ID")
	}
	if nodePoolName == "" {
		return errors.New("empty node pool name")
	}

	model := nodePoolModel{
		ClusterID: clusterID,
		Name:      nodePoolName,
	}
	if err := getError(s.db.Where(model).First(&model), "failed to load model from database"); err != nil {
		return err
	}
	fields := map[string]interface{}{
		"size": size,
	}

	return getError(s.db.Model(&model).Updates(fields), "failed to update model in database")
}

func (s gormVspherePKEClusterStore) Delete(clusterID uint) error {
	if err := validateClusterID(clusterID); err != nil {
		return errors.WrapIf(err, "invalid cluster ID")
	}

	model := clustermodel.ClusterModel{
		ID: clusterID,
	}
	if err := getError(s.db.Where(model).First(&model), "failed to load model from database"); err != nil {
		return err
	}

	return getError(s.db.Delete(model), "failed to soft-delete model from database")
}

func (s gormVspherePKEClusterStore) GetByID(clusterID uint) (cluster pke.PKEOnVsphereCluster, _ error) {
	if err := validateClusterID(clusterID); err != nil {
		return cluster, errors.WrapIf(err, "invalid cluster ID")
	}

	model := vspherePkeCluster{
		ClusterID: clusterID,
	}
	if err := getError(s.db.Preload("Cluster").Preload("NodePools").Where(&model).First(&model), "failed to load model from database"); err != nil {
		return cluster, err
	}
	if err := fillClusterFromModel(&cluster, model); err != nil {
		return cluster, errors.WrapIf(err, "failed to fill cluster from model")
	}
	return
}

func (s gormVspherePKEClusterStore) SetStatus(clusterID uint, status, message string) error {
	if err := validateClusterID(clusterID); err != nil {
		return errors.WrapIf(err, "invalid cluster ID")
	}

	model := clustermodel.ClusterModel{
		ID: clusterID,
	}
	if err := getError(s.db.Where(&model).First(&model), "failed to load cluster model"); err != nil {
		return err
	}

	if status != model.Status || message != model.StatusMessage {
		fields := map[string]interface{}{
			"status":        status,
			"statusMessage": message,
		}

		statusHistory := clustermodel.StatusHistoryModel{
			ClusterID:   model.ID,
			ClusterName: model.Name,

			FromStatus:        model.Status,
			FromStatusMessage: model.StatusMessage,
			ToStatus:          status,
			ToStatusMessage:   message,
		}
		if err := getError(s.db.Save(&statusHistory), "failed to save status history"); err != nil {
			return err
		}

		return getError(s.db.Model(&model).Updates(fields), "failed to update cluster model")
	}

	return nil
}

func (s gormVspherePKEClusterStore) getProviderData(clusterID uint) (ProviderSpec, error) {
	if err := validateClusterID(clusterID); err != nil {
		return ProviderSpec{}, errors.WrapIf(err, "invalid cluster ID")
	}

	model := vspherePkeCluster{
		ClusterID: clusterID,
	}
	if err := getError(s.db.Where(&model).First(&model), "failed to load cluster model"); err != nil {
		return ProviderSpec{}, err
	}

	return model.Spec, nil
}

func (s gormVspherePKEClusterStore) updateProviderData(clusterID uint, data ProviderSpec) error {
	if err := validateClusterID(clusterID); err != nil {
		return errors.WrapIf(err, "invalid cluster ID")
	}

	model := vspherePkeCluster{
		ClusterID: clusterID,
	}

	return getError(s.db.Model(&model).Where("cluster_id = ?", clusterID).Update("Spec", data), "failed to update PKE-on-Vsphere cluster model")
}

func (s gormVspherePKEClusterStore) SetActiveWorkflowID(clusterID uint, workflowID string) error {
	data, err := s.getProviderData(clusterID)
	if err != nil {
		return err
	}

	data.ActiveWorkflowID = workflowID

	return s.updateProviderData(clusterID, data)
}

func (s gormVspherePKEClusterStore) SetConfigSecretID(clusterID uint, secretID string) error {
	if err := validateClusterID(clusterID); err != nil {
		return errors.WrapIf(err, "invalid cluster ID")
	}

	model := clustermodel.ClusterModel{
		ID: clusterID,
	}

	fields := map[string]interface{}{
		"ConfigSecretID": secretID,
	}

	return getError(s.db.Model(&model).Updates(fields), "failed to update cluster model")
}

func (s gormVspherePKEClusterStore) SetSSHSecretID(clusterID uint, secretID string) error {
	if err := validateClusterID(clusterID); err != nil {
		return errors.WrapIf(err, "invalid cluster ID")
	}

	model := clustermodel.ClusterModel{
		ID: clusterID,
	}

	fields := map[string]interface{}{
		"SSHSecretID": secretID,
	}

	return getError(s.db.Model(&model).Updates(fields), "failed to update cluster model")
}

func (s gormVspherePKEClusterStore) GetConfigSecretID(clusterID uint) (string, error) {
	if err := validateClusterID(clusterID); err != nil {
		return "", errors.WrapIf(err, "invalid cluster ID")
	}

	model := clustermodel.ClusterModel{
		ID: clusterID,
	}
	if err := getError(s.db.Where(&model).First(&model), "failed to load cluster model"); err != nil {
		return "", err
	}
	return model.ConfigSecretID, nil
}

// Migrate executes the table migrations for the provider.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	tables := []interface{}{
		&vspherePkeCluster{},
		&nodePoolModel{},
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.WithFields(logrus.Fields{
		"provider":    pke.PKEOnVsphere,
		"table_names": strings.TrimSpace(tableNames),
	}).Info("migrating provider tables")

	return db.AutoMigrate(tables...).Error
}

func validateClusterID(clusterID uint) error {
	if clusterID == 0 {
		return errors.New("cluster ID cannot be 0")
	}
	return nil
}

func getError(db *gorm.DB, message string, args ...interface{}) error {
	err := db.Error
	if gorm.IsRecordNotFoundError(err) {
		err = recordNotFoundError{}
	}
	if len(args) == 0 {
		err = errors.WrapIf(err, message)
	} else {
		err = errors.WrapIff(err, message, args...)
	}
	return err
}
