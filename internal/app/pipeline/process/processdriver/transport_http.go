// Copyright © 2020 Banzai Cloud
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

package processdriver

import (
	"context"
	"net/http"

	"emperror.dev/errors"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	kitxhttp "github.com/sagikazarmark/kitx/transport/http"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	"github.com/banzaicloud/pipeline/internal/app/pipeline/process"
	apphttp "github.com/banzaicloud/pipeline/internal/platform/appkit/transport/http"
	"github.com/banzaicloud/pipeline/src/auth"
)

// RegisterHTTPHandlers mounts all of the service endpoints into an http.Handler.
func RegisterHTTPHandlers(endpoints WorkflowEndpoints, router *mux.Router, options ...kithttp.ServerOption) {
	errorEncoder := kitxhttp.NewJSONProblemErrorResponseEncoder(apphttp.NewDefaultProblemConverter())

	router.Methods(http.MethodGet).Path("").Handler(kithttp.NewServer(
		endpoints.ListProcesses,
		decodeListProcessesHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeListProcessesHTTPResponse, errorEncoder),
		options...,
	))

	router.Methods(http.MethodGet).Path("/{id}").Handler(kithttp.NewServer(
		endpoints.GetProcess,
		decodeGetProcessHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeGetProcessHTTPResponse, errorEncoder),
		options...,
	))

	router.Methods(http.MethodPost).Path("/{id}/cancel").Handler(kithttp.NewServer(
		endpoints.CancelProcess,
		decodeCancelProcessHTTPRequest,
		kitxhttp.ErrorResponseEncoder(kitxhttp.StatusCodeResponseEncoder(http.StatusAccepted), errorEncoder),
		options...,
	))
}

func encodeListProcessesHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(ListProcessesWorkflowResponse)

	return kitxhttp.JSONResponseEncoder(ctx, w, resp.Processes)
}

func decodeGetProcessHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok || id == "" {
		return nil, errors.NewWithDetails("missing parameter from the URL", "param", "id")
	}

	return GetProcessWorkflowRequest{Id: id}, nil
}

func decodeCancelProcessHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)

	id, ok := vars["id"]
	if !ok || id == "" {
		return nil, errors.NewWithDetails("missing parameter from the URL", "param", "id")
	}

	return CancelProcessWorkflowRequest{Id: id}, nil
}

func encodeGetProcessHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(GetProcessWorkflowResponse)

	return kitxhttp.JSONResponseEncoder(ctx, w, resp.Process)
}

func decodeListProcessesHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	org := auth.GetCurrentOrganization(r)

	query := process.Process{
		OrgId: int32(org.ID),
	}

	values := r.URL.Query()

	if rt := values["type"]; len(rt) > 0 {
		query.Type = rt[0]
	}

	if rt := values["resourceId"]; len(rt) > 0 {
		query.ResourceId = rt[0]
	}

	if rt := values["parentId"]; len(rt) > 0 {
		query.ParentId = rt[0]
	}

	if rt := values["status"]; len(rt) > 0 {
		query.Status = pipeline.ProcessStatus(rt[0])
	}

	return ListProcessesWorkflowRequest{Query: query}, nil
}
