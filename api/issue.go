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

	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/issue"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type CreatePipelineIssueRequest struct {
	Organization string   `json:"organization" binding:"required"`
	Title        string   `json:"title" binding:"required"`
	Text         string   `json:"text" binding:"required"`
	Labels       []string `json:"labels"`
}

func NewIssueHandler(version, commitHash, buildDate string) (gin.HandlerFunc, error) {

	versionInformation := issue.VersionInformation{
		Version:    version,
		CommitHash: commitHash,
		BuildDate:  buildDate,
	}

	issuer, err := issue.NewIssuer(versionInformation)

	if err != nil {
		return nil, errors.Wrap(err, "failed to created issuer")
	}

	return func(c *gin.Context) {

		userID := auth.GetCurrentUser(c.Request).ID

		var request CreatePipelineIssueRequest
		if err := c.BindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, pkgCommon.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "error parsing request",
				Error:   err.Error(),
			})
			return
		}

		err := issuer.CreateIssue(userID, request.Organization, request.Title, request.Text, request.Labels)

		if err != nil {
			errorHandler.Handle(errors.Wrapf(err, "failed to create issue"))
			c.JSON(http.StatusInternalServerError, pkgCommon.ErrorResponse{
				Code:    http.StatusInternalServerError,
				Message: "failed to create issue",
				Error:   err.Error(),
			})
			return
		}

		c.Status(http.StatusCreated)

	}, nil
}
