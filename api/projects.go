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

package api

import (
	"context"
	"net/http"

	"github.com/banzaicloud/pipeline/auth"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/secret/verify"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/cloudresourcemanager/v1"
)

type ListProjectsResponse struct {
	Projects []*cloudresourcemanager.Project `json:"projects,omitempty"`
}

// servicesContext encapsulates contextual information required for performing services related operations
// Primarily it's intended to be populated with information coming from the Gin context (header, path, request ...)
type servicesContext struct {
	log      logrus.FieldLogger
	orgId    uint
	secretId string
}

// newServicesCtx
func newServicesCtx(orgId uint, secretId string) *servicesContext {
	return &servicesContext{
		orgId:    orgId,
		secretId: secretId,
		log:      log.WithFields(logrus.Fields{"cloud": "google", "service": "projects"}),
	}
}

// GetProjects retrieves from the cloud the list of projects visible for the user represented by the secret passed in the header
func GetProjects(c *gin.Context) {

	organization := auth.GetCurrentOrganization(c.Request)

	secretID, ok := ginutils.GetRequiredHeader(c, secretIdHeader)
	if !ok {
		return
	}
	log.Debugf("retrieving projects for orgId: [%d], secretId [%s]",
		organization.ID, secretID)
	servicesCtx := newServicesCtx(organization.ID, secretID)

	cli, err := servicesCtx.httpClient()
	if err != nil {
		log.WithError(err).Error("could not build http client")
		ginutils.ReplyWithErrorResponse(c, ErrorResponseFrom(err))
		return
	}

	projectsSvc, err := servicesCtx.projectsService(cli)
	if err != nil {
		log.WithError(err).Error("could not build projects service")
		ginutils.ReplyWithErrorResponse(c, ErrorResponseFrom(err))
		return
	}

	req := projectsSvc.List()
	if err := req.Pages(context.Background(), func(page *cloudresourcemanager.ListProjectsResponse) error {
		c.JSON(http.StatusOK, ListProjectsResponse{Projects: page.Projects})
		return nil
	}); err != nil {
		log.WithError(err).Error("could not retrieve projects")
		ginutils.ReplyWithErrorResponse(c, ErrorResponseFrom(err))
		return
	}

}

// httpClient builds a http client with the service account available through the secret and organization
func (sc *servicesContext) httpClient() (*http.Client, error) {

	secret, err := getValidatedSecret(sc.orgId, sc.secretId, providers.Google)
	if err != nil {
		return nil, err
	}

	cl, err := verify.CreateOath2Client(verify.CreateServiceAccount(secret.Values))
	if err != nil {
		return nil, err
	}

	return cl, nil

}

// projectsService boilerplate for creating a projectsService instance to access cloud resources
func (sc *servicesContext) projectsService(cli *http.Client) (*cloudresourcemanager.ProjectsService, error) {

	svc, err := cloudresourcemanager.New(cli)
	if err != nil {
		return nil, err
	}

	return cloudresourcemanager.NewProjectsService(svc), nil
}
