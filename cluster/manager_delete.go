// Copyright © 2018 Banzai Cloud
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
	"context"

	"emperror.dev/emperror"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/banzaicloud/pipeline/helm"
	intClusterDNS "github.com/banzaicloud/pipeline/internal/cluster/dns"
	intClusterK8s "github.com/banzaicloud/pipeline/internal/cluster/kubernetes"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/secret"
)

// DeleteCluster deletes a cluster.
func (m *Manager) DeleteCluster(ctx context.Context, cluster CommonCluster, force bool) error {

	timer, err := m.getClusterStatusChangeMetricTimer(cluster.GetCloud(), cluster.GetLocation(), pkgCluster.Deleting, cluster.GetOrganizationId(), cluster.GetName())
	if err != nil {
		return err
	}

	errorHandler := m.getClusterErrorHandler(ctx, cluster)

	go func() {
		defer emperror.HandleRecover(errorHandler.WithStatus(pkgCluster.Error, "internal error while deleting cluster"))

		err := m.deleteCluster(context.Background(), cluster, force)
		if err != nil {
			errorHandler.Handle(err)
			return
		}
		timer.RecordDuration()
	}()

	return nil
}

func deleteAllResources(organizationID uint, clusterName string, kubeConfig []byte, namespaces *corev1.NamespaceList, logger *logrus.Entry) error {

	err := deleteUserNamespaces(organizationID, clusterName, kubeConfig, namespaces, logger)
	if err != nil {
		return emperror.Wrap(err, "failed to delete user namespaces")
	}

	deleteDefaultNamespaceResources := true
	if namespaces != nil {
		deleteDefaultNamespaceResources = false
		for _, ns := range namespaces.Items {
			if ns.Name == "default" {
				deleteDefaultNamespaceResources = true
				break
			}
		}
	}

	if deleteDefaultNamespaceResources {
		err = deleteResources(organizationID, clusterName, kubeConfig, "default", logger)
		if err != nil {
			return emperror.Wrap(err, "failed to delete resources in default namespace")
		}

		err = deleteServices(organizationID, clusterName, kubeConfig, "default", logger)
		if err != nil {
			return emperror.Wrap(err, "failed to delete services in default namespace")
		}
	}

	return nil
}

// deleteUserNamespaces deletes all namespace in the context expect the protected ones
func deleteUserNamespaces(organizationID uint, clusterName string, kubeConfig []byte, namespaces *corev1.NamespaceList, logger *logrus.Entry) error {
	deleter := intClusterK8s.MakeUserNamespaceDeleter(logger)
	_, err := deleter.Delete(organizationID, clusterName, namespaces, kubeConfig)
	return err
}

// deleteResources deletes all Services, Deployments, DaemonSets, StatefulSets, ReplicaSets, Pods, and PersistentVolumeClaims of a namespace
func deleteResources(organizationID uint, clusterName string, kubeConfig []byte, ns string, logger *logrus.Entry) error {
	deleter := intClusterK8s.MakeNamespaceResourcesDeleter(logger)
	return deleter.Delete(organizationID, clusterName, kubeConfig, ns)
}

// deleteServices deletes all services one by one from a namespace
func deleteServices(organizationID uint, clusterName string, kubeConfig []byte, ns string, logger *logrus.Entry) error {
	deleter := intClusterK8s.MakeNamespaceServicesDeleter(logger)
	return deleter.Delete(organizationID, clusterName, kubeConfig, ns)
}

// deleteDnsRecordsOwnedByCluster deletes DNS records owned by the cluster. These are the DNS records
// created for the public endpoints of the services hosted by the cluster.
func deleteDnsRecordsOwnedByCluster(cluster CommonCluster) error {
	deleter, err := intClusterDNS.MakeDefaultRecordsDeleter()
	if err != nil {
		return emperror.Wrap(err, "failed to create default cluster DNS records deleter")
	}

	return deleter.Delete(cluster.GetOrganizationId(), cluster.GetUID())
}

func deleteUnusedSecrets(cluster CommonCluster, logger *logrus.Entry) error {
	logger.Info("deleting unused cluster secrets")
	if err := secret.Store.DeleteByClusterUID(cluster.GetOrganizationId(), cluster.GetUID()); err != nil {
		return emperror.Wrap(err, "deleting cluster secret failed")
	}

	if cluster.GetCloud() == pkgCluster.Kubernetes {
		// in case of imported cluster delete the secret that holds the k8s config
		if err := secret.Store.Delete(cluster.GetOrganizationId(), cluster.GetSecretId()); err != nil {
			return emperror.Wrap(err, "deleting cluster secret failed")
		}
	}

	return nil
}

func (m *Manager) deleteCluster(ctx context.Context, cluster CommonCluster, force bool) error {
	logger := m.getLogger(ctx).WithFields(logrus.Fields{
		"organization": cluster.GetOrganizationId(),
		"cluster":      cluster.GetName(),
		"force":        force,
	})

	logger.Info("deleting cluster")

	if err := cluster.SetStatus(pkgCluster.Deleting, pkgCluster.DeletingMessage); err != nil {
		return emperror.WrapWith(err, "cluster status update failed", "cluster_id", cluster.GetID())
	}

	/*
		switch cls := cluster.(type) {
		case *EC2ClusterPKE:
			// the cluster is only deleted from the database for now
			if err = cls.DeleteFromDatabase(); err != nil {
				err = emperror.Wrap(err, "failed to delete from the database")
				if !force {
					cls.SetStatus(pkgCluster.Error, err.Error())
					return err
				}
				logger.Error(err)
			}
			// Send delete event before finish
			m.events.ClusterDeleted(cluster.GetOrganizationId(), cluster.GetName())
			return nil
		}
	*/

	// By default we try to delete resources from the cluster,
	// but in certain cases we want to skip that step
	deleteResources := true

	// delete k8s resources from the cluster
	config, err := cluster.GetK8sConfig()
	if err == ErrConfigNotExists {
		// if the config does not exist, then we were not able to create any k8s resources earlier, so we can proceed with removing the infra
		logger.Infof("deleting unavailable cluster without removing resources: %v", err)

		deleteResources = false
	} else if err != nil {
		err = emperror.Wrap(err, "cannot access Kubernetes cluster")

		if !force {
			_ = cluster.SetStatus(pkgCluster.Error, err.Error())

			return err
		}

		logger.Error(err)

		deleteResources = false
	}

	if deleteResources {

		var namespaceList *corev1.NamespaceList
		if cluster.GetCloud() == pkgCluster.Kubernetes {
			// in case of imported cluster delete only resources from namespaces created by Pipeline
			client, err := k8sclient.NewClientFromKubeConfig(config)
			if err != nil {
				return emperror.Wrap(err, "failed to get Kubernetes clientset from kubeconfig")
			}

			namespaceList, err = client.CoreV1().Namespaces().List(metav1.ListOptions{
				LabelSelector: labels.Set{"owner": "pipeline"}.AsSelector().String(),
			})
			if err != nil {
				err = emperror.Wrap(err, "can not list namespaces")

				if !force {
					_ = cluster.SetStatus(pkgCluster.Error, err.Error())

					return err
				}

				logger.Error(err)
			}
		}

		err = helm.DeleteAllDeployment(logger, config, namespaceList)
		if err != nil {
			err = emperror.Wrap(err, "failed to delete deployments")

			if !force {
				_ = cluster.SetStatus(pkgCluster.Error, err.Error())

				return err
			}

			logger.Error(err)
		}

		err = deleteAllResources(cluster.GetOrganizationId(), cluster.GetName(), config, namespaceList, logger)
		if err != nil {
			err = emperror.Wrap(err, "failed to delete Kubernetes resources")

			if !force {
				_ = cluster.SetStatus(pkgCluster.Error, err.Error())

				return err
			}

			logger.Error(err)
		}
	}

	// clean up dns registrations
	err = deleteDnsRecordsOwnedByCluster(cluster)
	if err != nil {
		err = emperror.Wrap(err, "failed to delete cluster's DNS records")
		logger.Error(err)
	}

	// delete cluster

	if deleter, ok := cluster.(interface {
		DeletePKECluster(context.Context, client.Client) error
	}); ok {
		err = deleter.DeletePKECluster(ctx, m.workflowClient)
	} else {
		err = cluster.DeleteCluster()
	}
	if err != nil {
		err = emperror.Wrap(err, "failed to delete cluster from the provider")
		if !force {
			cluster.SetStatus(pkgCluster.Error, err.Error()) // nolint: errcheck
			return err
		}
		logger.Error(err)
	}

	// delete from proxy from kubeProxyCache if any
	m.DeleteKubeProxy(cluster)

	err = deleteUnusedSecrets(cluster, logger)
	if err != nil {
		err = emperror.Wrap(err, "failed to delete unused cluster secrets")
		if !force {
			cluster.SetStatus(pkgCluster.Error, err.Error()) // nolint: errcheck
			return err
		}
		logger.Error(err)
	}

	// delete cluster from database
	orgID := cluster.GetOrganizationId()
	deleteName := cluster.GetName()
	err = cluster.DeleteFromDatabase()
	if err != nil {
		err = emperror.Wrap(err, "failed to delete from the database")
		if !force {
			cluster.SetStatus(pkgCluster.Error, err.Error()) // nolint: errcheck
			return err
		}
		logger.Error(err)
	}

	logger.Info("cluster deleted successfully")

	m.events.ClusterDeleted(orgID, deleteName)

	return nil
}
