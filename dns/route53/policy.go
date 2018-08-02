package route53

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/sirupsen/logrus"
)

// getHostedZoneRoute53Policy retrieves the Route53 IAM policy identified by the given Arn
func (dns *awsRoute53) getHostedZoneRoute53Policy(arn string) (*iam.Policy, error) {
	getPolicy := &iam.GetPolicyInput{PolicyArn: aws.String(arn)}
	policy, err := dns.iamSvc.GetPolicy(getPolicy)
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

// createHostedZoneRoute53Policy creates an AWS policy that allows listing route53 hosted zones and recordsets in general
// also modifying only the records of the hosted zone identified by the given id.
func (dns *awsRoute53) createHostedZoneRoute53Policy(orgId uint, hostedZoneId string) (*iam.Policy, error) {
	log := loggerWithFields(logrus.Fields{"hostedzone": hostedZoneId})

	org, err := dns.getOrganization(orgId)
	if err != nil {
		log.Errorf("retrieving organization with id %d failed: %s", orgId, extractErrorMessage(err))
		return nil, err
	}

	policyInput := &iam.CreatePolicyInput{
		Description: aws.String(fmt.Sprintf("Access permissions for hosted zone of the '%s' organization", org.Name)),
		PolicyName:  aws.String(fmt.Sprintf(hostedZoneAccessPolicyNameTemplate, org.Name)),
		PolicyDocument: aws.String(fmt.Sprintf(
			`{
						"Version": "2012-10-17",
    				"Statement": [
							{
            		"Effect": "Allow",
            		"Action": "route53:ChangeResourceRecordSets",
                "Resource": "arn:aws:route53:::hostedzone/%s"
        			},
        			{
            		"Effect": "Allow",
								"Action": [
                	"route53:ListHostedZones",
                	"route53:ListResourceRecordSets"
            		],
            		"Resource": "*"
        			}
    				]
					}`, hostedZoneId),
		),
	}

	policy, err := dns.iamSvc.CreatePolicy(policyInput)
	if err != nil {
		log.Errorf("creating access policy for hosted zone failed: %s", extractErrorMessage(err))
		return nil, err
	}

	log.Infof("access policy for hosted zone created: arn=%s", aws.StringValue(policy.Policy.Arn))

	return policy.Policy, nil
}

// deletePolicy deletes the amazon policy identified by the provided arn
func (dns *awsRoute53) deletePolicy(policyArn *string) error {
	log := loggerWithFields(logrus.Fields{"policy": aws.StringValue(policyArn)})

	_, err := dns.iamSvc.DeletePolicy(&iam.DeletePolicyInput{PolicyArn: policyArn})
	if err != nil {
		log.Errorf("deleting access policy failed: %s", extractErrorMessage(err))
	}

	log.Info("access policy deleted")

	return err
}

// isUserPolicyAttached returns true is the policy given its Arn is attached to the specified IAM user
func (dns *awsRoute53) isUserPolicyAttached(userName, policyArn *string) (bool, error) {
	attachedUserPoliciesInput := &iam.ListAttachedUserPoliciesInput{UserName: userName}
	attachedUserPolicies, err := dns.iamSvc.ListAttachedUserPolicies(attachedUserPoliciesInput)
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

// attachUserPolicy attaches the policy identified by the given arn to the IAM user identified
// by the given name
func (dns *awsRoute53) attachUserPolicy(userName, policyArn *string) error {
	log := loggerWithFields(logrus.Fields{"userName": aws.StringValue(userName), "policy": aws.StringValue(policyArn)})

	userPolicyInput := &iam.AttachUserPolicyInput{
		UserName:  userName,
		PolicyArn: policyArn,
	}

	_, err := dns.iamSvc.AttachUserPolicy(userPolicyInput)
	if err != nil {
		log.Errorf("attaching access policy to IAM user failed: %s", extractErrorMessage(err))
		return err
	}

	log.Info("access policy attached to IAM user")

	return nil
}

// detachUserPolicy detaches the access policy identified by policyArn from the IAM User identified by userName
func (dns *awsRoute53) detachUserPolicy(userName, policyArn *string) error {
	log := loggerWithFields(logrus.Fields{"userName": aws.StringValue(userName), "policy": aws.StringValue(policyArn)})

	_, err := dns.iamSvc.DetachUserPolicy(&iam.DetachUserPolicyInput{PolicyArn: policyArn, UserName: userName})
	if err != nil {
		log.Errorf("detaching policy from IAM user failed: %s", extractErrorMessage(err))
		return err
	}

	log.Info("policy detached from IAM user")

	return nil
}
