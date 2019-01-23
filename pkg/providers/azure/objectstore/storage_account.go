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

package objectstore

import (
	"context"

	"github.com/banzaicloud/pipeline/pkg/providers/azure"

	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2017-10-01/storage"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const storageAccountResourceType = "Microsoft.Storage/storageAccounts"

type storageAccount struct {
	client storage.AccountsClient
}

var (
	falseVal = false
	trueVal  = true
)

// NewStorageAccountClientFromSecret returns an Azure resource groups client from a secret.
func NewStorageAccountClientFromSecret(credentials azure.Credentials) storage.AccountsClient {
	return storage.NewAccountsClient(credentials.SubscriptionID)
}

// NewAuthorizedStorageAccountClientFromSecret returns an authorized Azure resource groups client from a secret.
func NewAuthorizedStorageAccountClientFromSecret(credentials azure.Credentials) (storageAccount, error) {
	cred := NewClientCredentialsConfigFromSecret(credentials)

	client := NewStorageAccountClientFromSecret(credentials)

	authorizer, err := cred.Authorizer()
	if err != nil {
		return storageAccount{client}, errors.Wrap(err, "failed to authorize")
	}

	client.Authorizer = authorizer

	return storageAccount{client}, nil
}

// GetStorageAccountKey returns a key for a storage account.
func (s *storageAccount) GetStorageAccountKey(resourceGroup, storageAccount string) (string, error) {
	keys, err := s.client.ListKeys(context.TODO(), resourceGroup, storageAccount)
	if err != nil {
		return "", emperror.With(
			errors.Wrap(err, "failed to retrieve keys for storage account"),
			"storage-account", storageAccount,
		)
	}

	key := (*keys.Keys)[0].Value

	return *key, nil
}

// GetAllStorageAccounts returns all storage accounts under the specified resource group
// using the Azure credentials referenced by the provided secret.
func (s *storageAccount) GetAllStorageAccounts(resourceGroup string) (*[]storage.Account, error) {
	storageAccountList, err := s.client.ListByResourceGroup(context.TODO(), resourceGroup)
	if err != nil {
		return nil, err
	}

	return storageAccountList.Value, nil
}

func (s *storageAccount) CheckStorageAccountExistence(resourceGroup, storageAccount string, logger logrus.FieldLogger) (*bool, error) {

	logger.Info("retrieving storage account name availability...")
	result, err := s.client.CheckNameAvailability(
		context.TODO(),
		storage.AccountCheckNameAvailabilityParameters{
			Name: to.StringPtr(storageAccount),
			Type: to.StringPtr(storageAccountResourceType),
		},
	)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve storage account name availability")
	}

	if *result.NameAvailable == false {
		// account name is already taken or it is invalid
		// retrieve the storage account
		if _, err = s.client.GetProperties(context.TODO(), resourceGroup, storageAccount); err != nil {
			logger.Errorf("could not retrieve storage account, %s", *result.Message)
			return nil, emperror.WrapWith(err, *result.Message, "storage_account", storageAccount, "resource_group", resourceGroup)
		}
		// storage name exists, available
		return &trueVal, nil
	}

	// storage name doesn't exist
	return &falseVal, nil
}

func (s *storageAccount) CreateStorageAccount(resourceGroup, storageAccount, location string, logger logrus.FieldLogger) error {
	logger = logger.WithFields(logrus.Fields{
		"resource_group":  resourceGroup,
		"storage_account": storageAccount,
	})

	logger.Info("creating storage account")

	future, err := s.client.Create(
		context.TODO(),
		resourceGroup,
		storageAccount,
		storage.AccountCreateParameters{
			Sku: &storage.Sku{
				Name: storage.StandardLRS,
			},
			Kind:     storage.BlobStorage,
			Location: to.StringPtr(location),
			AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{
				AccessTier: storage.Hot,
			},
		},
	)

	if err != nil {
		return errors.Wrap(err, "cannot create storage account")
	}

	logger.Info("storage account creation request sent")
	if future.WaitForCompletionRef(context.TODO(), s.client.Client) != nil {
		return err
	}

	logger.Info("storage account created")

	return nil
}
