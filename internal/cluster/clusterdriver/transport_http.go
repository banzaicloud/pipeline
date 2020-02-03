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

package clusterdriver

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"emperror.dev/errors"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/mitchellh/mapstructure"
	kitxhttp "github.com/sagikazarmark/kitx/transport/http"

	"github.com/banzaicloud/pipeline/.gen/pipeline/pipeline"
	apphttp "github.com/banzaicloud/pipeline/internal/platform/appkit/transport/http"
)

// RegisterHTTPHandlers mounts all of the service endpoints into an http.Handler
func RegisterHTTPHandlers(endpoints Endpoints, router *mux.Router, options ...kithttp.ServerOption) {
	errorEncoder := kitxhttp.NewJSONProblemErrorResponseEncoder(apphttp.NewDefaultProblemConverter())

	router.Methods(http.MethodDelete).Path("").Handler(kithttp.NewServer(
		endpoints.DeleteCluster,
		decodeDeleteClusterHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeDeleteClusterHTTPResponse, errorEncoder),
		options...,
	))

	router.Methods(http.MethodPost).Path("/nodepools").Handler(kithttp.NewServer(
		endpoints.CreateNodePool,
		decodeCreateNodePoolHTTPRequest,
		kitxhttp.ErrorResponseEncoder(kitxhttp.StatusCodeResponseEncoder(http.StatusAccepted), errorEncoder),
		options...,
	))

	router.Methods(http.MethodDelete).Path("/nodepools/{nodePoolName}").Handler(kithttp.NewServer(
		endpoints.DeleteNodePool,
		decodeDeleteNodePoolHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeDeleteNodePoolHTTPResponse, errorEncoder),
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

func decodeCreateNodePoolHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)

	rawClusterID, ok := vars["clusterId"]
	if !ok || rawClusterID == "" {
		return nil, errors.NewWithDetails("missing parameter from the URL", "param", "clusterId")
	}

	clusterID, err := strconv.ParseUint(rawClusterID, 10, 32)
	if err != nil {
		return nil, errors.NewWithDetails("invalid cluster ID", "rawClusterId", rawClusterID)
	}

	var rawNodePool pipeline.NodePool

	err = json.NewDecoder(r.Body).Decode(&rawNodePool)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode request")
	}

	var spec map[string]interface{}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &spec,
		TagName:  "json",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create decoder")
	}

	err = decoder.Decode(rawNodePool)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode request")
	}

	return createNodePoolRequest{ClusterID: uint(clusterID), Spec: spec}, nil
}

func decodeDeleteNodePoolHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)

	rawClusterID, ok := vars["clusterId"]
	if !ok || rawClusterID == "" {
		return nil, errors.NewWithDetails("missing parameter from the URL", "param", "clusterId")
	}

	clusterID, err := strconv.ParseUint(rawClusterID, 10, 32)
	if err != nil {
		return nil, errors.NewWithDetails("invalid cluster ID", "rawClusterId", rawClusterID)
	}

	nodePoolName, ok := vars["nodePoolName"]
	if !ok || nodePoolName == "" {
		return nil, errors.NewWithDetails("missing parameter from the URL", "param", "nodePoolName")
	}

	return deleteNodePoolRequest{ClusterID: uint(clusterID), NodePoolName: nodePoolName}, nil
}

func encodeDeleteNodePoolHTTPResponse(_ context.Context, w http.ResponseWriter, resp interface{}) error {
	deleted, ok := resp.(bool)
	if !ok || deleted {
		w.WriteHeader(http.StatusNoContent)

		return nil
	}

	w.WriteHeader(http.StatusAccepted)

	return nil
}
