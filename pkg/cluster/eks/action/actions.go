// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package action

import (
	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/secret/ssh"
	"github.com/banzaicloud/pipeline/src/utils"
)

// EksClusterContext describes the common fields used across EKS cluster create/update/delete operations
type EksClusterContext struct {
	Session     *session.Session
	ClusterName string
}

// EksClusterCreateUpdateContext describes the properties of an EKS cluster creation
type EksClusterCreateUpdateContext struct {
	EksClusterContext
	ClusterRoleArn             string
	NodeInstanceRoleID         string
	NodeInstanceRoleArn        string
	SecurityGroupID            *string
	NodeSecurityGroupID        *string
	Subnets                    []*EksSubnet
	SSHKeyName                 string
	SSHKey                     ssh.KeyPair
	VpcID                      *string
	VpcCidr                    *string
	ProvidedRoleArn            string
	APIEndpoint                *string
	CertificateAuthorityData   *string
	DefaultUser                bool
	ClusterRoleID              string
	ClusterUserArn             string
	ClusterUserAccessKeyId     string
	ClusterUserSecretAccessKey string
	RouteTableID               *string
	ScaleEnabled               bool
	LogTypes                   []string
	EndpointPrivateAccess      bool
	EndpointPublicAccess       bool
}

// NewEksClusterCreationContext creates a new EksClusterCreateUpdateContext
func NewEksClusterCreationContext(session *session.Session, clusterName, sshKeyName string) *EksClusterCreateUpdateContext {
	return &EksClusterCreateUpdateContext{
		EksClusterContext: EksClusterContext{
			Session:     session,
			ClusterName: clusterName,
		},
		SSHKeyName: sshKeyName,
	}
}

// EksClusterDeletionContext describes the properties of an EKS cluster deletion
type EksClusterDeletionContext struct {
	EksClusterContext
	VpcID            string
	SecurityGroupIDs []string
}

// ---

var _ utils.RevocableAction = (*LoadEksSettingsAction)(nil)

// LoadEksSettingsAction to describe the EKS cluster created
type LoadEksSettingsAction struct {
	context *EksClusterCreateUpdateContext
	log     logrus.FieldLogger
}

// NewLoadEksSettingsAction creates a new LoadEksSettingsAction
func NewLoadEksSettingsAction(log logrus.FieldLogger, context *EksClusterCreateUpdateContext) *LoadEksSettingsAction {
	return &LoadEksSettingsAction{
		context: context,
		log:     log,
	}
}

// GetName returns the name of this LoadEksSettingsAction
func (a *LoadEksSettingsAction) GetName() string {
	return "LoadEksSettingsAction"
}

// ExecuteAction executes this LoadEksSettingsAction
func (a *LoadEksSettingsAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Info("EXECUTE LoadEksSettingsAction")
	eksSvc := eks.New(a.context.Session)
	// Store API endpoint, etc..
	describeClusterInput := &eks.DescribeClusterInput{
		Name: aws.String(a.context.ClusterName),
	}
	clusterInfo, err := eksSvc.DescribeCluster(describeClusterInput)
	if err != nil {
		return nil, err
	}
	cluster := clusterInfo.Cluster
	if cluster == nil {
		return nil, errors.New("unable to get EKS Cluster info")
	}

	a.context.APIEndpoint = cluster.Endpoint
	a.context.CertificateAuthorityData = cluster.CertificateAuthority.Data
	// TODO store settings in db

	return input, nil
}

// UndoAction rolls back this LoadEksSettingsAction
func (a *LoadEksSettingsAction) UndoAction() (err error) {
	a.log.Info("EXECUTE UNDO LoadEksSettingsAction")
	return nil
}
