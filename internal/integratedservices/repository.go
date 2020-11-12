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

package integratedservices

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"emperror.dev/errors"
	"github.com/mitchellh/mapstructure"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"
)

// NewInMemoryIntegratedServiceRepository returns a new in-memory integrated service repository.
func NewInMemoryIntegratedServiceRepository(integratedServices map[uint][]IntegratedService) *InMemoryIntegratedServiceRepository {
	lookup := make(map[uint]map[string]IntegratedService, len(integratedServices))
	for clID, fs := range integratedServices {
		m := make(map[string]IntegratedService, len(fs))
		lookup[clID] = m
		for _, f := range fs {
			m[f.Name] = f
		}
	}
	return &InMemoryIntegratedServiceRepository{
		integratedServices: lookup,
	}
}

// InMemoryIntegratedServiceRepository keeps integrated services in the memory.
// Use it in tests or for development/demo purposes.
type InMemoryIntegratedServiceRepository struct {
	integratedServices map[uint]map[string]IntegratedService

	mu sync.RWMutex
}

// GetIntegratedServices returns a list of all the integrated services stored in the repository for the specified cluster
func (r *InMemoryIntegratedServiceRepository) GetIntegratedServices(ctx context.Context, clusterID uint) ([]IntegratedService, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	integratedServices, ok := r.integratedServices[clusterID]
	if !ok {
		return nil, nil
	}

	fs := make([]IntegratedService, 0, len(integratedServices))

	for _, integratedService := range integratedServices {
		fs = append(fs, integratedService)
	}

	return fs, nil
}

// GetIntegratedService returns the integrated service identified by the parameters if it is in the repository, otherwise an error is returned
func (r *InMemoryIntegratedServiceRepository) GetIntegratedService(ctx context.Context, clusterID uint, integratedServiceName string) (IntegratedService, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if integratedServices, ok := r.integratedServices[clusterID]; ok {
		if integratedService, ok := integratedServices[integratedServiceName]; ok {
			return integratedService, nil
		}
	}

	return IntegratedService{}, integratedServiceNotFoundError{
		clusterID:             clusterID,
		integratedServiceName: integratedServiceName,
	}
}

// SaveIntegratedService persists the integrated service to the repository
func (r *InMemoryIntegratedServiceRepository) SaveIntegratedService(ctx context.Context, clusterID uint, integratedServiceName string, spec IntegratedServiceSpec, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	integratedServices, ok := r.integratedServices[clusterID]
	if !ok {
		integratedServices = make(map[string]IntegratedService)
		r.integratedServices[clusterID] = integratedServices
	}

	integratedServices[integratedServiceName] = IntegratedService{
		Name:   integratedServiceName,
		Spec:   spec,
		Status: status,
	}

	return nil
}

// UpdateIntegratedServiceStatus sets the integrated service's status
func (r *InMemoryIntegratedServiceRepository) UpdateIntegratedServiceStatus(ctx context.Context, clusterID uint, integratedServiceName string, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if integratedServices, ok := r.integratedServices[clusterID]; ok {
		if integratedService, ok := integratedServices[integratedServiceName]; ok {
			integratedService.Status = status
			integratedServices[integratedServiceName] = integratedService
			return nil
		}
	}

	return integratedServiceNotFoundError{
		clusterID:             clusterID,
		integratedServiceName: integratedServiceName,
	}
}

// UpdateIntegratedServiceSpec sets the integrated service's specification
func (r *InMemoryIntegratedServiceRepository) UpdateIntegratedServiceSpec(ctx context.Context, clusterID uint, integratedServiceName string, spec IntegratedServiceSpec) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if integratedServices, ok := r.integratedServices[clusterID]; ok {
		if integratedService, ok := integratedServices[integratedServiceName]; ok {
			integratedService.Spec = spec
			integratedServices[integratedServiceName] = integratedService
			return nil
		}
	}

	return integratedServiceNotFoundError{
		clusterID:             clusterID,
		integratedServiceName: integratedServiceName,
	}
}

// DeleteIntegratedService removes the integrated service from the repository.
// It is an idempotent operation.
func (r *InMemoryIntegratedServiceRepository) DeleteIntegratedService(ctx context.Context, clusterID uint, integratedServiceName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if integratedServices, ok := r.integratedServices[clusterID]; ok {
		delete(integratedServices, integratedServiceName)
	}

	return nil
}

// Clear removes every entry from the repository
func (r *InMemoryIntegratedServiceRepository) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.integratedServices = make(map[uint]map[string]IntegratedService)
}

// Snapshot returns a snapshot of the repository's state that can be restored later
func (r *InMemoryIntegratedServiceRepository) Snapshot() map[uint]map[string]IntegratedService {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return copyClusterLookup(r.integratedServices)
}

// Restore sets the repository's state from a snapshot
func (r *InMemoryIntegratedServiceRepository) Restore(snapshot map[uint]map[string]IntegratedService) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.integratedServices = copyClusterLookup(snapshot)
}

func copyIntegratedServiceLookup(lookup map[string]IntegratedService) map[string]IntegratedService {
	if lookup == nil {
		return nil
	}
	result := make(map[string]IntegratedService, len(lookup))
	for n, f := range lookup {
		result[n] = f
	}
	return result
}

func copyClusterLookup(lookup map[uint]map[string]IntegratedService) map[uint]map[string]IntegratedService {
	if lookup == nil {
		return nil
	}
	result := make(map[uint]map[string]IntegratedService, len(lookup))
	for c, fs := range lookup {
		result[c] = copyIntegratedServiceLookup(fs)
	}
	return result
}

type integratedServiceNotFoundError struct {
	clusterID             uint
	integratedServiceName string
}

func (e integratedServiceNotFoundError) Error() string {
	return fmt.Sprintf("IntegratedService %q not found for cluster %d.", e.integratedServiceName, e.clusterID)
}

func (e integratedServiceNotFoundError) Details() []interface{} {
	return []interface{}{
		"clusterId", e.clusterID,
		"integrated service", e.integratedServiceName,
	}
}

func (integratedServiceNotFoundError) IntegratedServiceNotFound() bool {
	return true
}

type ClusterKubeConfigFunc func(ctx context.Context, clusterID uint) ([]byte, error)

func (c ClusterKubeConfigFunc) GetKubeConfig(ctx context.Context, clusterID uint) ([]byte, error) {
	return c(ctx, clusterID)
}

// clusterRepository repository implementation that directly accesses a cluster for resource operations
// TODO move this into the adapter package?
type clusterRepository struct {
	scheme       *runtime.Scheme
	kubeConfigFn ClusterKubeConfigFunc
}

// Creates a new cluster repository to access Integrated services in a k8s cluster
func NewClusterRepository(kubeConfigFn ClusterKubeConfigFunc) IntegratedServiceRepository {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)

	return clusterRepository{
		scheme:       scheme,
		kubeConfigFn: kubeConfigFn,
	}
}

func (c clusterRepository) GetIntegratedServices(ctx context.Context, clusterID uint) ([]IntegratedService, error) {
	client, err := c.k8sClientForCluster(ctx, clusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build cluster client")
	}

	lookupISvcs := &v1alpha1.ServiceInstanceList{}
	if err := client.List(ctx, lookupISvcs); err != nil {
		return nil, errors.Wrap(err, "failed to retrieve integrated service list")
	}

	iSvcs := make([]IntegratedService, 0, len(lookupISvcs.Items))
	for _, si := range lookupISvcs.Items {
		iSvcs = append(iSvcs, c.transform(si))
	}

	return iSvcs, nil
}

func (c clusterRepository) GetIntegratedService(ctx context.Context, clusterID uint, integratedServiceName string) (IntegratedService, error) {
	emptyIS := IntegratedService{}

	clusterClient, err := c.k8sClientForCluster(ctx, clusterID)
	if err != nil {
		return emptyIS, errors.Wrap(err, "failed to build cluster client")
	}

	lookupSI := &v1alpha1.ServiceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "external-dns",
			Namespace: "default", // TODO infer the proper namespace

		},
	}
	key, okErr := client.ObjectKeyFromObject(lookupSI)
	if okErr != nil {
		return emptyIS, errors.Wrap(err, "failed to get object key for lookup")
	}

	if err := clusterClient.Get(ctx, key, lookupSI); err != nil {
		return emptyIS, errors.Wrap(err, "failed to look up service instance")
	}

	return c.transform(*lookupSI), nil
}

func (c clusterRepository) SaveIntegratedService(ctx context.Context, clusterID uint, integratedServiceName string, spec IntegratedServiceSpec, status string) error {
	// NO op
	return nil
}

func (c clusterRepository) UpdateIntegratedServiceStatus(ctx context.Context, clusterID uint, integratedServiceName string, status string) error {
	// NO op
	return nil
}

func (c clusterRepository) UpdateIntegratedServiceSpec(ctx context.Context, clusterID uint, integratedServiceName string, spec IntegratedServiceSpec) error {
	// NO op
	return nil
}

func (c clusterRepository) DeleteIntegratedService(ctx context.Context, clusterID uint, integratedServiceName string) error {
	// NO op
	return nil
}

// k8sClientForCluster builds a client that accesses the cluster
// TODO the built client should be a caching one? (revise this)
func (c clusterRepository) k8sClientForCluster(ctx context.Context, clusterID uint) (client.Client, error) {

	kubeConfig, err := c.kubeConfigFn.GetKubeConfig(ctx, clusterID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to retrieve the k8s config")
	}

	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create rest config from cluster configuration")
	}

	cli, err := client.New(restCfg, client.Options{Scheme: c.scheme})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create the client from rest configuration")
	}

	return cli, nil
}

func (c clusterRepository) transform(instance v1alpha1.ServiceInstance) IntegratedService {
	transformedIS := IntegratedService{}
	transformedIS.Name = instance.Name

	// unmarshal the config string into json
	cfgJSON := make(map[string]interface{})
	json.Unmarshal([]byte(instance.Spec.Config), &cfgJSON)

	cfgJsonMap := map[string]interface{}{
		"externalDns": cfgJSON,
	}

	mapstructure.Decode(cfgJsonMap, &transformedIS.Spec)
	//mapstructure.Decode(instance.Status, &transformedIS.Output)

	transformedIS.Status = c.getISStatus(instance)

	return transformedIS
}

func (c clusterRepository) getISStatus(instance v1alpha1.ServiceInstance) IntegratedServiceStatus {
	// TODO transform all phases
	switch instance.Status.Phase {
	case v1alpha1.Installed:
		return IntegratedServiceStatusActive
	}
	return IntegratedServiceStatusError
}
