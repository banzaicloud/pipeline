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

package frontend

import (
	"emperror.dev/errors"
)

// Config contains configuration required by the frontend application.
type Config struct {
	Issue IssueConfig
}

// Validate validates the configuration.
func (c Config) Validate() error {
	if err := c.Issue.Validate(); err != nil {
		return err
	}

	return nil
}

// IssueConfig contains Issue configuration.
type IssueConfig struct {
	Driver string
	Labels []string

	Github GithubIssueConfig
}

// Validate validates the configuration.
func (c IssueConfig) Validate() error {
	if c.Driver != "github" {
		return errors.New("only github issue driver is supported")
	}

	if c.Driver == "github" {
		if err := c.Github.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// GithubIssueConfig contains GitHub issue driver configuration.
type GithubIssueConfig struct {
	Token      string
	Owner      string
	Repository string
}

// Validate validates the configuration.
func (c GithubIssueConfig) Validate() error {
	if c.Token == "" {
		return errors.New("github token is required")
	}

	if c.Owner == "" {
		return errors.New("github issue repository owner is required")
	}

	if c.Repository == "" {
		return errors.New("github issue repository is required")
	}

	return nil
}
