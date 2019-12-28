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

package nodelabels

import (
	"context"
	"strconv"
	"strings"

	"emperror.dev/errors"
	"github.com/sirupsen/logrus"

	"github.com/banzaicloud/pipeline/internal/global"
	"github.com/banzaicloud/pipeline/pkg/common"
)

type NodePoolInfo struct {
	Name         string
	SpotPrice    string
	Preemptible  bool
	InstanceType string
	Labels       map[string]string
}

func GetDesiredLabelsForNodePool(
	nodePool NodePoolInfo,
	noReturnIfNoUserLabels bool,
	cloud string,
	distribution string,
	region string,
) map[string]string {

	desiredLabels := make(map[string]string)
	if len(nodePool.Labels) == 0 && noReturnIfNoUserLabels {
		return desiredLabels
	}

	desiredLabels[common.LabelKey] = nodePool.Name
	desiredLabels[common.OnDemandLabelKey] = getOnDemandLabel(nodePool.SpotPrice, nodePool.Preemptible)

	// copy user labels unless they are not reserved keys
	for labelKey, labelValue := range nodePool.Labels {
		if !IsReservedDomainKey(labelKey) {
			desiredLabels[labelKey] = labelValue
		}
	}

	{
		labels, err := global.NodePoolLabelSource().GetLabels(
			context.Background(),
			cloud,
			distribution,
			region,
			nodePool.InstanceType,
		)

		if err != nil {
			log.WithFields(logrus.Fields{
				"instance":     nodePool.InstanceType,
				"cloud":        cloud,
				"distribution": distribution,
				"region":       region,
			}).Warn(errors.Wrap(err, "failed to get labels from Cloud Info"))
		} else {
			for key, value := range labels {
				desiredLabels[key] = value
			}
		}
	}

	return desiredLabels
}

func IsReservedDomainKey(labelKey string) bool {
	pipelineLabelDomain := global.Config.Cluster.Labels.Domain
	if strings.Contains(labelKey, pipelineLabelDomain) {
		return true
	}

	reservedNodeLabelDomains := global.Config.Cluster.Labels.ForbiddenDomains
	for _, reservedDomain := range reservedNodeLabelDomains {
		if strings.Contains(labelKey, reservedDomain) {
			return true
		}
	}
	return false
}

func getOnDemandLabel(spotPrice string, preemptible bool) string {
	if p, err := strconv.ParseFloat(spotPrice, 64); err == nil && p > 0.0 {
		return "false"
	}
	if preemptible {
		return "false"
	}
	return "true"
}
