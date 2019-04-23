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

package k8sutil

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

// GetOrCreateClusterRole gets the cluster role with the given name if exists otherwise creates new one and returns it
func GetOrCreateClusterRole(log logrus.FieldLogger, client *kubernetes.Clientset, name string, rules []rbacv1.PolicyRule) (*rbacv1.ClusterRole, error) {
	fieldSelector := fields.SelectorFromSet(fields.Set{"metadata.name": name})

	clusterRoles, err := client.RbacV1().ClusterRoles().List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
	if err != nil {
		log.Errorf("querying cluster roles failed: %s", err.Error())
		return nil, err
	}

	if len(clusterRoles.Items) > 1 {
		log.Errorf("duplicate cluster role with name %q found", name)
		return nil, fmt.Errorf("duplicate cluster role with name %q found", name)
	}

	if len(clusterRoles.Items) == 1 {
		log.Infof("cluster role %q already exists", name)
		return &clusterRoles.Items[0], nil
	}

	clusterRole, err := client.RbacV1().ClusterRoles().Create(
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Rules: rules,
		})

	if err != nil {
		log.Errorf("creating cluster role %q failed: %s", name, err.Error())
		return nil, err
	}

	log.Infof("cluster role %q created", name)

	return clusterRole, nil
}

// GetOrCreateServiceAccount checks is service account with given name exists in the specified namespace and returns it.
// if it doesn't exists it creates a new one and returns it to the caller.
func GetOrCreateServiceAccount(log logrus.FieldLogger, client *kubernetes.Clientset, namespace, name string) (*v1.ServiceAccount, error) {
	fieldSelector := fields.SelectorFromSet(fields.Set{"metadata.name": name})

	serviceAccounts, err := client.CoreV1().ServiceAccounts(namespace).List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
	if err != nil {
		log.Errorf("querying service accounts in namespace %q failed: %s", namespace, err.Error())
		return nil, err
	}

	if len(serviceAccounts.Items) > 1 {
		log.Errorf("duplicate service account with '%s/%s' found ", namespace, name)
		return nil, fmt.Errorf("duplicate service account with '%s/%s' found ", namespace, name)
	}

	if len(serviceAccounts.Items) == 1 {
		log.Infof("service account '%s/%s' already exists", namespace, name)
		return &serviceAccounts.Items[0], nil
	}

	serviceAccount, err := client.CoreV1().ServiceAccounts(namespace).Create(
		&v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
		})

	if err != nil {
		log.Errorf("creating service account '%s/%s' failed: %s", namespace, name, err.Error())
		return nil, err
	}

	log.Infof("service account '%s/%s' created", namespace, name)

	return serviceAccount, nil
}

// GetOrCreateClusterRoleBinding creates the cluster role binding given its name, service account and cluster role if not exists.
// It returns the found cluster role binding if one already exists or the newly created one.
func GetOrCreateClusterRoleBinding(log logrus.FieldLogger,
	client *kubernetes.Clientset,
	name string, serviceAccount *v1.ServiceAccount,
	clusterRole *rbacv1.ClusterRole) (*rbacv1.ClusterRoleBinding, error) {
	fieldSelector := fields.SelectorFromSet(fields.Set{"metadata.name": name})

	clusterRoleBindings, err := client.RbacV1().ClusterRoleBindings().List(metav1.ListOptions{FieldSelector: fieldSelector.String()})
	if err != nil {
		log.Errorf("querying cluster role bindings failed: %s", err.Error())
		return nil, err
	}

	if len(clusterRoleBindings.Items) > 1 {
		log.Errorf("duplicate cluster role binding with name %q found", name)
		return nil, fmt.Errorf("duplicate cluster role binding with name %q found", name)
	}

	if len(clusterRoleBindings.Items) == 1 {
		log.Infof("cluster role binding %q already exists", name)
		return &clusterRoleBindings.Items[0], nil
	}

	clusterRoleBinding, err := client.RbacV1().ClusterRoleBindings().Create(
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      serviceAccount.Name,
					Namespace: serviceAccount.Namespace,
				},
			},
			RoleRef: rbacv1.RoleRef{
				Kind: "ClusterRole",
				Name: clusterRole.Name,
			},
		})

	if err != nil {
		log.Errorf("creating cluster role binding %q failed: %s", name, err.Error())
		return nil, err
	}

	log.Infof("cluster role binding %q created", name)

	return clusterRoleBinding, nil
}

const (
	int64QuantityExpectedBytes = 18
)

func FormatResourceQuantity(resourceName v1.ResourceName, q *resource.Quantity) string {
	if resourceName == v1.ResourceCPU {
		return formatCPUQuantity(q)
	}
	return formatQuantity(q)
}

func GetResourceQuantityInBytes(q *resource.Quantity) int {
	if q.IsZero() {
		return 0
	}

	result := make([]byte, 0, int64QuantityExpectedBytes)

	rounded, exact := q.AsScale(0)
	if !exact {
		return 0
	}
	number, exponent := rounded.AsCanonicalBase1024Bytes(result)

	i, err := strconv.Atoi(string(number))
	if err != nil {
		// this should never happen, but in case it happens we fallback to default string representation
		return 0
	}

	return int(float64(i) * math.Pow(1024, float64(exponent)))
}

func formatQuantity(q *resource.Quantity) string {

	if q.IsZero() {
		return "0"
	}

	result := make([]byte, 0, int64QuantityExpectedBytes)

	rounded, exact := q.AsScale(0)
	if !exact {
		return q.String()
	}
	number, exponent := rounded.AsCanonicalBase1024Bytes(result)

	i, err := strconv.Atoi(string(number))
	if err != nil {
		// this should never happen, but in case it happens we fallback to default string representation
		return q.String()
	}

	b := float64(i) * math.Pow(1024, float64(exponent))

	if b < 1000 {
		return fmt.Sprintf("%.2f B", b)
	}

	b = b / 1000
	if b < 1000 {
		return fmt.Sprintf("%.2f KB", b)
	}

	b = b / 1000
	if b < 1000 {
		return fmt.Sprintf("%.2f MB", b)
	}

	b = b / 1000
	return fmt.Sprintf("%.2f GB", b)
}

func formatCPUQuantity(q *resource.Quantity) string {

	if q.IsZero() {
		return "0"
	}

	result := make([]byte, 0, int64QuantityExpectedBytes)
	number, suffix := q.CanonicalizeBytes(result)
	if string(suffix) == "m" {
		// the suffix m to mean mili. For example 100m cpu is 100 milicpu, and is the same as 0.1 cpu.
		i, err := strconv.Atoi(string(number))
		if err != nil {
			// this should never happen, but in case it happens we fallback to default string representation
			return q.String()
		}

		if i < 1000 {
			return fmt.Sprintf("%s mCPU", string(number))
		}

		f := float64(i) / 1000
		return fmt.Sprintf("%.2f CPU", f)
	}

	return fmt.Sprintf("%s CPU", string(number))

}

// IsK8sErrorPermanent checks if the given error is permanent error or not
func IsK8sErrorPermanent(err error) bool {
	if k8sapierrors.IsAlreadyExists(err) {
		return false
	} else if strings.Contains(err.Error(), "etcdserver: request timed out") {
		return false // Newly instantiated AKS cluster's ETCD is flaky
	} else if strings.Contains(err.Error(), "connection refused") {
		return false
	}

	return true
}
