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
	"fmt"
	"strings"

	"emperror.dev/errors"
	"github.com/jinzhu/gorm"

	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/database/sql/json"
	intPKE "github.com/banzaicloud/pipeline/internal/pke"
	"github.com/banzaicloud/pipeline/internal/providers/azure/pke"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/src/model"
)

const (
	ClustersTableName  = "azure_pke_clusters"
	NodePoolsTableName = "azure_pke_node_pools"
)

type ClusterStore struct {
	db  *gorm.DB
	log common.Logger
}

func NewClusterStore(db *gorm.DB, logger common.Logger) ClusterStore {
	return ClusterStore{
		db:  db,
		log: logger,
	}
}

type clusterModel struct {
	ID                     uint `gorm:"primary_key"`
	ClusterID              uint `gorm:"unique_index:idx_azure_pke_cluster_id"`
	ResourceGroupName      string
	VirtualNetworkLocation string
	VirtualNetworkName     string

	ActiveWorkflowID  string
	KubernetesVersion string

	HTTPProxy httpProxyModel `gorm:"type:json"`

	Cluster   clusteradapter.ClusterModel `gorm:"foreignkey:ClusterID"`
	NodePools []nodePoolModel             `gorm:"foreignkey:ClusterID;association_foreignkey:ClusterID"`

	AccessPoints          accessPointsModel          `gorm:"type:json"`
	ApiServerAccessPoints apiServerAccessPointsModel `gorm:"type:json"`
}

func (clusterModel) TableName() string {
	return ClustersTableName
}

type nodePoolModel struct {
	gorm.Model

	Autoscaling  bool
	ClusterID    uint `gorm:"unique_index:idx_azure_pke_np_cluster_id_name"`
	CreatedBy    uint
	DesiredCount uint
	InstanceType string
	Max          uint
	Min          uint
	Name         string     `gorm:"unique_index:idx_azure_pke_np_cluster_id_name"`
	Roles        rolesModel `gorm:"type:json"`
	SubnetName   string
	Zones        zonesModel `gorm:"type:json"`
}

func (nodePoolModel) TableName() string {
	return NodePoolsTableName
}

type rolesModel []string

func (m *rolesModel) Scan(v interface{}) error {
	return json.Scan(v, m)
}

func (m rolesModel) Value() (driver.Value, error) {
	return json.Value(m)
}

type zonesModel []string

func (m *zonesModel) Scan(v interface{}) error {
	return json.Scan(v, m)
}

func (m zonesModel) Value() (driver.Value, error) {
	return json.Value(m)
}

type httpProxyModel struct {
	HTTP       httpProxyOptionsModel `json:"http,omitempty"`
	HTTPS      httpProxyOptionsModel `json:"https,omitempty"`
	Exceptions []string              `json:"exceptions,omitempty"`
}

func (m *httpProxyModel) Scan(v interface{}) error {
	return json.Scan(v, m)
}

func (m httpProxyModel) Value() (driver.Value, error) {
	return json.Value(m)
}

func (m *httpProxyModel) fromEntity(e intPKE.HTTPProxy) {
	m.HTTP.fromEntity(e.HTTP)
	m.HTTPS.fromEntity(e.HTTPS)
	m.Exceptions = e.Exceptions
}

func (m httpProxyModel) toEntity() intPKE.HTTPProxy {
	return intPKE.HTTPProxy{
		HTTP:       m.HTTP.toEntity(),
		HTTPS:      m.HTTPS.toEntity(),
		Exceptions: m.Exceptions,
	}
}

type httpProxyOptionsModel struct {
	Host     string `json:"host"`
	Port     uint16 `json:"port,omitempty"`
	SecretID string `json:"secretId,omitempty"`
}

func (m *httpProxyOptionsModel) fromEntity(e intPKE.HTTPProxyOptions) {
	m.Host = e.Host
	m.Port = e.Port
	m.SecretID = e.SecretID
}

func (m httpProxyOptionsModel) toEntity() intPKE.HTTPProxyOptions {
	return intPKE.HTTPProxyOptions{
		Host:     m.Host,
		Port:     m.Port,
		SecretID: m.SecretID,
	}
}

type accessPointModel struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func (m *accessPointModel) fromEntity(e pke.AccessPoint) {
	m.Name = e.Name
	m.Address = e.Address
}

func (m accessPointModel) toEntity() pke.AccessPoint {
	return pke.AccessPoint{
		Name:    m.Name,
		Address: m.Address,
	}
}

type accessPointsModel []accessPointModel

func (m *accessPointsModel) Scan(v interface{}) error {
	return json.Scan(v, m)
}

func (m accessPointsModel) Value() (driver.Value, error) {
	return json.Value(m)
}

func (m *accessPointsModel) fromEntity(e pke.AccessPoints) {
	*m = make(accessPointsModel, len(e))
	for i, ap := range e {
		(*m)[i].fromEntity(ap)
	}
}

func (m accessPointsModel) toEntity() pke.AccessPoints {
	aps := make(pke.AccessPoints, len(m))
	for i, apm := range m {
		aps[i] = apm.toEntity()
	}
	return aps
}

type apiServerAccessPointModel string

func (m *apiServerAccessPointModel) fromEntity(e pke.APIServerAccessPoint) {
	*m = apiServerAccessPointModel(e)
}
func (m apiServerAccessPointModel) toEntity() pke.APIServerAccessPoint {
	return pke.APIServerAccessPoint(m)
}

type apiServerAccessPointsModel []apiServerAccessPointModel

func (m *apiServerAccessPointsModel) Scan(v interface{}) error {
	return json.Scan(v, m)
}

func (m apiServerAccessPointsModel) Value() (driver.Value, error) {
	return json.Value(m)
}

func (m *apiServerAccessPointsModel) fromEntity(e pke.APIServerAccessPoints) {
	*m = make(apiServerAccessPointsModel, len(e))
	for i, asap := range e {
		(*m)[i].fromEntity(asap)
	}
}

func (m apiServerAccessPointsModel) toEntity() pke.APIServerAccessPoints {
	asaps := make(pke.APIServerAccessPoints, len(m))
	for i, asapm := range m {
		asaps[i] = asapm.toEntity()
	}
	return asaps
}

func (m clusterModel) intoEntity(e *pke.Cluster) {
	// common fields

	e.CreatedBy = m.Cluster.CreatedBy
	e.CreationTime = m.Cluster.CreatedAt
	e.ID = m.Cluster.ID
	e.K8sSecretID = m.Cluster.ConfigSecretID
	e.Name = m.Cluster.Name
	e.OrganizationID = m.Cluster.OrganizationID
	e.SecretID = m.Cluster.SecretID
	e.SSHSecretID = m.Cluster.SSHSecretID
	e.Status = m.Cluster.Status
	e.StatusMessage = m.Cluster.StatusMessage
	e.UID = m.Cluster.UID

	e.ScaleOptions.DesiredCpu = m.Cluster.ScaleOptions.DesiredCpu
	e.ScaleOptions.DesiredGpu = m.Cluster.ScaleOptions.DesiredGpu
	e.ScaleOptions.DesiredMem = m.Cluster.ScaleOptions.DesiredMem
	e.ScaleOptions.Enabled = m.Cluster.ScaleOptions.Enabled
	_ = json.Scan(m.Cluster.ScaleOptions.Excludes, &e.ScaleOptions.Excludes)
	e.ScaleOptions.KeepDesiredCapacity = m.Cluster.ScaleOptions.KeepDesiredCapacity
	e.ScaleOptions.OnDemandPct = m.Cluster.ScaleOptions.OnDemandPct

	e.Kubernetes.RBAC = m.Cluster.RbacEnabled
	e.Kubernetes.OIDC.Enabled = m.Cluster.OidcEnabled
	e.TtlMinutes = m.Cluster.TtlMinutes

	// provider specific fields

	e.ResourceGroup.Name = m.ResourceGroupName
	e.Location = m.VirtualNetworkLocation

	e.NodePools = make([]pke.NodePool, len(m.NodePools))
	for i, np := range m.NodePools {
		np.intoEntity(&e.NodePools[i])
	}

	e.VirtualNetwork.Name = m.VirtualNetworkName
	e.VirtualNetwork.Location = m.VirtualNetworkLocation

	e.Kubernetes.Version = m.KubernetesVersion
	e.ActiveWorkflowID = m.ActiveWorkflowID

	e.HTTPProxy = m.HTTPProxy.toEntity()
	e.AccessPoints = m.AccessPoints.toEntity()
	e.APIServerAccessPoints = m.ApiServerAccessPoints.toEntity()
}

func (m clusterModel) toEntity() (e pke.Cluster) {
	m.intoEntity(&e)
	return
}

func (m *nodePoolModel) fromEntity(e pke.NodePool) {
	m.Autoscaling = e.Autoscaling
	m.CreatedBy = e.CreatedBy
	m.DesiredCount = e.DesiredCount
	m.InstanceType = e.InstanceType
	m.Max = e.Max
	m.Min = e.Min
	m.Name = e.Name
	m.Roles = rolesModel(e.Roles)
	m.SubnetName = e.Subnet.Name
	m.Zones = zonesModel(e.Zones)
}

func (m nodePoolModel) intoEntity(e *pke.NodePool) {
	e.Autoscaling = m.Autoscaling
	e.CreatedBy = m.CreatedBy
	e.DesiredCount = m.DesiredCount
	e.InstanceType = m.InstanceType
	e.Max = m.Max
	e.Min = m.Min
	e.Name = m.Name
	e.Roles = []string(m.Roles)
	e.Subnet.Name = m.SubnetName
	e.Zones = []string(m.Zones)
}

func (s ClusterStore) nodePools() *gorm.DB {
	return s.db.Table(NodePoolsTableName)
}

func (s ClusterStore) clusterDetails() *gorm.DB {
	return s.db.Table(ClustersTableName)
}

func (s ClusterStore) CreateNodePool(clusterID uint, nodePool pke.NodePool) error {
	var np nodePoolModel
	np.fromEntity(nodePool)
	np.ClusterID = clusterID
	return getError(s.db.Create(&np), "failed to create node pool model")
}

func (s ClusterStore) Create(params pke.CreateParams) (pke.Cluster, error) {
	nodePools := make([]nodePoolModel, len(params.NodePools))
	for i, np := range params.NodePools {
		nodePools[i].fromEntity(np)
	}

	excludesValue, err := json.Value(params.ScaleOptions.Excludes)
	if err != nil {
		return pke.Cluster{}, errors.WrapIf(err, "failed to jsonify scale options excludes")
	}

	var excludes string
	switch e := excludesValue.(type) {
	case string:
		excludes = e
	case []byte:
		excludes = string(e)
	default:
		return pke.Cluster{}, errors.Errorf("cannot convert type %T to string", e)
	}

	model := clusterModel{
		Cluster: clusteradapter.ClusterModel{
			CreatedBy:      params.CreatedBy,
			Name:           params.Name,
			Location:       params.Location,
			Cloud:          pkgCluster.Azure,
			Distribution:   pkgCluster.PKE,
			OrganizationID: params.OrganizationID,
			SecretID:       params.SecretID,
			SSHSecretID:    params.SSHSecretID,
			Status:         pkgCluster.Creating,
			StatusMessage:  pkgCluster.CreatingMessage,
			RbacEnabled:    params.RBAC,
			OidcEnabled:    params.OIDC,
			ScaleOptions: model.ScaleOptions{
				Enabled:             params.ScaleOptions.Enabled,
				DesiredCpu:          params.ScaleOptions.DesiredCpu,
				DesiredMem:          params.ScaleOptions.DesiredMem,
				DesiredGpu:          params.ScaleOptions.DesiredGpu,
				OnDemandPct:         params.ScaleOptions.OnDemandPct,
				Excludes:            excludes,
				KeepDesiredCapacity: params.ScaleOptions.KeepDesiredCapacity,
			},
		},
		KubernetesVersion:      params.KubernetesVersion,
		ResourceGroupName:      params.ResourceGroupName,
		VirtualNetworkLocation: params.Location,
		VirtualNetworkName:     params.VirtualNetworkName,
		NodePools:              nodePools,
	}

	model.HTTPProxy.fromEntity(params.HTTPProxy)
	model.AccessPoints.fromEntity(params.AccessPoints)
	model.ApiServerAccessPoints.fromEntity(params.APIServerAccessPoints)

	if err := getError(s.db.Preload("Cluster").Preload("NodePools").Create(&model), "failed to create cluster model"); err != nil {
		return pke.Cluster{}, err
	}

	return model.toEntity(), nil
}

func (s ClusterStore) DeleteNodePool(clusterID uint, nodePoolName string) error {
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

func (s ClusterStore) Delete(clusterID uint) error {
	if err := validateClusterID(clusterID); err != nil {
		return errors.WrapIf(err, "invalid cluster ID")
	}

	model := clusteradapter.ClusterModel{
		ID: clusterID,
	}
	if err := getError(s.db.Where(model).First(&model), "failed to load model from database"); err != nil {
		return err
	}

	return getError(s.db.Delete(model), "failed to soft-delete model from database")
}

func (s ClusterStore) GetByID(clusterID uint) (pke.Cluster, error) {
	if err := validateClusterID(clusterID); err != nil {
		return pke.Cluster{}, errors.WrapIf(err, "invalid cluster ID")
	}

	model := clusterModel{
		ClusterID: clusterID,
	}

	if err := getError(s.db.Preload("Cluster").Preload("NodePools").Where(&model).First(&model), "failed to load model from database"); err != nil {
		return pke.Cluster{}, err
	}

	return model.toEntity(), nil
}

func (s ClusterStore) SetStatus(clusterID uint, status, message string) error {
	if err := validateClusterID(clusterID); err != nil {
		return errors.WrapIf(err, "invalid cluster ID")
	}

	model := clusteradapter.ClusterModel{
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

		statusHistory := clusteradapter.StatusHistoryModel{
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

func (s ClusterStore) UpdateClusterAccessPoints(clusterID uint, accessPoints pke.AccessPoints) error {
	if err := validateClusterID(clusterID); err != nil {
		return errors.WrapIf(err, "invalid cluster ID")
	}

	model := clusterModel{
		ClusterID: clusterID,
	}
	if err := getError(s.db.Where(&model).First(&model), "failed to load cluster model"); err != nil {
		return err
	}

	s.log.Debug("access points from db", map[string]interface{}{"accesspoints": model.AccessPoints})

	for i := range model.AccessPoints {
		for _, update := range accessPoints {
			if model.AccessPoints[i].Name == update.Name {
				model.AccessPoints[i].fromEntity(update)
			}
		}
	}

	s.log.Debug("updated access points from db", map[string]interface{}{"accesspoints": model.AccessPoints})

	updates := clusterModel{AccessPoints: model.AccessPoints}
	return getError(s.db.Model(&model).Updates(updates), "failed to update PKE-on-Azure cluster access points model")
}

func (s ClusterStore) SetActiveWorkflowID(clusterID uint, workflowID string) error {
	if err := validateClusterID(clusterID); err != nil {
		return errors.WrapIf(err, "invalid cluster ID")
	}

	model := clusterModel{
		ClusterID: clusterID,
	}

	return getError(s.db.Model(&model).Where("cluster_id = ?", clusterID).Update("ActiveWorkflowID", workflowID), "failed to update PKE-on-Azure cluster model")
}

func (s ClusterStore) SetConfigSecretID(clusterID uint, secretID string) error {
	if err := validateClusterID(clusterID); err != nil {
		return errors.WrapIf(err, "invalid cluster ID")
	}

	model := clusteradapter.ClusterModel{
		ID: clusterID,
	}

	fields := map[string]interface{}{
		"ConfigSecretID": secretID,
	}

	return getError(s.db.Model(&model).Updates(fields), "failed to update cluster model")
}

func (s ClusterStore) SetSSHSecretID(clusterID uint, secretID string) error {
	if err := validateClusterID(clusterID); err != nil {
		return errors.WrapIf(err, "invalid cluster ID")
	}

	model := clusteradapter.ClusterModel{
		ID: clusterID,
	}

	fields := map[string]interface{}{
		"SSHSecretID": secretID,
	}

	return getError(s.db.Model(&model).Updates(fields), "failed to update cluster model")
}

func (s ClusterStore) GetConfigSecretID(clusterID uint) (string, error) {
	if err := validateClusterID(clusterID); err != nil {
		return "", errors.WrapIf(err, "invalid cluster ID")
	}

	model := clusteradapter.ClusterModel{
		ID: clusterID,
	}
	if err := getError(s.db.Where(&model).First(&model), "failed to load cluster model"); err != nil {
		return "", err
	}
	return model.ConfigSecretID, nil
}

func (s ClusterStore) SetNodePoolSizes(clusterID uint, nodePoolName string, min, max, desiredCount uint, autoscaling bool) error {
	if err := validateClusterID(clusterID); err != nil {
		return errors.WrapIf(err, "invalid cluster ID")
	}

	model := nodePoolModel{
		ClusterID: clusterID,
		Name:      nodePoolName,
	}

	fields := map[string]interface{}{
		"DesiredCount": desiredCount,
		"Min":          min,
		"Max":          max,
		"Autoscaling":  autoscaling,
	}

	return getError(s.db.Model(&model).Where("cluster_id = ? AND name = ?", clusterID, nodePoolName).Updates(fields), "failed to update nodepool model")
}

// Migrate executes the table migrations for the provider.
func Migrate(db *gorm.DB, logger common.Logger) error {
	tables := []interface{}{
		&nodePoolModel{},
		&clusterModel{},
	}

	var tableNames string
	for _, table := range tables {
		tableNames += fmt.Sprintf(" %s", db.NewScope(table).TableName())
	}

	logger.WithFields(map[string]interface{}{
		"provider":    pke.PKEOnAzure,
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

type recordNotFoundError struct{}

func (recordNotFoundError) Error() string {
	return "record was not found"
}

func (recordNotFoundError) NotFound() bool {
	return true
}
