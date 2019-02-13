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

package cluster

import (
	"regexp"
	"strconv"

	"github.com/banzaicloud/nodepool-labels-operator/pkg/npls"
	"github.com/banzaicloud/pipeline/config"
	pipConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/cloudinfo"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/ghodss/yaml"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/api/core/v1"
)

type nodePoolLabelSetOperatorConfig struct {
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
	Affinity    v1.Affinity     `json:"affinity,omitempty"`
}

// InstallNodePoolLabelSetOperator deploys node pool label set operator.
func InstallNodePoolLabelSetOperator(cluster CommonCluster) error {
	pipelineSystemNamespace := viper.GetString(config.PipelineSystemNamespace)
	headNodeAffinity := getHeadNodeAffinity(cluster)
	headNodeTolerations := getHeadNodeTolerations()

	chartName := helm.BanzaiRepository + "/nodepool-labels-operator"
	chartVersion := viper.GetString(config.NodePoolLabelSetOperatorChartVersion)

	config := nodePoolLabelSetOperatorConfig{
		Tolerations: headNodeTolerations,
		Affinity:    headNodeAffinity,
	}

	overrideValues, err := yaml.Marshal(config)
	if err != nil {
		return emperror.Wrap(err, "failed to marshal NodePoolLabelSet operator config to yaml values")
	}

	err = installDeployment(
		cluster,
		pipelineSystemNamespace,
		chartName,
		"npls",
		overrideValues,
		chartVersion,
		true,
	)

	if err != nil {
		return emperror.Wrap(err, "installing NodePoolLabelSet operator failed")
	}

	return nil
}

// SetupNodePoolLabelsSet deploys NodePoolLabelSet resources for each nodepool.
func SetupNodePoolLabelsSet(cluster CommonCluster) error {
	pipelineSystemNamespace := viper.GetString(config.PipelineSystemNamespace)
	headNodePoolName := viper.GetString(pipConfig.PipelineHeadNodePoolName)

	// add node pool name, head node, ondemand labels + cloudinfo + user definied labels
	desiredLabelSet, err := getDesiredLabelForCluster(cluster, headNodePoolName)
	if err != nil {
		return emperror.Wrap(err, "failed to retrieve desired set of labels for cluster")
	}

	k8sConfig, err := cluster.GetK8sConfig()
	if err != nil {
		return emperror.Wrap(err, "failed to set up desired set of labels for cluster")
	}
	k8sClientConfig, err := k8sclient.NewClientConfig(k8sConfig)
	if err != nil {
		return emperror.Wrap(err, "failed to set up desired set of labels for cluster")
	}

	m, err := npls.NewNPLSManager(k8sClientConfig, pipelineSystemNamespace)
	if err != nil {
		return emperror.Wrap(err, "failed to set up desired set of labels for cluster")
	}

	err = m.Sync(desiredLabelSet)
	if err != nil {
		return emperror.Wrap(err, "failed to set up desired set of labels for cluster")
	}

	return nil
}

func getDesiredLabelForCluster(cluster CommonCluster, headNodePoolName string) (npls.NodepoolLabelSets, error) {
	desiredLabels := make(npls.NodepoolLabelSets)

	clusterStatus, err := cluster.GetStatus()
	if err != nil {
		return desiredLabels, emperror.WrapWith(err, "failed to get cluster status", "cluster", cluster.GetName())
	}
	for name, nodePool := range clusterStatus.NodePools {
		desiredLabels[name] = getDesiredNodePoolLabels(clusterStatus, name, nodePool, headNodePoolName)
	}
	return desiredLabels, nil
}

func getDesiredNodePoolLabels(
	clusterStatus *pkgCluster.GetClusterStatusResponse,
	nodePoolName string,
	nodePool *pkgCluster.NodePoolStatus,
	headNodePoolName string) map[string]string {

	desiredLabels := make(map[string]string)
	desiredLabels[common.LabelKey] = nodePoolName
	if nodePoolName == headNodePoolName {
		desiredLabels[common.HeadNodeLabelKey] = "true"
	}
	desiredLabels[common.OnDemandLabelKey] = getOnDemandLabel(nodePool)

	// copy user labels unless they are not reserved keys
	for labelKey, labelValue := range nodePool.Labels {
		if !isReservedDomainKey(labelKey) {
			desiredLabels[labelKey] = labelValue
		}
	}

	// get CloudInfo labels for node
	machineDetails, err := cloudinfo.GetMachineDetails(clusterStatus.Cloud,
		clusterStatus.Distribution,
		clusterStatus.Region,
		nodePool.InstanceType)
	if err != nil {
		log.WithFields(logrus.Fields{
			"instance":     nodePool.InstanceType,
			"cloud":        clusterStatus.Cloud,
			"distribution": clusterStatus.Distribution,
			"region":       clusterStatus.Region,
		}).Warn(errors.Wrap(err, "failed to get instance attributes from Cloud Info"))
	} else {
		for attrKey, attrValue := range machineDetails.Attributes {
			cloudInfoAttrkey := common.CloudInfoLabelKeyPrefix + attrKey
			desiredLabels[cloudInfoAttrkey] = attrValue
		}
	}

	return desiredLabels
}

func isReservedDomainKey(labelKey string) bool {
	reservedNodeLabelDomains := viper.GetStringSlice(pipConfig.ReservedNodeLabelDomains)
	for _, reservedDomain := range reservedNodeLabelDomains {
		if match, _ := regexp.MatchString(reservedDomain, labelKey); match {
			return true
		}
	}
	return false
}

func getOnDemandLabel(nodePool *pkgCluster.NodePoolStatus) string {
	if p, err := strconv.ParseFloat(nodePool.SpotPrice, 64); err == nil && p > 0.0 {
		return "false"
	}
	if nodePool.Preemptible {
		return "false"
	}
	return "true"
}
