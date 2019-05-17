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

package deployment

import (
	"fmt"
	"net/http"

	"github.com/banzaicloud/pipeline/internal/clustergroup/deployment"
	pkgCommon "github.com/banzaicloud/pipeline/pkg/common"
	"github.com/gin-gonic/gin"
)

func (n *API) returnOperationErrorsIfAny(c *gin.Context, targetStatuses []deployment.TargetClusterStatus, releaseName string) bool {
	errMsg := ""
	for _, status := range targetStatuses {
		if len(status.Error) > 0 {
			if len(errMsg) > 0 {
				errMsg += " | "
			}
			errMsg += fmt.Sprint("CLUSTER: " + status.ClusterName + " CAUSE: " + status.Error)
		}
	}

	if len(errMsg) > 0 {
		fmtMsg := fmt.Sprintf("Some operations related to multi-cluster deployment %s were not successful: %s", releaseName, errMsg)
		c.JSON(http.StatusMultiStatus, pkgCommon.ErrorResponse{
			Code:    http.StatusMultiStatus,
			Message: fmtMsg,
			Error:   fmtMsg,
		})
		return true
	}

	return false
}
