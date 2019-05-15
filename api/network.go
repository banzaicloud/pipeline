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

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	"github.com/banzaicloud/pipeline/internal/providers"
	pkgProviders "github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// NetworkAPI implements network functions
type NetworkAPI struct {
	logger logrus.FieldLogger
}

// NewNetworkAPI returns a new NetworkAPI instance
func NewNetworkAPI(logger logrus.FieldLogger) *NetworkAPI {
	return &NetworkAPI{
		logger: logger,
	}
}

// NetworkInfo encapsulates VPC network information to be returned
type NetworkInfo struct {
	CIDRs []string `json:"cidrs" binding:"required"`
	ID    string   `json:"id" binding:"required"`
	Name  string   `json:"name,omitempty"`
}

// SubnetInfo encapsulates VPC subnetwork information to be returned
type SubnetInfo struct {
	CIDRs    []string `json:"cidrs" binding:"required"`
	ID       string   `json:"id" binding:"required"`
	Location string   `json:"location,omitempty"`
	Name     string   `json:"name,omitempty"`
}

// RouteTableInfo encapsulates VPC route table information to be returned
type RouteTableInfo struct {
	ID   string `json:"id" binding:"required"`
	Name string `json:"name,omitempty"`
}

func filterEmpty(strings []string) (result []string) {
	for _, s := range strings {
		if len(s) != 0 {
			result = append(result, s)
		}
	}
	return
}

// ListVPCNetworks lists all VPC networks of the specified organization
func (a *NetworkAPI) ListVPCNetworks(ctx *gin.Context) {
	logger := correlationid.Logger(a.logger, ctx)

	organization := auth.GetCurrentOrganization(ctx.Request)
	provider, ok := getRequiredProviderFromContext(ctx, logger)
	if !ok {
		return
	}
	region, resourceGroup, ok := getRequiredRegionOrResourceGroupFromContext(ctx, provider, logger)
	if !ok {
		return
	}
	secretID, ok := getRequiredSecretIDFromContext(ctx, logger)
	if !ok {
		return
	}

	logger = logger.WithFields(logrus.Fields{
		"organization":  organization.ID,
		"provider":      provider,
		"region":        region,
		"resourceGroup": resourceGroup,
		"secretID":      secretID,
	})

	sir, err := secret.Store.Get(organization.ID, secretID)
	if err != nil {
		replyWithError(ctx, err)
		return
	}

	err = sir.ValidateSecretType(provider)
	if err != nil {
		replyWithError(ctx, err)
		return
	}

	svcParams := providers.ServiceParams{
		Logger:            logger,
		Provider:          provider,
		Region:            region,
		ResourceGroupName: resourceGroup,
		Secret:            sir,
	}
	svc, err := providers.NewNetworkService(svcParams)
	if err != nil {
		replyWithError(ctx, err)
		return
	}
	networks, err := svc.ListNetworks()
	if err != nil {
		replyWithError(ctx, err)
		return
	}

	networkInfos := make([]NetworkInfo, len(networks))
	for i := range networks {
		networkInfos[i].CIDRs = filterEmpty(networks[i].CIDRs())
		networkInfos[i].ID = networks[i].ID()
		networkInfos[i].Name = networks[i].Name()
	}
	ctx.JSON(http.StatusOK, networkInfos)
}

// ListVPCSubnets lists all subnetworks of the specified VPC network
func (a *NetworkAPI) ListVPCSubnets(ctx *gin.Context) {
	logger := correlationid.Logger(a.logger, ctx)

	organization := auth.GetCurrentOrganization(ctx.Request)
	provider, ok := getRequiredProviderFromContext(ctx, logger)
	if !ok {
		return
	}
	region, resourceGroup, ok := getRequiredRegionOrResourceGroupFromContext(ctx, provider, logger)
	if !ok {
		return
	}
	secretID, ok := getRequiredSecretIDFromContext(ctx, logger)
	if !ok {
		return
	}
	networkID := ctx.Param("id")

	logger = logger.WithFields(logrus.Fields{
		"organization":  organization.ID,
		"provider":      provider,
		"region":        region,
		"resourceGroup": resourceGroup,
		"secretID":      secretID,
		"networkID":     networkID,
	})

	sir, err := secret.Store.Get(organization.ID, secretID)
	if err != nil {
		replyWithError(ctx, err)
		return
	}

	err = sir.ValidateSecretType(provider)
	if err != nil {
		replyWithError(ctx, err)
		return
	}

	svcParams := providers.ServiceParams{
		Logger:            logger,
		Provider:          provider,
		Region:            region,
		ResourceGroupName: resourceGroup,
		Secret:            sir,
	}
	svc, err := providers.NewNetworkService(svcParams)
	if err != nil {
		replyWithError(ctx, err)
		return
	}
	subnets, err := svc.ListSubnets(networkID)
	if err != nil {
		replyWithError(ctx, err)
		return
	}

	subnetInfos := make([]SubnetInfo, len(subnets))
	for i := range subnets {
		subnetInfos[i].CIDRs = filterEmpty(subnets[i].CIDRs())
		subnetInfos[i].ID = subnets[i].ID()
		subnetInfos[i].Location = subnets[i].Location()
		subnetInfos[i].Name = subnets[i].Name()
	}
	ctx.JSON(http.StatusOK, subnetInfos)
}

// ListRouteTables lists all route tables of the specified VPC network
func (a *NetworkAPI) ListRouteTables(ctx *gin.Context) {
	logger := correlationid.Logger(a.logger, ctx)

	organization := auth.GetCurrentOrganization(ctx.Request)
	provider, ok := getRequiredProviderFromContext(ctx, logger)
	if !ok {
		return
	}
	region, resourceGroup, ok := getRequiredRegionOrResourceGroupFromContext(ctx, provider, logger)
	if !ok {
		return
	}
	secretID, ok := getRequiredSecretIDFromContext(ctx, logger)
	if !ok {
		return
	}
	networkID := ctx.Param("id")

	logger = logger.WithFields(logrus.Fields{
		"organization":  organization.ID,
		"provider":      provider,
		"region":        region,
		"resourceGroup": resourceGroup,
		"secretID":      secretID,
		"networkID":     networkID,
	})

	sir, err := secret.Store.Get(organization.ID, secretID)
	if err != nil {
		replyWithError(ctx, err)
		return
	}

	err = sir.ValidateSecretType(provider)
	if err != nil {
		replyWithError(ctx, err)
		return
	}

	svcParams := providers.ServiceParams{
		Logger:            logger,
		Provider:          provider,
		Region:            region,
		ResourceGroupName: resourceGroup,
		Secret:            sir,
	}
	svc, err := providers.NewNetworkService(svcParams)
	if err != nil {
		replyWithError(ctx, err)
		return
	}
	routeTables, err := svc.ListRouteTables(networkID)
	if err != nil {
		replyWithError(ctx, err)
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
	return provider, ok
}

func getRequiredRegionOrResourceGroupFromContext(ctx *gin.Context, provider string, logger logrus.FieldLogger) (string, string, bool) {
	switch provider {
	case pkgProviders.Azure:
		resourceGroup, ok := ginutils.RequiredQueryOrAbort(ctx, "resourceGroup")
		return "", resourceGroup, ok
	default:
		region, ok := ginutils.RequiredQueryOrAbort(ctx, "region")
		return region, "", ok
	}
}

func getRequiredSecretIDFromContext(ctx *gin.Context, logger logrus.FieldLogger) (string, bool) {
	secretID, ok := ginutils.GetRequiredHeader(ctx, "secretId")
	return secretID, ok
}

func replyWithError(ctx *gin.Context, err error) {
	ginutils.ReplyWithErrorResponse(ctx, ErrorResponseFrom(err))
}
