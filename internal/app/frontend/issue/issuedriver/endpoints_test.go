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
	"testing"

	kitxendpoint "github.com/sagikazarmark/kitx/endpoint"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/app/frontend/issue"
)

func TestMakeEndpoints_ReportIssue(t *testing.T) {
	service := new(issue.MockService)

	newIssue := issue.NewIssue{
		OrganizationName: "example",
		Title:            "Something went wrong",
		Text:             "Here is my detailed issue",
		Labels:           []string{"bug"},
	}

	service.On("ReportIssue", mock.Anything, newIssue).Return(nil)

	e := MakeEndpoints(service, kitxendpoint.NewFactory()).ReportIssue

	_, err := e(context.Background(), newIssue)
	require.NoError(t, err)

	service.AssertExpectations(t)
}
