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

package action

import (
	"fmt"
	"strings"

	"emperror.dev/errors"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/src/secret"
)

// getSecretName returns the name that identifies the  cluster user access key in Vault
func getSecretName(userName string) string {
	return fmt.Sprintf("%s-key", strings.ToLower(userName))
}

// GetClusterUserAccessKeyIdAndSecretVault returns the AWS access key and access key secret from Vault
// for cluster user name
func GetClusterUserAccessKeyIdAndSecretVault(organizationID uint, userName string) (string, string, error) {
	secretName := getSecretName(userName)
	secretItem, err := secret.Store.GetByName(organizationID, secretName)
	if err != nil {
		return "", "", errors.WrapWithDetails(err, "failed to get secret from Vault", "secret", secretName)
	}
	clusterUserAccessKeyId := secretItem.Values[secrettype.AwsAccessKeyId]
	clusterUserSecretAccessKey := secretItem.Values[secrettype.AwsSecretAccessKey]

	return clusterUserAccessKeyId, clusterUserSecretAccessKey, nil
}
