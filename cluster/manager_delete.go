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
	"time"

	"github.com/banzaicloud/pipeline/dns"
	"github.com/banzaicloud/pipeline/helm"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/secret"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.uber.org/cadence/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteCluster deletes a cluster.
func (m *Manager) DeleteCluster(ctx context.Context, cluster CommonCluster, force bool) error {

	timer, err := m.getPrometheusTimer(cluster.GetCloud(), cluster.GetLocation(), pkgCluster.Deleting, cluster.GetOrganizationId(), cluster.GetName())
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
		timer.ObserveDuration()
	}()

	return nil
}

func retry(function func() error, count int, delaySeconds int) error {
	i := 1
	for {
		err := function()
		if err == nil || i == count {
			return err
		}
		time.Sleep(time.Second * time.Duration(delaySeconds))
		i++
	}
}

func deleteAllResources(kubeConfig []byte, logger *logrus.Entry) error {

	err := deleteUserNamespaces(kubeConfig, logger)
	if err != nil {
		return emperror.Wrap(err, "failed to delete user namespaces")
	}

	err = deleteResources(kubeConfig, "default", logger)
	if err != nil {
		return emperror.Wrap(err, "failed to delete resources in default namespace")
	}

	err = deleteServices(kubeConfig, "default", logger)
	if err != nil {
		return emperror.Wrap(err, "failed to delete services in default namespace")
	}

	return nil
}

// deleteUserNamespaces deletes all namespace in the context expect the protected ones
func deleteUserNamespaces(kubeConfig []byte, logger *logrus.Entry) error {
	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return err
	}
	namespaces, err := client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return emperror.Wrap(err, "could not list namespaces to delete")
	}

	for _, ns := range namespaces.Items {
		switch ns.Name {
		case "default", "kube-system", "kube-public":
			continue
		}
		err := retry(func() error {
			logger.Infof("deleting kubernetes namespace %q", ns.Name)
			err := client.CoreV1().Namespaces().Delete(ns.Name, &metav1.DeleteOptions{})
			if err != nil {
				return emperror.Wrapf(err, "failed to delete %q namespace", ns.Name)
			}
			return nil
		}, 3, 1)
		if err != nil {
			return err
		}
	}
	err = retry(func() error {
		namespaces, err := client.CoreV1().Namespaces().List(metav1.ListOptions{})
		if err != nil {
			return emperror.Wrap(err, "could not list remaining namespaces")
		}
		left := []string{}
		for _, ns := range namespaces.Items {
			switch ns.Name {
			case "default", "kube-system", "kube-public":
				continue
			default:
				logger.Infof("namespace %q still %s", ns.Name, ns.Status)
				left = append(left, ns.Name)
			}
		}
		if len(left) > 0 {
			return emperror.With(errors.Errorf("namespaces remained after deletion: %v", left), "namespaces", left)
		}
		return nil
	}, 20, 30)
	return err
}

// deleteResources deletes all Services, Deployments, DaemonSets, StatefulSets, ReplicaSets, Pods, and PersistentVolumeClaims of a namespace
func deleteResources(kubeConfig []byte, ns string, logger *logrus.Entry) error {
	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return err
	}
	resourceTypes := []struct {
		DeleteCollectioner interface {
			DeleteCollection(*metav1.DeleteOptions, metav1.ListOptions) error
		}
		Name string
	}{
		{client.AppsV1().Deployments(ns), "Deployments"},
		{client.AppsV1().DaemonSets(ns), "DaemonSets"},
		{client.AppsV1().StatefulSets(ns), "StatefulSets"},
		{client.AppsV1().ReplicaSets(ns), "ReplicaSets"},
		{client.CoreV1().Pods(ns), "Pods"},
		{client.CoreV1().PersistentVolumeClaims(ns), "PersistentVolumeClaims"},
	}

	for _, resourceType := range resourceTypes {
		err := retry(func() error {
			logger.Debugf("deleting %s", resourceType.Name)
			err := resourceType.DeleteCollectioner.DeleteCollection(&metav1.DeleteOptions{}, metav1.ListOptions{})
			if err != nil {
				logger.Infof("could not delete %s: %v", resourceType.Name, err)
			}
			return err
		}, 6, 1)
		if err != nil {
			return emperror.Wrapf(err, "could not delete %s", resourceType.Name)
		}
	}

	return nil
}

// deleteServices deletes all services one by one from a namespace
func deleteServices(kubeConfig []byte, ns string, logger *logrus.Entry) error {
	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return err
	}
	services, err := client.CoreV1().Services(ns).List(metav1.ListOptions{})
	if err != nil {
		return emperror.Wrap(err, "could not list services to delete")
	}

	for _, service := range services.Items {
		switch service.Name {
		case "kubernetes":
			continue
		}
		err := retry(func() error {
			logger.Infof("deleting kubernetes service %q", service.Name)
			err := client.CoreV1().Services(ns).Delete(service.Name, &metav1.DeleteOptions{})
			if err != nil {
				return emperror.Wrapf(err, "failed to delete %q service", service.Name)
			}
			return nil
		}, 3, 1)
		if err != nil {
			return err
		}
	}
	err = retry(func() error {
		services, err := client.CoreV1().Services(ns).List(metav1.ListOptions{})
		if err != nil {
			return emperror.Wrap(err, "could not list remaining services")
		}
		left := []string{}
		for _, svc := range services.Items {
			switch svc.Name {
			case "kubernetes":
				continue
			default:
				logger.Infof("service %q still %s", svc.Name, svc.Status)
				left = append(left, svc.Name)
			}
		}
		if len(left) > 0 {
			return emperror.With(errors.Errorf("services remained after deletion: %v", left), "services", left)
		}
		return nil
	}, 6, 30)
	return err
}

// deleteDnsRecordsOwnedByCluster deletes DNS records owned by the cluster. These are the DNS records
// created for the public endpoints of the services hosted by the cluster.
func deleteDnsRecordsOwnedByCluster(cluster CommonCluster) error {
	dnsSvc, err := dns.GetExternalDnsServiceClient()
	if err != nil {
		return emperror.Wrap(err, "getting external dns service client failed")
	}

	if dnsSvc == nil {
		return nil
	}

	err = dnsSvc.DeleteDnsRecordsOwnedBy(cluster.GetUID(), cluster.GetOrganizationId())
	if err != nil {
		return emperror.Wrapf(err, "deleting DNS records owned by cluster failed")
	}

	return nil
}

func deleteUnusedSecrets(cluster CommonCluster, logger *logrus.Entry) error {
	logger.Info("deleting unused cluster secrets")
	if err := secret.Store.DeleteByClusterUID(cluster.GetOrganizationId(), cluster.GetUID()); err != nil {
		return emperror.Wrap(err, "deleting cluster secret failed")
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

	err := cluster.SetStatus(pkgCluster.Deleting, pkgCluster.DeletingMessage)
	if err != nil {
		return emperror.With(
			emperror.Wrap(err, "cluster status update failed"),
			"cluster_id", cluster.GetID(),
		)
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
			cluster.SetStatus(pkgCluster.Error, err.Error())

			return err
		}

		logger.Error(err)

		deleteResources = false
	}

	if deleteResources {
		err = helm.DeleteAllDeployment(logger, config)
		if err != nil {
			err = emperror.Wrap(err, "failed to delete deployments")

			if !force {
				cluster.SetStatus(pkgCluster.Error, err.Error())

				return err
			}

			logger.Error(err)
		}

		err = deleteAllResources(config, logger)
		if err != nil {
			err = emperror.Wrap(err, "failed to delete Kubernetes resources")

			if !force {
				cluster.SetStatus(pkgCluster.Error, err.Error())

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
			cluster.SetStatus(pkgCluster.Error, err.Error())
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
			cluster.SetStatus(pkgCluster.Error, err.Error())
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
			cluster.SetStatus(pkgCluster.Error, err.Error())
			return err
		}
		logger.Error(err)
	}

	// clean statestore
	logger.Info("cleaning cluster's statestore folder")
	if err := CleanStateStore(deleteName); err != nil {
		return emperror.Wrap(err, "cleaning cluster statestore failed")
	}
	logger.Info("cluster's statestore folder cleaned")

	logger.Info("cluster deleted successfully")

	m.events.ClusterDeleted(orgID, deleteName)

	return nil
}
