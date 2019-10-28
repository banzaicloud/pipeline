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
	"net/url"

	"emperror.dev/errors"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/soap"
)

type VMOMIClientFactory struct {
	secretStore pkeworkflow.SecretStore
}

func NewVMOMIClientFactory(secretStore pkeworkflow.SecretStore) *VMOMIClientFactory {
	return &VMOMIClientFactory{secretStore: secretStore}
}

func (f *VMOMIClientFactory) New(organizationID uint, secretID string) (*govmomi.Client, error) {
	s, err := f.secretStore.GetSecret(organizationID, secretID)
	if err != nil {
		return nil, errors.WrapIf(err, "failed to get secret")
	}

	if err := s.ValidateSecretType(secrettype.Vsphere); err != nil {
		return nil, err
	}

	values := s.GetValues()

	u, err := soap.ParseURL(values[secrettype.VsphereURL])
	if err != nil {
		return nil, err
	}

	u.User = url.UserPassword(values[secrettype.VsphereUser], values[secrettype.VspherePassword])

	ctx := context.TODO()

	// Connect and log in to ESX or vCenter
	return govmomi.NewClient(ctx, u, true)
}
