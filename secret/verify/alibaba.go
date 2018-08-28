package verify

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
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
	client, err := createAlibabaECSClient(v.credentials, defaultAlibabaRegion)
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
func createAlibabaECSClient(auth *credentials.AccessKeyCredential, regionID string) (*ecs.Client, error) {
	cred := credentials.NewAccessKeyCredential(auth.AccessKeyId, auth.AccessKeySecret)
	return ecs.NewClientWithOptions(regionID, sdk.NewConfig(), cred)
}

func CreateAlibabaCredentials(values map[string]string) *credentials.AccessKeyCredential {
	return credentials.NewAccessKeyCredential(
		values[pkgSecret.AlibabaAccessKeyId],
		values[pkgSecret.AlibabaSecretAccessKey],
	)
}
