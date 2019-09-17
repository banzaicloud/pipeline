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
)

// Issue is reported by a user on the UI.
type Issue struct {
	Title  string   `json:"title"`
	Body   string   `json:"text"`
	Labels []string `json:"labels"`
}

// Service handles reported issues.
type Service interface {
	// ReportIssue accepts a new issue and sends it to an external issue tracker service.
	ReportIssue(ctx context.Context, newIssue NewIssue) error
}

type service struct {
	userExtractor UserExtractor
	formatter     Formatter
	reporter      Reporter

	logger Logger
}

// NewService returns a new Service.
func NewService(
	userExtractor UserExtractor,
	formatter Formatter,
	reporter Reporter,

	logger Logger,
) Service {
	return service{
		userExtractor: userExtractor,
		formatter:     formatter,
		reporter:      reporter,

		logger: logger,
	}
}

// UserExtractor extracts user information from the context.
type UserExtractor interface {
	// GetUserID returns the ID of the currently authenticated user.
	// If a user cannot be found in the context, it returns false as the second return value.
	GetUserID(ctx context.Context) (uint, bool)
}

// Formatter takes every input parameter and formats them into an issue that can be sent to an external issue tracker.
type Formatter interface {
	// FormatIssuer returns a formatted issue body.
	FormatIssue(data NewIssueData) (string, error)
}

// NewIssueData contains every information available from a reported issue.
type NewIssueData struct {
	Title            string
	Text             string
	OrganizationName string
	UserID           uint
	Labels           []string
}

// Reporter reports an issue to an external issue tracker.
type Reporter interface {
	// ReportIssue sends the formatted issue to an external issue tracker.
	ReportIssue(ctx context.Context, issue Issue) error
}

// NewIssue is reported by a user on the UI.
type NewIssue struct {
	OrganizationName string   `json:"organization"`
	Title            string   `json:"title"`
	Text             string   `json:"text"`
	Labels           []string `json:"labels"`
}

// ReportIssue accepts a new issue and sends it to an external issue tracker service.
func (s service) ReportIssue(ctx context.Context, newIssue NewIssue) error {
	logger := s.logger.WithContext(ctx)

	userID, ok := s.userExtractor.GetUserID(ctx)
	if !ok {
		logger.Warn("user not found in the context")
	}

	data := NewIssueData{
		Title:            newIssue.Title,
		Text:             newIssue.Text,
		OrganizationName: newIssue.OrganizationName,
		UserID:           userID,
		Labels:           newIssue.Labels,
	}

	issueBody, err := s.formatter.FormatIssue(data)
	if err != nil {
		return err
	}

	issue := Issue{
		Title:  newIssue.Title,
		Body:   issueBody,
		Labels: newIssue.Labels,
	}

	err = s.reporter.ReportIssue(ctx, issue)
	if err != nil {
		return err
	}

	return nil
}
