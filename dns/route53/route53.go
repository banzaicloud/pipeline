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
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/pkg/cluster"
	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/jinzhu/now"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
	"time"
)

var logger *logrus.Logger

func init() {
	logger = config.Logger()
}

const (
	createHostedZoneComment            = "HostedZone created by Banzaicloud Pipeline"
	iamUserNameTemplate                = "banzaicloud.route53.%s"
	hostedZoneAccessPolicyNameTemplate = "BanzaicloudRoute53-%s"
	iamUserAccessKeySecretName         = "route53"
)

func loggerWithFields(fields logrus.Fields) *logrus.Entry {
	fields["tag"] = "AmazonRoute53"
	log := logger.WithFields(fields)

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

	getOrganization func(orgId uint) (*auth.Organization, error)
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

	return &awsRoute53{route53Svc: route53.New(session), iamSvc: iam.New(session), stateStore: &awsRoute53DatabaseStateStore{}, getOrganization: getOrgById}, nil
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

	if found && (state.status == CREATING || state.status == REMOVING) {
		return false, fmt.Errorf("%s is in progress", state.status)
	}

	return found && state.status == CREATED, nil
}

// RegisterDomain registers the given domain with AWS Route53
func (dns *awsRoute53) RegisterDomain(orgId uint, domain string) error {
	log := loggerWithFields(logrus.Fields{"organisationId": orgId, "domain": domain})

	state := &domainState{}
	foundInStateStore, err := dns.stateStore.find(orgId, domain, state)
	if err != nil {
		log.Errorf("querying state store failed: %s", extractErrorMessage(err))
		return err
	}

	if foundInStateStore && (state.status == CREATING || state.status == REMOVING) {
		return fmt.Errorf("%s is in progress", state.status)
	}

	if foundInStateStore {
		state.errMsg = ""
		state.status = CREATING

		err = dns.stateStore.update(state)

	} else {
		state.organisationId = orgId
		state.domain = domain
		state.status = CREATING

		err = dns.stateStore.create(state)
	}

	if err != nil {
		log.Errorf("updating state store failed: %s", extractErrorMessage(err))
		return err
	}

	existingHostedZoneId, err := dns.hostedZoneExists(domain)
	if err != nil {
		log.Errorf("querying hosted zones for the domain failed: %s", extractErrorMessage(err))
		return err
	}

	hostedZoneIdShort := stripHostedZoneId(existingHostedZoneId)
	hostedZoneId := existingHostedZoneId
	ctx := &context{state: state}

	if existingHostedZoneId == "" {
		hostedZone, err := dns.createHostedZone(orgId, domain)
		if err != nil {
			dns.updateStateWithError(state, err)
			return err
		}

		hostedZoneIdShort = stripHostedZoneId(aws.StringValue(hostedZone.Id))
		hostedZoneId = aws.StringValue(hostedZone.Id)

	} else {
		log.Infof("skip creating hosted zone in route53 as it already exists with id: '%s'", hostedZoneIdShort)
	}

	// register rollback function
	ctx.registerRollback(func() error {
		return dns.deleteHostedZone(aws.String(hostedZoneId))
	})

	if state.hostedZoneId != hostedZoneIdShort {
		state.hostedZoneId = hostedZoneIdShort

		if err := dns.stateStore.update(state); err != nil {
			log.Errorf("updating state store failed: %s", extractErrorMessage(err))

			ctx.rollback()
			return err
		}
	}

	// set up authz for hosted zone
	if err := dns.setHostedZoneAuthorisation(hostedZoneIdShort, ctx); err != nil {
		// cleanup
		log.Errorf("setting authorisation for hosted zone '%s' failed: %s", hostedZoneIdShort, extractErrorMessage(err))

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

	org, err := dns.getOrganization(orgId)
	if err != nil {
		log.Errorf("retrieving organization details failed: %s", orgId, extractErrorMessage(err))
		return err
	}
	userName := getIAMUserName(org)
	iamUser, err := dns.getIAMUser(aws.String(userName))
	if err != nil {
		log.Errorf("querying IAM user with user name '%s' failed: %s", userName, extractErrorMessage(err))
		return err
	}

	// detach policy from user first to avoid access to hosted zone while it's being deleted
	if iamUser != nil && len(state.policyArn) > 0 {
		isPolicyAttached, err := dns.isUserPolicyAttached(aws.String(state.iamUser), aws.String(state.policyArn))
		if err != nil {
			log.Errorf("querying if policy '%s' is attached to user '%s' faied: %s", state.policyArn, state.iamUser, extractErrorMessage(err))
			return err
		}

		if isPolicyAttached {
			err := dns.detachUserPolicy(aws.String(state.iamUser), aws.String(state.policyArn))
			if err != nil {
				log.Errorf("detaching policy '%s' from IAM user '%s' failed: %s", state.policyArn, state.iamUser, extractErrorMessage(err))
				dns.updateStateWithError(state, err)
				return err
			}
		}
	}

	// delete  access policy
	if len(state.policyArn) > 0 {
		policy, err := dns.getHostedZoneRoute53Policy(state.policyArn)
		if err != nil {
			log.Errorf("querying policy '%s' failed: %s", state.policyArn, extractErrorMessage(err))
			return err
		}

		if policy != nil {
			if err := dns.deletePolicy(policy.Arn); err != nil {
				log.Errorf("deleting policy '%s' failed: %s", aws.StringValue(policy.Arn), extractErrorMessage(err))
				dns.updateStateWithError(state, err)
				return err
			}
		}

	}

	// delete route53  access keys
	if iamUser != nil {
		awsAccessKeys, err := dns.getUserAmazonAccessKeys(iamUser.UserName)
		if err != nil {
			log.Errorf("querying IAM user '%s' access keys failed: %s", state.iamUser, extractErrorMessage(err))
			dns.updateStateWithError(state, err)
			return err
		}
		for _, awsAccessKey := range awsAccessKeys {
			if err := dns.deleteAmazonAccessKey(awsAccessKey.UserName, awsAccessKey.AccessKeyId); err != nil {

				log.Errorf("deleting Amazon access key '%s' of user '%s' failed: %s",
					aws.StringValue(awsAccessKey.AccessKeyId),
					aws.StringValue(awsAccessKey.UserName), extractErrorMessage(err))

				dns.updateStateWithError(state, err)
				return err
			}
		}
	}

	// delete IAM user
	if iamUser != nil {
		if err := dns.deleteIAMUser(iamUser.UserName); err != nil {
			log.Errorf("deleting IAM user '%s' failed: %s", aws.StringValue(iamUser.UserName), extractErrorMessage(err))
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

	// delete hosted zone
	if len(state.domain) > 0 {
		hostedZoneId, err := dns.hostedZoneExists(state.domain)
		if err != nil {
			log.Errorf("checking if hosted zone for domain '%s' exists failed: %s", state.domain, extractErrorMessage(err))

			dns.updateStateWithError(state, err)
			return err
		}

		if hostedZoneId != "" {
			if err := dns.deleteHostedZone(aws.String(hostedZoneId)); err != nil {
				log.Errorf("deleting hosted zone '%s' failed: %s", hostedZoneId, extractErrorMessage(err))
				dns.updateStateWithError(state, err)
				return err
			}
		}
	}

	if err := dns.stateStore.delete(state); err != nil {
		log.Errorf("deleting domain state from state store failed: %s", extractErrorMessage(err))
		return err
	}

	log.Info("domain deleted")

	return nil
}

// Cleanup unregisters the domains that were registered for the given organizations
// with focus on optimizing hosted zones costs. This method expects a list of organizations
// that don't use route53 any more thus should be cleaned up
func (dns *awsRoute53) Cleanup() {
	log := loggerWithFields(logrus.Fields{})

	domainStates, err := dns.stateStore.listUnused()
	if err != nil {
		log.Errorf("retrieving domain states that are not used failed: %s", extractErrorMessage(err))
		return
	}

	for i := 0; i < len(domainStates); i++ {
		crtTime := time.Now()

		hostedZoneAge := crtTime.Sub(domainStates[i].createdAt)

		// According to Amazon Route53 pricing: https://aws.amazon.com/route53/pricing/
		//
		// To allow testing, a hosted zone that is deleted within 12 hours of creation is not charged

		if hostedZoneAge < 12*time.Hour { //grace period
			log.Infof("cleanup hosted zone '%s' as it is not used by organisation '%d' and it's age '%s' is less than 12hrs", domainStates[i].hostedZoneId, domainStates[i].organisationId, hostedZoneAge.String())

			err := dns.UnregisterDomain(domainStates[i].organisationId, domainStates[i].domain)
			if err != nil {
				log.Errorf("cleanup hosted zone '%s' failed: %s", domainStates[i].hostedZoneId, err.Error())
			}
		} else {
			// Since charging for hosted zones are not prorated for partial months if we exceeded the 12hrs we were already charged
			// It has no sense to delete the hosted zone until the next billing period starts (the first day of each subsequent month)
			// as the user may create a cluster in the organisation thus re-use the hosted zone that we were billed already for the month

			// If we are just before the next billing period and there are no clusters in the organisation we should cleanup the hosted zone
			// before entering the next billing period (the first day of the month)

			tillEndOfMonth := now.EndOfMonth().Sub(crtTime)

			maintenanceWindowMinute := viper.GetInt64("route53.maintenanceWindowMinute")

			if tillEndOfMonth <= time.Duration(maintenanceWindowMinute)*time.Minute {
				// if we are maintenanceWindowMinute minutes before the next billing period clean up the hosted zone

				// if the window is not long enough there will be few hosted zones slipping over into next billing period)
				log.Infof("cleanup hosted zone '%s' as it not used by organisation '%d'", domainStates[i].hostedZoneId, domainStates[i].organisationId)

				err := dns.UnregisterDomain(domainStates[i].organisationId, domainStates[i].domain)
				if err != nil {
					log.Errorf("cleanup hosted zone '%s' failed: %s", domainStates[i].hostedZoneId, err.Error())
				}
			}
		}

	}

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

// hostedZoneExists returns hosted zone id if there is already a hosted zone created for the
// given domain in Route53. If there are multiple hosted zones registered for the domain
// that is considered an error
func (dns *awsRoute53) hostedZoneExists(domain string) (string, error) {
	input := &route53.ListHostedZonesByNameInput{DNSName: aws.String(domain)}

	hostedZones, err := dns.route53Svc.ListHostedZonesByName(input)
	if err != nil {
		return "", err
	}

	var foundHostedZoneIds []string
	for _, hostedZone := range hostedZones.HostedZones {
		hostedZoneName := aws.StringValue(hostedZone.Name)
		hostedZoneName = hostedZoneName[:len(hostedZoneName)-1] // remove trailing '.' from name

		if hostedZoneName == domain {
			foundHostedZoneIds = append(foundHostedZoneIds, aws.StringValue(hostedZone.Id))
		}
	}

	if len(foundHostedZoneIds) > 1 {
		return "", fmt.Errorf("multiple hosted zones %v found for domain '%s'", foundHostedZoneIds, domain)
	}

	if len(foundHostedZoneIds) == 0 {
		return "", nil
	}

	return foundHostedZoneIds[0], nil
}

// deleteHostedZoneCallCount deletes the hosted zone with the given id from AWS Route53
func (dns *awsRoute53) deleteHostedZone(id *string) error {
	log := loggerWithFields(logrus.Fields{"hosted zone": aws.StringValue(id)})

	listResourceRecordSetsInput := &route53.ListResourceRecordSetsInput{HostedZoneId: id}
	resourceRecordSets, err := dns.route53Svc.ListResourceRecordSets(listResourceRecordSetsInput)
	if err != nil {
		log.Errorf("retrieving resource record sets of the hosted zone failed: %s", extractErrorMessage(err))
		return err
	}

	var resourceRecordSetChanges []*route53.Change

	for _, resourceRecordSet := range resourceRecordSets.ResourceRecordSets {
		if aws.StringValue(resourceRecordSet.Type) != "NS" && aws.StringValue(resourceRecordSet.Type) != "SOA" {
			resourceRecordSetChanges = append(resourceRecordSetChanges, &route53.Change{Action: aws.String("DELETE"), ResourceRecordSet: resourceRecordSet})
		}
	}

	if len(resourceRecordSetChanges) > 0 {
		deleteResourceRecordSets := &route53.ChangeResourceRecordSetsInput{
			HostedZoneId: id,
			ChangeBatch: &route53.ChangeBatch{
				Changes: resourceRecordSetChanges,
			},
		}
		_, err = dns.route53Svc.ChangeResourceRecordSets(deleteResourceRecordSets)
		if err != nil {
			log.Errorf("deleting all resource record sets of the hosted zone failed: %s", extractErrorMessage(err))
			return err
		}
	}

	hostedZoneInput := &route53.DeleteHostedZoneInput{Id: id}

	_, err = dns.route53Svc.DeleteHostedZone(hostedZoneInput)
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

	var policy *iam.Policy
	var err error

	if len(ctx.state.policyArn) > 0 {
		policy, err = dns.getHostedZoneRoute53Policy(ctx.state.policyArn)
		if err != nil {
			log.Errorf("retrieving route53 policy '%s' failed: %s", ctx.state.policyArn, extractErrorMessage(err))
			return err
		}
	}

	if policy == nil {
		// create route53 policy
		policy, err = dns.createHostedZoneRoute53Policy(ctx.state.organisationId, hostedZoneId)
		if err != nil {
			return err
		}
	} else {
		log.Infof("skip creating route53 policy for hosted zone as it already exists: arn='%s'", ctx.state.policyArn)
	}

	ctx.registerRollback(func() error {
		return dns.deletePolicy(policy.Arn)
	})

	if ctx.state.policyArn != aws.StringValue(policy.Arn) {
		ctx.state.policyArn = aws.StringValue(policy.Arn)
		if err = dns.stateStore.update(ctx.state); err != nil {
			log.Errorf("failed to update state store: %s", extractErrorMessage(err))
			return err
		}
	}

	// create IAM user
	org, err := dns.getOrganization(ctx.state.organisationId)
	if err != nil {
		log.Errorf("retrieving organization with id %d failed: %s", ctx.state.organisationId, extractErrorMessage(err))
		return err
	}

	userName := aws.String(getIAMUserName(org))
	err = dns.createHostedZoneIAMUser(userName, aws.String(ctx.state.policyArn), ctx)
	if err != nil {
		log.Errorf("setting up IAM user '%s' for hosted zone failed: %s", aws.StringValue(userName), extractErrorMessage(err))
		return err
	}
	log.Info("IAM user for hosted zone has been set up")

	return nil
}

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

// createHostedZoneIAMUser creates a IAM user and attaches the route53 policy identified by the given arn
func (dns *awsRoute53) createHostedZoneIAMUser(userName, route53PolicyArn *string, ctx *context) error {
	log := loggerWithFields(logrus.Fields{"IAMUser": aws.StringValue(userName), "policy": aws.StringValue(route53PolicyArn)})

	iamUser, err := dns.getIAMUser(userName)
	if err != nil {
		return err
	}

	if iamUser == nil {
		// create IAM User
		iamUser, err = dns.createIAMUser(userName)
		if err != nil {
			return err
		}
	}

	ctx.registerRollback(func() error {
		return dns.deleteIAMUser(iamUser.UserName)
	})

	if ctx.state.iamUser != aws.StringValue(iamUser.UserName) {
		ctx.state.iamUser = aws.StringValue(iamUser.UserName)

		if err := dns.stateStore.update(ctx.state); err != nil {
			return err
		}
	} else {
		log.Info("skip creating IAM user as it already exists")
	}

	// attach policy to user

	// check is the IAM user already has this policy attached
	policyAlreadyAttached, err := dns.isUserPolicyAttached(userName, route53PolicyArn)
	if err != nil {
		return err
	}

	if !policyAlreadyAttached {
		if err := dns.attachUserPolicy(aws.String(ctx.state.iamUser), route53PolicyArn); err != nil {
			return err
		}
	} else {
		log.Info("skip attaching policy to user as it is already attached")
	}

	ctx.registerRollback(func() error {
		return dns.detachUserPolicy(aws.String(ctx.state.iamUser), route53PolicyArn)
	})

	// setup Amazon access keys for IAM usser
	err = dns.setupAmazonAccess(aws.StringValue(userName), ctx)

	if err != nil {
		log.Errorf("setting up Amazon access key for user failed: %s", extractErrorMessage(err))
		return err
	}

	return nil
}

// setupAmazonAccess creates Amazon access key for the IAM user
// and stores it in Vault. If there is a stale Amazon access key in Vault
// creates a new Amazon access key and updates Vault
func (dns *awsRoute53) setupAmazonAccess(iamUser string, ctx *context) error {
	log := loggerWithFields(logrus.Fields{"userName": iamUser})

	// route53 secret from Vault
	route53Secret, err := dns.getRoute53Secret(ctx.state.organisationId)
	if err != nil {
		return err
	}

	// IAM user AWS access keys
	userAccessKeys, err := dns.getUserAmazonAccessKeys(aws.String(iamUser))
	if err != nil {
		return err
	}

	var userAccessKeyMap = make(map[string]*iam.AccessKeyMetadata)
	for _, userAccessKey := range userAccessKeys {
		userAccessKeyMap[aws.StringValue(userAccessKey.AccessKeyId)] = userAccessKey
	}

	// if either the Amazon access key or it's corresponding secret from Vault
	// we need to create(re-create in case of re-run) the Amazon access key
	// as the Amazon access secret can be obtained only at creation
	var createNewAccessKey = true
	var accessKeyId string

	if route53Secret != nil {
		if route53SecretAwsAccessKeyId, ok := route53Secret.Values[secretTypes.AwsAccessKeyId]; ok {
			if _, ok := userAccessKeyMap[route53SecretAwsAccessKeyId]; ok {
				createNewAccessKey = false // the access key in Amazon and Vault matches, no need to create a new onw
				accessKeyId = route53SecretAwsAccessKeyId
			}
		}
	}

	if !createNewAccessKey {
		if ctx.state.awsAccessKeyId != accessKeyId { // update state store as it contains stale access key id
			ctx.state.awsAccessKeyId = accessKeyId

			if err := dns.stateStore.update(ctx.state); err != nil {
				return err
			}
		}
		log.Info("skip creating Amazon access key for user as it is already set up")

		return nil
	}

	// new Amazon Access Key

	// remove old access key from Amazon if there is any
	if len(ctx.state.awsAccessKeyId) > 0 {
		if userAccessKey, ok := userAccessKeyMap[ctx.state.awsAccessKeyId]; ok {
			if err := dns.deleteAmazonAccessKey(userAccessKey.UserName, userAccessKey.AccessKeyId); err != nil {
				return err
			}
		}
	}

	if route53Secret != nil {
		if route53SecretAwsAccessKeyId, ok := route53Secret.Values[secretTypes.AwsAccessKeyId]; ok {
			if userAccessKey, ok := userAccessKeyMap[route53SecretAwsAccessKeyId]; ok {
				if err := dns.deleteAmazonAccessKey(userAccessKey.UserName, userAccessKey.AccessKeyId); err != nil {
					return err
				}
			}
		}
	}

	// create Amazon access key for user
	awsAccessKey, err := dns.createAmazonAccessKey(aws.String(iamUser))
	if err != nil {
		return err
	}

	ctx.registerRollback(func() error {
		return dns.deleteAmazonAccessKey(aws.String(iamUser), awsAccessKey.AccessKeyId)
	})

	ctx.state.awsAccessKeyId = aws.StringValue(awsAccessKey.AccessKeyId)

	if err := dns.stateStore.update(ctx.state); err != nil {
		return err
	}

	// store route53 secret in Vault
	if route53Secret != nil {
		err = dns.storeRoute53Secret(route53Secret, aws.StringValue(awsAccessKey.AccessKeyId), aws.StringValue(awsAccessKey.SecretAccessKey), ctx)
	} else {
		err = dns.storeRoute53Secret(nil, aws.StringValue(awsAccessKey.AccessKeyId), aws.StringValue(awsAccessKey.SecretAccessKey), ctx)
	}

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

// getRoute53Secret returns the secret from Vault that stores the IAM user
// aws access credentials that is used for accessing the Route53 Amazon service
func (dns *awsRoute53) getRoute53Secret(orgId uint) (*secret.SecretItemResponse, error) {
	awsAccessSecrets, err := secret.Store.List(orgId,
		&secretTypes.ListSecretsQuery{
			Type:   cluster.Amazon,
			Tag:    secretTypes.TagBanzaiHidden,
			Values: true,
		})

	if err != nil {
		return nil, err
	}

	// route53 secret
	var route53Secrets []*secret.SecretItemResponse
	for _, awsAccessSecret := range awsAccessSecrets {
		if awsAccessSecret.Name == iamUserAccessKeySecretName {
			route53Secrets = append(route53Secrets, awsAccessSecret)
		}
	}

	if len(route53Secrets) > 1 {
		return nil, fmt.Errorf("multiple secrets found with name '%s'", iamUserAccessKeySecretName)
	}

	if len(route53Secrets) == 1 {
		return route53Secrets[0], nil
	}

	return nil, nil
}

// storeRoute53Secret stores the provided Amazon access key in Vault
func (dns *awsRoute53) storeRoute53Secret(updateSecret *secret.SecretItemResponse, awsAccessKeyId, awsSecretAccessKey string, ctx *context) error {
	req := &secret.CreateSecretRequest{
		Name: iamUserAccessKeySecretName,
		Type: cluster.Amazon,
		Tags: []string{secretTypes.TagBanzaiHidden},
		Values: map[string]string{
			secretTypes.AwsAccessKeyId:     awsAccessKeyId,
			secretTypes.AwsSecretAccessKey: awsSecretAccessKey,
		},
	}

	var secretId string
	var err error

	if updateSecret != nil {
		ver := int(updateSecret.Version)
		req.Version = &ver

		if err = secret.Store.Update(ctx.state.organisationId, updateSecret.ID, req); err != nil {
			return err
		}
		secretId = updateSecret.ID
	} else {
		if secretId, err = secret.Store.Store(ctx.state.organisationId, req); err != nil {
			return err
		}
	}

	ctx.registerRollback(func() error {
		return secret.Store.Delete(ctx.state.organisationId, secretId)
	})

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

func stripHostedZoneId(id string) string {
	return strings.Replace(id, "/hostedzone/", "", 1)
}

func getIAMUserName(org *auth.Organization) string {
	return fmt.Sprintf(iamUserNameTemplate, org.Name)
}

func getOrgById(orgId uint) (*auth.Organization, error) {
	org, err := auth.GetOrganizationById(orgId)
	if err != nil {
		return nil, err
	}

	return org, nil
}
