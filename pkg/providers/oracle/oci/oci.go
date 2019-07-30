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

package oci

import (
	"fmt"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/identity"
	"github.com/sirupsen/logrus"
)

// OCI is for managing OCI API calls
type OCI struct {
	config          common.ConfigurationProvider
	logger          logrus.FieldLogger
	credential      *Credential
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
	Password          string
}

// NewOCI creates a new OCI Config and gets and caches tenancy info
func NewOCI(credential *Credential) (oci *OCI, err error) {

	config := common.NewRawConfigurationProvider(credential.TenancyOCID, credential.UserOCID, credential.Region, credential.APIKeyFingerprint, credential.APIKey, common.String(credential.Password))

	oci = &OCI{
		CompartmentOCID: credential.CompartmentOCID,

		config:     config,
		logger:     logrus.New(),
		credential: credential,
	}

	_, err = oci.GetTenancy()

	return oci, err
}

// ChangeRegion changes region in the config to the specified one
func (oci *OCI) ChangeRegion(regionName string) (err error) {

	i, err := oci.NewIdentityClient()
	if err != nil {
		return err
	}

	err = i.IsRegionAvailable(regionName)
	if err != nil {
		return err
	}

	credential := oci.credential
	config := common.NewRawConfigurationProvider(credential.TenancyOCID, credential.UserOCID, regionName, credential.APIKeyFingerprint, credential.APIKey, common.String(credential.Password))

	oci.config = config

	return nil
}

// SetLogger sets a logrus logger
func (oci *OCI) SetLogger(logger logrus.FieldLogger) {

	oci.logger = logger
}

// GetLogger gets the previously set logrus logger
func (oci *OCI) GetLogger() logrus.FieldLogger {

	return oci.logger
}

// Validate is validates the credentials by retrieving and checking the related tenancy information
func (oci *OCI) Validate() error {

	oci.GetTenancy() // nolint: errcheck

	tenancyID, err := oci.config.TenancyOCID()
	if err != nil {
		return err
	}

	if tenancyID != *oci.Tenancy.Id {
		return fmt.Errorf("Invalid Tenancy ID: %s != %s", tenancyID, *oci.Tenancy.Id)
	}

	// check Compartment OCID validity
	i, err := oci.NewIdentityClient()
	if err != nil {
		return err
	}

	_, err = i.GetCompartment(&oci.credential.CompartmentOCID)
	if err != nil {
		if err.Error() == "Service error:NotAuthorizedOrNotFound. Authorization failed or requested resource not found. http status code: 404" {
			err = fmt.Errorf("Invalid Compartment OCID: %s", oci.credential.CompartmentOCID)
		}

		return err
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

	i, err := oci.NewIdentityClient()
	if err != nil {
		return t, err
	}
	oci.Tenancy, err = i.GetTenancy(tenancyID)

	return oci.Tenancy, err
}

// GetConfig gives back oci.config
func (oci *OCI) GetConfig() common.ConfigurationProvider {

	return oci.config
}
