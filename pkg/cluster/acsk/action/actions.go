package action

import (
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/sirupsen/logrus"
)

// ACSKClusterContext describes the common fields used across ACSK cluster create/update/delete operations
type ACSKClusterContext struct {
	ClusterName  string
	RegionID     string
	ZoneID       string
	CSClient     *cs.Client
	ECSClient    *ecs.Client
}

type ACSKClusterCreateUpdateContext struct {
	ACSKClusterContext
	SSHKeyName         string
	SSHKey             *secret.SSHKeyPair
	VpcID              string
	SubNetIP           []string
}


func NewACSKClusterCreationContext(clusterName, zoneId, regionId, sshKeyName string, csClient *cs.Client, ecsClient *ecs.Client) *ACSKClusterCreateUpdateContext {
	return &ACSKClusterCreateUpdateContext{
		ACSKClusterContext: ACSKClusterContext{
			ClusterName:   clusterName,
			CSClient:   csClient,
			ECSClient:  ecsClient,
			ZoneID:     zoneId,
			RegionID:   regionId,
		},
		SSHKeyName: sshKeyName,
	}
}

// UploadSSHKeyAction describes how to upload an SSH key
type UploadSSHKeyAction struct {
	context   *ACSKClusterCreateUpdateContext
	sshSecret *secret.SecretItemResponse
	log       logrus.FieldLogger
}

// NewUploadSSHKeyAction creates a new UploadSSHKeyAction
func NewUploadSSHKeyAction(log logrus.FieldLogger,context *ACSKClusterCreateUpdateContext, sshSecret *secret.SecretItemResponse) *UploadSSHKeyAction {
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
	a.context.SSHKey = secret.NewSSHKeyPair(a.sshSecret)
	ecsClient := a.context.ECSClient

	req := ecs.CreateImportKeyPairRequest()
	req.SetScheme(requests.HTTPS)
	req.KeyPairName = a.context.ClusterName
	req.PublicKeyBody = strings.TrimSpace(a.context.SSHKey.PublicKeyData)
	req.RegionId = a.context.RegionID

	return ecsClient.ImportKeyPair(req)
}
