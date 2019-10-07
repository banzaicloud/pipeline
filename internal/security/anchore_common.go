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
	"github.com/banzaicloud/pipeline/secret"
)

const (
	anchoreUserNameTpl = "%v-anchore-user"
)

func getUserName(clusterID uint) string {
	return fmt.Sprintf(anchoreUserNameTpl, clusterID)
}

// createUserSecret creates a new password type secret, and returns the newly generated password string
func getUserSecret(ctx context.Context, secretStore common.SecretStore, userName string, logger common.Logger) (string, error) {

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
