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

package api

import (
	"net/http"

	"github.com/banzaicloud/pipeline/secret"

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	"github.com/banzaicloud/pipeline/internal/providers"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// NetworkAPI implements network functions
type NetworkAPI struct{}

// NewNetworkAPI returns a new NetworkAPI instance
func NewNetworkAPI() *NetworkAPI {
	return &NetworkAPI{}
}

// NetworkInfo encapsulates VPC network information to be returned
type NetworkInfo struct {
	CIDR string `json:"cidr" binding:"required"`
	ID   string `json:"id" binding:"required"`
	Name string `json:"name,omitempty"`
}

// SubnetInfo encapsulates VPC subnetwork information to be returned
type SubnetInfo struct {
	CIDR     string `json:"cidr" binding:"required"`
	ID       string `json:"id" binding:"required"`
	Location string `json:"location,omitempty"`
	Name     string `json:"name,omitempty"`
}

// RouteTableInfo encapsulates VPC route table information to be returned
type RouteTableInfo struct {
	ID   string `json:"id" binding:"required"`
	Name string `json:"name,omitempty"`
}

// ListVPCNetworks lists all VPC networks of the specified organization
func (a *NetworkAPI) ListVPCNetworks(ctx *gin.Context) {
	logger := correlationid.Logger(log, ctx)

	organization := auth.GetCurrentOrganization(ctx.Request)
	provider, ok := getRequiredProviderFromContext(ctx, logger)
	if !ok {
		return
	}
	secretID, ok := getRequiredSecretIDFromContext(ctx, logger)
	if !ok {
		return
	}

	logger = logger.WithFields(logrus.Fields{
		"organization": organization.ID,
		"provider":     provider,
		"secretID":     secretID,
	})

	sir, err := secret.Store.Get(organization.ID, secretID)
	if err != nil {
		ginutils.ReplyWithErrorResponse(ctx, errorResponseFrom(err))
		logger.Debug("no secret stored for ID")
		return
	}

	networkCtx := providers.NetworkContext{
		Logger:   logger,
		Provider: provider,
		Secret:   sir,
	}

	networks, err := providers.ListNetworks(networkCtx)
	if err != nil {
		ginutils.ReplyWithErrorResponse(ctx, errorResponseFrom(err))
		return
	}

	networkInfos := make([]NetworkInfo, len(networks))
	for i := range networks {
		networkInfos[i].CIDR = networks[i].CIDR()
		networkInfos[i].ID = networks[i].ID()
		networkInfos[i].Name = networks[i].Name()
	}
	ctx.JSON(http.StatusOK, networkInfos)
}

// ListVPCSubnets lists all subnetworks of the specified VPC network
func (a *NetworkAPI) ListVPCSubnets(ctx *gin.Context) {
	logger := correlationid.Logger(log, ctx)

	organization := auth.GetCurrentOrganization(ctx.Request)
	provider, ok := getRequiredProviderFromContext(ctx, logger)
	if !ok {
		return
	}
	secretID, ok := getRequiredSecretIDFromContext(ctx, logger)
	if !ok {
		return
	}
	networkID := ctx.Param("id")

	logger = logger.WithFields(logrus.Fields{
		"organization": organization.ID,
		"provider":     provider,
		"secretID":     secretID,
		"networkID":    networkID,
	})

	sir, err := secret.Store.Get(organization.ID, secretID)
	if err != nil {
		ginutils.ReplyWithErrorResponse(ctx, errorResponseFrom(err))
		logger.Debug("no secret stored for ID")
		return
	}

	networkCtx := providers.NetworkContext{
		Logger:   logger,
		Provider: provider,
		Secret:   sir,
	}

	subnets, err := providers.ListSubnets(networkCtx, networkID)
	if err != nil {
		ginutils.ReplyWithErrorResponse(ctx, errorResponseFrom(err))
		return
	}

	subnetInfos := make([]SubnetInfo, len(subnets))
	for i := range subnets {
		subnetInfos[i].CIDR = subnets[i].CIDR()
		subnetInfos[i].ID = subnets[i].ID()
		subnetInfos[i].Location = subnets[i].Location()
		subnetInfos[i].Name = subnets[i].Name()
	}
	ctx.JSON(http.StatusOK, subnetInfos)
}

// ListRouteTables lists all route tables of the specified VPC network
func (a *NetworkAPI) ListRouteTables(ctx *gin.Context) {
	logger := correlationid.Logger(log, ctx)

	organization := auth.GetCurrentOrganization(ctx.Request)
	provider, ok := getRequiredProviderFromContext(ctx, logger)
	if !ok {
		return
	}
	secretID, ok := getRequiredSecretIDFromContext(ctx, logger)
	if !ok {
		return
	}
	networkID := ctx.Param("id")

	logger = logger.WithFields(logrus.Fields{
		"organization": organization.ID,
		"provider":     provider,
		"secretID":     secretID,
		"networkID":    networkID,
	})

	sir, err := secret.Store.Get(organization.ID, secretID)
	if err != nil {
		ginutils.ReplyWithErrorResponse(ctx, errorResponseFrom(err))
		logger.Debug("no secret stored for ID")
		return
	}

	networkCtx := providers.NetworkContext{
		Logger:   logger,
		Provider: provider,
		Secret:   sir,
	}

	routeTables, err := providers.ListRouteTables(networkCtx, networkID)
	if err != nil {
		ginutils.ReplyWithErrorResponse(ctx, errorResponseFrom(err))
		return
	}

	routeTableInfos := make([]RouteTableInfo, len(routeTables))
	for i := range routeTables {
		routeTableInfos[i].ID = routeTables[i].ID()
		routeTableInfos[i].Name = routeTables[i].Name()
	}
	ctx.JSON(http.StatusOK, routeTableInfos)
}

func getRequiredProviderFromContext(ctx *gin.Context, logger logrus.FieldLogger) (string, bool) {
	provider, ok := ginutils.RequiredQueryOrAbort(ctx, "cloudType")
	if !ok {
		logger.Debug("missing provider")
	}
	return provider, ok
}

func getRequiredSecretIDFromContext(ctx *gin.Context, logger logrus.FieldLogger) (string, bool) {
	secretID, ok := ginutils.GetRequiredHeader(ctx, "secretId")
	if !ok {
		logger.Debug("missing secret ID")
	}
	return secretID, ok
}
