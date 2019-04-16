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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	aliErrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ess"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/pkg/cluster/ack"
	pkgErrors "github.com/banzaicloud/pipeline/pkg/errors"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func deleteCluster(log logrus.FieldLogger, clusterID string, csClient *cs.Client) error {

	if len(clusterID) == 0 {
		return errors.New("could not delete cluster, could not get clusterID " +
			"(there is a big chance that resources are in unrecognized state in Alibaba " +
			"please check your subscription)")
	}

	cluster, err := waitUntilClusterCreateOrScaleComplete(log, clusterID, csClient, true)
	if err != nil {
		if strings.Contains(err.Error(), "ErrorClusterNotFound") {
			return nil
		}

		if cluster == nil {
			return emperror.Wrap(err, "could not delete cluster!")
		}
	}

	req := cs.CreateDeleteClusterRequest()
	req.ClusterId = clusterID
	req.SetScheme(requests.HTTPS)
	req.SetDomain(ack.AlibabaApiDomain)

	resp, err := csClient.DeleteCluster(req)
	if err != nil {
		if sdkErr, ok := err.(*aliErrors.ServerError); ok {
			if strings.Contains(sdkErr.Message(), "ErrorClusterNotFound") {
				// Cluster has been already deleted
				return nil
			}
		}
		return errors.WithMessage(err, fmt.Sprintf("DeleteClusterResponse: %#v \n", resp.BaseResponse))
	}

	if resp.GetHttpStatus() != http.StatusAccepted {
		return fmt.Errorf("unexpected http status code: %d", resp.GetHttpStatus())
	}

	err = waitUntilClusterDeleteIsComplete(log, clusterID, csClient)
	if err != nil {
		return emperror.WrapWith(err, "cluster deletion failed", "clusterId", clusterID)
	}

	return nil
}

func describeScalingInstances(essClient *ess.Client, asgId, scalingConfId, regionId string) (*ess.DescribeScalingInstancesResponse, error) {
	describeScalingInstancesRequest := ess.CreateDescribeScalingInstancesRequest()
	describeScalingInstancesRequest.SetScheme(requests.HTTPS)
	describeScalingInstancesRequest.SetDomain(fmt.Sprintf(ack.AlibabaESSEndPointFmt, regionId))
	describeScalingInstancesRequest.SetContentType(requests.Json)

	describeScalingInstancesRequest.ScalingGroupId = asgId
	describeScalingInstancesRequest.ScalingConfigurationId = scalingConfId

	describeScalingInstancesResponse, err := essClient.DescribeScalingInstances(describeScalingInstancesRequest)
	if err != nil {
		return nil, emperror.WrapWith(err, "could not describe scaling instances", "scalingGroupId", asgId)
	}
	return describeScalingInstancesResponse, nil
}

func attachInstancesToCluster(log logrus.FieldLogger, clusterID string, instanceIds []string, csClient *cs.Client) (*ack.AlibabaDescribeClusterResponse, error) {
	log.Info("Attaching nodepools to cluster")
	attachInstanceRequest := cs.CreateAttachInstancesRequest()
	attachInstanceRequest.SetScheme(requests.HTTPS)
	attachInstanceRequest.SetDomain(ack.AlibabaApiDomain)
	attachInstanceRequest.SetContentType(requests.Json)

	attachInstanceRequest.ClusterId = clusterID

	content := map[string]interface{}{
		"instances": instanceIds,
		"password":  "Hello1234", // Dummy password should be used here otherwise the api will fail
	}
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}
	attachInstanceRequest.SetContent(contentJSON)

	_, err = csClient.AttachInstances(attachInstanceRequest)
	if err != nil {
		return nil, emperror.Wrap(err, "could not attach instances to cluster")
	}
	log.Info("Wait for nodepool attach")
	clusterWithPools, err := waitUntilClusterCreateOrScaleComplete(log, clusterID, csClient, false)
	if err != nil {
		return nil, emperror.Wrap(err, "attaching nodepool failed")
	}
	return clusterWithPools, nil
}

func deleteNodePools(log logrus.FieldLogger, nodePools []*model.ACKNodePoolModel, essClient *ess.Client, regionId string) error {
	errChan := make(chan error, len(nodePools))
	defer close(errChan)

	for _, nodePool := range nodePools {
		go deleteNodePool(log, nodePool, essClient, regionId, errChan)
	}
	var err error
	caughtErrors := emperror.NewMultiErrorBuilder()

	for i := 0; i < len(nodePools); i++ {
		err = <-errChan
		if err != nil {
			caughtErrors.Add(err)
		}
	}

	return pkgErrors.NewMultiErrorWithFormatter(caughtErrors.ErrOrNil())
}

func deleteNodePool(log logrus.FieldLogger, nodePool *model.ACKNodePoolModel, essClient *ess.Client, regionId string, errChan chan<- error) {
	deleteSGRequest := ess.CreateDeleteScalingGroupRequest()
	deleteSGRequest.SetScheme(requests.HTTPS)
	deleteSGRequest.SetDomain(fmt.Sprintf(ack.AlibabaESSEndPointFmt, regionId))
	deleteSGRequest.SetContentType(requests.Json)

	if nodePool.AsgID == "" {
		// Asg could not be created nothing to remove
		errChan <- nil
		return
	}

	deleteSGRequest.ScalingGroupId = nodePool.AsgID
	deleteSGRequest.ForceDelete = requests.NewBoolean(true)

	_, err := essClient.DeleteScalingGroup(deleteSGRequest)
	if err != nil {
		if sdkErr, ok := err.(*aliErrors.ServerError); ok {
			if strings.Contains(sdkErr.ErrorCode(), "InvalidScalingGroupId.NotFound") {
				log.WithFields(logrus.Fields{"scalingGroupId": nodePool.AsgID, "nodePoolName": nodePool.Name}).Info("scaling group to be deleted not found")

				errChan <- nil
				return
			}
		}

		errChan <- emperror.WrapWith(err, "could not delete scaling group", "scalingGroupId", nodePool.AsgID, "nodePoolName", nodePool.Name)
		return
	}

	err = waitUntilScalingInstancesDeleted(log, essClient, regionId, nodePool)
	if err != nil {
		errChan <- err
		return
	}

	errChan <- nil
}

func createNodePool(logger logrus.FieldLogger, nodePool *model.ACKNodePoolModel, essClient *ess.Client, cluster *ack.AlibabaDescribeClusterResponse, instanceIdsChan chan<- []string, errChan chan<- error) {
	scalingGroupRequest := ess.CreateCreateScalingGroupRequest()
	scalingGroupRequest.SetScheme(requests.HTTPS)
	scalingGroupRequest.SetDomain(fmt.Sprintf(ack.AlibabaESSEndPointFmt, cluster.RegionID))
	scalingGroupRequest.SetContentType(requests.Json)

	log := logger.WithFields(logrus.Fields{
		"region":        cluster.RegionID,
		"zone":          cluster.ZoneID,
		"instance_type": nodePool.InstanceType,
	})

	log.Info("creating scaling group")

	scalingGroupRequest.MinSize = requests.NewInteger(nodePool.MinCount)
	scalingGroupRequest.MaxSize = requests.NewInteger(nodePool.MaxCount)
	scalingGroupRequest.VSwitchId = cluster.VSwitchID
	scalingGroupRequest.ScalingGroupName = fmt.Sprintf("asg-%s-%s", nodePool.Name, cluster.ClusterID)

	createScalingGroupResponse, err := essClient.CreateScalingGroup(scalingGroupRequest)
	if err != nil {
		errChan <- emperror.WrapWith(err, "could not create Scaling Group", "nodePoolName", nodePool.Name, "cluster", cluster.Name)
		instanceIdsChan <- nil
		return
	}

	nodePool.AsgID = createScalingGroupResponse.ScalingGroupId
	log = log.WithField("scalingGroupId", nodePool.AsgID)

	log.Info("scaling group successfully created")
	log.Info("creating scaling configuration for scaling group")

	scalingConfigurationRequest := ess.CreateCreateScalingConfigurationRequest()
	scalingConfigurationRequest.SetScheme(requests.HTTPS)
	scalingConfigurationRequest.SetDomain(fmt.Sprintf(ack.AlibabaESSEndPointFmt, cluster.RegionID))
	scalingConfigurationRequest.SetContentType(requests.Json)

	scalingConfigurationRequest.ScalingGroupId = nodePool.AsgID
	scalingConfigurationRequest.SecurityGroupId = cluster.SecurityGroupID
	scalingConfigurationRequest.KeyPairName = cluster.Name
	scalingConfigurationRequest.InstanceType = nodePool.InstanceType
	scalingConfigurationRequest.SystemDiskCategory = "cloud_efficiency"
	scalingConfigurationRequest.ImageId = ack.AlibabaDefaultImageId
	scalingConfigurationRequest.Tags =
		fmt.Sprintf(`{"pipeline-created":"true","pipeline-cluster":"%s","pipeline-nodepool":"%s"`,
			cluster.Name, nodePool.Name)

	createConfigurationResponse, err := essClient.CreateScalingConfiguration(scalingConfigurationRequest)
	if err != nil {
		errChan <- emperror.WrapWith(err, "could not create Scaling Configuration", "nodePoolName", nodePool.Name, "scalingGroupId", nodePool.AsgID, "cluster", cluster.Name)
		instanceIdsChan <- nil
		return
	}

	nodePool.ScalingConfigID = createConfigurationResponse.ScalingConfigurationId

	log.Info("creating Scaling Configuration succeeded")

	enableSGRequest := ess.CreateEnableScalingGroupRequest()
	enableSGRequest.SetScheme(requests.HTTPS)
	enableSGRequest.SetDomain(fmt.Sprintf(ack.AlibabaESSEndPointFmt, cluster.RegionID))
	enableSGRequest.SetContentType(requests.Json)

	enableSGRequest.ScalingGroupId = nodePool.AsgID
	enableSGRequest.ActiveScalingConfigurationId = nodePool.ScalingConfigID

	_, err = essClient.EnableScalingGroup(enableSGRequest)
	if err != nil {
		errChan <- emperror.WrapWith(err, "could not enable Scaling Group", "nodePoolName", nodePool.Name, "scalingGroupId", nodePool.AsgID, "cluster", cluster.Name)
		instanceIdsChan <- nil
		return
	}

	instanceIds, err := waitUntilScalingInstanceUpdated(log, essClient, cluster.RegionID, nodePool)
	if err != nil {
		errChan <- emperror.With(err, "cluster", cluster.Name)
		instanceIdsChan <- nil
		return
	}
	// set running instance count for nodePool in DB
	nodePool.Count = len(instanceIds)

	errChan <- nil
	instanceIdsChan <- instanceIds
}

func updateNodePool(log logrus.FieldLogger, nodePool *model.ACKNodePoolModel, essClient *ess.Client, regionId, clusterName string, createdInstanceIdsChan chan<- []string, errChan chan<- error) {
	describeScalingInstancesResponseBeforeModify, err :=
		describeScalingInstances(essClient, nodePool.AsgID, nodePool.ScalingConfigID, regionId)
	if err != nil {
		errChan <- emperror.With(err, "nodePoolName", nodePool.Name, "cluster", clusterName)
		createdInstanceIdsChan <- nil
		return
	}

	modifyScalingGroupReq := ess.CreateModifyScalingGroupRequest()
	modifyScalingGroupReq.SetDomain(fmt.Sprintf(ack.AlibabaESSEndPointFmt, regionId))
	modifyScalingGroupReq.SetScheme(requests.HTTPS)
	modifyScalingGroupReq.RegionId = regionId
	modifyScalingGroupReq.ScalingGroupId = nodePool.AsgID
	modifyScalingGroupReq.MinSize = requests.NewInteger(nodePool.MinCount)
	modifyScalingGroupReq.MaxSize = requests.NewInteger(nodePool.MaxCount)

	_, err = essClient.ModifyScalingGroup(modifyScalingGroupReq)
	if err != nil {
		errChan <- emperror.WrapWith(err, "could not modify ScalingGroup", "scalingGroupId", nodePool.AsgID, "nodePoolName", nodePool.Name, "cluster", clusterName)
		createdInstanceIdsChan <- nil
		return
	}

	_, err = waitUntilScalingInstanceUpdated(log, essClient, regionId, nodePool)
	if err != nil {
		errChan <- emperror.With(err, "cluster", clusterName)
		createdInstanceIdsChan <- nil
		return
	}

	describeScalingInstancesResponseAfterModify, err :=
		describeScalingInstances(essClient, nodePool.AsgID, nodePool.ScalingConfigID, regionId)
	if err != nil {
		errChan <- emperror.With(err, "nodePoolName", nodePool.Name, "cluster", clusterName)
		createdInstanceIdsChan <- nil
		return
	}
	if describeScalingInstancesResponseBeforeModify.TotalCount < describeScalingInstancesResponseAfterModify.TotalCount {
		// add new instance to nodepool so we need to join them into the cluster
		var createdInstaceIds []string
		createdInstaces := difference(describeScalingInstancesResponseAfterModify.ScalingInstances.ScalingInstance, describeScalingInstancesResponseBeforeModify.ScalingInstances.ScalingInstance)
		for _, a := range createdInstaces {
			createdInstaceIds = append(createdInstaceIds, a.InstanceId)
		}
		// update running instance count for nodePool in DB
		nodePool.Count = describeScalingInstancesResponseAfterModify.TotalCount
		errChan <- nil
		createdInstanceIdsChan <- createdInstaceIds
		return
	}
	// instances removed from nodepool so we only need to set the count properly in the DB
	nodePool.Count = describeScalingInstancesResponseAfterModify.TotalCount
	errChan <- nil
	createdInstanceIdsChan <- nil
}

func waitUntilScalingInstanceUpdated(log logrus.FieldLogger, essClient *ess.Client, regionId string, nodePool *model.ACKNodePoolModel) ([]string, error) {
	log.WithField("nodePoolName", nodePool.Name).Info("waiting for instances to get ready")

	for {
		describeScalingInstancesResponse, err := describeScalingInstances(essClient, nodePool.AsgID, nodePool.ScalingConfigID, regionId)
		if err != nil {
			return nil, emperror.With(err, "nodePoolName", nodePool.Name)
		}
		if describeScalingInstancesResponse.TotalCount < nodePool.MinCount || describeScalingInstancesResponse.TotalCount > nodePool.MaxCount {
			continue
		}
		instanceIds := make([]string, 0)

		for _, instance := range describeScalingInstancesResponse.ScalingInstances.ScalingInstance {
			if instance.HealthStatus == ack.AlibabaInstanceHealthyStatus {
				instanceIds = append(instanceIds, instance.InstanceId)
			} else {
				time.Sleep(time.Second * 20)
				break
			}
		}
		if len(instanceIds) == len(describeScalingInstancesResponse.ScalingInstances.ScalingInstance) {
			return instanceIds, nil
		}
	}
}

func waitUntilScalingInstancesDeleted(log logrus.FieldLogger, essClient *ess.Client, regionId string, nodePool *model.ACKNodePoolModel) error {
	log.WithField("nodePoolName", nodePool.Name).Info("waiting for instances to be deleted")

	for {
		describeScalingInstancesResponse, err := describeScalingInstances(essClient, nodePool.AsgID, nodePool.ScalingConfigID, regionId)
		if err != nil {
			return emperror.With(err, "nodePoolName", nodePool.Name)
		}

		if describeScalingInstancesResponse.TotalCount == 0 {
			return nil
		}

		time.Sleep(time.Second * 20)
	}
}

func waitUntilClusterCreateOrScaleComplete(log logrus.FieldLogger, clusterID string, csClient *cs.Client, isClusterCreate bool) (*ack.AlibabaDescribeClusterResponse, error) {
	var (
		r     *ack.AlibabaDescribeClusterResponse
		state string
		err   error
	)
	for {
		r, err = GetClusterDetails(csClient, clusterID)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				log.Warn(err)
				continue
			}
			return nil, err
		}

		if r.State != state {
			log.Infof("%s cluster %s", r.State, clusterID)
			state = r.State
		}

		switch r.State {
		case ack.AlibabaClusterStateRunning:
			if !isClusterCreate {
				// in case of cluster scale the transition from 'scaling' -> 'running'
				// doesn't necessary mean that the scale succeeded.
				// If node count quota is hit than the cluster state transitions from 'scaling' to 'running'
				// without the scaling taking place thus we need to collect cluster event logs
				// to see if scaling succeeded

				logs, err := collectClusterScaleFailureLogs(clusterID, csClient)
				if err != nil {
					log.Error("failed to collect cluster failure event log")
				}
				if len(logs) > 0 {
					return r, AlibabaClusterFailureLogsError{clusterEventLogs: logs}
				}
			}

			return r, nil
		case ack.AlibabaClusterStateFailed:
			var logs []string
			var err error

			if isClusterCreate {
				logs, err = collectClusterCreateFailureLogs(clusterID, csClient)
			} else {
				logs, err = collectClusterScaleFailureLogs(clusterID, csClient)
			}
			if err != nil {
				log.Error("failed to collect cluster failure event log")
			}

			return r, AlibabaClusterFailureLogsError{clusterEventLogs: logs}
		default:
			time.Sleep(time.Second * 20)
		}
	}
}

func waitUntilClusterDeleteIsComplete(logger logrus.FieldLogger, clusterID string, csClient *cs.Client) error {
	log := logger.WithField("clusterId", clusterID)
	log.Info("waiting for cluster to be deleted")

	req := cs.CreateDescribeClusterDetailRequest()
	req.SetScheme(requests.HTTPS)
	req.SetDomain(ack.AlibabaApiDomain)
	req.ClusterId = clusterID

	for {
		resp, err := csClient.DescribeClusterDetail(req)
		if err != nil {
			if sdkErr, ok := err.(*aliErrors.ServerError); ok {
				if strings.Contains(sdkErr.Message(), "ErrorClusterNotFound") {
					// cluster has been deleted
					return nil
				}
			}

			return emperror.WrapWith(err, "could not get cluster details", "clusterId", clusterID)
		}

		var r *ack.AlibabaDescribeClusterResponse

		err = json.Unmarshal(resp.GetHttpContentBytes(), &r)
		if err != nil {
			return emperror.WrapWith(err, "could not unmarshall describe cluster details", "clusterId", clusterID)
		}

		if r.State == ack.AlibabaClusterStateFailed {
			var logs []string
			logs, err = collectClusterDeleteFailureLogs(clusterID, csClient)

			if err != nil {
				log.Error("failed to collect cluster failure event log")
			}

			return AlibabaClusterFailureLogsError{clusterEventLogs: logs}
		}

		time.Sleep(time.Second * 20)
	}
}

// GetClusterDetails retrieves cluster details from cloud provider
func GetClusterDetails(client *cs.Client, clusterID string) (r *ack.AlibabaDescribeClusterResponse, err error) {
	if clusterID == "" {
		return nil, errors.New("could not get cluster details clusterId is empty")
	}
	req := cs.CreateDescribeClusterDetailRequest()
	req.SetScheme(requests.HTTPS)
	req.SetDomain(ack.AlibabaApiDomain)
	req.ClusterId = clusterID
	resp, err := client.DescribeClusterDetail(req)
	if err != nil {
		return nil, emperror.WrapWith(err, "could not get cluster details", "clusterId", clusterID)
	}
	if !resp.IsSuccess() {
		return nil, emperror.WrapWith(err, "unexpected http status code", "statusCode", resp.GetHttpStatus())
	}

	err = json.Unmarshal(resp.GetHttpContentBytes(), &r)
	return r, emperror.WrapWith(err, "could not unmarshall describe cluster details", "clusterId", clusterID)
}

// collectClusterLogs returns the event logs associated with the cluster identified by clusterID
func collectClusterLogs(clusterID string, csClient *cs.Client) ([]*ack.AlibabaDescribeClusterLogResponseEntry, error) {
	clusterLogsRequest := cs.CreateDescribeClusterLogsRequest()
	clusterLogsRequest.ClusterId = clusterID
	clusterLogsRequest.SetScheme(requests.HTTPS)
	clusterLogsRequest.SetDomain(ack.AlibabaApiDomain)

	clusterLogsResp, err := csClient.DescribeClusterLogs(clusterLogsRequest)

	if clusterLogsResp != nil {
		if !clusterLogsResp.IsSuccess() {
			return nil, errors.Wrapf(err, "Unexpected http status code: %d", clusterLogsResp.GetHttpStatus())
		}

		var clusterLogs []*ack.AlibabaDescribeClusterLogResponseEntry
		err = json.Unmarshal(clusterLogsResp.GetHttpContentBytes(), &clusterLogs)
		if err != nil {
			return nil, err
		}

		return clusterLogs, nil
	}

	return nil, nil
}

// collectClusterLogsInRange returns the logs events in-between the provided start and end log line markers
func collectClusterLogsInRange(clusterID string, csClient *cs.Client, startMarker, endMarker string) ([]string, error) {
	logs, err := collectClusterLogs(clusterID, csClient)
	if err != nil {
		return nil, err
	}

	// process log lines in-between the starMarker and endMarker log lines
	// cluster event log collection received from Alibaba are in reverse chronological order, thus the endMarker precedes
	// the starMarker line
	insideMarkers := false
	var errorLogs []string

	for _, logEntry := range logs {
		logMsg := strings.ToLower(strings.TrimSpace(logEntry.Log))

		if strings.HasSuffix(logMsg, startMarker) {
			break
		} else if strings.HasSuffix(logMsg, endMarker) {
			insideMarkers = true
			continue
		} else if insideMarkers {
			errorLogs = append(errorLogs, fmt.Sprintf("%v - %v", logEntry.Updated.Format(time.RFC3339), logEntry.Log))
		}
	}

	return errorLogs, nil
}

// collectClusterCreateFailureLogs returns the logs of events that resulted in cluster creation to not succeed
func collectClusterCreateFailureLogs(clusterID string, csClient *cs.Client) ([]string, error) {
	return collectClusterLogsInRange(
		clusterID,
		csClient,
		ack.AlibabaStartCreateClusterLog,
		ack.AlibabaCreateClusterFailedLog)
}

// collectClusterScaleFailureLogs returns the logs of events that resulted in cluster creation to not succeed
func collectClusterScaleFailureLogs(clusterID string, csClient *cs.Client) ([]string, error) {
	return collectClusterLogsInRange(
		clusterID,
		csClient,
		ack.AlibabaStartScaleClusterLog,
		ack.AlibabaScaleClusterFailedLog)
}

// collectClusterDeleteFailureLogs returns the logs of events that resulted in cluster deletion to not succeed
func collectClusterDeleteFailureLogs(clusterID string, csClient *cs.Client) ([]string, error) {
	return collectClusterLogsInRange(
		clusterID,
		csClient,
		ack.AlibabaStartDeleteClusterLog,
		ack.AlibabaDeleteClusterFailedLog)
}
