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
	"github.com/banzaicloud/pipeline/pkg/cluster/acsk"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ACSKClusterContext describes the common fields used across ACSK cluster create/update/delete operations
type ACSKClusterContext struct {
	ClusterName string
	CSClient    *cs.Client
	ECSClient   *ecs.Client
}

type ACSKClusterCreateUpdateContext struct {
	ACSKClusterContext
	acsk.AlibabaClusterCreateParams
}

func NewACSKClusterCreationContext(clusterName string, csClient *cs.Client,
	ecsClient *ecs.Client, clusterCreateParams acsk.AlibabaClusterCreateParams) *ACSKClusterCreateUpdateContext {
	return &ACSKClusterCreateUpdateContext{
		ACSKClusterContext: ACSKClusterContext{
			ClusterName: clusterName,
			CSClient:    csClient,
			ECSClient:   ecsClient,
		},
		AlibabaClusterCreateParams: clusterCreateParams,
	}
}

// UploadSSHKeyAction describes how to upload an SSH key
type UploadSSHKeyAction struct {
	context   *ACSKClusterCreateUpdateContext
	sshSecret *secret.SecretItemResponse
	log       logrus.FieldLogger
}

// NewUploadSSHKeyAction creates a new UploadSSHKeyAction
func NewUploadSSHKeyAction(log logrus.FieldLogger, context *ACSKClusterCreateUpdateContext, sshSecret *secret.SecretItemResponse) *UploadSSHKeyAction {
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
	req.KeyPairName = a.context.ClusterName
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
	req.KeyPairNames = a.context.ClusterName
	req.RegionId = a.context.AlibabaClusterCreateParams.RegionID

	_, err = ecsClient.DeleteKeyPairs(req)
	return
}

// CreateACSKClusterAction describes the properties of an Alibaba cluster creation
type CreateACSKClusterAction struct {
	context   *ACSKClusterCreateUpdateContext
	log       logrus.FieldLogger
	clusterID string
}

// NewCreateACSKClusterAction creates a new CreateACSKClusterAction
func NewCreateACSKClusterAction(log logrus.FieldLogger, creationContext *ACSKClusterCreateUpdateContext) *CreateACSKClusterAction {
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
	a.log.Infoln("EXECUTE CreateACSKClusterAction, cluster name")
	csClient := a.context.CSClient

	// setup cluster creation request
	params := a.context.AlibabaClusterCreateParams
	p, err := json.Marshal(&params)
	if err != nil {
		return nil, err
	}

	req := cs.CreateCreateClusterRequest()
	req.SetScheme(requests.HTTPS)
	req.SetDomain("cs.aliyuncs.com")
	req.SetContent(p)
	req.SetContentType("application/json")

	// do a cluster creation
	resp, err := csClient.CreateCluster(req)
	if err != nil {
		a.log.Errorf("CreateCluster error: %s", err)
		return nil, err
	}
	if !resp.IsSuccess() || resp.GetHttpStatus() < 200 || resp.GetHttpStatus() > 299 {
		a.log.Errorf("CreateCluster error status code is: %s", resp.GetHttpStatus())
		return nil, errors.Errorf("create cluster error the returned status code is %s", resp.GetHttpStatus())
	}

	// parse response
	var r acsk.AlibabaClusterCreateResponse
	err = json.Unmarshal(resp.GetHttpContentBytes(), &r)
	if err != nil {
		return nil, err
	}

	a.log.Infof("Alibaba cluster creating with id %s", r.ClusterID)

	//We need this field to be able to implement the UndoAction for ClusterCreate
	a.clusterID = r.ClusterID

	// wait for cluster created
	a.log.Info("Waiting for cluster...")
	cluster, err := a.waitUntilClusterCreateComplete(r.ClusterID)
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

func (a *CreateACSKClusterAction) UndoAction() (err error) {
	a.log.Info("EXECUTE UNDO CreateACSKClusterAction")

	csClient := a.context.CSClient

	req := cs.CreateDeleteClusterRequest()
	req.ClusterId = a.clusterID
	req.SetScheme(requests.HTTPS)
	req.SetDomain("cs.aliyuncs.com")

	resp, err := csClient.DeleteCluster(req)
	if err != nil {
		if sdkErr, ok := err.(*aliErrors.ServerError); ok {
			if strings.Contains(sdkErr.Message(), "ErrorClusterNotFound") {
				// Cluster has been already deleted
				return nil
			}
		}
		a.log.Errorf("DeleteClusterResponse: %#v\n", resp.BaseResponse)
		return err
	}

	if resp.GetHttpStatus() != http.StatusAccepted {
		return fmt.Errorf("unexpected http status code: %d", resp.GetHttpStatus())
	}

	return
}

func (a *CreateACSKClusterAction) waitUntilClusterCreateComplete(clusterID string) (*acsk.AlibabaDescribeClusterResponse, error) {
	var (
		r     *acsk.AlibabaDescribeClusterResponse
		state string
		err   error
	)
	for {
		r, err = a.getClusterDetails(clusterID)
		if err != nil {
			return r, err
		}

		if r.State != state {
			a.log.Infof("%s cluster %s", r.State, clusterID)
			state = r.State
		}

		switch r.State {
		case acsk.AlibabaClusterStateRunning:
			return r, nil
		case acsk.AlibabaClusterStateFailed:
			return nil, errors.New("The cluster creation failed")
		default:
			time.Sleep(time.Second * 5)
		}
	}
}
func (a *CreateACSKClusterAction) getClusterDetails(clusterID string) (r *acsk.AlibabaDescribeClusterResponse, err error) {

	csClient := a.context.CSClient

	req := cs.CreateDescribeClusterDetailRequest()
	req.SetScheme(requests.HTTPS)
	req.SetDomain("cs.aliyuncs.com")
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
