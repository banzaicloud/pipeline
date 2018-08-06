package route53

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/sirupsen/logrus"
)

// createIAMUser creates a Amazon IAM user with the given name and with no login access to console
// Returns the created IAM user in case of success
func (dns *awsRoute53) createIAMUser(userName *string) (*iam.User, error) {
	log := loggerWithFields(logrus.Fields{"userName": aws.StringValue(userName)})

	userInput := &iam.CreateUserInput{
		UserName: userName,
	}

	iamUser, err := dns.iamSvc.CreateUser(userInput)
	if err != nil {
		log.Errorf("creating IAM user failed: %s", extractErrorMessage(err))
		return nil, err
	}

	log.Info("IAM user created")

	return iamUser.User, nil
}

// getIAMUser retrieves the Amazon IAM user with the given user name
func (dns *awsRoute53) getIAMUser(userName *string) (*iam.User, error) {
	log := loggerWithFields(logrus.Fields{"userName": aws.StringValue(userName)})

	user := &iam.GetUserInput{
		UserName: userName,
	}

	iamUser, err := dns.iamSvc.GetUser(user)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == iam.ErrCodeNoSuchEntityException {
				return nil, nil // no such IAM user
			}
		}

		log.Errorf("retrieving IAM user failed: %s", extractErrorMessage(err))
		return nil, err
	}

	return iamUser.User, nil
}

// deleteIAMUser deletes the Amazon IAM user with the given name
func (dns *awsRoute53) deleteIAMUser(userName *string) error {
	log := loggerWithFields(logrus.Fields{"userName": aws.StringValue(userName)})

	if _, err := dns.iamSvc.DeleteUser(&iam.DeleteUserInput{UserName: userName}); err != nil {
		log.Errorf("deleting IAM user failed: %s", extractErrorMessage(err))
		return err
	}

	log.Info("IAM user deleted")

	return nil
}

// isAmazonAccessKeyExists returns whether the specified IAM user has the given Amazon access key
func (dns *awsRoute53) isAmazonAccessKeyExists(userName, accessKeyId *string) (bool, error) {
	listAccessKeys := &iam.ListAccessKeysInput{
		UserName: userName,
	}

	accessKeys, err := dns.iamSvc.ListAccessKeys(listAccessKeys)
	if err != nil {
		return false, err
	}

	found := false
	for _, accessKey := range accessKeys.AccessKeyMetadata {
		if aws.StringValue(accessKey.AccessKeyId) == aws.StringValue(accessKeyId) {
			found = true
			break
		}
	}

	return found, nil
}

// getUserAmazonAccessKeys returns the list of Amazon access keys of the given IAM user
func (dns *awsRoute53) getUserAmazonAccessKeys(userName *string) ([]*iam.AccessKeyMetadata, error) {
	listAccessKeys := &iam.ListAccessKeysInput{
		UserName: userName,
	}

	accessKeys, err := dns.iamSvc.ListAccessKeys(listAccessKeys)
	if err != nil {
		return nil, err
	}

	return accessKeys.AccessKeyMetadata, nil
}

// createAmazonAccessKey create Amazon access key for the IAM user identified by userName
func (dns *awsRoute53) createAmazonAccessKey(userName *string) (*iam.AccessKey, error) {
	log := loggerWithFields(logrus.Fields{"userName": aws.StringValue(userName)})

	accessKeyInput := &iam.CreateAccessKeyInput{UserName: userName}

	accessKey, err := dns.iamSvc.CreateAccessKey(accessKeyInput)
	if err != nil {
		log.Errorf("creating Amazon access key for IAM user failed: %s", extractErrorMessage(err))
		return nil, err
	}

	log.Info("Amazon access key for IAM user created")

	return accessKey.AccessKey, nil
}

// deleteAmazonAccessKey deletes the Amazon access key of the user
func (dns *awsRoute53) deleteAmazonAccessKey(userName, accessKeyId *string) error {
	log := loggerWithFields(logrus.Fields{"userName": aws.StringValue(userName), "accessKeyId": aws.StringValue(accessKeyId)})

	accessKeyInput := &iam.DeleteAccessKeyInput{AccessKeyId: accessKeyId, UserName: userName}

	_, err := dns.iamSvc.DeleteAccessKey(accessKeyInput)
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
