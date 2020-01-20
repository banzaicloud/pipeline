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

package clusterdriver

import (
	"context"
	"net/http"
	"strconv"

	"emperror.dev/errors"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	kitxhttp "github.com/sagikazarmark/kitx/transport/http"

	apphttp "github.com/banzaicloud/pipeline/internal/platform/appkit/transport/http"
)

// RegisterClusterHTTPHandlers mounts all of the service endpoints into an http.Handler
func RegisterClusterHTTPHandlers(endpoints ClusterEndpoints, router *mux.Router, options ...kithttp.ServerOption) {
	errorEncoder := kitxhttp.NewJSONProblemErrorResponseEncoder(apphttp.NewDefaultProblemConverter())

	router.Methods(http.MethodDelete).Path("").Handler(kithttp.NewServer(
		endpoints.DeleteCluster,
		decodeDeleteClusterHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeDeleteClusterHTTPResponse, errorEncoder),
		options...,
	))
}

func decodeDeleteClusterHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	force := r.URL.Query().Get("force") == "true"

	switch field := r.URL.Query().Get("field"); field {
	case "id", "":
		clusterID, err := getClusterID(r)
		if err != nil {
			return nil, err
		}

		return deleteClusterRequest{
			ClusterID: clusterID,
			Force:     force,
		}, nil

	case "name":
		orgID, err := getOrgID(r)
		if err != nil {
			return nil, err
		}

		clusterName := mux.Vars(r)["clusterId"]

		return deleteClusterRequest{
			OrganizationID: orgID,
			ClusterName:    clusterName,
			Force:          force,
		}, nil

	default:
		return nil, errors.Errorf("field=%s is not supported", field)
	}
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

func getOrgID(req *http.Request) (uint, error) {
	vars := mux.Vars(req)

	orgIDStr, ok := vars["orgId"]
	if !ok {
		return 0, errors.New("organization ID not found in path variables")
	}

	orgID, err := strconv.ParseUint(orgIDStr, 0, 0)
	return uint(orgID), errors.WrapIf(err, "invalid organization ID format")
}

func encodeDeleteClusterHTTPResponse(_ context.Context, w http.ResponseWriter, resp interface{}) error {
	deleted, ok := resp.(bool)
	if !ok || deleted {
		w.WriteHeader(http.StatusNoContent)

		return nil
	}

	w.WriteHeader(http.StatusAccepted)

	return nil
}
