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
	"github.com/banzaicloud/pipeline/pkg/cluster/acsk"
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

	_, err := waitUntilClusterCreateOrScaleComplete(log, clusterID, csClient, true)
	if err != nil {
		if strings.Contains(err.Error(), "ErrorClusterNotFound") {
			return nil
		}

		return emperror.Wrap(err, "could not delete cluster!")
	}

	req := cs.CreateDeleteClusterRequest()
	req.ClusterId = clusterID
	req.SetScheme(requests.HTTPS)
	req.SetDomain(acsk.AlibabaApiDomain)

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

	return nil
}

func describeScalingInstances(essClient *ess.Client, asgId, scalingConfId, regionId string) (*ess.DescribeScalingInstancesResponse, error) {
	describeScalingInstancesRequest := ess.CreateDescribeScalingInstancesRequest()
	describeScalingInstancesRequest.SetScheme(requests.HTTPS)
	describeScalingInstancesRequest.SetDomain(fmt.Sprintf(acsk.AlibabaESSEndPointFmt, regionId))
	describeScalingInstancesRequest.SetContentType(requests.Json)

	describeScalingInstancesRequest.ScalingGroupId = asgId
	describeScalingInstancesRequest.ScalingConfigurationId = scalingConfId

	describeScalingInstancesResponse, err := essClient.DescribeScalingInstances(describeScalingInstancesRequest)
	if err != nil {
		return nil, emperror.WrapWith(err, "could not describe scaling instances", "scalingGroupId", asgId)
	}
	return describeScalingInstancesResponse, nil
}

func attachInstancesToCluster(log logrus.FieldLogger, clusterID string, instanceIds []string, csClient *cs.Client) (*acsk.AlibabaDescribeClusterResponse, error) {
	log.Info("Attaching nodepools to cluster")
	attachInstanceRequest := cs.CreateAttachInstancesRequest()
	attachInstanceRequest.SetScheme(requests.HTTPS)
	attachInstanceRequest.SetDomain(acsk.AlibabaApiDomain)
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

func deleteNodepools(log logrus.FieldLogger, nodePools []*model.ACSKNodePoolModel, essClient *ess.Client, regionId string) error {
	errChan := make(chan error, len(nodePools))
	defer close(errChan)

	for _, nodePool := range nodePools {
		go func(nodePool *model.ACSKNodePoolModel) {

			deleteSGRequest := ess.CreateDeleteScalingGroupRequest()
			deleteSGRequest.SetScheme(requests.HTTPS)
			deleteSGRequest.SetDomain(fmt.Sprintf(acsk.AlibabaESSEndPointFmt, regionId))
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

			_, err = waitUntilScalingInstanceCreated(log, essClient, regionId, nodePool)
			if err != nil {
				errChan <- err
				return
			}

			errChan <- nil
		}(nodePool)
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

func waitUntilScalingInstanceCreated(log logrus.FieldLogger, essClient *ess.Client, regionId string, nodePool *model.ACSKNodePoolModel) ([]string, error) {
	log.Infof("Waiting for instances to get ready in NodePool: %s", nodePool.Name)

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
			if instance.HealthStatus == acsk.AlibabaInstanceHealthyStatus {
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

func waitUntilClusterCreateOrScaleComplete(log logrus.FieldLogger, clusterID string, csClient *cs.Client, isClusterCreate bool) (*acsk.AlibabaDescribeClusterResponse, error) {
	var (
		r     *acsk.AlibabaDescribeClusterResponse
		state string
		err   error
	)
	for {
		r, err = getClusterDetails(clusterID, csClient)
		if err != nil {
			if strings.Contains(err.Error(), "timeout") {
				log.Warn(err)
				continue
			}
			return r, err
		}

		if r.State != state {
			log.Infof("%s cluster %s", r.State, clusterID)
			state = r.State
		}

		switch r.State {
		case acsk.AlibabaClusterStateRunning:
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
					return nil, AlibabaClusterFailureLogsError{clusterEventLogs: logs}
				}
			}

			return r, nil
		case acsk.AlibabaClusterStateFailed:
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

			return nil, AlibabaClusterFailureLogsError{clusterEventLogs: logs}
		default:
			time.Sleep(time.Second * 20)
		}
	}
}

func getClusterDetails(clusterID string, csClient *cs.Client) (r *acsk.AlibabaDescribeClusterResponse, err error) {

	req := cs.CreateDescribeClusterDetailRequest()
	req.SetScheme(requests.HTTPS)
	req.SetDomain(acsk.AlibabaApiDomain)
	req.ClusterId = clusterID

	resp, err := csClient.DescribeClusterDetail(req)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not get cluster details for ID: %s", clusterID)
	}
	if !resp.IsSuccess() || resp.GetHttpStatus() < 200 || resp.GetHttpStatus() > 299 {
		err = errors.Wrapf(err, "Unexpected http status code: %d", resp.GetHttpStatus())
		return
	}

	err = json.Unmarshal(resp.GetHttpContentBytes(), &r)
	return
}

// collectClusterLogs returns the event logs associated with the cluster identified by clusterID
func collectClusterLogs(clusterID string, csClient *cs.Client) ([]*acsk.AlibabaDescribeClusterLogResponseEntry, error) {
	clusterLogsRequest := cs.CreateDescribeClusterLogsRequest()
	clusterLogsRequest.ClusterId = clusterID
	clusterLogsRequest.SetScheme(requests.HTTPS)
	clusterLogsRequest.SetDomain(acsk.AlibabaApiDomain)

	clusterLogsResp, err := csClient.DescribeClusterLogs(clusterLogsRequest)

	if clusterLogsResp != nil {
		if !clusterLogsResp.IsSuccess() || clusterLogsResp.GetHttpStatus() < 200 || clusterLogsResp.GetHttpStatus() > 299 {
			return nil, errors.Wrapf(err, "Unexpected http status code: %d", clusterLogsResp.GetHttpStatus())
		}

		var clusterLogs []*acsk.AlibabaDescribeClusterLogResponseEntry
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
		acsk.AlibabaStartCreateClusterLog,
		acsk.AlibabaCreateClusterFailedLog)
}

// collectClusterScaleFailureLogs returns the logs of events that resulted in cluster creation to not succeed
func collectClusterScaleFailureLogs(clusterID string, csClient *cs.Client) ([]string, error) {
	return collectClusterLogsInRange(
		clusterID,
		csClient,
		acsk.AlibabaStartScaleClusterLog,
		acsk.AlibabaScaleClusterFailedLog)
}
