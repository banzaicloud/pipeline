package pkeworkflow

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/session"
)

type Clusters interface {
	GetCluster(ctx context.Context, id uint) (Cluster, error)
}

type Cluster interface {
	GetID() uint
	GetUID() string
	GetName() string
	GetOrganizationId() uint
	UpdateStatus(string, string) error
	GetNodePools() []NodePool
	GetLocation() string
}

type AWSCluster interface {
	GetAWSClient() (*session.Session, error)
	GetBootstrapCommand(string, string, string) (string, error)
	SaveNetworkCloudProvider(string, string, []string) error
	SaveNetworkApiServerAddress(string, string) error
}

type NodePool struct {
	Name              string
	MinCount          int
	MaxCount          int
	Count             int
	Master            bool
	Worker            bool
	InstanceType      string
	AvailabilityZones []string
	ImageID           string
}
