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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownFormatter_FormatIssue(t *testing.T) {
	const expectedIssueBody = `
**Version Information:**

| Version | Commit Hash | Build Date |
| ------- |:-----------:| ----------:|
| 0.30.0 | [link](https://github.com/banzaicloud/pipeline/commit/54dca25) | 2019-09-16T14:01:10+0000 |

**User Information:**

| User ID | Organization|
| ------- |:-----------:|
| 1      | example          |

**Description:**
Here is my detailed issue`

	formatter := NewMarkdownFormatter(VersionInformation{
		Version:    "0.30.0",
		CommitHash: "54dca25",
		BuildDate:  "2019-09-16T14:01:10+0000",
	})

	body, err := formatter.FormatIssue(NewIssueData{
		Title:            "Something went wrong",
		Text:             "Here is my detailed issue",
		OrganizationName: "example",
		UserID:           1,
		Labels:           []string{"bug"},
	})
	require.NoError(t, err)

	assert.Equal(t, expectedIssueBody, body)
}
