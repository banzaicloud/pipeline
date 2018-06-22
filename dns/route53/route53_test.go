package route53

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/banzaicloud/pipeline/pkg/cluster"
	secretTypes "github.com/banzaicloud/pipeline/pkg/secret"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/pkg/errors"
	"reflect"
	"testing"
)

const (
	testOrgId             = 1
	testDomain            = "test.domain"
	testDomainInUse       = "inuse.domain"
	testPolicyArn         = "testpolicyarn"
	testHostedZoneIdShort = "testhostedzone1"
	testHostedZoneId      = "/hostedzone/testhostedzone1"
	testIamUser           = "banzaicloud.route53.testhostedzone1"
	testAccessKeyId       = "testaccesskeyid1"
	testAccessSecretKey   = "testsecretkey1"
	testPolicyDocument    = `{
						"Version": "2012-10-17",
    				"Statement": [
							{
            		"Effect": "Allow",
            		"Action": "route53:ChangeResourceRecordSets",
                "Resource": "arn:aws:route53:::hostedzone/testhostedzone1"
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
					}`
	testSomeErrMsg = "some error"
)

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

	testDomainStateFailed1 = &domainState{
		organisationId: testOrgId,
		domain:         testDomain,
		status:         FAILED,
		errMsg:         testSomeErrMsg,
	}

	testDomainStateFailed2 = &domainState{
		organisationId: testOrgId,
		domain:         testDomain,
		hostedZoneId:   testHostedZoneIdShort,
		status:         FAILED,
		errMsg:         testSomeErrMsg,
	}

	testDomainStateFailed3 = &domainState{
		organisationId: testOrgId,
		domain:         testDomain,
		hostedZoneId:   testHostedZoneIdShort,
		policyArn:      testPolicyArn,
		status:         FAILED,
		errMsg:         testSomeErrMsg,
	}

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

	createHostedZoneCallCount int
	deleteHostedZoneCallCount int
}

func (mock *mockRoute53Svc) reset() {
	mock.createHostedZoneCallCount = 0
	mock.deleteHostedZoneCallCount = 0
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

func (mock *mockRoute53Svc) ListHostedZonesByName(listHostedZonesByName *route53.ListHostedZonesByNameInput) (*route53.ListHostedZonesByNameOutput, error) {
	if aws.StringValue(listHostedZonesByName.DNSName) != testDomain && aws.StringValue(listHostedZonesByName.DNSName) != testDomainInUse {
		return nil, errors.New("iam.ListHostedZonesByName invoked with wrong domain name")
	}

	if aws.StringValue(listHostedZonesByName.DNSName) == testDomainInUse {
		return &route53.ListHostedZonesByNameOutput{
			HostedZones: []*route53.HostedZone{
				{
					Name: aws.String(testDomainInUse),
				},
			},
		}, nil
	}

	return &route53.ListHostedZonesByNameOutput{}, nil
}

func (mock *mockRoute53Svc) DeleteHostedZone(deleteHostedZone *route53.DeleteHostedZoneInput) (*route53.DeleteHostedZoneOutput, error) {
	mock.deleteHostedZoneCallCount++

	if aws.StringValue(deleteHostedZone.Id) != testHostedZoneIdShort {
		return nil, errors.New("iam.DeleteHostedZone invoked with wrong hosted zone id")
	}
	return &route53.DeleteHostedZoneOutput{}, nil
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

	createPolicyCallCount     int
	deletePolicyCallCount     int
	createUserCallCount       int
	deleteUserCallCount       int
	attachUserPolicyCallCount int
	detachUserPolicyCallCount int
	createAccessKeyCallCount  int
	deleteAccessKeyCallCount  int
}

func (mock *mockIamSvc) reset() {
	mock.createPolicyCallCount = 0
	mock.deletePolicyCallCount = 0
	mock.createUserCallCount = 0
	mock.deleteUserCallCount = 0
	mock.attachUserPolicyCallCount = 0
	mock.detachUserPolicyCallCount = 0
	mock.createAccessKeyCallCount = 0
	mock.deleteAccessKeyCallCount = 0
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

func (mock *mockIamSvc) DeletePolicy(deletePolicy *iam.DeletePolicyInput) (*iam.DeletePolicyOutput, error) {
	mock.deletePolicyCallCount++

	if aws.StringValue(deletePolicy.PolicyArn) != testPolicyArn {
		return nil, errors.New("iam.DeletePolicy invoked with wrong policy arn")
	}
	return &iam.DeletePolicyOutput{}, nil
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

	awsRoute53 := &awsRoute53{route53Svc: &mockRoute53Svc{}, iamSvc: &mockIamSvc{}, stateStore: stateStore}

	err := awsRoute53.RegisterDomain(testOrgId, testDomain)

	if err != nil {
		t.Errorf("Register domain should succeed")
	}

	state := &domainState{}
	ok, _ := stateStore.find(testOrgId, testDomain, state)

	if !ok {
		t.Errorf("Statestore should contain an entry for the registered domain")
	}

	expected := testDomainStateCreated

	if reflect.DeepEqual(state, expected) == false {
		t.Errorf("Expected %v, got %v", expected, state)
	}

	secrets, _ := secret.Store.List(testOrgId, &secretTypes.ListSecretsQuery{
		Type:   cluster.Amazon,
		Tag:    secretTypes.TagBanzaiHidden,
		Values: true,
	})

	if len(secrets) != 1 {
		t.Errorf("There should be one secret with name '%s' in Vault", iamUserAccessKeySecretName)
	}

	route53SecretCount := 0

	for _, secretItem := range secrets {
		if secretItem.Name == iamUserAccessKeySecretName {
			if secretItem.Values[secretTypes.AwsAccessKeyId] == testAccessKeyId &&
				secretItem.Values[secretTypes.AwsSecretAccessKey] == testAccessSecretKey {
				route53SecretCount++

				defer func() { secret.Store.Delete(testOrgId, secretItem.ID) }()
			}
		}
	}

	if route53SecretCount != 1 {
		t.Errorf("There should be one route53 secret in Vault but got %d", route53SecretCount)
	}

}

func TestAwsRoute53_RegisterDomain_AlreadyRegistered(t *testing.T) {
	key := stateKey(testOrgId, testDomain)

	stateStore := &inMemoryStateStore{
		orgDomains: map[string]*domainState{key: testDomainStateCreated},
	}

	awsRoute53 := &awsRoute53{route53Svc: &mockRoute53Svc{}, iamSvc: &mockIamSvc{}, stateStore: stateStore}
	expectedErrMsg := fmt.Sprintf("domain '%s' already registered", testDomain)

	err := awsRoute53.RegisterDomain(testOrgId, testDomain)
	if err.Error() != expectedErrMsg {
		t.Errorf("Registering duplicate domain should return an error with error message: %s", expectedErrMsg)
	}
}

func TestAwsRoute53_RegisterDomain_AlreadyInUse(t *testing.T) {
	stateStore := &inMemoryStateStore{
		orgDomains: make(map[string]*domainState),
	}

	awsRoute53 := &awsRoute53{route53Svc: &mockRoute53Svc{}, iamSvc: &mockIamSvc{}, stateStore: stateStore}
	expectedErrMsg := fmt.Sprintf("domain '%s' is already in use", testDomainInUse)

	err := awsRoute53.RegisterDomain(testOrgId, testDomainInUse)
	if err.Error() != expectedErrMsg {
		t.Errorf("Registering duplicate domain should return an error with error message: %s", expectedErrMsg)
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
			expectedErrMsg: testSomeErrMsg,
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

			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stateStore := &inMemoryStateStore{
				orgDomains: make(map[string]*domainState),
			}

			awsRoute53 := &awsRoute53{route53Svc: tc.route53Svc, iamSvc: tc.iamSvc, stateStore: stateStore}

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

	route53Svc := &mockRoute53Svc{}
	iamSvc := &mockIamSvc{}

	awsRoute53 := &awsRoute53{route53Svc: route53Svc, iamSvc: iamSvc, stateStore: stateStore}

	err := awsRoute53.UnregisterDomain(testOrgId, testDomain)
	if err != nil {
		t.Errorf("Unregister domain should succeed")
	}

	ok, _ := stateStore.find(testOrgId, testDomain, &domainState{})
	if ok {
		t.Errorf("Statestore should not contain an entry for the registered domain '%s'", testDomain)
	}

	if route53Svc.deleteHostedZoneCallCount != 1 {
		t.Errorf("Hosted zone should be deleted")
	}

	if iamSvc.detachUserPolicyCallCount != 1 {
		t.Errorf("Policy should be detached from IAM user")
	}

	if iamSvc.deletePolicyCallCount != 1 {
		t.Errorf("Access policy should be deleted")
	}

	if iamSvc.deleteAccessKeyCallCount != 1 {
		t.Errorf("User Amazon access key should be deleted")
	}

	if iamSvc.deleteUserCallCount != 1 {
		t.Errorf("IAM user should be deleted")
	}

}
