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
	"github.com/vmware/govmomi/find"
	"go.uber.org/cadence/activity"
)

// GetPublicAddressActivityName is the default registration name of the activity
const GetPublicAddressActivityName = "pke-vsphere-get-public-address"

// GetPublicAddressActivity represents an activity to retrieve cluster public address
type GetPublicAddressActivity struct {
	vmomiClientFactory *VMOMIClientFactory
}

// MakeGetPublicAddressActivity returns a new GetPublicAddressActivity
func MakeGetPublicAddressActivity(vmomiClientFactory *VMOMIClientFactory) GetPublicAddressActivity {
	return GetPublicAddressActivity{
		vmomiClientFactory: vmomiClientFactory,
	}
}

// GetPublicAddressActivityInput represents the input needed for executing a GetPublicAddressActivity
type GetPublicAddressActivityInput struct {
	OrganizationID uint
	SecretID       string
	NodeName       string
}

// Execute performs the activity
// In case of success, it returns whether the given node existed or not.
func (a GetPublicAddressActivity) Execute(ctx context.Context, input GetPublicAddressActivityInput) (ipAddress string, err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"secret", input.SecretID,
		"nodeName", input.NodeName,
	)

	c, err := a.vmomiClientFactory.New(input.OrganizationID, input.SecretID)
	if err = errors.WrapIf(err, "failed to create cloud connection"); err != nil {
		return "", err
	}

	finder := find.NewFinder(c.Client)
	vms, err := finder.VirtualMachineList(ctx, input.NodeName)
	if err != nil {
		logger.Warnf("couldn't find a VM named %q: %s", input.NodeName, err.Error())
		return "", nil
	}
	if len(vms) != 1 {
		logger.Warnf("couldn't find a single VM named %q", input.NodeName)
		return "", nil
	}

	vm := vms[0]

	return vm.WaitForIP(ctx, true)
}
