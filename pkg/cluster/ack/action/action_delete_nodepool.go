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
	"github.com/goph/emperror"
	"github.com/sirupsen/logrus"
)

// DeleteACKNodePoolAction describes the properties of an Alibaba cluster deletion
type DeleteACKNodePoolAction struct {
	context *ACKClusterDeleteContext
	log     logrus.FieldLogger
}

// NewDeleteACKNodePoolAction create  a new DeleteACKNodePoolAction
func NewDeleteACKNodePoolAction(log logrus.FieldLogger, creationContext *ACKClusterDeleteContext) *DeleteACKNodePoolAction {
	return &DeleteACKNodePoolAction{
		context: creationContext,
		log:     log,
	}
}

// GetName returns the name of this DeleteACKNodePoolAction
func (a *DeleteACKNodePoolAction) GetName() string {
	return "DeleteACKNodePoolAction"
}

// ExecuteAction executes this DeleteACKNodePoolAction
func (a *DeleteACKNodePoolAction) ExecuteAction(input interface{}) (output interface{}, err error) {
	if len(a.context.NodePools) == 0 {
		return nil, nil
	}
	a.log.Info("EXECUTE DeleteNodePoolAction")
	return nil, emperror.With(deleteNodePools(a.log, a.context.NodePools, a.context.ESSClient, a.context.RegionId), "cluster", a.context.ClusterName)
}
