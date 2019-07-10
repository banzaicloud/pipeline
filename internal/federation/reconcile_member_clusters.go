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

package federation

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/banzaicloud/pipeline/cluster"
	"github.com/goph/emperror"
	"github.com/hashicorp/go-multierror"
	"github.com/kubernetes-sigs/kubefed/pkg/kubefedctl"
	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
)

func (m *FederationReconciler) ReconcileMemberClusters(desiredState DesiredState) error {
	m.logger.Debug("start reconciling members")
	defer m.logger.Debug("finished reconciling members")

	registeredMembers, err := m.getExistingClusters()
	if err != nil {
		return err
	}

	multiErr := multierror.Error{}

	// if desiredState == DesiredStatePresent join clusters unless they are already registered
	memberClusterIDs := make(map[uint]bool)
	if desiredState == DesiredStatePresent {
		if len(m.Members) > 0 {
			for _, cluster := range m.Members {
				memberClusterIDs[cluster.GetID()] = true
				if _, ok := registeredMembers[cluster.GetID()]; !ok {
					err := m.reconcileMemberCluster(DesiredStatePresent, cluster)
					if err != nil {
						err = emperror.Wrap(err, "Error joining cluster")
						multiErr = *multierror.Append(err, multiErr.Errors...)
					}
				}
			}
		}
	}

	// if desiredState == DesiredStatePresent unjoin registered clusters not members anymore or unjoin all clusters
	// if desiredState == DesiredStateAbsent
	for _, cluster := range registeredMembers {
		if memberClusterIDs[cluster.GetID()] == true {
			continue
		}

		err := m.reconcileMemberCluster(DesiredStateAbsent, cluster)
		if err != nil {
			err = emperror.Wrap(err, "Error unjoining cluster")
			multierror.Append(err, multiErr.Errors...)
		}
	}

	return multiErr.ErrorOrNil()
}

func (m *FederationReconciler) reconcileMemberCluster(desiredState DesiredState, c cluster.CommonCluster) error {
	logger := m.logger.WithField("memberClusterID", c.GetID())

	logger.Debug("start reconciling member cluster")
	defer logger.Debug("finished reconciling member cluster")

	hostConfig, err := m.getClientConfig(m.Host)
	if err != nil {
		return err
	}
	memberConfig, err := m.getClientConfig(c)
	if err != nil {
		return err
	}

	switch desiredState {
	case DesiredStatePresent:
		scope := apiextv1b1.ClusterScoped
		if !m.Configuration.GlobalScope {
			scope = apiextv1b1.NamespaceScoped
		}
		err := kubefedctl.JoinCluster(hostConfig, memberConfig,
			m.Configuration.TargetNamespace, m.Host.GetName(), c.GetName(),
			fmt.Sprintf("%s-cluster", c.GetName()), scope, false, false,
		)
		if err != nil {
			return err
		}
		// label clusters with id
		err = m.labelRegisteredCluster(c)
		if err != nil {
			return err
		}

	case DesiredStateAbsent:
		err := kubefedctl.UnjoinCluster(hostConfig, memberConfig,
			m.Configuration.TargetNamespace,
			m.Host.GetName(), m.Host.GetName(), c.GetName(), c.GetName(),
			true, false)
		if err != nil {
			if strings.Contains(err.Error(), "Failed to get kubefed cluster") {
				logger.Warnf("cluster not found anymore in registry")
			} else {
				return err
			}

		}
	}

	return nil
}

func (m *FederationReconciler) labelRegisteredCluster(c cluster.CommonCluster) error {
	client, err := m.getGenericClient()
	if err != nil {
		return err
	}

	clusterName := c.GetName()
	clusterId := c.GetID()

	clusterLabels := make(map[string]string, 0)
	clusterLabels[clusterLabelId] = fmt.Sprintf("%v", clusterId)
	clusterLabels[clusterLabelCloud] = c.GetCloud()
	clusterLabels[clusterLabelDistribution] = c.GetDistribution()
	clusterLabels[clusterLabelLocation] = c.GetLocation()
	clusterLabels[clusterLabelGroupName] = m.ClusterGroupName

	cluster := &fedv1b1.KubeFedCluster{}
	err = client.Get(context.TODO(), cluster, m.Configuration.TargetNamespace, clusterName)
	if err != nil {
		return err
	}
	if cluster != nil && cluster.Name == clusterName {
		cluster.Labels = clusterLabels
		err = client.Update(context.TODO(), cluster)
		if err != nil {
			return err
		}
	} else {
		return errors.New("cluster not found")
	}
	return nil
}
