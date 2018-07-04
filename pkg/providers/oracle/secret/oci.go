package secret

import (
	"github.com/banzaicloud/pipeline/pkg/providers/oracle/oci"
)

// Oracle keys
const (
	UserOCID          = "user_ocid"
	TenancyOCID       = "tenancy_ocid"
	APIKey            = "api_key"
	APIKeyFingerprint = "api_key_fingerprint"
	Region            = "region"
	CompartmentOCID   = "compartment_ocid"
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
		UserOCID:          values[UserOCID],
		TenancyOCID:       values[TenancyOCID],
		APIKey:            values[APIKey],
		APIKeyFingerprint: values[APIKeyFingerprint],
		Region:            values[Region],
		CompartmentOCID:   values[CompartmentOCID],
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
