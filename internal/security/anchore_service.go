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

package anchore

import (
	"context"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/anchore"
	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/secret"
)

const (
	accountEmail = "banzai@banzaicloud.com"
)

// AnchoreUserService service interface for Anchore related operations
type AnchoreUserService interface {
	EnsureUser(ctx context.Context, orgID uint, clusterID uint) (string, error)
	RemoveUser(ctx context.Context, orgID uint, clusterID uint) error
}

// anchoreService component struct implementing anchore account related operations
type anchoreService struct {
	configProvider  anchore.ConfigProvider
	userNameService UserNameService
	secretStore     common.SecretStore

	logger common.Logger
}

// UserNameService defines operations for generating unique names using cluster data
type UserNameService interface {
	GenerateUsername(ctx context.Context, clusterID uint) (string, error)
}

func MakeAnchoreUserService(
	configProvider anchore.ConfigProvider,
	userNameService UserNameService,
	secretStore common.SecretStore,

	logger common.Logger,
) AnchoreUserService {
	return anchoreService{
		configProvider:  configProvider,
		userNameService: userNameService,
		secretStore:     secretStore,

		logger: logger,
	}
}

func (a anchoreService) EnsureUser(ctx context.Context, orgID uint, clusterID uint) (string, error) {
	// add method context to the logger
	fnCtx := map[string]interface{}{"orgID": orgID, "clusterID": clusterID}
	a.logger.Info("ensuring anchore user", fnCtx)

	restClient, err := a.getAnchoreClient(ctx, clusterID)
	if err != nil {
		a.logger.Debug("failed to set up anchore client for cluster", fnCtx)

		return "", errors.WrapIfWithDetails(err, "failed to set up anchore client for cluster", fnCtx)
	}

	userName, err := a.userNameService.GenerateUsername(ctx, clusterID)
	if err != nil {
		a.logger.Debug("failed to generate anchore username")

		return "", errors.Wrap(err, "failed to generate anchore username")
	}

	exists, err := a.userExists(ctx, restClient, userName)
	if err != nil {
		a.logger.Debug("failed to check user", fnCtx)

		return "", errors.WrapIfWithDetails(err, "failed to ensure anchore user", fnCtx)
	}

	if exists {
		a.logger.Info("processing existing anchore user", fnCtx)

		err = a.ensureUserCredentials(ctx, orgID, restClient, userName)
		if err != nil {
			a.logger.Debug("failed to ensure user credentials", fnCtx)

			return "", errors.WrapIfWithDetails(err, "failed to ensure user credentials", fnCtx)
		}

		a.logger.Info("existing anchore user processed", fnCtx)
		return userName, nil
	}

	a.logger.Info("processing new anchore user", fnCtx)
	password, err := a.createUserSecret(ctx, orgID, clusterID)
	if err != nil {
		a.logger.Debug("failed to create secret for anchore user", fnCtx)

		return "", errors.WrapIfWithDetails(err, "failed to ensure user credentials", fnCtx)
	}

	if err := a.ensureAccount(ctx, restClient, userName); err != nil {
		a.logger.Debug("failed to create anchore account", fnCtx)

		return "", errors.WrapIfWithDetails(err, "failed to create anchore account", fnCtx)
	}

	if err := a.createUser(ctx, restClient, userName, password); err != nil {
		a.logger.Debug("failed to create anchore user", fnCtx)

		return "", errors.WrapIfWithDetails(err, "failed to create anchore user", fnCtx)
	}

	return userName, nil
}

func (a anchoreService) RemoveUser(ctx context.Context, orgID uint, clusterID uint) error {
	// add method context
	fnCtx := map[string]interface{}{"orgID": orgID, "clusterID": clusterID}
	a.logger.Info("ensuring anchore user", fnCtx)

	restClient, err := a.getAnchoreClient(ctx, clusterID)
	if err != nil {
		a.logger.Debug("failed to set up anchore client for cluster")

		return errors.WrapIfWithDetails(err, "failed to set up anchore client for cluster", fnCtx)
	}

	userName, err := a.userNameService.GenerateUsername(ctx, clusterID)
	if err != nil {
		a.logger.Debug("failed to generate anchore username")

		return errors.Wrap(err, "failed to generate anchore username")
	}

	exists, err := a.userExists(ctx, restClient, userName)
	if err != nil {
		a.logger.Debug("failed to check if anchore user exists", fnCtx)

		return errors.WrapIfWithDetails(err, "failed to check if anchore user exists", fnCtx)
	}

	if exists {

		if err := a.deleteUser(ctx, restClient, userName, userName); err != nil {
			a.logger.Debug("failed to delete anchore user", fnCtx)

			return errors.WrapIfWithDetails(err, "failed to delete anchore user", fnCtx)
		}

		if err := a.deleteAccount(ctx, restClient, userName); err != nil {
			a.logger.Debug("failed to delete anchore account", fnCtx)

			return errors.WrapIfWithDetails(err, "failed to delete anchore account", fnCtx)
		}
	}

	// remove the secret if exists / ignore the error
	if err := a.secretStore.Delete(ctx, secret.GenerateSecretIDFromName(userName)); err != nil {

		a.logger.Debug("failed to delete credentials for nonexistent anchore user", fnCtx)
	}

	a.logger.Info("anchore user successfully removed")
	return nil
}

func (a anchoreService) userExists(ctx context.Context, client AnchoreClient, userName string) (bool, error) {

	anchoreUsr, err := client.GetUser(ctx, userName)
	if err != nil {
		a.logger.Debug("failed to retrieve anchore user")

		return false, errors.WrapIf(err, "failed to check if the user exists")
	}

	return anchoreUsr != nil, nil
}

//  ensureUserCredentials makes sure the user credentials secret is up to date
func (a anchoreService) ensureUserCredentials(ctx context.Context, orgID uint, client AnchoreClient, userName string) error {

	// check the user at anchore
	_, err := client.GetUserCredentials(ctx, userName)
	if err != nil {
		a.logger.Debug("failed to get user credentials")

		return errors.Wrap(err, "failed to get user credentials")
	}

	// check the user in vault
	_, err = a.secretStore.GetIDByName(ctx, userName)
	if err != nil {
		a.logger.Debug("failed to store user credentials as a secret")

		return errors.Wrap(err, "failed to store user credentials as a secret")
	}

	return nil
}

// createUserSecret creates a new password type secret, and returns the newly generated password string
func (a anchoreService) createUserSecret(ctx context.Context, orgID uint, clusterID uint) (string, error) {

	userName, err := a.userNameService.GenerateUsername(ctx, clusterID)
	if err != nil {
		a.logger.Debug("failed to generate anchore username")

		return "", errors.Wrap(err, "failed to generate anchore username")
	}

	// a new password gets generated
	secretID, err := a.storeCredentialsSecret(ctx, orgID, userName, "")
	if err != nil {
		a.logger.Debug("failed to store credentials for a new user")

		return "", errors.Wrap(err, "failed to store credentials for a new user")
	}

	values, err := a.secretStore.GetSecretValues(ctx, secretID)
	if err != nil {
		a.logger.Debug("failed to get the newly stored secret")

		return "", errors.Wrap(err, "failed to get the newly stored secret")
	}

	password, ok := values["password"]
	if !ok {
		a.logger.Debug("there is no password in the secret")

		return "", errors.NewPlain("there is no password in the secret")
	}

	return password, nil
}

func (a anchoreService) ensureAccount(ctx context.Context, client AnchoreClient, accountName string) error {
	// ignoring the error, trying to create an account if this call failss
	acc, _ := client.GetAccount(ctx, accountName)
	if acc != "" {
		a.logger.Debug("account already exists")
		return nil
	}

	if err := client.CreateAccount(ctx, accountName, accountEmail); err != nil {
		a.logger.Debug("failed to create anchore account")

		return errors.Wrap(err, "failed to create anchore account")
	}

	a.logger.Info("created anchore account")
	return nil
}

func (a anchoreService) createUser(ctx context.Context, client AnchoreClient, userName string, password string) error {
	// the account name is the same as the username
	if err := client.CreateUser(ctx, userName, userName, password); err != nil {
		a.logger.Debug("failed to create anchore user")

		return errors.Wrap(err, "failed to create anchore user")
	}

	a.logger.Info("created anchore user")
	return nil
}

// storeCredentialsSecret stores the passed in userName and password asa secret and returns the related secretID
func (a anchoreService) storeCredentialsSecret(ctx context.Context, orgID uint, userName string, password string) (string, error) {

	secretRequest := secret.CreateSecretRequest{
		Name: userName,
		Type: "password",
		Values: map[string]string{
			"username": userName,
			"password": password,
		},
		Tags: []string{
			secret.TagBanzaiHidden,
		},
	}

	// todo remove this global reference
	secretID, err := secret.Store.CreateOrUpdate(orgID, &secretRequest)
	if err != nil {
		a.logger.Debug("failed to store user credentials as a secret")

		return "", errors.Wrap(err, "failed to store anchore user secret")
	}

	return secretID, nil
}

// getAnchoreClient returns a rest client wrapper instance with the proper configuration
func (a anchoreService) getAnchoreClient(ctx context.Context, clusterID uint) (AnchoreClient, error) {
	config, err := a.configProvider.GetConfiguration(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	return NewAnchoreClient(config.User, config.Password, config.Endpoint, a.logger), nil
}

func (a anchoreService) deleteAccount(ctx context.Context, client AnchoreClient, accountName string) error {
	// function context for logging and error context
	fnCtx := map[string]interface{}{"accountName": accountName}

	if err := client.DeleteAccount(ctx, accountName); err != nil {
		a.logger.Debug("failed to delete anchore account", fnCtx)

		return errors.WrapIfWithDetails(err, "failed to delete anchore account", fnCtx)
	}

	a.logger.Info("anchore account deleted", fnCtx)
	return nil
}

func (a anchoreService) deleteUser(ctx context.Context, client AnchoreClient, accountName string, userName string) error {
	// function context for logging and error context
	fnCtx := map[string]interface{}{"accountName": accountName, "userName": userName}

	if err := client.DeleteUser(ctx, accountName, userName); err != nil {
		a.logger.Debug("failed to delete anchore user", fnCtx)

		return errors.WrapIfWithDetails(err, "failed to delete anchore user", fnCtx)
	}

	a.logger.Info("anchore user deleted", fnCtx)
	return nil
}
