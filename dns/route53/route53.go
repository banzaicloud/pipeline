package route53

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/banzaicloud/pipeline/config"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/banzaicloud/pipeline/pkg/cluster"
	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
)

var logger *logrus.Logger

func init() {
	logger = config.Logger()
}

const (
	createHostedZoneComment            = "HostedZone created by Banzaicloud Pipeline"
	iamUserNameTemplate                = "banzaicloud.route53.%s"
	hostedZoneAccessPolicyNameTemplate = "BanzaicloudRoute53-%s"
	iamUserAccessKeySecretName				 = "route53"
)

func loggerWithFields(fields logrus.Fields) *logrus.Entry {
	log := logger.WithFields(fields)
	fields["tag"] = "AmazonRoute53"

	return log
}

// rollback defines the signature for functions that implement
// the rolling back of an operation
type rollbackFunc func() error

// context represents an object that can be used pass data across function calls
type context struct {

	// state collects current state of domain across multiple function calls
	state *domainState

	// collection of rollback functions gathered across a chain of function calls
	//
	// in case any of the function calls fails the already succeeded ones in the chain
	// can be rolled back by running the rollback functions gather by context
	rollbackFunctions []rollbackFunc
}

// registerRollback registers the provided rollback function
func (ctx *context) registerRollback(f rollbackFunc) {
	ctx.rollbackFunctions = append(ctx.rollbackFunctions, f)
}

// rollback executes all registered rollback functions in reverse order (LIFO)
func (ctx *context) rollback() {
	log := loggerWithFields(logrus.Fields{})

	log.Info("rolling back...")
	errCount := 0
	for i := len(ctx.rollbackFunctions) - 1; i >= 0; i-- {
		err := ctx.rollbackFunctions[i]()

		if err != nil {
			errCount++
		}
	}

	if errCount == 0 {
		log.Info("rollback succeeded")
	} else if errCount < len(ctx.rollbackFunctions) {
		log.Info("rollback partially succeeded")
	} else {
		log.Info("rollback failed")
	}

}

// awsRoute53 represents Amazon Route53 DNS service
// and provides methods for managing domains through hosted zones
// and roles to control access to the hosted zones
type awsRoute53 struct {
	route53Svc route53iface.Route53API
	iamSvc     iamiface.IAMAPI
	stateStore awsRoute53StateStore
}

// NewAwsRoute53 creates a new awsRoute53 using the provided region and route53 credentials
func NewAwsRoute53(region, awsSecretId, awsSecretKey string) (*awsRoute53, error) {
	log := loggerWithFields(logrus.Fields{"region": region})

	creds := credentials.NewStaticCredentials(awsSecretId, awsSecretKey, "")

	config := aws.NewConfig().
		WithRegion(region).
		WithCredentials(creds)

	session, err := session.NewSession(config)

	if err != nil {
		log.Errorf("creating new Amazon session failed: %s", err.Error())
		return nil, err
	}

	return &awsRoute53{route53Svc: route53.New(session), iamSvc: iam.New(session), stateStore: &awsRoute53DatabaseStateStore{}}, nil
}

// IsDomainRegistered returns true if the domain has already been registered in Route53 for the given organisation
func (dns *awsRoute53) IsDomainRegistered(orgId uint, domain string) (bool, error) {
	log := loggerWithFields(logrus.Fields{"organisationId": orgId, "domain": domain})

	// check statestore to see if domain was already registered
	state := &domainState{}
	found, err := dns.stateStore.find(orgId, domain, state)
	if err != nil {
		log.Errorf("querying state store failed: %s", extractErrorMessage(err))
		return false, err
	}

	return found, nil
}

// RegisterDomain registers the given domain with AWS Route53
func (dns *awsRoute53) RegisterDomain(orgId uint, domain string) error {
	log := loggerWithFields(logrus.Fields{"organisationId": orgId, "domain": domain})

	// check statestore to see if domain was already registered
	found, err := dns.IsDomainRegistered(orgId, domain)
	if err != nil {
		return err
	}

	if found {
		msg := fmt.Sprintf("domain '%s' already registered", domain)
		log.Error(msg)
		return fmt.Errorf(msg)
	}

	// check Route53 to see if domain is not is use yet
	found, err = dns.hostedZoneExists(domain)
	if err != nil {
		log.Errorf("querying if domain is already in use failed: %s", extractErrorMessage(err))
		return err
	}

	if found {
		msg := fmt.Sprintf("domain '%s' is already in use", domain)
		log.Error(msg)
		return fmt.Errorf(msg)
	}

	state := &domainState{
		organisationId: orgId,
		domain:         domain,
		status:         CREATING,
	}

	if err := dns.stateStore.create(state); err != nil {
		log.Errorf("updating state store failed: %s", extractErrorMessage(err))
		return err
	}

	hostedZone, err := dns.createHostedZone(orgId, domain)
	if err != nil {
		dns.updateStateWithError(state, err)
		return err
	}

	strippedHostedZoneId := strings.Replace(aws.StringValue(hostedZone.Id), "/hostedzone/", "", 1)

	ctx := &context{state: state}

	// register rollback function
	ctx.registerRollback(func() error {
		return dns.deleteHostedZone(aws.String(strippedHostedZoneId))
	})

	state.hostedZoneId = strippedHostedZoneId
	if err := dns.stateStore.update(state); err != nil {
		log.Errorf("updating state store failed: %s", extractErrorMessage(err))

		ctx.rollback()
		return err
	}

	// set up auth for hosted zone
	if err := dns.setHostedZoneAuthorisation(strippedHostedZoneId, ctx); err != nil {
		// cleanup
		log.Errorf("setting authorisation for hosted zone '%s' failed: %s", strippedHostedZoneId, extractErrorMessage(err))

		ctx.rollback()

		dns.updateStateWithError(state, err)
		return err
	}

	log.Info("authorisation for hosted zone configured")

	state.status = CREATED
	if err := dns.stateStore.update(state); err != nil {
		log.Errorf("updating state store failed: %s", extractErrorMessage(err))

		ctx.rollback()
		return err
	}

	return nil
}

// UnregisterDomain delete the hosted zone with given domain from AWS Route53, also it removes the user access policy
// that was created to allow access to the hosted zone and the IAM user that was created for accessing the hosted zone.
func (dns *awsRoute53) UnregisterDomain(orgId uint, domain string) error {
	log := loggerWithFields(logrus.Fields{"organisationId": orgId, "domain": domain})

	log.Info("unregistering domain")

	state := &domainState{}
	found, err := dns.stateStore.find(orgId, domain, state)
	if err != nil {
		log.Errorf("querying state store failed: %s", extractErrorMessage(err))
		return err
	}

	if !found {
		msg := fmt.Sprintf("domain '%s' not found in state store", domain)
		log.Errorf(msg)
		return fmt.Errorf(msg)
	}

	state.status = REMOVING
	if err := dns.stateStore.update(state); err != nil {
		log.Errorf("updating state store failed: %s", extractErrorMessage(err))
		return err
	}

	// delete hosted zone
	if len(state.domain) > 0 {
		if err := dns.deleteHostedZone(aws.String(state.hostedZoneId)); err != nil {
			dns.updateStateWithError(state, err)
			return err
		}
	}

	// detach policy from user
	if len(state.iamUser) > 0 && len(state.policyArn) > 0 {
		if err := dns.detachUserPolicy(aws.String(state.iamUser), aws.String(state.policyArn)); err != nil {
			dns.updateStateWithError(state, err)
			return err
		}
	}

	// delete  access policy
	if len(state.policyArn) > 0 {
		if err := dns.deletePolicy(aws.String(state.policyArn)); err != nil {
			dns.updateStateWithError(state, err)
			return err
		}
	}

	// delete route53  access key
	if len(state.iamUser) > 0 && len(state.awsAccessKeyId) > 0 {
		if err := dns.deleteAmazonAccessKey(aws.String(state.iamUser), aws.String(state.awsAccessKeyId)); err != nil {
			dns.updateStateWithError(state, err)
			return err
		}
	}

	// delete IAM user
	if len(state.iamUser) > 0 {
		if err := dns.deleteIAMUser(aws.String(state.iamUser)); err != nil {
			dns.updateStateWithError(state, err)
			return err
		}
	}

	// delete route53 secret
	secrets, err := secret.Store.List(orgId,
		&secretTypes.ListSecretsQuery{
			Type: cluster.Amazon,
			Tag:  secretTypes.TagBanzaiHidden,
		})

	if err != nil {
		dns.updateStateWithError(state, err)
		return err
	}

	for _, item := range secrets {
		if item.Name == iamUserAccessKeySecretName {
			if err := secret.Store.Delete(orgId, item.ID); err != nil {
				dns.updateStateWithError(state, err)
				return err
			}

			break
		}
	}



	if err := dns.stateStore.delete(state); err != nil {
		log.Errorf("deleting domain state from state store failed: %s", extractErrorMessage(err))
		return err
	}

	log.Info("domain deleted")

	return nil
}

// createHostedZone creates a hosted zone on AWS Route53 with the given domain name
func (dns *awsRoute53) createHostedZone(orgId uint, domain string) (*route53.HostedZone, error) {
	log := loggerWithFields(logrus.Fields{"domain": domain})

	hostedZoneInput := &route53.CreateHostedZoneInput{
		CallerReference: aws.String(fmt.Sprintf("banzaicloud-pipepine-%d", time.Now().UnixNano())),
		Name:            aws.String(domain),
		HostedZoneConfig: &route53.HostedZoneConfig{
			Comment:     aws.String(createHostedZoneComment),
			PrivateZone: aws.Bool(false),
		},
	}

	hostedZoneOutput, err := dns.route53Svc.CreateHostedZone(hostedZoneInput)

	if err != nil {
		log.Errorf("creating Route53 hosted zone failed: %s", extractErrorMessage(err))
		return nil, err
	}

	log.Infof("route53 hosted zone created")

	return hostedZoneOutput.HostedZone, nil
}

// getHostedZone returns the hosted zone with given id from AWS Route53
func (dns *awsRoute53) getHostedZone(id *string) (*route53.HostedZone, error) {

	hostedZoneInput := &route53.GetHostedZoneInput{Id: id}
	hostedZoneOutput, err := dns.route53Svc.GetHostedZone(hostedZoneInput)
	if err != nil {
		return nil, err
	}

	return hostedZoneOutput.HostedZone, nil
}

// hostedZoneExists returns true if there is already a hosted zone created for the
// given domain in Route53
func (dns *awsRoute53) hostedZoneExists(domain string) (bool, error) {
	input := &route53.ListHostedZonesByNameInput{DNSName: aws.String(domain)}

	hostedZones, err := dns.route53Svc.ListHostedZonesByName(input)
	if err != nil {
		return false, err
	}

	found := false
	for _, hostedZone := range hostedZones.HostedZones {
		if aws.StringValue(hostedZone.Name) == domain {
			found = true
			break
		}
	}

	return found, nil
}

// deleteHostedZoneCallCount deletes the hosted zone with the given id from AWS Route53
func (dns *awsRoute53) deleteHostedZone(id *string) error {
	log := loggerWithFields(logrus.Fields{"hostedzone": aws.StringValue(id)})

	hostedZoneInput := &route53.DeleteHostedZoneInput{Id: id}

	_, err := dns.route53Svc.DeleteHostedZone(hostedZoneInput)
	if err != nil {
		log.Errorf("deleting hosted zone failed: %s", extractErrorMessage(err))
	}
	log.Infof("hosted zone deleted")

	return err
}

// setHostedZoneAuthorisation sets up authorisation for the Route53 hosted zone identified by the specified id.
// It creates a policy that allows changing only the specified hosted zone and a IAM user with the policy attached.
func (dns *awsRoute53) setHostedZoneAuthorisation(hostedZoneId string, ctx *context) error {
	log := loggerWithFields(logrus.Fields{"hostedzone": hostedZoneId})

	// create route53 policy
	policy, err := dns.createHostedZoneRoute53Policy(hostedZoneId)
	if err != nil {
		return err
	}

	ctx.registerRollback(func() error {
		return dns.deletePolicy(policy.Arn)
	})
	ctx.state.policyArn = aws.StringValue(policy.Arn)
	if err := dns.stateStore.update(ctx.state); err != nil {
		log.Errorf("failed to update state store: %s", extractErrorMessage(err))
		return err
	}

	// create IAM user
	userName := aws.String(fmt.Sprintf(iamUserNameTemplate, hostedZoneId))
	err = dns.createHostedZoneIAMUser(userName, policy.Arn, ctx)
	if err != nil {
		log.Errorf("setting up IAM user '%s' for hosted zone failed: %s", aws.StringValue(userName), extractErrorMessage(err))
		return err
	}
	log.Info("IAM user for hosted zone has been set up")

	return nil
}

// createHostedZoneRoute53Policy creates an AWS policy that allows listing route53 hosted zones and recordsets in general
// also modifying only the records of the hosted zone identified by the given id.
func (dns *awsRoute53) createHostedZoneRoute53Policy(hostedZoneId string) (*iam.Policy, error) {
	log := loggerWithFields(logrus.Fields{"hostedzone": hostedZoneId})

	policyInput := &iam.CreatePolicyInput{
		Description: aws.String(fmt.Sprintf("Access permissions for hosted zone '%s'", hostedZoneId)),
		PolicyName:  aws.String(fmt.Sprintf(hostedZoneAccessPolicyNameTemplate, hostedZoneId)),
		PolicyDocument: aws.String(fmt.Sprintf(
			`{
						"Version": "2012-10-17",
    				"Statement": [
							{
            		"Effect": "Allow",
            		"Action": "route53:ChangeResourceRecordSets",
            		"Resource": "arn:route53:route53:::hostedzone/%s"
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

// createHostedZoneIAMUser creates a IAM user and attaches the route53 policy identified by the given arn
func (dns *awsRoute53) createHostedZoneIAMUser(userName, route53PolicyArn *string, ctx *context) error {

	// create IAM User
	iamUser, err := dns.createIAMUser(userName)
	if err != nil {
		return err
	}

	ctx.registerRollback(func() error {
		return dns.deleteIAMUser(iamUser.UserName)
	})
	ctx.state.iamUser = aws.StringValue(iamUser.UserName)
	if err := dns.stateStore.update(ctx.state); err != nil {
		return err
	}

	// attach policy to user
	if err = dns.attachUserPolicy(iamUser.UserName, route53PolicyArn); err != nil {
		return err
	}

	ctx.registerRollback(func() error {
		return dns.detachUserPolicy(iamUser.UserName, route53PolicyArn)
	})

	// create route53 secret for IAM user
	awsAccessKey, err := dns.createAmazonAccessKey(iamUser.UserName)
	if err != nil {
		return err
	}

	ctx.registerRollback(func() error {
		return dns.deleteAmazonAccessKey(iamUser.UserName, awsAccessKey.AccessKeyId)
	})
	ctx.state.awsAccessKeyId = aws.StringValue(awsAccessKey.AccessKeyId)
	if err := dns.stateStore.update(ctx.state); err != nil {
		return err
	}

	// store route53 secret in Vault
	secretId, err := secret.Store.Store(ctx.state.organisationId, &secret.CreateSecretRequest{
		Name: iamUserAccessKeySecretName,
		Type: cluster.Amazon,
		Tags: []string{ secretTypes.TagBanzaiHidden },
	})
	ctx.registerRollback(func() error {
		return secret.Store.Delete(ctx.state.organisationId, secretId)
	})

	if err != nil {
		return err
	}

	return nil
}

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

func (dns *awsRoute53) updateStateWithError(state *domainState, err error) {
	state.status = FAILED
	state.errMsg = extractErrorMessage(err)

	dns.stateStore.update(state)
}

func extractErrorMessage(err error) string {
	if awsErr, ok := err.(awserr.Error); ok {
		return awsErr.Message()
	}

	return err.Error()
}
