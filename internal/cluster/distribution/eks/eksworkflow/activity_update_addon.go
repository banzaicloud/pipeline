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

package eksworkflow

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/Masterminds/semver/v3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/eks"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/cluster/distribution/eks/eksprovider/workflow"
	"github.com/banzaicloud/pipeline/internal/cluster/infrastructure/aws/awsworkflow"
	"github.com/banzaicloud/pipeline/pkg/cadence/worker"
)

const UpdateAddonActivityName = "eks-update-addon"

// UpdateAddonActivity responsible for updating an EKS addon
// Addon is updated only in case it's already enable on cluster and there's a newer version available
// for given Kubernetes version.
type UpdateAddonActivity struct {
	awsSessionFactory awsworkflow.AWSFactory
	eksFactory        workflow.EKSAPIFactory
}

// UpdateAddonActivityInput holds data needed for updating an EKS addon
type UpdateAddonActivityInput struct {
	OrganizationID    uint
	ProviderSecretID  string
	Region            string
	ClusterID         uint
	ClusterName       string
	KubernetesVersion string

	AddonName string
}

// UpdateAddonActivityOutput holds the output data of the UpdateAddonActivityOutput
type UpdateAddonActivityOutput struct {
	UpdateID          string
	AddonNotInstalled bool
}

// NewUpdateAddonActivity instantiates a new EKS addon version update
func NewUpdateAddonActivity(
	awsSessionFactory awsworkflow.AWSFactory, eksFactory workflow.EKSAPIFactory,
) *UpdateAddonActivity {
	return &UpdateAddonActivity{
		awsSessionFactory: awsSessionFactory,
		eksFactory:        eksFactory,
	}
}

// Register registers the activity in the worker.
func (a UpdateAddonActivity) Register(worker worker.Registry) {
	worker.RegisterActivityWithOptions(a.Execute, activity.RegisterOptions{Name: UpdateAddonActivityName})
}

func (a *UpdateAddonActivity) Execute(ctx context.Context, input UpdateAddonActivityInput) (*UpdateAddonActivityOutput, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"region", input.Region,
		"addonName", input.AddonName,
	)

	session, err := a.awsSessionFactory.New(input.OrganizationID, input.ProviderSecretID, input.Region)
	if err = errors.WrapIf(err, "failed to create AWS session"); err != nil {
		return nil, err
	}

	eksSvc := a.eksFactory.New(session)

	describeAddonInput := &eks.DescribeAddonInput{
		AddonName:   aws.String(input.AddonName),
		ClusterName: aws.String(input.ClusterName),
	}
	addonOutput, err := eksSvc.DescribeAddon(describeAddonInput)
	if err != nil {
		if isAWSAddonNotFoundError(err, input.AddonName, input.ClusterName) { // Note: no update for not existing addons.
			logger.Infof("%s", err.Error())

			return &UpdateAddonActivityOutput{AddonNotInstalled: true}, nil
		}

		return nil, errors.WrapIfWithDetails(err, "failed to retrieve addon", "cluster", input.ClusterName, "addon", input.AddonName)
	}

	currentVersion := *addonOutput.Addon.AddonVersion
	logger.Infof("addon current version: %v", currentVersion)

	describeAddonVersionInput := &eks.DescribeAddonVersionsInput{
		AddonName:         aws.String(input.AddonName),
		KubernetesVersion: aws.String(input.KubernetesVersion),
	}
	addonVersionsOutput, err := eksSvc.DescribeAddonVersions(describeAddonVersionInput)
	if err != nil {
		var awsErr awserr.Error
		if errors.As(err, &awsErr) {
			err = errors.New(awsErr.Message())
		}
		return nil, errors.WrapIfWithDetails(err, "failed to retrieve addon versions", "cluster", input.ClusterName, "addon", input.AddonName)
	}

	selectedVersion, err := selectLatestVersion(addonVersionsOutput, currentVersion, input.KubernetesVersion)
	if err != nil {
		return nil, errors.WrapIfWithDetails(err, "error selecting new version", "cluster", input.ClusterName, "addon", input.AddonName)
	}
	if selectedVersion == currentVersion {
		logger.Infof("no newer version available then current version: %s", currentVersion)
		return &UpdateAddonActivityOutput{UpdateID: ""}, nil
	}

	logger.Infof("update addon to selected version: %v", selectedVersion)
	updateAddonInput := &eks.UpdateAddonInput{
		AddonName:        aws.String(input.AddonName),
		ClusterName:      aws.String(input.ClusterName),
		ResolveConflicts: aws.String(eks.ResolveConflictsOverwrite),
		AddonVersion:     aws.String(selectedVersion),
	}
	updateAddonOutput, err := eksSvc.UpdateAddon(updateAddonInput)
	if err != nil {
		var awsErr awserr.Error
		if errors.As(err, &awsErr) {
			err = errors.New(awsErr.Message())
		}
		return nil, errors.WrapIfWithDetails(err, "failed to update addon", "cluster", input.ClusterName, "addon", input.AddonName)
	}
	output := UpdateAddonActivityOutput{UpdateID: aws.StringValue(updateAddonOutput.Update.Id)}

	return &output, nil
}

func selectLatestVersion(addonVersions *eks.DescribeAddonVersionsOutput, currentVersion string, kubernetesVersion string) (string, error) {
	currentVersionSemver, err := semver.NewVersion(currentVersion)
	if err != nil {
		return "", err
	}
	latestVersion := currentVersionSemver

	for _, addon := range addonVersions.Addons {
		for _, version := range addon.AddonVersions {
			if !versionIsCompatible(version.Compatibilities, kubernetesVersion) {
				continue
			}
			newVersion, err := semver.NewVersion(*version.AddonVersion)
			if err != nil {
				return "", err
			}
			if newVersion.GreaterThan(latestVersion) {
				latestVersion = newVersion
			}
		}
	}
	return latestVersion.Original(), nil
}

// errorMessageAWSAddonNotFound is the error message returned by AWS when a
// non-existing cluster addon is queried (for example in DescribeAddon()).
func errorMessageAWSAddonNotFound(addonName, clusterName string) string {
	return fmt.Sprintf("No addon: %s found in cluster: %s", addonName, clusterName)
}

// isAWSAddonNotFoundError returns a boolean indicator of whether the specified
// error is an error indicating the cluster has no such addon.
func isAWSAddonNotFoundError(err error, addonName, clusterName string) bool {
	if err == nil {
		return false
	}

	var awsErr awserr.Error

	return errors.As(err, &awsErr) &&
		awsErr.Message() == errorMessageAWSAddonNotFound(addonName, clusterName)
}

func versionIsCompatible(compatibilities []*eks.Compatibility, kubernetesVersion string) bool {
	for _, c := range compatibilities {
		if c.ClusterVersion == nil {
			continue
		}
		if *c.ClusterVersion == kubernetesVersion {
			return true
		}
	}
	return false
}
