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
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow/pkeworkflowadapter"
	"github.com/vmware/govmomi/find"
	"go.uber.org/cadence/activity"
)

// DeleteNodeActivityName is the default registration name of the activity
const DeleteNodeActivityName = "pke-vsphere-delete-node"

// DeleteNodeActivity represents an activity for creating a vSphere virtual machine
type DeleteNodeActivity struct {
	vmomiClientFactory *VMOMIClientFactory
	tokenGenerator     pkeworkflowadapter.TokenGenerator
}

// MakeDeleteNodeActivity returns a new DeleteNodeActivity
func MakeDeleteNodeActivity(vmomiClientFactory *VMOMIClientFactory) DeleteNodeActivity {
	return DeleteNodeActivity{
		vmomiClientFactory: vmomiClientFactory,
	}
}

// DeleteNodeActivityInput represents the input needed for executing a DeleteNodeActivity
type DeleteNodeActivityInput struct {
	OrganizationID uint
	ClusterID      uint
	SecretID       string
	ClusterName    string
	Node
}

// Execute performs the activity
// In case of success, it returns whether the given node existed or not.
func (a DeleteNodeActivity) Execute(ctx context.Context, input DeleteNodeActivityInput) (existed bool, err error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"secret", input.SecretID,
		"node", input.Name,
	)

	/*keyvals := []interface{}{
		"cluster", input.ClusterName,
		"node", input.Node.Name,
	}*/

	c, err := a.vmomiClientFactory.New(input.OrganizationID, input.SecretID)
	if err = errors.WrapIf(err, "failed to create cloud connection"); err != nil {
		return true, err
	}

	finder := find.NewFinder(c.Client)
	vms, err := finder.VirtualMachineList(ctx, input.Name)
	if err != nil  {
		logger.Warnf("couldn't find a VM named %q: %s", input.Name, err.Error())
		return false, nil
	}
	if len(vms) != 1 {
		logger.Warnf("couldn't find a single VM named %q", input.Name)
		return false, nil
	}

	// TODO check tags
	//config, err := vm.QueryConfigTarget()
	//expectedTags := getClusterTags(input.Name, input.NodePoolName)

	vm := vms[0]
	task, err := vm.PowerOff(ctx)
	if err != nil {
		return true, errors.WrapIf(err, "failed to power off VM")
	}

	// power off
	logger.Info("wait for power off VM", "task", task.String())
	//err = vm.WaitForPowerState(ctx, types.VirtualMachinePowerStatePoweredOff)
	//if err != nil {
	//	return false, errors.WrapIf(err, "failed to power off VM")
	//}
	
	taskInfo, err := task.WaitForResult(ctx, nil)
	if err != nil {
		return true, err
	}
	logger.Infof("vm powered off: %+v\n", taskInfo)

	// destroy
	task, err = vm.Destroy(ctx)
	if err != nil {
		return true, errors.WrapIf(err, "failed to destroy VM")
	}
	logger.Info("destroying VM", "task", task.String())
	progressLogger := newProgressLogger("destroying VM - progress ", logger)
	defer progressLogger.Wait()
	taskInfo, err = task.WaitForResult(ctx, progressLogger)
	if err != nil {
		return true, err
	}
	logger.Infof("vm deleted: %+v\n", taskInfo)
	return true, nil
}
