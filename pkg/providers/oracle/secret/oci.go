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

package secret

import (
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
)

// CreateOCICredential creates an 'oci.Credential' instance from secret's values
func CreateOCICredential(values map[string]string) *oci.Credential {
	return &oci.Credential{
		UserOCID:          values[secrettype.OracleUserOCID],
		TenancyOCID:       values[secrettype.OracleTenancyOCID],
		APIKey:            values[secrettype.OracleAPIKey],
		APIKeyFingerprint: values[secrettype.OracleAPIKeyFingerprint],
		Region:            values[secrettype.OracleRegion],
		CompartmentOCID:   values[secrettype.OracleCompartmentOCID],
	}
}
