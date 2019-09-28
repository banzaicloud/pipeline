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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/utils"
)

var _ utils.RevocableAction = (*UploadSSHKeyAction)(nil)

// UploadSSHKeyAction describes how to upload an SSH key
type UploadSSHKeyAction struct {
	context   *EksClusterCreateUpdateContext
	sshSecret *secret.SecretItemResponse
	log       logrus.FieldLogger
}

// NewUploadSSHKeyAction creates a new UploadSSHKeyAction
func NewUploadSSHKeyAction(log logrus.FieldLogger, context *EksClusterCreateUpdateContext, sshSecret *secret.SecretItemResponse) *UploadSSHKeyAction {
	return &UploadSSHKeyAction{
		context:   context,
		sshSecret: sshSecret,
		log:       log,
	}
}

// GetName returns the name of this UploadSSHKeyAction
func (a *UploadSSHKeyAction) GetName() string {
	return "UploadSSHKeyAction"
}

// ExecuteAction executes this UploadSSHKeyAction
func (a *UploadSSHKeyAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Info("EXECUTE UploadSSHKeyAction")

	a.context.SSHKey = secret.NewSSHKeyPair(a.sshSecret)
	ec2srv := ec2.New(a.context.Session)
	importKeyPairInput := &ec2.ImportKeyPairInput{
		// A unique name for the key pair.
		// KeyName is a required field
		KeyName: aws.String(a.context.SSHKeyName),

		// The public key. For API calls, the text must be base64-encoded. For command
		// line tools, base64 encoding is performed for you.
		//
		// PublicKeyMaterial is automatically base64 encoded/decoded by the SDK.
		//
		// PublicKeyMaterial is a required field
		PublicKeyMaterial: []byte(a.context.SSHKey.PublicKeyData), // []byte `locationName:"publicKeyMaterial" type:"blob" required:"true"`
	}
	output, err = ec2srv.ImportKeyPair(importKeyPairInput)
	return output, err
}

// UndoAction rolls back this UploadSSHKeyAction
func (a *UploadSSHKeyAction) UndoAction() (err error) {
	a.log.Info("EXECUTE UNDO UploadSSHKeyAction")
	// delete uploaded keypair
	ec2srv := ec2.New(a.context.Session)

	deleteKeyPairInput := &ec2.DeleteKeyPairInput{
		KeyName: aws.String(a.context.SSHKeyName),
	}
	_, err = ec2srv.DeleteKeyPair(deleteKeyPairInput)
	return err
}

// --

var _ utils.Action = (*DeleteSSHKeyAction)(nil)

// DeleteSSHKeyAction deletes a generated SSH key
type DeleteSSHKeyAction struct {
	context    *EksClusterDeletionContext
	SSHKeyName string
	log        logrus.FieldLogger
}

// NewDeleteSSHKeyAction creates a new DeleteSSHKeyAction
func NewDeleteSSHKeyAction(log logrus.FieldLogger, context *EksClusterDeletionContext, sshKeyName string) *DeleteSSHKeyAction {
	return &DeleteSSHKeyAction{
		context:    context,
		SSHKeyName: sshKeyName,
		log:        log,
	}
}

// GetName returns the name of this DeleteSSHKeyAction
func (a *DeleteSSHKeyAction) GetName() string {
	return "DeleteSSHKeyAction"
}

// ExecuteAction executes this DeleteSSHKeyAction
func (a *DeleteSSHKeyAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Info("EXECUTE DeleteSSHKeyAction")

	ec2srv := ec2.New(a.context.Session)
	deleteKeyPairInput := &ec2.DeleteKeyPairInput{
		KeyName: aws.String(a.SSHKeyName),
	}
	_, err = ec2srv.DeleteKeyPair(deleteKeyPairInput)
	if awsErr, ok := err.(awserr.Error); ok {
		if awsErr.Code() == iam.ErrCodeNoSuchEntityException {
			return nil, nil
		}
	}

	return nil, err
}
