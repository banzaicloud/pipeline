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

	"github.com/go-kit/kit/endpoint"
	kitoc "github.com/go-kit/kit/tracing/opencensus"

	"github.com/banzaicloud/pipeline/internal/app/frontend/issue"
)

// Endpoints collects all of the endpoints that compose an issue service. It's
// meant to be used as a helper struct, to collect all of the endpoints into a
// single parameter.
type Endpoints struct {
	ReportIssue endpoint.Endpoint
}

// MakeEndpoints returns an Endpoints struct where each endpoint invokes
// the corresponding method on the provided service.
func MakeEndpoints(s issue.Service) Endpoints {
	return Endpoints{
		ReportIssue: kitoc.TraceEndpoint("issue.ReportIssue")(MakeReportIssueEndpoint(s)),
	}
}

// MakeReportIssueEndpoint returns an endpoint for the matching method of the underlying service.
func MakeReportIssueEndpoint(s issue.Service) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, s.ReportIssue(ctx, req.(issue.NewIssue))
	}
}
