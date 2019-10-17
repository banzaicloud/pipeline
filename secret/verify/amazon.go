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

package verify

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
)

const (
	DefaultRegion = "us-west-2" // TODO: move this to common amazon package?
)

// awsVerify for validation AWS credentials
type awsVerify struct {
	credentials *credentials.Credentials
}

// CreateAWSSecret create a new 'awsVerify' instance
func CreateAWSSecret(values map[string]string) *awsVerify {
	return &awsVerify{
		credentials: CreateAWSCredentials(values),
	}
}

// VerifySecret validates AKS credentials
func (a *awsVerify) VerifySecret() error {
	client, err := CreateEC2Client(a.credentials, DefaultRegion)
	if err != nil {
		return err
	}

	// currently the only way to verify AWS credentials is to actually use them to sign a request and see if it works
	// TODO: find a better way
	_, err = client.DescribeRegions(nil)
	return err
}

// CreateEC2Client create a new ec2 instance with the credentials
func CreateEC2Client(credentials *credentials.Credentials, region string) (*ec2.EC2, error) {

	// set aws log level
	var lv aws.LogLevelType
	if log.Level == logrus.DebugLevel {
		log.Info("set aws log level to debug")
		lv = aws.LogDebug
	} else {
		log.Info("set aws log off")
		lv = aws.LogOff
	}

	sess, err := session.NewSession(&aws.Config{
		Credentials: credentials,
		Region:      &region,
		LogLevel:    &lv,
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
