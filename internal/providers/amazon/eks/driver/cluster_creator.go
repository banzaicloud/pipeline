// Copyright © 2019 Banzai Cloud
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

package driver

import (
	"context"
	"net"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/banzaicloud/pipeline/internal/cluster/metrics"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/secret/ssh"
	"github.com/banzaicloud/pipeline/internal/secret/ssh/sshdriver"
	"github.com/banzaicloud/pipeline/src/auth"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	logrusadapter "logur.dev/adapter/logrus"

	"github.com/banzaicloud/pipeline/src/secret/verify"

	"go.uber.org/cadence/client"

	"github.com/banzaicloud/pipeline/internal/providers/amazon/eks/workflow"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgEks "github.com/banzaicloud/pipeline/pkg/cluster/eks"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	pkgEC2 "github.com/banzaicloud/pipeline/pkg/providers/amazon/ec2"
	"github.com/banzaicloud/pipeline/src/cluster"
)

type EksClusterCreator struct {
	logger                     logrus.FieldLogger
	workflowClient             client.Client
	serviceRegionLister        ServiceRegionLister
	clusters                   clusterRepository
	secrets                    secretValidator
	statusChangeDurationMetric metrics.ClusterStatusChangeDurationMetric
	clusterTotalMetric         *prometheus.CounterVec
}

type secretValidator interface {
	ValidateSecretType(organizationID uint, secretID string, cloud string) error
}

type clusterRepository interface {
	Exists(organizationID uint, name string) (bool, error)
}

// ServiceRegionLister lists regions where a service is available.
type ServiceRegionLister interface {
	// GetServiceRegions returns the cloud provider regions where the specified service is available.
	GetServiceRegions(ctx context.Context, cloudProvider string, service string) ([]string, error)
}

type invalidError struct {
	err error
}

func (e *invalidError) Error() string {
	return e.err.Error()
}

func (invalidError) IsInvalid() bool {
	return true
}

func NewEksClusterCreator(
	logger logrus.FieldLogger,
	workflowClient client.Client,
	serviceRegionLister ServiceRegionLister,
	clusters clusterRepository,
	secrets secretValidator,
	statusChangeDurationMetric metrics.ClusterStatusChangeDurationMetric,
	clusterTotalMetric *prometheus.CounterVec,
) EksClusterCreator {
	return EksClusterCreator{
		logger:                     logger,
		workflowClient:             workflowClient,
		serviceRegionLister:        serviceRegionLister,
		clusters:                   clusters,
		secrets:                    secrets,
		statusChangeDurationMetric: statusChangeDurationMetric,
		clusterTotalMetric:         clusterTotalMetric,
	}
}

func getNodePoolsForSubnet(subnetMapping map[string][]*pkgEks.Subnet, eksSubnet workflow.Subnet) []string {
	var nodePools []string
	for np, subnets := range subnetMapping {
		for _, subnet := range subnets {
			if (subnet.SubnetId != "" && eksSubnet.SubnetID == subnet.SubnetId) ||
				(subnet.Cidr != "" && eksSubnet.Cidr == subnet.Cidr) {
				nodePools = append(nodePools, np)
			}
		}
	}
	return nodePools
}

// Create implements the clusterCreator interface.
func (c *EksClusterCreator) create(ctx context.Context, logger logrus.FieldLogger, commonCluster cluster.CommonCluster, createRequest *pkgCluster.CreateClusterRequest) (cluster.CommonCluster, error) {
	logger.Info("start creating EKS Cluster")
	modelCluster := commonCluster.(*cluster.EKSCluster).GetEKSModel()

	if createRequest.PostHooks == nil {
		createRequest.PostHooks = make(pkgCluster.PostHooks)
	}

	org, err := auth.GetOrganizationById(commonCluster.GetOrganizationId())
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get organization name")
	}

	input := cluster.EKSCreateClusterWorkflowInput{
		CreateInfrastructureWorkflowInput: workflow.CreateInfrastructureWorkflowInput{
			Region:             commonCluster.GetLocation(),
			OrganizationID:     commonCluster.GetOrganizationId(),
			SecretID:           commonCluster.GetSecretId(),
			SSHSecretID:        commonCluster.GetSshSecretId(),
			ClusterUID:         commonCluster.GetUID(),
			ClusterID:          commonCluster.GetID(),
			ClusterName:        commonCluster.GetName(),
			VpcID:              aws.StringValue(modelCluster.VpcId),
			RouteTableID:       aws.StringValue(modelCluster.RouteTableId),
			VpcCidr:            aws.StringValue(modelCluster.VpcCidr),
			ScaleEnabled:       commonCluster.GetScaleOptions() != nil && commonCluster.GetScaleOptions().Enabled,
			DefaultUser:        modelCluster.DefaultUser,
			ClusterRoleID:      modelCluster.ClusterRoleId,
			NodeInstanceRoleID: modelCluster.NodeInstanceRoleId,
			KubernetesVersion:  modelCluster.Version,
			LogTypes:           modelCluster.LogTypes,
			GenerateSSH:        modelCluster.SSHGenerated,
		},
		PostHooks:        createRequest.PostHooks,
		OrganizationName: org.Name,
	}

	for _, mode := range modelCluster.APIServerAccessPoints {
		switch mode {
		case "public":
			input.EndpointPublicAccess = true
		case "private":
			input.EndpointPrivateAccess = true
		}
	}

	subnets := make([]workflow.Subnet, 0)
	subnetMapping := make(map[string][]workflow.Subnet)
	for _, eksSubnetModel := range modelCluster.Subnets {
		subnet := workflow.Subnet{
			SubnetID:         aws.StringValue(eksSubnetModel.SubnetId),
			Cidr:             aws.StringValue(eksSubnetModel.Cidr),
			AvailabilityZone: aws.StringValue(eksSubnetModel.AvailabilityZone)}

		subnets = append(subnets, subnet)

		nodePools := getNodePoolsForSubnet(commonCluster.(*cluster.EKSCluster).GetSubnetMapping(), subnet)
		logger.Debugf("node pools mapped to subnet %s: %v", subnet.SubnetID, nodePools)

		for _, np := range nodePools {
			subnetMapping[np] = append(subnetMapping[np], subnet)
		}
	}

	input.Subnets = subnets
	input.ASGSubnetMapping = subnetMapping

	asgList := make([]workflow.AutoscaleGroup, 0)
	nodePoolLabels := make([]cluster.NodePoolLabels, 0)

	for _, np := range modelCluster.NodePools {
		asg := workflow.AutoscaleGroup{
			Name:             np.Name,
			NodeSpotPrice:    np.NodeSpotPrice,
			Autoscaling:      np.Autoscaling,
			NodeMinCount:     np.NodeMinCount,
			NodeMaxCount:     np.NodeMaxCount,
			Count:            np.Count,
			NodeImage:        np.NodeImage,
			NodeInstanceType: np.NodeInstanceType,
			Labels:           np.Labels,
		}
		asgList = append(asgList, asg)

		nodePoolLabels = append(nodePoolLabels, cluster.NodePoolLabels{
			NodePoolName: np.Name,
			Existing:     false,
			InstanceType: np.NodeInstanceType,
			SpotPrice:    np.NodeSpotPrice,
			CustomLabels: np.Labels,
		})
	}

	input.AsgList = asgList

	labelsMap, err := cluster.GetDesiredLabelsForCluster(ctx, commonCluster, nodePoolLabels)
	if err != nil {
		_ = commonCluster.SetStatus(pkgCluster.Error, "failed to get desired labels")

		return nil, err
	}
	input.NodePoolLabels = labelsMap

	workflowOptions := client.StartWorkflowOptions{
		TaskList:                     "pipeline",
		ExecutionStartToCloseTimeout: 1 * 24 * time.Hour,
	}
	exec, err := c.workflowClient.ExecuteWorkflow(ctx, workflowOptions, cluster.EKSCreateClusterWorkflowName, input)
	if err != nil {
		return nil, err
	}

	err = commonCluster.(*cluster.EKSCluster).SetCurrentWorkflowID(exec.GetID())
	if err != nil {
		return nil, err
	}

	timer, err := getClusterStatusChangeMetricTimer(commonCluster.GetCloud(), commonCluster.GetLocation(), pkgCluster.Creating, commonCluster.GetOrganizationId(), commonCluster.GetName(), c.statusChangeDurationMetric)
	if err != nil {
		return nil, err
	}
	go func() {
		err = exec.Get(ctx, nil)
		if err != nil {
			logger.Error(errors.WrapIf(err, "cluster create workflow failed"))
			return
		}
		logger.Info("EKS cluster created.")
		timer.RecordDuration()
	}()

	return commonCluster, nil

}

func CreateAWSCredentialsFromSecret(eksCluster *cluster.EKSCluster) (*credentials.Credentials, error) {
	clusterSecret, err := eksCluster.GetSecretWithValidation()
	if err != nil {
		return nil, err
	}
	return verify.CreateAWSCredentials(clusterSecret.Values), nil
}

// ValidateCreationFields validates all fields
func (c *EksClusterCreator) validate(r *pkgCluster.CreateClusterRequest, logger logrus.FieldLogger, commonCluster cluster.CommonCluster) error {

	eksCluster := commonCluster.(*cluster.EKSCluster)

	logger.Debug("validating secretIDs")
	if len(r.SecretIds) > 0 {
		var err error
		for _, secretID := range r.SecretIds {
			err = c.secrets.ValidateSecretType(commonCluster.GetOrganizationId(), secretID, r.Cloud)
			if err == nil {
				break
			}
		}
		if err != nil {
			return err
		}
	} else {
		if err := c.secrets.ValidateSecretType(commonCluster.GetOrganizationId(), r.SecretId, r.Cloud); err != nil {
			return err
		}
	}

	logger.Debug("validating creation fields")

	regions, err := c.serviceRegionLister.GetServiceRegions(context.Background(), pkgCluster.Amazon, pkgCluster.EKS)
	if err != nil {
		return errors.WrapIf(err, "failed to list regions where EKS service is enabled")
	}

	regionFound := false
	for _, region := range regions {
		if region == r.Location {
			regionFound = true
			break
		}
	}

	if !regionFound {
		return pkgErrors.ErrorNotValidLocation
	}

	// validate VPC
	awsCred, err := CreateAWSCredentialsFromSecret(eksCluster)
	if err != nil {
		return errors.WrapIf(err, "failed to get Cluster AWS credentials")
	}

	awsSession, err := session.NewSession(&aws.Config{
		Region:      aws.String(commonCluster.GetLocation()),
		Credentials: awsCred,
	})
	if err != nil {
		return errors.WrapIf(err, "failed to create AWS session")
	}

	netSvc := pkgEC2.NewNetworkSvc(ec2.New(awsSession), logrusadapter.New(logrus.New()))
	if r.Properties.CreateClusterEKS.Vpc != nil {

		if r.Properties.CreateClusterEKS.Vpc.VpcId != "" && r.Properties.CreateClusterEKS.Vpc.Cidr != "" {
			return errors.NewWithDetails("specifying both CIDR and ID for VPC is not allowed", "vpc", *r.Properties.CreateClusterEKS.Vpc)
		}

		if r.Properties.CreateClusterEKS.Vpc.VpcId == "" && r.Properties.CreateClusterEKS.Vpc.Cidr == "" {
			return errors.NewWithDetails("either CIDR or ID is required for VPC", "vpc", *r.Properties.CreateClusterEKS.Vpc)
		}

		if r.Properties.CreateClusterEKS.Vpc.VpcId != "" {
			// verify that the provided VPC exists and is in available state
			exists, err := netSvc.VpcAvailable(r.Properties.CreateClusterEKS.Vpc.VpcId)

			if err != nil {
				return errors.WrapIfWithDetails(err, "failed to check if VPC is available", "vpc", *r.Properties.CreateClusterEKS.Vpc)
			}

			if !exists {
				return errors.NewWithDetails("VPC not found or it's not in 'available' state", "vpc", *r.Properties.CreateClusterEKS.Vpc)
			}
		}
	}

	// subnets
	allExistingSubnets := make(map[string]*pkgEks.Subnet)
	allNewSubnets := make(map[string]*pkgEks.Subnet)
	for _, subnet := range r.Properties.CreateClusterEKS.Subnets {
		if subnet.SubnetId != "" {
			allExistingSubnets[subnet.SubnetId] = subnet
		} else if subnet.Cidr != "" {
			if s, ok := allNewSubnets[subnet.Cidr]; ok && s.AvailabilityZone != subnet.AvailabilityZone {
				return errors.Errorf("subnets with same cidr %s but mismatching AZs found", subnet.Cidr)
			}
			allNewSubnets[subnet.Cidr] = subnet
		}
	}
	for _, np := range r.Properties.CreateClusterEKS.NodePools {
		if np.Subnet != nil {
			if np.Subnet.SubnetId != "" {
				allExistingSubnets[np.Subnet.SubnetId] = np.Subnet
			} else if np.Subnet.Cidr != "" {
				if s, ok := allNewSubnets[np.Subnet.Cidr]; ok && s.AvailabilityZone != np.Subnet.AvailabilityZone {
					return errors.Errorf("subnets with same cidr %s but mismatching AZs found", np.Subnet.Cidr)
				}
				allNewSubnets[np.Subnet.Cidr] = np.Subnet
			}
		}
	}

	for _, subnet := range allNewSubnets {
		if subnet.AvailabilityZone != "" && !strings.HasPrefix(strings.ToLower(subnet.AvailabilityZone), strings.ToLower(r.Location)) {
			return errors.Errorf("invalid AZ '%s' for region '%s'", subnet.AvailabilityZone, r.Location)
		}
	}

	if len(allExistingSubnets) > 0 && len(allNewSubnets) > 0 {
		return errors.New("mixing existing subnets identified by provided subnet id and new subnets to be created with given cidr is not allowed, specify either CIDR and optionally AZ or ID for all Subnets")
	}

	if len(allExistingSubnets)+len(allNewSubnets) < 2 {
		return errors.New("at least two subnets in two different AZs are required for EKS")
	}

	if len(allExistingSubnets) > 0 && r.Properties.CreateClusterEKS.Vpc.Cidr != "" {
		return errors.New("VPC ID must be provided")
	}

	// verify that the provided existing subnets exist
	for _, subnet := range allExistingSubnets {
		if subnet.Cidr != "" && subnet.SubnetId != "" {
			return errors.New("specifying both CIDR and ID for a Subnet is not allowed")
		}

		if subnet.Cidr == "" && subnet.SubnetId == "" {
			return errors.New("either CIDR or ID is required for Subnet")
		}

		if subnet.SubnetId != "" {
			exists, err := netSvc.SubnetAvailable(subnet.SubnetId, r.Properties.CreateClusterEKS.Vpc.VpcId)
			if err != nil {
				return errors.WrapIfWithDetails(err, "failed to check if Subnet is available in VPC")
			}
			if !exists {
				return errors.Errorf("subnet '%s' not found in VPC or it's not in 'available' state", subnet.SubnetId)
			}
		}
	}
	// verify that new subnets (to be created) do not overlap and are within the VPC's CIDR range
	if len(allNewSubnets) > 0 {
		_, vpcCidr, err := net.ParseCIDR(r.Properties.CreateClusterEKS.Vpc.Cidr)
		vpcMaskOnes, _ := vpcCidr.Mask.Size()
		if err != nil {
			return errors.WrapIf(err, "failed to parse vpc cidr")
		}

		subnetCidrs := make([]string, 0, len(allNewSubnets))
		for cidr := range allNewSubnets {
			subnetCidrs = append(subnetCidrs, cidr)
		}

		for i := range subnetCidrs {
			ip1, cidr1, err := net.ParseCIDR(subnetCidrs[i])
			if err != nil {
				return errors.WrapIf(err, "failed to parse subnet cidr")
			}

			if !vpcCidr.Contains(ip1) {
				return errors.Errorf("subnet cidr '%s' is outside of vpc cidr range '%s'", cidr1, vpcCidr)
			}

			ones, _ := cidr1.Mask.Size()
			if ones < vpcMaskOnes {
				return errors.Errorf("subnet cidr '%s' is is bigger than vpc cidr range '%s'", cidr1, vpcCidr)
			}

			for j := i + 1; j < len(subnetCidrs); j++ {
				ip2, cidr2, err := net.ParseCIDR(subnetCidrs[j])
				if err != nil {
					return errors.WrapIf(err, "failed to parse subnet cidr")
				}

				if cidr1.Contains(ip2) || cidr2.Contains(ip1) {
					return errors.Errorf("overlapping subnets found: '%s', '%s'", cidr1, cidr2)
				}
			}
		}
	}

	// route table
	// if VPC ID and Subnet CIDR is provided than Route Table ID is required as well.

	if r.Properties.CreateClusterEKS.Vpc.VpcId != "" && len(allNewSubnets) > 0 {
		if r.Properties.CreateClusterEKS.RouteTableId == "" {
			return errors.New("if VPC ID specified and CIDR for Subnets, Route Table ID must be provided as well")
		}

		// verify if provided route table exists
		exists, err := netSvc.RouteTableAvailable(r.Properties.CreateClusterEKS.RouteTableId, r.Properties.CreateClusterEKS.Vpc.VpcId)
		if err != nil {
			return errors.WrapIfWithDetails(err, "failed to check if RouteTable is available",
				"vpcId", r.Properties.CreateClusterEKS.Vpc.VpcId,
				"routeTableId", r.Properties.CreateClusterEKS.RouteTableId)
		}
		if !exists {
			return errors.New("Route Table not found in the given VPC or it's not in 'active' state")
		}

	} else {
		if r.Properties.CreateClusterEKS.RouteTableId != "" {
			return errors.New("Route Table ID should be provided only when VPC ID and CIDR for Subnets are specified")
		}
	}

	return nil
}

func (c *EksClusterCreator) assertNotExists(orgID uint, name string) error {
	exists, err := c.clusters.Exists(orgID, name)
	if err != nil {
		return err
	}

	if exists {
		return cluster.ErrAlreadyExists
	}

	return nil
}

func (c *EksClusterCreator) generateSSHkey(commonCluster cluster.CommonCluster) error {
	sshKey, err := ssh.NewKeyPairGenerator().Generate()
	if err != nil {
		_ = commonCluster.SetStatus(pkgCluster.Error, "internal error")
		return errors.WrapIf(err, "failed to generate SSH key")
	}

	sshSecretId, err := sshdriver.StoreSSHKeyPair(sshKey, commonCluster.GetOrganizationId(), commonCluster.GetID(), commonCluster.GetName(), commonCluster.GetUID())
	if err != nil {
		_ = commonCluster.SetStatus(pkgCluster.Error, "internal error")
		return errors.WrapIf(err, "failed to store SSH key")
	}

	if err := commonCluster.SaveSshSecretId(sshSecretId); err != nil {
		_ = commonCluster.SetStatus(pkgCluster.Error, "internal error")
		return errors.WrapIf(err, "failed to save SSH key secret ID")
	}
	return nil
}

func (c *EksClusterCreator) CreateCluster(ctx context.Context, commonCluster cluster.CommonCluster, createRequest *pkgCluster.CreateClusterRequest, organizationID uint, userID uint) (cluster.CommonCluster, error) {

	logger := c.logger.WithFields(logrus.Fields{
		"clusterName":    commonCluster.GetName(),
		"clusterID":      commonCluster.GetID(),
		"organizationID": commonCluster.GetOrganizationId(),
	})

	if err := c.assertNotExists(organizationID, commonCluster.GetName()); err != nil {
		return nil, err
	}

	if err := c.validate(createRequest, logger, commonCluster); err != nil {
		return nil, errors.Wrap(&invalidError{err}, "validation failed")
	}

	if err := commonCluster.Persist(); err != nil {
		return nil, err
	}

	c.clusterTotalMetric.WithLabelValues(commonCluster.GetCloud(), commonCluster.GetLocation()).Inc()

	if err := commonCluster.SetStatus(pkgCluster.Creating, pkgCluster.CreatingMessage); err != nil {
		return nil, err
	}

	// Check if public ssh key is needed for the cluster. If so and there is generate one and store it Vault
	var sshGenerated bool
	if len(commonCluster.GetSshSecretId()) == 0 && commonCluster.RequiresSshPublicKey() && global.Config.Distribution.EKS.SSH.Generate {
		logger.Debug("generating SSH Key for the cluster")
		err := c.generateSSHkey(commonCluster)
		if err != nil {
			return nil, err
		}

		sshGenerated = true
	} else {
		sshGenerated = false
	}

	// store SSH generation
	if err := commonCluster.(*cluster.EKSCluster).PersistSSHGenerate(sshGenerated); err != nil {
		return nil, err
	}

	return c.create(ctx, logger, commonCluster, createRequest)
}
