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
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"text/template"

	"emperror.dev/errors"
	"github.com/ghodss/yaml"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/cadence/activity"
	"go.uber.org/zap"

	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow"
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow/pkeworkflowadapter"
	"github.com/banzaicloud/pipeline/internal/secret/secrettype"
)

// CreateNodeActivityName is the default registration name of the activity
const CreateNodeActivityName = "pke-vsphere-create-node"

// CreateNodeActivity represents an activity for creating a vSphere virtual machine
type CreateNodeActivity struct {
	vmomiClientFactory *VMOMIClientFactory
	tokenGenerator     pkeworkflowadapter.TokenGenerator
	secretStore        pkeworkflow.SecretStore
}

// MakeCreateNodeActivity returns a new CreateNodeActivity
func MakeCreateNodeActivity(vmomiClientFactory *VMOMIClientFactory, tokenGenerator pkeworkflowadapter.TokenGenerator, secretStore pkeworkflow.SecretStore) CreateNodeActivity {
	return CreateNodeActivity{
		vmomiClientFactory: vmomiClientFactory,
		tokenGenerator:     tokenGenerator,
		secretStore:        secretStore,
	}
}

// CreateNodeActivityInput represents the input needed for executing a CreateNodeActivity
type CreateNodeActivityInput struct {
	OrganizationID   uint
	ClusterID        uint
	SecretID         string
	StorageSecretID  string
	ClusterName      string
	ResourcePoolName string
	FolderName       string
	DatastoreName    string
	Node
}

// Node represents a vSphere virtual machine
type Node struct {
	AdminUsername          string
	VCPU                   int
	RAM                    int // MiB
	Name                   string
	SSHPublicKey           string
	UserDataScriptParams   map[string]string
	UserDataScriptTemplate string
	TemplateName           string
	NodePoolName           string
	Master                 bool
}

func ensureVMIsRunning(ctx context.Context, logger *zap.SugaredLogger, vm *object.VirtualMachine) error {
	powerState, err := vm.PowerState(ctx)
	if err != nil {
		return err
	}
	logger.Infof("VM named %q found, power state: %q", vm.Name(), powerState)

	if powerState != types.VirtualMachinePowerStatePoweredOn {
		task, err := vm.PowerOn(ctx)
		if err != nil {
			return errors.WrapIf(err, "failed to power on VM")
		}

		logger.Info("wait for VM to power on", "task", task.String())
		err = vm.WaitForPowerState(ctx, types.VirtualMachinePowerStatePoweredOn)
		if err != nil {
			return errors.WrapIf(err, "failed to power on VM")
		}
	}
	return nil
}

func generateVMConfigs(input CreateNodeActivityInput) (*types.VirtualMachineConfigSpec, error) {
	userDataScriptTemplate, err := template.New(input.Name + "UserDataScript").Parse(input.UserDataScriptTemplate)
	if err != nil {
		return nil, err
	}

	var userDataScript strings.Builder
	err = userDataScriptTemplate.Execute(&userDataScript, input.UserDataScriptParams)
	if err = errors.WrapIf(err, "failed to execute user data script template"); err != nil {
		return nil, err
	}

	userData, err := encodeGuestInfo(generateCloudConfig(input.AdminUsername, input.SSHPublicKey, userDataScript.String(), input.Name))
	if err = errors.WrapIf(err, "failed to encode user data"); err != nil {
		return nil, err
	}

	vmConfig := types.VirtualMachineConfigSpec{
		NumCPUs:           int32(input.VCPU),
		NumCoresPerSocket: int32(input.VCPU),
		MemoryMB:          int64(input.RAM),
	}
	vmConfig.ExtraConfig = append(vmConfig.ExtraConfig,
		&types.OptionValue{Key: "disk.enableUUID", Value: "true"}, // needed for pv mounting
		&types.OptionValue{Key: "guestinfo.userdata.encoding", Value: "gzip+base64"},
		&types.OptionValue{Key: "guestinfo.userdata", Value: userData},
	)

	tags := getClusterTags(input.Name, input.NodePoolName)
	for key := range tags {
		vmConfig.ExtraConfig = append(vmConfig.ExtraConfig,
			&types.OptionValue{Key: fmt.Sprintf("guestinfo.%s", key), Value: tags[key]},
		)
	}

	return &vmConfig, nil
}

// Execute performs the activity
func (a CreateNodeActivity) Execute(ctx context.Context, input CreateNodeActivityInput) (types.ManagedObjectReference, error) {
	logger := activity.GetLogger(ctx).Sugar().With(
		"organization", input.OrganizationID,
		"cluster", input.ClusterName,
		"secret", input.SecretID,
		"node", input.Name,
	)
	vmRef := types.ManagedObjectReference{}

	c, err := a.vmomiClientFactory.New(input.OrganizationID, input.SecretID)
	if err = errors.WrapIf(err, "failed to create cloud connection"); err != nil {
		return vmRef, err
	}
	finder := find.NewFinder(c.Client)

	// first check if VM already exists and is in right power state
	vms, err := finder.VirtualMachineList(ctx, input.Name)
	if err != nil {
		if _, ok := err.(*find.NotFoundError); ok {
			logger.Infof("VM named %q not found", input.Name)
		} else {
			logger.Warnf("couldn't find a VM named %q: %s", input.Name, err.Error())
		}
	} else if len(vms) > 0 {
		vm := vms[0]
		err = ensureVMIsRunning(ctx, logger, vm)
		if err != nil {
			return vm.Reference(), err
		}
		return vm.Reference(), nil
	}

	// create new VM
	logger.Info("create virtual machine")

	_, token, err := a.tokenGenerator.GenerateClusterToken(input.OrganizationID, input.ClusterID)
	if err != nil {
		return vmRef, err
	}
	input.UserDataScriptParams["PipelineToken"] = token

	// use storageSecretId if provided, or secretId otherwise for setting up storage params
	secretID := input.StorageSecretID
	if secretID == "" {
		secretID = input.SecretID
	}
	err = a.setStorageParamsFromSecret(input.OrganizationID, secretID, input.UserDataScriptParams)
	if err != nil {
		return vmRef, errors.WrapIf(err, "failed to set storage params")
	}

	s, err := a.secretStore.GetSecret(input.OrganizationID, input.SecretID)
	if err != nil {
		return vmRef, errors.WrapIf(err, "failed to get secret")
	}
	if err := s.ValidateSecretType(secrettype.Vsphere); err != nil {
		return vmRef, err
	}
	secretValues := s.GetValues()

	vmConfig, err := generateVMConfigs(input)
	if err != nil {
		return vmRef, err
	}

	cloneSpec := types.VirtualMachineCloneSpec{
		Config:  vmConfig,
		PowerOn: true,
	}

	folderName := input.FolderName
	if folderName == "" {
		folderName =
			secretValues[secrettype.VsphereFolder]
	}

	folder, err := finder.FolderOrDefault(ctx, folderName)
	if err != nil {
		return vmRef, err
	}
	folderRef := folder.Reference()

	templateName := input.TemplateName
	if templateName == "" {
		templateName = secretValues[secrettype.VsphereDefaultNodeTemplate]
	}
	template, err := finder.VirtualMachine(ctx, templateName)
	if err != nil {
		return vmRef, err
	}
	templateRef := template.Reference()

	resourcePoolName := input.ResourcePoolName
	if resourcePoolName == "" {
		resourcePoolName = secretValues[secrettype.VsphereResourcePool]
	}
	pool, err := finder.ResourcePoolOrDefault(ctx, resourcePoolName)
	if err != nil {
		return vmRef, err
	}

	poolRef := pool.Reference()
	cloneSpec.Location.Pool = &poolRef

	dataStoreName := input.DatastoreName
	if dataStoreName == "" {
		dataStoreName = secretValues[secrettype.VsphereDatastore]
	}
	ds, err := finder.DatastoreOrDefault(ctx, dataStoreName)
	if err == nil {
		dsRef := ds.Reference()
		cloneSpec.Location.Datastore = &dsRef
	} else {
		if _, ok := err.(*find.NotFoundError); !ok {
			return vmRef, err
		}

		logger.Debugf("ds %s not found, fallback to drs", dataStoreName)
		storagePod, err := finder.DatastoreCluster(ctx, dataStoreName)
		if err != nil {
			if _, ok := err.(*find.NotFoundError); ok {
				return vmRef, fmt.Errorf("neither a datastore nor a datastore cluster named %q found", dataStoreName)
			}
			return vmRef, err
		}

		storagePodRef := storagePod.Reference()

		podSelectionSpec := types.StorageDrsPodSelectionSpec{
			StoragePod: &storagePodRef,
		}

		storagePlacementSpec := types.StoragePlacementSpec{
			Folder:           &folderRef,
			Vm:               &templateRef,
			CloneName:        input.Name,
			CloneSpec:        &cloneSpec,
			PodSelectionSpec: podSelectionSpec,
			Type:             string(types.StoragePlacementSpecPlacementTypeClone),
		}

		storageResourceManager := object.NewStorageResourceManager(c.Client)
		result, err := storageResourceManager.RecommendDatastores(ctx, storagePlacementSpec)
		if err != nil {
			return vmRef, err
		}

		if len(result.Recommendations) == 0 {
			return vmRef, fmt.Errorf("no datastore-cluster recommendations")
		}

		cloneSpec.Location.Datastore = &result.Recommendations[0].Action[0].(*types.StoragePlacementAction).Destination
		logger.Infof("deploying to %q datastore based on recommendation", cloneSpec.Location.Datastore)
	}

	task, err := template.Clone(ctx, folder, input.Name, cloneSpec)
	if err != nil {
		return vmRef, err
	}

	logger.Info("cloning VM template task: ", task.String())
	progressLogger := newProgressLogger("cloning VM template progress ", logger)
	defer progressLogger.Wait()
	taskInfo, err := task.WaitForResult(ctx, progressLogger)
	if err != nil {
		return vmRef, err
	}

	logger.Infof("VM created: %+v\n", taskInfo)

	if ref, ok := taskInfo.Result.(types.ManagedObjectReference); ok {
		vmRef = ref
	}

	vm := object.NewVirtualMachine(c.Client, vmRef)
	err = ensureVMIsRunning(ctx, logger, vm)
	if err != nil {
		return vm.Reference(), err
	}

	return vmRef, nil
}

func (a CreateNodeActivity) setStorageParamsFromSecret(orgID uint, secretID string, userDataScriptParams map[string]string) error {
	s, err := a.secretStore.GetSecret(orgID, secretID)
	if err != nil {
		return errors.WrapIf(err, "failed to get secret")
	}

	if err := s.ValidateSecretType(secrettype.Vsphere); err != nil {
		return err
	}

	values := s.GetValues()

	u, err := soap.ParseURL(values[secrettype.VsphereURL])
	if err != nil {
		return err
	}
	uA := strings.Split(u.Host, ":")
	vCenterServer := uA[0]
	vCenterPort := "443"
	if len(uA) > 1 {
		vCenterPort = uA[1]
	}

	userDataScriptParams["VCenterServer"] = vCenterServer
	userDataScriptParams["VCenterPort"] = vCenterPort
	userDataScriptParams["VCenterFingerprint"] = values[secrettype.VsphereFingerprint]
	userDataScriptParams["Datacenter"] = values[secrettype.VsphereDatacenter]
	userDataScriptParams["Datastore"] = values[secrettype.VsphereDatastore]
	userDataScriptParams["ResourcePool"] = values[secrettype.VsphereResourcePool]
	userDataScriptParams["Folder"] = values[secrettype.VsphereFolder]
	userDataScriptParams["Username"] = values[secrettype.VsphereUser]
	userDataScriptParams["Password"] = values[secrettype.VspherePassword]

	return nil
}

func encodeGuestInfo(data string) (string, error) {
	buffer := new(bytes.Buffer)
	encoder := base64.NewEncoder(base64.StdEncoding, buffer)
	compressor := gzip.NewWriter(encoder)

	_, err := compressor.Write([]byte(data))
	if err != nil {
		return "", err
	}

	compressor.Close()
	encoder.Close()

	return buffer.String(), nil
}

func generateCloudConfig(user, publicKey, script, hostname string) string {
	data := map[string]interface{}{
		"hostname":          hostname,
		"fqdn":              hostname,
		"preserve_hostname": false,
		"runcmd":            []string{script},
	}

	if publicKey != "" {
		if user == "" {
			user = "banzaicloud"
		}
		data["users"] = []map[string]interface{}{
			{
				"name":                user,
				"sudo":                "ALL=(ALL) NOPASSWD:ALL",
				"ssh-authorized-keys": []string{publicKey},
			},
		}
	}

	out, _ := yaml.Marshal(data)
	return "#cloud-config\n" + string(out)
}
