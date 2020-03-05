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
	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter/clustermodel"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"
	"github.com/banzaicloud/pipeline/internal/providers/vsphere/pke"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
)

type gormVspherePKEClusterStore struct {
	db *gorm.DB
}

func NewClusterStore(db *gorm.DB) pke.ClusterStore {
	return gormVspherePKEClusterStore{
		db: db,
	}
}

type nodePoolModel struct {
	CreatedBy uint
	Count     int
	VCPU      int
	RamMB     int
	Name      string
	Roles     []string
}

type vspherePkeCluster struct {
	gorm.Model

	ClusterID uint                        `gorm:"unique_index:idx_vsphere_pke_cluster_id"`
	Cluster   clustermodel.ClusterModel `gorm:"foreignkey:ClusterID"`

	ProviderData ProviderData `gorm:"type:json"`
}

type ProviderData struct {
	NodePools        []nodePoolModel
	Kubernetes       intPKE.Kubernetes
	ActiveWorkflowID string
	HTTPProxy        intPKE.HTTPProxy

	Monitoring   bool
	Logging      bool
	SecurityScan bool

	ResourcePoolName string
	FolderName       string
	DatastoreName    string
}

func (m *ProviderData) Scan(v interface{}) error {
	if s, ok := v.(string); ok {
		v = []byte(s)
	}
	return json.Unmarshal(v.([]byte), m)
}

func (m ProviderData) Value() (driver.Value, error) {
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

	cl.ScaleOptions.DesiredCpu = model.ScaleOptions.DesiredCpu
	cl.ScaleOptions.DesiredGpu = model.ScaleOptions.DesiredGpu
	cl.ScaleOptions.DesiredMem = model.ScaleOptions.DesiredMem
	cl.ScaleOptions.Enabled = model.ScaleOptions.Enabled
	cl.ScaleOptions.Excludes = unmarshalStringSlice(model.ScaleOptions.Excludes)
	cl.ScaleOptions.KeepDesiredCapacity = model.ScaleOptions.KeepDesiredCapacity
	cl.ScaleOptions.OnDemandPct = model.ScaleOptions.OnDemandPct

	cl.Kubernetes.RBAC = model.RbacEnabled
	cl.Kubernetes.OIDC.Enabled = model.OidcEnabled
}

func fillClusterFromModel(cluster *pke.PKEOnVsphereCluster, model vspherePkeCluster) error {
	fillClusterFromClusterModel(cluster, model.Cluster)

	cluster.NodePools = make([]pke.NodePool, len(model.ProviderData.NodePools))
	for i, np := range model.ProviderData.NodePools {
		fillNodePoolFromModel(&cluster.NodePools[i], np)
	}

	cluster.Kubernetes = model.ProviderData.Kubernetes
	cluster.ActiveWorkflowID = model.ProviderData.ActiveWorkflowID
	cluster.Datastore = model.ProviderData.DatastoreName
	cluster.Folder = model.ProviderData.FolderName
	cluster.ResourcePool = model.ProviderData.ResourcePoolName

	//cluster.HTTPProxy = model.HTTPProxy.toEntity()

	return nil
}

func fillNodePoolFromModel(nodePool *pke.NodePool, model nodePoolModel) {
	nodePool.CreatedBy = model.CreatedBy
	nodePool.Count = model.Count
	nodePool.VCPU = model.VCPU
	nodePool.RamMB = model.RamMB
	nodePool.Name = model.Name
	nodePool.Roles = model.Roles
}

func fillModelFromNodePool(model *nodePoolModel, nodePool pke.NodePool) {
	model.CreatedBy = nodePool.CreatedBy
	model.Count = nodePool.Count
	model.VCPU = nodePool.VCPU
	model.RamMB = nodePool.RamMB
	model.Name = nodePool.Name
	model.Roles = nodePool.Roles
}

func (s gormVspherePKEClusterStore) CreateNodePool(clusterID uint, nodePool pke.NodePool) error {
	data, err := s.getProviderData(clusterID)
	if err != nil {
		return err
	}

	var np nodePoolModel
	fillModelFromNodePool(&np, nodePool)

	data.NodePools = append(data.NodePools, np)
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
			ScaleOptions: clustermodel.ScaleOptions{
				Enabled:             params.ScaleOptions.Enabled,
				DesiredCpu:          params.ScaleOptions.DesiredCpu,
				DesiredMem:          params.ScaleOptions.DesiredMem,
				DesiredGpu:          params.ScaleOptions.DesiredGpu,
				OnDemandPct:         params.ScaleOptions.OnDemandPct,
				Excludes:            marshalStringSlice(params.ScaleOptions.Excludes),
				KeepDesiredCapacity: params.ScaleOptions.KeepDesiredCapacity,
			},
		},
		ProviderData: ProviderData{

			NodePools:        nodePools,
			ResourcePoolName: params.ResourcePoolName,
			FolderName:       params.FolderName,
			DatastoreName:    params.DatastoreName,

			Kubernetes: params.Kubernetes,
			//HTTPProxy        intPKE.HTTPProxy
			//Monitoring   bool
			//Logging      bool
			//SecurityScan bool
		},
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
	data, err := s.getProviderData(clusterID)
	if err != nil {
		return err
	}

	if nodePoolName == "" {
		return errors.New("empty node pool name")
	}

	newPools := []nodePoolModel{}

	for _, np := range data.NodePools {
		if np.Name != nodePoolName {
			newPools = append(newPools, np)
		}
	}
	if len(newPools) == len(data.NodePools) {
		return errors.New("can't find node pool")
	}

	data.NodePools = newPools

	return s.updateProviderData(clusterID, data)
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
	if err := getError(s.db.Preload("Cluster").Where(&model).First(&model), "failed to load model from database"); err != nil {
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

func (s gormVspherePKEClusterStore) getProviderData(clusterID uint) (ProviderData, error) {
	if err := validateClusterID(clusterID); err != nil {
		return ProviderData{}, errors.WrapIf(err, "invalid cluster ID")
	}

	model := vspherePkeCluster{
		ClusterID: clusterID,
	}
	if err := getError(s.db.Where(&model).First(&model), "failed to load cluster model"); err != nil {
		return ProviderData{}, err
	}

	return model.ProviderData, nil
}

func (s gormVspherePKEClusterStore) updateProviderData(clusterID uint, data ProviderData) error {
	if err := validateClusterID(clusterID); err != nil {
		return errors.WrapIf(err, "invalid cluster ID")
	}

	model := vspherePkeCluster{
		ClusterID: clusterID,
	}

	return getError(s.db.Model(&model).Where("cluster_id = ?", clusterID).Update("ProviderData", data), "failed to update PKE-on-Vsphere cluster model")
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

func (s gormVspherePKEClusterStore) SetFeature(clusterID uint, feature string, state bool) error {
	if err := validateClusterID(clusterID); err != nil {
		return errors.WrapIf(err, "invalid cluster ID")
	}

	model := clustermodel.ClusterModel{
		ID: clusterID,
	}

	features := map[string]bool{
		"SecurityScan": true,
		"Logging":      true,
		"Monitoring":   true,
	}

	if !features[feature] {
		return fmt.Errorf("unknown feature: %q", feature)
	}

	fields := map[string]interface{}{
		feature: state,
	}

	return getError(s.db.Model(&model).Updates(fields), "failed to update %q feature state", feature)
}

func (s gormVspherePKEClusterStore) SetNodePoolSizes(clusterID uint, nodePoolName string, min, max, desiredCount uint, autoscaling bool) error {
	// TODO
	return nil
}

// Migrate executes the table migrations for the provider.
func Migrate(db *gorm.DB, logger logrus.FieldLogger) error {
	tables := []interface{}{
		&vspherePkeCluster{},
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

func marshalStringSlice(s []string) string {
	data, err := json.Marshal(s)
	emperror.Panic(errors.WrapIf(err, "failed to marshal string slice"))
	return string(data)
}

func unmarshalStringSlice(s string) (result []string) {
	if s == "" {
		// empty list in legacy format
		return nil
	}
	err := errors.WrapIf(json.Unmarshal([]byte(s), &result), "failed to unmarshal string slice")
	if err != nil {
		// try to parse legacy format
		result = strings.Split(s, ",")
	}
	return
}
