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

package types

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/banzaicloud/pipeline/internal/secret"
)

const Amazon = "amazon"

const (
	FieldAmazonRegion          = "AWS_REGION"
	FieldAmazonAccessKeyId     = "AWS_ACCESS_KEY_ID"
	FieldAmazonSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
)

type AmazonType struct {
	// Region is used for secret verification.
	Region string
}

func (AmazonType) Name() string {
	return Amazon
}

func (AmazonType) Definition() secret.TypeDefinition {
	return secret.TypeDefinition{
		Fields: []secret.FieldDefinition{
			{Name: FieldAmazonRegion, Required: false, IsSafeToDisplay: true, Description: "Amazon Cloud region"},
			{Name: FieldAmazonAccessKeyId, Required: true, IsSafeToDisplay: true, Description: "Your Amazon Cloud access key id"},
			{Name: FieldAmazonSecretAccessKey, Required: true, Description: "Your Amazon Cloud secret access key id"},
		},
	}
}

func (t AmazonType) Validate(data map[string]string) error {
	return validateDefinition(data, t.Definition())
}

// TODO: rewrite this function!
func (t AmazonType) Verify(data map[string]string) error {
	creds := credentials.NewStaticCredentials(
		data[FieldAmazonAccessKeyId],
		data[FieldAmazonSecretAccessKey],
		"",
	)

	sess, err := session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String(t.Region),
	})
	if err != nil {
		return err
	}

	client := ec2.New(sess)

	// currently the only way to verify AWS credentials is to actually use them to sign a request and see if it works
	// TODO: find a better way
	_, err = client.DescribeRegions(nil)

	if err != nil {
		return secret.NewValidationError(err.Error(), nil)
	}

	return nil
}
