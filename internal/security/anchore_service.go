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
	"fmt"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/common"
	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
)

const (
	anchoreUserNameTpl = "%v-anchore-user"
	accountEmail       = "banzai@banzaicloud.com"
)

// AnchoreUserService service interface for Anchore related operations
type AnchoreUserService interface {
	EnsureUser(ctx context.Context, orgID uint, clusterID uint) error
	RemoveUser(ctx context.Context, orgID uint, clusterID uint) error
}

// anchoreService component struct implementing anchore account related operations
type anchoreService struct {
	configService ConfigurationService
	secretStore   common.SecretStore
	logger        common.Logger
}

func MakeAnchoreUserService(cfgService ConfigurationService, secretStore common.SecretStore, logger common.Logger) AnchoreUserService {
	return anchoreService{
		configService: cfgService,
		logger:        logger,
	}
}

func (a anchoreService) EnsureUser(ctx context.Context, orgID uint, clusterID uint) error {
	// add method context to the logger
	fnLog := a.logger.WithFields(map[string]interface{}{"orgID": orgID, "clusterID": clusterID})
	fnLog.Info("ensuring anchore user")

	restClient, err := a.getClient(ctx, clusterID)
	if err != nil {
		fnLog.Debug("failed to check user")

		return errors.WrapIf(err, "failed to ensure anchore user")
	}

	exists, err := a.userExists(ctx, restClient, a.getUserName(clusterID))
	if err != nil {
		fnLog.Debug("failed to check user")

		return errors.WrapIf(err, "failed to ensure anchore user")
	}

	if exists {
		fnLog.Info("processing existing anchore user")

		err = a.ensureUserCredentials(ctx, clusterID)
		if err != nil {
			fnLog.Debug("failed to ensure user credentials")

			return errors.WrapIf(err, "failed to ensure user credentials")
		}

		fnLog.Info("existing anchore user processed")
		return nil
	}

	fnLog.Info("processing new anchore user")
	password, err := a.createUserSecret(ctx, orgID, clusterID)
	if err != nil {
		fnLog.Debug("failed to create secret for anchore user")

		return errors.WrapIf(err, "failed to ensure user credentials")
	}

	if err := a.createAccount(ctx, clusterID); err != nil {
		fnLog.Debug("failed to create anchore account")

		return errors.WrapIf(err, "failed to create anchore account")
	}

	if err := a.createUser(ctx, clusterID, password); err != nil {
		fnLog.Debug("failed to create anchore user")

		return errors.WrapIf(err, "failed to create anchore user")
	}

	return nil
}

func (a anchoreService) RemoveUser(ctx context.Context, orgID uint, clusterID uint) error {
	panic("implement me")
}

func (a anchoreService) getUserName(clusterID uint) string {
	return fmt.Sprintf(anchoreUserNameTpl, clusterID)
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
func (a anchoreService) ensureUserCredentials(ctx context.Context, client AnchoreClient, userName string) error {

	password, err := client.GetUserCreadentials(ctx, userName)
	if err != nil {
		a.logger.Debug("failed to get user credentials")

		return errors.Wrap(err, "failed to get user credentials")
	}

	_, err = a.storeCredentialsSecret(ctx, a.getUserName(clusterID), password)
	if err != nil {
		a.logger.Debug("failed to store user credentials as a secret")

		return errors.Wrap(err, "failed to store user credentials as a secret")
	}

	return nil
}

// createUserSecret creates a new password type secret, and returns the newly generated password string
func (a anchoreService) createUserSecret(ctx context.Context, orgID uint, clusterID uint) (string, error) {

	// a new password gets generated
	secretID, err := a.storeCredentialsSecret(ctx, a.getUserName(clusterID), "")
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
		return "", errors.NewPlain("there is no password in secret")
	}

	return password, nil
}

func (a anchoreService) createAccount(ctx context.Context, clusterID uint) error {
	if err := a.anchoreCli.CreateAccount(ctx, a.getUserName(clusterID), accountEmail); err != nil {
		a.logger.Debug("failed to create anchore account")

		return errors.Wrap(err, "failed to create anchore account")
	}

	a.logger.Info("created anchore account")
	return nil
}

func (a anchoreService) createUser(ctx context.Context, clusterID uint, secret interface{}) error {
	if err := a.anchoreCli.CreateUser(ctx, a.getUserName(clusterID), accountEmail); err != nil {
		a.logger.Debug("failed to create anchore user")

		return errors.Wrap(err, "failed to create anchore user")
	}

	a.logger.Info("created anchore user")
	return nil
}

// storeCredentialsSecret stores the passed in userName and password asa secret and returns the related secretID
func (a anchoreService) storeCredentialsSecret(ctx context.Context, userName string, password string) (string, error) {

	secretRequest := secret.CreateSecretRequest{
		Name: userName,
		Type: "password",
		Values: map[string]string{
			"username": userName,
			"password": password,
		},
		Tags: []string{
			secretTypes.TagBanzaiHidden,
		},
	}

	secretID, err := a.secretStore.Store(ctx, &secretRequest)
	if err != nil {
		a.logger.Debug("failed to store user credentials as a secret")

		return "", errors.Wrap(err, "failed to store anchore user secret")
	}

	return secretID, nil
}

func (a anchoreService) getClient(ctx context.Context, clusterID uint) (AnchoreClient, error) {
	cfg, err := a.configService.GetConfiguration(ctx, clusterID)
	if err != nil {
		a.logger.Debug("failed to get anchore configuration")

		return nil, errors.Wrap(err, "failed to get anchore configuration")
	}

	return MakeAnchoreClient(cfg, a.logger), nil

}
