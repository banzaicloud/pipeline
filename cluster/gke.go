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

package cluster

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	pipConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgClusterGoogle "github.com/banzaicloud/pipeline/pkg/cluster/gke"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	gkeCompute "google.golang.org/api/compute/v1"
	gke "google.golang.org/api/container/v1"
	"google.golang.org/api/googleapi"
	"gopkg.in/yaml.v2"
	"k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	statusRunning = "RUNNING"
	statusDone    = "DONE"
)

const (
	defaultNamespace = "default"
	clusterAdmin     = "cluster-admin"
	netesDefault     = "netes-default"
)

// constants to find Kubernetes resources
const (
	kubernetesIO   = "kubernetes.io"
	targetPrefix   = "gke-"
	clusterNameKey = "cluster-name"
)

//CreateGKEClusterFromRequest creates ClusterModel struct from the request
func CreateGKEClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId, userId uint) (*GKECluster, error) {
	log.Debug("Create ClusterModel struct from the request")
	var cluster GKECluster

	nodePools, err := createNodePoolsModelFromRequestData(request.Properties.CreateClusterGKE.NodePools, userId)
	if err != nil {
		return nil, err
	}

	cluster.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		OrganizationId: orgId,
		SecretId:       request.SecretId,
		CreatedBy:      userId,
		Distribution:   pkgCluster.GKE,
		GKE: model.GKEClusterModel{
			MasterVersion: request.Properties.CreateClusterGKE.Master.Version,
			NodeVersion:   request.Properties.CreateClusterGKE.NodeVersion,
			NodePools:     nodePools,
		},
	}
	return &cluster, nil
}

//createNodePoolsModelFromRequestData creates an array of GoogleNodePoolModel from the nodePoolsData received through create/update requests
func createNodePoolsModelFromRequestData(nodePoolsData map[string]*pkgClusterGoogle.NodePool, userId uint) ([]*model.GKENodePoolModel, error) {

	nodePoolsCount := len(nodePoolsData)
	if nodePoolsCount == 0 {
		return nil, pkgErrors.ErrorNodePoolNotProvided
	}
	nodePoolsModel := make([]*model.GKENodePoolModel, nodePoolsCount)

	i := 0
	for nodePoolName, nodePoolData := range nodePoolsData {
		nodePoolsModel[i] = &model.GKENodePoolModel{
			CreatedBy:        userId,
			Name:             nodePoolName,
			Autoscaling:      nodePoolData.Autoscaling,
			NodeMinCount:     nodePoolData.MinCount,
			NodeMaxCount:     nodePoolData.MaxCount,
			NodeCount:        nodePoolData.Count,
			NodeInstanceType: nodePoolData.NodeInstanceType,
		}
		i++
	}

	return nodePoolsModel, nil
}

//GKECluster struct for GKE cluster
type GKECluster struct {
	googleCluster *gke.Cluster //Don't use this directly
	modelCluster  *model.ClusterModel
	APIEndpoint   string
	CommonClusterBase
}

// GetOrganizationId gets org where the cluster belongs
func (c *GKECluster) GetOrganizationId() uint {
	return c.modelCluster.OrganizationId
}

// GetLocation gets where the cluster is.
func (c *GKECluster) GetLocation() string {
	return c.modelCluster.Location
}

// GetSecretId retrieves the secret id
func (c *GKECluster) GetSecretId() string {
	return c.modelCluster.SecretId
}

// GetSshSecretId retrieves the secret id
func (c *GKECluster) GetSshSecretId() string {
	return c.modelCluster.SshSecretId
}

// SaveSshSecretId saves the ssh secret id to database
func (c *GKECluster) SaveSshSecretId(sshSecretId string) error {
	return c.modelCluster.UpdateSshSecret(sshSecretId)
}

// GetGoogleCluster returns with a Cluster from GKE
func (c *GKECluster) GetGoogleCluster() (*gke.Cluster, error) {
	if c.googleCluster != nil {
		return c.googleCluster, nil
	}
	svc, err := c.getGoogleServiceClient()
	if err != nil {
		return nil, err
	}

	secretItem, err := c.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}

	cc := googleCluster{
		Name:      c.modelCluster.Name,
		ProjectID: secretItem.GetValue(pkgSecret.ProjectId),
		Zone:      c.modelCluster.Location,
	}
	cluster, err := getClusterGoogle(svc, cc)
	if err != nil {
		return nil, err
	}
	c.googleCluster = cluster
	return c.googleCluster, nil
}

//GetAPIEndpoint returns the Kubernetes Api endpoint
func (c *GKECluster) GetAPIEndpoint() (string, error) {
	if c.APIEndpoint != "" {
		return c.APIEndpoint, nil
	}
	cluster, err := c.GetGoogleCluster()
	if err != nil {
		return "", err
	}
	c.APIEndpoint = cluster.Endpoint
	return c.APIEndpoint, nil
}

//CreateCluster creates a new cluster
func (c *GKECluster) CreateCluster() error {

	log.Info("Start create cluster (Google)")

	log.Info("Get Google Service Client")
	svc, err := c.getGoogleServiceClient()
	if err != nil {
		return err
	}

	log.Info("Get Google Service Client succeeded")

	nodePools, err := createNodePoolsFromClusterModel(&c.modelCluster.GKE)
	if err != nil {
		return err
	}

	secretItem, err := c.GetSecretWithValidation()
	if err != nil {
		return err
	}

	projectId := secretItem.GetValue(pkgSecret.ProjectId)

	cc := googleCluster{
		ProjectID:     projectId,
		Zone:          c.modelCluster.Location,
		Name:          c.modelCluster.Name,
		MasterVersion: c.modelCluster.GKE.MasterVersion,
		NodePools:     nodePools,
	}

	ccr := generateClusterCreateRequest(cc)

	log.Infof("Cluster request: %v", ccr)
	createCall, err := svc.Projects.Zones.Clusters.Create(cc.ProjectID, cc.Zone, ccr).Context(context.Background()).Do()

	log.Infof("Cluster request submitted: %v", ccr)

	if err != nil && !strings.Contains(err.Error(), "alreadyExists") {
		be := getBanzaiErrorFromError(err)
		// TODO status code !?
		return errors.New(be.Message)
	}

	if createCall != nil {
		log.Infof("Cluster %s create is called for project %s and zone %s", cc.Name, cc.ProjectID, cc.Zone)
		log.Info("Waiting for cluster...")

		if err := waitForOperation(newContainerOperation(svc, projectId, c.modelCluster.Location), createCall.Name); err != nil {
			return err
		}
	} else {
		log.Info("Cluster %s already exists.", c.modelCluster.Name)
	}

	gkeCluster, err := getClusterGoogle(svc, cc)
	if err != nil {
		be := getBanzaiErrorFromError(err)
		// TODO status code !?
		return errors.New(be.Message)
	}

	c.googleCluster = gkeCluster

	c.updateCurrentVersions(gkeCluster)

	// set region
	c.modelCluster.GKE.Region, err = c.getRegionByZone(projectId, gkeCluster.Zone)
	if err != nil {
		log.Warnf("error during getting region: %s", err.Error())
	}

	return nil

}

func (c *GKECluster) updateCurrentVersions(gkeCluster *gke.Cluster) {

	c.modelCluster.GKE.MasterVersion = gkeCluster.CurrentMasterVersion
	if len(gkeCluster.NodePools) != 0 && gkeCluster.NodePools[0] != nil {
		// currently we didn't support different node versions
		c.modelCluster.GKE.NodeVersion = gkeCluster.NodePools[0].Version
	}

}

//Persist save the cluster model
func (c *GKECluster) Persist(status, statusMessage string) error {
	log.Infof("Model before save: %v", c.modelCluster)
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

// DownloadK8sConfig downloads the kubeconfig file from cloud
func (c *GKECluster) DownloadK8sConfig() ([]byte, error) {

	config, err := c.getGoogleKubernetesConfig()
	if err != nil {
		// something went wrong
		be := getBanzaiErrorFromError(err)
		// TODO status code !?
		return nil, errors.New(be.Message)
	}
	// get config succeeded
	log.Info("Get k8s config succeeded")

	return config, nil

}

//GetName returns the name of the cluster
func (c *GKECluster) GetName() string {
	return c.modelCluster.Name
}

// GetCloud returns the cloud type of the cluster
func (c *GKECluster) GetCloud() string {
	return c.modelCluster.Cloud
}

// GetDistribution returns the distribution type of the cluster
func (c *GKECluster) GetDistribution() string {
	return c.modelCluster.Distribution
}

//GetStatus gets cluster status
func (c *GKECluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {
	log.Info("Create cluster status response")

	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range c.modelCluster.GKE.NodePools {
		if np != nil {
			nodePools[np.Name] = &pkgCluster.NodePoolStatus{
				Autoscaling:  np.Autoscaling,
				Count:        np.NodeCount,
				InstanceType: np.NodeInstanceType,
				MinCount:     np.NodeMinCount,
				MaxCount:     np.NodeMaxCount,
				Version:      c.modelCluster.GKE.NodeVersion,
			}
		}
	}

	return &pkgCluster.GetClusterStatusResponse{
		Status:            c.modelCluster.Status,
		StatusMessage:     c.modelCluster.StatusMessage,
		Name:              c.modelCluster.Name,
		Location:          c.modelCluster.Location,
		Cloud:             c.modelCluster.Cloud,
		Distribution:      c.modelCluster.Distribution,
		ResourceID:        c.modelCluster.ID,
		Version:           c.modelCluster.GKE.MasterVersion,
		NodePools:         nodePools,
		CreatorBaseFields: *NewCreatorBaseFields(c.modelCluster.CreatedAt, c.modelCluster.CreatedBy),
		Region:            c.modelCluster.GKE.Region,
	}, nil
}

// DeleteCluster deletes cluster from google
func (c *GKECluster) DeleteCluster() error {

	if err := c.waitForResourcesDelete(); err != nil {
		return err
	}

	log.Info("Start delete gke cluster")

	if c == nil {
		return pkgErrors.ErrorNilCluster
	}

	secretItem, err := c.GetSecretWithValidation()
	if err != nil {
		return err
	}

	gkec := googleCluster{
		ProjectID: secretItem.GetValue(pkgSecret.ProjectId),
		Name:      c.modelCluster.Name,
		Zone:      c.modelCluster.Location,
	}

	if err := c.callDeleteCluster(&gkec); err != nil {
		be := getBanzaiErrorFromError(err)
		// TODO status code !?
		return errors.New(be.Message)
	}
	log.Info("Delete succeeded")
	return nil

}

// waitForResourcesDelete waits until the Kubernetes destroys all the resources which it had created
func (c *GKECluster) waitForResourcesDelete() error {

	log := log.WithFields(logrus.Fields{"cluster": c.modelCluster.Name, "zone": c.modelCluster.Location})

	log.Info("Waiting for deleting cluster resources")

	log.Info("Create compute service")
	csv, err := c.getComputeService()
	if err != nil {
		return errors.Wrap(err, "Error during creating compute service")
	}

	log.Info("Get project id")
	project, err := c.getProjectId()
	if err != nil {
		return errors.Wrap(err, "Error during getting project id")
	}

	clusterName := c.modelCluster.Name
	zone := c.modelCluster.Location
	log.Info("Find region by zone")
	region, err := findRegionByZone(csv, project, zone)
	if err != nil {
		return errors.Wrap(err, "Error during finding region by zone")
	}

	regionName := region.Name
	log.Infof("Region name: %s", regionName)

	lb := newLoadBalancerHelper(csv, project, regionName, zone, clusterName)

	maxAttempts := viper.GetInt(pipConfig.GKEResourceDeleteWaitAttempt)
	sleepSeconds := viper.GetInt(pipConfig.GKEResourceDeleteSleepSeconds)

	checkers := resourceCheckers{
		newFirewallChecker(csv, project, clusterName),
		newForwardingRulesChecker(csv, project, regionName, lb),
		newTargetPoolsChecker(csv, project, clusterName, regionName, zone, lb),
	}

	err = checkResources(checkers, maxAttempts, sleepSeconds)
	if err != nil {
		return errors.Wrap(err, "Error during checking resources")
	}

	return nil
}

// findRegionByZone returns region by zone
func findRegionByZone(csv *gkeCompute.Service, project, zone string) (*gkeCompute.Region, error) {

	regions, err := listRegions(csv, project)
	if err != nil {
		return nil, err
	}

	for _, r := range regions {
		if r != nil {
			for _, z := range r.Zones {
				if z == getZoneScope(project, zone) {
					return r, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("cannot find zone[%s] in regions", zone)
}

func getZoneScope(project, zone string) string {
	return fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s", project, zone)
}

// listRegions returns all region in project
func listRegions(csv *gkeCompute.Service, project string) ([]*gkeCompute.Region, error) {
	regionList, err := csv.Regions.List(project).Context(context.Background()).Do()
	if err != nil {
		return nil, err
	}
	return regionList.Items, nil
}

func (c *GKECluster) getRegionByZone(project string, zone string) (string, error) {

	log.Infof("start getting region by zone[%s]", zone)
	csv, err := c.getComputeService()
	if err != nil {
		return "", errors.Wrap(err, "Error during creating compute service")
	}

	regions, err := csv.Regions.List(project).Context(context.Background()).Do()
	if err != nil {
		return "", errors.Wrap(err, "Error during listing regions")
	}

	for _, i := range regions.Items {
		for _, z := range i.Zones {
			zoneScope := getZoneScope(project, zone)
			if z == zoneScope {
				log.Infof("match region: %s", i.Name)
				return i.Name, nil
			}
		}
	}

	return "", fmt.Errorf("there's no zone [%s] in regions", zone)
}

// checkResources checks all load balancer resources deleted by Kubernetes
func checkResources(checkers resourceCheckers, maxAttempts, sleepSeconds int) error {

	for _, rc := range checkers {

		log := log.WithFields(logrus.Fields{"type": rc.getType()})

		log.Info("list resources")

		resources, err := rc.list()
		if err != nil {
			return err
		}

		log.Infof("Resource count: %d", len(resources))

		for _, resource := range resources {

			log := log.WithFields(logrus.Fields{"resource": resource, "type": rc.getType()})

			attempt := 0
			deleted := false

			for attempt <= maxAttempts && !deleted {
				log.Debugf("Waiting for resource to be deleted %d/%d", attempt, maxAttempts)
				err := rc.isResourceDeleted(resource)
				if err == nil {
					log.Info("Resource deleted")
					deleted = true
					break
				} else {
					log.Warn(err.Error())
					time.Sleep(time.Second * time.Duration(sleepSeconds))
				}
				attempt++
			}

			if !deleted {
				log.Info("force delete")
				if err := rc.forceDelete(resource); err != nil {
					return err
				}
			}

		}
	}

	return nil
}

// UpdateCluster updates GKE cluster in cloud
func (c *GKECluster) UpdateCluster(updateRequest *pkgCluster.UpdateClusterRequest, userId uint) error {

	log.Info("Start updating cluster (gke)")

	svc, err := c.getGoogleServiceClient()
	if err != nil {
		return err
	}

	updateNodePoolsModel, err := createNodePoolsModelFromRequestData(updateRequest.GKE.NodePools, userId)
	if err != nil {
		return err
	}

	googleClusterModel := model.GKEClusterModel{}

	copier.Copy(&googleClusterModel, &c.modelCluster.GKE)
	googleClusterModel.NodePools = updateNodePoolsModel

	googleClusterModel.NodeVersion = updateRequest.GKE.NodeVersion
	googleClusterModel.MasterVersion = updateRequest.GKE.Master.Version

	updatedNodePools, err := createNodePoolsFromClusterModel(&googleClusterModel)
	if err != nil {
		return err
	}

	secretItem, err := c.GetSecretWithValidation()
	if err != nil {
		return err
	}

	projectId := secretItem.GetValue(pkgSecret.ProjectId)

	cc := googleCluster{
		Name:          c.modelCluster.Name,
		ProjectID:     projectId,
		Zone:          c.modelCluster.Location,
		MasterVersion: updateRequest.GKE.Master.Version,
		NodePools:     updatedNodePools,
	}

	res, err := callUpdateClusterGoogle(svc, cc, c.modelCluster.Location, projectId)
	if err != nil {
		be := getBanzaiErrorFromError(err)
		// TODO status code !?
		return errors.New(be.Message)
	}
	log.Info("Cluster update succeeded")
	c.googleCluster = res

	c.updateCurrentVersions(res)

	// update model to save
	c.updateModel(res, updatedNodePools)

	return nil

}

func (c *GKECluster) updateModel(cluster *gke.Cluster, updatedNodePools []*gke.NodePool) {
	// Update the model from the cluster data read back from Google
	c.modelCluster.GKE.MasterVersion = cluster.CurrentMasterVersion
	c.modelCluster.GKE.NodeVersion = cluster.CurrentNodeVersion

	var newNodePoolsModels []*model.GKENodePoolModel
	for _, clusterNodePool := range cluster.NodePools {
		updated := false

		for _, nodePoolModel := range c.modelCluster.GKE.NodePools {
			if clusterNodePool.Name == nodePoolModel.Name {
				nodePoolModel.NodeInstanceType = clusterNodePool.Config.MachineType

				if clusterNodePool.Autoscaling != nil {
					nodePoolModel.Autoscaling = clusterNodePool.Autoscaling.Enabled
					nodePoolModel.NodeMinCount = int(clusterNodePool.Autoscaling.MinNodeCount)
					nodePoolModel.NodeMaxCount = int(clusterNodePool.Autoscaling.MaxNodeCount)
				}

				// TODO: This is ugly but Google API doesn't expose the current node count for a node pool
				for _, updatedNodePool := range updatedNodePools {
					if updatedNodePool.Name == clusterNodePool.Name {
						nodePoolModel.NodeCount = int(updatedNodePool.InitialNodeCount)

						break
					}
				}

				updated = true
				break
			}
		}

		if !updated {
			nodePoolModelAdd := &model.GKENodePoolModel{
				Name:             clusterNodePool.Name,
				NodeInstanceType: clusterNodePool.Config.MachineType,
				NodeCount:        int(clusterNodePool.InitialNodeCount),
			}
			if clusterNodePool.Autoscaling != nil {
				nodePoolModelAdd.Autoscaling = clusterNodePool.Autoscaling.Enabled
				nodePoolModelAdd.NodeMinCount = int(clusterNodePool.Autoscaling.MinNodeCount)
				nodePoolModelAdd.NodeMaxCount = int(clusterNodePool.Autoscaling.MaxNodeCount)
			}

			newNodePoolsModels = append(newNodePoolsModels, nodePoolModelAdd)
		}
	}

	for _, newNodePoolModel := range newNodePoolsModels {
		c.modelCluster.GKE.NodePools = append(c.modelCluster.GKE.NodePools, newNodePoolModel)
	}

	// mark for deletion the node pool model entries that has no corresponding node pool in the cluster
	for _, nodePoolModel := range c.modelCluster.GKE.NodePools {
		found := false

		for _, clusterNodePool := range cluster.NodePools {
			if nodePoolModel.Name == clusterNodePool.Name {
				found = true
				break
			}
		}

		if !found {
			nodePoolModel.Delete = true
		}

	}

}

//GetID returns the specified cluster id
func (c *GKECluster) GetID() uint {
	return c.modelCluster.ID
}

func (c *GKECluster) GetUID() string {
	return c.modelCluster.UID
}

//GetModel returns the whole clusterModel
func (c *GKECluster) GetModel() *model.ClusterModel {
	return c.modelCluster
}

func (c *GKECluster) getGoogleServiceClient() (*gke.Service, error) {

	client, err := c.newClientFromCredentials()
	if err != nil {
		return nil, err
	}

	//New client from credentials
	return gke.New(client)
}

// GKE cluster to google calls
type googleCluster struct {
	// ProjectID is the ID of your project to use when creating a cluster
	ProjectID string `json:"projectId,omitempty"`
	// The zone to launch the cluster
	Zone string
	// The IP address range of the container pods
	ClusterIpv4Cidr string
	// An optional description of this cluster
	Description string

	// the kubernetes master version
	MasterVersion string
	// The authentication information for accessing the master
	MasterAuth *gke.MasterAuth

	// The name of this cluster
	Name string
	// The path to the credential file(key.json)
	CredentialPath string
	// The content of the credential
	CredentialContent string
	// the temp file of the credential
	TempCredentialPath string
	// Enable alpha feature
	EnableAlphaFeature bool
	// Configuration for the HTTP (L7) load balancing controller addon
	HTTPLoadBalancing bool
	// Configuration for the horizontal pod autoscaling feature, which increases or decreases the number of replica pods a replication controller has based on the resource usage of the existing pods
	HorizontalPodAutoscaling bool
	// Configuration for the Kubernetes Dashboard
	KubernetesDashboard bool
	// Configuration for NetworkPolicy
	NetworkPolicyConfig bool
	// The list of Google Compute Engine locations in which the cluster's nodes should be located
	Locations []string
	// Network
	Network string
	// Sub Network
	SubNetwork string
	// Configuration for LegacyAbac
	LegacyAbac bool
	// Image Type
	ImageType string
	// The node pools the cluster's nodes are created from
	NodePools []*gke.NodePool
}

func generateClusterCreateRequest(cc googleCluster) *gke.CreateClusterRequest {
	request := gke.CreateClusterRequest{
		Cluster: &gke.Cluster{},
	}
	request.Cluster.Name = cc.Name
	request.Cluster.Zone = cc.Zone
	request.Cluster.InitialClusterVersion = cc.MasterVersion
	request.Cluster.ClusterIpv4Cidr = cc.ClusterIpv4Cidr
	request.Cluster.Description = cc.Description
	request.Cluster.EnableKubernetesAlpha = cc.EnableAlphaFeature
	request.Cluster.AddonsConfig = &gke.AddonsConfig{
		HttpLoadBalancing:        &gke.HttpLoadBalancing{Disabled: !cc.HTTPLoadBalancing},
		HorizontalPodAutoscaling: &gke.HorizontalPodAutoscaling{Disabled: !cc.HorizontalPodAutoscaling},
		KubernetesDashboard:      &gke.KubernetesDashboard{Disabled: !cc.KubernetesDashboard},
		//	NetworkPolicyConfig:      &gke.NetworkPolicyConfig{Disabled: !cc.NetworkPolicyConfig},
	}
	request.Cluster.Network = cc.Network
	request.Cluster.Subnetwork = cc.SubNetwork
	request.Cluster.LegacyAbac = &gke.LegacyAbac{
		Enabled: true,
	}
	request.Cluster.MasterAuth = &gke.MasterAuth{}
	request.Cluster.NodePools = cc.NodePools

	return &request
}

//createNodePoolsFromClusterModel creates an array of gke NodePool from the given cluster model
func createNodePoolsFromClusterModel(clusterModel *model.GKEClusterModel) ([]*gke.NodePool, error) {
	nodePoolsCount := len(clusterModel.NodePools)
	if nodePoolsCount == 0 {
		return nil, pkgErrors.ErrorNodePoolNotProvided
	}

	nodePools := make([]*gke.NodePool, nodePoolsCount)

	for i := 0; i < nodePoolsCount; i++ {
		nodePoolModel := clusterModel.NodePools[i]

		nodePools[i] = &gke.NodePool{
			Name: nodePoolModel.Name,
			Config: &gke.NodeConfig{
				Labels:      map[string]string{pkgCommon.LabelKey: nodePoolModel.Name},
				MachineType: nodePoolModel.NodeInstanceType,
				OauthScopes: []string{
					"https://www.googleapis.com/auth/logging.write",
					"https://www.googleapis.com/auth/monitoring",
					"https://www.googleapis.com/auth/devstorage.read_write",
					"https://www.googleapis.com/auth/cloud-platform",
					"https://www.googleapis.com/auth/compute",
				},
			},
			InitialNodeCount: int64(nodePoolModel.NodeCount),
			Version:          clusterModel.NodeVersion,
		}

		if nodePoolModel.Autoscaling {
			nodePools[i].Autoscaling = &gke.NodePoolAutoscaling{
				Enabled:      true,
				MinNodeCount: int64(nodePoolModel.NodeMinCount),
				MaxNodeCount: int64(nodePoolModel.NodeMaxCount),
			}
		} else {
			nodePools[i].Autoscaling = &gke.NodePoolAutoscaling{
				Enabled: false,
			}
		}

	}

	return nodePools, nil
}

// createNodePoolsRequestDataFromNodePoolModel returns a map of node pool name -> GoogleNodePool from the given nodePoolsModel
func createNodePoolsRequestDataFromNodePoolModel(nodePoolsModel []*model.GKENodePoolModel) (map[string]*pkgClusterGoogle.NodePool, error) {
	nodePoolsCount := len(nodePoolsModel)
	if nodePoolsCount == 0 {
		return nil, pkgErrors.ErrorNodePoolNotProvided
	}

	nodePools := make(map[string]*pkgClusterGoogle.NodePool)

	for i := 0; i < nodePoolsCount; i++ {
		nodePoolModel := nodePoolsModel[i]
		nodePools[nodePoolModel.Name] = &pkgClusterGoogle.NodePool{
			Autoscaling:      nodePoolModel.Autoscaling,
			MinCount:         nodePoolModel.NodeMinCount,
			MaxCount:         nodePoolModel.NodeMaxCount,
			Count:            nodePoolModel.NodeCount,
			NodeInstanceType: nodePoolModel.NodeInstanceType,
		}
	}

	return nodePools, nil
}

func getBanzaiErrorFromError(err error) *pkgCommon.BanzaiResponse {

	if err == nil {
		// error is nil
		return &pkgCommon.BanzaiResponse{
			StatusCode: http.StatusInternalServerError,
		}
	}

	googleErr, ok := err.(*googleapi.Error)
	if ok {
		// error is googleapi error
		return &pkgCommon.BanzaiResponse{
			StatusCode: googleErr.Code,
			Message:    googleErr.Message,
		}
	}

	// default
	return &pkgCommon.BanzaiResponse{
		StatusCode: http.StatusInternalServerError,
		Message:    err.Error(),
	}
}

func getClusterGoogle(svc *gke.Service, cc googleCluster) (*gke.Cluster, error) {
	return svc.Projects.Zones.Clusters.Get(cc.ProjectID, cc.Zone, cc.Name).Context(context.TODO()).Do()
}

func (c *GKECluster) callDeleteCluster(cc *googleCluster) error {
	svc, err := c.getGoogleServiceClient()
	if err != nil {
		return err
	}
	log.Info("Get Google Service Client succeeded")

	log.Infof("Removing cluster %v from project %v, zone %v", cc.Name, cc.ProjectID, cc.Zone)
	deleteCall, err := svc.Projects.Zones.Clusters.Delete(cc.ProjectID, cc.Zone, cc.Name).Context(context.Background()).Do()
	if err != nil && !strings.Contains(err.Error(), "notFound") {
		return err
	} else if err == nil {
		log.Infof("Cluster %v delete is called. Status Code %v", cc.Name, deleteCall.HTTPStatusCode)
	} else {
		log.Errorf("Cluster %s doesn't exist", cc.Name)
		return err
	}
	os.RemoveAll(cc.TempCredentialPath)
	return nil
}

func callUpdateClusterGoogle(svc *gke.Service, cc googleCluster, location, projectId string) (*gke.Cluster, error) {

	log.Infof("Updating cluster: %#v", cc)

	updatedCluster, err := getClusterGoogle(svc, cc)
	if err != nil {
		return nil, err
	}

	if cc.MasterVersion != "" && cc.MasterVersion != updatedCluster.CurrentMasterVersion {
		log.Infof("Updating master to %v version", cc.MasterVersion)
		updateCall, err := svc.Projects.Zones.Clusters.Update(cc.ProjectID, cc.Zone, cc.Name, &gke.UpdateClusterRequest{
			Update: &gke.ClusterUpdate{
				DesiredMasterVersion: cc.MasterVersion,
			},
		}).Context(context.Background()).Do()
		if err != nil {
			return nil, err
		}
		log.Infof("Cluster %s update is called for project %s and zone %s. Status Code %v", cc.Name, cc.ProjectID, cc.Zone, updateCall.HTTPStatusCode)
		if err = waitForOperation(newContainerOperation(svc, projectId, location), updateCall.Name); err != nil {
			return nil, err
		}

		updatedCluster, err = getClusterGoogle(svc, cc)
		if err != nil {
			return nil, err
		}

	}

	// Collect node pools that have to be deleted and delete them before
	// resizing exiting ones or creating new ones to minimize tha chance
	// of hitting quota limits
	var nodePoolsToDelete []string
	for _, currentClusterNodePool := range updatedCluster.NodePools {
		var i int
		for i = 0; i < len(cc.NodePools); i++ {
			if currentClusterNodePool.Name == cc.NodePools[i].Name {
				break
			}
		}

		if i == len(cc.NodePools) {
			// cluster node pool with given name not found in the update request thus we need to delete it
			nodePoolsToDelete = append(nodePoolsToDelete, currentClusterNodePool.Name)
		}
	}

	var nodePoolsToCreate []*gke.NodePool
	for _, nodePoolFromUpdReq := range cc.NodePools {
		var i int
		for i = 0; i < len(updatedCluster.NodePools); i++ {
			if nodePoolFromUpdReq.Name == updatedCluster.NodePools[i].Name {
				break
			}
		}
		if i == len(updatedCluster.NodePools) {
			nodePoolsToCreate = append(nodePoolsToCreate, nodePoolFromUpdReq)
		}
	}

	// Delete node pools
	for _, nodePoolName := range nodePoolsToDelete {
		log.Infof("Deleting node pool %s", nodePoolName)

		deleteCall, err :=
			svc.Projects.Zones.Clusters.NodePools.Delete(cc.ProjectID, cc.Zone, cc.Name, nodePoolName).Context(
				context.Background()).Do()

		if err != nil {
			return nil, err
		}
		log.Infof("Node pool %s delete is called for project %s, zone %s and cluster %s. Status Code %v", nodePoolName, cc.ProjectID, cc.Zone, cc.Name, deleteCall.HTTPStatusCode)
		if err = waitForOperation(newContainerOperation(svc, projectId, location), deleteCall.Name); err != nil {
			return nil, err
		}
		updatedCluster, err = getClusterGoogle(svc, cc)
		if err != nil {
			return nil, err
		}
	}

	// Update node pools
	for _, nodePool := range cc.NodePools {
		for i := 0; i < len(updatedCluster.NodePools); i++ {
			if updatedCluster.NodePools[i].Name == nodePool.Name {

				if nodePool.Version != "" && nodePool.Version != updatedCluster.NodePools[i].Version {
					log.Infof("Updating node pool %s to %v version", nodePool.Name, nodePool.Version)
					updateCall, err := svc.Projects.Zones.Clusters.NodePools.Update(cc.ProjectID, cc.Zone, cc.Name, nodePool.Name, &gke.UpdateNodePoolRequest{
						NodeVersion: nodePool.Version,
					}).Context(context.Background()).Do()
					if err != nil {
						return nil, err
					}
					log.Infof("Node pool %s update is called for project %s, zone %s and cluster %s. Status Code %v", nodePool.Name, cc.ProjectID, cc.Zone, cc.Name, updateCall.HTTPStatusCode)
					if err := waitForOperation(newContainerOperation(svc, projectId, location), updateCall.Name); err != nil {
						return nil, err
					}
				}

				if autoscalingHasBeenUpdated(nodePool, updatedCluster.NodePools[i]) {
					var err error

					autoScalingInput := &gke.SetNodePoolAutoscalingRequest{
						Autoscaling: &gke.NodePoolAutoscaling{
							Enabled: false,
						},
					}

					if nodePool.Autoscaling.Enabled {
						log.Infof("Updating node pool %s enable Autoscaling", nodePool.Name)
						autoScalingInput = &gke.SetNodePoolAutoscalingRequest{
							Autoscaling: &gke.NodePoolAutoscaling{
								Enabled:      true,
								MinNodeCount: nodePool.Autoscaling.MinNodeCount,
								MaxNodeCount: nodePool.Autoscaling.MaxNodeCount,
							},
						}
					} else {
						log.Infof("Updating node pool %s disable Autoscaling", nodePool.Name)
					}

					operation, err := svc.Projects.Zones.Clusters.NodePools.Autoscaling(cc.ProjectID, cc.Zone, cc.Name, nodePool.Name, autoScalingInput).Context(context.Background()).Do()
					if err != nil {
						return nil, err
					}

					log.Infof("Node pool %s update is called for project %s, zone %s and cluster %s", nodePool.Name, cc.ProjectID, cc.Zone, cc.Name)
					if err = waitForOperation(newContainerOperation(svc, projectId, location), operation.Name); err != nil {
						return nil, err
					}

					updatedCluster, err = getClusterGoogle(svc, cc)
					if err != nil {
						return nil, err
					}

				}

				if nodePool.InitialNodeCount > 0 && nodePool.InitialNodeCount != updatedCluster.NodePools[i].InitialNodeCount {
					log.Infof("Updating node size to %v for node pool %s", nodePool.InitialNodeCount, nodePool.Name)
					updateCall, err := svc.Projects.Zones.Clusters.NodePools.SetSize(cc.ProjectID, cc.Zone, cc.Name, nodePool.Name, &gke.SetNodePoolSizeRequest{
						NodeCount: nodePool.InitialNodeCount,
					}).Context(context.Background()).Do()
					if err != nil {
						return nil, err
					}
					log.Infof("Node pool %s size change is called for project %s, zone %s and cluster %s. Status Code %v", nodePool.Name, cc.ProjectID, cc.Zone, cc.Name, updateCall.HTTPStatusCode)
					if err = waitForOperation(newContainerOperation(svc, projectId, location), updateCall.Name); err != nil {
						return nil, err
					}

					updatedCluster, err = getClusterGoogle(svc, cc)
					if err != nil {
						return nil, err
					}
				}

				break
			}
		}
	}

	// Create node pools
	for _, nodePoolToCreate := range nodePoolsToCreate {
		log.Infof("Creating node pool %s", nodePoolToCreate.Name)

		createCall, err :=
			svc.Projects.Zones.Clusters.NodePools.Create(cc.ProjectID, cc.Zone, cc.Name, &gke.CreateNodePoolRequest{
				NodePool: nodePoolToCreate,
			}).Context(context.Background()).Do()

		if err != nil {
			return nil, err
		}
		log.Infof("Node pool %s create is called for project %s, zone %s and cluster %s. Status Code %v", nodePoolToCreate.Name, cc.ProjectID, cc.Zone, cc.Name, createCall.HTTPStatusCode)
		if err = waitForOperation(newContainerOperation(svc, projectId, location), createCall.Name); err != nil {
			return nil, err
		}

		updatedCluster, err = getClusterGoogle(svc, cc)
		if err != nil {
			return nil, err
		}

	}

	return updatedCluster, nil
}

func autoscalingHasBeenUpdated(updatedNodePool *gke.NodePool, actualNodePool *gke.NodePool) bool {
	if actualNodePool.Autoscaling == nil {
		return updatedNodePool.Autoscaling.Enabled
	}
	if updatedNodePool.Autoscaling.Enabled && actualNodePool.Autoscaling.Enabled {
		if updatedNodePool.Autoscaling.MinNodeCount != actualNodePool.Autoscaling.MinNodeCount {
			return true
		}
		if updatedNodePool.Autoscaling.MaxNodeCount != actualNodePool.Autoscaling.MaxNodeCount {
			return true
		}
		return false
	} else if !updatedNodePool.Autoscaling.Enabled && !actualNodePool.Autoscaling.Enabled {
		return false
	}
	return true
}

func (c *GKECluster) getGoogleKubernetesConfig() ([]byte, error) {

	log.Info("Get Google Service Client")
	svc, err := c.getGoogleServiceClient()
	if err != nil {
		return nil, err
	}
	log.Info("Get Google Service Client succeeded")

	secretItem, err := c.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}

	log.Infof("Get gke cluster with name %s", c.modelCluster.Name)
	cl, err := getClusterGoogle(svc, googleCluster{
		Name:      c.modelCluster.Name,
		ProjectID: secretItem.GetValue(pkgSecret.ProjectId),
		Zone:      c.modelCluster.Location,
	})

	if err != nil {
		return nil, err
	}

	log.Info("Generate Service Account token")
	serviceAccountToken, err := generateServiceAccountTokenForGke(cl)
	if err != nil {
		be := getBanzaiErrorFromError(err)
		// TODO status code !?
		return nil, errors.New(be.Message)
	}

	finalCl := kubernetesCluster{
		ClientCertificate:   cl.MasterAuth.ClientCertificate,
		ClientKey:           cl.MasterAuth.ClientKey,
		RootCACert:          cl.MasterAuth.ClusterCaCertificate,
		Username:            cl.MasterAuth.Username,
		Password:            cl.MasterAuth.Password,
		Version:             cl.CurrentMasterVersion,
		Endpoint:            cl.Endpoint,
		NodeCount:           cl.CurrentNodeCount,
		Metadata:            map[string]string{},
		ServiceAccountToken: serviceAccountToken,
		Status:              cl.Status,
	}

	finalCl.Metadata["nodePools"] = fmt.Sprintf("%v", cl.NodePools)

	// TODO if the final solution is NOT SAVE CONFIG TO FILE than rename the method and change log message
	log.Info("Start save config file")
	config, err := storeConfig(&finalCl, c.modelCluster.Name)
	if err != nil {
		be := getBanzaiErrorFromError(err)
		// TODO status code !?
		return nil, errors.New(be.Message)
	}
	return config, nil
}

func generateServiceAccountTokenForGke(cluster *gke.Cluster) (string, error) {
	capem, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return "", err
	}
	certData, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClientCertificate)
	if err != nil {
		return "", err
	}
	keyData, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClientKey)
	if err != nil {
		return "", err
	}
	host := cluster.Endpoint
	if !strings.HasPrefix(host, "https://") {
		host = fmt.Sprintf("https://%s", host)
	}

	// in here we have to use http basic auth otherwise we can't get the permission to create cluster role
	config := &rest.Config{
		Host: host,
		TLSClientConfig: rest.TLSClientConfig{
			CAData:   capem,
			CertData: certData,
			KeyData:  keyData,
		},
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}

	return generateServiceAccountToken(clientset)
}

func generateServiceAccountToken(clientset *kubernetes.Clientset) (string, error) {
	serviceAccount := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: netesDefault,
		},
	}

	_, err := clientset.CoreV1().ServiceAccounts(defaultNamespace).Create(serviceAccount)
	if err != nil && !k8sErrors.IsAlreadyExists(err) {
		return "", err
	}

	adminRole := &v1beta1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterAdmin,
		},
		Rules: []v1beta1.PolicyRule{
			{
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
			{
				NonResourceURLs: []string{"*"},
				Verbs:           []string{"*"},
			},
		},
	}
	clusterAdminRole, err := clientset.RbacV1beta1().ClusterRoles().Get(clusterAdmin, metav1.GetOptions{})
	if err != nil {
		clusterAdminRole, err = clientset.RbacV1beta1().ClusterRoles().Create(adminRole)
		if err != nil {
			return "", err
		}
	}

	clusterRoleBinding := &v1beta1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "netes-default-clusterRoleBinding",
		},
		Subjects: []v1beta1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount.Name,
				Namespace: "default",
				APIGroup:  v1.GroupName,
			},
		},
		RoleRef: v1beta1.RoleRef{
			Kind:     "ClusterRole",
			Name:     clusterAdminRole.Name,
			APIGroup: v1beta1.GroupName,
		},
	}
	if _, err = clientset.RbacV1beta1().ClusterRoleBindings().Create(clusterRoleBinding); err != nil && !k8sErrors.IsAlreadyExists(err) {
		return "", err
	}

	if serviceAccount, err = clientset.CoreV1().ServiceAccounts(defaultNamespace).Get(serviceAccount.Name, metav1.GetOptions{}); err != nil {
		return "", err
	}

	if len(serviceAccount.Secrets) > 0 {
		secret := serviceAccount.Secrets[0]
		secretObj, err := clientset.CoreV1().Secrets(defaultNamespace).Get(secret.Name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		if token, ok := secretObj.Data["token"]; ok {
			return string(token), nil
		}
	}
	return "", fmt.Errorf("failed to configure serviceAccountToken")
}

// storeConfig saves config file
func storeConfig(c *kubernetesCluster, name string) ([]byte, error) {
	isBasicOn := false
	if c.Username != "" && c.Password != "" {
		isBasicOn = true
	}
	username, password, token := "", "", ""
	if isBasicOn {
		username = c.Username
		password = c.Password
	} else {
		token = c.ServiceAccountToken
	}

	configFile := fmt.Sprintf("%s/%s/config", pipConfig.GetStateStorePath(""), name)
	config := kubeConfig{}
	if _, err := os.Stat(configFile); err == nil {
		data, err := ioutil.ReadFile(configFile)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, err
		}
	}
	config.APIVersion = "v1"
	config.Kind = "Config"

	// setup clusters
	host := c.Endpoint
	if !strings.HasPrefix(host, "https://") {
		host = fmt.Sprintf("https://%s", host)
	}
	cluster := configCluster{
		Cluster: dataCluster{
			CertificateAuthorityData: string(c.RootCACert),
			Server: host,
		},
		Name: c.Name,
	}
	if config.Clusters == nil || len(config.Clusters) == 0 {
		config.Clusters = []configCluster{cluster}
	} else {
		exist := false
		for _, cluster := range config.Clusters {
			if cluster.Name == c.Name {
				exist = true
				break
			}
		}
		if !exist {
			config.Clusters = append(config.Clusters, cluster)
		}
	}

	var provider authProvider
	if len(c.AuthProviderName) != 0 || len(c.AuthAccessToken) != 0 {
		provider = authProvider{
			ProviderConfig: providerConfig{
				AccessToken: c.AuthAccessToken,
				Expiry:      c.AuthAccessTokenExpiry,
			},
			Name: c.AuthProviderName,
		}
	}

	// setup users
	user := configUser{
		User: userData{
			Token:                 token,
			Username:              username,
			Password:              password,
			ClientCertificateData: c.ClientCertificate,
			ClientKeyData:         c.ClientKey,
			AuthProvider:          provider,
		},
		Name: c.Name,
	}
	if config.Users == nil || len(config.Users) == 0 {
		config.Users = []configUser{user}
	} else {
		exist := false
		for _, user := range config.Users {
			if user.Name == c.Name {
				exist = true
				break
			}
		}
		if !exist {
			config.Users = append(config.Users, user)
		}
	}

	config.CurrentContext = c.CurrentContext

	// setup context
	context := configContext{
		Context: contextData{
			Cluster: c.Name,
			User:    c.Name,
		},
		Name: c.Name,
	}
	if config.Contexts == nil || len(config.Contexts) == 0 {
		config.Contexts = []configContext{context}
	} else {
		exist := false
		for _, context := range config.Contexts {
			if context.Name == c.Name {
				exist = true
				break
			}
		}
		if !exist {
			config.Contexts = append(config.Contexts, context)
		}
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}

	// TODO save or not save, this is the question
	//fileToWrite := fmt.Sprintf("./statestore/%s/config", name)
	//if err := utils.WriteToFile(data, fileToWrite); err != nil {
	//	return nil, err
	//}
	//log.Infof("KubeConfig files is saved to %s", fileToWrite)

	return data, nil
}

// kubernetesCluster represents a kubernetes cluster
type kubernetesCluster struct {
	// The name of the cluster
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
	// The status of the cluster
	Status string `json:"status,omitempty" yaml:"status,omitempty"`
	// Kubernetes cluster version
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	// Service account token to access kubernetes API
	ServiceAccountToken string `json:"serviceAccountToken,omitempty" yaml:"service_account_token,omitempty"`
	// Kubernetes API master endpoint
	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`
	// Username for http basic authentication
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	// Password for http basic authentication
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	// Root CaCertificate for API server(base64 encoded)
	RootCACert string `json:"rootCACert,omitempty" yaml:"root_ca_cert,omitempty"`
	// Client Certificate(base64 encoded)
	ClientCertificate string `json:"clientCertificate,omitempty" yaml:"client_certificate,omitempty"`
	// Client private key(base64 encoded)
	ClientKey string `json:"clientKey,omitempty" yaml:"client_key,omitempty"`
	// Node count in the cluster
	NodeCount int64 `json:"nodeCount,omitempty" yaml:"node_count,omitempty"`
	// Metadata store specific driver options per cloud provider
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	AuthProviderName      string `json:"auth_provider_name,omitempty"`
	AuthAccessToken       string `json:"auth_access_token,omitempty"`
	AuthAccessTokenExpiry string `json:"auth_access_token_expiry,omitempty"`
	CurrentContext        string `json:"current_context,omitempty"`
}

type kubeConfig struct {
	APIVersion     string          `yaml:"apiVersion,omitempty"`
	Clusters       []configCluster `yaml:"clusters,omitempty"`
	Contexts       []configContext `yaml:"contexts,omitempty"`
	Users          []configUser    `yaml:"users,omitempty"`
	CurrentContext string          `yaml:"current-context,omitempty"`
	Kind           string          `yaml:"kind,omitempty"`
	//Kubernetes config contains an invalid map for the go yaml parser,
	//preferences field always look like this {} this should be {{}} so
	//yaml.Unmarshal fails with a cryptic error message which says string
	//cannot be casted as !map
	//Preferences    string          `yaml:"preferences,omitempty"`
}

type configCluster struct {
	Cluster dataCluster `yaml:"cluster,omitempty"`
	Name    string      `yaml:"name,omitempty"`
}

type dataCluster struct {
	CertificateAuthorityData string `yaml:"certificate-authority-data,omitempty"`
	Server                   string `yaml:"server,omitempty"`
}

type configContext struct {
	Context contextData `yaml:"context,omitempty"`
	Name    string      `yaml:"name,omitempty"`
}

type contextData struct {
	Cluster string `yaml:"cluster,omitempty"`
	User    string `yaml:"user,omitempty"`
}

type configUser struct {
	Name string   `yaml:"name,omitempty"`
	User userData `yaml:"user,omitempty"`
}

type userData struct {
	Token                 string       `yaml:"token,omitempty"`
	Username              string       `yaml:"username,omitempty"`
	Password              string       `yaml:"password,omitempty"`
	ClientCertificateData string       `yaml:"client-certificate-data,omitempty"`
	ClientKeyData         string       `yaml:"client-key-data,omitempty"`
	AuthProvider          authProvider `yaml:"auth-provider,omitempty"`
}

type authProvider struct {
	ProviderConfig providerConfig `yaml:"config,omitempty"`
	Name           string         `yaml:"name,omitempty"`
}

type providerConfig struct {
	AccessToken string `yaml:"access-token,omitempty"`
	Expiry      string `yaml:"expiry,omitempty"`
}

//CreateGKEClusterFromModel creates ClusterModel struct from model
func CreateGKEClusterFromModel(clusterModel *model.ClusterModel) (*GKECluster, error) {
	log.Debug("Create ClusterModel struct from the request")
	gkeCluster := GKECluster{
		modelCluster: clusterModel,
	}
	return &gkeCluster, nil
}

//AddDefaultsToUpdate adds defaults to update request
func (c *GKECluster) AddDefaultsToUpdate(r *pkgCluster.UpdateClusterRequest) {

	// TODO: error handling
	defGooglePools, _ := createNodePoolsFromClusterModel(&c.modelCluster.GKE)

	defGoogleMaster := &pkgClusterGoogle.Master{
		Version: c.modelCluster.GKE.MasterVersion,
	}

	// ---- [ Node check ] ---- //
	if r.GKE.NodePools == nil {
		log.Warn("'nodePools' field is empty. Load it from stored data.")

		r.GKE.NodePools = make(map[string]*pkgClusterGoogle.NodePool)
		for _, nodePool := range defGooglePools {
			r.GKE.NodePools[nodePool.Name] = &pkgClusterGoogle.NodePool{
				Count:            int(nodePool.InitialNodeCount),
				NodeInstanceType: nodePool.Config.MachineType,
			}
			if nodePool.Autoscaling != nil {
				r.GKE.NodePools[nodePool.Name].Autoscaling = nodePool.Autoscaling.Enabled
				r.GKE.NodePools[nodePool.Name].MinCount = int(nodePool.Autoscaling.MinNodeCount)
				r.GKE.NodePools[nodePool.Name].MaxCount = int(nodePool.Autoscaling.MaxNodeCount)
			}
		}
	}
	// ---- [ Master check ] ---- //
	if r.GKE.Master == nil {
		log.Warn("'master' field is empty. Load it from stored data.")
		r.GKE.Master = defGoogleMaster
	}

	// ---- [ NodeCount check] ---- //
	for name, nodePoolData := range r.GKE.NodePools {
		if nodePoolData.Count == 0 {
			// initialize with count read from db
			var i int
			for i = 0; i < len(c.modelCluster.GKE.NodePools); i++ {
				if c.modelCluster.GKE.NodePools[i].Name == name {
					nodePoolData.Count = c.modelCluster.GKE.NodePools[i].NodeCount
					log.Warnf("Node count for node pool %s initiated from database to value: %d", name, nodePoolData.Count)
					break
				}
			}
			if i == len(c.modelCluster.GKE.NodePools) {
				// node pool not found in db; set count to default value
				nodePoolData.Count = pkgCommon.DefaultNodeMinCount
				log.Warnf("Node count for node pool %s set to default value: ", name, nodePoolData.Count)
			}
		}
	}

	// ---- [ Node Version check] ---- //
	if len(r.GKE.NodeVersion) == 0 {
		nodeVersion := c.modelCluster.GKE.NodeVersion
		log.Warnf("Node K8s version: %s", nodeVersion)
		r.GKE.NodeVersion = nodeVersion
	}

	// ---- [ Master Version check] ---- //
	if len(r.GKE.Master.Version) == 0 {
		masterVersion := c.modelCluster.GKE.MasterVersion
		log.Warnf("Master K8s version: %s", masterVersion)
		r.GKE.Master.Version = masterVersion
	}

}

//CheckEqualityToUpdate validates the update request
func (c *GKECluster) CheckEqualityToUpdate(r *pkgCluster.UpdateClusterRequest) error {

	// create update request struct with the stored data to check equality
	nodePools, _ := createNodePoolsRequestDataFromNodePoolModel(c.modelCluster.GKE.NodePools)
	preCl := &pkgClusterGoogle.UpdateClusterGoogle{
		Master: &pkgClusterGoogle.Master{
			Version: c.modelCluster.GKE.MasterVersion,
		},
		NodeVersion: c.modelCluster.GKE.NodeVersion,
		NodePools:   nodePools,
	}

	log.Info("Check stored & updated cluster equals")

	// check equality
	return isDifferent(r.GKE, preCl)
}

//DeleteFromDatabase deletes model from the database
func (c *GKECluster) DeleteFromDatabase() error {
	err := c.modelCluster.Delete()
	if err != nil {
		return err
	}
	c.modelCluster = nil
	return nil
}

// GetGkeServerConfig returns all supported K8S versions
func GetGkeServerConfig(orgId uint, secretId, zone string) (*gke.ServerConfig, error) {
	g := GKECluster{
		modelCluster: &model.ClusterModel{
			OrganizationId: orgId,
			SecretId:       secretId,
			Cloud:          pkgCluster.Google,
		},
	}
	return g.GetGkeServerConfig(zone)
}

// GetGkeServerConfig returns configuration info about the Kubernetes Engine service.
func (c *GKECluster) GetGkeServerConfig(zone string) (*gke.ServerConfig, error) {

	log.Info("Start getting configuration info")

	log.Info("Get Google service client")
	svc, err := c.getGoogleServiceClient()
	if err != nil {
		return nil, err
	}

	projectId, err := c.getProjectId()
	if err != nil {
		return nil, err
	}

	serverConfig, err := svc.Projects.Zones.GetServerconfig(projectId, zone).Context(context.Background()).Do()
	if err != nil {
		return nil, err
	}

	log.Info("Getting server config succeeded")

	serverConfig.ValidMasterVersions = updateVersions(serverConfig.ValidMasterVersions)
	serverConfig.ValidNodeVersions = updateVersions(serverConfig.ValidNodeVersions)

	return serverConfig, nil

}

func updateVersions(validVersions []string) []string {

	log.Info("append `major.minor` K8S version format to valid GKE versions")

	var updatedVersions []string

	for _, v := range validVersions {

		version := strings.Split(v, ".")

		if len(version) >= 2 {
			majorMinor := fmt.Sprintf("%s.%s", version[0], version[1])
			if !utils.Contains(updatedVersions, majorMinor) && majorMinor != v {
				updatedVersions = append(updatedVersions, majorMinor, v)
			} else if !utils.Contains(updatedVersions, v) {
				updatedVersions = append(updatedVersions, v)
			}
		} else if !utils.Contains(updatedVersions, v) {
			updatedVersions = append(updatedVersions, v)
		}
	}

	return updatedVersions
}

// GetAllMachineTypesByZone returns all supported machine type by zone
func GetAllMachineTypesByZone(orgId uint, secretId, zone string) (map[string]pkgCluster.MachineType, error) {
	g := &GKECluster{
		modelCluster: &model.ClusterModel{
			OrganizationId: orgId,
			SecretId:       secretId,
			Cloud:          pkgCluster.Google,
		},
	}
	return g.GetAllMachineTypesByZone(zone)
}

// GetAllMachineTypesByZone lists supported machine types by zone
func (c *GKECluster) GetAllMachineTypesByZone(zone string) (map[string]pkgCluster.MachineType, error) {

	computeService, err := c.getComputeService()
	if err != nil {
		return nil, err
	}

	project, err := c.getProjectId()
	if err != nil {
		return nil, err
	}

	return getMachineTypes(computeService, project, zone)
}

// GetAllMachineTypes returns all supported machine types
func GetAllMachineTypes(orgId uint, secretId string) (map[string]pkgCluster.MachineType, error) {
	g := &GKECluster{
		modelCluster: &model.ClusterModel{
			OrganizationId: orgId,
			SecretId:       secretId,
			Cloud:          pkgCluster.Google,
		},
	}

	return g.GetAllMachineTypes()
}

// GetAllMachineTypes lists all supported machine types
func (c *GKECluster) GetAllMachineTypes() (map[string]pkgCluster.MachineType, error) {

	computeService, err := c.getComputeService()
	if err != nil {
		return nil, err
	}

	project, err := c.getProjectId()
	if err != nil {
		return nil, err
	}
	return getMachineTypesWithoutZones(computeService, project)
}

// getMachineTypesWithoutZones lists supported machine types in all zone
func getMachineTypesWithoutZones(csv *gkeCompute.Service, project string) (map[string]pkgCluster.MachineType, error) {
	response := make(map[string]pkgCluster.MachineType)
	req := csv.MachineTypes.AggregatedList(project)
	if err := req.Pages(context.Background(), func(list *gkeCompute.MachineTypeAggregatedList) error {
		for zone, item := range list.Items {
			var types []string
			for _, t := range item.MachineTypes {
				types = append(types, t.Name)
			}
			key := zone
			if strings.HasPrefix(key, zonePrefix) {
				key = zone[len(zonePrefix):]
			}
			if types != nil {
				response[key] = types
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return response, nil
}

const zonePrefix = "zones/"

// getMachineTypes returns supported machine types by zone
func getMachineTypes(csv *gkeCompute.Service, project, zone string) (map[string]pkgCluster.MachineType, error) {

	var machineTypes []string
	req := csv.MachineTypes.List(project, zone)
	if err := req.Pages(context.Background(), func(page *gkeCompute.MachineTypeList) error {
		for _, machineType := range page.Items {
			machineTypes = append(machineTypes, machineType.Name)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	response := make(map[string]pkgCluster.MachineType)
	response[zone] = machineTypes
	return response, nil
}

// getComputeService create a Compute Service from GKECluster
func (c *GKECluster) getComputeService() (*gkeCompute.Service, error) {

	//New client from credentials
	client, err := c.newClientFromCredentials()
	if err != nil {
		return nil, err
	}
	service, err := gkeCompute.New(client)
	if err != nil {
		return nil, err
	}
	return service, nil
}

// newClientFromCredentials creates new client from credentials
func (c *GKECluster) newClientFromCredentials() (*http.Client, error) {
	// Get Secret from Vault
	clusterSecret, err := c.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}

	// TODO https://github.com/mitchellh/mapstructure

	credentials := verify.CreateServiceAccount(clusterSecret.Values)
	return verify.CreateOath2Client(credentials)
}

// GetZones lists all supported zones
func GetZones(orgId uint, secretId string) ([]string, error) {
	g := &GKECluster{
		modelCluster: &model.ClusterModel{
			OrganizationId: orgId,
			SecretId:       secretId,
			Cloud:          pkgCluster.Google,
		},
	}
	return g.GetZones()
}

// GetZones lists supported zones
func (c *GKECluster) GetZones() ([]string, error) {

	computeService, err := c.getComputeService()
	if err != nil {
		return nil, err
	}

	project, err := c.getProjectId()
	if err != nil {
		return nil, err
	}
	var zones []string
	req := computeService.Zones.List(project)
	if err := req.Pages(context.Background(), func(page *gkeCompute.ZoneList) error {
		for _, zone := range page.Items {
			zones = append(zones, zone.Name)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return zones, nil
}

// getProjectId returns with project id from secret
func (c *GKECluster) getProjectId() (string, error) {
	s, err := c.GetSecretWithValidation()
	if err != nil {
		return "", err
	}

	return s.GetValue(pkgSecret.ProjectId), nil
}

// UpdateStatus updates cluster status in database
func (c *GKECluster) UpdateStatus(status, statusMessage string) error {
	return c.modelCluster.UpdateStatus(status, statusMessage)
}

// GetClusterDetails gets cluster details from cloud
func (c *GKECluster) GetClusterDetails() (*pkgCluster.DetailsResponse, error) {
	log.Info("Get Google Service Client")
	svc, err := c.getGoogleServiceClient()
	if err != nil {
		be := getBanzaiErrorFromError(err)
		return nil, errors.New(be.Message)
	}
	log.Info("Get Google Service Client success")

	secretItem, err := c.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}

	log.Infof("Get gke cluster with name %s", c.modelCluster.Name)
	cl, err := svc.Projects.Zones.Clusters.Get(secretItem.GetValue(pkgSecret.ProjectId), c.modelCluster.Location, c.modelCluster.Name).Context(context.Background()).Do()
	if err != nil {
		apiError := getBanzaiErrorFromError(err)
		return nil, errors.New(apiError.Message)
	}
	log.Info("Get cluster success")
	log.Infof("Cluster status is %s", cl.Status)

	if statusRunning == cl.Status {

		//userId, userName := GetUserIdAndName(c.modelCluster)
		nodePools := make(map[string]*pkgCluster.NodeDetails)

		for _, np := range c.modelCluster.GKE.NodePools {
			if np != nil {

				nodePools[np.Name] = &pkgCluster.NodeDetails{
					CreatorBaseFields: *NewCreatorBaseFields(np.CreatedAt, np.CreatedBy),
					Version:           c.modelCluster.GKE.NodeVersion,
					Count:             np.NodeCount,
					MinCount:          np.NodeMinCount,
					MaxCount:          np.NodeMaxCount,
				}
			}
		}

		response := &pkgCluster.DetailsResponse{
			CreatorBaseFields: *NewCreatorBaseFields(c.modelCluster.CreatedAt, c.modelCluster.CreatedBy),
			Name:              c.modelCluster.Name,
			Id:                c.modelCluster.ID,
			Location:          c.modelCluster.Location,
			MasterVersion:     c.modelCluster.GKE.MasterVersion,
			NodePools:         nodePools,
			Region:            c.modelCluster.GKE.Region,
			Status:            c.modelCluster.Status,
		}
		return response, nil
	}
	return nil, pkgErrors.ErrorClusterNotReady
}

// ValidateCreationFields validates all field
func (c *GKECluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {

	location := r.Location

	// Validate location
	log.Info("Validate location")
	if err := c.validateLocation(location); err != nil {
		return err
	}
	log.Info("Validate location passed")

	// Validate machine types
	nodePools := r.Properties.CreateClusterGKE.NodePools
	log.Info("Validate nodePools")
	if err := c.validateMachineType(nodePools, location); err != nil {
		return err
	}
	log.Info("Validate nodePools passed")

	// Validate kubernetes version
	log.Info("Validate kubernetesVersion")
	masterVersion := r.Properties.CreateClusterGKE.Master.Version
	nodeVersion := r.Properties.CreateClusterGKE.NodeVersion
	if err := c.validateKubernetesVersion(masterVersion, nodeVersion, location); err != nil {
		return err
	}
	log.Info("Validate kubernetesVersion passed")

	return nil
}

// validateLocation validates location
func (c *GKECluster) validateLocation(location string) error {
	log.Infof("Location: %s", location)
	validLocations, err := c.GetZones()
	if err != nil {
		return err
	}

	log.Infof("Valid locations: %v", validLocations)

	if isOk := utils.Contains(validLocations, location); !isOk {
		return pkgErrors.ErrorNotValidLocation
	}

	return nil
}

// validateMachineType validates nodeInstanceTypes
func (c *GKECluster) validateMachineType(nodePools map[string]*pkgClusterGoogle.NodePool, location string) error {

	var machineTypes []string
	for _, nodePool := range nodePools {
		if nodePool != nil {
			machineTypes = append(machineTypes, nodePool.NodeInstanceType)
		}
	}

	log.Infof("NodeInstanceTypes: %v", machineTypes)

	validMachineTypes, err := c.GetAllMachineTypesByZone(location)
	if err != nil {
		return err
	}
	log.Infof("Valid NodeInstanceTypes: %v", validMachineTypes[location])

	for _, mt := range machineTypes {
		if isOk := utils.Contains(validMachineTypes[location], mt); !isOk {
			return pkgErrors.ErrorNotValidNodeInstanceType
		}
	}

	return nil
}

// validateKubernetesVersion validates k8s versions
func (c *GKECluster) validateKubernetesVersion(masterVersion, nodeVersion, location string) error {

	log.Infof("Master version: %s", masterVersion)
	log.Infof("Node version: %s", nodeVersion)
	config, err := c.GetGkeServerConfig(location)
	if err != nil {
		return err
	}

	validNodeVersions := config.ValidNodeVersions
	log.Infof("Valid node versions: %s", validNodeVersions)

	if isOk := utils.Contains(validNodeVersions, nodeVersion); !isOk {
		return pkgErrors.ErrorNotValidNodeVersion
	}

	validMasterVersions := config.ValidMasterVersions
	log.Infof("Valid master versions: %s", validMasterVersions)

	if isOk := utils.Contains(validMasterVersions, masterVersion); !isOk {
		return pkgErrors.ErrorNotValidMasterVersion
	}

	return nil

}

// GetSecretWithValidation returns secret from vault
func (c *GKECluster) GetSecretWithValidation() (*secret.SecretItemResponse, error) {
	return c.CommonClusterBase.getSecret(c)
}

// SaveConfigSecretId saves the config secret id in database
func (c *GKECluster) SaveConfigSecretId(configSecretId string) error {
	return c.modelCluster.UpdateConfigSecret(configSecretId)
}

// GetConfigSecretId return config secret id
func (c *GKECluster) GetConfigSecretId() string {
	return c.modelCluster.ConfigSecretId
}

// GetK8sConfig returns the Kubernetes config
func (c *GKECluster) GetK8sConfig() ([]byte, error) {
	return c.CommonClusterBase.getConfig(c)
}

func waitForOperation(getter operationInfoer, operationName string) error {

	log := log.WithFields(logrus.Fields{"operation": operationName})

	log.Info("start checking operation status")

	var operationType string
	var err error
	operationStatus := statusRunning
	for operationStatus != statusDone {

		operationStatus, operationType, err = getter.getInfo(operationName)
		if err != nil {
			return err
		}

		log.Infof("Operation[%s] status: %s", operationType, operationStatus)
		time.Sleep(time.Second * 5)
	}

	return nil
}

// ListNodeNames returns node names to label them
func (c *GKECluster) ListNodeNames() (nodeNames pkgCommon.NodeNames, err error) {
	// nodes are labeled in create request
	return
}

// RbacEnabled returns true if rbac enabled on the cluster
func (c *GKECluster) RbacEnabled() bool {
	return c.modelCluster.RbacEnabled
}

// NeedAdminRights returns true if rbac is enabled and need to create a cluster role binding to user
func (c *GKECluster) NeedAdminRights() bool {
	return false
}

// GetKubernetesUserName returns the user ID which needed to create a cluster role binding which gives admin rights to the user
func (c *GKECluster) GetKubernetesUserName() (string, error) {
	return "", nil
}
