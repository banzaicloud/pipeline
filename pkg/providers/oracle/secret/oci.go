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

package secret

import (
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
	pkgSecret "github.com/banzaicloud/pipeline/pkg/secret"
)

// OCIVerify for validation OCI credentials
type OCIVerify struct {
	credential *oci.Credential
}

// CreateOCISecret creates a new 'OCIVerify' instance
func CreateOCISecret(values map[string]string) *OCIVerify {
	return &OCIVerify{
		credential: CreateOCICredential(values),
	}
}

// CreateOCICredential creates an 'oci.Credential' instance from secret's values
func CreateOCICredential(values map[string]string) *oci.Credential {
	return &oci.Credential{
		UserOCID:          values[pkgSecret.OracleUserOCID],
		TenancyOCID:       values[pkgSecret.OracleTenancyOCID],
		APIKey:            values[pkgSecret.OracleAPIKey],
		APIKeyFingerprint: values[pkgSecret.OracleAPIKeyFingerprint],
		Region:            values[pkgSecret.OracleRegion],
		CompartmentOCID:   values[pkgSecret.OracleCompartmentOCID],
	}
}

// VerifySecret validates OCI credentials
func (a *OCIVerify) VerifySecret() (err error) {

	client, err := oci.NewOCI(a.credential)
	if err != nil {
		return err
	}

	return client.Validate()
}
