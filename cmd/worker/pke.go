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

package main

import (
	"emperror.dev/errors"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/cloudformation"
	eksworkflow "github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws/pkeawsworkflow"
	pkeworkflow "github.com/banzaicloud/pipeline/internal/pke/workflow"
)

const (
	PKECloudFormationTemplateBasePath = "templates/pke"
	WorkerCloudFormationTemplate      = "worker.cf.yaml"
)

func registerPKEWorkflows(passwordSecrets pkeworkflow.PasswordSecretStore, secretStore eksworkflow.SecretStore) error {
	awsSessionFactory := eksworkflow.NewAWSSessionFactory(secretStore)

	nodePoolTemplate, err := cloudformation.GetCloudFormationTemplate(PKECloudFormationTemplateBasePath, WorkerCloudFormationTemplate)
	if err != nil {
		return errors.WrapIf(err, "failed to get CloudFormation template for node pools")
	}

	{
		a := pkeworkflow.NewAssembleHTTPProxySettingsActivity(passwordSecrets)
		activity.RegisterWithOptions(a.Execute, activity.RegisterOptions{Name: pkeworkflow.AssembleHTTPProxySettingsActivityName})
	}

	pkeawsworkflow.NewUpdateNodePoolWorkflow().Register()
	pkeawsworkflow.NewUpdateNodeGroupActivity(awsSessionFactory, nodePoolTemplate).Register()

	return nil
}
