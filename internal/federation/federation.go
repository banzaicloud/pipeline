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
	"strconv"
	"strings"

	"github.com/banzaicloud/pipeline/cluster"
	"github.com/banzaicloud/pipeline/internal/clustergroup/api"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/goph/emperror"
	"github.com/kubernetes-sigs/kubefed/pkg/client/generic"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	fedv1b1 "sigs.k8s.io/kubefed/pkg/apis/core/v1beta1"
	genericclient "sigs.k8s.io/kubefed/pkg/client/generic"
)

type Config struct {
	// HostClusterID contains the cluster ID where the control plane and the operator runs
	HostClusterID uint `json:"hostClusterID"`
	// TargetNamespace target namespace for federation controller
	TargetNamespace string `json:"targetNamespace,omitempty"`
	// GlobalScope if true TargetNamespace namespace will be the only target for the control plane.
	GlobalScope bool `json:"globalScope,omitempty"`
	// CrossClusterServiceDiscovery if true enables cross cluster service discovery feature
	CrossClusterServiceDiscovery bool `json:"crossClusterServiceDiscovery,omitempty"`
	// FederatedIngress if true enables FederatedIngress feature
	FederatedIngress bool `json:"federatedIngress,omitempty"`
	// SchedulerPreferences if true enables Scheduler preferences feature
	SchedulerPreferences bool `json:"schedulerPreferences,omitempty"`

	name         string
	enabled      bool
	clusterGroup api.ClusterGroup
}

type FederationReconciler struct {
	Configuration    Config
	ClusterGroupName string
	Host             cluster.CommonCluster
	Members          []cluster.CommonCluster

	clusterGetter            api.ClusterGetter
	logger                   logrus.FieldLogger
	errorHandler             emperror.Handler
	serviceDNSRecordResource *metav1.APIResource
	ingressDNSRecordResource *metav1.APIResource
}

type Reconciler func(desiredState DesiredState) error

type DesiredState string

const (
	federationReleaseName = "federationv2"
	federationCRDSuffix   = "kubefed.k8s.io"

	DesiredStatePresent DesiredState = "present"
	DesiredStateAbsent  DesiredState = "absent"

	clusterLabelId           = "clusterId"
	clusterLabelCloud        = "cloud"
	clusterLabelDistribution = "distribution"
	clusterLabelLocation     = "location"
	clusterLabelGroupName    = "groupName"

	multiClusterGroup        = "multiclusterdns.kubefed.k8s.io"
	multiClusterGroupVersion = "v1alpha1"
)

// NewFederationReconciler crates a new feature reconciler for Federation
func NewFederationReconciler(clusterGroupName string, config Config, clusterGetter api.ClusterGetter, logger logrus.FieldLogger, errorHandler emperror.Handler) *FederationReconciler {
	reconciler := &FederationReconciler{
		Configuration:    config,
		ClusterGroupName: clusterGroupName,
		clusterGetter:    clusterGetter,
		logger:           logger,
		errorHandler:     errorHandler,
	}

	reconciler.init()

	reconciler.logger = reconciler.logger.WithFields(logrus.Fields{
		"clusterID":   reconciler.Host.GetID(),
		"clusterName": reconciler.Host.GetName(),
	})

	return reconciler
}

func (m *FederationReconciler) init() error {
	m.Host = m.getHostCluster()
	m.Members = m.getMemberClusters()

	m.serviceDNSRecordResource = &metav1.APIResource{
		Group:      multiClusterGroup,
		Kind:       "ServiceDNSRecord",
		Version:    multiClusterGroupVersion,
		Namespaced: true,
		Name:       "servicednsrecords",
	}

	m.ingressDNSRecordResource = &metav1.APIResource{
		Group:      multiClusterGroup,
		Kind:       "IngressDNSRecord",
		Version:    multiClusterGroupVersion,
		Namespaced: true,
		Name:       "ingressdnsrecords",
	}
	return nil
}

func (m *FederationReconciler) getHostCluster() cluster.CommonCluster {
	for _, c := range m.Configuration.clusterGroup.Clusters {
		if m.Configuration.HostClusterID == c.GetID() {
			return c.(cluster.CommonCluster)
		}
	}

	return nil
}

func (m *FederationReconciler) getMemberClusters() []cluster.CommonCluster {
	remotes := make([]cluster.CommonCluster, 0)

	for _, c := range m.Configuration.clusterGroup.Clusters {
		remotes = append(remotes, c.(cluster.CommonCluster))
	}

	return remotes
}

func getRegisteredClusterStatus(fedCluster fedv1b1.KubeFedCluster) string {
	status := "unknown"
	conditions := fedCluster.Status.Conditions
	for _, c := range conditions {
		if c.Type == "Ready" {
			if c.Status == "True" {
				status = "ready"
			} else {
				status = "not ready"
			}
		} else if c.Type == "Offline" && c.Status == "True" {
			status = "offline"
		}
	}
	return status
}

func (m *FederationReconciler) getRegisteredClusters() (*fedv1b1.KubeFedClusterList, error) {
	client, err := m.getGenericClient()
	if err != nil {
		return nil, err
	}

	clusterList := &fedv1b1.KubeFedClusterList{}
	err = client.List(context.TODO(), clusterList, m.Configuration.TargetNamespace)
	if err != nil {
		if strings.Contains(err.Error(), "no matches for kind") {
			m.logger.Warnf("cluster not found anymore in registry")
			clusterList.Items = make([]fedv1b1.KubeFedCluster, 0)
		} else {
			return nil, err
		}
	}
	return clusterList, nil
}

func (m *FederationReconciler) getExistingClusters() (map[uint]cluster.CommonCluster, error) {
	clusters := make(map[uint]cluster.CommonCluster, 0)

	clusterNameIdMap := make(map[string]cluster.CommonCluster, 0)

	for _, memberCluster := range m.getMemberClusters() {
		clusterNameIdMap[memberCluster.GetName()] = memberCluster
	}

	existingClusterList, err := m.getRegisteredClusters()
	if err != nil {
		return nil, err
	}

	for _, cl := range existingClusterList.Items {
		if len(cl.Labels) == 0 {
			continue
		}

		clusterIdStr := cl.Labels[clusterLabelId]
		if len(clusterIdStr) == 0 {
			continue
		}

		clusterID, err := strconv.ParseUint(clusterIdStr, 10, 64)
		if err != nil {
			m.errorHandler.Handle(errors.WithStack(err))
			continue
		}

		c, err := m.clusterGetter.GetClusterByID(context.Background(), m.Host.GetOrganizationId(), uint(clusterID))
		if err != nil {
			m.errorHandler.Handle(errors.WithStack(err))
			continue
		}

		clusters[c.GetID()] = c.(cluster.CommonCluster)
	}

	return clusters, nil
}

func (m *FederationReconciler) GetStatus() (map[uint]string, error) {
	statusMap := make(map[uint]string, 0)

	clusterNameIdMap := make(map[string]uint, 0)
	for _, memberCluster := range m.Configuration.clusterGroup.Members {
		clusterNameIdMap[memberCluster.Name] = memberCluster.ID
	}

	existingClusterList, err := m.getRegisteredClusters()
	if err != nil {
		return nil, err
	}

	for _, cluster := range existingClusterList.Items {
		clusterId, ok := clusterNameIdMap[cluster.Name]
		if ok {
			statusMap[clusterId] = getRegisteredClusterStatus(cluster)
		} else {
			statusMap[0] = getRegisteredClusterStatus(cluster)
		}
	}

	return statusMap, nil
}

func (m *FederationReconciler) getClientConfig(c cluster.CommonCluster) (*rest.Config, error) {
	kubeConfig, err := c.GetK8sConfig()
	if err != nil {
		return nil, emperror.Wrap(err, "could not get k8s config")
	}

	clientConfig, err := k8sclient.NewClientConfig(kubeConfig)
	if err != nil {
		return nil, emperror.Wrap(err, "cloud not create client config from kubeconfig")
	}

	return clientConfig, nil
}

func (m *FederationReconciler) getGenericClient() (generic.Client, error) {
	clientConfig, err := m.getClientConfig(m.Host)
	if err != nil {
		return nil, err
	}
	client, err := genericclient.New(clientConfig)
	if err != nil {
		return nil, emperror.Wrap(err, "could not get kubefed clientset")
	}
	return client, nil
}
