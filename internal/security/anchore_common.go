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
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/secret"
)

const (
	anchoreUserUIDNameTpl = "%v-anchore-user"
)

func GetUserSecret(ctx context.Context, secretStore common.SecretStore, userName string, logger common.Logger) (string, error) {

	values, err := secretStore.GetSecretValues(ctx, secret.GenerateSecretIDFromName(userName))
	if err != nil {
		logger.Debug("failed to get the user secret")

		return "", errors.Wrap(err, "failed to get the newly stored secret")
	}

	password, ok := values["password"]
	if !ok {
		logger.Debug("there is no password in the secret")

		return "", errors.NewPlain("there is no password in the secret")
	}

	return password, nil
}

func GetCustomAnchoreCredentials(ctx context.Context, secretStore common.SecretStore, secretId string, logger common.Logger) (string, string, error) {
	logger.Debug("using custom anchore configuration")

	secretValues, err := secretStore.GetSecretValues(ctx, secretId)
	if err != nil {
		logger.Debug("failed to retrieve secret")

		return "", "", errors.WrapIf(err, "failed to retrieve custom anchore user secret")
	}

	credentials := struct {
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
	}{}

	if err := mapstructure.Decode(secretValues, &credentials); err != nil {
		logger.Debug("failed to decode secret values")

		return "", "", errors.WrapIf(err, "failed to decode custom anchore user secret")
	}

	return credentials.Username, credentials.Password, nil
}

// UserNameService defines operations for generating unique names using cluster data
type UserNameService interface {
	Generate(ctx context.Context, orgID uint, clusterID uint) (string, error)
}

type ClusterService interface {
	GetClusterUUID(context.Context, uint, uint) (string, error)
}

type AnchoreUserNameService struct {
	clusterService ClusterService
}

func NewAnchoreUsernameService(clusterService ClusterService) UserNameService {
	return AnchoreUserNameService{clusterService: clusterService}
}

// generates a unique username using the cluster's UUID
func (un AnchoreUserNameService) Generate(ctx context.Context, orgID uint, clusterID uint) (string, error) {
	uuid, err := un.clusterService.GetClusterUUID(ctx, orgID, clusterID)
	if err != nil {
		return "", errors.WrapIf(err, "failed to generate username")
	}

	return fmt.Sprintf(anchoreUserUIDNameTpl, uuid), nil
}
