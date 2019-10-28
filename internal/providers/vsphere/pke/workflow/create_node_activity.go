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
	"github.com/banzaicloud/pipeline/internal/providers/pke/pkeworkflow/pkeworkflowadapter"
	"github.com/ghodss/yaml"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/cadence/activity"
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
}

// Execute performs the activity
func (a CreateNodeActivity) Execute(ctx context.Context, input CreateNodeActivityInput) error {
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

	logger.Info("create virtual machine")

	userDataScriptTemplate, err := template.New(input.Name + "UserDataScript").Parse(input.UserDataScriptTemplate)
	if err != nil {
		return err
	}

	_, token, err := a.tokenGenerator.GenerateClusterToken(input.OrganizationID, input.ClusterID)
	if err != nil {
		return err
	}

	input.UserDataScriptParams["PipelineToken"] = token

	var userDataScript strings.Builder
	err = userDataScriptTemplate.Execute(&userDataScript, input.UserDataScriptParams)
	if err = errors.WrapIf(err, "failed to execute user data script template"); err != nil {
		return err
	}

	c, err := a.vmomiClientFactory.New(input.OrganizationID, input.SecretID)
	if err = errors.WrapIf(err, "failed to create cloud connection"); err != nil {
		return err
	}

	userData := encodeGuestinfo(generateCloudConfig(input.AdminUsername, input.SSHPublicKey, userDataScript.String()))

	vmConfig := types.VirtualMachineConfigSpec{}
	vmConfig.ExtraConfig = append(vmConfig.ExtraConfig,
		&types.OptionValue{Key: "disk.enableUUID", Value: "true"},
		&types.OptionValue{Key: "guestinfo.userdata.encoding", Value: "gzip+base64"},
		&types.OptionValue{Key: "guestinfo.userdata", Value: userData},
	)
	cloneSpec := types.VirtualMachineCloneSpec{
		Config:  &vmConfig,
		PowerOn: true,
	}

	finder := find.NewFinder(c.Client)
	folder, err := finder.FolderOrDefault(ctx, input.FolderName)
	if err != nil {
		return err
	}
	folderRef := folder.Reference()

	template, err := finder.VirtualMachine(ctx, input.TemplateName)
	if err != nil {
		return err
	}
	templateRef := template.Reference()

	pool, err := finder.ResourcePoolOrDefault(ctx, input.ResourcePoolName)
	if err != nil {
		return err
	}

	poolRef := pool.Reference()
	cloneSpec.Location.Pool = &poolRef

	ds, err := finder.DatastoreOrDefault(ctx, input.DatastoreName)
	if err == nil {
		dsRef := ds.Reference()
		cloneSpec.Location.Datastore = &dsRef
	} else {
		if _, ok := err.(*find.NotFoundError); !ok {
			return err
		}

		logger.Debugf("ds %s not found, fallback to drs", input.DatastoreName)
		storagePod, err := finder.DatastoreCluster(ctx, input.DatastoreName)
		if err != nil {
			if _, ok := err.(*find.NotFoundError); ok {
				return fmt.Errorf("neither a datastore nor a datastore cluster named %q found", input.DatastoreName)
			}
			return err
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
			return err
		}

		if len(result.Recommendations) == 0 {
			return fmt.Errorf("no datastore-cluster recommendations")
		}

		cloneSpec.Location.Datastore = &result.Recommendations[0].Action[0].(*types.StoragePlacementAction).Destination
	}

	task, err := template.Clone(ctx, folder, input.Name, cloneSpec)
	if err != nil {
		return err
	}

	logger.Info("cloning template", "task", task.String())

	taskInfo, err := task.WaitForResult(ctx, nil)
	if err != nil {
		return err
	}

	logger.Infof("vm created: %+v\n", taskInfo)
	return nil
}

func encodeGuestinfo(data string) string {
	buffer := new(bytes.Buffer)
	encoder := base64.NewEncoder(base64.StdEncoding, buffer)
	compressor := gzip.NewWriter(encoder)

	compressor.Write([]byte(data))

	compressor.Close()
	encoder.Close()

	return buffer.String()
}

func generateCloudConfig(user, publicKey, script string) string {

	data := map[string]interface{}{
		"runcmd": []string{script},
	}

	if publicKey != "" {
		if user == "" {
			user = "banzaicloud"
		}
		data["users"] = []map[string]interface{}{
			map[string]interface{}{
				"name":                user,
				"ssh-authorized-keys": []string{publicKey}}}
	}

	out, _ := yaml.Marshal(data)
	return "#cloud-config\n" + string(out)
}
