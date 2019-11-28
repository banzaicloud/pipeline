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

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/src/secret"
)

const UploadSSHKeyActivityName = "eks-upload-ssh-key"

//  UploadSSHKeyActivity responsible for uploading SSH key
type UploadSSHKeyActivity struct {
	awsSessionFactory *AWSSessionFactory
}

//  UploadSSHKeyActivityInput holds data needed to upload SSH key
type UploadSSHKeyActivityInput struct {
	EKSActivityInput
	SSHKeyName  string
	SSHSecretID string
}

//  UploadSSHKeyActivityOutput holds the output data of UploadSSHKeyActivity
type UploadSSHKeyActivityOutput struct {
}

//  UploadSSHKeyActivity instantiates a new  UploadSSHKeyActivity
func NewUploadSSHKeyActivity(awsSessionFactory *AWSSessionFactory) *UploadSSHKeyActivity {
	return &UploadSSHKeyActivity{
		awsSessionFactory: awsSessionFactory,
	}
}

func (a *UploadSSHKeyActivity) Execute(ctx context.Context, input UploadSSHKeyActivityInput) (*UploadSSHKeyActivityOutput, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"region", input.Region,
		"keyPairName", input.SSHKeyName,
		"sshSecretID", input.SSHSecretID,
	)

	output := UploadSSHKeyActivityOutput{}

	session, err := a.awsSessionFactory.New(input.OrganizationID, input.SecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	sshSecret, err := a.awsSessionFactory.GetSecretStore().Get(input.OrganizationID, input.SSHSecretID)
	if err = errors.WrapIf(err, "failed to retrieve SSH secret"); err != nil {
		return nil, err
	}

	sshKey := secret.NewSSHKeyPair(sshSecret)
	ec2srv := ec2.New(session)

	// create and import ssh key pair only if key pair with the same name doesn't exists yet
	_, err = ec2srv.DescribeKeyPairsWithContext(ctx, &ec2.DescribeKeyPairsInput{
		KeyNames: []*string{aws.String(input.SSHKeyName)},
	})

	keyPairNotFound := false
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "InvalidKeyPair.NotFound" {
			keyPairNotFound = true
		} else {
			return nil, errors.WrapIfWithDetails(err, "failed to get key pair from AWS", "keyName", input.SSHKeyName)
		}
	}

	if keyPairNotFound {
		logger.Info("importing ssh key pair")

		importKeyPairInput := &ec2.ImportKeyPairInput{
			// A unique name for the key pair.
			// KeyName is a required field
			KeyName: aws.String(input.SSHKeyName),

			// The public key. For API calls, the text must be base64-encoded. For command
			// line tools, base64 encoding is performed for you.
			//
			// PublicKeyMaterial is automatically base64 encoded/decoded by the SDK.
			//
			// PublicKeyMaterial is a required field
			PublicKeyMaterial: []byte(sshKey.PublicKeyData), // []byte `locationName:"publicKeyMaterial" type:"blob" required:"true"`
		}
		_, err = ec2srv.ImportKeyPairWithContext(ctx, importKeyPairInput)
		if err != nil {
			return nil, errors.WrapIf(err, "failed to import key pair")
		}
	} else {
		logger.Info("skip importing ssh key pair as already already exists")
	}

	return &output, nil
}
