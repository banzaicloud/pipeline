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

import "github.com/sirupsen/logrus"

// DeleteACSKNodePoolAction describes the properties of an Alibaba cluster deletion
type DeleteACSKNodePoolAction struct {
	context *ACSKClusterDeleteContext
	log     logrus.FieldLogger
}

// NewDeleteACSKNodePoolAction create  a new DeleteACSKNodePoolAction
func NewDeleteACSKNodePoolAction(log logrus.FieldLogger, creationContext *ACSKClusterDeleteContext) *DeleteACSKNodePoolAction {
	return &DeleteACSKNodePoolAction{
		context: creationContext,
		log:     log,
	}
}

// GetName returns the name of this DeleteACSKNodePoolAction
func (a *DeleteACSKNodePoolAction) GetName() string {
	return "DeleteACSKNodePoolAction"
}

// ExecuteAction executes this DeleteACSKNodePoolAction
func (a *DeleteACSKNodePoolAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	a.log.Info("EXECUTE DeleteNodePoolAction")
	return nil, deleteNodepools(a.log, a.context.NodePools, a.context.ESSClient, a.context.RegionId)
}
