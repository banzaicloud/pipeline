// Copyright © 2020 Banzai Cloud
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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
)

// CreateEC2Client create a new ec2 instance with the credentials
func CreateEC2Client(credentials *credentials.Credentials, region string) (*ec2.EC2, error) {
	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials,
		Region:      &region,
	})
	if err != nil {
		return nil, err
	}

	return ec2.New(sess), nil
}

// CreateAWSCredentials create a 'Credentials' instance from secret's values
func CreateAWSCredentials(values map[string]string) *credentials.Credentials {
	return credentials.NewStaticCredentials(
		values[secrettype.AwsAccessKeyId],
		values[secrettype.AwsSecretAccessKey],
		"",
	)
}
