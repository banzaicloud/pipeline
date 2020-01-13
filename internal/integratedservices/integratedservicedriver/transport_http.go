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

package integratedservicedriver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/pkg/problems"
)

const integratedServiceNameParamKey = "serviceName"

// RegisterHTTPHandlers mounts all of the service endpoints into an http.Handler.
func RegisterHTTPHandlers(endpoints Endpoints, router *mux.Router, errorHandler emperror.Handler, options ...kithttp.ServerOption) {
	options = append(
		options,
		kithttp.ServerErrorEncoder(encodeHTTPError),
		kithttp.ServerErrorHandler(emperror.MakeContextAware(errorFilter(errorHandler))),
	)

	router.Methods(http.MethodGet).Path("").Handler(kithttp.NewServer(
		endpoints.List,
		decodeListIntegratedServicesRequest,
		encodeListIntegratedServicesResponse,
		options...,
	))

	{
		router := router.Path(fmt.Sprintf("/{%s}", integratedServiceNameParamKey)).Subrouter()

		router.Methods(http.MethodGet).Handler(kithttp.NewServer(
			endpoints.Details,
			decodeIntegratedServiceDetailsRequest,
			encodeIntegratedServiceDetailsResponse,
			options...,
		))

		router.Methods(http.MethodPost).Handler(kithttp.NewServer(
			endpoints.Activate,
			decodeActivateIntegratedServiceRequest,
			encodeActivateIntegratedServicesResponse,
			options...,
		))

		router.Methods(http.MethodPut).Handler(kithttp.NewServer(
			endpoints.Update,
			decodeUpdateIntegratedServicesRequest,
			encodeUpdateIntegratedServiceResponse,
			options...,
		))

		router.Methods(http.MethodDelete).Handler(kithttp.NewServer(
			endpoints.Deactivate,
			decodeDeactivateIntegratedServicesRequest,
			encodeDeactivateIntegratedServiceResponse,
			options...,
		))
	}
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
	return errors.As(err, &integratedservices.UnknownIntegratedServiceError{}) || integratedservices.IsIntegratedServiceNotFoundError(err) || errors.As(err, &notFound)
}

func isBadRequestError(err error) bool {
	var badRequest interface{ BadRequest() bool }
	return integratedservices.IsInputValidationError(err) || errors.As(err, &integratedservices.ClusterIsNotReadyError{}) || errors.As(err, &badRequest)
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

func decodeListIntegratedServicesRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	clusterID, err := getClusterID(req)
	if err != nil {
		return nil, err
	}

	return ListIntegratedServicesRequest{
		ClusterID: clusterID,
	}, nil
}

func encodeListIntegratedServicesResponse(_ context.Context, w http.ResponseWriter, resp interface{}) error {
	w.Header().Set("Content-Type", "application/json")

	return json.NewEncoder(w).Encode(resp)
}

func decodeIntegratedServiceDetailsRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	clusterID, err := getClusterID(req)
	if err != nil {
		return nil, err
	}

	serviceName, err := getServiceName(req)
	if err != nil {
		return nil, err
	}

	return IntegratedServiceDetailsRequest{
		ClusterID:             clusterID,
		IntegratedServiceName: serviceName,
	}, nil
}

func encodeIntegratedServiceDetailsResponse(_ context.Context, w http.ResponseWriter, resp interface{}) error {
	w.Header().Set("Content-Type", "application/json")

	return json.NewEncoder(w).Encode(resp)
}

func decodeActivateIntegratedServiceRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	clusterID, err := getClusterID(req)
	if err != nil {
		return nil, err
	}

	serviceName, err := getServiceName(req)
	if err != nil {
		return nil, err
	}

	var requestBody pipeline.ActivateIntegratedServiceRequest
	if err := decodeRequestBody(req, &requestBody); err != nil {
		return nil, err
	}

	return ActivateIntegratedServiceRequest{
		ClusterID:             clusterID,
		IntegratedServiceName: serviceName,
		Spec:                  requestBody.Spec,
	}, nil
}

func encodeActivateIntegratedServicesResponse(_ context.Context, w http.ResponseWriter, _ interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	return nil
}

func decodeDeactivateIntegratedServicesRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	clusterID, err := getClusterID(req)
	if err != nil {
		return nil, err
	}

	serviceName, err := getServiceName(req)
	if err != nil {
		return nil, err
	}

	return DeactivateIntegratedServiceRequest{
		ClusterID:             clusterID,
		IntegratedServiceName: serviceName,
	}, nil
}

func encodeDeactivateIntegratedServiceResponse(_ context.Context, w http.ResponseWriter, _ interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)

	return nil
}

func decodeUpdateIntegratedServicesRequest(ctx context.Context, req *http.Request) (interface{}, error) {
	clusterID, err := getClusterID(req)
	if err != nil {
		return nil, err
	}

	serviceName, err := getServiceName(req)
	if err != nil {
		return nil, err
	}

	var requestBody pipeline.UpdateIntegratedServiceRequest
	if err := decodeRequestBody(req, &requestBody); err != nil {

		return nil, errors.WrapIf(err, "failed to decode request body")
	}

	return UpdateIntegratedServiceRequest{
		ClusterID:             clusterID,
		IntegratedServiceName: serviceName,
		Spec:                  requestBody.Spec,
	}, nil
}

func encodeUpdateIntegratedServiceResponse(_ context.Context, w http.ResponseWriter, _ interface{}) error {
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

func getClusterID(req *http.Request) (uint, error) {
	vars := mux.Vars(req)

	clusterIDStr, ok := vars["clusterId"]
	if !ok {
		return 0, errors.New("cluster ID not found in path variables")
	}

	clusterID, err := strconv.ParseUint(clusterIDStr, 0, 0)
	return uint(clusterID), errors.WrapIf(err, "invalid cluster ID format")
}

func getServiceName(req *http.Request) (string, error) {
	vars := mux.Vars(req)

	serviceName, ok := vars[integratedServiceNameParamKey]
	if !ok {
		return "", errors.New("integrated service name not found in path variables")
	}

	if serviceName == "" {
		return "", errors.New("integrated service name must not be empty")
	}

	return serviceName, nil
}

type invalidRequestBodyError struct {
	err error
}

func (invalidRequestBodyError) Error() string    { return "invalid request body" }
func (e invalidRequestBodyError) Cause() error   { return e.err }
func (e invalidRequestBodyError) Unwrap() error  { return e.err }
func (invalidRequestBodyError) BadRequest() bool { return true }
