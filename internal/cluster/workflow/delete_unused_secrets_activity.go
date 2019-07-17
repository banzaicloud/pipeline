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

package workflow

import (
	"context"

	"emperror.dev/emperror"
)

const DeleteUnusedClusterSecretsActivityName = "delete-unused-cluster-secrets"

type DeleteUnusedClusterSecretsActivityInput struct {
	OrganizationID uint
	ClusterUID     string
}

type DeleteUnusedClusterSecretsActivity struct {
	secrets SecretStore
}

type SecretStore interface {
	DeleteByClusterUID(organizationID uint, clusterUID string) error
}

func MakeDeleteUnusedClusterSecretsActivity(secrets SecretStore) DeleteUnusedClusterSecretsActivity {
	return DeleteUnusedClusterSecretsActivity{
		secrets: secrets,
	}
}

func (a DeleteUnusedClusterSecretsActivity) Execute(_ context.Context, input DeleteUnusedClusterSecretsActivityInput) error {
	return emperror.Wrap(a.secrets.DeleteByClusterUID(input.OrganizationID, input.ClusterUID), "failed to delete secrets by cluster unique ID")
}
