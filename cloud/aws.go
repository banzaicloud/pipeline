package cloud

import (
	"fmt"

	"github.com/kris-nova/kubicorn/apis/cluster"
	"github.com/kris-nova/kubicorn/cutil/kubeadm"
	"github.com/kris-nova/kubicorn/cutil/uuid"
	"github.com/sirupsen/logrus"
	"github.com/gin-gonic/gin"
	"net/http"
	"github.com/jinzhu/gorm"
	"github.com/banzaicloud/pipeline/notify"
)

const (
	amazonDefaultNodeImage          = "ami-bdba13c4"
	amazonDefaultMasterImage        = "ami-bdba13c4"
	amazonDefaultMasterInstanceType = "m4.xlarge"
	amazonDefaultNodeMinCount       = 1
	amazonDefaultNodeMaxCount       = 1
	amazonDefaultNodeSpotPrice      = "0.2"
)

// GetAWSCluster creates *cluster.Cluster from ClusterSimple struct
func (cs ClusterSimple) GetAWSCluster() *cluster.Cluster {
	uuid_suffix := uuid.TimeOrderedUUID()
	return &cluster.Cluster{
		Name:     cs.Name,
		Cloud:    cluster.CloudAmazon,
		Location: cs.Location,
		SSH: &cluster.SSH{
			Name:          cs.Name + "-" + uuid_suffix,
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
						Name: fmt.Sprintf("%s.master-external-%s", cs.Name, uuid_suffix),
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
						Name: fmt.Sprintf("%s.node-external-%s", cs.Name, uuid_suffix),
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

type CreateClusterAmazon struct {
	Node   *CreateAmazonNode   `json:"node"`
	Master *CreateAmazonMaster `json:"master"`
}

type UpdateClusterAmazon struct {
	*UpdateAmazonNode `json:"node"`
}

type CreateAmazonNode struct {
	SpotPrice string `json:"spotPrice"`
	MinCount  int    `json:"minCount"`
	MaxCount  int    `json:"maxCount"`
	Image     string `json:"image"`
}

type UpdateAmazonNode struct {
	MinCount int `json:"minCount"`
	MaxCount int `json:"maxCount"`
}

type CreateAmazonMaster struct {
	InstanceType string `json:"instanceType"`
	Image        string `json:"image"`
}

type AmazonClusterSimple struct {
	ClusterSimpleId    uint `gorm:"primary_key"`
	NodeSpotPrice      string
	NodeMinCount       int
	NodeMaxCount       int
	NodeImage          string
	MasterInstanceType string
	MasterImage        string
}

// TableName sets AmazonClusterSimple's table name
func (AmazonClusterSimple) TableName() string {
	return tableNameAmazonProperties
}

// Validate validates amazon cluster create request
func (amazon *CreateClusterAmazon) Validate(log *logrus.Logger) (bool, string) {

	if amazon == nil {
		msg := "Required field 'amazon' is empty."
		log.Info(msg)
		return false, msg
	}

	// ---- [ Master check ] ---- //
	if amazon.Master == nil {
		msg := "Required field 'master' is empty."
		log.Info(msg)
		return false, msg
	}

	if len(amazon.Master.Image) == 0 {
		log.Info("Master image set to default value: ", amazonDefaultMasterImage)
		amazon.Master.Image = amazonDefaultMasterImage
	}

	if len(amazon.Master.InstanceType) == 0 {
		log.Info("Master instance type set to default value: ", amazonDefaultMasterInstanceType)
		amazon.Master.InstanceType = amazonDefaultMasterInstanceType
	}

	// ---- [ Node check ] ---- //
	if amazon.Node == nil {
		msg := "Required field 'node' is empty."
		log.Info(msg)
		return false, msg
	}

	if len(amazon.Node.Image) == 0 {
		log.Info("Node image set to default value: ", amazonDefaultNodeImage)
		amazon.Node.Image = amazonDefaultNodeImage
	}

	if amazon.Node.MinCount == 0 {
		log.Info("Node minCount set to default value: ", amazonDefaultNodeMinCount)
		amazon.Node.MinCount = amazonDefaultNodeMinCount
	}

	if amazon.Node.MaxCount == 0 {
		log.Info("Node maxCount set to default value: ", amazonDefaultNodeMaxCount)
		amazon.Node.MaxCount = amazonDefaultNodeMaxCount
	}

	if amazon.Node.MaxCount < amazon.Node.MinCount {
		log.Info("Node maxCount is lower than minCount")
		return false, "maxCount must be greater than mintCount"
	}

	if len(amazon.Node.SpotPrice) == 0 {
		log.Info("Node spot price set to default value: ", amazonDefaultNodeSpotPrice)
		amazon.Node.SpotPrice = amazonDefaultNodeSpotPrice
	}

	return true, ""
}

// ValidateAmazonRequest validates the update request (only amazon part). If any of the fields is missing, the method fills
// with stored data.
func (r *UpdateClusterRequest) ValidateAmazonRequest(log *logrus.Logger, defaultValue ClusterSimple) (bool, string) {

	// reset azure fields
	r.UpdateClusterAzure = nil

	defAmazonNode := &UpdateAmazonNode{
		MinCount: defaultValue.Amazon.NodeMinCount,
		MaxCount: defaultValue.Amazon.NodeMaxCount,
	}

	// ---- [ Amazon field check ] ---- //
	if r.UpdateClusterAmazon == nil {
		log.Info("'amazon' field is empty, Load it from stored data.")
		r.UpdateClusterAmazon = &UpdateClusterAmazon{
			UpdateAmazonNode: defAmazonNode,
		}
	}

	// ---- [ Node check ] ---- //
	if r.UpdateAmazonNode == nil {
		log.Info("'node' field is empty. Fill from stored data")
		r.UpdateAmazonNode = defAmazonNode
	}

	// ---- [ Node min count check ] ---- //
	if r.UpdateAmazonNode.MinCount == 0 {
		defMinCount := defaultValue.Amazon.NodeMinCount
		log.Info("Node minCount set to default value: ", defMinCount)
		r.UpdateAmazonNode.MinCount = defMinCount
	}

	// ---- [ Node max count check ] ---- //
	if r.UpdateAmazonNode.MaxCount == 0 {
		defMaxCount := defaultValue.Amazon.NodeMaxCount
		log.Info("Node maxCount set to default value: ", defMaxCount)
		r.UpdateAmazonNode.MaxCount = defMaxCount
	}

	// ---- [ Node max count > min count check ] ---- //
	if r.UpdateAmazonNode.MaxCount < r.UpdateAmazonNode.MinCount {
		log.Info("Node maxCount is lower than minCount")
		return false, "maxCount must be greater than mintCount"
	}

	// create update request struct with the stored data to check equality
	preCl := &UpdateClusterRequest{
		Cloud: defaultValue.Cloud,
		UpdateProperties: UpdateProperties{
			UpdateClusterAmazon: &UpdateClusterAmazon{
				UpdateAmazonNode: defAmazonNode,
			},
		},
	}

	// check equality
	return isUpdateEqualsWithStoredCluster(r, preCl, log)
}

func (cs *ClusterSimple) GetAmazonClusterStatus(c *gin.Context, log *logrus.Logger) {
	cl, err := cs.GetClusterWithDbCluster(c, log)
	if err != nil {
		log.Info("Error during read cluster from db")
		return
	}
	isAvailable, _ := IsKubernetesClusterAvailable(cl)
	if isAvailable {
		msg := "Kubernetes cluster available"
		log.Info(msg)
		SetResponseBodyJson(c, http.StatusOK, gin.H{
			JsonKeyStatus:  http.StatusOK,
			JsonKeyMessage: msg,
		})
	} else {
		msg := "Kubernetes cluster not ready yet"
		log.Info(msg)
		SetResponseBodyJson(c, http.StatusNoContent, gin.H{
			JsonKeyStatus:  http.StatusNoContent,
			JsonKeyMessage: msg,
		})
	}

}

// UpdateClusterAmazonInCloud updates amazon cluster in cloud
func (r UpdateClusterRequest) UpdateClusterAmazonInCloud(c *gin.Context, db *gorm.DB, log *logrus.Logger, preCluster ClusterSimple) bool {

	cluster2Db := ClusterSimple{
		Model:            preCluster.Model,
		Name:             preCluster.Name,
		Location:         preCluster.Location,
		NodeInstanceType: preCluster.NodeInstanceType,
		Cloud:            r.Cloud,
		Amazon: AmazonClusterSimple{
			NodeSpotPrice:      preCluster.Amazon.NodeSpotPrice,
			NodeMinCount:       r.UpdateClusterAmazon.MinCount,
			NodeMaxCount:       r.UpdateClusterAmazon.MaxCount,
			NodeImage:          preCluster.Amazon.NodeImage,
			MasterInstanceType: preCluster.Amazon.MasterInstanceType,
			MasterImage:        preCluster.Amazon.MasterImage,
		},
	}

	if _, err := UpdateClusterAws(cluster2Db); err != nil {
		log.Warning("Can't update cluster in the cloud!", err)

		SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			JsonKeyStatus:     http.StatusBadRequest,
			JsonKeyMessage:    "Can't update cluster in the cloud!",
			JsonKeyResourceId: cluster2Db.ID,
			JsonKeyError:      err,
		})

		return false
	} else {
		log.Info("Cluster updated in the cloud!")
		if updateClusterInDb(c, db, log, cluster2Db) {
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
func (request CreateClusterRequest) CreateClusterAmazon(c *gin.Context, db *gorm.DB, log *logrus.Logger) (bool, *cluster.Cluster) {

	cluster2Db := ClusterSimple{
		Name:             request.Name,
		Location:         request.Location,
		NodeInstanceType: request.NodeInstanceType,
		Cloud:            request.Cloud,
		Amazon: AmazonClusterSimple{
			NodeSpotPrice:      request.Properties.CreateClusterAmazon.Node.SpotPrice,
			NodeMinCount:       request.Properties.CreateClusterAmazon.Node.MinCount,
			NodeMaxCount:       request.Properties.CreateClusterAmazon.Node.MaxCount,
			NodeImage:          request.Properties.CreateClusterAmazon.Node.Image,
			MasterInstanceType: request.Properties.CreateClusterAmazon.Master.InstanceType,
			MasterImage:        request.Properties.CreateClusterAmazon.Master.Image,
		},
	}

	// create aws cluster
	if createdCluster, err := CreateCluster(cluster2Db); err != nil {
		// creation failed
		log.Info("Cluster creation failed!", err)
		SetResponseBodyJson(c, http.StatusBadRequest, gin.H{
			JsonKeyMessage: "Could not launch cluster!",
			JsonKeyName:    cluster2Db.Name,
			JsonKeyError:   err,
		})
		return false, nil
	} else {
		// cluster creation success
		log.Info("Cluster created successfully!")

		// save db
		if err := db.Save(&cluster2Db).Error; err != nil {
			DbSaveFailed(c, log, err, cluster2Db.Name)
			return false, nil
		}

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

func (cs *ClusterSimple) GetClusterWithDbCluster(c *gin.Context, log *logrus.Logger) (*cluster.Cluster, error) {
	cl, err := cs.GetKubicornCluster(log)
	if err != nil {
		errorMsg := fmt.Sprintf("Error read cluster: %s", err)
		SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			JsonKeyStatus:  http.StatusNotFound,
			JsonKeyMessage: errorMsg,
		})
		return nil, err
	}
	return cl, nil
}

// GetCluster based on ClusterSimple object
// This will read the persisted Kubicorn cluster format
func (cs *ClusterSimple) GetKubicornCluster(log *logrus.Logger) (*cluster.Cluster, error) {
	clust, err := ReadCluster(*cs)
	if err != nil {
		return nil, err
	}
	log.Info("Cluster read successful")
	return clust, nil
}

func GetCluster(c *gin.Context, db *gorm.DB, log *logrus.Logger) (*cluster.Cluster, error) {
	cl, err := GetClusterFromDB(c, db)
	if err != nil {
		return nil, err
	}
	return cl.GetClusterWithDbCluster(c, log)
}

// ReadClusterAmazon load amazon props from cloud to list clusters
func (cs ClusterSimple) ReadClusterAmazon(log *logrus.Logger) *ClusterRepresentation {
	log.Info("Read aws cluster with ", cs.ID, " id")
	c, err := ReadCluster(cs)
	if err == nil {
		log.Info("Read aws cluster success")
		clust := ClusterRepresentation{
			Id:        cs.ID,
			Name:      cs.Name,
			CloudType: Amazon,
			AmazonRepresentation: &AmazonRepresentation{
				Ip: c.KubernetesAPI.Endpoint,
			},
		}
		return &clust
	} else {
		log.Info("Something went wrong under read: ", err.Error())
	}
	return nil
}

// GetClusterInfoAmazon fetches amazon cluster props
func (cs *ClusterSimple) GetClusterInfoAmazon(c *gin.Context, log *logrus.Logger) {
	cl, err := cs.GetClusterWithDbCluster(c, log)
	if err != nil {
		log.Info("Error during fetch amazon cluster: ", err.Error())
		return
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
func (cs *ClusterSimple) DeleteAmazonCluster(c *gin.Context, db *gorm.DB, log *logrus.Logger) bool {

	if _, err := cs.DeleteClusterAmazon(); err != nil {
		// delete failed
		log.Warning("Can't delete cluster from cloud!", err)

		SetResponseBodyJson(c, http.StatusNotFound, gin.H{
			JsonKeyStatus:     http.StatusBadRequest,
			JsonKeyMessage:    "Can't delete cluster!",
			JsonKeyResourceId: cs.ID,
			JsonKeyError:      err,
		})
		return false
	} else {
		// delete success
		log.Info("Cluster deleted from the cloud!")
		notify.SlackNotify("Cluster deleted from the cloud!")

		SetResponseBodyJson(c, http.StatusCreated, gin.H{
			JsonKeyStatus:     http.StatusCreated,
			JsonKeyMessage:    "Cluster deleted successfully!",
			JsonKeyResourceId: cs.ID,
		})
		return true
	}

}
