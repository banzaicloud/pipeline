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
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	oracle "github.com/banzaicloud/pipeline/pkg/providers/oracle/secret"
)

// Verifier validates cloud credentials
type Verifier interface {
	VerifySecret() error
}

// NewVerifier create new instance which implements `Verifier` interface
func NewVerifier(cloudType string, values map[string]string) Verifier {
	switch cloudType {

	case pkgCluster.Alibaba:
		return CreateAlibabaSecret(values)
	case pkgCluster.Amazon:
		return CreateAWSSecret(values)
	case pkgCluster.Azure:
		return CreateAzureSecretVerifier(values)
	case pkgCluster.Google:
		return CreateGCPSecretVerifier(values)
	case pkgCluster.Oracle:
		return oracle.CreateOCISecret(values)
	default:
		return nil
	}
}
