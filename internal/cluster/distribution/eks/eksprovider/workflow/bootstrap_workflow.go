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

package workflow

import (
	"time"

	"emperror.dev/errors"
	"github.com/Masterminds/semver/v3"
	"go.uber.org/cadence"
	"go.uber.org/cadence/workflow"

	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	pkgCadence "github.com/banzaicloud/pipeline/pkg/cadence"
)

const BootstrapWorkflowName = "eks-bootstrap-flow"

// BootstrapWorkflow creates EKS addons and other EKS related configs
type BootstrapWorkflow struct {
	awsSessionFactory *awsworkflow.AWSSessionFactory
	enableAddons      bool
}

// BootstrapWorkflowInput holds input data
type BootstrapWorkflowInput struct {
	EKSActivityInput

	KubernetesVersion   string
	NodeInstanceRoleArn string
	ClusterUserArn      string
	AuthConfigMap       string
}

// NewBootstrapWorkflow instantiates a new BootstrapWorkflow
func NewBootstrapWorkflow(awsSessionFactory *awsworkflow.AWSSessionFactory, enableAddons bool) *BootstrapWorkflow {
	return &BootstrapWorkflow{
		awsSessionFactory: awsSessionFactory,
		enableAddons:      enableAddons,
	}
}

func (a *BootstrapWorkflow) Execute(ctx workflow.Context, input BootstrapWorkflowInput) error {
	ao := workflow.ActivityOptions{
		ScheduleToStartTimeout: 10 * time.Minute,
		StartToCloseTimeout:    5 * time.Minute,
		WaitForCancellation:    true,
		RetryPolicy: &cadence.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 1.5,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    5,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	commonActivityInput := EKSActivityInput{
		OrganizationID: input.OrganizationID,
		SecretID:       input.SecretID,
		Region:         input.Region,
		ClusterName:    input.ClusterName,
	}

	// install EKS addons

	// check add-on are enabled and K8s version is >= 1.18
	constraint, err := semver.NewConstraint(">=1.18")
	if err != nil {
		return errors.WrapIf(err, "could not set 1.18 constraint for semver")
	}
	kubeVersion, err := semver.NewVersion(input.KubernetesVersion)
	if err != nil {
		return errors.WrapIf(err, "could not set eks version for semver check")
	}

	enableAddonFutures := make([]workflow.Future, 0, 3)
	enableAddonErrors := make([]error, 0, 3)
	if a.enableAddons && constraint.Check(kubeVersion) {
		enableAddonFutures = append(enableAddonFutures,
			workflow.ExecuteActivity(ctx, CreateAddonActivityName,
				CreateAddonActivityInput{
					EKSActivityInput:  commonActivityInput,
					KubernetesVersion: input.KubernetesVersion,
					AddonName:         "coredns",
				}),
			workflow.ExecuteActivity(ctx, CreateAddonActivityName,
				CreateAddonActivityInput{
					EKSActivityInput:  commonActivityInput,
					KubernetesVersion: input.KubernetesVersion,
					AddonName:         "vpc-cni",
				}),
			workflow.ExecuteActivity(ctx, CreateAddonActivityName,
				CreateAddonActivityInput{
					EKSActivityInput:  commonActivityInput,
					KubernetesVersion: input.KubernetesVersion,
					AddonName:         "kube-proxy",
				}),
		)
	}

	for _, future := range enableAddonFutures {
		enableAddonErrors = append(enableAddonErrors, pkgCadence.UnwrapError(future.Get(ctx, nil)))
	}
	if err := errors.Combine(enableAddonErrors...); err != nil {
		return err
	}

	// initial setup of K8s cluster
	{
		activityInput := &BootstrapActivityInput{
			EKSActivityInput:    commonActivityInput,
			KubernetesVersion:   input.KubernetesVersion,
			NodeInstanceRoleArn: input.NodeInstanceRoleArn,
			ClusterUserArn:      input.ClusterUserArn,
			AuthConfigMap:       input.AuthConfigMap,
		}
		bootstrapActivityOutput := &BootstrapActivityOutput{}
		// wait for initial cluster setup to terminate
		err = workflow.ExecuteActivity(ctx, BootstrapActivityName, activityInput).Get(ctx, &bootstrapActivityOutput)
		if err != nil {
			return err
		}
	}

	return nil
}
