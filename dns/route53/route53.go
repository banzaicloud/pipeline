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
	"sync"
	"time"

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
	"github.com/banzaicloud/pipeline/pkg/amazon"
	"github.com/banzaicloud/pipeline/pkg/cluster"
	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/jinzhu/now"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var logger *logrus.Logger

func init() {
	logger = config.Logger()
}

const (
	createHostedZoneComment            = "HostedZone created by Banzai Cloud Pipeline"
	iamUserNameTemplate                = "banzaicloud.route53.%s"
	hostedZoneAccessPolicyNameTemplate = "BanzaicloudRoute53-%s"
	IAMUserAccessKeySecretName         = "route53"
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

type workerTask struct {
	operation        operationType
	organisationId   uint
	domain           *string
	dnsRecordownerId *string

	responseQueue chan<- workerResponse
}

type workerResponse struct {
	result interface{}
	error  error
}

// awsRoute53 represents Amazon Route53 DNS service
// and provides methods for managing domains through hosted zones
// and roles to control access to the hosted zones
type awsRoute53 struct {
	// used to sync access to workers
	muxWorkers sync.Mutex

	// orgDomainWorkers maps organisation id to the channel of the worker that
	// serves Route53 operations for the organisation. There is one worker allocated for each
	// organisation
	orgDomainWorkers map[uint]chan workerTask

	route53Svc       route53iface.Route53API
	iamSvc           iamiface.IAMAPI
	stateStore       awsRoute53StateStore
	baseHostedZoneId string // the id of the hosted zone of the base domain

	getOrganization func(orgId uint) (*auth.Organization, error)

	notificationChannel chan<- interface{}
	region              string
}

// NewAwsRoute53 creates a new awsRoute53 using the provided region and route53 credentials
func NewAwsRoute53(region, awsSecretId, awsSecretKey string, notifications chan interface{}) (*awsRoute53, error) {
	log := loggerWithFields(logrus.Fields{"region": region})

	baseDomain := viper.GetString(config.DNSBaseDomain)
	if len(baseDomain) == 0 {
		log.Errorf("base domain is not configured !")
		return nil, errors.New("base domain is not configured !")
	}

	creds := credentials.NewStaticCredentials(awsSecretId, awsSecretKey, "")

	config := aws.NewConfig().
		WithRegion(region).
		WithCredentials(creds)

	session, err := session.NewSession(config)

	if err != nil {
		log.Errorf("creating new Amazon session failed: %s", err.Error())
		return nil, err
	}

	awsRoute53 := &awsRoute53{
		route53Svc:          route53.New(session),
		iamSvc:              iam.New(session),
		stateStore:          &awsRoute53DatabaseStateStore{},
		getOrganization:     getOrgById,
		notificationChannel: notifications,
		region:              region,
	}

	baseHostedZoneId, err := awsRoute53.hostedZoneExistsByDomain(baseDomain)
	if err != nil {
		log.Errorf("retrieving hosted zone for base domain '%s' failed: %s", baseDomain, extractErrorMessage(err))
		return nil, err
	}

	if len(baseHostedZoneId) == 0 {
		return nil, fmt.Errorf("hosted zone for base domain '%s' not found", baseDomain)
	}

	awsRoute53.baseHostedZoneId = baseHostedZoneId

	return awsRoute53, nil
}

// IsDomainRegistered returns true if the domain has already been registered in Route53 for the given organisation
func (dns *awsRoute53) IsDomainRegistered(orgId uint, domain string) (bool, error) {
	responseQueue := make(chan workerResponse)

	dns.getWorker(orgId) <- newWorkerTask(isDomainRegistered, orgId, &domain, responseQueue)
	defer close(responseQueue)

	response := <-responseQueue
	return response.result.(bool), response.error
}

// RegisterDomain registers the given domain with AWS Route53
func (dns *awsRoute53) RegisterDomain(orgId uint, domain string) error {
	responseQueue := make(chan workerResponse)

	dns.getWorker(orgId) <- newWorkerTask(registerDomain, orgId, &domain, responseQueue)
	defer close(responseQueue)

	response := <-responseQueue

	if dns.notificationChannel != nil {
		if response.error != nil {
			dns.notificationChannel <- RegisterDomainFailedEvent{
				DomainEvent: *createCommonEvent(orgId, domain),
				Cause:       response.error,
			}

		} else {
			dns.notificationChannel <- RegisterDomainSucceededEvent{
				DomainEvent: *createCommonEvent(orgId, domain),
			}
		}
	}

	return response.error
}

// UnregisterDomain delete the hosted zone with given domain from AWS Route53, also it removes the user access policy
// that was created to allow access to the hosted zone and the IAM user that was created for accessing the hosted zone.
func (dns *awsRoute53) UnregisterDomain(orgId uint, domain string) error {
	responseQueue := make(chan workerResponse)

	dns.getWorker(orgId) <- newWorkerTask(unregisterDomain, orgId, &domain, responseQueue)
	defer close(responseQueue)

	response := <-responseQueue

	if dns.notificationChannel != nil {
		if response.error != nil {
			dns.notificationChannel <- UnregisterDomainFailedEvent{
				DomainEvent: *createCommonEvent(orgId, domain),
				Cause:       response.error,
			}

		} else {
			dns.notificationChannel <- UnregisterDomainSucceededEvent{
				DomainEvent: *createCommonEvent(orgId, domain),
			}
		}
	}

	return response.error
}

// isDomainRegistered returns true if the domain has already been registered in Route53 for the given organisation
func (dns *awsRoute53) isDomainRegistered(orgId uint, domain string) (bool, error) {
	log := loggerWithFields(logrus.Fields{"organisationId": orgId, "domain": domain})

	// check statestore to see if domain was already registered
	state := &domainState{}
	found, err := dns.stateStore.find(orgId, domain, state)
	if err != nil {
		log.Errorf("querying state store failed: %s", extractErrorMessage(err))
		return false, err
	}

	return found && state.status == CREATED, nil
}

// RegisterDomain registers the given domain with AWS Route53
func (dns *awsRoute53) registerDomain(orgId uint, domain string) error {
	log := loggerWithFields(logrus.Fields{"organisationId": orgId, "domain": domain})

	state := &domainState{}
	foundInStateStore, err := dns.stateStore.find(orgId, domain, state)
	if err != nil {
		log.Errorf("querying state store failed: %s", extractErrorMessage(err))
		return err
	}

	if foundInStateStore && state.status == REMOVING {
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

	existingHostedZoneId, err := dns.hostedZoneExistsByDomain(domain)
	if err != nil {
		log.Errorf("querying hosted zones for the domain failed: %s", extractErrorMessage(err))
		return err
	}

	hostedZoneIdShort := stripHostedZoneId(existingHostedZoneId)
	hostedZoneId := existingHostedZoneId
	ctx := &context{state: state}

	if existingHostedZoneId == "" {
		hostedZone, err := dns.createHostedZone(domain)
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

	// link the registered domain to base domain
	if err := dns.chainToBaseDomain(hostedZoneId, ctx); err != nil {
		log.Errorf("adding domain %q to base domain failed: %s", domain, extractErrorMessage(err))

		ctx.rollback()
		dns.updateStateWithError(state, err)
		return err
	}

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
func (dns *awsRoute53) unregisterDomain(orgId uint, domain string) error {
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

	if found && state.status == CREATING {
		return fmt.Errorf("%s is in progress", state.status)
	}

	state.status = REMOVING
	if err := dns.stateStore.update(state); err != nil {
		log.Errorf("updating state store failed: %s", extractErrorMessage(err))
		return err
	}

	org, err := dns.getOrganization(orgId)
	if err != nil {
		log.Errorf("retrieving organization details failed: %s", extractErrorMessage(err))
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
		isPolicyAttached, err := amazon.IsUserPolicyAttached(dns.iamSvc, aws.String(state.iamUser), aws.String(state.policyArn))
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
		policy, err := amazon.GetPolicy(dns.iamSvc, state.policyArn)
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
		awsAccessKeys, err := amazon.GetUserAccessKeys(dns.iamSvc, iamUser.UserName)
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
			Tags: []string{secretTypes.TagBanzaiHidden},
		})

	if err != nil {
		dns.updateStateWithError(state, err)
		return err
	}

	for _, item := range secrets {
		if item.Name == IAMUserAccessKeySecretName {
			if err := secret.Store.Delete(orgId, item.ID); err != nil {
				dns.updateStateWithError(state, err)
				return err
			}

			break
		}
	}

	// delete hosted zone
	if len(state.domain) > 0 {
		hostedZoneId, err := dns.hostedZoneExistsByDomain(state.domain)
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

	// unlink from parent base domain
	if err := dns.unChainFromBaseDomain(state.domain); err != nil {
		log.Errorf("removing domain '%s' from base domain failed: %s", state.domain, extractErrorMessage(err))
		dns.updateStateWithError(state, err)
		return err
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
		log.Errorf("retrieving domains that are not used failed: %s", extractErrorMessage(err))
		return
	}

	if len(domainStates) == 0 {
		return
	}

	var wg sync.WaitGroup

	wg.Add(len(domainStates))
	for _, domainState := range domainStates {
		go dns.cleanup(&wg, &domainState)
	}

	wg.Wait()
}

func (dns *awsRoute53) cleanup(wg *sync.WaitGroup, domainState *domainState) {
	log := loggerWithFields(logrus.Fields{})

	defer wg.Done()

	crtTime := time.Now()

	hostedZoneAge := crtTime.Sub(domainState.createdAt)

	// According to Amazon Route53 pricing: https://aws.amazon.com/route53/pricing/
	//
	// To allow testing, a hosted zone that is deleted within 12 hours of creation is not charged

	if hostedZoneAge < 12*time.Hour { //grace period
		log.Infof("cleanup hosted zone '%s' as it is not used by organisation '%d' and it's age '%s' is less than 12hrs", domainState.hostedZoneId, domainState.organisationId, hostedZoneAge.String())

		err := dns.UnregisterDomain(domainState.organisationId, domainState.domain)
		if err != nil {
			log.Errorf("cleanup hosted zone '%s' failed: %s", domainState.hostedZoneId, err.Error())
		}
	} else {
		// Since charging for hosted zones are not prorated for partial months if we exceeded the 12hrs we were already charged
		// It has no sense to delete the hosted zone until the next billing period starts (the first day of each subsequent month)
		// as the user may create a cluster in the organisation thus re-use the hosted zone that we were billed already for the month

		// If we are just before the next billing period and there are no clusters in the organisation we should cleanup the hosted zone
		// before entering the next billing period (the first day of the month)

		tillEndOfMonth := now.EndOfMonth().Sub(crtTime)

		maintenanceWindowMinute := viper.GetInt64(config.Route53MaintenanceWndMinute)

		if tillEndOfMonth <= time.Duration(maintenanceWindowMinute)*time.Minute {
			// if we are maintenanceWindowMinute minutes before the next billing period clean up the hosted zone

			// if the window is not long enough there will be few hosted zones slipping over into next billing period)
			log.Infof("cleanup hosted zone '%s' as it not used by organisation '%d'", domainState.hostedZoneId, domainState.organisationId)

			err := dns.UnregisterDomain(domainState.organisationId, domainState.domain)
			if err != nil {
				log.Errorf("cleanup hosted zone '%s' failed: %s", domainState.hostedZoneId, err.Error())
			}
		}
	}

}

// ProcessUnfinishedTasks continues processing in-progress domain registrations/unregistrations
func (dns *awsRoute53) ProcessUnfinishedTasks() {
	log := loggerWithFields(logrus.Fields{})

	// continue processing unfinished domain registrations
	pendingUnregister, err := dns.stateStore.findByStatus(REMOVING)
	if err != nil {
		log.Errorf("retrieving domains pending removal failed: %s", err.Error())
		return
	}

	for i := 0; i < len(pendingUnregister); i++ {
		domainState := pendingUnregister[i]
		log.Infof("continue un-registering domain '%s'", domainState.domain)

		go dns.UnregisterDomain(domainState.organisationId, domainState.domain)
	}

	// continue processing unfinished domain registrations
	pendingRegister, err := dns.stateStore.findByStatus(CREATING)
	if err != nil {
		log.Errorf("retrieving domains pending registration failed: %s", err.Error())
		return
	}

	for i := 0; i < len(pendingRegister); i++ {
		domainState := pendingRegister[i]
		log.Infof("continue registering domain '%s'", domainState.domain)

		go dns.RegisterDomain(domainState.organisationId, domainState.domain)
	}
}

// DeleteDnsRecordsOwnedBy deletes DNS records that belong to the specified owner
func (dns *awsRoute53) DeleteDnsRecordsOwnedBy(ownerId string, orgId uint) error {
	responseQueue := make(chan workerResponse)

	task := newWorkerTask(deleteDnsRecordsOwnedBy, orgId, nil, responseQueue)
	task.dnsRecordownerId = &ownerId

	dns.getWorker(orgId) <- task
	defer close(responseQueue)

	response := <-responseQueue
	return response.error
}

// GetOrgDomain returns the DNS domain name registered for the organization with given id
func (dns *awsRoute53) GetOrgDomain(orgId uint) (string, error) {
	responseQueue := make(chan workerResponse)

	dns.getWorker(orgId) <- newWorkerTask(getOrgDomain, orgId, nil, responseQueue)
	defer close(responseQueue)

	response := <-responseQueue
	if response.error != nil {
		return "", response.error
	}

	if response.result == nil {
		return "", nil
	}
	return fmt.Sprintf("%s", response.result), nil
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
	userAccessKeys, err := amazon.GetUserAccessKeys(dns.iamSvc, aws.String(iamUser))
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

// getRoute53Secret returns the secret from Vault that stores the IAM user
// aws access credentials that is used for accessing the Route53 Amazon service
func (dns *awsRoute53) getRoute53Secret(orgId uint) (*secret.SecretItemResponse, error) {
	awsAccessSecrets, err := secret.Store.List(orgId,
		&secretTypes.ListSecretsQuery{
			Type:   cluster.Amazon,
			Tags:   []string{secretTypes.TagBanzaiHidden},
			Values: true,
		})

	if err != nil {
		return nil, err
	}

	// route53 secret
	var route53Secrets []*secret.SecretItemResponse
	for _, awsAccessSecret := range awsAccessSecrets {
		if awsAccessSecret.Name == IAMUserAccessKeySecretName {
			route53Secrets = append(route53Secrets, awsAccessSecret)
		}
	}

	if len(route53Secrets) > 1 {
		return nil, fmt.Errorf("multiple secrets found with name '%s'", IAMUserAccessKeySecretName)
	}

	if len(route53Secrets) == 1 {
		return route53Secrets[0], nil
	}

	return nil, nil
}

// storeRoute53Secret stores the provided Amazon access key in Vault
func (dns *awsRoute53) storeRoute53Secret(updateSecret *secret.SecretItemResponse, awsAccessKeyId, awsSecretAccessKey string, ctx *context) error {
	req := &secret.CreateSecretRequest{
		Name: IAMUserAccessKeySecretName,
		Type: cluster.Amazon,
		Tags: []string{
			secretTypes.TagBanzaiHidden,
			secretTypes.TagBanzaiReadonly,
		},
		Values: map[string]string{
			secretTypes.AwsAccessKeyId:     awsAccessKeyId,
			secretTypes.AwsSecretAccessKey: awsSecretAccessKey,
			secretTypes.AwsRegion:          dns.region,
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

func (dns *awsRoute53) getOrgDomain(orgId uint) (string, error) {
	state := domainState{}
	found, err := dns.stateStore.findByOrgId(orgId, &state)

	if err != nil {
		return "", err
	}

	if found {
		return state.domain, nil
	}
	return "", nil
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

func getOrgById(orgId uint) (*auth.Organization, error) {
	org, err := auth.GetOrganizationById(orgId)
	if err != nil {
		return nil, err
	}

	return org, nil
}

// getWorker returns a worker that handles route53 related requests for the given organisation.
// It ensures that there is only one worker assigned for an organisation
func (dns *awsRoute53) getWorker(orgId uint) chan workerTask {
	dns.muxWorkers.Lock()
	defer dns.muxWorkers.Unlock()

	if dns.orgDomainWorkers == nil {
		dns.orgDomainWorkers = make(map[uint]chan workerTask)
	}

	worker, ok := dns.orgDomainWorkers[orgId]
	if !ok {
		worker = dns.startNewWorker()
		dns.orgDomainWorkers[orgId] = worker
	}

	return worker
}

// startNewWorker launches a new worker and returns the input channel through which
// it accepts tasks to executed. Tasks are executed in receiving order sequentially.
func (dns *awsRoute53) startNewWorker() chan workerTask {
	workQueue := make(chan workerTask)

	go func() {
		for task := range workQueue {
			switch task.operation {
			case isDomainRegistered:
				ok, err := dns.isDomainRegistered(task.organisationId, *task.domain)
				task.responseQueue <- workerResponse{error: err, result: ok}
			case registerDomain:
				err := dns.registerDomain(task.organisationId, *task.domain)
				task.responseQueue <- workerResponse{error: err}
			case unregisterDomain:
				err := dns.unregisterDomain(task.organisationId, *task.domain)
				task.responseQueue <- workerResponse{error: err}
			case deleteDnsRecordsOwnedBy:
				var err error
				var domain, hostedZoneId string
				domain, err = dns.getOrgDomain(task.organisationId)
				if err == nil {
					hostedZoneId, err = dns.hostedZoneExistsByDomain(domain)
					if err == nil {
						err = dns.deleteHostedZoneResourceRecordSetsOwnedBy(aws.String(hostedZoneId), aws.StringValue(task.dnsRecordownerId))
					}
				}
				task.responseQueue <- workerResponse{error: err}
			case getOrgDomain:
				domain, err := dns.getOrgDomain(task.organisationId)
				task.responseQueue <- workerResponse{error: err, result: domain}
			default:
				task.responseQueue <- workerResponse{error: fmt.Errorf("operation %q not supported", task.operation)}
			}
		}

	}()

	return workQueue
}

func newWorkerTask(operation operationType, orgId uint, domain *string, responseQueue chan<- workerResponse) workerTask {
	return workerTask{
		operation:      operation,
		organisationId: orgId,
		domain:         domain,
		responseQueue:  responseQueue,
	}
}

func createCommonEvent(orgId uint, domain string) *DomainEvent {
	return &DomainEvent{
		Domain:         domain,
		OrganisationId: orgId,
	}
}
