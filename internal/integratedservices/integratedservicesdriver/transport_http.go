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

package integratedservicesdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"emperror.dev/errors"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	kitxhttp "github.com/sagikazarmark/kitx/transport/http"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	apphttp "github.com/banzaicloud/pipeline/internal/platform/appkit/transport/http"
)

const integratedServiceNameParamKey = "serviceName"

// RegisterHTTPHandlers mounts all of the service endpoints into an http.Handler.
func RegisterHTTPHandlers(endpoints Endpoints, router *mux.Router, options ...kithttp.ServerOption) {
	errorEncoder := kitxhttp.NewJSONProblemErrorResponseEncoder(apphttp.NewDefaultProblemConverter())

	router.Methods(http.MethodGet).Path("").Handler(kithttp.NewServer(
		endpoints.List,
		decodeListIntegratedServicesRequest,
		kitxhttp.ErrorResponseEncoder(encodeListIntegratedServicesResponse, errorEncoder),
		options...,
	))

	{
		router := router.Path(fmt.Sprintf("/{%s}", integratedServiceNameParamKey)).Subrouter()

		router.Methods(http.MethodGet).Handler(kithttp.NewServer(
			endpoints.Details,
			decodeIntegratedServiceDetailsRequest,
			kitxhttp.ErrorResponseEncoder(encodeIntegratedServiceDetailsResponse, errorEncoder),
			options...,
		))

		router.Methods(http.MethodPost).Handler(kithttp.NewServer(
			endpoints.Activate,
			decodeActivateIntegratedServiceRequest,
			kitxhttp.ErrorResponseEncoder(encodeActivateIntegratedServicesResponse, errorEncoder),
			options...,
		))

		router.Methods(http.MethodPut).Handler(kithttp.NewServer(
			endpoints.Update,
			decodeUpdateIntegratedServicesRequest,
			kitxhttp.ErrorResponseEncoder(encodeUpdateIntegratedServiceResponse, errorEncoder),
			options...,
		))

		router.Methods(http.MethodDelete).Handler(kithttp.NewServer(
			endpoints.Deactivate,
			decodeDeactivateIntegratedServicesRequest,
			kitxhttp.ErrorResponseEncoder(encodeDeactivateIntegratedServiceResponse, errorEncoder),
			options...,
		))
	}
}

func decodeListIntegratedServicesRequest(_ context.Context, req *http.Request) (interface{}, error) {
	clusterID, err := getClusterID(req)
	if err != nil {
		return nil, err
	}

	return ListRequest{
		ClusterID: clusterID,
	}, nil
}

func encodeListIntegratedServicesResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(ListResponse)

	integratedServiceDetails := make(map[string]pipeline.IntegratedServiceDetails, len(resp.Services))

	for _, s := range resp.Services {
		integratedServiceDetails[s.Name] = pipeline.IntegratedServiceDetails{
			Spec:   s.Spec,
			Output: s.Output,
			Status: s.Status,
		}
	}

	w.Header().Set("Content-Type", "application/json")

	return json.NewEncoder(w).Encode(integratedServiceDetails)
}

func decodeIntegratedServiceDetailsRequest(_ context.Context, req *http.Request) (interface{}, error) {
	clusterID, err := getClusterID(req)
	if err != nil {
		return nil, err
	}

	serviceName, err := getServiceName(req)
	if err != nil {
		return nil, err
	}

	return DetailsRequest{
		ClusterID:   clusterID,
		ServiceName: serviceName,
	}, nil
}

func encodeIntegratedServiceDetailsResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(DetailsResponse)

	service := pipeline.IntegratedServiceDetails{
		Spec:   resp.Service.Spec,
		Output: resp.Service.Output,
		Status: resp.Service.Status,
	}

	w.Header().Set("Content-Type", "application/json")

	return json.NewEncoder(w).Encode(service)
}

func decodeActivateIntegratedServiceRequest(_ context.Context, req *http.Request) (interface{}, error) {
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

	return ActivateRequest{
		ClusterID:   clusterID,
		ServiceName: serviceName,
		Spec:        requestBody.Spec,
	}, nil
}

func encodeActivateIntegratedServicesResponse(_ context.Context, w http.ResponseWriter, _ interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

	return nil
}

func decodeDeactivateIntegratedServicesRequest(_ context.Context, req *http.Request) (interface{}, error) {
	clusterID, err := getClusterID(req)
	if err != nil {
		return nil, err
	}

	serviceName, err := getServiceName(req)
	if err != nil {
		return nil, err
	}

	return DeactivateRequest{
		ClusterID:   clusterID,
		ServiceName: serviceName,
	}, nil
}

func encodeDeactivateIntegratedServiceResponse(_ context.Context, w http.ResponseWriter, _ interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)

	return nil
}

func decodeUpdateIntegratedServicesRequest(_ context.Context, req *http.Request) (interface{}, error) {
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

	return UpdateRequest{
		ClusterID:   clusterID,
		ServiceName: serviceName,
		Spec:        requestBody.Spec,
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
