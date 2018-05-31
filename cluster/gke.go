package cluster

import (
	"encoding/base64"
	"fmt"
	"github.com/banzaicloud/banzai-types/components"
	bGoogle "github.com/banzaicloud/banzai-types/components/google"
	"github.com/banzaicloud/banzai-types/constants"
	pipConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/gin-gonic/gin/json"
	"github.com/go-errors/errors"
	"github.com/jinzhu/copier"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	gkeCompute "google.golang.org/api/compute/v1"
	gke "google.golang.org/api/container/v1"
	"google.golang.org/api/googleapi"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	statusRunning = "RUNNING"
)

const (
	defaultNamespace = "default"
	clusterAdmin     = "cluster-admin"
	netesDefault     = "netes-default"
)

// ServiceAccount describes a GKE service account
type ServiceAccount struct {
	Type                   string `json:"type"`
	ProjectId              string `json:"project_id"`
	PrivateKeyId           string `json:"private_key_id"`
	PrivateKey             string `json:"private_key"`
	ClientEmail            string `json:"client_email"`
	ClientId               string `json:"client_id"`
	AuthUri                string `json:"auth_uri"`
	TokenUri               string `json:"token_uri"`
	AuthProviderX50CertUrl string `json:"auth_provider_x509_cert_url"`
	ClientX509CertUrl      string `json:"client_x509_cert_url"`
}

//CreateGKEClusterFromRequest creates ClusterModel struct from the request
func CreateGKEClusterFromRequest(request *components.CreateClusterRequest, orgId uint) (*GKECluster, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagCreateCluster})
	log.Debug("Create ClusterModel struct from the request")
	var cluster GKECluster

	nodePools, err := createNodePoolsModelFromRequestData(request.Properties.CreateClusterGoogle.NodePools)

	if err != nil {
		return nil, err
	}

	cluster.modelCluster = &model.ClusterModel{
		Name:           request.Name,
		Location:       request.Location,
		Cloud:          request.Cloud,
		OrganizationId: orgId,
		SecretId:       request.SecretId,
		Google: model.GoogleClusterModel{
			MasterVersion: request.Properties.CreateClusterGoogle.Master.Version,
			NodeVersion:   request.Properties.CreateClusterGoogle.NodeVersion,
			NodePools:     nodePools,
		},
	}
	return &cluster, nil
}

//createNodePoolsModelFromRequestData creates an array of GoogleNodePoolModel from the nodePoolsData received through create/update requests
func createNodePoolsModelFromRequestData(nodePoolsData map[string]*bGoogle.NodePool) ([]*model.GoogleNodePoolModel, error) {

	nodePoolsCount := len(nodePoolsData)
	if nodePoolsCount == 0 {
		return nil, constants.ErrorNodePoolNotProvided
	}
	nodePoolsModel := make([]*model.GoogleNodePoolModel, nodePoolsCount)

	i := 0
	for nodePoolName, nodePoolData := range nodePoolsData {
		nodePoolsModel[i] = &model.GoogleNodePoolModel{
			Name:             nodePoolName,
			Autoscaling:      nodePoolData.Autoscaling,
			NodeMinCount:     nodePoolData.MinCount,
			NodeMaxCount:     nodePoolData.MaxCount,
			NodeCount:        nodePoolData.Count,
			NodeInstanceType: nodePoolData.NodeInstanceType,
			ServiceAccount:   nodePoolData.ServiceAccount,
		}
		i++
	}

	return nodePoolsModel, nil
}

//GKECluster struct for GKE cluster
type GKECluster struct {
	googleCluster *gke.Cluster //Don't use this directly
	modelCluster  *model.ClusterModel
	k8sConfig     []byte
	APIEndpoint   string
	commonSecret
}

// GetOrganizationId gets org where the cluster belongs
func (g *GKECluster) GetOrganizationId() uint {
	return g.modelCluster.OrganizationId
}

// GetSecretID retrieves the secret id
func (g *GKECluster) GetSecretID() string {
	return g.modelCluster.SecretId
}

// GetGoogleCluster returns with a Cluster from GKE
func (g *GKECluster) GetGoogleCluster() (*gke.Cluster, error) {
	if g.googleCluster != nil {
		return g.googleCluster, nil
	}
	svc, err := g.getGoogleServiceClient()
	if err != nil {
		return nil, err
	}

	secretItem, err := g.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}

	cc := googleCluster{
		Name:      g.modelCluster.Name,
		ProjectID: secretItem.GetValue(secret.ProjectId),
		Zone:      g.modelCluster.Location,
	}
	cluster, err := getClusterGoogle(svc, cc)
	if err != nil {
		return nil, err
	}
	g.googleCluster = cluster
	return g.googleCluster, nil
}

//GetAPIEndpoint returns the Kubernetes Api endpoint
func (g *GKECluster) GetAPIEndpoint() (string, error) {
	if g.APIEndpoint != "" {
		return g.APIEndpoint, nil
	}
	cluster, err := g.GetGoogleCluster()
	if err != nil {
		return "", err
	}
	g.APIEndpoint = cluster.Endpoint
	return g.APIEndpoint, nil
}

//CreateCluster creates a new cluster
func (g *GKECluster) CreateCluster() error {

	log := logger.WithFields(logrus.Fields{"action": constants.TagCreateCluster})

	log.Info("Start create cluster (Google)")

	log.Info("Get Google Service Client")
	svc, err := g.getGoogleServiceClient()
	if err != nil {
		return err
	}

	log.Info("Get Google Service Client succeeded")

	nodePools, err := createNodePoolsFromClusterModel(&g.modelCluster.Google)
	if err != nil {
		return err
	}

	secretItem, err := g.GetSecretWithValidation()
	if err != nil {
		return err
	}

	cc := googleCluster{
		ProjectID:     secretItem.GetValue(secret.ProjectId),
		Zone:          g.modelCluster.Location,
		Name:          g.modelCluster.Name,
		MasterVersion: g.modelCluster.Google.MasterVersion,
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
	log.Infof("Cluster %s create is called for project %s and zone %s. Status Code %v", cc.Name, cc.ProjectID, cc.Zone, createCall.HTTPStatusCode)

	log.Info("Waiting for cluster...")
	gkeCluster, err := waitForCluster(svc, cc)
	if err != nil {
		be := getBanzaiErrorFromError(err)
		// TODO status code !?
		return errors.New(be.Message)
	}

	g.googleCluster = gkeCluster

	return nil

}

//Persist save the cluster model
func (g *GKECluster) Persist(status, statusMessage string) error {
	log.Infof("Model before save: %v", g.modelCluster)
	return g.modelCluster.UpdateStatus(status, statusMessage)
}

//GetK8sConfig returns the Kubernetes config
func (g *GKECluster) GetK8sConfig() ([]byte, error) {

	if g.k8sConfig != nil {
		return g.k8sConfig, nil
	}
	log := logger.WithFields(logrus.Fields{"action": constants.TagFetchClusterConfig})

	config, err := g.getGoogleKubernetesConfig()
	if err != nil {
		// something went wrong
		be := getBanzaiErrorFromError(err)
		// TODO status code !?
		return nil, errors.New(be.Message)
	}
	// get config succeeded
	log.Info("Get k8s config succeeded")

	g.k8sConfig = config

	return config, nil

}

//GetName returns the name of the cluster
func (g *GKECluster) GetName() string {
	return g.modelCluster.Name
}

//GetType returns the cloud type of the cluster
func (g *GKECluster) GetType() string {
	return g.modelCluster.Cloud
}

//GetStatus gets cluster status
func (g *GKECluster) GetStatus() (*components.GetClusterStatusResponse, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagGetClusterStatus})
	log.Info("Create cluster status response")

	nodePools := make(map[string]*components.NodePoolStatus)
	for _, np := range g.modelCluster.Google.NodePools {
		if np != nil {
			nodePools[np.Name] = &components.NodePoolStatus{
				Count:          np.NodeCount,
				InstanceType:   np.NodeInstanceType,
				ServiceAccount: np.ServiceAccount,
			}
		}
	}

	return &components.GetClusterStatusResponse{
		Status:        g.modelCluster.Status,
		StatusMessage: g.modelCluster.StatusMessage,
		Name:          g.modelCluster.Name,
		Location:      g.modelCluster.Location,
		Cloud:         g.modelCluster.Cloud,
		ResourceID:    g.modelCluster.ID,
		NodePools:     nodePools,
	}, nil
}

// DeleteCluster deletes cluster from google
func (g *GKECluster) DeleteCluster() error {

	log := logger.WithFields(logrus.Fields{"action": constants.TagDeleteCluster})

	log.Info("Start delete google cluster")

	if g == nil {
		return constants.ErrorNilCluster
	}

	secretItem, err := g.GetSecretWithValidation()
	if err != nil {
		return err
	}

	gkec := googleCluster{
		ProjectID: secretItem.GetValue(secret.ProjectId),
		Name:      g.modelCluster.Name,
		Zone:      g.modelCluster.Location,
	}

	if err := g.callDeleteCluster(&gkec); err != nil {
		be := getBanzaiErrorFromError(err)
		// TODO status code !?
		return errors.New(be.Message)
	}
	log.Info("Delete succeeded")
	return nil

}

// UpdateCluster updates GKE cluster in cloud
func (g *GKECluster) UpdateCluster(updateRequest *components.UpdateClusterRequest) error {

	log := logger.WithFields(logrus.Fields{"action": constants.TagUpdateCluster})
	log.Info("Start updating cluster (google)")

	svc, err := g.getGoogleServiceClient()
	if err != nil {
		return err
	}

	updateNodePoolsModel, err := createNodePoolsModelFromRequestData(updateRequest.Google.NodePools)
	if err != nil {
		return err
	}

	googleClusterModel := model.GoogleClusterModel{}

	copier.Copy(&googleClusterModel, &g.modelCluster.Google)
	googleClusterModel.NodePools = updateNodePoolsModel

	updatedNodePools, err := createNodePoolsFromClusterModel(&googleClusterModel)
	if err != nil {
		return err
	}

	secretItem, err := g.GetSecretWithValidation()
	if err != nil {
		return err
	}

	cc := googleCluster{
		Name:          g.modelCluster.Name,
		ProjectID:     secretItem.GetValue(secret.ProjectId),
		Zone:          g.modelCluster.Location,
		MasterVersion: updateRequest.Google.Master.Version,
		NodePools:     updatedNodePools,
	}

	res, err := callUpdateClusterGoogle(svc, cc)
	if err != nil {
		be := getBanzaiErrorFromError(err)
		// TODO status code !?
		return errors.New(be.Message)
	}
	log.Info("Cluster update succeeded")
	g.googleCluster = res

	// update model to save
	g.updateModel(res, updatedNodePools)

	return nil

}

func (g *GKECluster) updateModel(c *gke.Cluster, updatedNodePools []*gke.NodePool) {
	// Update the model from the cluster data read back from Google
	g.modelCluster.Google.MasterVersion = c.CurrentMasterVersion
	g.modelCluster.Google.NodeVersion = c.CurrentNodeVersion

	var newNodePoolsModels []*model.GoogleNodePoolModel
	for _, clusterNodePool := range c.NodePools {
		updated := false

		for _, nodePoolModel := range g.modelCluster.Google.NodePools {
			if clusterNodePool.Name == nodePoolModel.Name {
				nodePoolModel.ServiceAccount = clusterNodePool.Config.ServiceAccount
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
			nodePoolModelAdd := &model.GoogleNodePoolModel{
				Name:             clusterNodePool.Name,
				ServiceAccount:   clusterNodePool.Config.ServiceAccount,
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
		g.modelCluster.Google.NodePools = append(g.modelCluster.Google.NodePools, newNodePoolModel)
	}

	// mark for deletion the node pool model entries that has no corresponding node pool in the cluster
	for _, nodePoolModel := range g.modelCluster.Google.NodePools {
		found := false

		for _, clusterNodePool := range c.NodePools {
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
func (g *GKECluster) GetID() uint {
	return g.modelCluster.ID
}

//GetModel returns the whole clusterModel
func (g *GKECluster) GetModel() *model.ClusterModel {
	return g.modelCluster
}

func (g *GKECluster) getGoogleServiceClient() (*gke.Service, error) {

	client, err := g.newClientFromCredentials()
	if err != nil {
		return nil, err
	}

	//New client from credentials
	service, err := gke.New(client)
	if err != nil {
		return nil, err
	}
	return service, nil
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

//createNodePoolsFromClusterModel creates an array of google NodePool from the given cluster model
func createNodePoolsFromClusterModel(clusterModel *model.GoogleClusterModel) ([]*gke.NodePool, error) {
	nodePoolsCount := len(clusterModel.NodePools)
	if nodePoolsCount == 0 {
		return nil, constants.ErrorNodePoolNotProvided
	}

	nodePools := make([]*gke.NodePool, nodePoolsCount)

	for i := 0; i < nodePoolsCount; i++ {
		nodePoolModel := clusterModel.NodePools[i]

		nodePools[i] = &gke.NodePool{
			Name: nodePoolModel.Name,
			Config: &gke.NodeConfig{
				MachineType:    nodePoolModel.NodeInstanceType,
				ServiceAccount: nodePoolModel.ServiceAccount,
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
func createNodePoolsRequestDataFromNodePoolModel(nodePoolsModel []*model.GoogleNodePoolModel) (map[string]*bGoogle.NodePool, error) {
	nodePoolsCount := len(nodePoolsModel)
	if nodePoolsCount == 0 {
		return nil, constants.ErrorNodePoolNotProvided
	}

	nodePools := make(map[string]*bGoogle.NodePool)

	for i := 0; i < nodePoolsCount; i++ {
		nodePoolModel := nodePoolsModel[i]
		nodePools[nodePoolModel.Name] = &bGoogle.NodePool{
			Autoscaling:      nodePoolModel.Autoscaling,
			MinCount:         nodePoolModel.NodeMinCount,
			MaxCount:         nodePoolModel.NodeMaxCount,
			Count:            nodePoolModel.NodeCount,
			NodeInstanceType: nodePoolModel.NodeInstanceType,
			ServiceAccount:   nodePoolModel.ServiceAccount,
		}
	}

	return nodePools, nil
}

func getBanzaiErrorFromError(err error) *components.BanzaiResponse {

	if err == nil {
		// error is nil
		return &components.BanzaiResponse{
			StatusCode: http.StatusInternalServerError,
		}
	}

	googleErr, ok := err.(*googleapi.Error)
	if ok {
		// error is googleapi error
		return &components.BanzaiResponse{
			StatusCode: googleErr.Code,
			Message:    googleErr.Message,
		}
	}

	// default
	return &components.BanzaiResponse{
		StatusCode: http.StatusInternalServerError,
		Message:    err.Error(),
	}
}
func waitForCluster(svc *gke.Service, cc googleCluster) (*gke.Cluster, error) {

	var message string
	for {

		cluster, err := getClusterGoogle(svc, cc)
		if err != nil {
			return nil, err
		}

		log.Infof("Cluster status: %s", cluster.Status)

		if cluster.Status == statusRunning {
			return cluster, nil
		}

		if cluster.Status != message {
			log.Infof("%s cluster %s", string(cluster.Status), cc.Name)
			message = cluster.Status
		}

		time.Sleep(time.Second * 5)

	}
}

func getClusterGoogle(svc *gke.Service, cc googleCluster) (*gke.Cluster, error) {
	return svc.Projects.Zones.Clusters.Get(cc.ProjectID, cc.Zone, cc.Name).Context(context.TODO()).Do()
}

func (g *GKECluster) callDeleteCluster(cc *googleCluster) error {
	svc, err := g.getGoogleServiceClient()
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

func callUpdateClusterGoogle(svc *gke.Service, cc googleCluster) (*gke.Cluster, error) {

	var updatedCluster *gke.Cluster

	log.Infof("Updating cluster: %#v", cc)

	cluster, err := getClusterGoogle(svc, cc)
	if err != nil {
		return nil, err
	}

	if cc.MasterVersion != "" && cc.MasterVersion != cluster.CurrentMasterVersion {
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
		if updatedCluster, err = waitForCluster(svc, cc); err != nil {
			return nil, err
		}
	}

	// Collect node pools that have to be deleted and delete them before
	// resizing exiting ones or creating new ones to minimize tha chance
	// of hitting quota limits
	var nodePoolsToDelete []string
	for _, currentClusterNodePool := range cluster.NodePools {
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
		for i = 0; i < len(cluster.NodePools); i++ {
			if nodePoolFromUpdReq.Name == cluster.NodePools[i].Name {
				break
			}
		}
		if i == len(cluster.NodePools) {
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
		if updatedCluster, err = waitForCluster(svc, cc); err != nil {
			return nil, err
		}
	}

	// Update node pools
	for _, nodePool := range cc.NodePools {
		for i := 0; i < len(cluster.NodePools); i++ {
			if cluster.NodePools[i].Name == nodePool.Name {

				if nodePool.Version != "" && nodePool.Version != cluster.NodePools[i].Version {
					log.Infof("Updating node pool %s to %v version", nodePool.Name, nodePool.Version)
					updateCall, err := svc.Projects.Zones.Clusters.NodePools.Update(cc.ProjectID, cc.Zone, cc.Name, nodePool.Name, &gke.UpdateNodePoolRequest{
						NodeVersion: nodePool.Version,
					}).Context(context.Background()).Do()
					if err != nil {
						return nil, err
					}
					log.Infof("Node pool %s update is called for project %s, zone %s and cluster %s. Status Code %v", nodePool.Name, cc.ProjectID, cc.Zone, cc.Name, updateCall.HTTPStatusCode)
					if err := waitForNodePool(svc, &cc, nodePool.Name); err != nil {
						return nil, err
					}
				}

				if autoscalingHasBeenUpdated(nodePool, cluster.NodePools[i]) {
					var updateCall *gke.Operation
					var err error
					if nodePool.Autoscaling.Enabled {
						log.Infof("Updating node pool %s enable Autoscaling", nodePool.Name)
						updateCall, err = svc.Projects.Zones.Clusters.NodePools.Autoscaling(cc.ProjectID, cc.Zone, cc.Name, nodePool.Name, &gke.SetNodePoolAutoscalingRequest{
							Autoscaling: &gke.NodePoolAutoscaling{
								Enabled:      true,
								MinNodeCount: nodePool.Autoscaling.MinNodeCount,
								MaxNodeCount: nodePool.Autoscaling.MaxNodeCount,
							},
						}).Context(context.Background()).Do()
					} else {
						log.Infof("Updating node pool %s disable Autoscaling", nodePool.Name)
						updateCall, err = svc.Projects.Zones.Clusters.NodePools.Autoscaling(cc.ProjectID, cc.Zone, cc.Name, nodePool.Name, &gke.SetNodePoolAutoscalingRequest{
							Autoscaling: &gke.NodePoolAutoscaling{
								Enabled: false,
							},
						}).Context(context.Background()).Do()
					}

					if err != nil {
						return nil, err
					}
					log.Infof("Node pool %s update is called for project %s, zone %s and cluster %s. Status Code %v", nodePool.Name, cc.ProjectID, cc.Zone, cc.Name, updateCall.HTTPStatusCode)
					if updatedCluster, err = waitForCluster(svc, cc); err != nil {
						return nil, err
					}
				}

				if nodePool.InitialNodeCount > 0 {
					log.Infof("Updating node size to %v for node pool %s", nodePool.InitialNodeCount, nodePool.Name)
					updateCall, err := svc.Projects.Zones.Clusters.NodePools.SetSize(cc.ProjectID, cc.Zone, cc.Name, nodePool.Name, &gke.SetNodePoolSizeRequest{
						NodeCount: nodePool.InitialNodeCount,
					}).Context(context.Background()).Do()
					if err != nil {
						return nil, err
					}
					log.Infof("Node pool %s size change is called for project %s, zone %s and cluster %s. Status Code %v", nodePool.Name, cc.ProjectID, cc.Zone, cc.Name, updateCall.HTTPStatusCode)
					if updatedCluster, err = waitForCluster(svc, cc); err != nil {
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
		if updatedCluster, err = waitForCluster(svc, cc); err != nil {
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

func waitForNodePool(svc *gke.Service, cc *googleCluster, nodePoolName string) error {
	var message string
	for {
		nodepool, err := svc.Projects.Zones.Clusters.NodePools.Get(cc.ProjectID, cc.Zone, cc.Name, nodePoolName).Context(context.TODO()).Do()
		if err != nil {
			return err
		}
		if nodepool.Status == statusRunning {
			log.Infof("NodePool %v is running", nodePoolName)
			return nil
		}
		if nodepool.Status != message {
			log.Infof("%v NodePool %v", string(nodepool.Status), nodePoolName)
			message = nodepool.Status
		}
		time.Sleep(time.Second * 5)
	}
}

func (g *GKECluster) getGoogleKubernetesConfig() ([]byte, error) {

	log.Info("Get Google Service Client")
	svc, err := g.getGoogleServiceClient()
	if err != nil {
		return nil, err
	}
	log.Info("Get Google Service Client succeeded")

	secretItem, err := g.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}

	log.Infof("Get google cluster with name %s", g.modelCluster.Name)
	cl, err := getClusterGoogle(svc, googleCluster{
		Name:      g.modelCluster.Name,
		ProjectID: secretItem.GetValue(secret.ProjectId),
		Zone:      g.modelCluster.Location,
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
	config, err := storeConfig(&finalCl, g.modelCluster.Name)
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
	log := logger.WithFields(logrus.Fields{"action": constants.TagGetCluster})
	log.Debug("Create ClusterModel struct from the request")
	gkeCluster := GKECluster{
		modelCluster: clusterModel,
	}
	return &gkeCluster, nil
}

//AddDefaultsToUpdate adds defaults to update request
func (g *GKECluster) AddDefaultsToUpdate(r *components.UpdateClusterRequest) {

	log := logger.WithFields(logrus.Fields{"action": "AddDefaultsToUpdate"})

	// TODO: error handling
	defGooglePools, _ := createNodePoolsFromClusterModel(&g.modelCluster.Google)

	defGoogleMaster := &bGoogle.Master{
		Version: g.modelCluster.Google.MasterVersion,
	}

	// ---- [ Node check ] ---- //
	if r.Google.NodePools == nil {
		log.Warn("'nodePools' field is empty. Load it from stored data.")

		r.Google.NodePools = make(map[string]*bGoogle.NodePool)
		for _, nodePool := range defGooglePools {
			r.Google.NodePools[nodePool.Name] = &bGoogle.NodePool{
				Count:            int(nodePool.InitialNodeCount),
				NodeInstanceType: nodePool.Config.MachineType,
				ServiceAccount:   nodePool.Config.ServiceAccount,
			}
			if nodePool.Autoscaling != nil {
				r.Google.NodePools[nodePool.Name].Autoscaling = nodePool.Autoscaling.Enabled
				r.Google.NodePools[nodePool.Name].MinCount = int(nodePool.Autoscaling.MinNodeCount)
				r.Google.NodePools[nodePool.Name].MaxCount = int(nodePool.Autoscaling.MaxNodeCount)
			}
		}
	}
	// ---- [ Master check ] ---- //
	if r.Google.Master == nil {
		log.Warn("'master' field is empty. Load it from stored data.")
		r.Google.Master = defGoogleMaster
	}

	// ---- [ NodeCount check] ---- //
	for name, nodePoolData := range r.Google.NodePools {
		if nodePoolData.Count == 0 {
			// initialize with count read from db
			var i int
			for i = 0; i < len(g.modelCluster.Google.NodePools); i++ {
				if g.modelCluster.Google.NodePools[i].Name == name {
					nodePoolData.Count = g.modelCluster.Google.NodePools[i].NodeCount
					log.Warnf("Node count for node pool %s initiated from database to value: %d", name, nodePoolData.Count)
					break
				}
			}
			if i == len(g.modelCluster.Google.NodePools) {
				// node pool not found in db; set count to default value
				nodePoolData.Count = constants.DefaultNodeMinCount
				log.Warnf("Node count for node pool %s set to default value: ", name, nodePoolData.Count)
			}
		}
	}

	// ---- [ Node Version check] ---- //
	if len(r.Google.NodeVersion) == 0 {
		nodeVersion := g.modelCluster.Google.NodeVersion
		log.Warnf("Node K8s version: %s", nodeVersion)
		r.Google.NodeVersion = nodeVersion
	}

	// ---- [ Master Version check] ---- //
	if len(r.Google.Master.Version) == 0 {
		masterVersion := g.modelCluster.Google.MasterVersion
		log.Warnf("Master K8s version: %s", masterVersion)
		r.Google.Master.Version = masterVersion
	}

}

//CheckEqualityToUpdate validates the update request
func (g *GKECluster) CheckEqualityToUpdate(r *components.UpdateClusterRequest) error {

	log := logger.WithFields(logrus.Fields{"action": "CheckEqualityToUpdate"})

	// create update request struct with the stored data to check equality
	nodePools, _ := createNodePoolsRequestDataFromNodePoolModel(g.modelCluster.Google.NodePools)
	preCl := &bGoogle.UpdateClusterGoogle{
		Master: &bGoogle.Master{
			Version: g.modelCluster.Google.MasterVersion,
		},
		NodeVersion: g.modelCluster.Google.NodeVersion,
		NodePools:   nodePools,
	}

	log.Info("Check stored & updated cluster equals")

	// check equality
	return utils.IsDifferent(r.Google, preCl)
}

//DeleteFromDatabase deletes model from the database
func (g *GKECluster) DeleteFromDatabase() error {
	err := g.modelCluster.Delete()
	if err != nil {
		return err
	}
	g.modelCluster = nil
	return nil
}

// GetGkeServerConfig returns all supported K8S versions
func GetGkeServerConfig(orgId uint, secretId, zone string) (*gke.ServerConfig, error) {
	g := GKECluster{
		modelCluster: &model.ClusterModel{
			OrganizationId: orgId,
			SecretId:       secretId,
			Cloud:          constants.Google,
		},
	}
	return g.GetGkeServerConfig(zone)
}

// GetGkeServerConfig returns configuration info about the Kubernetes Engine service.
func (g *GKECluster) GetGkeServerConfig(zone string) (*gke.ServerConfig, error) {

	log := logger.WithFields(logrus.Fields{"action": "GetGkeServerConfig"})

	log.Info("Start getting configuration info")

	log.Info("Get Google service client")
	svc, err := g.getGoogleServiceClient()
	if err != nil {
		return nil, err
	}

	projectId, err := g.getProjectId()
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
func GetAllMachineTypesByZone(orgId uint, secretId, zone string) (map[string]components.MachineType, error) {
	g := &GKECluster{
		modelCluster: &model.ClusterModel{
			OrganizationId: orgId,
			SecretId:       secretId,
			Cloud:          constants.Google,
		},
	}
	return g.GetAllMachineTypesByZone(zone)
}

// GetAllMachineTypesByZone lists supported machine types by zone
func (g *GKECluster) GetAllMachineTypesByZone(zone string) (map[string]components.MachineType, error) {

	computeService, err := g.getComputeService()
	if err != nil {
		return nil, err
	}

	project, err := g.getProjectId()
	if err != nil {
		return nil, err
	}

	return getMachineTypes(computeService, project, zone)
}

// GetAllMachineTypes returns all supported machine types
func GetAllMachineTypes(orgId uint, secretId string) (map[string]components.MachineType, error) {
	g := &GKECluster{
		modelCluster: &model.ClusterModel{
			OrganizationId: orgId,
			SecretId:       secretId,
			Cloud:          constants.Google,
		},
	}

	return g.GetAllMachineTypes()
}

// GetAllMachineTypes lists all supported machine types
func (g *GKECluster) GetAllMachineTypes() (map[string]components.MachineType, error) {

	computeService, err := g.getComputeService()
	if err != nil {
		return nil, err
	}

	project, err := g.getProjectId()
	if err != nil {
		return nil, err
	}
	return getMachineTypesWithoutZones(computeService, project)
}

// getMachineTypesWithoutZones lists supported machine types in all zone
func getMachineTypesWithoutZones(csv *gkeCompute.Service, project string) (map[string]components.MachineType, error) {
	response := make(map[string]components.MachineType)
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
func getMachineTypes(csv *gkeCompute.Service, project, zone string) (map[string]components.MachineType, error) {

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

	response := make(map[string]components.MachineType)
	response[zone] = machineTypes
	return response, nil
}

// getComputeService create a Compute Service from GKECluster
func (g *GKECluster) getComputeService() (*gkeCompute.Service, error) {

	//New client from credentials
	client, err := g.newClientFromCredentials()
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
func (g *GKECluster) newClientFromCredentials() (*http.Client, error) {
	// Get Secret from Vault
	clusterSecret, err := g.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}

	// TODO https://github.com/mitchellh/mapstructure

	credentials := ServiceAccount{
		Type:                   clusterSecret.Values[secret.Type],
		ProjectId:              clusterSecret.Values[secret.ProjectId],
		PrivateKeyId:           clusterSecret.Values[secret.PrivateKeyId],
		PrivateKey:             clusterSecret.Values[secret.PrivateKey],
		ClientEmail:            clusterSecret.Values[secret.ClientEmail],
		ClientId:               clusterSecret.Values[secret.ClientId],
		AuthUri:                clusterSecret.Values[secret.AuthUri],
		TokenUri:               clusterSecret.Values[secret.TokenUri],
		AuthProviderX50CertUrl: clusterSecret.Values[secret.AuthX509Url],
		ClientX509CertUrl:      clusterSecret.Values[secret.ClientX509Url],
	}
	jsonConfig, err := json.Marshal(credentials)
	if err != nil {
		return nil, err
	}

	// Parse credentials from JSON
	config, err := google.JWTConfigFromJSON(jsonConfig, gke.CloudPlatformScope)
	if err != nil {
		return nil, err
	}

	// Create oauth2 client with credential
	return config.Client(context.TODO()), nil
}

// GetZones lists all supported zones
func GetZones(orgId uint, secretId string) ([]string, error) {
	g := &GKECluster{
		modelCluster: &model.ClusterModel{
			OrganizationId: orgId,
			SecretId:       secretId,
			Cloud:          constants.Google,
		},
	}
	return g.GetZones()
}

// GetZones lists supported zones
func (g *GKECluster) GetZones() ([]string, error) {

	computeService, err := g.getComputeService()
	if err != nil {
		return nil, err
	}

	project, err := g.getProjectId()
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
func (g *GKECluster) getProjectId() (string, error) {
	s, err := g.GetSecretWithValidation()
	if err != nil {
		return "", err
	}

	return s.GetValue(secret.ProjectId), nil
}

// UpdateStatus updates cluster status in database
func (g *GKECluster) UpdateStatus(status, statusMessage string) error {
	return g.modelCluster.UpdateStatus(status, statusMessage)
}

// GetClusterDetails gets cluster details from cloud
func (g *GKECluster) GetClusterDetails() (*components.ClusterDetailsResponse, error) {
	log := logger.WithFields(logrus.Fields{"tag": "GetClusterDetails"})
	log.Info("Get Google Service Client")
	svc, err := g.getGoogleServiceClient()
	if err != nil {
		be := getBanzaiErrorFromError(err)
		return nil, errors.New(be.Message)
	}
	log.Info("Get Google Service Client success")

	secretItem, err := g.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}

	log.Infof("Get google cluster with name %s", g.modelCluster.Name)
	cl, err := svc.Projects.Zones.Clusters.Get(secretItem.GetValue(secret.ProjectId), g.modelCluster.Location, g.modelCluster.Name).Context(context.Background()).Do()
	if err != nil {
		apiError := getBanzaiErrorFromError(err)
		return nil, errors.New(apiError.Message)
	}
	log.Info("Get cluster success")
	log.Infof("Cluster status is %s", cl.Status)
	if statusRunning == cl.Status {
		response := &components.ClusterDetailsResponse{
			//Status:           g.modelCluster.Status,
			Name: g.modelCluster.Name,
			Id:   g.modelCluster.ID,
			//Location:         g.modelCluster.Location,
			//Cloud:            g.modelCluster.Cloud,
			//NodeInstanceType: g.modelCluster.NodeInstanceType,
			//ResourceID:       g.modelCluster.ID,
		}
		return response, nil
	}
	return nil, constants.ErrorClusterNotReady
}

// ValidateCreationFields validates all field
func (g *GKECluster) ValidateCreationFields(r *components.CreateClusterRequest) error {

	location := r.Location

	// Validate location
	log.Info("Validate location")
	if err := g.validateLocation(location); err != nil {
		return err
	}
	log.Info("Validate location passed")

	// Validate machine types
	nodePools := r.Properties.CreateClusterGoogle.NodePools
	log.Info("Validate nodePools")
	if err := g.validateMachineType(nodePools, location); err != nil {
		return err
	}
	log.Info("Validate nodePools passed")

	// Validate kubernetes version
	log.Info("Validate kubernetesVersion")
	masterVersion := r.Properties.CreateClusterGoogle.Master.Version
	nodeVersion := r.Properties.CreateClusterGoogle.NodeVersion
	if err := g.validateKubernetesVersion(masterVersion, nodeVersion, location); err != nil {
		return err
	}
	log.Info("Validate kubernetesVersion passed")

	return nil
}

// validateLocation validates location
func (g *GKECluster) validateLocation(location string) error {
	log.Infof("Location: %s", location)
	validLocations, err := g.GetZones()
	if err != nil {
		return err
	}

	log.Infof("Valid locations: %v", validLocations)

	if isOk := utils.Contains(validLocations, location); !isOk {
		return constants.ErrorNotValidLocation
	}

	return nil
}

// validateMachineType validates nodeInstanceTypes
func (g *GKECluster) validateMachineType(nodePools map[string]*bGoogle.NodePool, location string) error {

	var machineTypes []string
	for _, nodePool := range nodePools {
		if nodePool != nil {
			machineTypes = append(machineTypes, nodePool.NodeInstanceType)
		}
	}

	log.Infof("NodeInstanceTypes: %v", machineTypes)

	validMachineTypes, err := g.GetAllMachineTypesByZone(location)
	if err != nil {
		return err
	}
	log.Infof("Valid NodeInstanceTypes: %v", validMachineTypes[location])

	for _, mt := range machineTypes {
		if isOk := utils.Contains(validMachineTypes[location], mt); !isOk {
			return constants.ErrorNotValidNodeInstanceType
		}
	}

	return nil
}

// validateKubernetesVersion validates k8s versions
func (g *GKECluster) validateKubernetesVersion(masterVersion, nodeVersion, location string) error {

	log.Infof("Master version: %s", masterVersion)
	log.Infof("Node version: %s", nodeVersion)
	config, err := g.GetGkeServerConfig(location)
	if err != nil {
		return err
	}

	validNodeVersions := config.ValidNodeVersions
	log.Infof("Valid node versions: %s", validNodeVersions)

	if isOk := utils.Contains(validNodeVersions, nodeVersion); !isOk {
		return constants.ErrorNotValidNodeVersion
	}

	validMasterVersions := config.ValidMasterVersions
	log.Infof("Valid master versions: %s", validMasterVersions)

	if isOk := utils.Contains(validMasterVersions, masterVersion); !isOk {
		return constants.ErrorNotValidMasterVersion
	}

	return nil

}

// GetSecretWithValidation returns secret from vault
func (g *GKECluster) GetSecretWithValidation() (*secret.SecretsItemResponse, error) {
	return g.commonSecret.get(g)
}
