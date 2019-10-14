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

package clustersetup

import (
	"context"
	"testing"
	"text/template"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"

	"github.com/banzaicloud/pipeline/internal/cluster"
)

// nolint: gochecknoglobals
var testInitManifestActivity = InitManifestActivity{}

func testInitManifestActivityExecute(ctx context.Context, input InitManifestActivityInput) error {
	return testInitManifestActivity.Execute(ctx, input)
}

// nolint: gochecknoinits
func init() {
	activity.RegisterWithOptions(testInitManifestActivityExecute, activity.RegisterOptions{Name: InitManifestActivityName})
}

func TestInitManifestActivity(t *testing.T) {
	rawTpl := `clusterID: {{ .Cluster.ID }}
clusterUID: {{ .Cluster.UID }}
clusterName: {{ .Cluster.Name }}

organizationID: {{ .Organization.ID }}
organizationName: {{ .Organization.Name }}
`

	manifest := `clusterID: 1
clusterUID: 260e50ee-d817-4b62-85bd-3260f0e019a0
clusterName: example-cluster

organizationID: 1
organizationName: example-organization
`

	tpl, err := template.New("").Parse(rawTpl)
	require.NoError(t, err)

	client := new(cluster.MockDynamicFileClient)
	client.On("Create", mock.Anything, []byte(manifest)).Return(nil)

	clientFactory := new(cluster.MockDynamicFileClientFactory)
	clientFactory.On("FromSecret", mock.Anything, "secret").Return(client, nil)

	testInitManifestActivity = NewInitManifestActivity(tpl, clientFactory)

	env := (&testsuite.WorkflowTestSuite{}).NewTestActivityEnvironment()

	_, err = env.ExecuteActivity(InitManifestActivityName, InitManifestActivityInput{
		ConfigSecretID: "secret",
		Cluster:        testCluster,
		Organization:   testOrganization,
	})
	require.NoError(t, err)

	clientFactory.AssertExpectations(t)
	client.AssertExpectations(t)
}
