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
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/moogar0880/problems"

	"github.com/banzaicloud/pipeline/client"
	"github.com/banzaicloud/pipeline/pkg/ctxutil"
)

// Endpoints collects all of the HTTP handlers that compose the cluster feature service.
// It's meant to be used as a helper struct, to collect all of the handlers into a
// single parameter.
type HTTPHandlers struct {
	List       http.Handler
	Details    http.Handler
	Activate   http.Handler
	Deactivate http.Handler
	Update     http.Handler
}

// MakeHTTPHandlers returns an HTTP Handlers struct where each handler invokes
// the corresponding method on the provided service.
func MakeHTTPHandlers(endpoints Endpoints, errorHandler emperror.Handler) HTTPHandlers {
	errorEncoder := httptransport.ServerErrorEncoder(func(ctx context.Context, err error, w http.ResponseWriter) {
		errorHandler.Handle(err)

		if err := encodeHTTPError(err, w); err != nil {
			errorHandler.Handle(err)
		}
	})

	return HTTPHandlers{
		List: httptransport.NewServer(
			endpoints.List,
			decodeListClusterFeaturesRequest,
			encodeListClusterFeaturesResponse,
			errorEncoder,
		),
		Details: httptransport.NewServer(
			endpoints.Details,
			decodeClusterFeatureDetailsRequest,
			encodeClusterFeatureDetailsResponse,
			errorEncoder,
		),
		Activate: httptransport.NewServer(
			endpoints.Activate,
			decodeActivateClusterFeatureRequest,
			encodeActivateClusterFeatureResponse,
			errorEncoder,
		),
		Deactivate: httptransport.NewServer(
			endpoints.Deactivate,
			decodeDeactivateClusterFeatureRequest,
			encodeDeactivateClusterFeatureResponse,
			errorEncoder,
		),
		Update: httptransport.NewServer(
			endpoints.Update,
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
	case isNotFound(err):
		problem = problems.NewDetailedProblem(http.StatusNotFound, err.Error())

	default:
		problem = problems.NewDetailedProblem(http.StatusInternalServerError, "something went wrong")
	}

	w.Header().Set("Content-Type", problems.ProblemMediaType)

	err = json.NewEncoder(w).Encode(problem)

	return errors.WrapIfWithDetails(err, "failed to encode error response", "error", problem.Detail)
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

	var requestBody client.ActivateClusterFeatureRequest
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

	var requestBody client.UpdateClusterFeatureRequest
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
	badRequest := false
	errors.UnwrapEach(err, func(err error) bool {
		if brErr, ok := err.(interface{ BadRequest() bool }); ok {
			badRequest = brErr.BadRequest()
			return !badRequest
		}
		return true
	})

	return badRequest
}

func isNotFound(err error) bool {
	notFound := false
	errors.UnwrapEach(err, func(err error) bool {
		if brErr, ok := err.(interface{ NotFound() bool }); ok {
			notFound = brErr.NotFound()
			return !notFound
		}
		return true
	})

	return notFound
}
