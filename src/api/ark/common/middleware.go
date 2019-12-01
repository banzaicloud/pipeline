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

package common

import (
	"context"
	"net/http"

	"emperror.dev/emperror"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/ark"
	"github.com/banzaicloud/pipeline/internal/cluster/clusteradapter"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	"github.com/banzaicloud/pipeline/src/auth"
	"github.com/banzaicloud/pipeline/src/cluster"
)

const (
	arkServiceName = "arkService"
)

// ARKMiddleware is a middleware for initializing a CommonCluster and an ARKService
// from the request parameters for later use
func ARKMiddleware(db *gorm.DB, logger logrus.FieldLogger) gin.HandlerFunc {

	return func(c *gin.Context) {
		logger = logger.WithField("middleware", "ARK")

		clusters := clusteradapter.NewClusters(db)
		clusterID, ok := ginutils.UintParam(c, "id")
		if !ok {
			c.Abort()
			return
		}
		org := auth.GetCurrentOrganization(c.Request)
		if org == nil {
			err := errors.New("invalid organization")
			ErrorResponse(c, err)
			ErrorHandler.Handle(err)
			return
		}
		cl, err := clusters.FindOneByID(org.ID, clusterID)
		if err != nil {
			err = emperror.Wrap(err, "could not find cluster")
			ErrorResponse(c, err)
			ErrorHandler.Handle(err)
			return
		}

		cluster, err := cluster.GetCommonClusterFromModel(cl)
		if err != nil {
			err = emperror.Wrap(err, "could not get cluster from model")
			ErrorResponse(c, err)
			ErrorHandler.Handle(err)
			return
		}

		svc := ark.NewARKService(org, cluster, db, logger)
		c.Request = setVariableToContext(c.Request, arkServiceName, svc)

		c.Next()
	}
}

// GetARKService return the current ark.Service model
func GetARKService(req *http.Request) *ark.Service {
	if svc := req.Context().Value(arkServiceName); svc != nil {
		return svc.(*ark.Service)
	}
	return nil
}

func setVariableToContext(req *http.Request, key interface{}, val interface{}) *http.Request {

	return req.WithContext(context.WithValue(req.Context(), key, val))
}
