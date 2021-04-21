// Copyright © 2021 Banzai Cloud
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

package backup

import (
	"context"

	"emperror.dev/errors"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1/backup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/integratedservices"
	"github.com/banzaicloud/pipeline/internal/integratedservices/integratedserviceadapter"
	"github.com/banzaicloud/pipeline/src/auth"
)

// Operator component implementing the operations related to the backup integrated service
type BackupOperator struct {
	clusterGetter  integratedserviceadapter.ClusterGetter
	clusterService integratedservices.ClusterService
	namespace      string
	reconciler     integratedserviceadapter.Reconciler
	logger         common.Logger
}

func NewBackupOperator(
	clusterGetter integratedserviceadapter.ClusterGetter,
	clusterService integratedservices.ClusterService,
	namespace string,
	logger common.Logger,
) BackupOperator {
	return BackupOperator{
		clusterGetter:  clusterGetter,
		clusterService: clusterService,
		namespace:      namespace,
		reconciler:     integratedserviceadapter.NewISReconciler(logger),
		logger:         logger,
	}
}

func (o BackupOperator) Apply(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) error {
	ctx, err := o.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {
		return err
	}

	if err := o.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	boundSpec, err := backup.BindIntegratedServiceSpec(spec)
	if err != nil {
		return errors.WrapIf(err, "failed to bind integrated service spec")
	}

	cl, err := o.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to retrieve the cluster")
	}

	k8sConfig, err := cl.GetK8sConfig()
	if err != nil {
		return errors.WrapIf(err, "failed to retrieve the k8s config")
	}
	b := true
	si := v1alpha1.ServiceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: o.namespace,
			Name:      IntegratedServiceName,
		},
		Spec: v1alpha1.ServiceInstanceSpec{
			Service: IntegratedServiceName,
			Enabled: &b,
			Backup: v1alpha1.Backup{
				Spec: &boundSpec,
			},
		},
	}

	if rErr := o.reconciler.Reconcile(ctx, k8sConfig, si); rErr != nil {
		return errors.Wrap(rErr, "failed to reconcile the integrated service resource")
	}

	return nil
}

func (o BackupOperator) Deactivate(ctx context.Context, clusterID uint, spec integratedservices.IntegratedServiceSpec) error {
	ctx, err := o.ensureOrgIDInContext(ctx, clusterID)
	if err != nil {
		return err
	}

	if err := o.clusterService.CheckClusterReady(ctx, clusterID); err != nil {
		return err
	}

	cl, err := o.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to retrieve the cluster")
	}

	k8sConfig, err := cl.GetK8sConfig()
	if err != nil {
		return errors.WrapIf(err, "failed to retrieve the k8s config")
	}

	si := v1alpha1.ServiceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: o.namespace,
			Name:      IntegratedServiceName,
		},
	}
	if rErr := o.reconciler.Disable(ctx, k8sConfig, si); rErr != nil {
		return errors.Wrap(rErr, "failed to reconcile the integrated service resource")
	}

	return nil
}

func (o BackupOperator) ensureOrgIDInContext(ctx context.Context, clusterID uint) (context.Context, error) {
	if _, ok := auth.GetCurrentOrganizationID(ctx); !ok {
		cluster, err := o.clusterGetter.GetClusterByIDOnly(ctx, clusterID)
		if err != nil {
			return ctx, errors.WrapIf(err, "failed to get cluster by ID")
		}
		ctx = auth.SetCurrentOrganizationID(ctx, cluster.GetOrganizationId())
	}
	return ctx, nil
}

func (o BackupOperator) Name() string {
	return IntegratedServiceName
}
