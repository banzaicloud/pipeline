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

package projectdriver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/pipeline/internal/app/pipeline/cloud/google/project"
)

func TestMakeEndpoints_ListProjects(t *testing.T) {
	service := new(project.MockService)

	ctx := context.Background()
	req := listProjectsRequest{
		SecretID: "secret",
	}

	projects := []project.Project{
		{
			Name:      "my-project",
			ProjectId: "1234",
		},
	}

	service.On("ListProjects", ctx, req.SecretID).Return(projects, nil)

	e := MakeEndpoints(service).ListProjects

	result, err := e(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, listProjectsResponse{Projects: projects}, result)

	service.AssertExpectations(t)
}
