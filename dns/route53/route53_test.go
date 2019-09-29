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
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/pkg/errors"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/pkg/cluster"
	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
)

const (
	testOrgId                uint = 1
	testOrgName                   = "testorg"
	testBaseDomain                = "domain"
	testDomain                    = "test.domain"
	testDomainInUse               = "inuse.domain"
	testDomainMismatch            = "domain.mismatch"
	testPolicyArn                 = "testpolicyarn"
	testHostedZoneIdShort         = "testhostedzone1"
	testBaseHostedZoneId          = "/hostedzone/testhostedzonebase"
	testHostedZoneId              = "/hostedzone/testhostedzone1"
	testInUseHostedZoneId         = "/hostedzone/inuse.hostedzone.id"
	testMismatchHostedZoneId      = "/hostedzone/mismatch.hostedzone.id"
	testIamUser                   = "a05932df.r53.testorg" // getHashedControlPlaneHostName("example.org")
	testAccessKeyId               = "testaccesskeyid1"
	testAccessSecretKey           = "testsecretkey1"
	testPolicyDocument            = `{
		"Version": "2012-10-17",
		"Statement": [{
				"Effect": "Allow",
				"Action": "route53:ChangeResourceRecordSets",
				"Resource": "arn:aws:route53:::hostedzone/testhostedzone1"
			},
			{
				"Effect": "Allow",
				"Action": [
					"route53:ListHostedZones",
					"route53:ListHostedZonesByName",
					"route53:ListResourceRecordSets"
				],
				"Resource": "*"
			},
			{
				"Effect": "Allow",
				"Action": "route53:GetChange",
				"Resource": "arn:aws:route53:::change/*"
			}
		]}`
	testSomeErrMsg = "some error"
)

const (
	tcRerunHostedZoneCreation    = "Rerun previously failed hosted zone creation"
	tcRerunRoute53PolicyCreation = "Rerun previously failed route53 policy creation"
	tcRerunIAMUserCreation       = "Rerun previously failed IAM user creation"
	tcRerunAttachUserPolicy      = "Rerun previously failed attach policy to user"
	tcUnregisterDomain           = "Unregister domain"
	tcCleanup                    = "Cleanup"
)

// nolint: gochecknoglobals
var (
	testDomainStateCreated = &domainState{
		organisationId: testOrgId,
		domain:         testDomain,
		hostedZoneId:   testHostedZoneIdShort,
		policyArn:      testPolicyArn,
		iamUser:        testIamUser,
		awsAccessKeyId: testAccessKeyId,
		status:         CREATED,
		errMsg:         "",
	}

	testDomainStateCreatedYoung = &domainState{
		createdAt:      time.Now().Add(-1 * time.Hour),
		organisationId: testOrgId,
		domain:         testDomain,
		hostedZoneId:   testHostedZoneIdShort,
		policyArn:      testPolicyArn,
		iamUser:        testIamUser,
		awsAccessKeyId: testAccessKeyId,
		status:         CREATED,
		errMsg:         "",
	}

	testDomainStateCreatedAged = &domainState{
		createdAt:      time.Now().Add(-13 * time.Hour),
		organisationId: testOrgId,
		domain:         testDomain,
		hostedZoneId:   testHostedZoneIdShort,
		policyArn:      testPolicyArn,
		iamUser:        testIamUser,
		awsAccessKeyId: testAccessKeyId,
		status:         CREATED,
		errMsg:         "",
	}

	// case when hosted zone creation failed
	testDomainStateFailed1 = &domainState{
		organisationId: testOrgId,
		domain:         testDomain,
		status:         FAILED,
		errMsg:         testSomeErrMsg,
	}

	// case when route53 policy creation failed
	testDomainStateFailed2 = &domainState{
		organisationId: testOrgId,
		domain:         testDomain,
		hostedZoneId:   testHostedZoneIdShort,
		status:         FAILED,
		errMsg:         testSomeErrMsg,
	}

	// case when IAM user creation failed
	testDomainStateFailed3 = &domainState{
		organisationId: testOrgId,
		domain:         testDomain,
		hostedZoneId:   testHostedZoneIdShort,
		policyArn:      testPolicyArn,
		status:         FAILED,
		errMsg:         "failed to create IAM user: " + testSomeErrMsg,
	}

	// case when IAM user AWS access key creation failed
	// or attaching route53 policy to user failed
	testDomainStateFailed4 = &domainState{
		organisationId: testOrgId,
		domain:         testDomain,
		hostedZoneId:   testHostedZoneIdShort,
		policyArn:      testPolicyArn,
		iamUser:        testIamUser,
		status:         FAILED,
		errMsg:         testSomeErrMsg,
	}
)

type inMemoryStateStore struct {
	orgDomains map[string]*domainState
}

func (stateStore *inMemoryStateStore) create(state *domainState) error {
	key := stateKey(state.organisationId, state.domain)

	if _, ok := stateStore.orgDomains[key]; ok {
		return errors.New("unique domain constraint violation")
	}

	stateStore.orgDomains[key] = state
	return nil
}

func (stateStore *inMemoryStateStore) update(state *domainState) error {
	key := stateKey(state.organisationId, state.domain)

	if _, ok := stateStore.orgDomains[key]; !ok {
		return errors.New("State to be updated not found")
	}

	stateStore.orgDomains[key] = state

	return nil
}

func (stateStore *inMemoryStateStore) find(orgId uint, domain string, state *domainState) (bool, error) {
	key := stateKey(orgId, domain)

	s, ok := stateStore.orgDomains[key]

	if ok {
		state.organisationId = s.organisationId
		state.domain = s.domain
		state.status = s.status
		state.policyArn = s.policyArn
		state.hostedZoneId = s.hostedZoneId
		state.iamUser = s.iamUser
		state.awsAccessKeyId = s.awsAccessKeyId
		state.errMsg = s.errMsg
	}

	return ok, nil
}

func (stateStore *inMemoryStateStore) findByStatus(status string) ([]domainState, error) {
	var res []domainState
	for _, v := range stateStore.orgDomains {
		if v.status == status {
			res = append(res, *v)
		}
	}

	return res, nil
}

func (stateStore *inMemoryStateStore) findByOrgId(orgId uint, state *domainState) (bool, error) {

	for _, v := range stateStore.orgDomains {
		if v.organisationId == orgId {
			state.organisationId = v.organisationId
			state.domain = v.domain
			state.status = v.status
			state.policyArn = v.policyArn
			state.hostedZoneId = v.hostedZoneId
			state.iamUser = v.iamUser
			state.awsAccessKeyId = v.awsAccessKeyId
			state.errMsg = v.errMsg

			return true, nil
		}
	}

	return false, nil
}

func (stateStore *inMemoryStateStore) listUnused() ([]domainState, error) {
	key := stateKey(testOrgId, testDomain)
	return []domainState{*stateStore.orgDomains[key]}, nil
}

func (stateStore *inMemoryStateStore) delete(state *domainState) error {
	key := stateKey(state.organisationId, state.domain)

	if _, ok := stateStore.orgDomains[key]; !ok {
		return errors.New("State to be deleted not found")
	}

	delete(stateStore.orgDomains, key)

	return nil
}

func stateKey(orgId uint, domain string) string {
	return fmt.Sprintf("%d-%s", orgId, domain)
}

// Route53 API mocks

type mockRoute53Svc struct {
	route53iface.Route53API

	testCaseName string

	createHostedZoneCallCount         int
	deleteHostedZoneCallCount         int
	listResourceRecordSetsCallCount   int
	changeResourceRecordSetsCallCount int
}

func (mock *mockRoute53Svc) reset() {

	mock.testCaseName = ""

	mock.createHostedZoneCallCount = 0
	mock.deleteHostedZoneCallCount = 0
	mock.listResourceRecordSetsCallCount = 0
	mock.changeResourceRecordSetsCallCount = 0
}

func (mock *mockRoute53Svc) CreateHostedZone(createHostedZone *route53.CreateHostedZoneInput) (*route53.CreateHostedZoneOutput, error) {
	mock.createHostedZoneCallCount++

	return &route53.CreateHostedZoneOutput{
		HostedZone: &route53.HostedZone{
			Id:   aws.String(testHostedZoneId),
			Name: createHostedZone.Name,
		},
	}, nil
}

func (mock *mockRoute53Svc) GetHostedZone(getHostedZoneInput *route53.GetHostedZoneInput) (*route53.GetHostedZoneOutput, error) {
	return &route53.GetHostedZoneOutput{
		DelegationSet: &route53.DelegationSet{
			NameServers: []*string{aws.String("ns1")},
		},
		HostedZone: &route53.HostedZone{
			Id:   getHostedZoneInput.Id,
			Name: aws.String(testDomain),
		},
	}, nil
}

func (mock *mockRoute53Svc) ListHostedZonesByName(listHostedZonesByName *route53.ListHostedZonesByNameInput) (*route53.ListHostedZonesByNameOutput, error) {
	if aws.StringValue(listHostedZonesByName.DNSName) != testDomain &&
		aws.StringValue(listHostedZonesByName.DNSName) != testBaseDomain &&
		aws.StringValue(listHostedZonesByName.DNSName) != testDomainInUse &&
		aws.StringValue(listHostedZonesByName.DNSName) != testDomainMismatch {
		return nil, errors.New("route53.ListHostedZonesByName invoked with wrong domain name")
	}

	switch mock.testCaseName {
	case tcRerunHostedZoneCreation:
		return &route53.ListHostedZonesByNameOutput{}, nil
	case tcRerunRoute53PolicyCreation,
		tcRerunIAMUserCreation,
		tcRerunAttachUserPolicy,
		tcUnregisterDomain,
		tcCleanup:
		return &route53.ListHostedZonesByNameOutput{
			HostedZones: []*route53.HostedZone{
				{
					Name: aws.String(testDomain + "."),
					Id:   aws.String(testHostedZoneId),
				},
			},
		}, nil
	}

	if aws.StringValue(listHostedZonesByName.DNSName) == testDomainInUse {
		return &route53.ListHostedZonesByNameOutput{
			HostedZones: []*route53.HostedZone{
				{
					Name: aws.String(testDomainInUse),
					Id:   aws.String(testInUseHostedZoneId),
				},
			},
		}, nil
	}

	if aws.StringValue(listHostedZonesByName.DNSName) == testDomainMismatch {
		return &route53.ListHostedZonesByNameOutput{
			HostedZones: []*route53.HostedZone{
				{
					Name: aws.String(testDomainMismatch + "."),
					Id:   aws.String(testMismatchHostedZoneId),
				},
			},
		}, nil
	}

	return &route53.ListHostedZonesByNameOutput{}, nil
}

func (mock *mockRoute53Svc) DeleteHostedZone(deleteHostedZone *route53.DeleteHostedZoneInput) (*route53.DeleteHostedZoneOutput, error) {
	mock.deleteHostedZoneCallCount++

	if aws.StringValue(deleteHostedZone.Id) != testHostedZoneId {
		return nil, errors.New("route53.DeleteHostedZone invoked with wrong hosted zone id")
	}
	return &route53.DeleteHostedZoneOutput{}, nil
}

func (mock *mockRoute53Svc) ListResourceRecordSets(listResourceRecordSets *route53.ListResourceRecordSetsInput) (*route53.ListResourceRecordSetsOutput, error) {
	mock.listResourceRecordSetsCallCount++

	if aws.StringValue(listResourceRecordSets.HostedZoneId) != testHostedZoneId &&
		aws.StringValue(listResourceRecordSets.HostedZoneId) != testBaseHostedZoneId {
		return nil, errors.New("route53.ListResourceRecordSets invoked with wrong hosted zone id")
	}

	switch mock.testCaseName {
	case tcUnregisterDomain, tcCleanup:
		return &route53.ListResourceRecordSetsOutput{
			ResourceRecordSets: []*route53.ResourceRecordSet{
				{Type: aws.String("NS")},
				{Type: aws.String("SOA")},
				{Type: aws.String("A")},
				{Type: aws.String("AAA")},
			},
		}, nil
	}

	return &route53.ListResourceRecordSetsOutput{
		ResourceRecordSets: []*route53.ResourceRecordSet{
			{Type: aws.String("NS")},
			{Type: aws.String("SOA")},
		},
	}, nil
}

func (mock *mockRoute53Svc) ChangeResourceRecordSets(changeResourceRecordSets *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error) {
	mock.changeResourceRecordSetsCallCount++

	if aws.StringValue(changeResourceRecordSets.HostedZoneId) != testBaseHostedZoneId &&
		aws.StringValue(changeResourceRecordSets.HostedZoneId) != testHostedZoneId {

		return nil, errors.New("route53.ChangeResourceRecordSets invoked with wrong hosted zone id")
	}

	switch aws.StringValue(changeResourceRecordSets.HostedZoneId) {
	case testBaseHostedZoneId:
		if aws.StringValue(changeResourceRecordSets.ChangeBatch.Changes[0].Action) != route53.ChangeActionCreate {
			return nil, errors.New("route53.ChangeResourceRecordSets invoked with wrong action")
		}
	case testHostedZoneId:
		if aws.StringValue(changeResourceRecordSets.ChangeBatch.Changes[0].Action) != route53.ChangeActionDelete {
			return nil, errors.New("route53.ChangeResourceRecordSets invoked with wrong action")
		}
	}

	return &route53.ChangeResourceRecordSetsOutput{
		ChangeInfo: &route53.ChangeInfo{Id: aws.String("changeid")},
	}, nil
}

func (mock *mockRoute53Svc) WaitUntilResourceRecordSetsChanged(changeInput *route53.GetChangeInput) error {
	return nil
}

// mockRoute53SvcWithCreateHostedZoneFailing is a Route53 API mock with CreateHostedZone always failing
type mockRoute53SvcWithCreateHostedZoneFailing struct {
	mockRoute53Svc
}

func (mock *mockRoute53SvcWithCreateHostedZoneFailing) CreateHostedZone(createHostedZone *route53.CreateHostedZoneInput) (*route53.CreateHostedZoneOutput, error) {
	mock.createHostedZoneCallCount++

	return nil, errors.New(testSomeErrMsg)
}

// IAM API mocks

type mockIamSvc struct {
	iamiface.IAMAPI

	testCaseName string

	createPolicyCallCount           int
	getPolicyCallCount              int
	getUserCallCount                int
	deletePolicyCallCount           int
	createUserCallCount             int
	deleteUserCallCount             int
	listAttachedUserPolicyCallCount int
	attachUserPolicyCallCount       int
	detachUserPolicyCallCount       int
	createAccessKeyCallCount        int
	deleteAccessKeyCallCount        int
	listAccessKeyCallCount          int
}

func (mock *mockIamSvc) reset() {
	mock.testCaseName = ""

	mock.createPolicyCallCount = 0
	mock.getPolicyCallCount = 0
	mock.deletePolicyCallCount = 0
	mock.getUserCallCount = 0
	mock.createUserCallCount = 0
	mock.deleteUserCallCount = 0
	mock.listAttachedUserPolicyCallCount = 0
	mock.attachUserPolicyCallCount = 0
	mock.detachUserPolicyCallCount = 0
	mock.createAccessKeyCallCount = 0
	mock.deleteAccessKeyCallCount = 0
	mock.listAccessKeyCallCount = 0
}

func (mock *mockIamSvc) CreatePolicy(createPolicy *iam.CreatePolicyInput) (*iam.CreatePolicyOutput, error) {
	mock.createPolicyCallCount++

	if aws.StringValue(createPolicy.PolicyDocument) != testPolicyDocument {
		return nil, errors.New("iam.CreatePolicy invoked with wrong policy document")
	}
	return &iam.CreatePolicyOutput{
		Policy: &iam.Policy{
			Arn:          aws.String(testPolicyArn),
			PolicyName:   createPolicy.PolicyName,
			IsAttachable: aws.Bool(true),
			Description:  createPolicy.Description,
		},
	}, nil
}

func (mock *mockIamSvc) GetPolicy(getPolicy *iam.GetPolicyInput) (*iam.GetPolicyOutput, error) {
	mock.getPolicyCallCount++

	switch mock.testCaseName {
	case tcRerunIAMUserCreation,
		tcRerunAttachUserPolicy,
		tcUnregisterDomain,
		tcCleanup:
		return &iam.GetPolicyOutput{
			Policy: &iam.Policy{
				Arn:          getPolicy.PolicyArn,
				PolicyName:   aws.String(fmt.Sprintf("BanzaicloudRoute53-%s", testOrgName)),
				IsAttachable: aws.Bool(true),
			},
		}, nil
	}

	return &iam.GetPolicyOutput{}, nil
}

func (mock *mockIamSvc) DeletePolicy(deletePolicy *iam.DeletePolicyInput) (*iam.DeletePolicyOutput, error) {
	mock.deletePolicyCallCount++

	if aws.StringValue(deletePolicy.PolicyArn) != testPolicyArn {
		return nil, errors.New("iam.DeletePolicy invoked with wrong policy arn")
	}
	return &iam.DeletePolicyOutput{}, nil
}

func (mock *mockIamSvc) GetUser(user *iam.GetUserInput) (*iam.GetUserOutput, error) {
	mock.getUserCallCount++

	if aws.StringValue(user.UserName) != testIamUser {
		return nil, errors.New("iam.GetUser invoked with wrong user name")
	}

	switch mock.testCaseName {
	case tcRerunAttachUserPolicy,
		tcUnregisterDomain,
		tcCleanup:
		return &iam.GetUserOutput{
			User: &iam.User{
				UserName: user.UserName,
			},
		}, nil
	}

	return &iam.GetUserOutput{}, nil
}

func (mock *mockIamSvc) CreateUser(createUser *iam.CreateUserInput) (*iam.CreateUserOutput, error) {
	mock.createUserCallCount++

	return &iam.CreateUserOutput{
		User: &iam.User{
			UserName: createUser.UserName,
		},
	}, nil
}

func (mock *mockIamSvc) DeleteUser(deleteUser *iam.DeleteUserInput) (*iam.DeleteUserOutput, error) {
	mock.deleteUserCallCount++

	if aws.StringValue(deleteUser.UserName) != testIamUser {
		return nil, errors.New("iam.DeleteUser invoked with wrong IAM user name")
	}
	return &iam.DeleteUserOutput{}, nil
}

func (mock *mockIamSvc) ListAttachedUserPolicies(listUserAttachedPolicies *iam.ListAttachedUserPoliciesInput) (*iam.ListAttachedUserPoliciesOutput, error) {
	mock.listAttachedUserPolicyCallCount++

	switch mock.testCaseName {
	case tcUnregisterDomain, tcCleanup:
		return &iam.ListAttachedUserPoliciesOutput{
			AttachedPolicies: []*iam.AttachedPolicy{
				{PolicyArn: aws.String(testPolicyArn)},
			},
		}, nil
	}

	return &iam.ListAttachedUserPoliciesOutput{}, nil
}

func (mock *mockIamSvc) AttachUserPolicy(attachUserPolicy *iam.AttachUserPolicyInput) (*iam.AttachUserPolicyOutput, error) {
	mock.attachUserPolicyCallCount++

	return &iam.AttachUserPolicyOutput{}, nil
}

func (mock *mockIamSvc) DetachUserPolicy(detachPolicy *iam.DetachUserPolicyInput) (*iam.DetachUserPolicyOutput, error) {
	mock.detachUserPolicyCallCount++

	if aws.StringValue(detachPolicy.UserName) != testIamUser {
		return nil, errors.New("iam.DetachUserPolicy invoked with wrong IAM user name")
	}
	if aws.StringValue(detachPolicy.PolicyArn) != testPolicyArn {
		return nil, errors.New("iam.DetachUserPolicy invoked with wrong policy arn")
	}

	return &iam.DetachUserPolicyOutput{}, nil
}

func (mock *mockIamSvc) CreateAccessKey(createAccessKey *iam.CreateAccessKeyInput) (*iam.CreateAccessKeyOutput, error) {
	mock.createAccessKeyCallCount++

	return &iam.CreateAccessKeyOutput{
		AccessKey: &iam.AccessKey{
			UserName:        createAccessKey.UserName,
			AccessKeyId:     aws.String(testAccessKeyId),
			SecretAccessKey: aws.String(testAccessSecretKey),
		},
	}, nil
}

func (mock *mockIamSvc) DeleteAccessKey(deleteAccessKey *iam.DeleteAccessKeyInput) (*iam.DeleteAccessKeyOutput, error) {
	mock.deleteAccessKeyCallCount++

	if aws.StringValue(deleteAccessKey.UserName) != testIamUser {
		return nil, errors.New("iam.DeleteAccessKey invoked with wrong IAM user name")
	}
	if aws.StringValue(deleteAccessKey.AccessKeyId) != testAccessKeyId {
		return nil, errors.New("iam.DeleteAccessKey invoked with wrong access key id")
	}

	return &iam.DeleteAccessKeyOutput{}, nil
}

func (mock *mockIamSvc) ListAccessKeys(listAccessKeys *iam.ListAccessKeysInput) (*iam.ListAccessKeysOutput, error) {
	mock.listAccessKeyCallCount++

	switch mock.testCaseName {
	case tcUnregisterDomain, tcCleanup:
		return &iam.ListAccessKeysOutput{
			AccessKeyMetadata: []*iam.AccessKeyMetadata{
				{UserName: listAccessKeys.UserName, AccessKeyId: aws.String(testAccessKeyId)},
			},
		}, nil
	}
	return &iam.ListAccessKeysOutput{}, nil
}

// mockIamSvcWithCreatePolicyFailing is a IAM service API mock with CreatePolicy always failing
type mockIamSvcWithCreatePolicyFailing struct {
	mockIamSvc
}

func (mock *mockIamSvcWithCreatePolicyFailing) CreatePolicy(*iam.CreatePolicyInput) (*iam.CreatePolicyOutput, error) {
	mock.createPolicyCallCount++

	return nil, errors.New(testSomeErrMsg)
}

// mockIamSvcWithCreateIAMUserFailing  is a IAM service API mock with CreateUser always failing
type mockIamSvcWithCreateIAMUserFailing struct {
	mockIamSvc
}

func (*mockIamSvcWithCreateIAMUserFailing) CreateUser(*iam.CreateUserInput) (*iam.CreateUserOutput, error) {
	return nil, errors.New(testSomeErrMsg)
}

// mockIamSvcWithAttachUserPolicyFailing  is a IAM service API mock with AttachUserPolicy always failing
type mockIamSvcWithAttachUserPolicyFailing struct {
	mockIamSvc
}

func (*mockIamSvcWithAttachUserPolicyFailing) AttachUserPolicy(*iam.AttachUserPolicyInput) (*iam.AttachUserPolicyOutput, error) {
	return nil, errors.New(testSomeErrMsg)
}

// mockIamSvcWithCreateAccessKeyFailing  is a IAM service API mock with AttachUserPolicy always failing
type mockIamSvcWithCreateAccessKeyFailing struct {
	mockIamSvc
}

func (*mockIamSvcWithCreateAccessKeyFailing) CreateAccessKey(*iam.CreateAccessKeyInput) (*iam.CreateAccessKeyOutput, error) {
	return nil, errors.New(testSomeErrMsg)
}

func TestAwsRoute53_RegisterDomain(t *testing.T) {
	stateStore := &inMemoryStateStore{
		orgDomains: make(map[string]*domainState),
	}

	awsRoute53 := &awsRoute53{route53Svc: &mockRoute53Svc{}, iamSvc: &mockIamSvc{}, stateStore: stateStore, getOrganization: getTestOrgById, baseHostedZoneId: testBaseHostedZoneId}

	err := awsRoute53.RegisterDomain(testOrgId, testDomain)

	if err != nil {
		t.Fatalf("Register domain should succeed: %s", err.Error())
	}

	state := &domainState{}
	ok, _ := stateStore.find(testOrgId, testDomain, state)

	if !ok {
		t.Fatalf("Statestore should contain an entry for the registered domain")
	}

	expected := testDomainStateCreated

	if reflect.DeepEqual(state, expected) == false {
		t.Fatalf("Expected %v, got %v", expected, state)
	}

	secrets, err := secret.Store.List(testOrgId, &secretTypes.ListSecretsQuery{
		Type:   cluster.Amazon,
		Tags:   []string{secretTypes.TagBanzaiHidden},
		Values: true,
	})

	if err != nil {
		t.Fatalf("Failed to list '%s' in Vault: %s", IAMUserAccessKeySecretName, err.Error())
	}

	if len(secrets) != 1 {
		t.Fatalf("There should be one secret with name '%s' in Vault", IAMUserAccessKeySecretName)
	}

	route53SecretCount := 0

	secretItem := secrets[0]
	defer func() { _ = secret.Store.Delete(testOrgId, secretItem.ID) }()

	if secretItem.Name == IAMUserAccessKeySecretName {
		if secretItem.Values[secretTypes.AwsAccessKeyId] == testAccessKeyId &&
			secretItem.Values[secretTypes.AwsSecretAccessKey] == testAccessSecretKey {
			route53SecretCount++
		}
	}

	if route53SecretCount != 1 {
		t.Errorf("There should be one route53 secret in Vault but got %d", route53SecretCount)
	}

}

func TestAwsRoute53_RegisterDomain_Fail(t *testing.T) {
	route53Svc := &mockRoute53Svc{}
	route53SvcWithCreateHostedZoneFailing := &mockRoute53SvcWithCreateHostedZoneFailing{}
	iamSvcWithCreatePolicyFailing := &mockIamSvcWithCreatePolicyFailing{}
	iamSvcWithCreateIAMUserFailing := &mockIamSvcWithCreateIAMUserFailing{}
	iamSvcWithAttachUserPolicyFailing := &mockIamSvcWithAttachUserPolicyFailing{}
	iamSvcWithCreateAccessKeyFailing := &mockIamSvcWithCreateAccessKeyFailing{}

	tests := []struct {
		name            string
		route53Svc      route53iface.Route53API
		iamSvc          iamiface.IAMAPI
		expectedErrMsg  string
		expectedState   *domainState
		verifyRollbacks func(*testing.T)
	}{
		{
			name:            "Register domain should fail due to hosted zone creation failing",
			route53Svc:      route53SvcWithCreateHostedZoneFailing,
			iamSvc:          &mockIamSvc{},
			expectedErrMsg:  testSomeErrMsg,
			expectedState:   testDomainStateFailed1,
			verifyRollbacks: nil,
		},
		{
			name:           "Register domain should fail due to policy creation failing",
			route53Svc:     route53Svc,
			iamSvc:         iamSvcWithCreatePolicyFailing,
			expectedErrMsg: testSomeErrMsg,
			expectedState:  testDomainStateFailed2,
			verifyRollbacks: func(t *testing.T) {
				if route53Svc.deleteHostedZoneCallCount != 1 {
					t.Errorf("Created hosted zone should be rolled back")
				}

				route53Svc.reset()
			},
		},
		{
			name:           "Register domain should fail due to IAM user creation failing",
			route53Svc:     route53Svc,
			iamSvc:         iamSvcWithCreateIAMUserFailing,
			expectedErrMsg: "failed to create IAM user: " + testSomeErrMsg,
			expectedState:  testDomainStateFailed3,
			verifyRollbacks: func(t *testing.T) {
				if route53Svc.deleteHostedZoneCallCount != 1 {
					t.Errorf("Created hosted zone should be rolled back")
				}

				if iamSvcWithCreateIAMUserFailing.deletePolicyCallCount != 1 {
					t.Errorf("Created access policy should be rolled back")
				}

				route53Svc.reset()
				iamSvcWithCreateIAMUserFailing.reset()

			},
		},
		{
			name:           "Register domain should fail due to attaching policy to user failing",
			route53Svc:     route53Svc,
			iamSvc:         iamSvcWithAttachUserPolicyFailing,
			expectedErrMsg: testSomeErrMsg,
			expectedState:  testDomainStateFailed4,
			verifyRollbacks: func(t *testing.T) {
				if route53Svc.deleteHostedZoneCallCount != 1 {
					t.Errorf("Created hosted zone should be rolled back")
				}

				if iamSvcWithAttachUserPolicyFailing.deletePolicyCallCount != 1 {
					t.Errorf("Created access policy should be rolled back")
				}

				if iamSvcWithAttachUserPolicyFailing.deleteUserCallCount != 1 {
					t.Errorf("Created IAM user should be rolled back")
				}

				route53Svc.reset()
				iamSvcWithAttachUserPolicyFailing.reset()

			},
		},
		{
			name:           "Register domain should fail due to Amazon access key creation failing",
			route53Svc:     route53Svc,
			iamSvc:         iamSvcWithCreateAccessKeyFailing,
			expectedErrMsg: testSomeErrMsg,
			expectedState:  testDomainStateFailed4,
			verifyRollbacks: func(t *testing.T) {
				if route53Svc.deleteHostedZoneCallCount != 1 {
					t.Errorf("Created hosted zone should be rolled back")
				}

				if iamSvcWithCreateAccessKeyFailing.deletePolicyCallCount != 1 {
					t.Errorf("Created access policy should be rolled back")
				}

				if iamSvcWithCreateAccessKeyFailing.deleteUserCallCount != 1 {
					t.Errorf("Created IAM user should be rolled back")
				}

				if iamSvcWithCreateAccessKeyFailing.detachUserPolicyCallCount != 1 {
					t.Errorf("Policy should be detached from IAM user")
				}

				route53Svc.reset()
				iamSvcWithCreateAccessKeyFailing.reset()
				cleanupVaultTestSecrets()

			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stateStore := &inMemoryStateStore{
				orgDomains: make(map[string]*domainState),
			}

			awsRoute53 := &awsRoute53{route53Svc: tc.route53Svc, iamSvc: tc.iamSvc, stateStore: stateStore, getOrganization: getTestOrgById, baseHostedZoneId: testBaseHostedZoneId}

			err := awsRoute53.RegisterDomain(testOrgId, testDomain)
			if err.Error() != tc.expectedErrMsg {
				t.Errorf("Register domain should fail with: %s", testSomeErrMsg)
			}

			actualState := &domainState{}
			ok, _ := stateStore.find(testOrgId, testDomain, actualState)
			if !ok || reflect.DeepEqual(actualState, tc.expectedState) == false {
				t.Errorf("State store should contain: %v", tc.expectedState)
			}

			// verify rollbacks
			if tc.verifyRollbacks != nil {
				tc.verifyRollbacks(t)
			}

		})
	}

}

func TestAwsRoute53_UnregisterDomain(t *testing.T) {

	key := stateKey(testOrgId, testDomain)

	stateStore := &inMemoryStateStore{
		orgDomains: map[string]*domainState{key: testDomainStateCreated},
	}

	route53Svc := &mockRoute53Svc{testCaseName: tcUnregisterDomain}
	iamSvc := &mockIamSvc{testCaseName: tcUnregisterDomain}

	awsRoute53 := &awsRoute53{route53Svc: route53Svc, iamSvc: iamSvc, stateStore: stateStore, getOrganization: getTestOrgById, baseHostedZoneId: testBaseHostedZoneId}

	err := awsRoute53.UnregisterDomain(testOrgId, testDomain)
	if err != nil {
		t.Errorf("Unregister domain should succeed")
	}

	ok, _ := stateStore.find(testOrgId, testDomain, &domainState{})
	if ok {
		t.Errorf("Statestore should not contain an entry for the registered domain '%s'", testDomain)
	}

	if route53Svc.deleteHostedZoneCallCount != 1 {
		t.Error("Hosted zone should be deleted")
	}

	if iamSvc.detachUserPolicyCallCount != 1 {
		t.Error("Policy should be detached from IAM user")
	}

	if iamSvc.deletePolicyCallCount != 1 {
		t.Error("Access policy should be deleted")
	}

	if iamSvc.deleteAccessKeyCallCount != 1 {
		t.Error("User Amazon access key should be deleted")
	}

	if iamSvc.deleteUserCallCount != 1 {
		t.Error("IAM user should be deleted")
	}

	if route53Svc.changeResourceRecordSetsCallCount != 1 {
		t.Error("Record resource sets of the hosted zone should be deleted")
	}

	//reset mock call count
	route53Svc.reset()
	iamSvc.reset()

	cleanupVaultTestSecrets()

}

func TestAwsRoute53_Cleanup(t *testing.T) {
	key := stateKey(testOrgId, testDomain)

	route53Svc := &mockRoute53Svc{testCaseName: tcCleanup}
	iamSvc := &mockIamSvc{testCaseName: tcCleanup}

	tests := []struct {
		name                              string
		state                             *domainState
		found                             bool
		deleteHostedZoneCallCount         int
		changeResourceRecordSetsCallCount int
		detachUserPolicyCallCount         int
		deletePolicyCallCount             int
		deleteAccessKeyCallCount          int
		deleteUserCallCount               int
		deleteHostedZoneCallMsg           string
		detachUserPolicyCallMsg           string
		deletePolicyCallMsg               string
		deleteAccessKeyCallMsg            string
		deleteUserCallMsg                 string
		changeResourceRecordSetsCallMsg   string
	}{
		{
			name:                              "Hosted zone younger than 12 hours should be cleaned up",
			state:                             testDomainStateCreatedYoung,
			found:                             false,
			deleteHostedZoneCallCount:         1,
			changeResourceRecordSetsCallCount: 1,
			detachUserPolicyCallCount:         1,
			deletePolicyCallCount:             1,
			deleteAccessKeyCallCount:          1,
			deleteUserCallCount:               1,
			deleteHostedZoneCallMsg:           "Hosted zone should be deleted",
			detachUserPolicyCallMsg:           "Policy should be detached from IAM user",
			deletePolicyCallMsg:               "Access policy should be deleted",
			deleteAccessKeyCallMsg:            "User Amazon access key should be deleted",
			deleteUserCallMsg:                 "IAM user should be deleted",
			changeResourceRecordSetsCallMsg:   "Hosted Zone resource record sets should be deleted",
		},
		{
			name:                              "Hosted zone older than 12 hours should not be cleaned up",
			state:                             testDomainStateCreatedAged,
			found:                             true,
			deleteHostedZoneCallCount:         0,
			changeResourceRecordSetsCallCount: 0,
			detachUserPolicyCallCount:         0,
			deletePolicyCallCount:             0,
			deleteAccessKeyCallCount:          0,
			deleteUserCallCount:               0,
			deleteHostedZoneCallMsg:           "Hosted zone should not be deleted",
			detachUserPolicyCallMsg:           "Policy should not be detached from IAM user",
			deletePolicyCallMsg:               "Access policy should not be deleted",
			deleteAccessKeyCallMsg:            "User Amazon access key should not be deleted",
			deleteUserCallMsg:                 "IAM user should not be deleted",
			changeResourceRecordSetsCallMsg:   "Hosted Zone resource record sets should not be deleted",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stateStore := &inMemoryStateStore{
				orgDomains: map[string]*domainState{key: tc.state},
			}

			awsRoute53 := &awsRoute53{route53Svc: route53Svc, iamSvc: iamSvc, stateStore: stateStore, getOrganization: getTestOrgById, baseHostedZoneId: testBaseHostedZoneId}
			awsRoute53.Cleanup()

			found, _ := stateStore.find(testOrgId, testDomain, &domainState{})
			if found != tc.found {
				if tc.found {
					t.Errorf("Statestore should contain an entry for the registered domain '%s'", testDomain)
				} else {
					t.Errorf("Statestore should not contain an entry for the registered domain '%s'", testDomain)
				}
			}

			if route53Svc.deleteHostedZoneCallCount != tc.deleteHostedZoneCallCount {
				t.Error(tc.deleteHostedZoneCallMsg)
			}
			if route53Svc.changeResourceRecordSetsCallCount != tc.changeResourceRecordSetsCallCount {
				t.Errorf(tc.deleteUserCallMsg)
			}

			if iamSvc.detachUserPolicyCallCount != tc.detachUserPolicyCallCount {
				t.Error(tc.detachUserPolicyCallMsg)
			}

			if iamSvc.deletePolicyCallCount != tc.deletePolicyCallCount {
				t.Errorf(tc.deletePolicyCallMsg)
			}

			if iamSvc.deleteAccessKeyCallCount != tc.deleteAccessKeyCallCount {
				t.Errorf(tc.deleteAccessKeyCallMsg)
			}

			if iamSvc.deleteUserCallCount != tc.deleteUserCallCount {
				t.Errorf(tc.deleteUserCallMsg)
			}

			//reset mock call count
			route53Svc.reset()
			iamSvc.reset()

			cleanupVaultTestSecrets()
		})
	}

}

func TestAwsRoute53_RegisterDomainRerun(t *testing.T) {
	key := stateKey(testOrgId, testDomain)

	tests := []struct {
		name  string
		state *domainState
	}{
		{
			name:  tcRerunHostedZoneCreation,
			state: testDomainStateFailed1,
		},
		{
			name:  tcRerunRoute53PolicyCreation,
			state: testDomainStateFailed2,
		},
		{
			name:  tcRerunIAMUserCreation,
			state: testDomainStateFailed3,
		},
		{
			name:  tcRerunAttachUserPolicy,
			state: testDomainStateFailed4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			route53Svc := &mockRoute53Svc{testCaseName: tc.name}
			iamSvc := &mockIamSvc{testCaseName: tc.name}
			stateStore := &inMemoryStateStore{
				orgDomains: map[string]*domainState{key: tc.state},
			}

			awsRoute53 := &awsRoute53{route53Svc: route53Svc, iamSvc: iamSvc, stateStore: stateStore, getOrganization: getTestOrgById, baseHostedZoneId: testBaseHostedZoneId}

			err := awsRoute53.RegisterDomain(testOrgId, testDomain)
			if err != nil {
				t.Errorf("Failed with unexpected error: %v", err)
			}

			actualState := &domainState{}
			ok, _ := stateStore.find(testOrgId, testDomain, actualState)
			if !ok {
				t.Errorf("Statestore should not contain an entry for the registered domain '%s'", testDomain)
			}

			if reflect.DeepEqual(testDomainStateCreated, actualState) == false {
				t.Errorf("Expected state for domain '%s': %v, got: %v", testDomain, testDomainStateCreated, actualState)
			}

			cleanupVaultTestSecrets()
		})
	}
}

func Test_nameServerMatch(t *testing.T) {
	tests := []struct {
		name     string
		ds       *route53.DelegationSet
		rrs      *route53.ResourceRecordSet
		expected bool
		msg      string
	}{
		{
			name: "Test equality",
			ds: &route53.DelegationSet{
				NameServers: []*string{aws.String("server1"), aws.String("server2")},
			},
			rrs: &route53.ResourceRecordSet{
				Type: aws.String(route53.RRTypeNs),
				ResourceRecords: []*route53.ResourceRecord{
					{Value: aws.String("server2")},
					{Value: aws.String("server1")},
				},
			},
			expected: true,
			msg:      "Resource record set should match name servers from delegation set",
		},
		{
			name: "Test inequality due to different name server list",
			ds: &route53.DelegationSet{
				NameServers: []*string{aws.String("server1"), aws.String("server2")},
			},
			rrs: &route53.ResourceRecordSet{
				Type: aws.String(route53.RRTypeNs),
				ResourceRecords: []*route53.ResourceRecord{
					{Value: aws.String("server2")},
				},
			},
			expected: false,
			msg:      "Resource record set should not match name servers from delegation set",
		},
		{
			name: "Test inequality due to resource record set type",
			ds: &route53.DelegationSet{
				NameServers: []*string{aws.String("server1"), aws.String("server2")},
			},
			rrs: &route53.ResourceRecordSet{
				Type: aws.String(route53.RRTypeSoa),
				ResourceRecords: []*route53.ResourceRecord{
					{Value: aws.String("server2")},
					{Value: aws.String("server1")},
				},
			},
			expected: false,
			msg:      "Resource record set should not match name servers from delegation set",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if nameServerMatch(tc.ds, tc.rrs) != tc.expected {
				t.Errorf(tc.msg)
			}
		})
	}

}

func cleanupVaultTestSecrets() {
	secrets, _ := secret.Store.List(testOrgId, &secretTypes.ListSecretsQuery{
		Type:   cluster.Amazon,
		Tags:   []string{secretTypes.TagBanzaiHidden},
		Values: true,
	})

	for _, secretItem := range secrets {
		if secretItem.Name == IAMUserAccessKeySecretName {
			if secretItem.Values[secretTypes.AwsAccessKeyId] == testAccessKeyId &&
				secretItem.Values[secretTypes.AwsSecretAccessKey] == testAccessSecretKey {

				_ = secret.Store.Delete(testOrgId, secretItem.ID)
			}
		}
	}

}

func getTestOrgById(orgId uint) (*auth.Organization, error) {
	return &auth.Organization{ID: testOrgId, Name: testOrgName}, nil
}
