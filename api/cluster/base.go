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

	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/secret"
)

// CreateClusterRequestBase describes the common base of cluster creation requests
type CreateClusterRequestBase struct {
	Name       string               `json:"name" yaml:"name" binding:"required"`
	PostHooks  pkgCluster.PostHooks `json:"postHooks" yaml:"postHooks"`
	SecretID   string               `json:"secretId" yaml:"secretId"`
	SecretIDs  []string             `json:"secretIds,omitempty" yaml:"secretIds,omitempty"`
	SecretName string               `json:"secretName" yaml:"secretName"`
	Type       string               `json:"type" yaml:"type" binding:"required"`
}

func getSecretByID(organizationID uint, secretID string) (*secret.SecretItemResponse, *pkgCommon.ErrorResponse) {
	if secretID == "" {
		return nil, nil
	}
	sir, err := secret.Store.Get(organizationID, secretID)
	if err == nil {
		return sir, nil
	}
	if err == secret.ErrSecretNotExists {
		return nil, &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "no secret exists with the specified ID",
			Error:   err.Error(),
		}
	}
	return nil, &pkgCommon.ErrorResponse{
		Code:    http.StatusInternalServerError,
		Message: "failed to retreive secret by ID",
		Error:   err.Error(),
	}
}

func getSecretByName(organizationID uint, secretName string) (*secret.SecretItemResponse, *pkgCommon.ErrorResponse) {
	if secretName == "" {
		return nil, nil
	}
	sir, err := secret.Store.GetByName(organizationID, secretName)
	if err == nil {
		return sir, nil
	}
	if err == secret.ErrSecretNotExists {
		return nil, &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "no secret exists with the specified name",
			Error:   err.Error(),
		}
	}
	return nil, &pkgCommon.ErrorResponse{
		Code:    http.StatusInternalServerError,
		Message: "failed to retreive secret by name",
		Error:   err.Error(),
	}
}

func getSecretWithType(organizationID uint, secretIDs []string, secretType string) (*secret.SecretItemResponse, *pkgCommon.ErrorResponse) {
	for _, id := range secretIDs {
		sir, err := secret.Store.Get(organizationID, id)
		if err == nil && sir.Type == secretType {
			return sir, nil
		}
		if err != secret.ErrSecretNotExists {
			return nil, &pkgCommon.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "failed to retreive secret by ID",
				Error:   err.Error(),
			}
		}
	}
	return nil, nil
}

func getSecretFromRequest(orgID uint, req CreateClusterRequestBase, providerID string) (*secret.SecretItemResponse, *pkgCommon.ErrorResponse) {
	sir, errRes := getSecretByID(orgID, req.SecretID)
	if errRes != nil {
		return nil, errRes
	}
	if sir == nil {
		sir, errRes = getSecretByName(orgID, req.SecretName)
	}
	if errRes != nil {
		return nil, errRes
	}
	if sir == nil {
		sir, errRes = getSecretWithType(orgID, req.SecretIDs, providerID)
	}
	if errRes != nil {
		return nil, errRes
	}
	if sir == nil {
		msg := "no suitable secret provided in request"
		return nil, &pkgCommon.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: msg,
			Error:   msg,
		}
	}
	return sir, nil
}
