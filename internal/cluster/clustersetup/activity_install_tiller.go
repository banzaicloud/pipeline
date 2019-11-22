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

package clustersetup

import (
	"context"
	"fmt"
	"strings"

	"emperror.dev/errors"
	"go.uber.org/cadence/activity"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/cmd/helm/installer"

	"github.com/banzaicloud/pipeline/internal/cluster"
	pkgCluster "github.com/banzaicloud/pipeline/pkg/cluster"
	"github.com/banzaicloud/pipeline/pkg/cluster/pke"
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
)

const InstallTillerActivityName = "install-tiller"

type InstallTillerActivity struct {
	tillerVersion string

	clientFactory cluster.ClientFactory
}

// NewInstallTillerActivity returns a new InstallTillerActivity.
func NewInstallTillerActivity(
	tillerVersion string,
	clientFactory cluster.ClientFactory,
) InstallTillerActivity {
	return InstallTillerActivity{
		tillerVersion: tillerVersion,
		clientFactory: clientFactory,
	}
}

type InstallTillerActivityInput struct {
	// Kubernetes cluster config secret ID.
	ConfigSecretID string

	Distribution string
}

func (a InstallTillerActivity) Execute(ctx context.Context, input InstallTillerActivityInput) error {
	logger := activity.GetLogger(ctx).Sugar()

	opts := a.getOptions(input)

	client, err := a.clientFactory.FromSecret(ctx, input.ConfigSecretID)
	if err != nil {
		return err
	}

	err = a.setupRBAC(ctx, client, opts)
	if err != nil {
		return err
	}

	err = installer.Install(client, &opts)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.WithMessage(err, "failed to install tiller")
	} else if apierrors.IsAlreadyExists(err) {
		logger.Info("tiller is already installed, upgrading")

		if err := installer.Upgrade(client, &opts); err != nil {
			return errors.WithMessage(err, "failed to upgrade tiller")
		}
	}

	return nil
}

func (a InstallTillerActivity) setupRBAC(ctx context.Context, client kubernetes.Interface, opts installer.Options) error {
	logger := activity.GetLogger(ctx).Sugar()

	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: opts.ServiceAccount,
		},
	}

	logger.With("serviceAccount", opts.ServiceAccount).Info("creating service account")

	_, err := client.CoreV1().ServiceAccounts(opts.Namespace).Create(serviceAccount)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to create service account")
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: opts.ServiceAccount,
		},
		Rules: []rbacv1.PolicyRule{{
			APIGroups: []string{
				"*",
			},
			Resources: []string{
				"*",
			},
			Verbs: []string{
				"*",
			},
		},
			{
				NonResourceURLs: []string{
					"*",
				},
				Verbs: []string{
					"*",
				},
			}},
	}

	clusterRoleName := opts.ServiceAccount

	logger.With("clusterRole", opts.ServiceAccount).Info("creating cluster role")

	_, err = client.RbacV1().ClusterRoles().Create(clusterRole)
	if err != nil && strings.Contains(err.Error(), "is forbidden") {
		logger.With("clusterRole", opts.ServiceAccount).Info("creating cluster role is forbidden, falling back to admin")

		_, err := client.RbacV1().ClusterRoles().Get("cluster-admin", metav1.GetOptions{})
		if err != nil {
			return errors.WrapIf(err, "cluster-admin clusterrole not found")
		}

		clusterRoleName = "cluster-admin"
	} else if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to create cluster role")
	}

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: opts.ServiceAccount,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRoleName, // "tiller",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      opts.ServiceAccount, // "tiller",
				Namespace: opts.Namespace,
			},
		},
	}

	logger.With("clusterRoleBinding", opts.ServiceAccount).Info("creating cluster role binding")

	_, err = client.RbacV1().ClusterRoleBindings().Create(clusterRoleBinding)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return errors.Wrap(err, "failed to create cluster role binding")
	}

	return nil
}

func (a InstallTillerActivity) getOptions(input InstallTillerActivityInput) installer.Options {
	opts := installer.Options{
		Namespace:                    k8sutil.KubeSystemNamespace,
		ServiceAccount:               "tiller",
		ImageSpec:                    fmt.Sprintf("gcr.io/kubernetes-helm/tiller:%s", a.tillerVersion),
		AutoMountServiceAccountToken: true,
		ForceUpgrade:                 true,
	}

	if input.Distribution == pkgCluster.PKE {
		tolerations := []corev1.Toleration{
			{
				Key:      pke.TaintKeyMaster,
				Operator: corev1.TolerationOpExists,
			},
		}

		// try to schedule to master or master-worker node
		nodeAffinity := &corev1.NodeAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
				{
					Weight: 100,
					Preference: corev1.NodeSelectorTerm{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      pke.TaintKeyMaster,
								Operator: corev1.NodeSelectorOpExists,
							},
						},
					},
				},
				{
					Weight: 100,
					Preference: corev1.NodeSelectorTerm{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      pke.NodeLabelKeyMasterWorker,
								Operator: corev1.NodeSelectorOpExists,
							},
						},
					},
				},
			},
		}

		for i := range tolerations {
			if tolerations[i].Key != "" {
				opts.Values = append(opts.Values, fmt.Sprintf("spec.template.spec.tolerations[%d].key=%s", i, tolerations[i].Key))
			}

			if tolerations[i].Operator != "" {
				opts.Values = append(opts.Values, fmt.Sprintf("spec.template.spec.tolerations[%d].operator=%s", i, tolerations[i].Operator))
			}

			if tolerations[i].Value != "" {
				opts.Values = append(opts.Values, fmt.Sprintf("spec.template.spec.tolerations[%d].value=%s", i, tolerations[i].Value))
			}

			if tolerations[i].Effect != "" {
				opts.Values = append(opts.Values, fmt.Sprintf("spec.template.spec.tolerations[%d].effect=%s", i, tolerations[i].Effect))
			}
		}

		for i := range nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
			preferredSchedulingTerm := nodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution[i]

			schedulingTermString := fmt.Sprintf("spec.template.spec.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[%d]", i)
			opts.Values = append(opts.Values, fmt.Sprintf("%s.weight=%d", schedulingTermString, preferredSchedulingTerm.Weight))

			for j := range preferredSchedulingTerm.Preference.MatchExpressions {
				matchExpression := preferredSchedulingTerm.Preference.MatchExpressions[j]

				matchExpressionString := fmt.Sprintf("%s.preference.matchExpressions[%d]", schedulingTermString, j)

				opts.Values = append(opts.Values, fmt.Sprintf("%s.key=%s", matchExpressionString, matchExpression.Key))
				opts.Values = append(opts.Values, fmt.Sprintf("%s.operator=%s", matchExpressionString, matchExpression.Operator))

				for k := range matchExpression.Values {
					opts.Values = append(opts.Values, fmt.Sprintf("%s.values[%d]=%v", matchExpressionString, k, matchExpression.Values[i]))
				}
			}
		}
	}

	return opts
}
