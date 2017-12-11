package cloud

import (
	"fmt"

	"github.com/kris-nova/kubicorn/apis/cluster"
	"github.com/kris-nova/kubicorn/cutil/kubeadm"
	"github.com/kris-nova/kubicorn/cutil/uuid"
	"github.com/sirupsen/logrus"
)

const (
	amazonDefaultNodeImage          = "ami-bdba13c4"
	amazonDefaultMasterImage        = "ami-bdba13c4"
	amazonDefaultMasterInstanceType = "m4.xlarge"
	amazonDefaultNodeMinCount       = 1
	amazonDefaultNodeMaxCount       = 1
	amazonDefaultNodeSpotPrice      = "0.2"
)

// getAWSCluster creates *cluster.Cluster from ClusterSimple struct
func getAWSCluster(clusterType ClusterSimple) *cluster.Cluster {
	return &cluster.Cluster{
		Name:     clusterType.Name,
		Cloud:    cluster.CloudAmazon,
		Location: clusterType.Location,
		SSH: &cluster.SSH{
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
				Name:     fmt.Sprintf("%s.master", clusterType.Name),
				MinCount: 1,
				MaxCount: 1,
				Image:    clusterType.Amazon.MasterInstanceType, //"ami-835b4efa"
				Size:     clusterType.NodeInstanceType,
				BootstrapScripts: []string{
					"https://raw.githubusercontent.com/banzaicloud/banzai-charts/master/stable/pipeline/bootstrap/amazon_k8s_ubuntu_16.04_master_pipeline.sh",
				},
				InstanceProfile: &cluster.IAMInstanceProfile{
					Name: fmt.Sprintf("%s-KubicornMasterInstanceProfile", clusterType.Name),
					Role: &cluster.IAMRole{
						Name: fmt.Sprintf("%s-KubicornMasterRole", clusterType.Name),
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
						Name:     fmt.Sprintf("%s.master", clusterType.Name),
						CIDR:     "10.0.0.0/24",
						Location: clusterType.Location,
					},
				},

				Firewalls: []*cluster.Firewall{
					{
						Name: fmt.Sprintf("%s.master-external-%s", clusterType.Name, uuid.TimeOrderedUUID()),
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
				Name:     fmt.Sprintf("%s.node", clusterType.Name),
				MinCount: clusterType.Amazon.NodeMinCount,
				MaxCount: clusterType.Amazon.NodeMaxCount,
				Image:    clusterType.Amazon.NodeImage, //"ami-835b4efa"
				Size:     clusterType.NodeInstanceType,
				AwsConfiguration: &cluster.AwsConfiguration{
					SpotPrice: clusterType.Amazon.NodeSpotPrice,
				},
				BootstrapScripts: []string{
					"https://raw.githubusercontent.com/banzaicloud/banzai-charts/master/stable/pipeline/bootstrap/amazon_k8s_ubuntu_16.04_node_pipeline.sh",
				},
				InstanceProfile: &cluster.IAMInstanceProfile{
					Name: fmt.Sprintf("%s-KubicornNodeInstanceProfile", clusterType.Name),
					Role: &cluster.IAMRole{
						Name: fmt.Sprintf("%s-KubicornNodeRole", clusterType.Name),
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
						Name:     fmt.Sprintf("%s.node", clusterType.Name),
						CIDR:     "10.0.100.0/24",
						Location: clusterType.Location,
					},
				},
				Firewalls: []*cluster.Firewall{
					{
						Name: fmt.Sprintf("%s.node-external-%s", clusterType.Name, uuid.TimeOrderedUUID()),
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
	CreateClusterSimpleId uint `gorm:"primary_key"`
	NodeSpotPrice         string
	NodeMinCount          int
	NodeMaxCount          int
	NodeImage             string
	MasterInstanceType    string
	MasterImage           string
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
