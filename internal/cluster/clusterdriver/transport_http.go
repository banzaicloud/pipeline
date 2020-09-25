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
	"github.com/banzaicloud/pipeline/internal/cluster"
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

	router.Methods(http.MethodPut).Path("/update").Handler(kithttp.NewServer(
		endpoints.UpdateCluster,
		decodeUpdateClusterHTTPRequest,
		kitxhttp.ErrorResponseEncoder(kitxhttp.StatusCodeResponseEncoder(http.StatusAccepted), errorEncoder),
		options...,
	))

	router.Methods(http.MethodGet).Path("/nodepools").Handler(kithttp.NewServer(
		endpoints.ListNodePools,
		decodeListNodePoolsHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeListNodePoolsHTTPResponse, errorEncoder),
		options...,
	))

	router.Methods(http.MethodPost).Path("/nodepools").Handler(kithttp.NewServer(
		endpoints.CreateNodePool,
		decodeCreateNodePoolHTTPRequest,
		kitxhttp.ErrorResponseEncoder(kitxhttp.StatusCodeResponseEncoder(http.StatusAccepted), errorEncoder),
		options...,
	))

	router.Methods(http.MethodPost).Path("/nodepools/{nodePoolName}/update").Handler(kithttp.NewServer(
		endpoints.UpdateNodePool,
		decodeUpdateNodePoolHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeUpdateNodePoolHTTPResponse, errorEncoder),
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

		return DeleteClusterRequest{
			ClusterIdentifier: cluster.Identifier{
				ClusterID: clusterID,
			},
			Options: cluster.DeleteClusterOptions{
				Force: force,
			},
		}, nil

	case "name":
		orgID, err := getOrgID(r)
		if err != nil {
			return nil, err
		}

		clusterName := mux.Vars(r)["clusterId"]

		return DeleteClusterRequest{
			ClusterIdentifier: cluster.Identifier{
				OrganizationID: orgID,
				ClusterName:    clusterName,
			},
			Options: cluster.DeleteClusterOptions{
				Force: force,
			},
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

func encodeDeleteClusterHTTPResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	resp, ok := response.(DeleteClusterResponse)
	if !ok || resp.Deleted {
		w.WriteHeader(http.StatusNoContent)

		return nil
	}

	w.WriteHeader(http.StatusAccepted)

	return nil
}

func decodeListNodePoolsHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)

	rawClusterID, ok := vars["clusterId"]
	if !ok || rawClusterID == "" {
		return nil, errors.NewWithDetails("missing parameter from the URL", "param", "clusterId")
	}

	clusterID, err := strconv.ParseUint(rawClusterID, 10, 32)
	if err != nil {
		return nil, errors.NewWithDetails("invalid cluster ID", "rawClusterId", rawClusterID)
	}

	return ListNodePoolsRequest{ClusterID: uint(clusterID)}, nil
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

	return CreateNodePoolRequest{ClusterID: uint(clusterID), RawNodePool: spec}, nil
}

func decodeUpdateNodePoolHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
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

	var rawNodePoolUpdate pipeline.UpdateNodePoolRequest

	err = json.NewDecoder(r.Body).Decode(&rawNodePoolUpdate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode request")
	}

	var update map[string]interface{}

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &update,
		TagName:  "json",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create decoder")
	}

	err = decoder.Decode(rawNodePoolUpdate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode request")
	}

	return UpdateNodePoolRequest{
		ClusterID:         uint(clusterID),
		NodePoolName:      nodePoolName,
		RawNodePoolUpdate: update,
	}, nil
}

func encodeUpdateNodePoolHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(UpdateNodePoolResponse)

	apiResp := pipeline.UpdateNodePoolResponse{
		ProcessId: resp.ProcessID,
	}

	return kitxhttp.JSONResponseEncoder(ctx, w, kitxhttp.WithStatusCode(apiResp, http.StatusAccepted))
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

	return DeleteNodePoolRequest{ClusterID: uint(clusterID), Name: nodePoolName}, nil
}

func encodeDeleteNodePoolHTTPResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	resp, ok := response.(DeleteNodePoolResponse)
	if !ok || resp.Deleted {
		w.WriteHeader(http.StatusNoContent)

		return nil
	}

	w.WriteHeader(http.StatusAccepted)

	return nil
}

func encodeListNodePoolsHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	resp := response.(ListNodePoolsResponse)

	var apiResp []pipeline.NodePoolSummary
	for _, nodePool := range resp.NodePoolList {
		var result pipeline.NodePoolSummary
		err := mapstructure.Decode(nodePool, &result)
		if err != nil {
			return errors.Wrap(err, "failed to decode service response")
		}

		apiResp = append(apiResp, result)
	}

	return kitxhttp.JSONResponseEncoder(ctx, w, kitxhttp.WithStatusCode(apiResp, http.StatusOK))
}

func decodeUpdateClusterHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	clusterID, err := getClusterID(r)
	if err != nil {
		return nil, err
	}

	var rawClusterUpdate pipeline.UpdateClusterRequest

	err = json.NewDecoder(r.Body).Decode(&rawClusterUpdate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode request")
	}

	var update cluster.ClusterUpdate

	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &update,
		TagName:  "json",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create decoder")
	}

	err = decoder.Decode(rawClusterUpdate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode request")
	}

	return UpdateClusterRequest{
		ClusterIdentifier: cluster.Identifier{
			ClusterID: clusterID,
		},
		ClusterUpdate: update,
	}, nil
}
