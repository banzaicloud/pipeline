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

package pkeservicedriver

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
	cluster "github.com/banzaicloud/pipeline/internal/cluster"
	apphttp "github.com/banzaicloud/pipeline/internal/platform/appkit/transport/http"
	"github.com/banzaicloud/pipeline/src/api/cluster/pke/pkeservice"
)

// RegisterHTTPHandlers mounts all of the service endpoints into an http.Handler
func RegisterHTTPHandlers(endpoints Endpoints, router *mux.Router, options ...kithttp.ServerOption) {
	errorEncoder := kitxhttp.NewJSONProblemErrorResponseEncoder(apphttp.NewDefaultProblemConverter())

	router.Methods(http.MethodPost).Path("/pke/status").Handler(kithttp.NewServer(
		endpoints.RegisterNodeStatus,
		decodeRegisterNodeStatusHTTPRequest,
		kitxhttp.ErrorResponseEncoder(encodeRegisterNodeStatusHTTPResponse, errorEncoder),
		options...,
	))
}

func decodeRegisterNodeStatusHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	cluster, err := getClusterIentifiers(r)
	if err != nil {
		return nil, err
	}

	var rawStatus pipeline.ReportPkeNodeStatusRequest
	err = json.NewDecoder(r.Body).Decode(&rawStatus)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode request")
	}

	var status pkeservice.NodeStatus
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &status,
		TagName:  "json",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create decoder")
	}

	err = decoder.Decode(rawStatus)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode request")
	}

	return RegisterNodeStatusRequest{
		ClusterIdentifier: cluster,
		NodeStatus:        status,
	}, nil
}

func encodeRegisterNodeStatusHTTPResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	apiResp := response.(RegisterNodeStatusResponse)

	return kitxhttp.JSONResponseEncoder(ctx, w, kitxhttp.WithStatusCode(apiResp.Resp, http.StatusAccepted))
}

func getClusterIentifiers(req *http.Request) (cluster.Identifier, error) {
	vars := mux.Vars(req)

	clusterIDStr, ok := vars["clusterId"]
	if !ok {
		return cluster.Identifier{}, errors.New("cluster ID not found in path variables")
	}

	clusterID, err := strconv.ParseUint(clusterIDStr, 0, 0)
	if err != nil {
		return cluster.Identifier{}, errors.WrapIf(err, "invalid cluster ID format")
	}

	orgIDStr, ok := vars["orgId"]
	if !ok {
		return cluster.Identifier{}, errors.New("org ID not found in path variables")
	}

	orgID, err := strconv.ParseUint(orgIDStr, 0, 0)
	if err != nil {
		return cluster.Identifier{}, errors.WrapIf(err, "invalid org ID format")
	}

	return cluster.Identifier{
		OrganizationID: uint(orgID),
		ClusterID:      uint(clusterID),
		ClusterName:    "",
	}, nil
}
