// Copyright © 2018 Banzai Cloud
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

package action

import (
	"encoding/json"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/sirupsen/logrus"
)

// DeleteSSHKeyAction describes how to delete an SSH key
type DeleteSSHKeyAction struct {
	context        *ACSKClusterDeleteContext
	sshKeyName     string
	sshKeyRegionID string
	log            logrus.FieldLogger
}

// NewDeleteSSHKeyAction creates a new UploadSSHKeyAction
func NewDeleteSSHKeyAction(log logrus.FieldLogger, context *ACSKClusterDeleteContext, sshKeyName, regionID string) *DeleteSSHKeyAction {
	return &DeleteSSHKeyAction{
		context:        context,
		sshKeyName:     sshKeyName,
		sshKeyRegionID: regionID,
		log:            log,
	}
}

// GetName returns the name of this DeleteSSHKeyAction
func (a *DeleteSSHKeyAction) GetName() string {
	return "DeleteSSHKeyAction"
}

// ExecuteAction executes this UploadSSHKeyAction
func (a *DeleteSSHKeyAction) ExecuteAction(input interface{}) (interface{}, error) {
	a.log.Info("EXECUTE DeleteSSHKeyAction")
	ecsClient := a.context.ECSClient

	req := ecs.CreateDeleteKeyPairsRequest()
	req.SetScheme(requests.HTTPS)
	jsonData := []string{a.sshKeyName}
	marshaledValue, err := json.Marshal(jsonData)
	if err != nil {
		return nil, err
	}
	req.KeyPairNames = string(marshaledValue)
	req.RegionId = a.sshKeyRegionID

	return ecsClient.DeleteKeyPairs(req)
}
