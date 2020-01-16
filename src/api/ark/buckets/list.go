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

	"emperror.dev/errors"
	"github.com/gin-gonic/gin"

	"github.com/banzaicloud/pipeline/internal/ark"
	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	"github.com/banzaicloud/pipeline/src/api/ark/common"
	"github.com/banzaicloud/pipeline/src/auth"
)

// List lists ARK backup buckets
func List(c *gin.Context) {
	logger := correlationid.Logger(common.Log, c)
	logger.Info("getting buckets")

	bs := ark.BucketsServiceFactory(auth.GetCurrentOrganization(c.Request), global.DB(), logger)
	buckets, err := bs.List()
	if err != nil {
		err = errors.WrapIf(err, "could not get buckets")
		common.ErrorHandler.Handle(err)
		common.ErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, buckets)
}
