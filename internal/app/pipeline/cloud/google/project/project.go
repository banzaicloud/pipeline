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

// Package project provides tools for interacting with Google projects.
package project

import (
	"context"
	"net/http"

	"emperror.dev/errors"
	"google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/option"
)

// Project represents a Google Cloud project.
// TODO: actualize this type from the Google API SDK.
type Project = cloudresourcemanager.Project

// Service interacts with Google projects.
//go:generate mga gen kit endpoint --outdir projectdriver --with-oc --oc-root "cloud/google/project" Service
//go:generate mockery -name Service -inpkg
type Service interface {
	// ListProjects lists Google projects.
	ListProjects(ctx context.Context, secretID string) ([]Project, error)
}

// NewService returns a new Service.
func NewService(clientFactory ClientFactory) Service {
	return service{
		clientFactory: clientFactory,
	}
}

type service struct {
	clientFactory ClientFactory
}

// ClientFactory creates an authenticated HTTP client based on a secret.
type ClientFactory interface {
	// CreateClient creates an authenticated HTTP client based on a secret.
	//
	// If the secret is not of a required type, an error is returned.
	CreateClient(ctx context.Context, secretID string) (*http.Client, error)
}

func (s service) ListProjects(ctx context.Context, secretID string) ([]Project, error) {
	client, err := s.clientFactory.CreateClient(ctx, secretID)
	if err != nil {
		return nil, err
	}

	svc, err := cloudresourcemanager.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cloud service")
	}

	projectSvc := cloudresourcemanager.NewProjectsService(svc)

	var projects []Project

	err = projectSvc.List().Pages(ctx, func(resp *cloudresourcemanager.ListProjectsResponse) error {
		for _, project := range resp.Projects {
			projects = append(projects, *project)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list projects")
	}

	return projects, nil
}
