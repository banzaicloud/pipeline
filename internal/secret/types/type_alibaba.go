// Copyright © 2020 Banzai Cloud
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

package types

import (
	"net/http"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	"github.com/banzaicloud/pipeline/internal/secret"
)

const Alibaba = "alibaba"

const (
	FieldAlibabaRegion          = "ALIBABA_REGION_ID"
	FieldAlibabaAccessKeyId     = "ALIBABA_ACCESS_KEY_ID"
	FieldAlibabaSecretAccessKey = "ALIBABA_ACCESS_KEY_SECRET"
)

type AlibabaType struct{}

func (AlibabaType) Name() string {
	return Alibaba
}

func (AlibabaType) Definition() secret.TypeDefinition {
	return secret.TypeDefinition{
		Fields: []secret.FieldDefinition{
			{Name: FieldAlibabaRegion, Required: false, Description: "Alibaba Cloud region"},
			{Name: FieldAlibabaAccessKeyId, Required: true, Description: "Your Alibaba Cloud access key id"},
			{Name: FieldAlibabaSecretAccessKey, Required: true, Description: "Your Alibaba Cloud secret access key id"},
		},
	}
}

func (t AlibabaType) Validate(data map[string]string) error {
	return validateDefinition(data, t.Definition())
}

const defaultAlibabaRegion = "cn-beijing"

// TODO: rewrite this function!
func (AlibabaType) Verify(data map[string]string) error {
	creds := credentials.NewAccessKeyCredential(
		data[FieldAlibabaAccessKeyId],
		data[FieldAlibabaSecretAccessKey],
	)

	client, err := ecs.NewClientWithOptions(defaultAlibabaRegion, sdk.NewConfig(), creds)
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
		return secret.NewValidationError("invalid credentials", nil)
	}

	return nil
}
