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

package helm

import (
	"fmt"
	"strings"
	"time"

	"emperror.dev/emperror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/cmd/helm/installer"

	"github.com/banzaicloud/pipeline/pkg/backoff"
	phelm "github.com/banzaicloud/pipeline/pkg/helm"
	"github.com/banzaicloud/pipeline/pkg/k8sclient"
	"github.com/banzaicloud/pipeline/pkg/k8sutil"
)

// PreInstall create's serviceAccount and AccountRoleBinding
func PreInstall(log logrus.FieldLogger, helmInstall *phelm.Install, kubeConfig []byte) error {
	log.Info("start pre-install")

	var backoffConfig = backoff.ConstantBackoffConfig{
		Delay:      10 * time.Second,
		MaxRetries: 5,
	}
	var backoffPolicy = backoff.NewConstantBackoffPolicy(backoffConfig)

	client, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		log.Errorf("could not get kubernetes client: %s", err)
		return err
	}

	v1MetaData := metav1.ObjectMeta{
		Name: helmInstall.ServiceAccount, // "tiller",
	}

	serviceAccount := &apiv1.ServiceAccount{
		ObjectMeta: v1MetaData,
	}
	log.Info("create serviceaccount")

	err = backoff.Retry(func() error {
		if _, err := client.CoreV1().ServiceAccounts(helmInstall.Namespace).Create(serviceAccount); err != nil {
			if k8sutil.IsK8sErrorPermanent(err) {
				return backoff.MarkErrorPermanent(err)
			}
		}
		return nil
	}, backoffPolicy)
	if err != nil {
		return emperror.WrapWith(err, "could not create serviceaccount", "serviceaccount", serviceAccount, "namespace", helmInstall.Namespace)
	}

	clusterRole := &v1.ClusterRole{
		ObjectMeta: v1MetaData,
		Rules: []v1.PolicyRule{{
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
	log.Info("create clusterroles")

	clusterRoleName := helmInstall.ServiceAccount
	err = backoff.Retry(func() error {
		if _, err := client.RbacV1().ClusterRoles().Create(clusterRole); err != nil {
			if k8sutil.IsK8sErrorPermanent(err) {
				return backoff.MarkErrorPermanent(err)
			}
		}
		return nil
	}, backoffPolicy)
	if err != nil && strings.Contains(err.Error(), "is forbidden") {
		_, errGet := client.RbacV1().ClusterRoles().Get("cluster-admin", metav1.GetOptions{})
		if errGet != nil {
			return emperror.Wrap(errGet, "cluster-admin clusterrole not found")
		}
		clusterRoleName = "cluster-admin"
	}
	if err != nil {
		return emperror.WrapWith(err, "could not create clusterrole", "clusterrole", clusterRole.Name)
	}

	log.Debugf("ClusterRole Name: %s", clusterRoleName)
	log.Debugf("serviceAccount Name: %s", helmInstall.ServiceAccount)
	clusterRoleBinding := &v1.ClusterRoleBinding{
		ObjectMeta: v1MetaData,
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRoleName, // "tiller",
		},
		Subjects: []v1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      helmInstall.ServiceAccount, // "tiller",
				Namespace: helmInstall.Namespace,
			}},
	}
	log.Info("create clusterrolebinding")

	err = backoff.Retry(func() error {
		if _, err := client.RbacV1().ClusterRoleBindings().Create(clusterRoleBinding); err != nil {
			if k8sutil.IsK8sErrorPermanent(err) {
				return backoff.MarkErrorPermanent(err)
			}
		}
		return nil
	}, backoffPolicy)

	if err != nil {
		return emperror.WrapWith(err, "could not create clusterrolebinding", "clusterrolebinding", clusterRoleBinding.Name)
	}

	return nil
}

// RetryHelmInstall retries for a configurable time/interval
// Azure AKS sometimes failing because of TLS handshake timeout, there are several issues on GitHub about that:
// https://github.com/Azure/AKS/issues/112, https://github.com/Azure/AKS/issues/116, https://github.com/Azure/AKS/issues/14
func RetryHelmInstall(log logrus.FieldLogger, helmInstall *phelm.Install, kubeconfig []byte) error {
	retryAttempts := viper.GetInt(phelm.HELM_RETRY_ATTEMPT_CONFIG)
	retrySleepSeconds := viper.GetInt(phelm.HELM_RETRY_SLEEP_SECONDS)
	for i := 0; i <= retryAttempts; i++ {
		log.Infof("Waiting %d/%d", i, retryAttempts)
		err := Install(log, helmInstall, kubeconfig)
		if err != nil {
			if strings.Contains(err.Error(), "net/http: TLS handshake timeout") {
				time.Sleep(time.Duration(retrySleepSeconds) * time.Second)
				continue
			}

			log.Warnln("error during installing tiller", err.Error())
		}
		return nil
	}
	return fmt.Errorf("timeout during helm install")
}

// Install uses Kubernetes client to install Tiller.
func Install(log logrus.FieldLogger, helmInstall *phelm.Install, kubeConfig []byte) error {

	err := PreInstall(log, helmInstall, kubeConfig)
	if err != nil {
		return err
	}

	opts := installer.Options{
		Namespace:                    helmInstall.Namespace,
		ServiceAccount:               helmInstall.ServiceAccount,
		UseCanary:                    helmInstall.Canary,
		ImageSpec:                    helmInstall.ImageSpec,
		MaxHistory:                   helmInstall.MaxHistory,
		AutoMountServiceAccountToken: true,
		ForceUpgrade:                 helmInstall.ForceUpgrade,
	}

	for i := range helmInstall.Tolerations {
		if helmInstall.Tolerations[i].Key != "" {
			opts.Values = append(opts.Values, fmt.Sprintf("spec.template.spec.tolerations[%d].key=%s", i, helmInstall.Tolerations[i].Key))
		}

		if helmInstall.Tolerations[i].Operator != "" {
			opts.Values = append(opts.Values, fmt.Sprintf("spec.template.spec.tolerations[%d].operator=%s", i, helmInstall.Tolerations[i].Operator))
		}

		if helmInstall.Tolerations[i].Value != "" {
			opts.Values = append(opts.Values, fmt.Sprintf("spec.template.spec.tolerations[%d].value=%s", i, helmInstall.Tolerations[i].Value))
		}

		if helmInstall.Tolerations[i].Effect != "" {
			opts.Values = append(opts.Values, fmt.Sprintf("spec.template.spec.tolerations[%d].effect=%s", i, helmInstall.Tolerations[i].Effect))
		}
	}

	if helmInstall.NodeAffinity != nil {
		for i := range helmInstall.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
			preferredSchedulingTerm := helmInstall.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution[i]

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

	kubeClient, err := k8sclient.NewClientFromKubeConfig(kubeConfig)
	if err != nil {
		return err
	}
	if err := installer.Install(kubeClient, &opts); err != nil {
		if !k8sapierrors.IsAlreadyExists(err) {
			// TODO shouldn'T we just skipp?
			return err
		}
		log.Info("Tiller already installed")
		if helmInstall.Upgrade {
			log.Info("upgrading Tiller")
			if err := installer.Upgrade(kubeClient, &opts); err != nil {
				return errors.Wrap(err, "error when upgrading")
			}
			log.Info("Tiller (the Helm server-side component) has been upgraded to the current version.")
		} else {
			log.Info("Warning: Tiller is already installed in the cluster.")
		}
	} else {
		log.Info("Tiller (the Helm server-side component) has been installed into your Kubernetes Cluster.")
	}
	log.Info("Helm install finished")
	return nil
}
