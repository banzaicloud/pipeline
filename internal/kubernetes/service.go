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

package kubernetes

import (
	"context"

	"emperror.dev/errors"
	corev1 "k8s.io/api/core/v1"
	k8sapierrors "k8s.io/apimachinery/pkg/api/errors"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/pipeline/internal/common"
	"github.com/banzaicloud/pipeline/internal/helm"
)

// ConfigSecretGetter returns a config secret ID for a cluster.
type ConfigSecretGetter interface {
	// GetConfigSecretID returns a config secret ID for a cluster.
	GetConfigSecretID(ctx context.Context, clusterID uint) (string, error)
}

// ClusterService provides a thin access layer to clusters.
type ClusterService interface {
	// GetCluster retrieves the cluster representation based on the cluster identifier.
	GetCluster(ctx context.Context, clusterID uint) (*helm.Cluster, error)
}

// Service provides an interface for using clieng-go on a specific cluster.
type Service struct {
	configSecretGetter ConfigSecretGetter
	configFactory      ConfigFactory

	logger common.Logger
}

// NewService returns a new NewService.
func NewService(
	configSecretGetter ConfigSecretGetter,
	configFactory ConfigFactory,
	logger common.Logger,
) *Service {
	return &Service{
		configSecretGetter: configSecretGetter,
		configFactory:      configFactory,

		logger: logger.WithFields(map[string]interface{}{"component": "kubernetes"}),
	}
}

// GetKubeConfig gets a kube config for a specific cluster.
func (s *Service) GetKubeConfig(ctx context.Context, clusterID uint) (*rest.Config, error) {
	secretID, err := s.configSecretGetter.GetConfigSecretID(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	config, err := s.configFactory.FromSecret(ctx, secretID)
	if err != nil {
		return nil, err
	}

	return config, nil
}

// GetObject gets an Object from a specific cluster.
func (s *Service) GetObject(ctx context.Context, clusterID uint, objRef corev1.ObjectReference, obj runtime.Object) error {
	kubeClient, err := s.newClientForCluster(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to create Kubernetes client")
	}

	return kubeClient.Get(ctx, client.ObjectKey{Namespace: objRef.Namespace, Name: objRef.Name}, obj)
}

// DeleteObject deletes an Object from a specific cluster.
func (s *Service) DeleteObject(ctx context.Context, clusterID uint, o runtime.Object) error {
	kubeClient, err := s.newClientForCluster(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to create Kubernetes client")
	}

	err = kubeClient.Delete(ctx, o)
	if err != nil && !k8sapierrors.IsNotFound(err) {
		return errors.WrapIf(err, "failed to delete Object")
	}

	return nil
}

// EnsureObject makes sure that a given Object is on the cluster and returns it.
func (s *Service) EnsureObject(ctx context.Context, clusterID uint, o runtime.Object) error {
	kubeClient, err := s.newClientForCluster(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to create Kubernetes client")
	}

	err = kubeClient.Create(ctx, o)
	if err != nil && !k8sapierrors.IsAlreadyExists(err) {
		return errors.WrapIf(err, "failed to create Object")
	}

	objectKey, err := client.ObjectKeyFromObject(o)
	if err != nil {
		return errors.WrapIf(err, "failed to create ObjectKey")
	}

	return kubeClient.Get(ctx, objectKey, o)
}

func (s *Service) newClientForCluster(ctx context.Context, clusterID uint) (client.Client, error) {
	kubeConfig, err := s.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	kubeClient, err := client.New(kubeConfig, client.Options{})
	if err != nil {
		return nil, errors.WrapIf(err, "failed to create Kubernetes client")
	}

	return kubeClient, nil
}

// List lists Objects a specific cluster.
func (s *Service) List(ctx context.Context, clusterID uint, labels map[string]string, obj runtime.Object) error {
	kubeClient, err := s.newClientForCluster(ctx, clusterID)
	if err != nil {
		return errors.WrapIf(err, "failed to create Kubernetes client")
	}
	return kubeClient.List(ctx, &client.ListOptions{
		LabelSelector: k8slabels.SelectorFromSet(labels),
	}, obj)
}
