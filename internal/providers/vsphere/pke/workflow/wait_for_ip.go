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

	"emperror.dev/errors"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/cadence/activity"
)

// WaitForIPActivityName is the default registration name of the activity
const WaitForIPActivityName = "pke-vsphere-wait-for-ip"

// WaitForIPActivity represents an activity for creating a vSphere virtual machine
type WaitForIPActivity struct {
	vmomiClientFactory *VMOMIClientFactory
}

// MakeWaitForIPActivity returns a new WaitForIPActivity
func MakeWaitForIPActivity(vmomiClientFactory *VMOMIClientFactory) WaitForIPActivity {
	return WaitForIPActivity{
		vmomiClientFactory: vmomiClientFactory,
	}
}

// WaitForIPActivityInput represents the input needed for executing a WaitForIPActivity
type WaitForIPActivityInput struct {
	Ref            types.ManagedObjectReference
	OrganizationID uint
	SecretID       string
	ClusterName    string
}

// Execute performs the activity
func (a WaitForIPActivity) Execute(ctx context.Context, input WaitForIPActivityInput) (string, error) {
	_ = activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"secret", input.SecretID,
	)
	client, err := a.vmomiClientFactory.New(input.OrganizationID, input.SecretID)
	if err != nil {
		return "", errors.WrapIf(err, "failed to create VMOMI client")
	}

	vm := object.NewVirtualMachine(client.Client, input.Ref)

	return vm.WaitForIP(ctx, true)
}
