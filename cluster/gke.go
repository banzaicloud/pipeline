package cluster

import (
	"encoding/base64"
	"fmt"
	pipConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/model"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgClusterGoogle "github.com/banzaicloud/pipeline/pkg/cluster/google"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
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

// constants to find Kubernetes resources
const (
	kubernetesIO   = "kubernetes.io"
	targetPrefix   = "gke-"
	clusterNameKey = "cluster-name"
)

//CreateGKEClusterFromRequest creates ClusterModel struct from the request
func CreateGKEClusterFromRequest(request *pkgCluster.CreateClusterRequest, orgId uint) (*GKECluster, error) {
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
func createNodePoolsModelFromRequestData(nodePoolsData map[string]*pkgClusterGoogle.NodePool) ([]*model.GoogleNodePoolModel, error) {

	nodePoolsCount := len(nodePoolsData)
	if nodePoolsCount == 0 {
		return nil, pkgErrors.ErrorNodePoolNotProvided
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
	APIEndpoint   string
	CommonClusterBase
}

// GetOrganizationId gets org where the cluster belongs
func (g *GKECluster) GetOrganizationId() uint {
	return g.modelCluster.OrganizationId
}

// GetSecretId retrieves the secret id
func (g *GKECluster) GetSecretId() string {
	return g.modelCluster.SecretId
}

// GetSshSecretId retrieves the secret id
func (g *GKECluster) GetSshSecretId() string {
	return g.modelCluster.SshSecretId
}

// SaveSshSecretId saves the ssh secret id to database
func (g *GKECluster) SaveSshSecretId(sshSecretId string) error {
	return g.modelCluster.UpdateSshSecret(sshSecretId)
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
		ProjectID: secretItem.GetValue(pkgSecret.ProjectId),
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
		ProjectID:     secretItem.GetValue(pkgSecret.ProjectId),
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

// DownloadK8sConfig downloads the kubeconfig file from cloud
func (g *GKECluster) DownloadK8sConfig() ([]byte, error) {

	config, err := g.getGoogleKubernetesConfig()
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
func (g *GKECluster) GetName() string {
	return g.modelCluster.Name
}

//GetType returns the cloud type of the cluster
func (g *GKECluster) GetType() string {
	return g.modelCluster.Cloud
}

//GetStatus gets cluster status
func (g *GKECluster) GetStatus() (*pkgCluster.GetClusterStatusResponse, error) {
	log.Info("Create cluster status response")

	nodePools := make(map[string]*pkgCluster.NodePoolStatus)
	for _, np := range g.modelCluster.Google.NodePools {
		if np != nil {
			nodePools[np.Name] = &pkgCluster.NodePoolStatus{
				Count:          np.NodeCount,
				InstanceType:   np.NodeInstanceType,
				ServiceAccount: np.ServiceAccount,
			}
		}
	}

	return &pkgCluster.GetClusterStatusResponse{
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

	if err := g.waitForResourcesDelete(); err != nil {
		log.Warnf("error during wait for resources: %s", err.Error())
	}

	log.Info("Start delete google cluster")

	if g == nil {
		return pkgErrors.ErrorNilCluster
	}

	secretItem, err := g.GetSecretWithValidation()
	if err != nil {
		return err
	}

	gkec := googleCluster{
		ProjectID: secretItem.GetValue(pkgSecret.ProjectId),
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

// waitForResourcesDelete waits until the Kubernetes destroys all the resources which it had created
func (g *GKECluster) waitForResourcesDelete() error {

	log.Info("Waiting for deleting deployments resources")

	log.Info("Create compute service")
	csv, err := g.getComputeService()
	if err != nil {
		return errors.Wrap(err, "Error during creating compute service")
	}

	log.Info("Get project id")
	project, err := g.getProjectId()
	if err != nil {
		return errors.Wrap(err, "Error during getting project id")
	}

	err = checkFirewalls(csv, project, g.modelCluster.Name)
	if err != nil {
		return errors.Wrap(err, "Error during checking firewalls")
	}

	err = checkLoadBalancerResources(csv, project, g.modelCluster.Location, g.modelCluster.Name)
	if err != nil {
		return errors.Wrap(err, "Error during checking load balancer resources")
	}

	return nil
}

// checkLoadBalancerResources checks all load balancer resources deleted by Kubernetes
func checkLoadBalancerResources(csv *gkeCompute.Service, project, zone, clusterName string) error {

	log.Info("Check load balancer resources")

	log.Infof("Find region by zone[%s]", zone)
	region, err := findRegionByZone(csv, project, zone)
	if err != nil {
		return err
	}

	regionName := region.Name
	log.Infof("Region name: %s", regionName)

	targetPools, err := checkTargetPools(csv, project, zone, regionName, clusterName)
	if err != nil {
		return err
	}

	return checkForwardingRules(csv, targetPools, project, regionName)
}

// checkTargetPools checks all target pools deleted by Kubernetes
func checkTargetPools(csv *gkeCompute.Service, project, zone, regionName, clusterName string) ([]*gkeCompute.TargetPool, error) {

	log.Infof("Check target pools(backends) in project[%s] and region[%s]", project, regionName)

	log.Info("List target pools")
	pools, err := listTargetPools(csv, project, regionName)
	if err != nil {
		return nil, err
	}

	log.Info("List instances")
	instance, err := findInstanceByClusterName(csv, project, zone, clusterName)
	if err != nil {
		return nil, err
	}

	log.Infof("Find target pool(s) by instance[%s]", instance.Name)
	clusterTargetPools := findTargetPoolsByInstances(pools, instance.Name)

	for _, pool := range clusterTargetPools {
		if pool != nil {
			for {
				err := isTargetPoolDeleted(csv, project, regionName, pool.Name)
				if err == nil {
					log.Infof("Target pool[%s] deleted", pool.Name)
					break
				} else {
					log.Warn(err.Error())
					time.Sleep(time.Second * 5)
				}
			}
		}
	}

	return clusterTargetPools, nil
}

// findTargetPoolsByInstances returns all target pools which created by Kubernetes
func findTargetPoolsByInstances(pools []*gkeCompute.TargetPool, instanceName string) []*gkeCompute.TargetPool {

	var filteredPools []*gkeCompute.TargetPool
	for _, p := range pools {
		if p != nil {
			for _, i := range p.Instances {
				if i == instanceName {
					filteredPools = append(filteredPools, p)
				}
			}
		}
	}

	return filteredPools
}

// isTargetPoolDeleted checks the given target pool is deleted by Kubernetes
func isTargetPoolDeleted(csv *gkeCompute.Service, project, region, targetPoolName string) error {
	log.Infof("Get target pool[%s]", targetPoolName)
	_, err := csv.TargetPools.Get(project, region, targetPoolName).Context(context.Background()).Do()
	if err != nil {
		return notFoundGoogleError(err)
	}

	return fmt.Errorf("target pool[%s] is still alive", targetPoolName)
}

// listTargetPools returns all target pools in project and region
func listTargetPools(csv *gkeCompute.Service, project, regionName string) ([]*gkeCompute.TargetPool, error) {
	list, err := csv.TargetPools.List(project, regionName).Context(context.Background()).Do()
	if err != nil {
		return nil, err
	}

	return list.Items, nil
}

// checkForwardingRules checks all forwarding rules deleted by Kubernetes
func checkForwardingRules(csv *gkeCompute.Service, targetPools []*gkeCompute.TargetPool, project, regionName string) error {

	log.Infof("Check forwarding rules(frontends) in project[%s] and region[%s]", project, regionName)

	log.Info("List forwarding rules")
	forwardingRules, err := listForwardingRules(csv, project, regionName)
	if err != nil {
		return err
	}

	log.Debugf("Forwarding rules: %d", len(forwardingRules))

	for _, rule := range forwardingRules {
		if rule != nil && isClusterTarget(targetPools, project, regionName, rule.Target) {
			for {
				err := isForwardingRuleDeleted(csv, project, regionName, rule.Name)
				if err == nil {
					log.Infof("Forwarding rule[%s] deleted", rule.Name)
					break
				} else {
					log.Warn(err.Error())
					time.Sleep(time.Second * 5)
				}
			}
		}
	}

	return nil
}

// isClusterTarget checks the target match with the deleting cluster
func isClusterTarget(targetPools []*gkeCompute.TargetPool, project, region, targetPoolName string) bool {
	for _, tp := range targetPools {
		if tp != nil && tp.Name == getTargetUrl(project, region, targetPoolName) {
			return true
		}
	}
	return false
}

// getTargetUrl returns target url for gke cluster
func getTargetUrl(project, region, targetPoolName string) string {
	return fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/regions/%s/targetPools/%s", project, region, targetPoolName)
}

// isForwardingRuleDeleted checks the given forwarding rule is deleted by Kubernetes
func isForwardingRuleDeleted(csv *gkeCompute.Service, project, region, forwardingRule string) error {
	log.Infof("Get forwarding rule[%s]", forwardingRule)
	_, err := csv.ForwardingRules.Get(project, region, forwardingRule).Context(context.Background()).Do()
	if err != nil {
		return notFoundGoogleError(err)
	}

	return fmt.Errorf("forwarding rule[%s] is still alive", forwardingRule)
}

// notFoundGoogleError transforms an error into googleapi.Error
func notFoundGoogleError(err error) error {
	apiError, isOk := err.(*googleapi.Error)
	if isOk {
		if apiError.Code == http.StatusNotFound {
			return nil
		}
	}
	return err
}

// listForwardingRules returns all forwarding rule in project in region
func listForwardingRules(csv *gkeCompute.Service, project, region string) ([]*gkeCompute.ForwardingRule, error) {
	list, err := csv.ForwardingRules.List(project, region).Context(context.Background()).Do()
	if err != nil {
		return nil, err
	}

	return list.Items, nil
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

// checkFirewalls checks all load balancer resources deleted by Kubernetes
func checkFirewalls(csv *gkeCompute.Service, project, clusterName string) error {

	log.Info("Check firewalls")
	log.Info("List firewalls")
	firewalls, err := csv.Firewalls.List(project).Context(context.Background()).Do()
	if err != nil {
		return errors.Wrap(err, "Error during listing firewalls")
	}

	log.Infof("Find firewall(s) by target[%s]", clusterName)
	k8sFirewalls := findFirewallRulesByTarget(firewalls.Items, clusterName)

	for _, f := range k8sFirewalls {
		for {
			err := isFirewallDeleted(csv, project, f.Name)
			if err == nil {
				log.Infof("Firewall[%s] deleted", f.Name)
				break
			} else {
				log.Warn(err.Error())
				time.Sleep(time.Second * 5)
			}
		}
	}

	return nil
}

// isFirewallDeleted checks the given firewall is deleted by Kubernetes
func isFirewallDeleted(csv *gkeCompute.Service, project, firewall string) error {

	log.Infof("get firewall[%s] in project[%s]", firewall, project)

	_, err := csv.Firewalls.Get(project, firewall).Context(context.Background()).Do()
	if err != nil {
		return notFoundGoogleError(err)
	}

	return fmt.Errorf("firewall[%s] is still alive", firewall)
}

// findFirewallRulesByTarget returns all firewalls which created by Kubernetes
func findFirewallRulesByTarget(rules []*gkeCompute.Firewall, clusterName string) []*gkeCompute.Firewall {

	var firewalls []*gkeCompute.Firewall
	for _, r := range rules {
		if r != nil {

			if strings.Contains(r.Description, kubernetesIO) {

				for _, tag := range r.TargetTags {
					log.Debugf("Firewall rule[%s] target tag: %s", r.Name, tag)
					if strings.HasPrefix(tag, targetPrefix+clusterName) {
						log.Debugf("Append firewall list[%s]", r.Name)
						firewalls = append(firewalls, r)
					}
				}

			}
		}
	}

	return firewalls
}

// UpdateCluster updates GKE cluster in cloud
func (g *GKECluster) UpdateCluster(updateRequest *pkgCluster.UpdateClusterRequest) error {

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
		ProjectID:     secretItem.GetValue(pkgSecret.ProjectId),
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

//createNodePoolsFromClusterModel creates an array of google NodePool from the given cluster model
func createNodePoolsFromClusterModel(clusterModel *model.GoogleClusterModel) ([]*gke.NodePool, error) {
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
func createNodePoolsRequestDataFromNodePoolModel(nodePoolsModel []*model.GoogleNodePoolModel) (map[string]*pkgClusterGoogle.NodePool, error) {
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
			ServiceAccount:   nodePoolModel.ServiceAccount,
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
		ProjectID: secretItem.GetValue(pkgSecret.ProjectId),
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
	log.Debug("Create ClusterModel struct from the request")
	gkeCluster := GKECluster{
		modelCluster: clusterModel,
	}
	return &gkeCluster, nil
}

//AddDefaultsToUpdate adds defaults to update request
func (g *GKECluster) AddDefaultsToUpdate(r *pkgCluster.UpdateClusterRequest) {

	// TODO: error handling
	defGooglePools, _ := createNodePoolsFromClusterModel(&g.modelCluster.Google)

	defGoogleMaster := &pkgClusterGoogle.Master{
		Version: g.modelCluster.Google.MasterVersion,
	}

	// ---- [ Node check ] ---- //
	if r.Google.NodePools == nil {
		log.Warn("'nodePools' field is empty. Load it from stored data.")

		r.Google.NodePools = make(map[string]*pkgClusterGoogle.NodePool)
		for _, nodePool := range defGooglePools {
			r.Google.NodePools[nodePool.Name] = &pkgClusterGoogle.NodePool{
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
				nodePoolData.Count = pkgCluster.DefaultNodeMinCount
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
func (g *GKECluster) CheckEqualityToUpdate(r *pkgCluster.UpdateClusterRequest) error {

	// create update request struct with the stored data to check equality
	nodePools, _ := createNodePoolsRequestDataFromNodePoolModel(g.modelCluster.Google.NodePools)
	preCl := &pkgClusterGoogle.UpdateClusterGoogle{
		Master: &pkgClusterGoogle.Master{
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
			Cloud:          pkgCluster.Google,
		},
	}
	return g.GetGkeServerConfig(zone)
}

// GetGkeServerConfig returns configuration info about the Kubernetes Engine service.
func (g *GKECluster) GetGkeServerConfig(zone string) (*gke.ServerConfig, error) {

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
func (g *GKECluster) GetAllMachineTypesByZone(zone string) (map[string]pkgCluster.MachineType, error) {

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
func (g *GKECluster) GetAllMachineTypes() (map[string]pkgCluster.MachineType, error) {

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

	return s.GetValue(pkgSecret.ProjectId), nil
}

// UpdateStatus updates cluster status in database
func (g *GKECluster) UpdateStatus(status, statusMessage string) error {
	return g.modelCluster.UpdateStatus(status, statusMessage)
}

// GetClusterDetails gets cluster details from cloud
func (g *GKECluster) GetClusterDetails() (*pkgCluster.ClusterDetailsResponse, error) {
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
	cl, err := svc.Projects.Zones.Clusters.Get(secretItem.GetValue(pkgSecret.ProjectId), g.modelCluster.Location, g.modelCluster.Name).Context(context.Background()).Do()
	if err != nil {
		apiError := getBanzaiErrorFromError(err)
		return nil, errors.New(apiError.Message)
	}
	log.Info("Get cluster success")
	log.Infof("Cluster status is %s", cl.Status)
	if statusRunning == cl.Status {
		response := &pkgCluster.ClusterDetailsResponse{
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
	return nil, pkgErrors.ErrorClusterNotReady
}

// ValidateCreationFields validates all field
func (g *GKECluster) ValidateCreationFields(r *pkgCluster.CreateClusterRequest) error {

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
		return pkgErrors.ErrorNotValidLocation
	}

	return nil
}

// validateMachineType validates nodeInstanceTypes
func (g *GKECluster) validateMachineType(nodePools map[string]*pkgClusterGoogle.NodePool, location string) error {

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
			return pkgErrors.ErrorNotValidNodeInstanceType
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
func (g *GKECluster) GetSecretWithValidation() (*secret.SecretsItemResponse, error) {
	return g.CommonClusterBase.getSecret(g)
}

// GetSshSecretWithValidation returns ssh secret from vault
func (g *GKECluster) GetSshSecretWithValidation() (*secret.SecretsItemResponse, error) {
	return g.CommonClusterBase.getSecret(g)
}

// SaveConfigSecretId saves the config secret id in database
func (g *GKECluster) SaveConfigSecretId(configSecretId string) error {
	return g.modelCluster.UpdateConfigSecret(configSecretId)
}

// GetConfigSecretId return config secret id
func (g *GKECluster) GetConfigSecretId() string {
	return g.modelCluster.ConfigSecretId
}

// GetK8sConfig returns the Kubernetes config
func (g *GKECluster) GetK8sConfig() ([]byte, error) {
	return g.CommonClusterBase.getConfig(g)
}

// findInstanceByClusterName returns the cluster's instance
func findInstanceByClusterName(csv *gkeCompute.Service, project, zone, clusterName string) (*gkeCompute.Instance, error) {

	instances, err := csv.Instances.List(project, zone).Context(context.Background()).Do()
	if err != nil {
		return nil, err
	}

	for _, instance := range instances.Items {
		if instance != nil && instance.Metadata != nil {
			for _, item := range instance.Metadata.Items {
				if item != nil && item.Key == clusterNameKey && item.Value != nil && *item.Value == clusterName {
					return instance, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("instance not found by cluster[%s]", clusterName)
}

// ReloadFromDatabase load cluster from DBd
func (g *GKECluster) ReloadFromDatabase() error {
	return g.modelCluster.ReloadFromDatabase()
}
