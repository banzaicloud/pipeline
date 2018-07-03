package oci

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/identity"
	"github.com/sirupsen/logrus"
)

// OCI is for managing OCI API calls
type OCI struct {
	config          common.ConfigurationProvider
	logger          *logrus.Logger
	Tenancy         identity.Tenancy
	CompartmentOCID string
}

// Credential describe OCI credentials for access
type Credential struct {
	UserOCID          string
	TenancyOCID       string
	CompartmentOCID   string
	APIKey            string
	APIKeyFingerprint string
	Region            string
}

// NewOCI creates a new OCI Config and gets and caches tenancy info
func NewOCI(credential *Credential) (oci *OCI, err error) {

	config := common.NewRawConfigurationProvider(credential.TenancyOCID, credential.UserOCID, credential.Region, credential.APIKeyFingerprint, credential.APIKey, nil)

	oci = &OCI{
		config:          config,
		logger:          logrus.New(),
		CompartmentOCID: credential.CompartmentOCID,
	}

	_, err = oci.GetTenancy()

	return oci, err
}

// SetLogger sets a logrus logger
func (oci *OCI) SetLogger(logger *logrus.Logger) {

	oci.logger = logger
}

// GetLogger gets the previously set logrus logger
func (oci *OCI) GetLogger() *logrus.Logger {

	return oci.logger
}

// Validate is validates the credentials by retrieving and checking the related tenancy information
func (oci *OCI) Validate() error {

	oci.GetTenancy()

	tenancyID, err := oci.config.TenancyOCID()
	if err != nil {
		return err
	}

	if tenancyID != *oci.Tenancy.Id {
		return fmt.Errorf("Invalid Tenancy ID: %s != %s", tenancyID, *oci.Tenancy.Id)
	}

	return nil
}

// GetTenancy gets and caches tenancy info
func (oci *OCI) GetTenancy() (t identity.Tenancy, err error) {

	if oci.Tenancy.Id != nil {
		return oci.Tenancy, nil
	}

	tenancyID, err := oci.config.TenancyOCID()
	if err != nil {
		return t, err
	}

	oci.Tenancy, err = oci.getTenancy(tenancyID)

	return oci.Tenancy, err
}

func (oci *OCI) getTenancy(id string) (t identity.Tenancy, err error) {

	client, err := identity.NewIdentityClientWithConfigurationProvider(oci.config)
	if err != nil {
		return t, err
	}

	r, err := client.GetTenancy(context.Background(), identity.GetTenancyRequest{
		TenancyId: &id,
	})

	return r.Tenancy, err
}
