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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	kitxendpoint "github.com/sagikazarmark/kitx/endpoint"
	kitxhttp "github.com/sagikazarmark/kitx/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/app/frontend/issue"
)

func TestMakeHTTPHandler_ReportIssue(t *testing.T) {
	service := new(issue.MockService)

	newIssue := issue.NewIssue{
		OrganizationName: "example",
		Title:            "Something went wrong",
		Text:             "Here is my detailed issue",
		Labels:           []string{"bug"},
	}

	service.On("ReportIssue", mock.Anything, newIssue).Return(nil)

	handler := mux.NewRouter()
	RegisterHTTPHandlers(
		MakeEndpoints(service, kitxendpoint.NewFactory()),
		handler.PathPrefix("/issues").Subrouter(),
		kitxhttp.NewServerFactory(),
	)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	tsClient := ts.Client()

	body, err := json.Marshal(newIssue)
	require.NoError(t, err)

	resp, err := tsClient.Post(ts.URL+"/issues", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	service.AssertExpectations(t)
}
