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

package features

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/goph/emperror"
	"github.com/moogar0880/problems"

	"github.com/banzaicloud/pipeline/client"
	ginutils "github.com/banzaicloud/pipeline/internal/platform/gin/utils"
	"github.com/banzaicloud/pipeline/pkg/errors"
)

type Handlers struct {
	ListClusterFeatures http.Handler

	ActivateClusterFeature   http.Handler
	ClusterFeatureDetails    http.Handler
	DeactivateClusterFeature http.Handler
	UpdateClusterFeature     http.Handler
}

func MakeHandlers(endpoints Endpoints, errorHandler emperror.Handler) Handlers {

	errorEncoder := httptransport.ServerErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) {
		errorHandler.Handle(err)

		if err := encodeHTTPError(err, w); err != nil {
			errorHandler.Handle(err)
		}
	})

	return Handlers{
		ListClusterFeatures: httptransport.NewServer(
			endpoints.ListClusterFeatures,
			decodeListClusterFeaturesRequest,
			encodeListClusterFeaturesResponse,
			errorEncoder,
		),
		ActivateClusterFeature: httptransport.NewServer(
			endpoints.ActivateClusterFeature,
			decodeActivateClusterFeatureRequest,
			encodeActivateClusterFeatureResponse,
			errorEncoder,
		),
		ClusterFeatureDetails: httptransport.NewServer(
			endpoints.ClusterFeatureDetails,
			decodeClusterFeatureDetailsRequest,
			encodeClusterFeatureDetailsResponse,
			errorEncoder,
		),
		DeactivateClusterFeature: httptransport.NewServer(
			endpoints.DeactivateClusterFeature,
			decodeDeactivateClusterFeatureRequest,
			encodeDeactivateClusterFeatureResponse,
			errorEncoder,
		),
		UpdateClusterFeature: httptransport.NewServer(
			endpoints.UpdateClusterFeature,
			decodeUpdateClusterFeatureRequest,
			encodeUpdateClusterFeatureResponse,
			errorEncoder,
		),
	}
}

func encodeHTTPError(err error, w http.ResponseWriter) error {
	var problem *problems.DefaultProblem
	switch {
	case isBadRequest(err):
		problem = problems.NewDetailedProblem(http.StatusBadRequest, err.Error())
	default:
		problem = problems.NewDetailedProblem(http.StatusInternalServerError, err.Error())
	}
	w.Header().Set("Content-Type", problems.ProblemMediaType)
	err = json.NewEncoder(w).Encode(problem)
	return emperror.WrapWith(err, "failed to encode error response", "error", problem.Detail)
}

func decodeListClusterFeaturesRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	params := ginutils.GetParams(ctx)

	orgID, err := getOrganizationID(params)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get organization ID")
	}

	clusterID, err := getClusterID(params)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get cluster ID")
	}

	return ListClusterFeaturesRequest{
		OrganizationID: orgID,
		ClusterID:      clusterID,
	}, nil
}

func encodeListClusterFeaturesResponse(ctx context.Context, w http.ResponseWriter, resp interface{}) error {
	return json.NewEncoder(w).Encode(resp)
}

func decodeActivateClusterFeatureRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	params := ginutils.GetParams(ctx)

	orgID, err := getOrganizationID(params)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get organization ID")
	}

	clusterID, err := getClusterID(params)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get cluster ID")
	}

	featureName := getFeatureName(params)

	var requestBody client.ActivateClusterFeatureRequest
	if err := decodeRequestBody(req, &requestBody); err != nil {
		return nil, emperror.Wrap(err, "failed to decode request body")
	}

	return ActivateClusterFeatureRequest{
		OrganizationID: orgID,
		ClusterID:      clusterID,
		FeatureName:    featureName,
		Spec:           requestBody.Spec,
	}, nil
}

func encodeActivateClusterFeatureResponse(ctx context.Context, w http.ResponseWriter, resp interface{}) error {
	return json.NewEncoder(w).Encode(resp)
}

func decodeDeactivateClusterFeatureRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	params := ginutils.GetParams(ctx)

	orgID, err := getOrganizationID(params)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get organization ID")
	}

	clusterID, err := getClusterID(params)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get cluster ID")
	}

	featureName := getFeatureName(params)

	return DeactivateClusterFeatureRequest{
		OrganizationID: orgID,
		ClusterID:      clusterID,
		FeatureName:    featureName,
	}, nil
}

func encodeDeactivateClusterFeatureResponse(ctx context.Context, w http.ResponseWriter, resp interface{}) error {
	return json.NewEncoder(w).Encode(resp)
}

func decodeClusterFeatureDetailsRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	params := ginutils.GetParams(ctx)

	orgID, err := getOrganizationID(params)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get organization ID")
	}

	clusterID, err := getClusterID(params)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get cluster ID")
	}

	featureName := getFeatureName(params)

	return ClusterFeatureDetailsRequest{
		OrganizationID: orgID,
		ClusterID:      clusterID,
		FeatureName:    featureName,
	}, nil
}

func encodeClusterFeatureDetailsResponse(ctx context.Context, w http.ResponseWriter, resp interface{}) error {
	return json.NewEncoder(w).Encode(resp)
}

func decodeUpdateClusterFeatureRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	params := ginutils.GetParams(ctx)

	orgID, err := getOrganizationID(params)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get organization ID")
	}

	clusterID, err := getClusterID(params)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get cluster ID")
	}

	featureName := getFeatureName(params)

	var requestBody client.UpdateClusterFeatureRequest
	if err := decodeRequestBody(req, &requestBody); err != nil {
		return nil, emperror.Wrap(err, "failed to decode request body")
	}

	return UpdateClusterFeatureRequest{
		OrganizationID: orgID,
		ClusterID:      clusterID,
		FeatureName:    featureName,
		Spec:           requestBody.Spec,
	}, nil
}

func encodeUpdateClusterFeatureResponse(ctx context.Context, w http.ResponseWriter, resp interface{}) error {
	return json.NewEncoder(w).Encode(resp)
}

func getOrganizationID(params gin.Params) (uint, error) {
	id, err := parseBase10Uint(params.ByName("orgid"))
	if err != nil {
		err = invalidOrganizationIDError{err}
	}
	return id, err
}

func getClusterID(params gin.Params) (uint, error) {
	id, err := parseBase10Uint(params.ByName("id"))
	if err != nil {
		err = invalidClusterIDError{err}
	}
	return id, err
}

func getFeatureName(params gin.Params) string {
	return params.ByName("featureName")
}

func decodeRequestBody(req *http.Request, result interface{}) error {
	if err := json.NewDecoder(req.Body).Decode(result); err != nil {
		return invalidRequestBodyError{err}
	}
	return nil
}

func parseBase10Uint(value string) (uint, error) {
	res, err := strconv.ParseUint(value, 10, 0)
	return uint(res), err
}

type invalidOrganizationIDError struct {
	cause error
}

func (e invalidOrganizationIDError) Cause() error {
	return e.cause
}

func (e invalidOrganizationIDError) Error() string {
	return "invalid organization ID"
}

func (e invalidOrganizationIDError) BadRequest() bool {
	return true
}

type invalidClusterIDError struct {
	cause error
}

func (e invalidClusterIDError) Cause() error {
	return e.cause
}

func (e invalidClusterIDError) Error() string {
	return "invalid cluster ID"
}

func (e invalidClusterIDError) BadRequest() bool {
	return true
}

type invalidRequestBodyError struct {
	cause error
}

func (e invalidRequestBodyError) Cause() error {
	return e.cause
}

func (e invalidRequestBodyError) Error() string {
	return "invalid request body"
}

func (e invalidRequestBodyError) BadRequest() bool {
	return true
}

func isBadRequest(err error) bool {
	causeIter := errors.CauseIterator{
		Error: err,
	}
	return causeIter.Any(func(err error) bool {
		if brErr, ok := err.(interface{ BadRequest() bool }); ok {
			return brErr.BadRequest()
		}
		return false
	})
}
