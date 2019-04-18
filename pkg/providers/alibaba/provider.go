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

package alibaba

import (
	"fmt"
	"strings"
)

const Provider = "alibaba"

// GetESSServiceEndpoint returns the endpoint of the ESS Service in the
// given region (https://www.alibabacloud.com/help/doc-detail/25927.htm)
func GetESSServiceEndpoint(region string) string {

	region = strings.ToLower(region)

	switch strings.ToLower(region) {
	case "cn-zhangjiakou":
	case "cn-huhehaote":
	case "ap-southeast-2":
	case "ap-southeast-3":
	case "ap-southeast-5":
	case "ap-northeast-1":
	case "eu-west-1":
	case "eu-central-1":
	case "me-east-1":
	case "ap-south-1":
		return fmt.Sprintf("ess.%s.aliyuncs.com", region)
	}

	return "ess.aliyuncs.com"
}
