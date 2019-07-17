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
	"net/http"
	"strings"

	"emperror.dev/emperror"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/cluster"
	pipConfig "github.com/banzaicloud/pipeline/config"
)

// DomainAPI implements the Domain API actions
type DomainAPI struct {
	clusterManager *cluster.Manager

	logger       logrus.FieldLogger
	errorHandler emperror.Handler
}

// NewDomainAPI returns a new DomainAPI instance.
func NewDomainAPI(clusterManager *cluster.Manager, logger logrus.FieldLogger, errorHandler emperror.Handler) *DomainAPI {
	return &DomainAPI{
		clusterManager: clusterManager,

		logger:       logger,
		errorHandler: errorHandler,
	}
}

// GetDomainResponse describes Pipeline's GetDomain API response
type GetDomainResponse struct {
	DomainName string `json:"domainName" yaml:"domainName"`
}

// GetDomain returns the base domain for the cluster/org
func (a *DomainAPI) GetDomain(c *gin.Context) {
	organizationID := auth.GetCurrentOrganization(c.Request).ID

	logger := a.logger.WithFields(logrus.Fields{
		"organization": organizationID,
	})

	logger.Info("Fetching domain information")
	clusterID := c.Query("clusterid")
	var baseDomain string
	if clusterID != "" {
		// TODO implement cluster based domain separation
	}
	// TODO implement org based domain separation
	baseDomain = strings.ToLower(viper.GetString(pipConfig.DNSBaseDomain))

	c.JSON(http.StatusOK, GetDomainResponse{DomainName: baseDomain})
}
