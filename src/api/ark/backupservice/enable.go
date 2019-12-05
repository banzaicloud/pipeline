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

package backupservice

import (
	"net/http"
	"time"

	"emperror.dev/emperror"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/banzaicloud/pipeline/internal/ark/api"
	"github.com/banzaicloud/pipeline/internal/platform/gin/correlationid"
	"github.com/banzaicloud/pipeline/pkg/providers"
	"github.com/banzaicloud/pipeline/src/api/ark/common"
	"github.com/banzaicloud/pipeline/src/auth"
)

// Enable create an ARK service deployment and adding a base scheduled full backup
func Enable(c *gin.Context) {
	logger := correlationid.Logger(common.Log, c)
	logger.Debug("deploying backup service to cluster")

	svc := common.GetARKService(c.Request)

	var request api.EnableBackupServiceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		err = emperror.Wrap(err, "could not parse request")
		common.ErrorHandler.Handle(err)
		common.ErrorResponse(c, err)
		return
	}

	scheduleTTL, err := time.ParseDuration(request.TTL)
	if err != nil {
		err = emperror.Wrap(err, "could not parse request")
		common.ErrorHandler.Handle(err)
		common.ErrorResponse(c, err)
		return
	}

	_, err = svc.GetDeploymentsService().GetActiveDeployment()
	if err == nil {
		err = errors.New("backup service already deployed")
		common.ErrorHandler.Handle(err)
		common.ErrorResponse(c, err)
		return
	}

	if len(request.Location) == 0 && (request.Cloud == providers.Alibaba || request.Cloud == providers.Amazon) {
		// location field is empty in request, get bucket location
		organizationID := auth.GetCurrentOrganization(c.Request).ID

		location, err := common.GetBucketLocation(request.Cloud, request.BucketName, request.SecretID, organizationID, logger)
		if err != nil {
			err = emperror.WrapWith(err, "failed to get bucket region", "bucket", request.BucketName)
			common.ErrorHandler.Handle(err)
			common.ErrorResponse(c, err)
			return
		}

		request.Location = location
	}

	bucketService := svc.GetBucketsService()
	bucket, err := bucketService.FindOrCreateBucket(&api.CreateBucketRequest{
		Cloud:      request.Cloud,
		BucketName: request.BucketName,
		Location:   request.Location,
		SecretID:   request.SecretID,
		AzureBucketProperties: api.AzureBucketProperties{
			StorageAccount: request.StorageAccount,
			ResourceGroup:  request.ResourceGroup,
		},
	})
	if err != nil {
		err = emperror.Wrap(err, "could not persist bucket")
		common.ErrorHandler.Handle(err)
		common.ErrorResponse(c, err)
		return
	}

	err = bucketService.IsBucketInUse(bucket)
	if err != nil {
		common.ErrorHandler.Handle(err)
		common.ErrorResponse(c, err)
		return
	}

	err = svc.GetDeploymentsService().Deploy(bucket, false)
	if err != nil {
		err = emperror.Wrap(err, "could not deploy backup service")
		common.ErrorHandler.Handle(err)
		common.ErrorResponse(c, err)
		return
	}

	spec := &api.CreateBackupRequest{
		Name:   api.BaseScheduleName,
		Labels: request.Labels,
		TTL: metav1.Duration{
			Duration: scheduleTTL,
		},
	}

	if spec.Labels == nil {
		spec.Labels = make(labels.Set, 0)
	}
	spec.Labels[api.LabelKeyDistribution] = string(svc.GetDeploymentsService().GetCluster().GetDistribution())
	spec.Labels[api.LabelKeyCloud] = svc.GetDeploymentsService().GetCluster().GetCloud()

	err = svc.GetSchedulesService().Create(spec, request.Schedule)
	if err != nil {
		err = emperror.Wrap(err, "could not create schedule")
		common.ErrorHandler.Handle(err)
		common.ErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, api.EnableBackupServiceResponse{
		Status: http.StatusOK,
	})
}
