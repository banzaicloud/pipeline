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
	"fmt"
)

const markdownIssueTemplate = `
**Version Information:**

| Version | Commit Hash | Build Date |
| ------- |:-----------:| ----------:|
| %s | [link](https://github.com/banzaicloud/pipeline/commit/%s) | %s |

**User Information:**

| User ID | Organization|
| ------- |:-----------:|
| %d      | %s          |

**Description:**
%s`

// VersionInformation contains version information about the current Pipeline version.
type VersionInformation struct {
	Version    string
	CommitHash string
	BuildDate  string
}

// MarkdownFormatter formats an issue into a simple markdown document with tables.
type MarkdownFormatter struct {
	version VersionInformation
}

// NewMarkdownFormatter returns a new MarkdownFormatter.
func NewMarkdownFormatter(version VersionInformation) MarkdownFormatter {
	return MarkdownFormatter{
		version: version,
	}
}

// FormatIssuer returns a formatted issue.
func (m MarkdownFormatter) FormatIssue(data NewIssueData) (string, error) {
	return fmt.Sprintf(
		markdownIssueTemplate,
		m.version.Version,
		m.version.CommitHash,
		m.version.BuildDate,
		data.UserID,
		data.OrganizationName,
		data.Text,
	), nil
}
