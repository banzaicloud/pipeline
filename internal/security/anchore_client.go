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
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/antihax/optional"

	"github.com/banzaicloud/pipeline/.gen/anchore"
	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/internal/common"
)

type UserManagementClient interface {
	CreateAccount(ctx context.Context, accountName string, email string) error
	DeleteAccount(ctx context.Context, accountName string) error
	GetAccount(ctx context.Context, accountName string) (string, error)
	CreateUser(ctx context.Context, accountName string, userName string, password string) error
	DeleteUser(ctx context.Context, accountName string, userName string) error
	GetUser(ctx context.Context, userName string) (interface{}, error)
	GetUserCredentials(ctx context.Context, userName string) (string, error)
}

type ImagesClient interface {
	// GetImageVulnerabilities gets the vulnerabilities for the given image digest
	ScanImage(ctx context.Context, image pipeline.ClusterImage) (interface{}, error)
	// GetImageVulnerabilities gets the vulnerabilities for the given image digest
	GetImageVulnerabilities(ctx context.Context, imageDigest string) (interface{}, error)
	// CheckImage cheks rthe image for anchore metadata
	CheckImage(ctx context.Context, imageDigest string) (interface{}, error)
}

type PolicyClient interface {
	GetPolicy(ctx context.Context, policyID string) (*pipeline.PolicyBundleRecord, error)
	ListPolicies(ctx context.Context) (interface{}, error)
	CreatePolicy(ctx context.Context, policy pipeline.PolicyBundleRecord) (interface{}, error)
	DeletePolicy(ctx context.Context, policyID string) error
	UpdatePolicy(ctx context.Context, policyID string, policy pipeline.PolicyBundleRecord) error
}

// AnchoreClient "facade" for supported Anchore operations
type AnchoreClient interface {
	UserManagementClient
	ImagesClient
	PolicyClient
}

type anchoreClient struct {
	userName string
	password string
	endpoint string
	logger   common.Logger
}

func NewAnchoreClient(userName string, password string, endpoint string, logger common.Logger) AnchoreClient {
	return anchoreClient{
		userName: userName,
		password: password,
		endpoint: endpoint,
		logger:   logger.WithFields(map[string]interface{}{"anchore-client": ""}),
	}
}

func (a anchoreClient) UpdatePolicy(ctx context.Context, policyID string, policy pipeline.PolicyBundleRecord) error {
	fnCtx := map[string]interface{}{"policyID": policyID}
	a.logger.Info("updating policy", fnCtx)

	var toUpdate anchore.PolicyBundleRecord
	if err := a.transform(&policy, &toUpdate); err != nil {
		return errors.WrapIf(err, "failed to transform")
	}

	_, r, err := a.getRestClient().PoliciesApi.UpdatePolicy(a.authorizedContext(ctx), policyID, toUpdate, &anchore.UpdatePolicyOpts{})
	if err != nil || r.StatusCode != http.StatusOK {
		a.logger.Debug("failed to update policy", fnCtx)

		return errors.WrapIfWithDetails(err, "failed to update policy", fnCtx)
	}

	a.logger.Info("policy successfully updated", fnCtx)
	return nil
}

func (a anchoreClient) GetPolicy(ctx context.Context, policyID string) (*pipeline.PolicyBundleRecord, error) {
	fnCtx := map[string]interface{}{"policyID": policyID}
	a.logger.Info("retrieving policy", fnCtx)

	policyBundles, r, err := a.getRestClient().PoliciesApi.GetPolicy(a.authorizedContext(ctx), policyID, &anchore.GetPolicyOpts{
		Detail: optional.NewBool(true),
	})
	if err != nil || r.StatusCode != http.StatusOK {
		a.logger.Debug("failed to retrieve policy", fnCtx)

		return nil, errors.WrapIfWithDetails(err, "failed to retrieve policy", fnCtx)
	}

	a.logger.Info("policy successfully retrieved", fnCtx)
	var bundle pipeline.PolicyBundleRecord
	if err = a.transform(policyBundles[0], &bundle); err != nil {
		a.logger.Debug("failed to transform policy", fnCtx)

		return nil, errors.WrapIfWithDetails(err, "failed to transform policy", fnCtx)
	}

	return &bundle, nil
}

func (a anchoreClient) ListPolicies(ctx context.Context) (interface{}, error) {
	a.logger.Info("retrieving policies ...")

	policies, r, err := a.getRestClient().PoliciesApi.ListPolicies(a.authorizedContext(ctx),
		&anchore.ListPoliciesOpts{
			Detail: optional.NewBool(true),
		})

	if err != nil || r.StatusCode != http.StatusOK {
		a.logger.Debug("failed to retrieve policies")

		return nil, errors.WrapIfWithDetails(err, "failed to retrieve policies")
	}

	a.logger.Info("policies successfully retrieved")
	return policies, nil
}

func (a anchoreClient) CreatePolicy(ctx context.Context, policy pipeline.PolicyBundleRecord) (interface{}, error) {
	fnCtx := map[string]interface{}{"policyID": policy}
	a.logger.Info("creating policy ...", fnCtx)

	var bundle anchore.PolicyBundle
	if err := a.transform(policy, &bundle); err != nil {
		a.logger.Debug("failed to transform policy", fnCtx)

		return nil, errors.WrapIfWithDetails(err, "failed to transform policy")
	}

	policyBundleReecord, r, err := a.getRestClient().PoliciesApi.AddPolicy(a.authorizedContext(ctx), bundle, &anchore.AddPolicyOpts{})
	if err != nil || r.StatusCode != http.StatusOK {
		a.logger.Debug("failed to create policy", fnCtx)

		return nil, errors.WrapIfWithDetails(err, "failed to create policy")
	}

	a.logger.Info("policy successfully created", fnCtx)
	return policyBundleReecord, nil
}

func (a anchoreClient) DeletePolicy(ctx context.Context, policyID string) error {
	fnCtx := map[string]interface{}{"policyID": policyID}
	a.logger.Info("deleting policy ...", fnCtx)

	r, err := a.getRestClient().PoliciesApi.DeletePolicy(a.authorizedContext(ctx), policyID, &anchore.DeletePolicyOpts{})
	if err != nil || r.StatusCode != http.StatusOK {
		a.logger.Debug("failed to delete policy", fnCtx)

		return errors.WrapIfWithDetails(err, "failed to delete policy")
	}

	a.logger.Info("policy successfully deleted")
	return nil

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

	a.logger.Info("deleted anchore account", fnCtx)
	return nil
}

// ScanImage registers an image for security scanning
func (a anchoreClient) ScanImage(ctx context.Context, image pipeline.ClusterImage) (interface{}, error) {

	aImg, resp, err := a.getRestClient().ImagesApi.AddImage(a.authorizedContext(ctx), anchore.ImageAnalysisRequest{
		Digest:    image.ImageDigest,
		Tag:       strings.Join([]string{image.ImageName, image.ImageTag}, ":"),
		CreatedAt: time.Now().UTC(),
	}, &anchore.AddImageOpts{})

	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, errors.WrapIfWithDetails(err, "failed to add image", image.ImageDigest)
	}

	a.logger.Debug("image added for security scan", map[string]interface{}{"image": aImg})
	return aImg, nil
}

func (a anchoreClient) GetImageVulnerabilities(ctx context.Context, imageDigest string) (interface{}, error) {
	a.logger.Debug("retrieving image vulnerabilities")

	vulnerabilities, resp, err := a.getRestClient().ImagesApi.GetImageVulnerabilitiesByType(a.authorizedContext(ctx),
		imageDigest, "all", &anchore.GetImageVulnerabilitiesByTypeOpts{})

	if err != nil || resp.StatusCode != http.StatusOK {
		a.logger.Debug("failed to retrieve image vulnerabilities")

		return nil, errors.WrapIf(err, "failed to retrieve vulnerabilities")
	}

	a.logger.Debug("successfully retrieved image vulnerabilities")
	return vulnerabilities, nil
}

func (a anchoreClient) CheckImage(ctx context.Context, imageDigest string) (interface{}, error) {
	a.logger.Debug("retrieving image metadata", map[string]interface{}{"imageDigest": imageDigest})

	imageMeta, resp, err := a.getRestClient().ImagesApi.GetImage(a.authorizedContext(ctx), imageDigest, &anchore.GetImageOpts{})

	if err != nil || resp.StatusCode != http.StatusOK {
		a.logger.Debug("failure while retrieving image metadata", map[string]interface{}{"imageDigest": imageDigest})

		return nil, errors.WrapIf(err, "failure while retrieving image metadata")
	}

	a.logger.Debug("successfully retrieved image metadata", map[string]interface{}{"imageDigest": imageDigest})
	return imageMeta, nil
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
	})
}

// transform quick and dirty solution for transforming anchore types to pipeline types
func (a anchoreClient) transform(fromType interface{}, toType interface{}) error {
	anchoreBytes, err := json.Marshal(fromType)
	if err != nil {
		return errors.WrapIf(err, "failed to marshal the 'from' type")
	}

	if err := json.Unmarshal(anchoreBytes, toType); err != nil {
		return errors.WrapIf(err, "failed to unmarshal to 'toType' type")
	}

	return nil
}
