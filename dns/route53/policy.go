package route53

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/banzaicloud/pipeline/pkg/amazon"
	"github.com/sirupsen/logrus"
)

// createHostedZoneRoute53Policy creates an AWS policy that allows listing route53 hosted zones and record  sets in general
// also modifying only the records of the hosted zone identified by the given id.
func (dns *awsRoute53) createHostedZoneRoute53Policy(orgId uint, hostedZoneId string) (*iam.Policy, error) {
	log := loggerWithFields(logrus.Fields{"hostedzone": hostedZoneId})

	org, err := dns.getOrganization(orgId)
	if err != nil {
		log.Errorf("retrieving organization with id %d failed: %s", orgId, extractErrorMessage(err))
		return nil, err
	}

	policyName := aws.String(fmt.Sprintf(hostedZoneAccessPolicyNameTemplate, org.Name))
	policyDocument := aws.String(fmt.Sprintf(
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
					}`, hostedZoneId))
	policyDescription := aws.String(fmt.Sprintf("Access permissions for hosted zone of the '%s' organization", org.Name))

	policy, err := amazon.CreatePolicy(dns.iamSvc, policyName, policyDocument, policyDescription)
	if err != nil {
		log.Errorf("creating access policy for hosted zone failed: %s", extractErrorMessage(err))
		return nil, err
	}

	log.Infof("access policy for hosted zone created: arn=%s", aws.StringValue(policy.Arn))

	return policy, nil
}

// deletePolicy deletes the amazon policy identified by the provided arn
func (dns *awsRoute53) deletePolicy(policyArn *string) error {
	log := loggerWithFields(logrus.Fields{"policy": aws.StringValue(policyArn)})

	if err := amazon.DeletePolicy(dns.iamSvc, policyArn); err != nil {
		log.Errorf("deleting access policy failed: %s", extractErrorMessage(err))
		return err
	}

	log.Info("access policy deleted")

	return nil
}

// attachUserPolicy attaches the policy identified by the given arn to the IAM user identified
// by the given name
func (dns *awsRoute53) attachUserPolicy(userName, policyArn *string) error {
	log := loggerWithFields(logrus.Fields{"userName": aws.StringValue(userName), "policy": aws.StringValue(policyArn)})

	err := amazon.AttachUserPolicy(dns.iamSvc, userName, policyArn)
	if err != nil {
		log.Errorf("attaching access policy to IAM user failed: %s", extractErrorMessage(err))
		return err
	}

	log.Infoln("access policy attached to IAM user")

	return nil
}

// detachUserPolicy detaches the access policy identified by policyArn from the IAM User identified by userName
func (dns *awsRoute53) detachUserPolicy(userName, policyArn *string) error {
	log := loggerWithFields(logrus.Fields{"userName": aws.StringValue(userName), "policy": aws.StringValue(policyArn)})

	err := amazon.DetachUserPolicy(dns.iamSvc, userName, policyArn)
	if err != nil {
		log.Errorf("detaching policy from IAM user failed: %s", extractErrorMessage(err))
		return err
	}

	log.Infoln("policy detached from IAM user")

	return nil
}
