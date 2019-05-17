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

package workflow

import (
	"context"

	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

const WaitPersistentVolumesDeletionActivityName = "wait-persistent-volumes-deletion"

type WaitPersistentVolumesDeletionActivityInput struct {
	OrganizationID uint
	ClusterName    string
	K8sSecretID    string
}

// WaitPersistentVolumesDeletionActivity collects the PVs that were created through PVCs
// and are expected to be deleted by Kubernetes upon cluster deletion.
type WaitPersistentVolumesDeletionActivity struct {
	k8sConfigGetter K8sConfigGetter
	logger          logrus.FieldLogger
}

func MakeWaitPersistentVolumesDeletionActivity(k8sConfigGetter K8sConfigGetter, logger logrus.FieldLogger) WaitPersistentVolumesDeletionActivity {
	return WaitPersistentVolumesDeletionActivity{
		k8sConfigGetter: k8sConfigGetter,
		logger:          logger,
	}
}

func (a WaitPersistentVolumesDeletionActivity) Execute(ctx context.Context, input DeleteHelmDeploymentsActivityInput) (err error) {
	logger := a.logger.WithField("organizationID", input.OrganizationID).WithField("clusterName", input.ClusterName)

	k8sConfig, err := a.k8sConfigGetter.Get(input.OrganizationID, input.K8sSecretID)
	if err = emperror.Wrap(err, "failed to get k8s config"); err != nil {
		return
	}

	client, err := k8sclient.NewClientFromKubeConfig(k8sConfig)
	if err = emperror.Wrap(err, "failed to instantiate k8s client"); err != nil {
		return
	}

	// watch persistent volumes
	watcher, err := client.CoreV1().PersistentVolumes().Watch(metav1.ListOptions{})
	if err = emperror.Wrap(err, "failed start watcher for persistent volumes") ;err != nil {
		return
	}
	defer watcher.Stop()

	pvcList, err := client.CoreV1().PersistentVolumeClaims(corev1.NamespaceAll).List(metav1.ListOptions{})
	if err = emperror.Wrap(err, "failed to retrieve persistent volume claims"); err != nil {
		return
	}

	pvList, err := client.CoreV1().PersistentVolumes().List(metav1.ListOptions{})
	if err = emperror.Wrap(err, "failed to retrieve persistent volumes"); err != nil {
		return
	}

	var pvsToWatch = make(map[types.UID]corev1.PersistentVolume)
	for _, pvc := range pvcList.Items {
		if pvc.Status.Phase == corev1.ClaimBound {
			for _, pv := range pvList.Items {
				if pv.Spec.ClaimRef != nil && pv.Spec.ClaimRef.UID == pvc.UID && pv.Spec.PersistentVolumeReclaimPolicy == corev1.PersistentVolumeReclaimDelete {
					pvsToWatch[pv.UID] = pv
					break
				}
			}
		}
	}

	if len(pvsToWatch) == 0 {
		logger.Info("no persistent volumes found to wait for")
		return
	}

	for {
		select {
		case e := <-watcher.ResultChan():
			if e.Object != nil {
				pv, ok := e.Object.(*corev1.PersistentVolume)

				if !ok {
					continue
				}

				switch e.Type {
				case watch.Deleted:
					delete(pvsToWatch, pv.UID)

					if len(pvsToWatch) == 0 { // all watched pvs deleted
						logger.Debug("all watched persistent volumes have been deleted")
						return
					}
				}
			}
		case <-ctx.Done():
			return
		}

	}
}
