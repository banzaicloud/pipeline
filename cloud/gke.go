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
	"encoding/base64"
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"gopkg.in/yaml.v2"
	"path/filepath"
	"github.com/spf13/viper"
)

var credentialPath string

const (
	statusRunning = "RUNNING"
)

const (
	defaultNamespace = "default"
	clusterAdmin     = "cluster-admin"
	netesDefault     = "netes-default"
)

const googleAppCredential = "dev.gkeCredentialPath"

func getCredentialPath() string {
	banzaiUtils.LogInfo(banzaiConstants.TagInit, "Get gke credential path")
	if len(credentialPath) == 0 {
		credentialPath = viper.GetString(googleAppCredential)
		banzaiUtils.LogDebugf(banzaiConstants.TagInit, "Credential path is %s", credentialPath)
	} else {
		banzaiUtils.LogError(banzaiConstants.TagInit, "Credential path is not configured")
	}
	return credentialPath
}

func CreateClusterGoogle(request *banzaiTypes.CreateClusterRequest, c *gin.Context) (bool, *banzaiSimpleTypes.ClusterSimple) {

	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Start create cluster (Google)")

	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Read credential path")
	data, err := ioutil.ReadFile(getCredentialPath())
	if err != nil {
		banzaiUtils.LogErrorf(banzaiConstants.TagCreateCluster, "GKE credential path is not specified: %s", err.Error())
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus:  http.StatusInternalServerError,
			JsonKeyMessage: "GKE credential path is not specified",
		})
		return false, nil
	}
	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Read success")

	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Get Google Service Client")
	svc, err := GetGoogleServiceClient()
	if err != nil {
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus:  http.StatusInternalServerError,
			JsonKeyMessage: err,
		})
		return false, nil
	}
	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Get Google Service Client succeeded")

	cc := GKECluster{
		ProjectID:         request.Properties.CreateClusterGoogle.Project,
		Zone:              request.Location,
		Name:              request.Name,
		NodeCount:         int64(request.Properties.CreateClusterGoogle.Node.Count),
		CredentialPath:    getCredentialPath(),
		CredentialContent: string(data),
		MasterVersion:     request.Properties.CreateClusterGoogle.Master.Version,
		NodeVersion:       request.Properties.CreateClusterGoogle.Node.Version,
	}

	ccr := generateClusterCreateRequest(cc)

	banzaiUtils.LogInfof(banzaiConstants.TagCreateCluster, "Cluster request: %v", ccr)
	createCall, err := svc.Projects.Zones.Clusters.Create(cc.ProjectID, cc.Zone, ccr).Context(context.Background()).Do()

	banzaiUtils.LogInfof(banzaiConstants.TagCreateCluster, "Cluster request submitted: %v", ccr)

	if err != nil && !strings.Contains(err.Error(), "alreadyExists") {
		banzaiUtils.LogErrorf(banzaiConstants.TagCreateCluster, "Contains error: %s", err.Error())
		be := getBanzaiErrorFromError(err)
		SetResponseBodyJson(c, be.StatusCode, gin.H{
			JsonKeyStatus:  be.StatusCode,
			JsonKeyMessage: be.Message,
		})
		return false, nil
	} else {
		banzaiUtils.LogInfof(banzaiConstants.TagCreateCluster, "Cluster %s create is called for project %s and zone %s. Status Code %v", cc.Name, cc.ProjectID, cc.Zone, createCall.HTTPStatusCode)
	}

	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Waiting for cluster...")

	gkeCluster, err := waitForCluster(svc, cc)
	if err != nil {
		banzaiUtils.LogErrorf(banzaiConstants.TagCreateCluster, "Cluster create failed: %s", err.Error())
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
			Project:       request.Properties.CreateClusterGoogle.Project,
			NodeCount:     request.Properties.CreateClusterGoogle.Node.Count,
			MasterVersion: request.Properties.CreateClusterGoogle.Master.Version,
			NodeVersion:   request.Properties.CreateClusterGoogle.Node.Version,
		},
	}

	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Save created cluster into database: %v", cluster2Db)
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

func GetGoogleServiceClient() (*gke.Service, error) {

	// See https://cloud.google.com/docs/authentication/.
	// Use GOOGLE_APPLICATION_CREDENTIALS environment variable to specify
	// a service account key file to authenticate to the API.

	client, err := google.DefaultClient(context.Background(), gke.CloudPlatformScope)
	if err != nil {
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

	var message string
	for {

		cluster, err := svc.Projects.Zones.Clusters.Get(cc.ProjectID, cc.Zone, cc.Name).Context(context.TODO()).Do()
		if err != nil {
			banzaiUtils.LogErrorf(banzaiConstants.TagCreateCluster, "error during getting cluster: %s", err.Error())
			return nil, err
		}

		banzaiUtils.LogInfof(banzaiConstants.TagCreateCluster, "Cluster status: %s", cluster.Status)

		if cluster.Status == statusRunning {
			banzaiUtils.LogInfof(banzaiConstants.TagCreateCluster, "Cluster %s is running", cc.Name)
			return cluster, nil
		}

		if cluster.Status != message {
			banzaiUtils.LogInfof(banzaiConstants.TagCreateCluster, "%s cluster %s", string(cluster.Status), cc.Name)
			message = cluster.Status
		}

		time.Sleep(time.Second * 5)

	}
}

func GetClusterGoogle(svc *gke.Service, cc GKECluster) (*gke.Cluster, error) {
	return svc.Projects.Zones.Clusters.Get(cc.ProjectID, cc.Zone, cc.Name).Context(context.TODO()).Do()
}

func ReadClusterGoogle(cs *banzaiSimpleTypes.ClusterSimple, svc *gke.Service) *ClusterRepresentation {
	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Read google cluster with", cs.Name, "id")

	if cs == nil {
		banzaiUtils.LogError(banzaiConstants.TagGetCluster, "<nil> cluster")
		return nil
	}

	gkec := GKECluster{
		ProjectID: cs.Google.Project,
		Name:      cs.Name,
		Zone:      cs.Location,
	}

	response, err := GetClusterGoogle(svc, gkec)
	if err != nil {
		banzaiUtils.LogError(banzaiConstants.TagGetCluster, "Something went wrong under read:", err)
		return nil
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Read cluster success")
		clust := ClusterRepresentation{
			Id:        cs.ID,
			Name:      cs.Name,
			CloudType: banzaiConstants.Google,
			GoogleRepresentation: &GoogleRepresentation{
				GoogleCluster: response,
			},
		}
		return &clust
	}
}

type GoogleRepresentation struct {
	GoogleCluster *gke.Cluster `json:"value,omitempty"`
}

func GetClusterInfoGoogle(cs *banzaiSimpleTypes.ClusterSimple, c *gin.Context) {
	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Fetch aks cluster with name:", cs.Name, "in", cs.Azure.ResourceGroup, "resource group.")

	if cs == nil {
		banzaiUtils.LogError(banzaiConstants.TagGetCluster, "<nil> cluster")
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus: http.StatusInternalServerError,
		})
		return
	}

	svc, err := GetGoogleServiceClient()
	if err != nil {
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus:  http.StatusInternalServerError,
			JsonKeyMessage: err,
		})
		return
	}

	gkec := GKECluster{
		ProjectID: cs.Google.Project,
		Name:      cs.Name,
		Zone:      cs.Location,
	}

	response, err := GetClusterGoogle(svc, gkec)
	if err != nil {
		// fetch failed
		googleApiErr := getBanzaiErrorFromError(err)
		banzaiUtils.LogErrorf(banzaiConstants.TagGetCluster, "Status code: %d", googleApiErr.StatusCode)
		banzaiUtils.LogErrorf(banzaiConstants.TagGetCluster, "Error during get cluster details: %s", googleApiErr.Message)
		SetResponseBodyJson(c, googleApiErr.StatusCode, googleApiErr)
	} else {
		// fetch success
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Fetch success")
		SetResponseBodyJson(c, http.StatusOK, gin.H{
			JsonKeyResourceId: cs.ID,
			JsonKeyData:       response,
		})
	}

}

// UpdateClusterGoogleInCloud updates google cluster in cloud
func UpdateClusterGoogleInCloud(r *banzaiTypes.UpdateClusterRequest, c *gin.Context, preCluster banzaiSimpleTypes.ClusterSimple) bool {

	banzaiUtils.LogInfo(banzaiConstants.TagUpdateCluster, "Start updating cluster (google)")

	if r == nil {
		banzaiUtils.LogError(banzaiConstants.TagUpdateCluster, "<nil> update cluster")
		return false
	}

	cluster2Db := banzaiSimpleTypes.ClusterSimple{
		Model:            preCluster.Model,
		Name:             preCluster.Name,
		Location:         preCluster.Location,
		NodeInstanceType: preCluster.NodeInstanceType,
		Cloud:            preCluster.Cloud,
		Google: banzaiSimpleTypes.GoogleClusterSimple{
			Project:       preCluster.Google.Project,
			NodeCount:     r.GoogleNode.Count,
			MasterVersion: r.GoogleMaster.Version,
			NodeVersion:   r.GoogleNode.Version,
		},
	}

	svc, err := GetGoogleServiceClient()
	if err != nil {
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus:  http.StatusInternalServerError,
			JsonKeyMessage: err,
		})
		return false
	}

	cc := GKECluster{
		Name:          preCluster.Name,
		ProjectID:     preCluster.Google.Project,
		Zone:          preCluster.Location,
		MasterVersion: r.GoogleMaster.Version,
		NodeVersion:   r.GoogleNode.Version,
		NodeCount:     int64(r.GoogleNode.Count),
	}

	res, err := callUpdateClusterGoogle(svc, cc)
	if err != nil {
		googleApiErr := getBanzaiErrorFromError(err)
		banzaiUtils.LogError(banzaiConstants.TagUpdateCluster, "Cluster update failed!", googleApiErr)
		SetResponseBodyJson(c, googleApiErr.StatusCode, gin.H{
			JsonKeyStatus:  googleApiErr.StatusCode,
			JsonKeyMessage: googleApiErr.Message,
		})
		return false
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagUpdateCluster, "Cluster update succeeded")
		// updateDb
		if updateClusterInDb(c, cluster2Db) {
			// success update
			SetResponseBodyJson(c, http.StatusCreated, gin.H{
				JsonKeyResourceId: cluster2Db.ID,
				JsonKeyData:       res,
			})
			return true
		} else {
			return false
		}
	}
}

func callUpdateClusterGoogle(svc *gke.Service, cc GKECluster) (*gke.Cluster, error) {

	var updatedCluster *gke.Cluster

	banzaiUtils.LogInfof(banzaiConstants.TagUpdateCluster, "Updating cluster: %#v", cc)
	if cc.NodePoolID == "" {
		cluster, err := svc.Projects.Zones.Clusters.Get(cc.ProjectID, cc.Zone, cc.Name).Context(context.Background()).Do()
		if err != nil {
			banzaiUtils.LogError(banzaiConstants.TagUpdateCluster, "Contains error", err)
			return nil, err
		}
		if cluster.NodePools != nil && len(cluster.NodePools) != 0 {
			cc.NodePoolID = cluster.NodePools[0].Name
		}
	}

	if cc.MasterVersion != "" {
		banzaiUtils.LogInfof(banzaiConstants.TagUpdateCluster, "Updating master to %v version", cc.MasterVersion)
		updateCall, err := svc.Projects.Zones.Clusters.Update(cc.ProjectID, cc.Zone, cc.Name, &gke.UpdateClusterRequest{
			Update: &gke.ClusterUpdate{
				DesiredMasterVersion: cc.MasterVersion,
			},
		}).Context(context.Background()).Do()
		if err != nil {
			return nil, err
		}
		banzaiUtils.LogInfof(banzaiConstants.TagUpdateCluster, "Cluster %s update is called for project %s and zone %s. Status Code %v", cc.Name, cc.ProjectID, cc.Zone, updateCall.HTTPStatusCode)
		if updatedCluster, err = waitForCluster(svc, cc); err != nil {
			banzaiUtils.LogError(banzaiConstants.TagUpdateCluster, "Contains error", err)
			return nil, err
		}
	}

	if cc.NodeVersion != "" {
		banzaiUtils.LogInfof(banzaiConstants.TagUpdateCluster, "Updating node to %v verison", cc.NodeVersion)
		updateCall, err := svc.Projects.Zones.Clusters.NodePools.Update(cc.ProjectID, cc.Zone, cc.Name, cc.NodePoolID, &gke.UpdateNodePoolRequest{
			NodeVersion: cc.NodeVersion,
		}).Context(context.Background()).Do()
		if err != nil {
			banzaiUtils.LogInfof(banzaiConstants.TagUpdateCluster, "Contains error", err)
			return nil, err
		}
		banzaiUtils.LogInfof(banzaiConstants.TagUpdateCluster, "Nodepool %s update is called for project %s, zone %s and cluster %s. Status Code %v", cc.NodePoolID, cc.ProjectID, cc.Zone, cc.Name, updateCall.HTTPStatusCode)
		if err := waitForNodePool(svc, cc); err != nil {
			banzaiUtils.LogError(banzaiConstants.TagUpdateCluster, "Contains error", err)
			return nil, err
		}
	}

	if cc.NodeCount != 0 {
		banzaiUtils.LogInfof(banzaiConstants.TagUpdateCluster, "Updating node size to %v", cc.NodeCount)
		updateCall, err := svc.Projects.Zones.Clusters.NodePools.SetSize(cc.ProjectID, cc.Zone, cc.Name, cc.NodePoolID, &gke.SetNodePoolSizeRequest{
			NodeCount: cc.NodeCount,
		}).Context(context.Background()).Do()
		if err != nil {
			return nil, err
		}
		banzaiUtils.LogInfof(banzaiConstants.TagUpdateCluster, "Nodepool %s size change is called for project %s, zone %s and cluster %s. Status Code %v", cc.NodePoolID, cc.ProjectID, cc.Zone, cc.Name, updateCall.HTTPStatusCode)
		if updatedCluster, err = waitForCluster(svc, cc); err != nil {
			banzaiUtils.LogError(banzaiConstants.TagUpdateCluster, "Contains error", err)
			return nil, err
		}
	}
	return updatedCluster, nil
}

func waitForNodePool(svc *gke.Service, cc GKECluster) error {
	const TAG = "waitForNodePool"
	message := ""
	for {
		nodepool, err := svc.Projects.Zones.Clusters.NodePools.Get(cc.ProjectID, cc.Zone, cc.Name, cc.NodePoolID).Context(context.TODO()).Do()
		if err != nil {
			return err
		}
		if nodepool.Status == statusRunning {
			banzaiUtils.LogInfof(TAG, "Nodepool %v is running", cc.Name)
			return nil
		}
		if nodepool.Status != message {
			banzaiUtils.LogInfof(TAG, "%v nodepool %v", string(nodepool.Status), cc.NodePoolID)
			message = nodepool.Status
		}
		time.Sleep(time.Second * 5)
	}
}

func DeleteGoogleCluster(cs *banzaiSimpleTypes.ClusterSimple, c *gin.Context) bool {

	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Start delete google cluster")

	if cs == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "<nil> cluster")
		return false
	}

	// set google props
	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Load Google props from database")
	database.SelectFirstWhere(&cs.Google, banzaiSimpleTypes.GoogleClusterSimple{ClusterSimpleId: cs.ID})
	gkec := GKECluster{
		ProjectID: cs.Google.Project,
		Name:      cs.Name,
		Zone:      cs.Location,
	}

	if deleteCluster(&gkec, c) {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Delete succeeded")
		return true
	} else {
		banzaiUtils.LogWarn(banzaiConstants.TagGetCluster, "Can't delete cluster from cloud!")
		SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			JsonKeyStatus:     http.StatusBadRequest,
			JsonKeyMessage:    "Can't delete cluster!",
			JsonKeyResourceId: cs.ID,
		})
		return false
	}
}

func deleteCluster(cc *GKECluster, c *gin.Context) bool {

	banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Get Google Service Client")

	svc, err := GetGoogleServiceClient()
	if err != nil {
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus:  http.StatusInternalServerError,
			JsonKeyMessage: err,
		})
		return false
	}
	banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Get Google Service Client succeeded")

	banzaiUtils.LogInfof(banzaiConstants.TagDeleteCluster, "Removing cluster %v from project %v, zone %v", cc.Name, cc.ProjectID, cc.Zone)
	deleteCall, err := svc.Projects.Zones.Clusters.Delete(cc.ProjectID, cc.Zone, cc.Name).Context(context.Background()).Do()
	if err != nil && !strings.Contains(err.Error(), "notFound") {
		banzaiUtils.LogErrorf(banzaiConstants.TagDeleteCluster, "Error during delete %s", err.Error())
		SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			JsonKeyStatus:  http.StatusNotFound,
			JsonKeyMessage: err,
		})
		return false
	} else if err == nil {
		banzaiUtils.LogInfof(banzaiConstants.TagDeleteCluster, "Cluster %v delete is called. Status Code %v", cc.Name, deleteCall.HTTPStatusCode)
		SetResponseBodyJson(c, deleteCall.HTTPStatusCode, gin.H{
			JsonKeyStatus: deleteCall.HTTPStatusCode,
		})
	} else {
		banzaiUtils.LogErrorf(banzaiConstants.TagDeleteCluster, "Cluster %s doesn't exist", cc.Name)
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus:  http.StatusInternalServerError,
			JsonKeyMessage: err,
		})
	}
	os.RemoveAll(cc.TempCredentialPath)
	return true
}

//GetAzureK8SConfig retrieves kubeconfig for Azure AKS
func GetGoogleK8SConfig(cs *banzaiSimpleTypes.ClusterSimple, c *gin.Context) {
	banzaiUtils.LogInfo(banzaiConstants.TagFetchClusterConfig, "Start loading google k8s config")

	if cs == nil {
		banzaiUtils.LogError(banzaiConstants.TagGetCluster, "<nil> cluster")
		return
	}

	// set google props
	banzaiUtils.LogInfo(banzaiConstants.TagFetchClusterConfig, "Load Google props from database")
	database.SelectFirstWhere(&cs.Google, banzaiSimpleTypes.GoogleClusterSimple{ClusterSimpleId: cs.ID})
	config, err := getGoogleKubernetesConfig(cs)
	if err != nil {
		// something went wrong
		banzaiUtils.LogError(banzaiConstants.TagFetchClusterConfig, "Error getting K8S config")
		SetResponseBodyJson(c, err.StatusCode, gin.H{
			JsonKeyStatus: err.StatusCode,
			JsonKeyData:   err.Message,
		})
	} else {
		// get config succeeded
		banzaiUtils.LogInfo(banzaiConstants.TagFetchClusterConfig, "Get k8s config succeeded")
		encodedConfig := base64.StdEncoding.EncodeToString(config)

		if c != nil {
			ctype := c.NegotiateFormat(gin.MIMEPlain, gin.MIMEJSON)
			switch ctype {
			case gin.MIMEJSON:
				SetResponseBodyJson(c, http.StatusOK, gin.H{
					JsonKeyStatus: http.StatusOK,
					JsonKeyData:   encodedConfig,
				})
			default:
				banzaiUtils.LogDebug(banzaiConstants.TagFetchClusterConfig, "Content-Type: ", ctype)
				SetResponseBodyString(c, http.StatusOK, encodedConfig)
			}
		}
	}

}

func getGoogleKubernetesConfig(cs *banzaiSimpleTypes.ClusterSimple) ([]byte, *banzaiTypes.BanzaiResponse) {

	banzaiUtils.LogInfo(banzaiConstants.TagFetchClusterConfig, "Get Google Service Client")
	svc, err := GetGoogleServiceClient()
	if err != nil {
		banzaiUtils.LogErrorf(banzaiConstants.TagFetchClusterConfig, "Error during get service client %v", err)
		return nil, getBanzaiErrorFromError(err)
	}
	banzaiUtils.LogInfo(banzaiConstants.TagFetchClusterConfig, "Get Google Service Client succeeded")

	banzaiUtils.LogInfof(banzaiConstants.TagFetchClusterConfig, "Get google cluster with name %s", cs.Name)
	cl, err := svc.Projects.Zones.Clusters.Get(cs.Google.Project, cs.Location, cs.Name).Context(context.Background()).Do()
	if err != nil {
		banzaiUtils.LogErrorf(banzaiConstants.TagFetchClusterConfig, "Error during get cluster %v", err)
		return nil, getBanzaiErrorFromError(err)
	}

	banzaiUtils.LogInfo(banzaiConstants.TagFetchClusterConfig, "Generate Service Account token")
	serviceAccountToken, err := generateServiceAccountTokenForGke(cl)
	if err != nil {
		banzaiUtils.LogErrorf(banzaiConstants.TagFetchClusterConfig, "Error during generate service account token %v", err)
		return nil, getBanzaiErrorFromError(err)
	}

	finalCl := Cluster{
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

	banzaiUtils.LogInfo(banzaiConstants.TagFetchClusterConfig, "Start save config file")
	config, err := storeConfig(&finalCl, cs.Name)

	return config, nil
}

func generateServiceAccountTokenForGke(cluster *gke.Cluster) (string, error) {
	capem, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
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
			CAData: capem,
		},
		Username: cluster.MasterAuth.Username,
		Password: cluster.MasterAuth.Password,
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}

	return GenerateServiceAccountToken(clientset)
}

// GenerateServiceAccountToken generate a serviceAccountToken for clusterAdmin given a rest clientset
func GenerateServiceAccountToken(clientset *kubernetes.Clientset) (string, error) {
	serviceAccount := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: netesDefault,
		},
	}

	_, err := clientset.CoreV1().ServiceAccounts(defaultNamespace).Create(serviceAccount)
	if err != nil && !errors.IsAlreadyExists(err) {
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
	if _, err = clientset.RbacV1beta1().ClusterRoleBindings().Create(clusterRoleBinding); err != nil && !errors.IsAlreadyExists(err) {
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

// Cluster represents a kubernetes cluster
type Cluster struct {
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

// storeConfig saves config file
func storeConfig(c *Cluster, name string) ([]byte, error) {
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
			Server:                   host,
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
	fileToWrite := fmt.Sprintf("./statestore/%s/config", name)
	if err := writeToFile(data, fileToWrite); err != nil {
		return nil, err
	}
	banzaiUtils.LogInfof(banzaiConstants.TagFetchClusterConfig, "KubeConfig files is saved to %s", fileToWrite)

	return data, nil
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
	Token    string `yaml:"token,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

func writeToFile(data []byte, file string) error {
	if err := os.MkdirAll(filepath.Dir(file), os.ModePerm); err != nil {
		return err
	}
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return ioutil.WriteFile(file, data, 0644)
	}

	tmpfi, err := ioutil.TempFile(filepath.Dir(file), "file.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfi.Name())

	if err = ioutil.WriteFile(tmpfi.Name(), data, 0644); err != nil {
		return err
	}

	if err = tmpfi.Close(); err != nil {
		return err
	}

	if err = os.Remove(file); err != nil {
		return err
	}

	err = os.Rename(tmpfi.Name(), file)
	return err
}

func GetGoogleClusterStatus(cs *banzaiSimpleTypes.ClusterSimple, c *gin.Context) {

	banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "Start get cluster status (google)")

	if cs == nil {
		banzaiUtils.LogError(banzaiConstants.TagGetClusterStatus, "<nil> cluster struct")
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus: http.StatusInternalServerError,
		})
		return
	}

	banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "Load Google props from database")

	// load google props from db
	database.SelectFirstWhere(&cs.Google, banzaiSimpleTypes.GoogleClusterSimple{ClusterSimpleId: cs.ID})

	banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "Get Google Service Client")
	svc, err := GetGoogleServiceClient()
	if err != nil {
		banzaiUtils.LogErrorf(banzaiConstants.TagGetClusterStatus, "Error during get service client %v", err)
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus:  http.StatusInternalServerError,
			JsonKeyMessage: err,
		})
		return
	}
	banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "Get Google Service Client success")

	banzaiUtils.LogInfof(banzaiConstants.TagGetClusterStatus, "Get google cluster with name %s", cs.Name)
	cl, err := svc.Projects.Zones.Clusters.Get(cs.Google.Project, cs.Location, cs.Name).Context(context.Background()).Do()
	if err != nil {
		apiError := getBanzaiErrorFromError(err)
		banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "Error during get cluster info: ", apiError.Message)
		SetResponseBodyJson(c, http.StatusInternalServerError, gin.H{
			JsonKeyStatus:  http.StatusInternalServerError,
			JsonKeyMessage: apiError.Message,
		})
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "Get cluster success")
		banzaiUtils.LogInfof(banzaiConstants.TagGetClusterStatus, "Cluster status is %s", cl.Status)
		var msg string
		var code int
		if statusRunning == cl.Status {
			msg = "Cluster available"
			code = http.StatusOK
		} else {
			msg = "Cluster not ready yet"
			code = http.StatusNoContent
		}
		SetResponseBodyJson(c, code, gin.H{
			JsonKeyStatus:  code,
			JsonKeyMessage: msg,
		})
	}
}
