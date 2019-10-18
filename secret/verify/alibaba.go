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

package verify

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
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
		values[secrettype.AlibabaAccessKeyId],
		values[secrettype.AlibabaSecretAccessKey],
	)
}
