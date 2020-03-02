// Copyright © 2019 Banzai Cloud
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

package restricted

import (
	"fmt"

	"github.com/banzaicloud/pipeline/src/secret"
)

// GlobalSecretStore object that wraps the main secret store and restricts access to certain items
// nolint: gochecknoglobals
var GlobalSecretStore *restrictedSecretStore

// InitSecretStore initializes the global secret store.
func InitSecretStore(store secretStore) {
	GlobalSecretStore = &restrictedSecretStore{
		secretStore: store,
	}
}

// restrictedSecretStore checks whether the user can access a certain secret.
// For now this only means checking for forbidden tags.
type restrictedSecretStore struct {
	secretStore
}

type secretStore interface {
	Delete(orgID uint, secretID string) error
	Get(orgID uint, secretID string) (*secret.SecretItemResponse, error)
	List(orgID uint, query *secret.ListSecretsQuery) ([]*secret.SecretItemResponse, error)
	Store(orgID uint, request *secret.CreateSecretRequest) (string, error)
	Update(orgID uint, secretID string, request *secret.CreateSecretRequest) error
	Verify(organizationID uint, secretID string) error
}

func (s *restrictedSecretStore) List(orgid uint, query *secret.ListSecretsQuery) ([]*secret.SecretItemResponse, error) {
	responseItems, err := s.secretStore.List(orgid, query)
	if err != nil {
		return nil, err
	}

	newResponseItems := []*secret.SecretItemResponse{}

	for _, item := range responseItems {
		if HasForbiddenTag(item.Tags) == nil {
			newResponseItems = append(newResponseItems, item)
		}
	}

	return newResponseItems, nil
}

func (s *restrictedSecretStore) Update(organizationID uint, secretID string, value *secret.CreateSecretRequest) error {
	if err := s.checkBlockingTags(organizationID, secretID); err != nil {
		return err
	}

	return s.secretStore.Update(organizationID, secretID, value)
}

func (s *restrictedSecretStore) Delete(organizationID uint, secretID string) error {
	if err := s.checkBlockingTags(organizationID, secretID); err != nil {
		return err
	}

	return s.secretStore.Delete(organizationID, secretID)
}

func (s *restrictedSecretStore) Verify(organizationID uint, secretID string) error {
	if err := s.checkBlockingTags(organizationID, secretID); err != nil {
		return err
	}

	return s.secretStore.Verify(organizationID, secretID)
}

func (s *restrictedSecretStore) checkBlockingTags(organizationID uint, secretID string) error {
	secretItem, err := s.secretStore.Get(organizationID, secretID)
	if err != nil {
		return err
	}

	// check forbidden tags
	if err := HasForbiddenTag(secretItem.Tags); err != nil {
		return err
	}

	// check read only tag
	if err := s.isSecretReadOnly(secretItem); err != nil {
		return err
	}

	return nil
}

func (s *restrictedSecretStore) checkForbiddenTags(organizationID uint, secretID string) error {
	secretItem, err := s.secretStore.Get(organizationID, secretID)
	if err != nil {
		return err
	}

	return HasForbiddenTag(secretItem.Tags)
}

func (s *restrictedSecretStore) isSecretReadOnly(secretItem *secret.SecretItemResponse) error {
	for _, tag := range secretItem.Tags {
		if tag == secret.TagBanzaiReadonly {
			return ReadOnlyError{
				SecretID: secretItem.ID,
			}
		}
	}

	return nil
}

// ReadOnlyError describes a secret error where it contains read only tag
type ReadOnlyError struct {
	SecretID string
}

func (roe ReadOnlyError) Error() string {
	return fmt.Sprintf("secret [%s] is read only, cannot be updated/deleted", roe.SecretID)
}

// ForbiddenError describes a secret error where it contains forbidden tag
type ForbiddenError struct {
	ForbiddenTag string
}

func (f ForbiddenError) Error() string {
	return fmt.Sprintf("secret contains a forbidden tag: %s", f.ForbiddenTag)
}

// HasForbiddenTag is looking for forbidden tags
func HasForbiddenTag(tags []string) error {
	for _, tag := range tags {
		for _, forbiddenTag := range secret.ForbiddenTags {
			if tag == forbiddenTag {
				return ForbiddenError{
					ForbiddenTag: tag,
				}
			}
		}
	}
	return nil
}
