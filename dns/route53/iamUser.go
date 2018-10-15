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

package route53

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/pkg/amazon"
	"github.com/sirupsen/logrus"
)

// createIAMUser creates a Amazon IAM user with the given name and with no login access to console
// Returns the created IAM user in case of success
func (dns *awsRoute53) createIAMUser(userName *string) (*iam.User, error) {
	log := loggerWithFields(logrus.Fields{"userName": aws.StringValue(userName)})

	user, err := amazon.CreateIAMUser(dns.iamSvc, userName)
	if err != nil {
		log.Errorf("creating IAM user failed: %s", extractErrorMessage(err))
		return nil, err
	}

	log.Infoln("IAM user created")
	return user, nil
}

// getIAMUser retrieves the Amazon IAM user with the given user name
func (dns *awsRoute53) getIAMUser(userName *string) (*iam.User, error) {
	log := loggerWithFields(logrus.Fields{"userName": aws.StringValue(userName)})

	user, err := amazon.GetIAMUser(dns.iamSvc, userName)
	if err != nil {
		log.Errorf("retrieving IAM user failed: %s", extractErrorMessage(err))
		return nil, err
	}

	return user, nil

}

// deleteIAMUser deletes the Amazon IAM user with the given name
func (dns *awsRoute53) deleteIAMUser(userName *string) error {
	log := loggerWithFields(logrus.Fields{"userName": aws.StringValue(userName)})

	if err := amazon.DeleteIAMUser(dns.iamSvc, userName); err != nil {
		log.Errorf("deleting IAM user failed: %s", extractErrorMessage(err))
		return err
	}

	log.Info("IAM user deleted")

	return nil
}

// createAmazonAccessKey create Amazon access key for the IAM user identified by userName
func (dns *awsRoute53) createAmazonAccessKey(userName *string) (*iam.AccessKey, error) {
	log := loggerWithFields(logrus.Fields{"userName": aws.StringValue(userName)})

	accessKey, err := amazon.CreateUserAccessKey(dns.iamSvc, userName)
	if err != nil {
		log.Errorf("creating Amazon access key for IAM user failed: %s", extractErrorMessage(err))
		return nil, err
	}

	log.Infoln("Amazon access key for IAM user created")

	return accessKey, nil
}

// deleteAmazonAccessKey deletes the Amazon access key of the user
func (dns *awsRoute53) deleteAmazonAccessKey(userName, accessKeyId *string) error {
	log := loggerWithFields(logrus.Fields{"userName": aws.StringValue(userName), "accessKeyId": aws.StringValue(accessKeyId)})

	err := amazon.DeleteUserAccessKey(dns.iamSvc, userName, accessKeyId)
	if err != nil {
		log.Errorf("deleting Amazon access key failed: %s", extractErrorMessage(err))
		return err
	}

	log.Info("Amazon access key deleted")

	return nil
}

func getIAMUserName(org *auth.Organization) string {
	return fmt.Sprintf(iamUserNameTemplate, org.Name)
}
