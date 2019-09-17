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

package issue

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/common/commonadapter"
)

//go:generate sh -c "test -x ${MOCKERY} && ${MOCKERY} -name UserExtractor -inpkg -testonly"
//go:generate sh -c "test -x ${MOCKERY} && ${MOCKERY} -name Formatter -inpkg -testonly"
//go:generate sh -c "test -x ${MOCKERY} && ${MOCKERY} -name Reporter -inpkg -testonly"

func TestService_ReportIssue(t *testing.T) {
	ctx := context.Background()

	userExtractor := new(MockUserExtractor)
	userExtractor.On("GetUserID", ctx).Return(uint(1), true)

	newIssue := NewIssue{
		OrganizationName: "example",
		Title:            "Something went wrong",
		Text:             "Here is my detailed issue",
		Labels:           []string{"bug"},
	}

	data := NewIssueData{
		Title:            newIssue.Title,
		Text:             newIssue.Text,
		OrganizationName: newIssue.OrganizationName,
		UserID:           1,
		Labels:           newIssue.Labels,
	}

	formatter := new(MockFormatter)
	formatter.On("FormatIssue", data).Return("Here is my detailed issue", nil)

	issue := Issue{
		Title:  "Something went wrong",
		Body:   "Here is my detailed issue",
		Labels: []string{"bug"},
	}

	reporter := new(MockReporter)
	reporter.On("ReportIssue", ctx, issue).Return(nil)

	service := NewService(userExtractor, formatter, reporter, commonadapter.NewNoopLogger())

	err := service.ReportIssue(ctx, newIssue)
	require.NoError(t, err)

	userExtractor.AssertExpectations(t)
	formatter.AssertExpectations(t)
	reporter.AssertExpectations(t)
}
