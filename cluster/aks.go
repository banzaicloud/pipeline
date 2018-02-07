package cluster

import (
	"github.com/banzaicloud/banzai-types/components"
	"github.com/sirupsen/logrus"
	"github.com/banzaicloud/banzai-types/constants"
	"github.com/banzaicloud/pipeline/model"
	bTypes "github.com/banzaicloud/banzai-types/components"
	azureClient "github.com/banzaicloud/azure-aks-client/client"
	azureCluster "github.com/banzaicloud/azure-aks-client/cluster"
	banzaiAzureTypes "github.com/banzaicloud/banzai-types/components/azure"
	"github.com/go-errors/errors"
	"fmt"
	"encoding/base64"
	"os"
	"io/ioutil"
	"github.com/banzaicloud/banzai-types/utils"
)

func CreateAKSClusterFromRequest(request *components.CreateClusterRequest) (*AKSCluster, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagCreateCluster})
	log.Debug("Create ClusterModel struct from the request")
	var cluster AKSCluster

	cluster.modelCluster = &model.ClusterModel{
		Name:             request.Name,
		Location:         request.Location,
		NodeInstanceType: request.NodeInstanceType,
		Cloud:            request.Cloud,
		Azure: model.AzureClusterModel{
			ResourceGroup:     request.Properties.CreateClusterAzure.Node.ResourceGroup,
			AgentCount:        request.Properties.CreateClusterAzure.Node.AgentCount,
			AgentName:         request.Properties.CreateClusterAzure.Node.AgentName,
			KubernetesVersion: request.Properties.CreateClusterAzure.Node.KubernetesVersion,
		},
	}
	return &cluster, nil
}

type AKSCluster struct {
	azureCluster *banzaiAzureTypes.Value //Don't use this directly
	modelCluster *model.ClusterModel
	k8sConfig    *[]byte
	APIEndpoint  string
}

func (c *AKSCluster) CreateCluster() error {

	log := logger.WithFields(logrus.Fields{"action": constants.TagCreateCluster})

	r := azureCluster.CreateClusterRequest{
		Name:              c.modelCluster.Name,
		Location:          c.modelCluster.Location,
		VMSize:            c.modelCluster.NodeInstanceType,
		ResourceGroup:     c.modelCluster.Azure.ResourceGroup,
		AgentCount:        c.modelCluster.Azure.AgentCount,
		AgentName:         c.modelCluster.Azure.AgentName,
		KubernetesVersion: c.modelCluster.Azure.KubernetesVersion,
	}

	// call creation
	createdCluster, err := azureClient.CreateUpdateCluster(r)
	if err != nil {
		// creation failed
		log.Infof("Cluster creation failed! %s", err.Message)
		// todo status code!??
		return errors.New(err.Message)
	} else {
		// creation success
		log.Info("Cluster created successfully!")

		c.azureCluster = &createdCluster.Value

		// todo save cluster to DB before polling

		// polling cluster
		pollingResult, err := azureClient.PollingCluster(r.Name, r.ResourceGroup)
		if err != nil {
			// polling error
			// todo status code!??
			return errors.New(err.Message)
		} else {
			log.Info("Cluster is ready...")
			c.azureCluster = &pollingResult.Value
			return nil
		}
	}
}

// TODO same as AWS, GKE, maybe put common function
func (c *AKSCluster) Persist() error {
	db := model.GetDB()
	err := db.Save(c.modelCluster).Error
	if err != nil {
		return err
	}
	return nil
}

func (c *AKSCluster) GetK8sConfig() (*[]byte, error) {
	if c.k8sConfig != nil {
		return c.k8sConfig, nil
	}

	database := model.GetDB()
	database.Where(model.AzureClusterModel{ClusterModelId: c.modelCluster.ID}).First(&c.modelCluster.Azure)
	config, err := azureClient.GetClusterConfig(c.modelCluster.Name, c.modelCluster.Azure.ResourceGroup, "clusterUser")
	if err != nil {
		// TODO status code !?
		return nil, errors.New(err.Message)
	} else {

		// TODO save or not save, this is the question
		//// get config succeeded
		//localDir := fmt.Sprintf("./statestore/%s", c.modelCluster.Name)
		//if err := writeConfig2File(localDir, config); err != nil {
		//	return nil, err
		//}

		//kubeConfigPath, err := getKubeConfigPath(localDir)
		//if err != nil {
		//	return nil, err
		//}

		// TODO this needs to prometheus
		// writeKubernetesKeys(kubeConfigPath, localDir)

		log.Info("Get k8s config succeeded")
		decodedConfig, err := base64.StdEncoding.DecodeString(config.Properties.KubeConfig)
		if err != nil {
			return nil, err
		}

		c.k8sConfig = &decodedConfig

		return &decodedConfig, nil

	}

}

func (c *AKSCluster) GetName() string {
	return c.modelCluster.Name
}

func (c *AKSCluster) GetType() string {
	return c.modelCluster.Cloud
}

func (c *AKSCluster) GetStatus() (*bTypes.GetClusterStatusResponse, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagGetClusterStatus})

	log.Info("Load azure props from database")

	// load azure props from db
	database := model.GetDB()
	database.Where(model.AzureClusterModel{ClusterModelId: c.modelCluster.ID}).First(&c.modelCluster.Azure)
	resp, err := azureClient.GetCluster(c.modelCluster.Name, c.modelCluster.Azure.ResourceGroup)
	if err != nil {
		return nil, errors.New(err)
	} else {
		log.Info("Get cluster success")
		stage := resp.Value.Properties.ProvisioningState
		log.Info("Cluster stage is", stage)
		if stage == "Succeeded" {
			response := &components.GetClusterStatusResponse{
				Name:             c.modelCluster.Name,
				Location:         c.modelCluster.Location,
				Cloud:            c.modelCluster.Cloud,
				NodeInstanceType: c.modelCluster.NodeInstanceType,
			}

			return response, nil
		} else {
			return nil, errors.New("Cluster not ready yet")
		}
	}
}

func (c *AKSCluster) DeleteCluster() error {
	log := logger.WithFields(logrus.Fields{"action": constants.TagDeleteCluster})

	// set azure props
	database := model.GetDB()
	database.Where(model.AzureClusterModel{ClusterModelId: c.modelCluster.ID}).First(&c.modelCluster.Azure)

	res, isSuccess := azureClient.DeleteCluster(c.modelCluster.Name, c.modelCluster.Azure.ResourceGroup)
	if isSuccess {
		log.Info("Delete succeeded")
		return nil
	} else {
		// todo status code !?
		return errors.New(res.Message)
	}
}

func (c *AKSCluster) UpdateCluster(request *bTypes.UpdateClusterRequest) error {
	log := logger.WithFields(logrus.Fields{"action": constants.TagUpdateCluster})

	ccr := azureCluster.CreateClusterRequest{
		Name:              c.modelCluster.Name,
		Location:          c.modelCluster.Location,
		VMSize:            c.modelCluster.NodeInstanceType,
		ResourceGroup:     c.modelCluster.Azure.ResourceGroup,
		AgentCount:        c.modelCluster.Azure.AgentCount,
		AgentName:         c.modelCluster.Azure.AgentName,
		KubernetesVersion: c.modelCluster.Azure.KubernetesVersion,
	}

	updatedCluster, err := azureClient.CreateUpdateCluster(ccr)
	if err != nil {
		return errors.New(err.Message)
	} else {
		log.Info("Cluster update succeeded")
		//Update AWS model
		log.Info("Create updated model")
		updateCluster := &model.ClusterModel{
			Model:            c.modelCluster.Model,
			Name:             c.modelCluster.Name,
			Location:         c.modelCluster.Location,
			NodeInstanceType: c.modelCluster.NodeInstanceType,
			Cloud:            c.modelCluster.Cloud,
			Amazon: model.AmazonClusterModel{
				NodeSpotPrice:      c.modelCluster.Amazon.NodeSpotPrice,
				NodeMinCount:       request.UpdateClusterAmazon.MinCount,
				NodeMaxCount:       request.UpdateClusterAmazon.MaxCount,
				NodeImage:          c.modelCluster.Amazon.NodeImage,
				MasterInstanceType: c.modelCluster.Amazon.MasterInstanceType,
				MasterImage:        c.modelCluster.Amazon.MasterImage,
			},
		}
		c.modelCluster = updateCluster
		c.azureCluster = &updatedCluster.Value
		return nil
	}

}

func (c *AKSCluster) GetID() uint {
	return c.modelCluster.ID
}

func (c *AKSCluster) GetModel() *model.ClusterModel {
	return c.modelCluster
}

func writeConfig2File(path string, config *banzaiAzureTypes.Config) error {

	if config == nil {
		// log.Warn("config is nil")
		return errors.New("config is null")
	}

	decodedConfig, _ := base64.StdEncoding.DecodeString(config.Properties.KubeConfig)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.Mkdir(path, 0777); err != nil {
			// log.Warn("error during write to file: %s", err.Error())
			return err
		}
	}

	if err := ioutil.WriteFile(fmt.Sprintf("%s/config", path), decodedConfig, 0777); err != nil {
		// log.Warn("error during write to file: %s", err.Error())
		return err
	} else {
		log.Info("write config file succeeded")
	}
	return nil
}

func CreateAKSClusterFromModel(clusterModel *model.ClusterModel) (*AKSCluster, error) {
	log := logger.WithFields(logrus.Fields{"action": constants.TagGetCluster})
	log.Debug("Create ClusterModel struct from the request")
	aksCluster := AKSCluster{
		modelCluster: clusterModel,
	}
	return &aksCluster, nil
}

func AddDefaultsAzureUpdate(r *components.UpdateClusterRequest, existsCluster *model.ClusterModel) {

	// ---- [ Node check ] ---- //
	if r.UpdateAzureNode == nil {
		log.Info(constants.TagValidateCreateCluster, "'node' field is empty. Load it from stored data.")
		r.UpdateAzureNode = &banzaiAzureTypes.UpdateAzureNode{
			AgentCount: existsCluster.Azure.AgentCount,
		}
	}

	// ---- [ Node - Agent count check] ---- //
	if r.AgentCount == 0 {
		def := existsCluster.Azure.AgentCount
		log.Info(constants.TagValidateCreateCluster, "Node agentCount set to default value: ", def)
		r.AgentCount = def
	}

}

func IsUpdateRequestDifferentAzure(r *components.UpdateClusterRequest, existsCluster *model.ClusterModel) error {
	// create update request struct with the stored data to check equality
	preCl := &banzaiAzureTypes.UpdateClusterAzure{
		UpdateAzureNode: &banzaiAzureTypes.UpdateAzureNode{
			AgentCount: existsCluster.Azure.AgentCount,
		},
	}

	log.Info("Check stored & updated cluster equals")

	// check equality
	return utils.IsDifferent(r, preCl, constants.TagValidateUpdateCluster)
}
