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
	"fmt"
	"time"

	"emperror.dev/errors"
	"github.com/Masterminds/semver"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/pkg/backoff"
)

const InstallTillerWaitActivityName = "install-tiller-wait"

type InstallTillerWaitActivity struct {
	tillerVersion string

	clientFactory HelmClientFactory
}

// NewInstallTillerWaitActivity returns a new InstallTillerWaitActivity.
func NewInstallTillerWaitActivity(
	tillerVersion string,
	clientFactory HelmClientFactory,
) InstallTillerWaitActivity {
	return InstallTillerWaitActivity{
		tillerVersion: tillerVersion,
		clientFactory: clientFactory,
	}
}

type InstallTillerWaitActivityInput struct {
	// Kubernetes cluster config secret ID.
	ConfigSecretID string
}

func (a InstallTillerWaitActivity) Execute(ctx context.Context, input InstallTillerWaitActivityInput) error {
	logger := activity.GetLogger(ctx).Sugar()

	requiredHelmVersion, err := semver.NewVersion(a.tillerVersion)
	if err != nil {
		return err
	}

	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      15 * time.Second,
		MaxRetries: 30,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(backoffConfig)

	i := 0

	err = backoff.Retry(func() error {
		activity.RecordHeartbeat(ctx)

		logger.With("attempt", fmt.Sprintf("%d/%d", i, backoffConfig.MaxRetries)).Info("waiting for tiller to come up")
		i++

		client, err := a.clientFactory.FromSecret(ctx, input.ConfigSecretID)
		if err != nil {
			return err
		}
		defer client.Close()

		resp, err := client.GetVersion()
		if err != nil {
			logger.Debug("error during retrieving tiller version")

			return err
		}

		if semver.MustParse(resp.Version.SemVer).LessThan(requiredHelmVersion) {
			logger.Info("tiller version is not up to date yet")

			return err
		}

		return nil
	}, backoffPolicy)

	if err != nil {
		return errors.New("timeout during waiting for tiller to get ready")
	}

	return nil
}
