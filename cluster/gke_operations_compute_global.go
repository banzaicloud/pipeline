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

package cluster

import (
	"context"

	gkeCompute "google.golang.org/api/compute/v1"
)

type computeGlobalOperation struct {
	csv       *gkeCompute.Service
	projectId string
}

func (co *computeGlobalOperation) getInfo(operationName string) (string, string, error) {

	op, err := co.csv.GlobalOperations.Get(co.projectId, operationName).Context(context.Background()).Do()
	if err != nil {
		return "", "", err
	}

	return op.Status, op.OperationType, nil
}

func newComputeGlobalOperation(csv *gkeCompute.Service, projectId string) *computeGlobalOperation {
	return &computeGlobalOperation{
		csv:       csv,
		projectId: projectId,
	}
}
