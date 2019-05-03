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

package buckets

import (
	"net/http"

	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/jinzhu/gorm"

	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/pkg/errors"

	"github.com/banzaicloud/pipeline/api/ark/common"
	"github.com/banzaicloud/pipeline/auth"
	"github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/ark"
	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
)

// Create creates an ARK backup bucket
func Create(c *gin.Context) {
	logger := correlationid.Logger(common.Log, c)
	logger.Info("creating bucket")

	var request api.CreateBucketRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		err = emperror.Wrap(err, "could not parse request")
		common.ErrorHandler.Handle(err)
		common.ErrorResponse(c, err)
		return
	}

	org := auth.GetCurrentOrganization(c.Request)

	if len(request.Location) == 0 && (request.Cloud == providers.Alibaba || request.Cloud == providers.Amazon) {
		// location field is empty in request, get bucket location
		location, err := common.GetBucketLocation(request.Cloud, request.BucketName, request.SecretID, org.ID, logger)
		if err != nil {
			err = emperror.WrapWith(err, "failed to get bucket region", "bucket", request.BucketName)
			common.ErrorHandler.Handle(err)
			common.ErrorResponse(c, err)
			return
		}

		request.Location = location
	}

	bs := ark.BucketsServiceFactory(org, config.DB(), logger)

	_, err := bs.GetByRequest(api.FindBucketRequest{
		Cloud:      request.Cloud,
		BucketName: request.BucketName,
		Location:   request.Location,
	})
	if err == nil {
		err = errors.New("bucket already exists")
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		err = emperror.Wrap(err, "could not create bucket")
		common.ErrorHandler.Handle(err)
		common.ErrorResponse(c, err)
		return
	}

	bucket, err := bs.FindOrCreateBucket(&api.CreateBucketRequest{
		Cloud:      request.Cloud,
		BucketName: request.BucketName,
		Location:   request.Location,
		SecretID:   request.SecretID,
	})
	if err != nil {
		err = emperror.Wrap(err, "could not persist bucket")
		common.ErrorHandler.Handle(err)
		common.ErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, bucket.ConvertModelToEntity())
}
