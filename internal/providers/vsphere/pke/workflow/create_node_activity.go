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
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/cadence/activity"

	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow/pkeworkflowadapter"
)

// CreateNodeActivityName is the default registration name of the activity
const CreateNodeActivityName = "pke-vsphere-create-node"

// CreateNodeActivity represents an activity for creating a vSphere virtual machine
type CreateNodeActivity struct {
	vmomiClientFactory *VMOMIClientFactory
	tokenGenerator     pkeworkflowadapter.TokenGenerator
}

// MakeCreateNodeActivity returns a new CreateNodeActivity
func MakeCreateNodeActivity(vmomiClientFactory *VMOMIClientFactory, tokenGenerator pkeworkflowadapter.TokenGenerator) CreateNodeActivity {
	return CreateNodeActivity{
		vmomiClientFactory: vmomiClientFactory,
		tokenGenerator:     tokenGenerator,
	}
}

// CreateNodeActivityInput represents the input needed for executing a CreateNodeActivity
type CreateNodeActivityInput struct {
	OrganizationID   uint
	ClusterID        uint
	SecretID         string
	ClusterName      string
	ResourcePoolName string
	FolderName       string
	DatastoreName    string
	//HTTPProxy         intPKEWorkflow.HTTPProxy
	Node
}

// Node represents a vSphere virtual machine
type Node struct {
	AdminUsername          string
	VCPU                   int
	RamMB                  int
	Name                   string
	SSHPublicKey           string
	UserDataScriptParams   map[string]string
	UserDataScriptTemplate string
	TemplateName           string
	NodePoolName           string
	Master                 bool
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

	vmConfig := types.VirtualMachineConfigSpec{}
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

	logger.Info("create virtual machine")

	vmRef := types.ManagedObjectReference{}

	_, token, err := a.tokenGenerator.GenerateClusterToken(input.OrganizationID, input.ClusterID)
	if err != nil {
		return vmRef, err
	}
	input.UserDataScriptParams["PipelineToken"] = token

	vmConfig, err := generateVMConfigs(input)
	if err != nil {
		return vmRef, err
	}

	cloneSpec := types.VirtualMachineCloneSpec{
		Config:  vmConfig,
		PowerOn: true,
	}

	c, err := a.vmomiClientFactory.New(input.OrganizationID, input.SecretID)
	if err = errors.WrapIf(err, "failed to create cloud connection"); err != nil {
		return vmRef, err
	}

	finder := find.NewFinder(c.Client)
	folder, err := finder.FolderOrDefault(ctx, input.FolderName)
	if err != nil {
		return vmRef, err
	}
	folderRef := folder.Reference()

	template, err := finder.VirtualMachine(ctx, input.TemplateName)
	if err != nil {
		return vmRef, err
	}
	templateRef := template.Reference()

	pool, err := finder.ResourcePoolOrDefault(ctx, input.ResourcePoolName)
	if err != nil {
		return vmRef, err
	}

	poolRef := pool.Reference()
	cloneSpec.Location.Pool = &poolRef

	ds, err := finder.DatastoreOrDefault(ctx, input.DatastoreName)
	if err == nil {
		dsRef := ds.Reference()
		cloneSpec.Location.Datastore = &dsRef
	} else {
		if _, ok := err.(*find.NotFoundError); !ok {
			return vmRef, err
		}

		logger.Debugf("ds %s not found, fallback to drs", input.DatastoreName)
		storagePod, err := finder.DatastoreCluster(ctx, input.DatastoreName)
		if err != nil {
			if _, ok := err.(*find.NotFoundError); ok {
				return vmRef, fmt.Errorf("neither a datastore nor a datastore cluster named %q found", input.DatastoreName)
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

	logger.Info("cloning template task: ", task.String())
	progressLogger := newProgressLogger("cloning template progress ", logger)
	defer progressLogger.Wait()
	taskInfo, err := task.WaitForResult(ctx, progressLogger)
	if err != nil {
		return vmRef, err
	}

	logger.Infof("vm created: %+v\n", taskInfo)

	if ref, ok := taskInfo.Result.(types.ManagedObjectReference); ok {
		vmRef = ref
	}
	return vmRef, nil
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
				"ssh-authorized-keys": []string{publicKey}}}
	}

	out, _ := yaml.Marshal(data)
	return "#cloud-config\n" + string(out)
}
