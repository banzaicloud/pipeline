// Copyright © 2020 Banzai Cloud
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

package pkeawsadapter

import (
	"context"
	"strconv"
	"strings"

	"emperror.dev/errors"
	"github.com/Masterminds/semver/v3"

	"github.com/banzaicloud/pipeline/.gen/cloudinfo"
	"github.com/banzaicloud/pipeline/internal/cluster/distribution/pke/pkeaws"
)

// CloudinfoImageSelector lists the available images provided by Cloudinfo and
// selects one based on the provided criteria.
type CloudinfoImageSelector struct {
	client *cloudinfo.APIClient
}

// NewCloudinfoImageSelector returns a new CloudinfoImageSelector.
func NewCloudinfoImageSelector(client *cloudinfo.APIClient) CloudinfoImageSelector {
	return CloudinfoImageSelector{
		client: client,
	}
}

func (s CloudinfoImageSelector) SelectImage(ctx context.Context, criteria pkeaws.ImageSelectionCriteria) (string, error) {
	if s.client == nil {
		return "", errors.New("cloudinfo: client not configured")
	}

	// TODO: validate kubernetes version earlier
	kubeVersion, err := semver.NewVersion(criteria.KubernetesVersion)
	if err != nil {
		return "", errors.WrapWithDetails(
			err, "parse kubernetes version",
			"kubernetesVersion", criteria.KubernetesVersion,
		)
	}

	isGPUInstance := func(instanceType string) bool {
		return strings.HasPrefix(instanceType, "p2.") || strings.HasPrefix(instanceType, "p3.") ||
			strings.HasPrefix(instanceType, "g3.") || strings.HasPrefix(instanceType, "g4.")
	}

	const (
		cloudProvider = "amazon"
		serviceName   = "pke"
	)

	getImagesRequest := s.client.ImagesApi.GetImages(ctx, cloudProvider, serviceName, criteria.Region)
	getImagesRequest.Version(kubeVersion.String())
	getImagesRequest.Os(criteria.OperatingSystem)
	getImagesRequest.PkeVersion(criteria.PKEVersion)
	getImagesRequest.LatestOnly("true")
	getImagesRequest.Gpu(strconv.FormatBool(isGPUInstance(criteria.InstanceType)))
	getImagesRequest.Cr(criteria.ContainerRuntime)

	images, _, err := s.client.ImagesApi.GetImagesExecute(getImagesRequest)
	if err != nil {
		return "", errors.WrapIfWithDetails(
			err, "get images from cloudinfo",
			"cloudProvider", cloudProvider,
			"service", serviceName,
			"region", criteria.Region,
			"version", kubeVersion.String(),
			"os", criteria.OperatingSystem,
			"PkeVersion", criteria.PKEVersion,
			"LatestOnly", "true",
			"Gpu", strconv.FormatBool(isGPUInstance(criteria.InstanceType)),
			"Cr", criteria.ContainerRuntime,
		)
	}
	if len(images) == 0 {
		return "", errors.WithDetails(
			errors.WithStack(pkeaws.ImageNotFoundError),
			"cloudProvider", cloudProvider,
			"service", serviceName,
			"region", criteria.Region,
			"region", criteria.Region,
			"version", kubeVersion.String(),
			"os", criteria.OperatingSystem,
			"PkeVersion", criteria.PKEVersion,
			"LatestOnly", "true",
			"Gpu", strconv.FormatBool(isGPUInstance(criteria.InstanceType)),
			"Cr", criteria.ContainerRuntime,
		)
	}

	// As a result of a bug in cloudinfo,
	// the returned item might be empty
	// See https://github.com/banzaicloud/cloudinfo/pull/356
	e := ""
	if images[0].Name == &e {
		return "", errors.WithDetails(
			errors.WithStack(pkeaws.ImageNotFoundError),
			"cloudProvider", cloudProvider,
			"service", serviceName,
			"region", criteria.Region,
			"region", criteria.Region,
			"version", kubeVersion.String(),
			"os", criteria.OperatingSystem,
			"PkeVersion", criteria.PKEVersion,
			"LatestOnly", "true",
			"Gpu", strconv.FormatBool(isGPUInstance(criteria.InstanceType)),
			"Cr", criteria.ContainerRuntime,
		)
	}

	return *images[0].Name, nil
}
