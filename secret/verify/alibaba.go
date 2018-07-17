package verify

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/cs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
)

const (
	defaultAlibabaRegion = "cn-beijing"
)

type alibabaVerify struct {
	credentials *credentials.AccessKeyCredential
}

// CreateAlibabaSecret create a new 'alibabaVerify' instance
func CreateAlibabaSecret(values map[string]string) *alibabaVerify {
	return &alibabaVerify{
		credentials: CreateAlibabaCredentials(values),
	}
}

var _ Verifier = (*alibabaVerify)(nil)

// VerifySecret validates Alibaba credentials
func (v *alibabaVerify) VerifySecret() error {
	client, err := CreateAlibabaECSClient(v.credentials, defaultAlibabaRegion, nil)
	if err != nil {
		return err
	}

	req := ecs.CreateDescribeRegionsRequest()
	req.SetScheme(requests.HTTPS)
	resp, err := client.DescribeRegions(req)
	if err != nil {
		return err
	}
	if resp.GetHttpStatus() != http.StatusOK {
		return errors.New("Unexpected http status code: " + strconv.Itoa(resp.GetHttpStatus()))
	}

	return nil
}

func CreateAlibabaCSClient(auth *credentials.AccessKeyCredential, regionID string, cfg *sdk.Config) (*cs.Client, error) {
	if cfg == nil {
		cfg = sdk.NewConfig()
	}
	cred := credentials.NewAccessKeyCredential(auth.AccessKeyId, auth.AccessKeySecret)
	return cs.NewClientWithOptions(defaultAlibabaRegion, cfg, cred)
}

func CreateAlibabaECSClient(auth *credentials.AccessKeyCredential, regionID string, cfg *sdk.Config) (*ecs.Client, error) {
	if cfg == nil {
		cfg = sdk.NewConfig()
	}
	cred := credentials.NewAccessKeyCredential(auth.AccessKeyId, auth.AccessKeySecret)
	return ecs.NewClientWithOptions(defaultAlibabaRegion, cfg, cred)
}

func CreateAlibabaCredentials(values map[string]string) *credentials.AccessKeyCredential {
	return credentials.NewAccessKeyCredential(
		values[pkgSecret.AlibabaAccessKeyId],
		values[pkgSecret.AlibabaSecretAccessKey],
	)
}
