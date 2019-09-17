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

package issueadapter

import (
	"context"

	"emperror.dev/errors"
	"github.com/google/go-github/github"

	"github.com/banzaicloud/pipeline/internal/app/frontend/issue"
)

// GitHubReporter sends issues to a GitHub.
type GitHubReporter struct {
	client *github.Client

	owner      string
	repository string
}

// NewGitHubReporter returns a new GitHubReporter.
func NewGitHubReporter(client *github.Client, account string, repository string) GitHubReporter {
	return GitHubReporter{
		client:     client,
		owner:      account,
		repository: repository,
	}
}

// ReportIssue accepts a new issue and sends it to an external issue tracker service.
func (r GitHubReporter) ReportIssue(ctx context.Context, issue issue.Issue) error {
	req := github.IssueRequest{
		Title:  github.String(issue.Title),
		Body:   github.String(issue.Body),
		Labels: &issue.Labels,
	}

	_, _, err := r.client.Issues.Create(ctx, r.owner, r.repository, &req)

	return errors.Wrap(err, "failed to create github issue")
}
