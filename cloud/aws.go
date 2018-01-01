package cloud

import (
	"fmt"

	"github.com/banzaicloud/pipeline/notify"
	"github.com/gin-gonic/gin"
	"github.com/go-errors/errors"
	"github.com/kris-nova/kubicorn/apis/cluster"
	"github.com/kris-nova/kubicorn/cutil/kubeadm"
	"github.com/kris-nova/kubicorn/cutil/uuid"
	"net/http"

	banzaiUtils "github.com/banzaicloud/banzai-types/utils"
	banzaiConstants "github.com/banzaicloud/banzai-types/constants"
	banzaiTypes "github.com/banzaicloud/banzai-types/components"
	banzaiSimpleTypes "github.com/banzaicloud/banzai-types/components/database"
	"github.com/banzaicloud/banzai-types/database"
)

// GetAWSCluster creates *cluster.Cluster from ClusterSimple struct
func GetAWSCluster(cs *banzaiSimpleTypes.ClusterSimple) *cluster.Cluster {
	uuidSuffix := uuid.TimeOrderedUUID()
	return &cluster.Cluster{
		Name:     cs.Name,
		Cloud:    cluster.CloudAmazon,
		Location: cs.Location,
		SSH: &cluster.SSH{
			Name:          cs.Name + "-" + uuidSuffix,
			PublicKeyPath: "/.ssh/id_rsa.pub",
			User:          "ubuntu",
		},
		KubernetesAPI: &cluster.KubernetesAPI{
			Port: "443",
		},
		Network: &cluster.Network{
			Type:       cluster.NetworkTypePublic,
			CIDR:       "10.0.0.0/16",
			InternetGW: &cluster.InternetGW{},
		},
		Values: &cluster.Values{
			ItemMap: map[string]string{
				"INJECTEDTOKEN": kubeadm.GetRandomToken(),
			},
		},
		ServerPools: []*cluster.ServerPool{
			{
				Type:     cluster.ServerPoolTypeMaster,
				Name:     fmt.Sprintf("%s.master", cs.Name),
				MinCount: 1,
				MaxCount: 1,
				Image:    cs.Amazon.MasterImage, //"ami-835b4efa"
				Size:     cs.NodeInstanceType,
				BootstrapScripts: []string{
					"https://raw.githubusercontent.com/banzaicloud/banzai-charts/master/stable/pipeline/bootstrap/amazon_k8s_ubuntu_16.04_master_pipeline.sh",
				},
				InstanceProfile: &cluster.IAMInstanceProfile{
					Name: fmt.Sprintf("%s-KubicornMasterInstanceProfile", cs.Name),
					Role: &cluster.IAMRole{
						Name: fmt.Sprintf("%s-KubicornMasterRole", cs.Name),
						Policies: []*cluster.IAMPolicy{
							{
								Name: "MasterPolicy",
								Document: `{
                  "Version": "2012-10-17",
                  "Statement": [
                     {
                        "Effect": "Allow",
                        "Action": [
                           "ec2:*",
                           "elasticloadbalancing:*",
                           "ecr:GetAuthorizationToken",
                           "ecr:BatchCheckLayerAvailability",
                           "ecr:GetDownloadUrlForLayer",
                           "ecr:GetRepositoryPolicy",
                           "ecr:DescribeRepositories",
                           "ecr:ListImages",
                           "ecr:BatchGetImage",
                           "autoscaling:DescribeAutoScalingGroups",
                           "autoscaling:UpdateAutoScalingGroup"
                        ],
                        "Resource": "*"
                     }
                  ]
								}`,
							},
						},
					},
				},
				Subnets: []*cluster.Subnet{
					{
						Name:     fmt.Sprintf("%s.master", cs.Name),
						CIDR:     "10.0.0.0/24",
						Location: cs.Location,
					},
				},

				Firewalls: []*cluster.Firewall{
					{
						Name: fmt.Sprintf("%s.master-external-%s", cs.Name, uuidSuffix),
						IngressRules: []*cluster.IngressRule{
							{
								IngressFromPort: "22",
								IngressToPort:   "22",
								IngressSource:   "0.0.0.0/0",
								IngressProtocol: "tcp",
							},
							{
								IngressFromPort: "443",
								IngressToPort:   "443",
								IngressSource:   "0.0.0.0/0",
								IngressProtocol: "tcp",
							},
							{
								IngressFromPort: "30080",
								IngressToPort:   "30080",
								IngressSource:   "0.0.0.0/0",
								IngressProtocol: "tcp",
							},
							{
								IngressFromPort: "0",
								IngressToPort:   "65535",
								IngressSource:   "10.0.100.0/24",
								IngressProtocol: "-1",
							},
						},
					},
				},
			},
			{
				Type:     cluster.ServerPoolTypeNode,
				Name:     fmt.Sprintf("%s.node", cs.Name),
				MinCount: cs.Amazon.NodeMinCount,
				MaxCount: cs.Amazon.NodeMaxCount,
				Image:    cs.Amazon.NodeImage, //"ami-835b4efa"
				Size:     cs.NodeInstanceType,
				AwsConfiguration: &cluster.AwsConfiguration{
					SpotPrice: cs.Amazon.NodeSpotPrice,
				},
				BootstrapScripts: []string{
					"https://raw.githubusercontent.com/banzaicloud/banzai-charts/master/stable/pipeline/bootstrap/amazon_k8s_ubuntu_16.04_node_pipeline.sh",
				},
				InstanceProfile: &cluster.IAMInstanceProfile{
					Name: fmt.Sprintf("%s-KubicornNodeInstanceProfile", cs.Name),
					Role: &cluster.IAMRole{
						Name: fmt.Sprintf("%s-KubicornNodeRole", cs.Name),
						Policies: []*cluster.IAMPolicy{
							{
								Name: "NodePolicy",
								Document: `{
                  "Version": "2012-10-17",
                  "Statement": [
                     {
                        "Effect": "Allow",
                        "Action": [
            							"ec2:Describe*",
            							"ecr:GetAuthorizationToken",
            							"ecr:BatchCheckLayerAvailability",
            							"ecr:GetDownloadUrlForLayer",
            							"ecr:GetRepositoryPolicy",
            							"ecr:DescribeRepositories",
            							"ecr:ListImages",
            							"ecr:BatchGetImage"
                        ],
                        "Resource": "*"
                     }
                  ]
								}`,
							},
						},
					},
				},
				Subnets: []*cluster.Subnet{
					{
						Name:     fmt.Sprintf("%s.node", cs.Name),
						CIDR:     "10.0.100.0/24",
						Location: cs.Location,
					},
				},
				Firewalls: []*cluster.Firewall{
					{
						Name: fmt.Sprintf("%s.node-external-%s", cs.Name, uuidSuffix),
						IngressRules: []*cluster.IngressRule{
							{
								IngressFromPort: "22",
								IngressToPort:   "22",
								IngressSource:   "0.0.0.0/0",
								IngressProtocol: "tcp",
							},
							{
								IngressFromPort: "0",
								IngressToPort:   "65535",
								IngressSource:   "10.0.0.0/24",
								IngressProtocol: "-1",
							},
						},
					},
				},
			},
		},
	}
}

func GetAmazonClusterStatus(cs *banzaiSimpleTypes.ClusterSimple, c *gin.Context) {

	banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "Start get cluster status (amazon)")

	if cs == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "<nil> cluster")
		return
	}

	// --- [ Get cluster with stored data ] --- //
	cl, err := GetClusterWithDbCluster(cs, c)
	if err != nil {
		banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "Error during read cluster")
		return
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, "Read cluster succeeded")
	}

	isAvailable, _ := IsKubernetesClusterAvailable(cl)
	if isAvailable {
		msg := "Kubernetes cluster available"
		banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, msg)
		SetResponseBodyJson(c, http.StatusOK, gin.H{
			JsonKeyStatus:  http.StatusOK,
			JsonKeyMessage: msg,
		})
	} else {
		msg := "Kubernetes cluster not ready yet"
		banzaiUtils.LogInfo(banzaiConstants.TagGetClusterStatus, msg)
		SetResponseBodyJson(c, http.StatusNoContent, gin.H{
			JsonKeyStatus:  http.StatusNoContent,
			JsonKeyMessage: msg,
		})
	}

}

// UpdateClusterAmazonInCloud updates Amazon cluster in cloud
func UpdateClusterAmazonInCloud(r *banzaiTypes.UpdateClusterRequest, c *gin.Context, preCluster banzaiSimpleTypes.ClusterSimple) bool {

	banzaiUtils.LogInfo(banzaiConstants.TagUpdateCluster, "Start updating cluster (amazon)")

	if r == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagUpdateCluster, "<nil> update cluster request")
		return false
	}

	cluster2Db := banzaiSimpleTypes.ClusterSimple{
		Model:            preCluster.Model,
		Name:             preCluster.Name,
		Location:         preCluster.Location,
		NodeInstanceType: preCluster.NodeInstanceType,
		Cloud:            r.Cloud,
		Amazon: banzaiSimpleTypes.AmazonClusterSimple{
			NodeSpotPrice:      preCluster.Amazon.NodeSpotPrice,
			NodeMinCount:       r.UpdateClusterAmazon.MinCount,
			NodeMaxCount:       r.UpdateClusterAmazon.MaxCount,
			NodeImage:          preCluster.Amazon.NodeImage,
			MasterInstanceType: preCluster.Amazon.MasterInstanceType,
			MasterImage:        preCluster.Amazon.MasterImage,
		},
	}

	banzaiUtils.LogInfo(banzaiConstants.TagUpdateCluster, "Call amazon to updating")

	if _, err := UpdateClusterAws(cluster2Db); err != nil {
		banzaiUtils.LogWarn(banzaiConstants.TagUpdateCluster, "Can't update cluster in the cloud!", err)

		SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			JsonKeyStatus:     http.StatusBadRequest,
			JsonKeyMessage:    "Can't update cluster in the cloud!",
			JsonKeyResourceId: cluster2Db.ID,
			JsonKeyError:      err,
		})

		return false
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagUpdateCluster, "Cluster updated in the cloud!")
		if updateClusterInDb(c, cluster2Db) {
			SetResponseBodyJson(c, http.StatusCreated, gin.H{
				JsonKeyStatus:     http.StatusCreated,
				JsonKeyMessage:    "Cluster updated successfully!",
				JsonKeyResourceId: cluster2Db.ID,
			})

			return true
		}

		return false
	}

}

// CreateClusterAmazon creates amazon cluster in cloud
func CreateClusterAmazon(request *banzaiTypes.CreateClusterRequest, c *gin.Context) (bool, *cluster.Cluster) {

	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Create ClusterSimple struct from the request")

	if request == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "<nil> create request")
		return false, nil
	}

	cluster2Db := banzaiSimpleTypes.ClusterSimple{
		Name:             request.Name,
		Location:         request.Location,
		NodeInstanceType: request.NodeInstanceType,
		Cloud:            request.Cloud,
		Amazon: banzaiSimpleTypes.AmazonClusterSimple{
			NodeSpotPrice:      request.Properties.CreateClusterAmazon.Node.SpotPrice,
			NodeMinCount:       request.Properties.CreateClusterAmazon.Node.MinCount,
			NodeMaxCount:       request.Properties.CreateClusterAmazon.Node.MaxCount,
			NodeImage:          request.Properties.CreateClusterAmazon.Node.Image,
			MasterInstanceType: request.Properties.CreateClusterAmazon.Master.InstanceType,
			MasterImage:        request.Properties.CreateClusterAmazon.Master.Image,
		},
	}

	banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Call amazon with the request")

	// create aws cluster
	if createdCluster, err := CreateCluster(cluster2Db); err != nil {
		// creation failed
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Cluster creation failed!", err.Error())
		SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			JsonKeyMessage: "Could not launch cluster!",
			JsonKeyName:    cluster2Db.Name,
			JsonKeyError:   err,
		})
		return false, nil
	} else {
		// cluster creation success
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Cluster created successfully!")
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Save created cluster into database")

		// save db
		if err := database.Save(&cluster2Db).Error; err != nil {
			DbSaveFailed(c, err, cluster2Db.Name)
			return false, nil
		}

		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Database save succeeded")
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "Create response")
		SetResponseBodyJson(c, http.StatusCreated, gin.H{
			JsonKeyStatus:     http.StatusCreated,
			JsonKeyMessage:    "Cluster created successfully!",
			JsonKeyResourceId: cluster2Db.ID,
			JsonKeyName:       cluster2Db.Name,
			JsonKeyIp:         createdCluster.KubernetesAPI.Endpoint,
		})

		return true, createdCluster
	}

}

//GetClusterWithDbCluster legacy AWS
func GetClusterWithDbCluster(cs *banzaiSimpleTypes.ClusterSimple, c *gin.Context) (*cluster.Cluster, error) {

	if cs == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagCreateCluster, "<nil> create request")
		return nil, errors.New("Error read cluster")
	}

	cl, err := GetKubicornCluster(cs)
	if err != nil {
		errorMsg := fmt.Sprintf("Error read cluster: %s", err)
		banzaiUtils.LogWarn(banzaiConstants.TagGetCluster, errorMsg)
		SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			JsonKeyStatus:  http.StatusNotFound,
			JsonKeyMessage: errorMsg,
		})
		return nil, err
	}
	banzaiUtils.LogDebug(banzaiConstants.TagGetCluster, "Get cluster succeeded:", cl)
	return cl, nil
}

// GetKubicornCluster based on ClusterSimple object
// This will read the persisted Kubicorn cluster format
func GetKubicornCluster(cs *banzaiSimpleTypes.ClusterSimple) (*cluster.Cluster, error) {

	if cs == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "<nil> cluster")
		return nil, errors.New("Read Kubicorn cluster failed")
	}

	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Read persisted Kubicorn cluster format")
	clust, err := ReadCluster(*cs)
	if err != nil {
		banzaiUtils.LogWarn(banzaiConstants.TagGetCluster, "Read Kubicorn cluster failed", err)
		return nil, err
	}
	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Cluster read succeeded")
	return clust, nil
}

//GetCluster
func GetCluster(c *gin.Context) (*cluster.Cluster, error) {
	cl, err := GetClusterFromDB(c)
	if err != nil {
		return nil, err
	}
	return GetClusterWithDbCluster(cl, c)
}

// ReadClusterAmazon load amazon props from cloud to list clusters
func ReadClusterAmazon(cs *banzaiSimpleTypes.ClusterSimple) *ClusterRepresentation {

	if cs == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "<nil> cluster")
		return nil
	}

	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Read aws cluster with", cs.ID, "id")
	c, err := ReadCluster(*cs)
	if err == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Read aws cluster success")
		clust := ClusterRepresentation{
			Id:        cs.ID,
			Name:      cs.Name,
			CloudType: banzaiConstants.Amazon,
			AmazonRepresentation: &AmazonRepresentation{
				Ip: c.KubernetesAPI.Endpoint,
			},
		}
		return &clust
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Something went wrong under read: ", err.Error())
	}
	return nil
}

// GetClusterInfoAmazon fetches amazon cluster props
func GetClusterInfoAmazon(cs *banzaiSimpleTypes.ClusterSimple, c *gin.Context) {

	banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Get cluster info (amazon)")

	if cs == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "<nil> cluster")
		return
	}

	cl, err := GetClusterWithDbCluster(cs, c)
	if err != nil {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Error during fetch amazon cluster: ", err.Error())
		return
	} else {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "Get cluster info succeeded")
	}

	isAvailable, _ := IsKubernetesClusterAvailable(cl)
	SetResponseBodyJson(c, http.StatusOK, gin.H{
		JsonKeyStatus:    http.StatusOK,
		JsonKeyData:      cl,
		JsonKeyAvailable: isAvailable,
		JsonKeyIp:        cl.KubernetesAPI.Endpoint,
	})
}

// DeleteAmazonCluster deletes cluster from amazon
func DeleteAmazonCluster(cs *banzaiSimpleTypes.ClusterSimple, c *gin.Context) bool {

	banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, "Start delete amazon cluster")

	if cs == nil {
		banzaiUtils.LogInfo(banzaiConstants.TagGetCluster, "<nil> cluster")
		return false
	}

	if _, err := DeleteClusterAmazon(cs); err != nil {
		// delete failed
		banzaiUtils.LogWarn(banzaiConstants.TagDeleteCluster, "Can't delete cluster from cloud!", err)

		SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			JsonKeyStatus:     http.StatusBadRequest,
			JsonKeyMessage:    "Can't delete cluster!",
			JsonKeyResourceId: cs.ID,
			JsonKeyError:      err,
		})
		return false
	} else {
		// delete success
		msg := "Cluster deleted from the cloud!"
		banzaiUtils.LogInfo(banzaiConstants.TagDeleteCluster, msg)
		notify.SlackNotify(msg)

		SetResponseBodyJson(c, http.StatusCreated, gin.H{
			JsonKeyStatus:     http.StatusCreated,
			JsonKeyMessage:    "Cluster deleted successfully!",
			JsonKeyResourceId: cs.ID,
		})
		return true
	}

}
