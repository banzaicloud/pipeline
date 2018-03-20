package cluster

import (
	"encoding/base64"
	"fmt"
	"github.com/banzaicloud/banzai-types/components"
	bGoogle "github.com/banzaicloud/banzai-types/components/google"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/utils"
	"github.com/gin-gonic/gin"
	"github.com/go-errors/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
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

var credentialPath string

const googleAppCredentialKey = "cloud.gkeCredentialPath"

const (
	statusRunning = "RUNNING"
)

const (
	defaultNamespace = "default"
	clusterAdmin     = "cluster-admin"
	netesDefault     = "netes-default"
)

//CreateGKEClusterFromRequest creates ClusterModel struct from the request
func CreateGKEClusterFromRequest(request *components.CreateClusterRequest, orgId uint) (*GKECluster, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagCreateCluster})
	log.Debug("Create ClusterModel struct from the request")
	var cluster GKECluster

	cluster.modelCluster = &model.ClusterModel{
		Name:             request.Name,
		Location:         request.Location,
		NodeInstanceType: request.NodeInstanceType,
		Cloud:            request.Cloud,
		OrganizationId:   orgId,
		Google: model.GoogleClusterModel{
			Project:        request.Properties.CreateClusterGoogle.Project,
			MasterVersion:  request.Properties.CreateClusterGoogle.Master.Version,
			NodeVersion:    request.Properties.CreateClusterGoogle.Node.Version,
			NodeCount:      request.Properties.CreateClusterGoogle.Node.Count,
			ServiceAccount: request.Properties.CreateClusterGoogle.Node.ServiceAccount,
		},
	}
	return &cluster, nil
}

//GKECluster struct for GKE cluster
type GKECluster struct {
	googleCluster *gke.Cluster //Don't use this directly
	modelCluster  *model.ClusterModel
	k8sConfig     *[]byte
	APIEndpoint   string
}

func (g *GKECluster) GetOrg() uint {
	return g.modelCluster.OrganizationId
}

func (g *GKECluster) GetSecretID() string {
	return g.modelCluster.SecretId
}

func (g *GKECluster) GetGoogleCluster() (*gke.Cluster, error) {
	if g.googleCluster != nil {
		return g.googleCluster, nil
	}
	getCredentialPath()
	svc, err := getGoogleServiceClient()
	if err != nil {
		return nil, err
	}
	cc := googleCluster{
		Name:      g.modelCluster.Name,
		ProjectID: g.modelCluster.Google.Project,
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

	log.Info("Read google application credential path")
	data, err := ioutil.ReadFile(getCredentialPath())
	if err != nil {
		return err
	}

	log.Info("Read credential path success")

	log.Info("Get Google Service Client")
	svc, err := getGoogleServiceClient()
	if err != nil {
		return err
	}

	log.Info("Get Google Service Client succeeded")

	cc := googleCluster{
		NodeConfig: &gke.NodeConfig{
			MachineType:    g.modelCluster.NodeInstanceType,
			ServiceAccount: g.modelCluster.Google.ServiceAccount,
			OauthScopes: []string{
				"https://www.googleapis.com/auth/logging.write",
				"https://www.googleapis.com/auth/monitoring",
				"https://www.googleapis.com/auth/devstorage.read_write",
			},
		},
		ProjectID:         g.modelCluster.Google.Project,
		Zone:              g.modelCluster.Location,
		Name:              g.modelCluster.Name,
		NodeCount:         int64(g.modelCluster.Google.NodeCount),
		CredentialPath:    getCredentialPath(),
		CredentialContent: string(data),
		MasterVersion:     g.modelCluster.Google.MasterVersion,
		NodeVersion:       g.modelCluster.Google.NodeVersion,
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

	// save to database before polling
	if err := g.Persist(); err != nil {
		log.Errorf("Cluster save failed! %s", err.Error())
	}

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
func (g *GKECluster) Persist() error {
	log.Infof("Model before save: %v", g.modelCluster)
	return g.modelCluster.Save()
}

//GetK8sConfig returns the Kubernetes config
func (g *GKECluster) GetK8sConfig() (*[]byte, error) {

	if g.k8sConfig != nil {
		return g.k8sConfig, nil
	}
	log := logger.WithFields(logrus.Fields{"action": constants.TagFetchClusterConfig})

	// to set env var
	_ = getCredentialPath()

	config, err := getGoogleKubernetesConfig(g.modelCluster)
	if err != nil {
		// something went wrong
		be := getBanzaiErrorFromError(err)
		// TODO status code !?
		return nil, errors.New(be.Message)
	}
	// get config succeeded
	log.Info("Get k8s config succeeded")

	g.k8sConfig = &config

	return &config, nil

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

	log := logger.WithFields(logrus.Fields{"action": constants.TagFetchClusterConfig})

	log.Info("Start get cluster status (google)")

	// to set env var
	_ = getCredentialPath()

	log.Info("Get Google Service Client")
	svc, err := getGoogleServiceClient()
	if err != nil {
		be := getBanzaiErrorFromError(err)
		// TODO status code !?
		return nil, errors.New(be.Message)
	}
	log.Info("Get Google Service Client success")

	log.Infof("Get google cluster with name %s", g.modelCluster.Name)
	cl, err := svc.Projects.Zones.Clusters.Get(g.modelCluster.Google.Project, g.modelCluster.Location, g.modelCluster.Name).Context(context.Background()).Do()
	if err != nil {
		apiError := getBanzaiErrorFromError(err)
		// TODO status code !?
		return nil, errors.New(apiError.Message)
	}
	log.Info("Get cluster success")
	log.Infof("Cluster status is %s", cl.Status)
	if statusRunning == cl.Status {
		response := &components.GetClusterStatusResponse{
			Status:           http.StatusOK,
			Name:             g.modelCluster.Name,
			Location:         g.modelCluster.Location,
			Cloud:            g.modelCluster.Cloud,
			NodeInstanceType: g.modelCluster.NodeInstanceType,
			ResourceID:       g.modelCluster.ID,
		}
		return response, nil
	}
	return nil, constants.ErrorClusterNotReady

}

// DeleteCluster deletes cluster from google
func (g *GKECluster) DeleteCluster() error {

	log := logger.WithFields(logrus.Fields{"action": constants.TagDeleteCluster})

	log.Info("Start delete google cluster")

	if g == nil {
		return constants.ErrorNilCluster
	}

	gkec := googleCluster{
		ProjectID: g.modelCluster.Google.Project,
		Name:      g.modelCluster.Name,
		Zone:      g.modelCluster.Location,
	}

	if err := callDeleteCluster(&gkec); err != nil {
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

	// to set env var
	_ = getCredentialPath()

	svc, err := getGoogleServiceClient()
	if err != nil {
		return err
	}

	cc := googleCluster{
		Name:          g.modelCluster.Name,
		ProjectID:     g.modelCluster.Google.Project,
		Zone:          g.modelCluster.Location,
		MasterVersion: updateRequest.GoogleMaster.Version,
		NodeVersion:   updateRequest.GoogleNode.Version,
		NodeCount:     int64(updateRequest.GoogleNode.Count),
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
	g.updateModel(res)

	return nil

}

func (g *GKECluster) updateModel(c *gke.Cluster) {
	g.modelCluster.Google.MasterVersion = c.CurrentMasterVersion
	g.modelCluster.Google.NodeVersion = c.CurrentNodeVersion
	g.modelCluster.Google.NodeCount = int(c.CurrentNodeCount)
	if c.NodeConfig != nil {
		g.modelCluster.Google.ServiceAccount = c.NodeConfig.ServiceAccount
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

// getCredentialPath returns the Google application credential path and set the env variable
func getCredentialPath() string {
	log.Info("Get gke credential path")
	if len(credentialPath) == 0 {
		credentialPath = viper.GetString(googleAppCredentialKey)
		// set GOOGLE_APPLICATION_CREDENTIALS environment variable to specify
		// a service account key file to authenticate to the Google Cloud API
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credentialPath)
	}

	log.Debugf("Credential path is %s", credentialPath)

	return credentialPath
}

func getGoogleServiceClient() (*gke.Service, error) {

	// See https://cloud.google.com/docs/authentication/.
	// Use GOOGLE_APPLICATION_CREDENTIALS environment variable to specify
	// a service account key file to authenticate to the API.

	client, err := google.DefaultClient(context.Background(), gke.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
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
	// The number of nodes to create in this cluster
	NodeCount int64
	// the kubernetes master version
	MasterVersion string
	// The authentication information for accessing the master
	MasterAuth *gke.MasterAuth
	// the kubernetes node version
	NodeVersion string
	// The name of this cluster
	Name string
	// Parameters used in creating the cluster's nodes
	NodeConfig *gke.NodeConfig
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
	// NodePool id
	NodePoolID string
	// Image Type
	ImageType string
}

func generateClusterCreateRequest(cc googleCluster) *gke.CreateClusterRequest {
	request := gke.CreateClusterRequest{
		Cluster: &gke.Cluster{},
	}
	request.Cluster.Name = cc.Name
	request.Cluster.Zone = cc.Zone
	request.Cluster.InitialClusterVersion = cc.MasterVersion
	request.Cluster.InitialNodeCount = cc.NodeCount
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
	request.Cluster.NodeConfig = cc.NodeConfig
	return &request
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

func callDeleteCluster(cc *googleCluster) error {

	_ = getCredentialPath()

	log.Info("Get Google Service Client")

	svc, err := getGoogleServiceClient()
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
	if cc.NodePoolID == "" {
		cluster, err := getClusterGoogle(svc, cc)
		if err != nil {
			return nil, err
		}
		if cluster.NodePools != nil && len(cluster.NodePools) != 0 {
			cc.NodePoolID = cluster.NodePools[0].Name
		}
	}

	if cc.MasterVersion != "" {
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

	if cc.NodeVersion != "" {
		log.Infof("Updating node to %v version", cc.NodeVersion)
		updateCall, err := svc.Projects.Zones.Clusters.NodePools.Update(cc.ProjectID, cc.Zone, cc.Name, cc.NodePoolID, &gke.UpdateNodePoolRequest{
			NodeVersion: cc.NodeVersion,
		}).Context(context.Background()).Do()
		if err != nil {
			return nil, err
		}
		log.Infof("Nodepool %s update is called for project %s, zone %s and cluster %s. Status Code %v", cc.NodePoolID, cc.ProjectID, cc.Zone, cc.Name, updateCall.HTTPStatusCode)
		if err := waitForNodePool(svc, &cc); err != nil {
			return nil, err
		}
	}

	if cc.NodeCount != 0 {
		log.Infof("Updating node size to %v", cc.NodeCount)
		updateCall, err := svc.Projects.Zones.Clusters.NodePools.SetSize(cc.ProjectID, cc.Zone, cc.Name, cc.NodePoolID, &gke.SetNodePoolSizeRequest{
			NodeCount: cc.NodeCount,
		}).Context(context.Background()).Do()
		if err != nil {
			return nil, err
		}
		log.Infof("Nodepool %s size change is called for project %s, zone %s and cluster %s. Status Code %v", cc.NodePoolID, cc.ProjectID, cc.Zone, cc.Name, updateCall.HTTPStatusCode)
		if updatedCluster, err = waitForCluster(svc, cc); err != nil {
			return nil, err
		}
	}
	return updatedCluster, nil
}

func waitForNodePool(svc *gke.Service, cc *googleCluster) error {
	var message string
	for {
		nodepool, err := svc.Projects.Zones.Clusters.NodePools.Get(cc.ProjectID, cc.Zone, cc.Name, cc.NodePoolID).Context(context.TODO()).Do()
		if err != nil {
			return err
		}
		if nodepool.Status == statusRunning {
			log.Infof("Nodepool %v is running", cc.Name)
			return nil
		}
		if nodepool.Status != message {
			log.Infof("%v nodepool %v", string(nodepool.Status), cc.NodePoolID)
			message = nodepool.Status
		}
		time.Sleep(time.Second * 5)
	}
}

func getGoogleKubernetesConfig(cs *model.ClusterModel) ([]byte, error) {

	log.Info("Get Google Service Client")
	svc, err := getGoogleServiceClient()
	if err != nil {
		return nil, err
	}
	log.Info("Get Google Service Client succeeded")

	log.Infof("Get google cluster with name %s", cs.Name)
	cl, err := getClusterGoogle(svc, googleCluster{
		Name:      cs.Name,
		ProjectID: cs.Google.Project,
		Zone:      cs.Location,
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

	finalCl.Metadata["nodePool"] = cl.NodePools[0].Name

	// TODO if the final solution is NOT SAVE CONFIG TO FILE than rename the method and change log message
	log.Info("Start save config file")
	config, err := storeConfig(&finalCl, cs.Name)
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

	configFile := fmt.Sprintf("./statestore/%s/config", name)
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

	// setup users
	user := configUser{
		User: userData{
			Username: username,
			Password: password,
			Token:    token,
			ClientCertificateData: c.ClientCertificate,
			ClientKeyData:         c.ClientKey,
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
}

type kubeConfig struct {
	APIVersion     string          `yaml:"apiVersion,omitempty"`
	Clusters       []configCluster `yaml:"clusters,omitempty"`
	Contexts       []configContext `yaml:"contexts,omitempty"`
	Users          []configUser    `yaml:"users,omitempty"`
	CurrentContext string          `yaml:"current-context,omitempty"`
	Kind           string          `yaml:"kind,omitempty"`
	Preferences    string          `yaml:"preferences,omitempty"`
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
	Token                 string `yaml:"token,omitempty"`
	Username              string `yaml:"username,omitempty"`
	Password              string `yaml:"password,omitempty"`
	ClientCertificateData string `yaml:"client-certificate-data,omitempty"`
	ClientKeyData         string `yaml:"client-key-data,omitempty"`
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

	defGoogleNode := &bGoogle.GoogleNode{
		Version: g.modelCluster.Google.NodeVersion,
		Count:   g.modelCluster.Google.NodeCount,
	}

	defGoogleMaster := &bGoogle.GoogleMaster{
		Version: g.modelCluster.Google.MasterVersion,
	}

	// ---- [ Node check ] ---- //
	if r.GoogleNode == nil {
		log.Warn("'node' field is empty. Load it from stored data.")
		r.GoogleNode = defGoogleNode
	}

	// ---- [ Master check ] ---- //
	if r.GoogleMaster == nil {
		log.Warn("'master' field is empty. Load it from stored data.")
		r.GoogleMaster = defGoogleMaster
	}

	// ---- [ NodeCount check] ---- //
	if r.UpdateClusterGoogle.GoogleNode.Count == 0 {
		def := g.modelCluster.Google.NodeCount
		log.Warn("Node count set to default value: ", def)
		r.UpdateClusterGoogle.GoogleNode.Count = def
	}

	// ---- [ Node Version check] ---- //
	if len(r.UpdateClusterGoogle.GoogleNode.Version) == 0 {
		nodeVersion := g.modelCluster.Google.NodeVersion
		log.Warn("Node K8s version: ", nodeVersion)
		r.UpdateClusterGoogle.GoogleNode.Version = nodeVersion
	}

	// ---- [ Master Version check] ---- //
	if len(r.UpdateClusterGoogle.GoogleMaster.Version) == 0 {
		masterVersion := g.modelCluster.Google.MasterVersion
		log.Warn("Master K8s version: ", masterVersion)
		r.UpdateClusterGoogle.GoogleMaster.Version = masterVersion
	}

}

//CheckEqualityToUpdate validates the update request
func (g *GKECluster) CheckEqualityToUpdate(r *components.UpdateClusterRequest) error {

	log := logger.WithFields(logrus.Fields{"action": "CheckEqualityToUpdate"})

	// create update request struct with the stored data to check equality
	preCl := &bGoogle.UpdateClusterGoogle{
		GoogleMaster: &bGoogle.GoogleMaster{
			Version: g.modelCluster.Google.MasterVersion,
		},
		GoogleNode: &bGoogle.GoogleNode{
			Version: g.modelCluster.Google.NodeVersion,
			Count:   g.modelCluster.Google.NodeCount,
		},
	}

	log.Info("Check stored & updated cluster equals")

	// check equality
	return utils.IsDifferent(r.UpdateClusterGoogle, preCl)
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

// GetGkeServerConfig returns configuration info about the Kubernetes Engine service.
func GetGkeServerConfig(c *gin.Context) {

	log := logger.WithFields(logrus.Fields{"action": "GetGkeServerConfig"})

	projectId := c.Param("projectid")
	zone := c.Param("zone")

	log.Info("Start getting configuration info")

	_ = getCredentialPath()

	log.Info("Get Google service client")
	if svc, err := getGoogleServiceClient(); err != nil {
		apiErr := getBanzaiErrorFromError(err)
		log.Errorf("Error during getting service client: %s", apiErr.Message)
		c.JSON(apiErr.StatusCode, components.ErrorResponse{
			Code:    apiErr.StatusCode,
			Message: "Error during getting service client",
			Error:   apiErr.Message,
		})
	} else {
		if serverConfig, err := svc.Projects.Zones.GetServerconfig(projectId, zone).Context(context.Background()).Do(); err != nil {
			apiErr := getBanzaiErrorFromError(err)
			log.Errorf("Error during getting server config: %s", apiErr.Message)
			c.JSON(apiErr.StatusCode, components.ErrorResponse{
				Code:    apiErr.StatusCode,
				Message: "Error during getting server config",
				Error:   apiErr.Message,
			})
		} else {
			log.Info("Getting server config succeeded")
			c.JSON(http.StatusOK, convertServerConfig(serverConfig))
		}

	}

}

type GetServerConfigResponse struct {
	// Version of Kubernetes the service deploys by default.
	DefaultClusterVersion string `json:"defaultClusterVersion"`
	// Default image type.
	DefaultImageType string `json:"defaultImageType"`
	// List of valid image types.
	ValidImageTypes []string `json:"validImageTypes"`
	// List of valid master versions.
	ValidMasterVersions []string `json:"validMasterVersions"`
	// List of valid node upgrade target versions.
	ValidNodeVersions []string `json:"validNodeVersions"`
}

// convertServerConfig create a GetServerConfigResponse from ServerConfig
func convertServerConfig(config *gke.ServerConfig) *GetServerConfigResponse {
	return &GetServerConfigResponse{
		DefaultClusterVersion: config.DefaultClusterVersion,
		DefaultImageType:      config.DefaultImageType,
		ValidImageTypes:       config.ValidImageTypes,
		ValidMasterVersions:   config.ValidMasterVersions,
		ValidNodeVersions:     config.ValidNodeVersions,
	}
}
