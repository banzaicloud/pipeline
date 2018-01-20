package cloud

import (
	"os"
	"io/ioutil"
	"github.com/banzaicloud/banzai-types/utils"
	"github.com/banzaicloud/banzai-types/constants"
	"golang.org/x/oauth2/google"
	gke "google.golang.org/api/container/v1"
	"golang.org/x/net/context"
	"strings"
	"time"
	"google.golang.org/api/googleapi"
	"github.com/banzaicloud/banzai-types/components"
	"net/http"
)

var credentialPath string

const (
	statusRunning = "RUNNING"
)

func init() {
	// todo key
	credentialPath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	utils.LogDebugf(constants.TagInit, "GOOGLE_APPLICATION_CREDENTIALS is %s", credentialPath)
}

func CreateClusterGoogle(request *components.CreateClusterRequest) *components.BanzaiResponse {
	// todo change tags
	data, err := ioutil.ReadFile(credentialPath)
	if err != nil {
		utils.LogFatalf(constants.TagCreateCluster, "GOOGLE_APPLICATION_CREDENTIALS env var is not specified: %s", err)
		return &components.BanzaiResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    "GOOGLE_APPLICATION_CREDENTIALS env var is not specified",
		}
	}

	svc, err := getServiceClient()
	if err != nil {
		// todo log?
		return &components.BanzaiResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    err.Error(),
		}
	}

	cc := GKECluster{
		ProjectID:         request.Properties.CreateClusterGoogle.Project,
		Zone:              request.Location,
		Name:              request.Name,
		NodeCount:         int64(request.Properties.CreateClusterGoogle.Node.Count),
		CredentialPath:    credentialPath,
		CredentialContent: string(data),
	}

	utils.LogInfof(constants.TagCreateCluster, "Cluster request: %v", generateClusterCreateRequest(cc))
	createCall, err := svc.Projects.Zones.Clusters.Create(cc.ProjectID, cc.Zone, generateClusterCreateRequest(cc)).Context(context.Background()).Do()

	utils.LogInfof(constants.TagCreateCluster, "Cluster request submitted: %v", generateClusterCreateRequest(cc))

	if err != nil && !strings.Contains(err.Error(), "alreadyExists") {
		utils.LogInfof(constants.TagCreateCluster, "Contains error: %s", err)
		return getBanzaiErrorFromError(err)
	} else {
		utils.LogInfof(constants.TagCreateCluster, "Cluster %s create is called for project %s and zone %s. Status Code %v", cc.Name, cc.ProjectID, cc.Zone, createCall.HTTPStatusCode)
	}

	err = waitForCluster(svc, cc)
	if err != nil {
		utils.LogErrorf(constants.TagCreateCluster, "Cluster create failed", err)
		return getBanzaiErrorFromError(err)
	}

	// everything is ok
	return nil
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

func generateClusterCreateRequest(cc GKECluster) *gke.CreateClusterRequest {
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
		Enabled: cc.LegacyAbac,
	}
	request.Cluster.MasterAuth = &gke.MasterAuth{
		Username: "admin",
	}
	request.Cluster.NodeConfig = cc.NodeConfig
	return &request
}

func getServiceClient() (*gke.Service, error) {

	// See https://cloud.google.com/docs/authentication/.
	// Use GOOGLE_APPLICATION_CREDENTIALS environment variable to specify
	// a service account key file to authenticate to the API.

	client, err := google.DefaultClient(context.Background(), gke.CloudPlatformScope)
	if err != nil {
		// todo replace banzai-types tag
		utils.LogFatalf(constants.TagCreateCluster, "Could not get authenticated client: %v", err)
		return nil, err
	}
	service, err := gke.New(client)
	if err != nil {
		utils.LogFatalf(constants.TagCreateCluster, "Could not initialize gke client: %v", err)
		return nil, err
	}
	utils.LogInfof(constants.TagCreateCluster, "Using service acc: %v", service)
	return service, nil
}

// todo replace to banzai-types
// Struct of GKE
type GKECluster struct {
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

func waitForCluster(svc *gke.Service, cc GKECluster) error {
	message := ""
	for {
		cluster, err := svc.Projects.Zones.Clusters.Get(cc.ProjectID, cc.Zone, cc.Name).Context(context.TODO()).Do()
		if err != nil {
			return err
		}
		if cluster.Status == statusRunning {
			// todo tag
			utils.LogInfof(constants.TagCreateCluster, "Cluster %v is running", cc.Name)
			return nil
		}
		if cluster.Status != message {
			utils.LogInfof(constants.TagCreateCluster, "%v cluster %v", string(cluster.Status), cc.Name)
			message = cluster.Status
		}
		time.Sleep(time.Second * 5)
	}
}
