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

package amazon

import (
	"github.com/pelletier/go-toml"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/banzaicloud/pipeline/src/secret"
)

type secretContents struct {
	ClusterCredentials credentials `toml:"default"`
	BucketCredentials  credentials `toml:"bucket"`
}

type credentials struct {
	KeyID string `toml:"aws_access_key_id"`
	Key   string `toml:"aws_secret_access_key"`
}

// GetSecret gets formatted secret for ARK
func GetSecret(clusterSecret, bucketSecret *secret.SecretItemResponse) (string, error) {
	a := secretContents{}

	if clusterSecret != nil {
		a.ClusterCredentials = credentials{
			KeyID: clusterSecret.Values[secrettype.AwsAccessKeyId],
			Key:   clusterSecret.Values[secrettype.AwsSecretAccessKey],
		}
	}

	if bucketSecret != nil {
		a.BucketCredentials = credentials{
			KeyID: bucketSecret.Values[secrettype.AwsAccessKeyId],
			Key:   bucketSecret.Values[secrettype.AwsSecretAccessKey],
		}
	}

	values, err := toml.Marshal(a)
	if err != nil {
		return "", err
	}

	return string(values), nil
}
