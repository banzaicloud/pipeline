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

package dns

import (
	"context"
	"reflect"

	"emperror.dev/errors"
	"github.com/banzaicloud/integrated-service-sdk/api/v1alpha1"
	"github.com/banzaicloud/operator-tools/pkg/reconciler"
	"github.com/banzaicloud/operator-tools/pkg/utils"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/banzaicloud/pipeline/internal/integratedservices"
)

// Reconciler decouples creation of kubernetes resources (IS Cr-s
type Reconciler interface {
	// Reconcile creates and applies CRs to a cluster
	Reconcile(ctx context.Context, kubeConfig []byte, config Config, values []byte, spec integratedservices.IntegratedServiceSpec) error
}

// isvcReconciler components struct in charge for assembling the CR manifest  and applying it to a cluster (by delegating to a cluster client)
type isvcReconciler struct {
	scheme *runtime.Scheme
}

// NewISReconciler builds an integrated service reconciler
func NewISReconciler() Reconciler {
	// register needed shemes
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	return isvcReconciler{
		scheme: scheme,
	}
}

func (is isvcReconciler) Reconcile(ctx context.Context, kubeConfig []byte, config Config, values []byte, spec integratedservices.IntegratedServiceSpec) error {
	si := &v1alpha1.ServiceInstance{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "external-dns",
		},
		Spec: v1alpha1.ServiceInstanceSpec{
			Service: "external-dns",
			Config:  string(values), // TODO to be verified (is it properly encoded)
		},
	}

	restCfg, err := clientcmd.RESTConfigFromKubeConfig(kubeConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create rest config from cluster configuration")
	}

	cli, err := client.New(restCfg, client.Options{Scheme: is.scheme})
	if err != nil {
		return errors.Wrap(err, "failed to create the client from rest configuration")
	}

	key, okErr := client.ObjectKeyFromObject(si)
	if okErr != nil {
		return errors.Wrap(err, "failed to get object key for lookup")
	}
	lookupSI := &v1alpha1.ServiceInstance{}
	if err := cli.Get(ctx, key, lookupSI); err != nil {
		if !errors2.IsNotFound(err) {
			return errors.Wrap(err, "failed to look up service instance")
		}
	}

	resourceReconciler := reconciler.NewReconcilerWith(cli)
	if reflect.DeepEqual(lookupSI, &v1alpha1.ServiceInstance{}) {
		// the service instance is not found, should be created
		_, _, err := resourceReconciler.CreateIfNotExist(si, reconciler.StateCreated)
		if err != nil {
			return errors.Wrap(err, "failed to create the service instance resource")
		}

		// retrieve the newly created CR
		// TODO is it possible to "cast" the result of the above call?
		if err := cli.Get(ctx, key, lookupSI); err != nil {
			return errors.Wrap(err, "failed to look up service instance")
		}
	}

	// at this point we have the CR created and retrieved
	if lookupSI.Spec.Enabled == nil {
		si.Spec.Enabled = utils.BoolPointer(true)
		// TODO the latest version is wired here
		idx := len(lookupSI.Status.AvailableVersions)
		si.Spec.Version = lookupSI.Status.AvailableVersions[idx-1]
	}

	// TODO should we care of disabled services here? (lookupIS.Spec.Enabled == false case)
	if _, err := resourceReconciler.ReconcileResource(si, reconciler.StatePresent); err != nil {
		return errors.Wrap(err, "failed to reconcile the integrated service")
	}

	return nil
}
