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
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/banzaicloud/pipeline/model"
	"github.com/banzaicloud/pipeline/pkg/cluster/acsk"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ACSKClusterContext describes the common fields used across ACSK cluster create/update/delete operations
type ACSKClusterContext struct {
	ClusterID string
	CSClient  *cs.Client
	ECSClient *ecs.Client
}

// NewACSKClusterContext creates a new ACSKClusterContext
func NewACSKClusterContext(csClient *cs.Client,
	ecsClient *ecs.Client, clusterID string) *ACSKClusterContext {
	return &ACSKClusterContext{
		CSClient:  csClient,
		ECSClient: ecsClient,
		ClusterID: clusterID,
	}
}

// ACSKClusterCreateContext describes the fields used across ACSK cluster create operation
type ACSKClusterCreateContext struct {
	ACSKClusterContext
	acsk.AlibabaClusterCreateParams
	NodePools []*model.ACSKNodePoolModel
}

type AlibabaClusterFailureLogsError struct {
	clusterEventLogs []string
}

func (e AlibabaClusterFailureLogsError) Error() string {
	if len(e.clusterEventLogs) > 0 {
		return "\n" + strings.Join(e.clusterEventLogs, "\n")
	}

	return ""
}

// NewACSKClusterCreationContext creates a new ACSKClusterCreateContext
func NewACSKClusterCreationContext(
	csClient *cs.Client,
	ecsClient *ecs.Client,
	clusterCreateParams acsk.AlibabaClusterCreateParams,
	nodePools []*model.ACSKNodePoolModel,
) *ACSKClusterCreateContext {
	return &ACSKClusterCreateContext{
		ACSKClusterContext: ACSKClusterContext{
			CSClient:  csClient,
			ECSClient: ecsClient,
		},
		AlibabaClusterCreateParams: clusterCreateParams,
		NodePools:                  nodePools,
	}
}

// ACSKClusterDeleteContext describes the fields used across ACSK cluster delete operation
type ACSKClusterDeleteContext struct {
	ACSKClusterContext
}

// NewACSKClusterDeletionContext creates a new ACSKClusterDeleteContext
func NewACSKClusterDeletionContext(csClient *cs.Client,
	ecsClient *ecs.Client, clusterID string) *ACSKClusterDeleteContext {
	return &ACSKClusterDeleteContext{
		ACSKClusterContext: ACSKClusterContext{
			CSClient:  csClient,
			ECSClient: ecsClient,
			ClusterID: clusterID,
		},
	}
}

// UploadSSHKeyAction describes how to upload an SSH key
type UploadSSHKeyAction struct {
	context   *ACSKClusterCreateContext
	sshSecret *secret.SecretItemResponse
	log       logrus.FieldLogger
}

// NewUploadSSHKeyAction creates a new UploadSSHKeyAction
func NewUploadSSHKeyAction(log logrus.FieldLogger, context *ACSKClusterCreateContext, sshSecret *secret.SecretItemResponse) *UploadSSHKeyAction {
	return &UploadSSHKeyAction{
		context:   context,
		sshSecret: sshSecret,
		log:       log,
	}
}

// GetName returns the name of this UploadSSHKeyAction
func (a *UploadSSHKeyAction) GetName() string {
	return "UploadSSHKeyAction"
}

// ExecuteAction executes this UploadSSHKeyAction
func (a *UploadSSHKeyAction) ExecuteAction(input interface{}) (interface{}, error) {
	a.log.Info("EXECUTE UploadSSHKeyAction")
	ecsClient := a.context.ECSClient

	req := ecs.CreateImportKeyPairRequest()
	req.SetScheme(requests.HTTPS)
	req.KeyPairName = a.context.AlibabaClusterCreateParams.Name
	req.PublicKeyBody = strings.TrimSpace(secret.NewSSHKeyPair(a.sshSecret).PublicKeyData)
	req.RegionId = a.context.AlibabaClusterCreateParams.RegionID

	return ecsClient.ImportKeyPair(req)
}

// UndoAction rolls back this UploadSSHKeyAction
func (a *UploadSSHKeyAction) UndoAction() (err error) {
	a.log.Info("EXECUTE UNDO UploadSSHKeyAction")
	//delete uploaded keypair
	ecsClient := a.context.ECSClient

	req := ecs.CreateDeleteKeyPairsRequest()
	req.SetScheme(requests.HTTPS)
	req.KeyPairNames = a.context.AlibabaClusterCreateParams.Name
	req.RegionId = a.context.AlibabaClusterCreateParams.RegionID

	_, err = ecsClient.DeleteKeyPairs(req)
	return
}

// CreateACSKClusterAction describes the properties of an Alibaba cluster creation
type CreateACSKClusterAction struct {
	context *ACSKClusterCreateContext
	log     logrus.FieldLogger
}

// NewCreateACSKClusterAction creates a new CreateACSKClusterAction
func NewCreateACSKClusterAction(log logrus.FieldLogger, creationContext *ACSKClusterCreateContext) *CreateACSKClusterAction {
	return &CreateACSKClusterAction{
		context: creationContext,
		log:     log,
	}
}

// GetName returns the name of this CreateACSKClusterAction
func (a *CreateACSKClusterAction) GetName() string {
	return "CreateACSKClusterAction"
}

// ExecuteAction executes this CreateACSKClusterAction
func (a *CreateACSKClusterAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Infoln("EXECUTE CreateACSKClusterAction, cluster name", a.context.Name)
	csClient := a.context.CSClient

	// setup cluster creation request
	params := a.context.AlibabaClusterCreateParams
	p, err := json.Marshal(&params)
	if err != nil {
		return nil, err
	}

	req := cs.CreateCreateClusterRequest()
	req.SetScheme(requests.HTTPS)
	req.SetDomain(acsk.AlibabaApiDomain)
	req.SetContent(p)
	req.SetContentType(requests.Json)

	// do a cluster creation
	resp, err := csClient.CreateCluster(req)
	if err != nil {
		a.log.Errorf("CreateCluster error: %s", err)
		return nil, err
	}
	if !resp.IsSuccess() || resp.GetHttpStatus() < 200 || resp.GetHttpStatus() > 299 {
		a.log.Errorf("CreateCluster error status code is: %d", resp.GetHttpStatus())
		return nil, errors.Errorf("create cluster error the returned status code is %d", resp.GetHttpStatus())
	}

	// parse response
	var r acsk.AlibabaClusterCreateResponse
	err = json.Unmarshal(resp.GetHttpContentBytes(), &r)
	if err != nil {
		return nil, err
	}

	a.log.Infof("Alibaba cluster creating with id %s", r.ClusterID)

	// We need this field to be able to implement the UndoAction for ClusterCreate
	a.context.ClusterID = r.ClusterID

	// wait for cluster created
	a.log.Info("Waiting for cluster...")
	cluster, err := waitUntilClusterCreateOrScaleComplete(a.log, r.ClusterID, csClient, true)
	if err != nil {
		return nil, errors.Wrap(err, "cluster create failed")
	}

	return cluster, nil
}

// UndoAction rolls back this CreateACSKClusterAction
func (a *CreateACSKClusterAction) UndoAction() (err error) {
	a.log.Info("EXECUTE UNDO CreateACSKClusterAction")

	return deleteCluster(a.context.ClusterID, a.context.CSClient)
}

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
		errors.Wrapf(err, "Could not get cluster details for ID: %s", clusterID)
		return
	}
	if !resp.IsSuccess() || resp.GetHttpStatus() < 200 || resp.GetHttpStatus() > 299 {
		err = errors.Wrapf(err, "Unexpected http status code: %d", resp.GetHttpStatus())
		return
	}

	err = json.Unmarshal(resp.GetHttpContentBytes(), &r)
	return
}

// DeleteACSKClusterAction describes the properties of an Alibaba cluster deletion
type DeleteACSKClusterAction struct {
	context *ACSKClusterDeleteContext
	log     logrus.FieldLogger
}

// NewCreateACSKClusterAction creates a new CreateACSKClusterAction
func NewDeleteACSKClusterAction(log logrus.FieldLogger, deletionContext *ACSKClusterDeleteContext) *DeleteACSKClusterAction {
	return &DeleteACSKClusterAction{
		context: deletionContext,
		log:     log,
	}
}

// GetName returns the name of this DeleteACSKClusterAction
func (a *DeleteACSKClusterAction) GetName() string {
	return "DeleteACSKClusterAction"
}

// ExecuteAction executes this DeleteACSKClusterAction
func (a *DeleteACSKClusterAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Info("EXECUTE DeleteClusterAction")
	return nil, deleteCluster(a.context.ClusterID, a.context.CSClient)
}

// DeleteSSHKeyAction describes how to delete an SSH key
type DeleteSSHKeyAction struct {
	context        *ACSKClusterDeleteContext
	sshKeyName     string
	sshKeyRegionID string
	log            logrus.FieldLogger
}

// NewDeleteSSHKeyAction creates a new UploadSSHKeyAction
func NewDeleteSSHKeyAction(log logrus.FieldLogger, context *ACSKClusterDeleteContext, sshKeyName, regionID string) *DeleteSSHKeyAction {
	return &DeleteSSHKeyAction{
		context:        context,
		sshKeyName:     sshKeyName,
		sshKeyRegionID: regionID,
		log:            log,
	}
}

// GetName returns the name of this DeleteSSHKeyAction
func (a *DeleteSSHKeyAction) GetName() string {
	return "DeleteSSHKeyAction"
}

// ExecuteAction executes this UploadSSHKeyAction
func (a *DeleteSSHKeyAction) ExecuteAction(input interface{}) (interface{}, error) {
	a.log.Info("EXECUTE DeleteSSHKeyAction")
	ecsClient := a.context.ECSClient

	req := ecs.CreateDeleteKeyPairsRequest()
	req.SetScheme(requests.HTTPS)
	jsonData := []string{a.sshKeyName}
	marshaledValue, err := json.Marshal(jsonData)
	if err != nil {
		return nil, err
	}
	req.KeyPairNames = string(marshaledValue)
	req.RegionId = a.sshKeyRegionID

	return ecsClient.DeleteKeyPairs(req)
}

// UpdateACSKClusterAction describes the fields used across ACSK cluster update operation
type UpdateACSKClusterAction struct {
	log       logrus.FieldLogger
	nodePools []*model.ACSKNodePoolModel
	context   *ACSKClusterContext
}

// NewUpdateACSKClusterAction creates a new UpdateACSKClusterAction
func NewUpdateACSKClusterAction(log logrus.FieldLogger, nodepools []*model.ACSKNodePoolModel, clusterContext *ACSKClusterContext) *UpdateACSKClusterAction {
	return &UpdateACSKClusterAction{
		log:       log,
		nodePools: nodepools,
		context:   clusterContext,
	}
}

// GetName returns the name of this UpdateACSKClusterAction
func (a *UpdateACSKClusterAction) GetName() string {
	return "UpdateACSKClusterAction"
}

// ExecuteAction executes this UpdateACSKClusterAction
func (a *UpdateACSKClusterAction) ExecuteAction(input interface{}) (interface{}, error) {
	a.log.Infof("EXECUTE UpdateACSKClusterAction on cluster, %s", a.context.ClusterID)
	csClient := a.context.CSClient

	//setup cluster update request
	params := acsk.AlibabaScaleClusterParams{
		DisableRollback:    true,
		TimeoutMins:        60,
		WorkerInstanceType: a.nodePools[0].InstanceType,
		NumOfNodes:         a.nodePools[0].Count,
	}
	p, err := json.Marshal(&params)
	if err != nil {
		return nil, err
	}

	req := cs.CreateScaleClusterRequest()
	req.ClusterId = a.context.ClusterID
	req.SetScheme(requests.HTTPS)
	req.SetDomain(acsk.AlibabaApiDomain)
	req.SetContent(p)
	req.SetContentType(requests.Json)

	//do a cluster scale
	resp, err := csClient.ScaleCluster(req)
	if err != nil {
		a.log.Errorf("ScaleCluster error %s", err)
		return nil, err
	}

	// parse response
	var r acsk.AlibabaClusterCreateResponse
	err = json.Unmarshal(resp.GetHttpContentBytes(), &r)
	if err != nil {
		return nil, err
	}

	a.context.ClusterID = r.ClusterID

	cluster, err := waitUntilClusterCreateOrScaleComplete(a.log, r.ClusterID, csClient, false)
	if err != nil {
		return nil, errors.Wrap(err, "cluster scale failed")
	}

	return cluster, nil
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
