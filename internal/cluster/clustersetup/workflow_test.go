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

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/cadence/activity"
	"go.uber.org/cadence/testsuite"
	"go.uber.org/cadence/workflow"
)

// nolint: gochecknoglobals
var testCluster = Cluster{
	ID:           1,
	UID:          "260e50ee-d817-4b62-85bd-3260f0e019a0",
	Name:         "example-cluster",
	Distribution: "pke",
}

// nolint: gochecknoglobals
var testOrganization = Organization{
	ID:   1,
	Name: "example-organization",
}

// nolint: gochecknoglobals
var testNodePoolLabels = map[string]map[string]string{
	"pool1": {
		"key": "value",
	},
}

type WorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

type DeployClusterAutoscalerActivity struct {
}

func (a DeployClusterAutoscalerActivity) Execute(ctx context.Context, input DeployClusterAutoscalerActivityInput) error {
	return nil
}

func TestWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(WorkflowTestSuite))
}

func (s *WorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()

	s.env.RegisterActivityWithOptions(InitManifestActivity{}.Execute, activity.RegisterOptions{Name: InitManifestActivityName})
	s.env.RegisterActivityWithOptions(InstallNodePoolLabelSetOperatorActivity{}.Execute, activity.RegisterOptions{Name: InstallNodePoolLabelSetOperatorActivityName})
	s.env.RegisterActivityWithOptions(ConfigureNodePoolLabelsActivity{}.Execute, activity.RegisterOptions{Name: ConfigureNodePoolLabelsActivityName})
	s.env.RegisterActivityWithOptions(CreatePipelineNamespaceActivity{}.Execute, activity.RegisterOptions{Name: CreatePipelineNamespaceActivityName})
	s.env.RegisterActivityWithOptions(LabelKubeSystemNamespaceActivity{}.Execute, activity.RegisterOptions{Name: LabelKubeSystemNamespaceActivityName})
	s.env.RegisterActivityWithOptions(DeployClusterAutoscalerActivity{}.Execute, activity.RegisterOptions{Name: DeployClusterAutoscalerActivityName})
	s.env.RegisterActivityWithOptions(DeployIngressControllerActivity{}.Execute, activity.RegisterOptions{Name: DeployIngressControllerActivityName})
	s.env.RegisterActivityWithOptions(DeployInstanceTerminationHandlerActivity{}.Execute, activity.RegisterOptions{Name: DeployInstanceTerminationHandlerActivityName})
}

func (s *WorkflowTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func (s *WorkflowTestSuite) Test_Success() {
	wf := Workflow{}
	s.env.RegisterWorkflowWithOptions(wf.Execute, workflow.RegisterOptions{Name: s.T().Name()})

	s.env.OnActivity(
		CreatePipelineNamespaceActivityName,
		mock.Anything,
		CreatePipelineNamespaceActivityInput{ConfigSecretID: "secret"},
	).Return(nil)

	s.env.OnActivity(
		LabelKubeSystemNamespaceActivityName,
		mock.Anything,
		LabelKubeSystemNamespaceActivityInput{ConfigSecretID: "secret"},
	).Return(nil)

	s.env.OnActivity(
		InstallNodePoolLabelSetOperatorActivityName,
		mock.Anything,
		InstallNodePoolLabelSetOperatorActivityInput{ClusterID: 1},
	).Return(nil)

	s.env.OnActivity(
		ConfigureNodePoolLabelsActivityName,
		mock.Anything,
		ConfigureNodePoolLabelsActivityInput{
			ConfigSecretID: "secret",
			Labels:         testNodePoolLabels,
		},
	).Return(nil)

	s.env.OnActivity(
		ConfigureNodePoolLabelsActivityName,
		mock.Anything,
		ConfigureNodePoolLabelsActivityInput{
			ConfigSecretID: "secret",
			Labels:         testNodePoolLabels,
		},
	).Return(nil)

	s.env.OnActivity(
		DeployClusterAutoscalerActivityName,
		mock.Anything,
		DeployClusterAutoscalerActivityInput{ClusterID: 1},
	).Return(nil)

	s.env.OnActivity(
		DeployIngressControllerActivityName,
		mock.Anything,
		DeployIngressControllerActivityInput{
			ClusterID: 1,
			OrgID:     1,
			Cloud:     "",
		},
	).Return(nil)

	s.env.OnActivity(
		DeployInstanceTerminationHandlerActivityName,
		mock.Anything,
		DeployInstanceTerminationHandlerActivityInput{
			ClusterID:   1,
			OrgID:       1,
			Cloud:       "",
			ClusterName: "example-cluster",
		},
	).Return(nil)

	workflowInput := WorkflowInput{
		ConfigSecretID: "secret",
		Cluster:        testCluster,
		Organization:   testOrganization,
		NodePoolLabels: testNodePoolLabels,
	}

	s.env.ExecuteWorkflow(s.T().Name(), workflowInput)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}

func (s *WorkflowTestSuite) Test_Success_InstallInitManifest() {
	wf := Workflow{
		InstallInitManifest: true,
	}
	s.env.RegisterWorkflowWithOptions(wf.Execute, workflow.RegisterOptions{Name: s.T().Name()})

	s.env.OnActivity(
		InitManifestActivityName,
		mock.Anything,
		InitManifestActivityInput{ConfigSecretID: "secret", Cluster: testCluster, Organization: testOrganization},
	).Return(nil)

	s.env.OnActivity(
		CreatePipelineNamespaceActivityName,
		mock.Anything,
		CreatePipelineNamespaceActivityInput{ConfigSecretID: "secret"},
	).Return(nil)

	s.env.OnActivity(
		LabelKubeSystemNamespaceActivityName,
		mock.Anything,
		LabelKubeSystemNamespaceActivityInput{ConfigSecretID: "secret"},
	).Return(nil)

	s.env.OnActivity(
		InstallNodePoolLabelSetOperatorActivityName,
		mock.Anything,
		InstallNodePoolLabelSetOperatorActivityInput{ClusterID: 1},
	).Return(nil)

	s.env.OnActivity(
		ConfigureNodePoolLabelsActivityName,
		mock.Anything,
		ConfigureNodePoolLabelsActivityInput{
			ConfigSecretID: "secret",
			Labels:         testNodePoolLabels,
		},
	).Return(nil)

	s.env.OnActivity(
		DeployClusterAutoscalerActivityName,
		mock.Anything,
		DeployClusterAutoscalerActivityInput{ClusterID: 1},
	).Return(nil)

	s.env.OnActivity(
		DeployIngressControllerActivityName,
		mock.Anything,
		DeployIngressControllerActivityInput{
			ClusterID: 1,
			OrgID:     1,
			Cloud:     "",
		},
	).Return(nil)

	s.env.OnActivity(
		DeployInstanceTerminationHandlerActivityName,
		mock.Anything,
		DeployInstanceTerminationHandlerActivityInput{
			ClusterID:   1,
			OrgID:       1,
			Cloud:       "",
			ClusterName: "example-cluster",
		},
	).Return(nil)

	workflowInput := WorkflowInput{
		ConfigSecretID: "secret",
		Cluster:        testCluster,
		Organization:   testOrganization,
		NodePoolLabels: testNodePoolLabels,
	}

	s.env.ExecuteWorkflow(s.T().Name(), workflowInput)

	s.True(s.env.IsWorkflowCompleted())
	s.NoError(s.env.GetWorkflowError())
}
