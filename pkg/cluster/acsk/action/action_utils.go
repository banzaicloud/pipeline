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
	"github.com/banzaicloud/pipeline/pkg/cluster/acsk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func deleteCluster(clusterID string, csClient *cs.Client) error {

	if len(clusterID) == 0 {
		return nil
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

func waitUntilScalingInstanceCreated(log logrus.FieldLogger, essClient *ess.Client, regionId, scalingGroupID, scalingConfID string) ([]string, error) {
	log.Info("Waiting for instances to get ready")
	var instanceIds []string
	describeScalingInstancesrequest := ess.CreateDescribeScalingInstancesRequest()
	describeScalingInstancesrequest.SetScheme(requests.HTTPS)
	describeScalingInstancesrequest.SetDomain("ess."+ regionId +".aliyuncs.com")
	describeScalingInstancesrequest.SetContentType(requests.Json)

	describeScalingInstancesrequest.ScalingGroupId = scalingGroupID
	describeScalingInstancesrequest.ScalingConfigurationId = scalingConfID

	for {
		describeScalingInstancesResponse, err := essClient.DescribeScalingInstances(describeScalingInstancesrequest)
		if err != nil {
			return nil, err
		}

		for _, instance := range describeScalingInstancesResponse.ScalingInstances.ScalingInstance {
			if instance.HealthStatus == acsk.AlibabaInstanceHealthyStatus {
				instanceIds = append(instanceIds, instance.InstanceId)
				continue
			} else {
				time.Sleep(time.Second * 5)
				break
			}
		}
		if len(instanceIds) == len(describeScalingInstancesResponse.ScalingInstances.ScalingInstance){
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
			time.Sleep(time.Second * 5)
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
