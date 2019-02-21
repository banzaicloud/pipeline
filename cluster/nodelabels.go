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
	"strconv"
	"strings"

	"github.com/banzaicloud/nodepool-labels-operator/pkg/npls"
	"github.com/banzaicloud/pipeline/config"
	pipConfig "github.com/banzaicloud/pipeline/config"
	"github.com/banzaicloud/pipeline/internal/cloudinfo"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/common"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// DeployNodePoolLabelsSet deploys NodePoolLabelSet resources for each node pool.
// if dontUpdateIfNoUserLabels = true in case there are no labels specified in NodePoolStatus the NodePoolLabelSet for that node pool is not updated,
// to avoid deleting user specified labels in case of cluster update with empty label map.
func DeployNodePoolLabelsSet(cluster CommonCluster, nodePools map[string]*pkgCluster.NodePoolStatus, dontUpdateIfNoUserLabels bool) error {
	clusterStatus, err := cluster.GetStatus()
	if err != nil {
		return emperror.WrapWith(err, "failed to get cluster status", "cluster", cluster.GetName())
	}

	pipelineSystemNamespace := viper.GetString(config.PipelineSystemNamespace)
	headNodePoolName := viper.GetString(pipConfig.PipelineHeadNodePoolName)

	if nodePools == nil {
		nodePools = clusterStatus.NodePools
	}

	// gather desired node pool labels
	desiredLabelSet, err := getDesiredNodePoolLabelSets(clusterStatus, headNodePoolName, nodePools, dontUpdateIfNoUserLabels)
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
		return err
	}

	return nil
}

// getDesiredNodePoolLabelSets add node pool name, head node, ondemand labels + cloudinfo + user definied labels
func getDesiredNodePoolLabelSets(clusterStatus *pkgCluster.GetClusterStatusResponse, headNodePoolName string, nodePools map[string]*pkgCluster.NodePoolStatus, dontUpdateIfNoUserLabels bool) (npls.NodepoolLabelSets, error) {
	desiredLabels := make(npls.NodepoolLabelSets)

	for name, nodePool := range nodePools {
		labelsMap := getDesiredNodePoolLabels(clusterStatus, name, nodePool, headNodePoolName, dontUpdateIfNoUserLabels)
		if len(labelsMap) > 0 {
			desiredLabels[name] = labelsMap
		}
	}
	return desiredLabels, nil
}

func getDesiredNodePoolLabels(clusterStatus *pkgCluster.GetClusterStatusResponse, nodePoolName string,
	nodePool *pkgCluster.NodePoolStatus, headNodePoolName string, dontUpdateIfNoUserLabels bool) map[string]string {

	desiredLabels := make(map[string]string)
	if len(nodePool.Labels) == 0 && dontUpdateIfNoUserLabels {
		return desiredLabels
	}

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
	pipelineLabelDomain := viper.GetString(pipConfig.PipelineLabelDomain)
	if strings.Contains(labelKey, pipelineLabelDomain) {
		return true
	}

	reservedNodeLabelDomains := viper.GetStringSlice(pipConfig.ForbiddenLabelDomains)
	for _, reservedDomain := range reservedNodeLabelDomains {
		if strings.Contains(labelKey, reservedDomain) {
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
