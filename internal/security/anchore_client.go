// Copyright © 2019 Banzai Cloud
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

package anchore

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"regexp"

	"emperror.dev/errors"
	"github.com/antihax/optional"
	"gopkg.in/resty.v1"

	"github.com/banzaicloud/pipeline/.gen/anchore"
	"github.com/banzaicloud/pipeline/internal/common"
)

var ecrRegexp = regexp.MustCompile("[0-9]+\\.dkr\\.ecr\\..*\\.amazonaws\\.com") // nolint

type UserManagementClient interface {
	CreateAccount(ctx context.Context, accountName string, email string) error
	DeleteAccount(ctx context.Context, accountName string) error
	GetAccount(ctx context.Context, accountName string) (string, error)
	CreateUser(ctx context.Context, accountName string, userName string, password string) error
	DeleteUser(ctx context.Context, accountName string, userName string) error
	GetUser(ctx context.Context, userName string) (interface{}, error)
	GetUserCredentials(ctx context.Context, userName string) (string, error)
}

type PolicyClient interface {
	ActivatePolicy(ctx context.Context, policyID string) error
	CreatePolicy(ctx context.Context, policyRaw map[string]interface{}) (string, error)
}

type RegistryClient interface {
	AddRegistry(ctx context.Context, registry Registry) error
	GetRegistry(ctx context.Context, registryName string) ([]anchore.RegistryConfiguration, error)
	UpdateRegistry(ctx context.Context, registry Registry) error
	DeleteRegistry(ctx context.Context, registry Registry) error
}

type Registry struct {
	Username string
	Password string
	Type     string
	Registry string
	Verify   bool
}

func IsEcrRegistry(registry string) bool {
	return ecrRegexp.MatchString(registry)
}

// AnchoreClient "facade" for supported Anchore operations, decouples anchore specifics from the application
type AnchoreClient interface {
	UserManagementClient
	PolicyClient
	RegistryClient
}

type anchoreClient struct {
	userName string
	password string
	endpoint string
	logger   common.Logger
	insecure bool
}

func NewAnchoreClient(userName string, password string, endpoint string, insecure bool, logger common.Logger) AnchoreClient {
	return anchoreClient{
		userName: userName,
		password: password,
		endpoint: endpoint,
		logger:   logger.WithFields(map[string]interface{}{"anchore-client": ""}),
		insecure: insecure,
	}
}

func (a anchoreClient) CreateAccount(ctx context.Context, accountName string, email string) error {
	fnCtx := map[string]interface{}{"accountName": accountName, "email": email}
	a.logger.Info("creating anchore account", fnCtx)

	_, resp, err := a.getRestClient().UserManagementApi.CreateAccount(a.authorizedContext(ctx),
		anchore.AccountCreationRequest{
			Name:  accountName,
			Email: email,
		})

	if err != nil || (resp.StatusCode != http.StatusOK) {
		a.logger.Debug("failed to create anchore account", fnCtx)

		return errors.WrapIfWithDetails(err, "failed to create anchore account", fnCtx)
	}

	a.logger.Info("anchore account created", fnCtx)
	return nil
}

func (a anchoreClient) CreateUser(ctx context.Context, accountName string, userName string, password string) error {
	fnCtx := map[string]interface{}{"accountName": accountName, "userName": userName}
	a.logger.Info("creating anchore user", fnCtx)

	_, resp, err := a.getRestClient().UserManagementApi.CreateUser(a.authorizedContext(ctx),
		accountName, anchore.UserCreationRequest{
			Username: userName,
			Password: password,
		})

	if err != nil || (resp.StatusCode != http.StatusOK) {
		a.logger.Debug("failed to create anchore user", fnCtx)

		return errors.WrapIfWithDetails(err, "failed to create anchore account", fnCtx)
	}

	a.logger.Info("anchore user created", fnCtx)
	return nil
}

func (a anchoreClient) GetUser(ctx context.Context, userName string) (interface{}, error) {
	fnCtx := map[string]interface{}{"userName": userName}
	a.logger.Info("retrieving anchore user", fnCtx)

	usr, resp, err := a.getRestClient().UserManagementApi.GetAccountUser(a.authorizedContext(ctx), userName, userName)
	if err != nil && resp == nil { // TODO: simplify error checking (openapi returns a generic error for 404 as well)
		a.logger.Debug("failed to retrieve user from anchore", fnCtx)

		return nil, errors.WrapIfWithDetails(err, "failed to retrieve user from anchore", fnCtx)
	}

	if resp.StatusCode == http.StatusNotFound {
		// user not found
		return nil, nil
	}

	if err != nil {
		a.logger.Debug("failed to retrieve user from anchore", fnCtx)

		return nil, errors.WrapIfWithDetails(err, "failed to retrieve user from anchore", fnCtx)
	}

	return usr, nil
}

func (a anchoreClient) GetUserCredentials(ctx context.Context, userName string) (string, error) {
	fnCtx := map[string]interface{}{"userName": userName}
	a.logger.Info("retrieving anchore credentials", fnCtx)

	credentials, resp, err := a.getRestClient().UserManagementApi.ListUserCredentials(a.authorizedContext(ctx), userName, userName)
	if err != nil || (resp.StatusCode != http.StatusOK) {
		a.logger.Debug("failed to retrieve user credentials from anchore", fnCtx)

		return "", errors.WrapIfWithDetails(err, "failed to retrieve user credentials from anchore", fnCtx)
	}

	for _, credential := range credentials {
		if credential.Value != "" {
			return credential.Value, nil
		}
	}

	return "", errors.NewWithDetails("no credentials found", "userName", userName)
}

func (a anchoreClient) DeleteAccount(ctx context.Context, accountName string) error {
	fnCtx := map[string]interface{}{"accountName": accountName}
	a.logger.Info("deleting anchore account", fnCtx)

	// update the status of the account before delete
	s, ur, err := a.getRestClient().UserManagementApi.UpdateAccountState(a.authorizedContext(ctx), accountName, anchore.AccountStatus{State: "disabled"})

	if err != nil || ur.StatusCode != http.StatusOK || s.State != "disabled" {
		a.logger.Debug("failed to deactivate anchore account", fnCtx)

		return errors.WrapIfWithDetails(err, "failed to deactivate anchore account", fnCtx)
	}

	// delete the account upon successful disable
	dr, err := a.getRestClient().UserManagementApi.DeleteAccount(a.authorizedContext(ctx), accountName)
	if err != nil || (dr.StatusCode != http.StatusOK && dr.StatusCode != http.StatusNoContent) {
		a.logger.Debug("failed to delete anchore account", fnCtx)

		return errors.WrapIfWithDetails(err, "failed to delete anchore account", fnCtx)
	}

	a.logger.Info("deleted anchore account", fnCtx)
	return nil
}

func (a anchoreClient) GetAccount(ctx context.Context, accountName string) (string, error) {
	fnCtx := map[string]interface{}{"accountName": accountName}
	a.logger.Info("retrieving anchore account", fnCtx)

	acc, r, err := a.getRestClient().UserManagementApi.GetAccount(a.authorizedContext(ctx), accountName)
	if err != nil || r.StatusCode != http.StatusOK {
		a.logger.Debug("failed to get anchore account", fnCtx)

		return "", errors.WrapIfWithDetails(err, "failed to get anchore account", fnCtx)
	}

	a.logger.Info("retrieved anchore account", fnCtx)
	return acc.Name, nil
}

func (a anchoreClient) DeleteUser(ctx context.Context, accountName string, userName string) error {
	fnCtx := map[string]interface{}{"accountName": accountName, "userName": userName}
	a.logger.Info("deleting anchore user", fnCtx)

	r, err := a.getRestClient().UserManagementApi.DeleteUser(a.authorizedContext(ctx), accountName, userName)
	if err != nil || r.StatusCode != http.StatusNoContent {
		a.logger.Debug("failed to delete anchore user", fnCtx)

		return errors.WrapIfWithDetails(err, "failed to delete anchore user", fnCtx)
	}

	a.logger.Info("deleted anchore user", fnCtx)
	return nil
}

func (a anchoreClient) ActivatePolicy(ctx context.Context, policyID string) error {
	fnCtx := map[string]interface{}{"policyId": policyID}
	a.logger.Info("activating anchore policy", fnCtx)

	getOpts := &anchore.GetPolicyOpts{Detail: optional.NewBool(true)}

	policyBundle, resp, err := a.getRestClient().PoliciesApi.GetPolicy(a.authorizedContext(ctx), policyID, getOpts)
	if err != nil || (resp.StatusCode != http.StatusOK) {
		a.logger.Debug("failed to get anchore policy", fnCtx)

		return errors.WrapIfWithDetails(err, "failed to get anchore policy", fnCtx)
	}

	policyBundle[0].Active = true

	updateOpts := &anchore.UpdatePolicyOpts{Active: optional.NewBool(true)}

	_, resp, err = a.getRestClient().PoliciesApi.UpdatePolicy(a.authorizedContext(ctx), policyID, policyBundle[0], updateOpts)
	if err != nil || (resp.StatusCode != http.StatusOK) {
		a.logger.Debug("failed to activate policy", fnCtx)

		return errors.WrapIfWithDetails(err, "failed to activate policy", fnCtx)
	}

	a.logger.Info("anchore policy activated", fnCtx)
	return nil
}

func (a anchoreClient) CreatePolicy(ctx context.Context, policyRaw map[string]interface{}) (string, error) {
	fnCtx := map[string]interface{}{"policy": policyRaw}
	a.logger.Info("creating anchore policy", fnCtx)

	rawPolicyData, err := json.Marshal(policyRaw)
	if err != nil {
		return "", errors.WrapIfWithDetails(err, "failed to marshal policy", fnCtx)
	}

	var policy anchore.PolicyBundle
	err = json.Unmarshal(rawPolicyData, &policy)
	if err != nil {
		return "", errors.WrapIfWithDetails(err, "failed to unmarshal policy", fnCtx)
	}

	policyRecord, resp, err := a.getRestClient().PoliciesApi.AddPolicy(a.authorizedContext(ctx), policy, nil)
	if err != nil || (resp.StatusCode != http.StatusOK) {
		a.logger.Debug("failed to create anchore policy", fnCtx)

		return "", errors.WrapIfWithDetails(err, "failed to create anchore policy", fnCtx)
	}

	a.logger.Info("anchore policy created", fnCtx)
	return policyRecord.PolicyId, nil
}

func (a anchoreClient) AddRegistry(ctx context.Context, registry Registry) error {
	fnCtx := map[string]interface{}{"registry": registry.Registry}
	a.logger.Info("adding anchore registry", fnCtx)

	registryType := registry.Type
	if registryType == "" {
		if IsEcrRegistry(registry.Registry) {
			registryType = "awsecr"
		} else {
			registryType = "docker_v2"
		}
	}

	request := anchore.RegistryConfigurationRequest{
		Registry:       registry.Registry,
		RegistryName:   registry.Registry,
		RegistryUser:   registry.Username,
		RegistryPass:   registry.Password,
		RegistryType:   registryType,
		RegistryVerify: registry.Verify,
	}

	opts := &anchore.CreateRegistryOpts{Validate: optional.NewBool(true)}

	_, resp, err := a.getRestClient().RegistriesApi.CreateRegistry(a.authorizedContext(ctx), request, opts)

	if err != nil || resp.StatusCode != http.StatusOK {
		a.logger.Debug("failed to add anchore registry", fnCtx)

		return errors.WrapIfWithDetails(err, "failed to add anchore registry", fnCtx)
	}

	a.logger.Info("anchore registry added", fnCtx)
	return nil
}

func (a anchoreClient) GetRegistry(ctx context.Context, registryName string) ([]anchore.RegistryConfiguration, error) {
	a.logger.Info("getting anchore registry", map[string]interface{}{
		"registryName": registryName,
	})

	opts := &anchore.GetRegistryOpts{}
	registry, resp, err := a.getRestClient().RegistriesApi.GetRegistry(a.authorizedContext(ctx), registryName, opts)

	if err != nil || (resp.StatusCode != http.StatusOK) {
		return nil, errors.WrapIfWithDetails(err, "failed to get registry", registryName)
	}

	return registry, nil
}

func (a anchoreClient) UpdateRegistry(ctx context.Context, registry Registry) error {
	fnCtx := map[string]interface{}{"registry": registry.Registry}
	a.logger.Info("updating anchore registry", fnCtx)

	// https://github.com/anchore/anchore-engine/issues/847
	// using DeleteRegistry and AddRegistry instead of updateRegistry because UpdateRegistry doesn't work in anchore-engine
	if err := a.DeleteRegistry(ctx, registry); err != nil {
		return err
	}

	if err := a.AddRegistry(ctx, registry); err != nil {
		return err
	}

	a.logger.Info("anchore registry updated", fnCtx)
	return nil
}

func (a anchoreClient) DeleteRegistry(ctx context.Context, registry Registry) error {
	fnCtx := map[string]interface{}{"registry": registry.Registry}
	a.logger.Info("deleting anchore registry", fnCtx)

	opts := &anchore.DeleteRegistryOpts{}
	resp, err := a.getRestClient().RegistriesApi.DeleteRegistry(a.authorizedContext(ctx), registry.Registry, opts)
	if err != nil || resp.StatusCode != http.StatusOK {
		return errors.WrapIfWithDetails(err, "failed to delete anchore registry", fnCtx)
	}

	a.logger.Info("anchore registry deleted", fnCtx)
	return nil
}

func (a anchoreClient) authorizedContext(ctx context.Context) context.Context {
	basicAuth := anchore.BasicAuth{
		UserName: a.userName,
		Password: a.password,
	}

	return context.WithValue(ctx, anchore.ContextBasicAuth, basicAuth)
}

func (a anchoreClient) getRestClient() *anchore.APIClient {
	return anchore.NewAPIClient(&anchore.Configuration{
		BasePath:      a.endpoint,
		DefaultHeader: make(map[string]string),
		UserAgent:     "Pipeline/go",
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: a.insecure,
				},
			},
		},
	})
}

// authenticatedResty sets up an authenticated resty client (this might be cached probably)
// WARNING: resty is temporarily used only as the generated client / openAPI spec seems not to be complete.
func (a anchoreClient) authenticatedResty() *resty.Request {
	return resty.R().SetBasicAuth(a.userName, a.password).SetHeader("User-Agent", "Pipeline/go")
}
