package cluster

import (
	"context"
	"fmt"
	"sync"

	"github.com/banzaicloud/pipeline/helm"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeleteCluster deletes a cluster.
func (m *Manager) DeleteCluster(ctx context.Context, cluster CommonCluster, force bool, kubeProxyCache *sync.Map) error {
	errorHandler := emperror.HandlerWith(
		m.getErrorHandler(ctx),
		"organization", cluster.GetOrganizationId(),
		"cluster", cluster.GetID(),
		"force", force,
	)

	go func() {
		err := m.deleteCluster(ctx, cluster, force, kubeProxyCache)
		if err != nil {
			errorHandler.Handle(err)
		}
	}()

	return nil
}

func deleteAllResource(kubeConfig []byte, logger *logrus.Entry) error {
	client, err := helm.GetK8sConnection(kubeConfig)
	if err != nil {
		return err
	}
	// Delete all resources, log errors but ignore them
	err = client.CoreV1().Services("").Delete("", metav1.NewDeleteOptions(0))
	if err != nil {
		logger.Info(err.Error())
	}
	err = client.AppsV1().Deployments("").Delete("", metav1.NewDeleteOptions(0))
	if err != nil {
		logger.Info(err.Error())
	}
	err = client.AppsV1().DaemonSets("").Delete("", metav1.NewDeleteOptions(0))
	if err != nil {
		logger.Info(err.Error())
	}
	err = client.AppsV1().StatefulSets("").Delete("", metav1.NewDeleteOptions(0))
	if err != nil {
		logger.Info(err.Error())
	}
	err = client.AppsV1().ReplicaSets("").Delete("", metav1.NewDeleteOptions(0))
	if err != nil {
		logger.Info(err.Error())
	}
	return nil
}

func (m *Manager) deleteCluster(ctx context.Context, cluster CommonCluster, force bool, kubeProxyCache *sync.Map) error {
	logger := m.getLogger(ctx).WithFields(logrus.Fields{
		"organization": cluster.GetOrganizationId(),
		"cluster":      cluster.GetID(),
		"force":        force,
	})

	logger.Info("deleting cluster")

	err := cluster.UpdateStatus(pkgCluster.Deleting, pkgCluster.DeletingMessage)
	if err != nil {
		return emperror.With(
			emperror.Wrap(err, "cluster status update failed"),
			"cluster", cluster.GetID(),
		)
	}

	// get kubeconfig
	c, err := cluster.GetK8sConfig()
	if err != nil {
		if !force {
			cluster.UpdateStatus(pkgCluster.Error, err.Error())

			return emperror.Wrap(err, "error getting kubeconfig")
		}

		logger.Errorf("error during getting kubeconfig: %s", err.Error())
	}

	if !(force && c == nil) {
		// delete deployments
		err = deleteAllResource(c, logger)
		if err != nil && !force {
			return emperror.Wrap(err, "deleting deployments failed")
		} else if err != nil {
			logger.Errorf("deleting deployments failed: %s", err.Error())
		}
	} else {
		logger.Info("skipping deployment deletion without kubeconfig")
	}

	// delete cluster
	err = cluster.DeleteCluster()
	if err != nil {
		if !force {
			cluster.UpdateStatus(pkgCluster.Error, err.Error())

			return emperror.Wrap(err, "error deleting cluster")
		}

		logger.Errorf("error during deleting cluster: %s", err.Error())
	}

	// delete from proxy from kubeProxyCache if any
	// TODO: this should be handled somewhere else
	kubeProxyCache.Delete(fmt.Sprint(cluster.GetOrganizationId(), "-", cluster.GetID()))

	// delete cluster from database
	deleteName := cluster.GetName()
	err = cluster.DeleteFromDatabase()
	if err != nil {
		if !force {
			cluster.UpdateStatus(pkgCluster.Error, err.Error())

			return emperror.Wrap(err, "error deleting cluster from the database")
		}

		logger.Errorf("error during deleting cluster from the database: %s", err.Error())
	}

	// Asyncron update prometheus
	go func() {
		err := UpdatePrometheusConfig()
		if err != nil {
			logger.Warnf("could not update prometheus configmap: %v", err)
		}
	}()

	// clean statestore
	logger.Info("cleaning cluster's statestore folder")
	if err := CleanStateStore(deleteName); err != nil {
		return emperror.Wrap(err, "cleaning cluster statestore failed")
	}

	logger.Info("cluster's statestore folder cleaned")

	logger.Info("cluster deleted successfully")

	return nil
}
