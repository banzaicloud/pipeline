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

// RegisterNodePoolHTTPHandlers mounts all of the service endpoints into an http.Handler.
func RegisterNodePoolHTTPHandlers(endpoints NodePoolEndpoints, router *mux.Router, options ...kithttp.ServerOption) {
	errorEncoder := kitxhttp.NewJSONProblemErrorResponseEncoder(apphttp.NewDefaultProblemConverter())

	router.Methods(http.MethodPost).Path("").Handler(kithttp.NewServer(
		endpoints.CreateNodePool,
		decodeCreateNodePoolHTTPRequest,
		kitxhttp.ErrorResponseEncoder(kitxhttp.StatusCodeResponseEncoder(http.StatusAccepted), errorEncoder),
		options...,
	))

	router.Methods(http.MethodDelete).Path("/{nodePoolName}").Handler(kithttp.NewServer(
		endpoints.DeleteNodePool,
		decodeDeleteNodePoolHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeDeleteNodePoolHTTPResponse, errorEncoder),
		options...,
	))
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
