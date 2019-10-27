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

package clusterfeaturedriver

import (
	"context"
	"encoding/json"
	"net/http"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/internal/clusterfeature"
	"github.com/banzaicloud/pipeline/pkg/ctxutil"
	"github.com/banzaicloud/pipeline/pkg/problems"
)

// RegisterHTTPHandlers mounts all of the service endpoints into an http.Handler.
func RegisterHTTPHandlers(endpoints Endpoints, router *mux.Router, errorHandler emperror.Handler, options ...kithttp.ServerOption) {
	options = append(
		options,
		kithttp.ServerErrorEncoder(encodeHTTPError),
		kithttp.ServerErrorHandler(emperror.MakeContextAware(errorFilter(errorHandler))),
	)

	router.Methods(http.MethodGet).Path("").Handler(kithttp.NewServer(
		endpoints.List,
		decodeListClusterFeaturesRequest,
		encodeListClusterFeaturesResponse,
		options...,
	))

	router.Methods(http.MethodGet).Path("/{featureName}").Handler(kithttp.NewServer(
		endpoints.Details,
		decodeClusterFeatureDetailsRequest,
		encodeClusterFeatureDetailsResponse,
		options...,
	))

	router.Methods(http.MethodPost).Path("/{featureName}").Handler(kithttp.NewServer(
		endpoints.Activate,
		decodeActivateClusterFeatureRequest,
		encodeActivateClusterFeatureResponse,
		options...,
	))

	router.Methods(http.MethodPut).Path("/{featureName}").Handler(kithttp.NewServer(
		endpoints.Update,
		decodeUpdateClusterFeatureRequest,
		encodeUpdateClusterFeatureResponse,
		options...,
	))

	router.Methods(http.MethodDelete).Path("/{featureName}").Handler(kithttp.NewServer(
		endpoints.Deactivate,
		decodeDeactivateClusterFeatureRequest,
		encodeDeactivateClusterFeatureResponse,
		options...,
	))
}

func encodeHTTPError(_ context.Context, err error, w http.ResponseWriter) {
	var problem problems.StatusProblem

	switch {
	case isNotFoundError(err):
		problem = problems.NewDetailedProblem(http.StatusNotFound, err.Error())
	case isBadRequestError(err):
		problem = problems.NewDetailedProblem(http.StatusBadRequest, err.Error())

	default:
		problem = problems.NewDetailedProblem(http.StatusInternalServerError, "something went wrong")
	}

	w.Header().Set("Content-Type", problems.ProblemMediaType)
	w.WriteHeader(problem.ProblemStatus())

	_ = json.NewEncoder(w).Encode(problem)
}

func isNotFoundError(err error) bool {
	var notFound interface{ NotFound() bool }
	return errors.As(err, &clusterfeature.UnknownFeatureError{}) || clusterfeature.IsFeatureNotFoundError(err) || errors.As(err, &notFound)
}

func isBadRequestError(err error) bool {
	var badRequest interface{ BadRequest() bool }
	return clusterfeature.IsInputValidationError(err) || errors.As(err, &clusterfeature.ClusterIsNotReadyError{}) || errors.As(err, &badRequest)
}

func errorFilter(errorHandler emperror.Handler) emperror.Handler {
	return emperror.HandlerFunc(func(err error) {
		switch {
		case isNotFoundError(err), isBadRequestError(err):
			// ignore
		default:
			errorHandler.Handle(err)
		}
	})
}

func decodeListClusterFeaturesRequest(ctx context.Context, _ *http.Request) (interface{}, error) {
	clusterID, ok := ctxutil.ClusterID(ctx)
	if !ok {
		// TODO: better error handling?
		return nil, errors.New("cluster ID not found in the context")
	}

	return ListClusterFeaturesRequest{
		ClusterID: clusterID,
	}, nil
}

func encodeListClusterFeaturesResponse(_ context.Context, w http.ResponseWriter, resp interface{}) error {
	w.Header().Set("Content-Type", "application/json")

	return json.NewEncoder(w).Encode(resp)
}

func decodeClusterFeatureDetailsRequest(ctx context.Context, _ *http.Request) (interface{}, error) {
	clusterID, ok := ctxutil.ClusterID(ctx)
	if !ok {
		// TODO: better error handling?
		return nil, errors.New("cluster ID not found in the context")
	}

	params, _ := ctxutil.Params(ctx)
	featureName := params["featureName"]

	return ClusterFeatureDetailsRequest{
		ClusterID:   clusterID,
		FeatureName: featureName,
	}, nil
}

func encodeClusterFeatureDetailsResponse(_ context.Context, w http.ResponseWriter, resp interface{}) error {
	w.Header().Set("Content-Type", "application/json")

	return json.NewEncoder(w).Encode(resp)
}

func decodeActivateClusterFeatureRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	clusterID, ok := ctxutil.ClusterID(ctx)
	if !ok {
		// TODO: better error handling?
		return nil, errors.New("cluster ID not found in the context")
	}

	params, _ := ctxutil.Params(ctx)
	featureName := params["featureName"]

	var requestBody pipeline.ActivateClusterFeatureRequest
	if err := decodeRequestBody(req, &requestBody); err != nil {
		return nil, err
	}

	return ActivateClusterFeatureRequest{
		ClusterID:   clusterID,
		FeatureName: featureName,
		Spec:        requestBody.Spec,
	}, nil
}

func encodeActivateClusterFeatureResponse(_ context.Context, w http.ResponseWriter, _ interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	return nil
}

func decodeDeactivateClusterFeatureRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	clusterID, ok := ctxutil.ClusterID(ctx)
	if !ok {
		// TODO: better error handling?
		return nil, errors.New("cluster ID not found in the context")
	}

	params, _ := ctxutil.Params(ctx)
	featureName := params["featureName"]

	return DeactivateClusterFeatureRequest{
		ClusterID:   clusterID,
		FeatureName: featureName,
	}, nil
}

func encodeDeactivateClusterFeatureResponse(_ context.Context, w http.ResponseWriter, _ interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)

	return nil
}

func decodeUpdateClusterFeatureRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	clusterID, ok := ctxutil.ClusterID(ctx)
	if !ok {
		// TODO: better error handling?
		return nil, errors.New("cluster ID not found in the context")
	}

	params, _ := ctxutil.Params(ctx)
	featureName := params["featureName"]

	var requestBody pipeline.UpdateClusterFeatureRequest
	if err := decodeRequestBody(req, &requestBody); err != nil {

		return nil, errors.WrapIf(err, "failed to decode request body")
	}

	return UpdateClusterFeatureRequest{
		ClusterID:   clusterID,
		FeatureName: featureName,
		Spec:        requestBody.Spec,
	}, nil
}

func encodeUpdateClusterFeatureResponse(_ context.Context, w http.ResponseWriter, _ interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	return nil
}

func decodeRequestBody(req *http.Request, result interface{}) error {
	if err := json.NewDecoder(req.Body).Decode(result); err != nil {
		return invalidRequestBodyError{errors.WrapIf(err, "failed to decode request body")}
	}
	return nil
}

type invalidRequestBodyError struct {
	err error
}

func (invalidRequestBodyError) Error() string    { return "invalid request body" }
func (e invalidRequestBodyError) Cause() error   { return e.err }
func (e invalidRequestBodyError) Unwrap() error  { return e.err }
func (invalidRequestBodyError) BadRequest() bool { return true }
