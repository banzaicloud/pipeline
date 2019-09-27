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

package issuedriver

import (
	"context"
	"encoding/json"
	"net/http"

	"emperror.dev/errors"
	"github.com/gorilla/mux"
	kitxhttp "github.com/sagikazarmark/kitx/transport/http"

	"github.com/banzaicloud/pipeline/internal/app/frontend/issue"
)

// RegisterHTTPHandlers mounts all of the service endpoints into an http.Handler.
func RegisterHTTPHandlers(endpoints Endpoints, router *mux.Router, factory kitxhttp.ServerFactory) {
	router.Methods(http.MethodPost).Path("").Handler(factory.NewServer(
		endpoints.ReportIssue,
		decodeReportIssueHTTPRequest,
		kitxhttp.StatusCodeResponseEncoder(http.StatusCreated),
	))
}

func decodeReportIssueHTTPRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var newIssue issue.NewIssue

	err := json.NewDecoder(r.Body).Decode(&newIssue)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode request")
	}

	return newIssue, nil
}
