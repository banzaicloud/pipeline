// Copyright © 2020 Banzai Cloud
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

package monitoring

import (
	"context"
	"fmt"

	"emperror.dev/errors"
	"github.com/Masterminds/semver/v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type (
	K8sClientFactory func() (kubernetes.Interface, error)
	Migrator         func(ctx context.Context, configFactory K8sClientFactory, namespace, oldChartVersion, newChartVersion string) error
)

func Migrate(ctx context.Context, clientFactory K8sClientFactory, namespace, oldChartVersion, newChartVersion string) error {
	referenceVersion, err := semver.NewVersion("8.13.0")
	if err != nil {
		return errors.WrapIf(err, "invalid reference version")
	}

	oldSemVer, err := semver.NewVersion(oldChartVersion)
	if err != nil {
		return errors.WrapIf(err, "invalid old chart version")
	}

	newSemVer, err := semver.NewVersion(newChartVersion)
	if err != nil {
		return errors.WrapIf(err, "invalid new chart version")
	}

	if oldSemVer.LessThan(referenceVersion) && newSemVer.GreaterThan(referenceVersion) {
		clientset, err := clientFactory()
		if err != nil {
			return errors.WrapIf(err, "unable to creates kubernetes config for migration")
		}

		ingresses, err := clientset.ExtensionsV1beta1().Ingresses(namespace).List(
			ctx,
			v1.ListOptions{
				LabelSelector: labels.SelectorFromSet(map[string]string{"release": prometheusOperatorReleaseName}).String(),
			})
		if err != nil {
			return errors.WrapIf(err, "unable to remove legacy ingresses")
		}
		for _, i := range ingresses.Items {
			err = clientset.ExtensionsV1beta1().Ingresses(namespace).Delete(ctx, i.Name, v1.DeleteOptions{})
			if err != nil {
				if apierrors.IsNotFound(err) {
					continue
				}
				return errors.WrapIf(err, "unable to remove ingress")
			}
		}
		err = clientset.AppsV1().Deployments(namespace).Delete(ctx, fmt.Sprintf("%s-grafana", prometheusOperatorReleaseName), v1.DeleteOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return errors.WrapIf(err, "unable to remove grafana deployment")
			}
		}
	}
	return nil
}
