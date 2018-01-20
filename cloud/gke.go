package cloud

import (
	"os"
	"io/ioutil"
	"golang.org/x/oauth2/google"
	gke "google.golang.org/api/container/v1"
	"golang.org/x/net/context"
	"strings"
	"time"
	"google.golang.org/api/googleapi"
	"net/http"
	"github.com/gin-gonic/gin"

	banzaiUtils "github.com/banzaicloud/banzai-types/utils"
	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
	banzaiTypes "github.com/banzaicloud/banzai-types/components"
	banzaiSimpleTypes "github.com/banzaicloud/banzai-types/components/database"
	"github.com/banzaicloud/banzai-types/database"
)

var credentialPath string

const (
	statusRunning = "RUNNING"
)

func init() {
	// todo key
	credentialPath = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	banzaiUtils.LogDebugf(banzaiConstants.TagInit, "GOOGLE_APPLICATION_CREDENTIALS is %s", credentialPath)
}

func CreateClusterGoogle(request *banzaiTypes.CreateClusterRequest, c *gin.Context) (bool, *banzaiSimpleTypes.ClusterSimple) {
	// todo change tags
	data, err := ioutil.ReadFile(credentialPath)
	if err != nil {
		banzaiUtils.LogFatalf(banzaiConstants.TagCreateCluster, "GOOGLE_APPLICATION_CREDENTIALS env var is not specified: %s", err)
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus:  http.StatusInternalServerError,
			JsonKeyMessage: "GOOGLE_APPLICATION_CREDENTIALS env var is not specified",
		})
		return false, nil
	}

	svc, err := getServiceClient()
	if err != nil {
		// todo log?
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus:  http.StatusInternalServerError,
			JsonKeyMessage: err,
		})
		return false, nil
	}

	cc := GKECluster{
		ProjectID:         request.Properties.CreateClusterGoogle.Project,
		Zone:              request.Location,
		Name:              request.Name,
		NodeCount:         int64(request.Properties.CreateClusterGoogle.Node.Count),
		CredentialPath:    credentialPath,
		CredentialContent: string(data),
	}

	banzaiUtils.LogInfof(banzaiConstants.TagCreateCluster, "Cluster request: %v", generateClusterCreateRequest(cc))
	createCall, err := svc.Projects.Zones.Clusters.Create(cc.ProjectID, cc.Zone, generateClusterCreateRequest(cc)).Context(context.Background()).Do()

	banzaiUtils.LogInfof(banzaiConstants.TagCreateCluster, "Cluster request submitted: %v", generateClusterCreateRequest(cc))

	if err != nil && !strings.Contains(err.Error(), "alreadyExists") {
		banzaiUtils.LogInfof(banzaiConstants.TagCreateCluster, "Contains error: %s", err)
		be := getBanzaiErrorFromError(err)
		SetResponseBodyJson(c, be.StatusCode, gin.H{
			JsonKeyStatus:  be.StatusCode,
			JsonKeyMessage: be.Message,
		})
		return false, nil
	} else {
		banzaiUtils.LogInfof(banzaiConstants.TagCreateCluster, "Cluster %s create is called for project %s and zone %s. Status Code %v", cc.Name, cc.ProjectID, cc.Zone, createCall.HTTPStatusCode)
	}

	gkeCluster, err := waitForCluster(svc, cc)
	if err != nil {
		banzaiUtils.LogErrorf(banzaiConstants.TagCreateCluster, "Cluster create failed", err)
		be := getBanzaiErrorFromError(err)
		SetResponseBodyJson(c, be.StatusCode, gin.H{
			JsonKeyStatus:  be.StatusCode,
			JsonKeyMessage: be.Message,
		})
		return false, nil
	}

	cluster2Db := banzaiSimpleTypes.ClusterSimple{
		Name:             request.Name,
		Location:         request.Location,
		NodeInstanceType: request.NodeInstanceType,
		Cloud:            request.Cloud,
		Google: banzaiSimpleTypes.GoogleClusterSimple{
			Project:   request.Properties.CreateClusterGoogle.Project,
			NodeCount: request.Properties.CreateClusterGoogle.Node.Count,
		},
	}

	if err := database.Save(&cluster2Db).Error; err != nil {
		DbSaveFailed(c, err, cluster2Db.Name)
		return false, nil
	}

	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Save create cluster into database succeeded")
	SetResponseBodyJson(c, http.StatusCreated, gin.H{
		JsonKeyStatus:     http.StatusCreated,
		JsonKeyResourceId: cluster2Db.ID,
		JsonKeyData:       gkeCluster,
	})
	return true, &cluster2Db
}

func getBanzaiErrorFromError(err error) *banzaiTypes.BanzaiResponse {

	if err == nil {
		// error is nil
		return &banzaiTypes.BanzaiResponse{
			StatusCode: http.StatusInternalServerError,
		}
	}

	googleErr, ok := err.(*googleapi.Error)
	if ok {
		// error is googleapi error
		return &banzaiTypes.BanzaiResponse{
			StatusCode: googleErr.Code,
			Message:    googleErr.Message,
		}
	}

	// default
	return &banzaiTypes.BanzaiResponse{
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
		banzaiUtils.LogFatalf(banzaiConstants.TagCreateCluster, "Could not get authenticated client: %v", err)
		return nil, err
	}
	service, err := gke.New(client)
	if err != nil {
		banzaiUtils.LogFatalf(banzaiConstants.TagCreateCluster, "Could not initialize gke client: %v", err)
		return nil, err
	}
	banzaiUtils.LogInfof(banzaiConstants.TagCreateCluster, "Using service acc: %v", service)
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

func waitForCluster(svc *gke.Service, cc GKECluster) (*gke.Cluster, error) {

	message := ""
	for {

		cluster, err := svc.Projects.Zones.Clusters.Get(cc.ProjectID, cc.Zone, cc.Name).Context(context.TODO()).Do()
		if err != nil {
			return nil, err
		}

		if cluster.Status == statusRunning {
			// todo tag
			banzaiUtils.LogInfof(banzaiConstants.TagCreateCluster, "Cluster %v is running", cc.Name)
			return cluster, nil
		}

		if cluster.Status != message {
			banzaiUtils.LogInfof(banzaiConstants.TagCreateCluster, "%v cluster %v", string(cluster.Status), cc.Name)
			message = cluster.Status
		}
		time.Sleep(time.Second * 5)

	}
}
