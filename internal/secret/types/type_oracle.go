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
	"github.com/banzaicloud/pipeline/internal/secret"
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
)

const Oracle = "oracle"

const (
	FieldOracleUserOCID          = "user_ocid"
	FieldOracleTenancyOCID       = "tenancy_ocid"
	FieldOracleAPIKey            = "api_key"
	FieldOracleAPIKeyFingerprint = "api_key_fingerprint"
	FieldOracleRegion            = "region"
	FieldOracleCompartmentOCID   = "compartment_ocid"
)

type OracleType struct{}

func (OracleType) Name() string {
	return Oracle
}

func (OracleType) Definition() secret.TypeDefinition {
	return secret.TypeDefinition{
		Fields: []secret.FieldDefinition{
			{Name: FieldOracleUserOCID, Required: true, Description: "Your Oracle user OCID. Find more about, generating public key and fingerprint here: https://banzaicloud.com/docs/pipeline/secrets/providers/oci_auth_credentials/"},
			{Name: FieldOracleTenancyOCID, Required: true, Description: "Your tenancy OCID"},
			{Name: FieldOracleAPIKey, Required: true, Description: "Your public key"},
			{Name: FieldOracleAPIKeyFingerprint, Required: true, Description: "Fingerprint of you public key"},
			{Name: FieldOracleRegion, Required: true, Description: "Oracle region"},
			{Name: FieldOracleCompartmentOCID, Required: true, Description: "Your compartment OCID"},
		},
	}
}

func (t OracleType) Validate(data map[string]string) error {
	return validateDefinition(data, t.Definition())
}

func (t OracleType) Verify(data map[string]string) error {
	creds := &oci.Credential{
		UserOCID:          data[FieldOracleUserOCID],
		TenancyOCID:       data[FieldOracleTenancyOCID],
		APIKey:            data[FieldOracleAPIKey],
		APIKeyFingerprint: data[FieldOracleAPIKeyFingerprint],
		Region:            data[FieldOracleRegion],
		CompartmentOCID:   data[FieldOracleCompartmentOCID],
	}

	client, err := oci.NewOCI(creds)
	if err != nil {
		return err
	}

	return client.Validate()
}
