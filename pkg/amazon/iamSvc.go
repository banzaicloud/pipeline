package amazon

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
)

// GetIAMUser retrieves the Amazon IAM user with the given user name
func GetIAMUser(svc iamiface.IAMAPI, userName *string) (*iam.User, error) {

	user := &iam.GetUserInput{
		UserName: userName,
	}

	iamUser, err := svc.GetUser(user)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == iam.ErrCodeNoSuchEntityException {
				return nil, nil // no such IAM user
			}
		}

		return nil, err
	}

	return iamUser.User, nil
}

// CreateIAMUser creates a Amazon IAM user with the given name and with no login access to console
// Returns the created IAM user in case of success
func CreateIAMUser(svc iamiface.IAMAPI, userName *string) (*iam.User, error) {
	userInput := &iam.CreateUserInput{
		UserName: userName,
	}

	iamUser, err := svc.CreateUser(userInput)
	if err != nil {
		return nil, err
	}

	return iamUser.User, nil
}

// DeleteIAMUser deletes the Amazon IAM user with the given name
func DeleteIAMUser(svc iamiface.IAMAPI, userName *string) error {
	_, err := svc.DeleteUser(&iam.DeleteUserInput{UserName: userName})
	return err
}

// IsAmazonAccessKeyExists returns whether the specified IAM user has the given Amazon access key
func IsAmazonAccessKeyExists(svc iamiface.IAMAPI, userName, accessKeyId *string) (bool, error) {
	listAccessKeys := &iam.ListAccessKeysInput{
		UserName: userName,
	}

	accessKeys, err := svc.ListAccessKeys(listAccessKeys)
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

// GetUserAmazonAccessKeys returns the list of Amazon access keys of the given IAM user
func GetUserAmazonAccessKeys(svc iamiface.IAMAPI, userName *string) ([]*iam.AccessKeyMetadata, error) {
	listAccessKeys := &iam.ListAccessKeysInput{
		UserName: userName,
	}

	accessKeys, err := svc.ListAccessKeys(listAccessKeys)
	if err != nil {
		return nil, err
	}

	return accessKeys.AccessKeyMetadata, nil
}

// CreateAmazonAccessKey create Amazon access key for the IAM user identified by userName
func CreateAmazonAccessKey(svc iamiface.IAMAPI, userName *string) (*iam.AccessKey, error) {
	accessKeyInput := &iam.CreateAccessKeyInput{UserName: userName}

	accessKey, err := svc.CreateAccessKey(accessKeyInput)
	if err != nil {
		return nil, err
	}

	return accessKey.AccessKey, nil
}

// GetPolicy retrieves the IAM policy identified by the given Arn
func GetPolicy(svc iamiface.IAMAPI, arn string) (*iam.Policy, error) {
	getPolicy := &iam.GetPolicyInput{
		PolicyArn: aws.String(arn),
	}

	policy, err := svc.GetPolicy(getPolicy)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == iam.ErrCodeNoSuchEntityException {
				return nil, nil // no such policy
			}
		}

		return nil, err
	}

	return policy.Policy, nil

}

// CreatePolicy creates an AWS policy with given name, description and JSON policy document
func CreatePolicy(svc iamiface.IAMAPI, policyName, policyDocument, policyDescription *string) (*iam.Policy, error) {
	policyInput := &iam.CreatePolicyInput{
		Description:    policyDescription,
		PolicyName:     policyName,
		PolicyDocument: policyDocument,
	}

	policy, err := svc.CreatePolicy(policyInput)
	if err != nil {
		return nil, err
	}

	return policy.Policy, nil
}

// DeletePolicy deletes the policy identified by the specified arn
func DeletePolicy(svc iamiface.IAMAPI, policyArn *string) error {
	_, err := svc.DeletePolicy(&iam.DeletePolicyInput{PolicyArn: policyArn})

	return err
}

// IsUserPolicyAttached returns true is the policy given its Arn is attached to the specified IAM user
func IsUserPolicyAttached(svc iamiface.IAMAPI, userName, policyArn *string) (bool, error) {
	attachedUserPoliciesInput := &iam.ListAttachedUserPoliciesInput{UserName: userName}
	attachedUserPolicies, err := svc.ListAttachedUserPolicies(attachedUserPoliciesInput)
	if err != nil {
		return false, err
	}

	found := false
	for _, attachedPolicy := range attachedUserPolicies.AttachedPolicies {
		if aws.StringValue(attachedPolicy.PolicyArn) == aws.StringValue(policyArn) {
			found = true
			break
		}
	}

	return found, nil
}

// AttachUserPolicy attaches the policy identified by the given arn to the IAM user identified
// by the given name
func AttachUserPolicy(svc iamiface.IAMAPI, userName, policyArn *string) error {
	userPolicyInput := &iam.AttachUserPolicyInput{
		UserName:  userName,
		PolicyArn: policyArn,
	}
	_, err := svc.AttachUserPolicy(userPolicyInput)

	return err
}

// DetachUserPolicy detaches the access policy identified by policyArn from the IAM User identified by userName
func DetachUserPolicy(svc iamiface.IAMAPI, userName, policyArn *string) error {
	_, err := svc.DetachUserPolicy(&iam.DetachUserPolicyInput{
		PolicyArn: policyArn,
		UserName:  userName,
	})

	return err
}
